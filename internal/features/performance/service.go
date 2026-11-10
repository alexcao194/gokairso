package performance

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/attendance"
	"github.com/alexcao194/kairso/internal/features/workspace"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/alexcao194/kairso/pkg/realtime"
)

var (
	ErrForbidden = errors.New("Bạn không có quyền thực hiện thao tác này")
	ErrNotFound  = errors.New("Không tìm thấy dữ liệu")
)

type InitiativeService interface {
	CreateInitiative(ctx context.Context, workspaceAlias, callerID string, req CreateInitiativeRequest) (*InitiativeResponse, error)
	ListInitiatives(ctx context.Context, workspaceAlias, callerID string, params pagination.Params, filter ListInitiativesFilter) (pagination.Response[InitiativeResponse], error)
	UpdateInitiative(ctx context.Context, initiativeID, callerID string, req UpdateInitiativeRequest) (*InitiativeResponse, error)
	DeleteInitiative(ctx context.Context, initiativeID, callerID string) error
}

type LateLogService interface {
	CreateLateLog(ctx context.Context, workspaceAlias, callerID string, req CreateLateLogRequest) (*LateLogResponse, error)
	ListLateLogs(ctx context.Context, workspaceAlias, callerID string, params pagination.Params, filter ListLateLogsFilter) (pagination.Response[LateLogResponse], error)
	UpdateLateLog(ctx context.Context, lateLogID, callerID string, req UpdateLateLogRequest) (*LateLogResponse, error)
	DeleteLateLog(ctx context.Context, lateLogID, callerID string) error
}

type FeedbackService interface {
	CreateFeedback(ctx context.Context, workspaceAlias, callerID string, req CreateFeedbackRequest) (*FeedbackResponse, error)
	ListFeedbacks(ctx context.Context, workspaceAlias, callerID string, params pagination.Params, filter ListFeedbacksFilter) (pagination.Response[FeedbackResponse], error)
	UpdateFeedback(ctx context.Context, feedbackID, callerID string, req UpdateFeedbackRequest) (*FeedbackResponse, error)
	DeleteFeedback(ctx context.Context, feedbackID, callerID string) error
}

type KPIService interface {
	GetKPI(ctx context.Context, workspaceAlias, callerID string, filter KPIFilter) ([]KPIMemberResponse, error)
	GetMyKPI(ctx context.Context, workspaceAlias, callerID string, filter KPIFilter) (*KPIMemberResponse, error)
	SetAttitude(ctx context.Context, workspaceAlias, callerID string, req SetAttitudeRequest) (*AttitudeResponse, error)
	GetKPISettings(ctx context.Context, workspaceAlias, callerID string) (*KPISettingsResponse, error)
	UpsertKPISettings(ctx context.Context, workspaceAlias, callerID string, req UpsertKPISettingsRequest) (*KPISettingsResponse, error)
	GetMemberTaskBreakdown(ctx context.Context, workspaceAlias, callerID, memberID, month string) (*MemberTaskBreakdownResponse, error)
	ExportKPIExcel(ctx context.Context, workspaceAlias, callerID string, filter KPIFilter) (*KPIExportFile, error)
}

