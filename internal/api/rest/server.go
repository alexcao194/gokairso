package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	_ "github.com/alexcao194/kairso/docs"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/internal/config"
	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/attendance"
	"github.com/alexcao194/kairso/internal/features/auth"
	"github.com/alexcao194/kairso/internal/features/device"
	"github.com/alexcao194/kairso/internal/features/event"
	"github.com/alexcao194/kairso/internal/features/files"
	"github.com/alexcao194/kairso/internal/features/meeting"
	"github.com/alexcao194/kairso/internal/features/module"
	"github.com/alexcao194/kairso/internal/features/notification"
	"github.com/alexcao194/kairso/internal/features/performance"
	"github.com/alexcao194/kairso/internal/features/project"
	"github.com/alexcao194/kairso/internal/features/projectfeedback"
	"github.com/alexcao194/kairso/internal/features/quote"
	"github.com/alexcao194/kairso/internal/features/task"
	"github.com/alexcao194/kairso/internal/features/user"
	"github.com/alexcao194/kairso/internal/features/whatsnew"
	"github.com/alexcao194/kairso/internal/features/workspace"
	"github.com/alexcao194/kairso/pkg/firebase"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/jwt"
	storage "github.com/alexcao194/kairso/pkg/minio"
	"github.com/alexcao194/kairso/pkg/realtime"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
)

type Server struct {
	cfg              config.Config
	db               *pgxpool.Pool
	http             *http.Server
	authHandler      *auth.Handler
	userHandler      *user.Handler
	workspaceHandler *workspace.Handler
	projectHandler   *project.Handler
	moduleHandler    *module.Handler
	taskHandler      *task.Handler
	attendance       *attendance.Handler
	performance      *performance.Handler
	event            *event.Handler
	files            *files.Handler
	device           *device.Handler
	quote            *quote.Handler
	notificationSvc  notification.Service
	notification     *notification.Handler
	whatsNewHandler  *whatsnew.Handler
	meeting          *meeting.Handler
	uploadHandler    *UploadHandler
	realtimeHandler  *RealtimeHandler
	projectFeedback  *projectfeedback.Handler
	publicFeedback   *projectfeedback.PublicHandler
	authMid          *middleware.AuthMiddleware
	appTokenMid      *middleware.AppTokenMiddleware
	errorLogMid      *middleware.ErrorLogMiddleware
	corsMid          *middleware.CORSMiddleware
	backgroundCancel context.CancelFunc
}

