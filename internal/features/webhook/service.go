package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/alexcao194/kairso/pkg/randtoken"
	"github.com/alexcao194/kairso/pkg/realtime"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrForbidden = errors.New("Bạn không có quyền thực hiện thao tác này")
	ErrNotFound  = errors.New("Không tìm thấy webhook")
)

type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

func invalid(format string, a ...any) error { return ValidationError{Msg: fmt.Sprintf(format, a...)} }

const maxWebhooksPerWorkspace = 10

type Service struct {
	repo         Repository
	client       *http.Client
	allowPrivate bool
	events       chan realtime.Event
	jobs         chan job

	mu    sync.Mutex
	cache map[string]cacheEntry
}

// NewService builds the webhook service. allowPrivate permits targets on
// loopback/private networks (for self-hosted setups that trust their users).
func NewService(repo Repository, allowPrivate bool) *Service {
	return &Service{
		repo:         repo,
		client:       newHTTPClient(allowPrivate),
		allowPrivate: allowPrivate,
		events:       make(chan realtime.Event, eventBuffer),
		jobs:         make(chan job, jobBuffer),
		cache:        make(map[string]cacheEntry),
	}
}

func parseID(s, what string) (pgtype.UUID, error) {
	id, err := pgutil.ParseUUID(s)
	if err != nil {
		return pgtype.UUID{}, invalid("%s không hợp lệ", what)
	}
	return id, nil
}

// manager resolves the workspace and caller, requiring a holder/admin role:
// webhooks expose every workspace event and carry a secret.
func (s *Service) manager(ctx context.Context, workspaceID, callerID string) (pgtype.UUID, pgtype.UUID, error) {
	ws, err := parseID(workspaceID, "workspaceId")
	if err != nil {
		return ws, pgtype.UUID{}, err
	}
	caller, err := parseID(callerID, "userId")
	if err != nil {
		return ws, caller, err
	}
	role, err := s.repo.GetCallerRole(ctx, ws, caller)
	if err != nil {
		return ws, caller, err
	}
	if !role.CanManageMembers() {
		return ws, caller, ErrForbidden
	}
	return ws, caller, nil
}

func (s *Service) validate(req *WebhookRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)
	req.Token = strings.TrimSpace(req.Token)
	if req.Name == "" || len(req.Name) > 255 {
		return invalid("Tên webhook không được để trống và tối đa 255 ký tự")
	}
	if len(req.URL) > 2048 {
		return invalid("URL quá dài")
	}
	u, err := url.Parse(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return invalid("URL phải bắt đầu bằng http:// hoặc https://")
	}
	if u.User != nil {
		return invalid("URL không được chứa tên đăng nhập hoặc mật khẩu")
	}
	if !s.allowPrivate {
		host := strings.ToLower(u.Hostname())
		if host == "localhost" || strings.HasSuffix(host, ".localhost") {
			return invalid("URL không được trỏ vào mạng nội bộ")
		}
		if ip, err := netip.ParseAddr(host); err == nil && !isPublicAddr(ip) {
			return invalid("URL không được trỏ vào mạng nội bộ")
		}
	}
	if len(req.Token) > 255 {
		return invalid("Token tối đa 255 ký tự")
	}
	return nil
}