type Repository interface {
	GetCallerRole(ctx context.Context, workspaceAlias, callerID string) (workspace.WorkspaceRole, error)
	ListWorkspaceAdminHolderIDs(ctx context.Context, workspaceAlias string) ([]string, error)
	GetWorkspaceID(ctx context.Context, workspaceAlias string) (string, error)
	GetWorkspaceAliasByID(ctx context.Context, workspaceID string) (string, error)
	ListWorkspaceMembersForStatistics(ctx context.Context, workspaceAlias, memberID string) ([]database.ListWorkspaceMembersForStatisticsRow, error)

	CreateInitiative(ctx context.Context, workspaceAlias, proposerID, refMemberID string, req CreateInitiativeRequest) (*InitiativeResponse, error)
	ListInitiatives(ctx context.Context, workspaceAlias string, params pagination.Params, filter ListInitiativesFilter) (pagination.Response[InitiativeResponse], error)
	GetInitiative(ctx context.Context, initiativeID string) (*InitiativeResponse, error)
	UpdateInitiative(ctx context.Context, initiativeID string, refMemberID *string, req UpdateInitiativeRequest, setApprovedAt bool) (*InitiativeResponse, error)
	DeleteInitiative(ctx context.Context, initiativeID string) error

	CreateLateLog(ctx context.Context, workspaceAlias string, creatorID string, req CreateLateLogRequest, lateUserID string) (*LateLogResponse, error)
	ListLateLogs(ctx context.Context, workspaceAlias string, params pagination.Params, filter ListLateLogsFilter) (pagination.Response[LateLogResponse], error)
	GetLateLog(ctx context.Context, lateLogID string) (*LateLogResponse, error)
	UpdateLateLog(ctx context.Context, lateLogID string, req UpdateLateLogRequest) (*LateLogResponse, error)
	DeleteLateLog(ctx context.Context, lateLogID string) error

	CreateFeedback(ctx context.Context, workspaceAlias string, creatorID string, req CreateFeedbackRequest) (*FeedbackResponse, error)
	ListFeedbacks(ctx context.Context, workspaceAlias string, params pagination.Params, filter ListFeedbacksFilter) (pagination.Response[FeedbackResponse], error)
	GetFeedback(ctx context.Context, feedbackID string) (*FeedbackResponse, error)
	UpdateFeedback(ctx context.Context, feedbackID string, req UpdateFeedbackRequest) (*FeedbackResponse, error)
	DeleteFeedback(ctx context.Context, feedbackID string) error

	SumInitiativeEstimates(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]float64, error)
	SumLateLogEstimates(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]float64, error)
	SumFeedbackEstimates(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]float64, error)
	GetKPITaskStats(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) ([]kpiTaskStats, error)
	ListMemberTaskBreakdown(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) ([]memberTaskRow, error)
	GetKPIAttitudes(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]kpiAttitude, error)
	SetAttitude(ctx context.Context, workspaceAlias, callerID string, req SetAttitudeRequest) (*AttitudeResponse, error)
	GetKPISettings(ctx context.Context, workspaceAlias string) (*KPISettingsResponse, error)
	UpsertKPISettings(ctx context.Context, workspaceAlias string, req UpsertKPISettingsRequest) (*KPISettingsResponse, error)
	GetWorkspaceHolderName(ctx context.Context, workspaceAlias string) (string, error)

	ListWorkspaceScheduleStats(ctx context.Context, workspaceAlias string, from, to time.Time, memberID string) ([]attendance.WorkScheduleDayResponse, error)
	ListWorkspaceAttendanceStats(ctx context.Context, workspaceAlias string, from, to time.Time, memberID string) ([]attendance.AttendanceRecordResponse, error)
}

type services struct {
	repo      Repository
	publisher NotificationPublisher
	realtime  realtime.Broadcaster
}

func NewServices(repo Repository, publisher NotificationPublisher) (*services, InitiativeService, LateLogService, FeedbackService, KPIService) {
	s := &services{repo: repo, publisher: publisher}
	return s, s, s, s, s
}

func (s *services) SetRealtime(rt realtime.Broadcaster) {
	s.realtime = rt
}

func (s *services) broadcast(ctx context.Context, workspaceID, eventType, entityID, actorID string, entity any) {
	if s.realtime == nil || workspaceID == "" {
		return
	}
	s.realtime.Broadcast(realtime.Event{
		Type:        eventType,
		WorkspaceID: workspaceID,
		EntityID:    entityID,
		ActorID:     actorID,
		ActorConnID: middleware.GetConnIDFromContext(ctx),
		Entity:      entity,
	})
}

func (s *services) checkWorkspaceAccess(ctx context.Context, workspaceAlias, callerID string) (workspace.WorkspaceRole, error) {
	role, err := s.repo.GetCallerRole(ctx, workspaceAlias, callerID)
	if err != nil {
		return "", err
	}
	if role == "" {
		return "", ErrForbidden
	}
	return role, nil
}