func NewServer(cfg config.Config, db *pgxpool.Pool) *Server {
	jwtSvc := jwt.NewJWTService(cfg.App.JWTSecret)
	firebaseVerifier, err := auth.NewFirebaseVerifier(context.Background(), cfg.App.FirebaseCredentialsPath)
	if err != nil {
		slog.Warn("Firebase verifier is not available", "error", err)
	}

	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, firebaseVerifier, jwtSvc, &cfg.App)

	notificationRepo := notification.NewRepository(db)
	notificationPubSub := gochannel.NewGoChannel(gochannel.Config{
		Persistent:                     true,
		BlockPublishUntilSubscriberAck: true,
	}, watermill.NewStdLogger(false, false))
	notificationSvc := notification.NewService(notificationRepo, notificationPubSub)
	if err := notificationSvc.StartConsumer(context.Background(), notificationPubSub); err != nil {
		slog.Warn("Notification consumer is not available", "error", err)
	}

	// Push notifications (FCM). The messenger is optional: without Firebase
	// credentials the device service still stores tokens but sends nothing.
	messenger, err := firebase.NewMessenger(context.Background(), cfg.App.FirebaseCredentialsPath)
	if err != nil {
		slog.Warn("Push messenger is not available", "error", err)
		messenger = nil
	}
	deviceSvc := device.NewService(device.NewRepository(db), messenger)
	notificationSvc.SetPusher(deviceSvc)

	minIOClient, err := storage.NewClient(cfg.Storage)
	if err != nil {
		slog.Warn("MinIO client is not available", "error", err)
	} else if err := minIOClient.EnsureBucketReady(context.Background()); err != nil {
		slog.Warn("MinIO bucket is not ready yet, will retry on first upload", "error", err)
	}

	realtimeHub := realtime.NewHub()

	notificationSvc.SetRealtime(realtimeHub)

	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo)
	userSvc.SetRealtime(realtimeHub)

	workspaceRepo := workspace.NewRepository(db)
	workspaceSvc := workspace.NewService(workspaceRepo, workspace.NewNotificationPublisher(notificationSvc))
	workspaceSvc.SetRealtime(realtimeHub)

	projectRepo := project.NewRepository(db)
	projectSvc := project.NewService(projectRepo, project.NewNotificationPublisher(notificationSvc))
	projectSvc.SetRealtime(realtimeHub)

	moduleRepo := module.NewRepository(db)
	moduleSvc := module.NewService(moduleRepo)
	moduleSvc.SetRealtime(realtimeHub)

	taskRepo := task.NewRepository(db)
	taskSvc := task.NewService(taskRepo, task.NewNotificationPublisher(notificationSvc))
	taskSvc.SetRealtime(realtimeHub)

	attendanceRepo := attendance.NewRepository(db)
	attendancePublisher := attendance.NewNotificationPublisher(notificationSvc)
	attendanceCoreSvc, attendanceSettingsSvc, attendanceSvc, workScheduleSvc, attendanceStatsSvc, taskStatsSvc, workWeekSvc, checkRequestSvc, dailyReportSvc := attendance.NewServices(attendanceRepo, attendancePublisher)
	attendanceCoreSvc.SetRealtime(realtimeHub)
	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())
	if db != nil {
		attendance.NewAttendanceReminderService(attendanceRepo, attendancePublisher).Start(backgroundCtx)
		task.NewDeadlineReminderService(taskRepo, task.NewNotificationPublisher(notificationSvc)).Start(backgroundCtx)
	}

	performanceRepo := performance.NewRepository(db)
	performanceCoreSvc, initiativeSvc, lateLogSvc, feedbackSvc, kpiSvc := performance.NewServices(performanceRepo, performance.NewNotificationPublisher(notificationSvc))
	performanceCoreSvc.SetRealtime(realtimeHub)

	eventRepo := event.NewRepository(db)
	eventSvc := event.NewService(eventRepo)

	whatsNewSvc := whatsnew.NewService(whatsnew.NewRepository(db))

	meetingSvc := meeting.NewService(meeting.NewRepository(db))
	meetingSvc.SetRealtime(realtimeHub)

	projectFeedbackRepo := projectfeedback.NewRepository(db)
	projectFeedbackSvc := projectfeedback.NewService(projectFeedbackRepo)

	filesRepo := files.NewRepository(db)
	var filesStorage files.Storage
	if minIOClient != nil {
		filesStorage = minIOClient
	}
	filesSvc := files.NewService(filesRepo, filesStorage)

	s := &Server{
		cfg:              cfg,
		db:               db,
		authHandler:      auth.NewHandler(authSvc),
		userHandler:      user.NewHandler(userSvc),
		workspaceHandler: workspace.NewHandler(workspaceSvc),
		projectHandler:   project.NewHandler(projectSvc),
		moduleHandler:    module.NewHandler(moduleSvc),
		taskHandler:      task.NewHandler(taskSvc),
		attendance:       attendance.NewHandler(attendanceSettingsSvc, attendanceSvc, workScheduleSvc, attendanceStatsSvc, taskStatsSvc, workWeekSvc, checkRequestSvc, dailyReportSvc),
		performance:      performance.NewHandler(initiativeSvc, lateLogSvc, feedbackSvc, kpiSvc),
		event:            event.NewHandler(eventSvc),
		files:            files.NewHandler(filesSvc),
		device:           device.NewHandler(deviceSvc),
		quote:            quote.NewHandler(quote.NewService(quote.NewRepository(db))),
		notificationSvc:  notificationSvc,
		notification:     notification.NewHandler(notificationSvc, &workspaceResolver{svc: workspaceSvc}),
		whatsNewHandler:  whatsnew.NewHandler(whatsNewSvc),
		meeting:          meeting.NewHandler(meetingSvc),
		uploadHandler:    NewUploadHandler(minIOClient),
		realtimeHandler:  NewRealtimeHandler(realtimeHub, jwtSvc, database.New(db)),
		projectFeedback:  projectfeedback.NewHandler(projectFeedbackSvc),
		publicFeedback:   projectfeedback.NewPublicHandler(projectFeedbackSvc, minIOClient),
		authMid:          middleware.NewAuthMiddleware(jwtSvc),
		appTokenMid:      middleware.NewAppTokenMiddleware(cfg.App.AppTokenSecret),
		errorLogMid:      middleware.NewErrorLogMiddleware(),
		corsMid:          middleware.NewCORSMiddleware(),
		backgroundCancel: backgroundCancel,
	}
	mux := http.NewServeMux()
	s.setupRoutes(mux)
	s.http = &http.Server{
		Addr:         ":8080",
		Handler:      s.errorLogMid.Handle(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

func (s *Server) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)
	s.realtimeHandler.RegisterRoutes(mux)

	publicMux := http.NewServeMux()
	s.publicFeedback.RegisterRoutes(publicMux)
	s.files.RegisterPublicRoutes(publicMux)
	mux.Handle("/public/", s.corsMid.Handle(publicMux))

	apiMux := http.NewServeMux()
	s.authHandler.RegisterRoutes(apiMux)
	s.userHandler.RegisterRoutes(apiMux, s.authMid)
	s.device.RegisterRoutes(apiMux, s.authMid)
	s.quote.RegisterRoutes(apiMux, s.authMid)
	s.workspaceHandler.RegisterRoutes(apiMux, s.authMid)
	s.projectHandler.RegisterRoutes(apiMux, s.authMid)
	s.moduleHandler.RegisterRoutes(apiMux, s.authMid)
	s.projectFeedback.RegisterRoutes(apiMux, s.authMid)
	s.taskHandler.RegisterRoutes(apiMux, s.authMid)
	s.attendance.RegisterRoutes(apiMux, s.authMid)
	s.performance.RegisterRoutes(apiMux, s.authMid)
	s.event.RegisterRoutes(apiMux, s.authMid)
	s.files.RegisterRoutes(apiMux, s.authMid)
	s.notification.RegisterRoutes(apiMux, s.authMid)
	s.whatsNewHandler.RegisterRoutes(apiMux, s.authMid)
	s.meeting.RegisterRoutes(apiMux, s.authMid)
	s.uploadHandler.RegisterRoutes(apiMux, s.authMid)

	mux.Handle("/api/", s.appTokenMid.Handle(http.StripPrefix("/api", apiMux)))
}

func (s *Server) Run() error {
	slog.Info("REST Server is listening", "addr", ":8080")
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down REST server")
	s.backgroundCancel()
	return s.http.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if s.db == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "unhealthy: database is not configured")
		return
	}
	if err := s.db.Ping(r.Context()); err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, fmt.Sprintf("unhealthy: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"message": "Chào mừng đến với Kairso API"})
}

type workspaceResolver struct {
	svc workspace.Service
}

func (r *workspaceResolver) GetWorkspaceID(ctx context.Context, workspaceAlias string) (string, error) {
	ws, err := r.svc.GetWorkspace(ctx, workspaceAlias)
	if err != nil {
		return "", err
	}
	return ws.ID, nil
}