func (s *Service) List(ctx context.Context, workspaceID, callerID string) ([]WebhookResponse, error) {
	ws, _, err := s.manager(ctx, workspaceID, callerID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make([]WebhookResponse, 0, len(rows))
	for _, w := range rows {
		out = append(out, webhookResponse(w))
	}
	return out, nil
}

func (s *Service) Create(ctx context.Context, workspaceID, callerID string, req WebhookRequest) (*WebhookResponse, error) {
	ws, caller, err := s.manager(ctx, workspaceID, callerID)
	if err != nil {
		return nil, err
	}
	if err := s.validate(&req); err != nil {
		return nil, err
	}
	n, err := s.repo.Count(ctx, ws)
	if err != nil {
		return nil, err
	}
	if n >= maxWebhooksPerWorkspace {
		return nil, invalid("Mỗi workspace tối đa %d webhook", maxWebhooksPerWorkspace)
	}
	if req.Token == "" {
		if req.Token, err = randtoken.Generate(24); err != nil {
			return nil, err
		}
	}
	enabled := req.Enabled == nil || *req.Enabled
	w, err := s.repo.Create(ctx, database.CreateWebhookParams{
		WorkspaceID: ws, Name: req.Name, Url: req.URL, Token: req.Token, Enabled: enabled, CreatedBy: caller,
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(workspaceID)
	resp := webhookResponse(w)
	return &resp, nil
}

func (s *Service) Update(ctx context.Context, workspaceID, callerID, webhookID string, req WebhookRequest) (*WebhookResponse, error) {
	ws, _, err := s.manager(ctx, workspaceID, callerID)
	if err != nil {
		return nil, err
	}
	id, err := parseID(webhookID, "webhookId")
	if err != nil {
		return nil, err
	}
	if err := s.validate(&req); err != nil {
		return nil, err
	}
	cur, ok, err := s.repo.Get(ctx, id, ws)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	if req.Token == "" {
		req.Token = cur.Token
	}
	enabled := cur.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	w, ok, err := s.repo.Update(ctx, database.UpdateWebhookParams{
		ID: id, WorkspaceID: ws, Name: req.Name, Url: req.URL, Token: req.Token, Enabled: enabled,
	})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	s.invalidate(workspaceID)
	resp := webhookResponse(w)
	return &resp, nil
}

func (s *Service) Delete(ctx context.Context, workspaceID, callerID, webhookID string) error {
	ws, _, err := s.manager(ctx, workspaceID, callerID)
	if err != nil {
		return err
	}
	id, err := parseID(webhookID, "webhookId")
	if err != nil {
		return err
	}
	n, err := s.repo.Delete(ctx, id, ws)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	s.invalidate(workspaceID)
	return nil
}

func (s *Service) ListDeliveries(ctx context.Context, workspaceID, callerID, webhookID string) ([]WebhookDeliveryResponse, error) {
	ws, _, err := s.manager(ctx, workspaceID, callerID)
	if err != nil {
		return nil, err
	}
	id, err := parseID(webhookID, "webhookId")
	if err != nil {
		return nil, err
	}
	if _, ok, err := s.repo.Get(ctx, id, ws); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotFound
	}
	rows, err := s.repo.ListDeliveries(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]WebhookDeliveryResponse, 0, len(rows))
	for _, d := range rows {
		out = append(out, deliveryResponse(d))
	}
	return out, nil
}

// Test sends a single "webhook.test" event right away (even to a disabled
// webhook) and returns what the endpoint answered.
func (s *Service) Test(ctx context.Context, workspaceID, callerID, webhookID string) (*WebhookDeliveryResponse, error) {
	ws, _, err := s.manager(ctx, workspaceID, callerID)
	if err != nil {
		return nil, err
	}
	id, err := parseID(webhookID, "webhookId")
	if err != nil {
		return nil, err
	}
	w, ok, err := s.repo.Get(ctx, id, ws)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrNotFound
	}
	eventID := uuid.New()
	now := time.Now().UTC()
	body, err := json.Marshal(Payload{
		ID: eventID.String(), Type: EventTest, WorkspaceID: workspaceID, ActorID: callerID, OccurredAt: now,
		Data: map[string]string{"message": "Kairso webhook test", "webhook": w.Name},
	})
	if err != nil {
		return nil, err
	}
	j := job{workspaceID: workspaceID, target: target{id: w.ID, url: w.Url, token: w.Token}, eventID: eventID, eventType: EventTest, body: body, attempt: 1}
	res := s.send(ctx, j)
	s.record(ctx, j, res)
	return &WebhookDeliveryResponse{
		EventID: eventID.String(), EventType: EventTest, Success: res.success, StatusCode: res.statusCode,
		Error: res.errText, Attempts: 1, DurationMs: int(res.duration.Milliseconds()), CreatedAt: now,
	}, nil
}