func (s *services) notifyAdmins(ctx context.Context, workspaceID, entityID, notifType, callerID string) error {
	workspaceAlias, err := s.repo.GetWorkspaceAliasByID(ctx, workspaceID)
	if err != nil {
		return err
	}
	recipientIDs, err := s.repo.ListWorkspaceAdminHolderIDs(ctx, workspaceID)
	if err != nil {
		return err
	}
	for _, recipientID := range recipientIDs {
		if recipientID == callerID {
			continue
		}
		if err := s.publisher.PublishPerformanceNotification(ctx, PerformanceNotificationEvent{
			Type:           notifType,
			RecipientID:    recipientID,
			WorkspaceID:    workspaceID,
			WorkspaceAlias: workspaceAlias,
			EntityID:       entityID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *services) CreateInitiative(ctx context.Context, workspaceAlias, callerID string, req CreateInitiativeRequest) (*InitiativeResponse, error) {
	if _, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID); err != nil {
		return nil, err
	}
	if req.Content == "" {
		return nil, fmt.Errorf("Nội dung là bắt buộc")
	}
	if req.RefMemberID == "" {
		return nil, fmt.Errorf("refMemberId là bắt buộc")
	}
	resp, err := s.repo.CreateInitiative(ctx, workspaceAlias, callerID, req.RefMemberID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, resp.WorkspaceID, "initiative.created", resp.ID, callerID, resp)
	_ = s.notifyAdmins(ctx, workspaceAlias, resp.ID, NotificationTypeInitiativeCreated, callerID)
	return resp, nil
}

func (s *services) ListInitiatives(ctx context.Context, workspaceAlias, callerID string, params pagination.Params, filter ListInitiativesFilter) (pagination.Response[InitiativeResponse], error) {
	if _, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID); err != nil {
		return pagination.Response[InitiativeResponse]{}, err
	}
	return s.repo.ListInitiatives(ctx, workspaceAlias, params, filter)
}

func (s *services) UpdateInitiative(ctx context.Context, initiativeID, callerID string, req UpdateInitiativeRequest) (*InitiativeResponse, error) {
	initiative, err := s.repo.GetInitiative(ctx, initiativeID)
	if err != nil {
		return nil, ErrNotFound
	}
	role, err := s.checkWorkspaceAccess(ctx, initiative.WorkspaceID, callerID)
	if err != nil {
		return nil, err
	}
	if req.Status != nil && *req.Status == InitiativeStatusAccepted {
		if !role.CanManageMembers() {
			return nil, fmt.Errorf("%w: chỉ quản trị viên hoặc chủ workspace mới được chấp nhận sáng kiến", ErrForbidden)
		}
	}
	if req.Status != nil && !req.Status.Valid() {
		return nil, fmt.Errorf("Trạng thái không hợp lệ: %s", *req.Status)
	}
	setApprovedAt := req.Status != nil && *req.Status == InitiativeStatusAccepted

	var refMemberIDPtr *string
	if req.RefMemberID != nil {
		refMemberIDPtr = req.RefMemberID
	}

	updated, err := s.repo.UpdateInitiative(ctx, initiativeID, refMemberIDPtr, req, setApprovedAt)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, updated.WorkspaceID, "initiative.updated", updated.ID, callerID, updated)
	return updated, nil
}

func (s *services) DeleteInitiative(ctx context.Context, initiativeID, callerID string) error {
	initiative, err := s.repo.GetInitiative(ctx, initiativeID)
	if err != nil {
		return ErrNotFound
	}
	if _, err := s.checkWorkspaceAccess(ctx, initiative.WorkspaceID, callerID); err != nil {
		return err
	}
	if err := s.repo.DeleteInitiative(ctx, initiativeID); err != nil {
		return err
	}
	s.broadcast(ctx, initiative.WorkspaceID, "initiative.deleted", initiativeID, callerID, nil)
	return nil
}

func (s *services) CreateLateLog(ctx context.Context, workspaceAlias, callerID string, req CreateLateLogRequest) (*LateLogResponse, error) {
	if _, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID); err != nil {
		return nil, err
	}
	if req.Content == "" {
		return nil, fmt.Errorf("Nội dung là bắt buộc")
	}
	if req.LateUserID == "" {
		return nil, fmt.Errorf("lateUserId là bắt buộc")
	}
	if req.LateTime.IsZero() {
		return nil, fmt.Errorf("lateTime là bắt buộc")
	}
	resp, err := s.repo.CreateLateLog(ctx, workspaceAlias, callerID, req, req.LateUserID)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, resp.WorkspaceID, "late_log.created", resp.ID, callerID, resp)
	_ = s.notifyAdmins(ctx, workspaceAlias, resp.ID, NotificationTypeLateLogCreated, callerID)
	return resp, nil
}

func (s *services) ListLateLogs(ctx context.Context, workspaceAlias, callerID string, params pagination.Params, filter ListLateLogsFilter) (pagination.Response[LateLogResponse], error) {
	if _, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID); err != nil {
		return pagination.Response[LateLogResponse]{}, err
	}
	return s.repo.ListLateLogs(ctx, workspaceAlias, params, filter)
}

func (s *services) UpdateLateLog(ctx context.Context, lateLogID, callerID string, req UpdateLateLogRequest) (*LateLogResponse, error) {
	lateLog, err := s.repo.GetLateLog(ctx, lateLogID)
	if err != nil {
		return nil, ErrNotFound
	}
	if _, err := s.checkWorkspaceAccess(ctx, lateLog.WorkspaceID, callerID); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateLateLog(ctx, lateLogID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, updated.WorkspaceID, "late_log.updated", updated.ID, callerID, updated)
	return updated, nil
}

func (s *services) DeleteLateLog(ctx context.Context, lateLogID, callerID string) error {
	lateLog, err := s.repo.GetLateLog(ctx, lateLogID)
	if err != nil {
		return ErrNotFound
	}
	if _, err := s.checkWorkspaceAccess(ctx, lateLog.WorkspaceID, callerID); err != nil {
		return err
	}
	if err := s.repo.DeleteLateLog(ctx, lateLogID); err != nil {
		return err
	}
	s.broadcast(ctx, lateLog.WorkspaceID, "late_log.deleted", lateLogID, callerID, nil)
	return nil
}

func (s *services) CreateFeedback(ctx context.Context, workspaceAlias, callerID string, req CreateFeedbackRequest) (*FeedbackResponse, error) {
	if _, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID); err != nil {
		return nil, err
	}
	if req.FeedbackText == "" {
		return nil, fmt.Errorf("feedbackText là bắt buộc")
	}
	if req.Content == "" {
		return nil, fmt.Errorf("Nội dung là bắt buộc")
	}
	resp, err := s.repo.CreateFeedback(ctx, workspaceAlias, callerID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, resp.WorkspaceID, "feedback.created", resp.ID, callerID, resp)
	_ = s.notifyAdmins(ctx, workspaceAlias, resp.ID, NotificationTypeFeedbackCreated, callerID)
	return resp, nil
}

func (s *services) ListFeedbacks(ctx context.Context, workspaceAlias, callerID string, params pagination.Params, filter ListFeedbacksFilter) (pagination.Response[FeedbackResponse], error) {
	if _, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID); err != nil {
		return pagination.Response[FeedbackResponse]{}, err
	}
	return s.repo.ListFeedbacks(ctx, workspaceAlias, params, filter)
}

func (s *services) UpdateFeedback(ctx context.Context, feedbackID, callerID string, req UpdateFeedbackRequest) (*FeedbackResponse, error) {
	feedback, err := s.repo.GetFeedback(ctx, feedbackID)
	if err != nil {
		return nil, ErrNotFound
	}
	if _, err := s.checkWorkspaceAccess(ctx, feedback.WorkspaceID, callerID); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateFeedback(ctx, feedbackID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, updated.WorkspaceID, "feedback.updated", updated.ID, callerID, updated)
	return updated, nil
}

func (s *services) DeleteFeedback(ctx context.Context, feedbackID, callerID string) error {
	feedback, err := s.repo.GetFeedback(ctx, feedbackID)
	if err != nil {
		return ErrNotFound
	}
	if _, err := s.checkWorkspaceAccess(ctx, feedback.WorkspaceID, callerID); err != nil {
		return err
	}
	if err := s.repo.DeleteFeedback(ctx, feedbackID); err != nil {
		return err
	}
	s.broadcast(ctx, feedback.WorkspaceID, "feedback.deleted", feedbackID, callerID, nil)
	return nil
}

func (s *services) GetKPI(ctx context.Context, workspaceAlias, callerID string, filter KPIFilter) ([]KPIMemberResponse, error) {
	role, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID)
	if err != nil {
		return nil, err
	}
	memberID := filter.MemberID
	if !role.CanManageMembers() {
		memberID = callerID
	}
	return s.computeKPI(ctx, workspaceAlias, memberID, filter)
}

func (s *services) GetMemberTaskBreakdown(ctx context.Context, workspaceAlias, callerID, memberID, month string) (*MemberTaskBreakdownResponse, error) {
	role, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID)
	if err != nil {
		return nil, err
	}
	if !role.CanManageMembers() || memberID == "" {
		memberID = callerID
	}
	from, to, err := resolveKPIPeriod(month)
	if err != nil {
		return nil, err
	}
	label := month
	if month == "" {
		label = from.Format("2006-01")
	}
	rows, err := s.repo.ListMemberTaskBreakdown(ctx, workspaceAlias, memberID, from, to)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	resp := &MemberTaskBreakdownResponse{
		MemberID: memberID, Month: month, Label: label,
		BugTasks:           []MemberTaskBrief{},
		CompletedLateTasks: []MemberTaskBrief{},
		OverdueTasks:       []MemberTaskBrief{},
		ReworkTasks:        []MemberTaskBrief{},
	}
	for _, r := range rows {
		b := r.brief
		if r.isFaultHolder {
			resp.BugTasks = append(resp.BugTasks, b)
		}
		if !r.isWorker {
			continue
		}
		if b.FirstCompletedAt != nil && b.EndDate != nil && b.FirstCompletedAt.After(*b.EndDate) {
			resp.CompletedLateTasks = append(resp.CompletedLateTasks, b)
		}
		if b.Status != "done" && b.Status != "canceled" && b.EndDate != nil && b.EndDate.Before(now) {
			resp.OverdueTasks = append(resp.OverdueTasks, b)
		}
		if b.ReturnCount > 0 {
			resp.ReworkTasks = append(resp.ReworkTasks, b)
		}
	}
	return resp, nil
}

func (s *services) ExportKPIExcel(ctx context.Context, workspaceAlias, callerID string, filter KPIFilter) (*KPIExportFile, error) {
	role, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID)
	if err != nil {
		return nil, err
	}
	if !role.CanManageMembers() {
		return nil, fmt.Errorf("%w: chỉ quản trị viên hoặc chủ workspace mới được xuất phiếu KPI", ErrForbidden)
	}

	members, err := s.computeKPI(ctx, workspaceAlias, filter.MemberID, filter)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, ErrNotFound
	}

	settings, err := s.repo.GetKPISettings(ctx, workspaceAlias)
	if err != nil {
		return nil, err
	}
	controllerName, err := s.repo.GetWorkspaceHolderName(ctx, workspaceAlias)
	if err != nil {
		return nil, err
	}

	from, to, err := resolveKPIPeriod(filter.Month)
	if err != nil {
		return nil, err
	}
	records, err := s.gatherKPINoteRecords(ctx, workspaceAlias, filter.MemberID, from, to)
	if err != nil {
		return nil, err
	}

	monthLabel := from.Format("01/2006")
	data, err := buildKPIWorkbook(members, *settings, records, monthLabel, controllerName)
	if err != nil {
		return nil, err
	}

	filename := "KPI_" + from.Format("01-2006") + ".xlsx"
	if filter.MemberID != "" {
		filename = "KPI_" + safeFilenamePart(members[0].MemberAlias) + "_" + from.Format("01-2006") + ".xlsx"
	}
	return &KPIExportFile{Filename: filename, Data: data}, nil
}

func (s *services) gatherKPINoteRecords(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]*kpiMemberRecords, error) {
	fromStr := from.Format(time.RFC3339)
	toStr := to.Format(time.RFC3339)

	feedbacks, err := listAllPages(func(p pagination.Params) (pagination.Response[FeedbackResponse], error) {
		return s.repo.ListFeedbacks(ctx, workspaceAlias, p, ListFeedbacksFilter{MemberID: memberID, From: fromStr, To: toStr})
	})
	if err != nil {
		return nil, err
	}
	initiatives, err := listAllPages(func(p pagination.Params) (pagination.Response[InitiativeResponse], error) {
		return s.repo.ListInitiatives(ctx, workspaceAlias, p, ListInitiativesFilter{RefMemberID: memberID, To: toStr})
	})
	if err != nil {
		return nil, err
	}
	lateLogs, err := listAllPages(func(p pagination.Params) (pagination.Response[LateLogResponse], error) {
		return s.repo.ListLateLogs(ctx, workspaceAlias, p, ListLateLogsFilter{LateUserID: memberID, From: fromStr, To: toStr})
	})
	if err != nil {
		return nil, err
	}

	out := map[string]*kpiMemberRecords{}
	bucket := func(id string) *kpiMemberRecords {
		b := out[id]
		if b == nil {
			b = &kpiMemberRecords{}
			out[id] = b
		}
		return b
	}
	for _, f := range feedbacks {
		for _, b := range f.Beneficiaries {
			bk := bucket(b.ID)
			bk.feedbacks = append(bk.feedbacks, f)
		}
	}
	for _, i := range initiatives {
		if i.ApprovedAt == nil || i.ApprovedAt.Before(from) || i.ApprovedAt.After(to) {
			continue
		}
		bk := bucket(i.RefMember.ID)
		bk.initiatives = append(bk.initiatives, i)
	}
	for _, l := range lateLogs {
		bk := bucket(l.LateUser.ID)
		bk.lateLogs = append(bk.lateLogs, l)
	}
	for _, b := range out {
		sort.SliceStable(b.feedbacks, func(x, y int) bool { return b.feedbacks[x].CreatedAt.Before(b.feedbacks[y].CreatedAt) })
		sort.SliceStable(b.initiatives, func(x, y int) bool {
			return derefTime(b.initiatives[x].ApprovedAt).Before(derefTime(b.initiatives[y].ApprovedAt))
		})
		sort.SliceStable(b.lateLogs, func(x, y int) bool { return b.lateLogs[x].LateTime.Before(b.lateLogs[y].LateTime) })
	}
	return out, nil
}

func listAllPages[T any](fetch func(pagination.Params) (pagination.Response[T], error)) ([]T, error) {
	var all []T
	for page := 1; ; page++ {
		resp, err := fetch(pagination.Params{Page: page, Limit: 200})
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Items...)
		if !resp.HasMore {
			return all, nil
		}
	}
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

func (s *services) GetMyKPI(ctx context.Context, workspaceAlias, callerID string, filter KPIFilter) (*KPIMemberResponse, error) {
	if _, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID); err != nil {
		return nil, err
	}
	filter.MemberID = callerID
	rows, err := s.computeKPI(ctx, workspaceAlias, callerID, filter)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}

func (s *services) SetAttitude(ctx context.Context, workspaceAlias, callerID string, req SetAttitudeRequest) (*AttitudeResponse, error) {
	role, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID)
	if err != nil {
		return nil, err
	}
	if !role.CanManageMembers() {
		return nil, fmt.Errorf("%w: chỉ quản trị viên hoặc chủ workspace mới được chấm điểm thái độ", ErrForbidden)
	}
	if req.MemberID == "" {
		return nil, fmt.Errorf("memberId là bắt buộc")
	}
	if req.Month == "" {
		return nil, fmt.Errorf("month là bắt buộc (YYYY-MM)")
	}
	resp, err := s.repo.SetAttitude(ctx, workspaceAlias, callerID, req)
	if err != nil {
		return nil, err
	}
	if workspaceID, wErr := s.repo.GetWorkspaceID(ctx, workspaceAlias); wErr == nil {
		s.broadcast(ctx, workspaceID, "attitude.updated", req.MemberID, callerID, nil)
	}
	return resp, nil
}

func (s *services) GetKPISettings(ctx context.Context, workspaceAlias, callerID string) (*KPISettingsResponse, error) {
	role, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID)
	if err != nil {
		return nil, err
	}
	if role != workspace.WorkspaceRoleHolder {
		return nil, fmt.Errorf("%w: chỉ chủ workspace mới được xem cấu hình KPI", ErrForbidden)
	}
	return s.repo.GetKPISettings(ctx, workspaceAlias)
}

func (s *services) UpsertKPISettings(ctx context.Context, workspaceAlias, callerID string, req UpsertKPISettingsRequest) (*KPISettingsResponse, error) {
	role, err := s.checkWorkspaceAccess(ctx, workspaceAlias, callerID)
	if err != nil {
		return nil, err
	}
	if role != workspace.WorkspaceRoleHolder {
		return nil, fmt.Errorf("%w: chỉ chủ workspace mới được sửa cấu hình KPI", ErrForbidden)
	}
	settings, err := s.repo.UpsertKPISettings(ctx, workspaceAlias, req)
	if err != nil {
		return nil, err
	}
	if workspaceID, wErr := s.repo.GetWorkspaceID(ctx, workspaceAlias); wErr == nil {
		s.broadcast(ctx, workspaceID, "kpi_settings.updated", workspaceID, callerID, settings)
	}
	return settings, nil
}

func (s *services) computeKPI(ctx context.Context, workspaceAlias, memberID string, filter KPIFilter) ([]KPIMemberResponse, error) {
	from, to, err := resolveKPIPeriod(filter.Month)
	if err != nil {
		return nil, err
	}
	period, label := "month", filter.Month
	if filter.Month == "" {
		label = from.Format("2006-01")
	}

	taskRows, err := s.repo.GetKPITaskStats(ctx, workspaceAlias, memberID, from, to)
	if err != nil {
		return nil, err
	}
	feedbackMap, err := s.repo.SumFeedbackEstimates(ctx, workspaceAlias, memberID, from, to)
	if err != nil {
		return nil, err
	}
	initiativeMap, err := s.repo.SumInitiativeEstimates(ctx, workspaceAlias, memberID, from, to)
	if err != nil {
		return nil, err
	}
	lateLogMap, err := s.repo.SumLateLogEstimates(ctx, workspaceAlias, memberID, from, to)
	if err != nil {
		return nil, err
	}
	attitudeMap, err := s.repo.GetKPIAttitudes(ctx, workspaceAlias, memberID, from, to)
	if err != nil {
		return nil, err
	}
	schedules, err := s.repo.ListWorkspaceScheduleStats(ctx, workspaceAlias, from, to, memberID)
	if err != nil {
		return nil, err
	}
	schedules = attendance.ApprovedSchedulesOnly(schedules)
	records, err := s.repo.ListWorkspaceAttendanceStats(ctx, workspaceAlias, from, to, memberID)
	if err != nil {
		return nil, err
	}

	membersRow, err := s.repo.ListWorkspaceMembersForStatistics(ctx, workspaceAlias, memberID)
	if err != nil {
		return nil, err
	}

	settings, err := s.repo.GetKPISettings(ctx, workspaceAlias)
	if err != nil {
		return nil, err
	}

	taskStatsMap := make(map[string]*kpiTaskStats)
	for i, r := range taskRows {
		taskStatsMap[r.memberID] = &taskRows[i]
	}

	results := make([]KPIMemberResponse, 0, len(membersRow))
	for _, m := range membersRow {
		idStr := pgutil.UUIDToString(m.MemberID)
		att := attitudeMap[idStr]
		memberSchedules := schedulesForMember(schedules, idStr)
		memberRecords := recordsForMember(records, idStr)
		lateDays, earlyLeaveDays := lateEarlyDays(memberSchedules, memberRecords)
		resp := buildKPIMember(
			idStr, m.MemberAlias, m.MemberName, textString(m.MemberAvatar), period, label,
			taskStatsMap[idStr],
			feedbackMap[idStr],
			initiativeMap[idStr],
			lateLogMap[idStr],
			att,
			lateDays,
			earlyLeaveDays,
			*settings,
		)
		results = append(results, resp)
	}
	return results, nil
}

func resolveKPIPeriod(month string) (from, to time.Time, err error) {
	loc := time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)
	now := time.Now().In(loc)
	if month == "" {
		month = now.Format("2006-01")
	}
	t, parseErr := time.ParseInLocation("2006-01", month, loc)
	if parseErr != nil {
		err = fmt.Errorf("month phải theo định dạng YYYY-MM")
		return
	}
	from = t
	to = t.AddDate(0, 1, 0).Add(-time.Second)
	return
}

func vietnamLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)
	}
	return loc
}

func lateEarlyDays(schedules []attendance.WorkScheduleDayResponse, records []attendance.AttendanceRecordResponse) (int, int) {
	recordsByDate := map[string][]attendance.AttendanceRecordResponse{}
	for _, record := range records {
		date := record.RecordedAt.In(vietnamLocation()).Format(time.DateOnly)
		recordsByDate[date] = append(recordsByDate[date], record)
	}
	var lateDays, earlyDays int
	for _, schedule := range schedules {
		if len(schedule.Sessions) == 0 {
			continue
		}
		dayRecords := recordsByDate[schedule.Date]
		if len(dayRecords) == 0 {
			continue
		}
		firstSessionStart := schedule.Sessions[0].StartAt
		lastSessionEnd := schedule.Sessions[0].EndAt
		for _, session := range schedule.Sessions {
			if session.StartAt.Before(firstSessionStart) {
				firstSessionStart = session.StartAt
			}
			if session.EndAt.After(lastSessionEnd) {
				lastSessionEnd = session.EndAt
			}
		}
		var firstIn *time.Time
		var lastOut *time.Time
		for i := range dayRecords {
			record := dayRecords[i]
			if record.Type == attendance.AttendanceTypeCheckIn && (firstIn == nil || record.RecordedAt.Before(*firstIn)) {
				t := record.RecordedAt
				firstIn = &t
			}
			if record.Type == attendance.AttendanceTypeCheckOut && (lastOut == nil || record.RecordedAt.After(*lastOut)) {
				t := record.RecordedAt
				lastOut = &t
			}
		}
		if firstIn != nil && firstIn.After(firstSessionStart) {
			lateDays++
		}
		if lastOut != nil && lastOut.Before(lastSessionEnd) {
			earlyDays++
		}
	}
	return lateDays, earlyDays
}

func schedulesForMember(schedules []attendance.WorkScheduleDayResponse, memberID string) []attendance.WorkScheduleDayResponse {
	var out []attendance.WorkScheduleDayResponse
	for _, schedule := range schedules {
		if schedule.MemberID == memberID {
			out = append(out, schedule)
		}
	}
	return out
}

func recordsForMember(records []attendance.AttendanceRecordResponse, memberID string) []attendance.AttendanceRecordResponse {
	var out []attendance.AttendanceRecordResponse
	for _, record := range records {
		if record.MemberID == memberID {
			out = append(out, record)
		}
	}
	return out
}
