package module

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/realtime"
)

type Service interface {
	CreateModule(ctx context.Context, projectID string, req CreateModuleRequest) (*ModuleResponse, error)
	ListModules(ctx context.Context, projectAlias string, params pagination.StatsParams) (pagination.Response[ModuleResponse], error)
	GetModule(ctx context.Context, moduleID string) (*ModuleResponse, error)
	AuthorizeModuleAccess(ctx context.Context, moduleID, callerID string) error
	AuthorizeProjectAccess(ctx context.Context, projectID, callerID string) error
	UpdateModule(ctx context.Context, moduleID, callerID string, req UpdateModuleRequest) (*ModuleResponse, error)
	ListModuleHistory(ctx context.Context, moduleID string, params pagination.Params) (pagination.Response[ModuleHistoryResponse], error)
	DeleteModule(ctx context.Context, moduleID string) error

	AddAttachment(ctx context.Context, moduleID, uploaderID string, req CreateModuleAttachmentRequest) (*ModuleAttachmentResponse, error)
	ListAttachments(ctx context.Context, moduleID string) ([]ModuleAttachmentResponse, error)
	DeleteAttachment(ctx context.Context, attachmentID string) error

	CreateSprint(ctx context.Context, moduleID, callerID string, req CreateModuleSprintRequest) (*ModuleResponse, error)
	UpdateSprint(ctx context.Context, moduleID, sprintID, callerID string, req UpdateModuleSprintRequest) (*ModuleResponse, error)
	DeleteSprint(ctx context.Context, moduleID, sprintID, callerID string) (*ModuleResponse, error)
}

var (
	ErrForbidden      = errors.New("Bạn không có quyền thực hiện thao tác này")
	ErrSprintNotFound = errors.New("Không tìm thấy sprint")
	ErrInvalidSprint  = errors.New("Sprint không hợp lệ")
)

type service struct {
	repo     Repository
	realtime realtime.Broadcaster
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) SetRealtime(rt realtime.Broadcaster) {
	s.realtime = rt
}

func (s *service) broadcast(ctx context.Context, projectID, eventType, entityID, actorID string, entity any) {
	if s.realtime == nil {
		return
	}
	workspaceID, err := s.repo.GetWorkspaceIDByProjectID(ctx, projectID)
	if err != nil || workspaceID == "" {
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

func (s *service) CreateModule(ctx context.Context, projectID string, req CreateModuleRequest) (*ModuleResponse, error) {
	module, err := s.repo.CreateModule(ctx, projectID, req)
	if err != nil {
		return nil, err
	}
	if module != nil {
		s.broadcast(ctx, projectID, "module.created", module.ID, "", module)
	}
	return module, nil
}

func (s *service) ListModules(ctx context.Context, projectAlias string, params pagination.StatsParams) (pagination.Response[ModuleResponse], error) {
	return s.repo.ListModules(ctx, projectAlias, params)
}

func (s *service) AuthorizeModuleAccess(ctx context.Context, moduleID, callerID string) error {
	ok, err := s.repo.CheckModuleMembership(ctx, moduleID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: bạn không thuộc dự án của module này", ErrForbidden)
	}
	return nil
}

func (s *service) AuthorizeProjectAccess(ctx context.Context, projectID, callerID string) error {
	ok, err := s.repo.CheckProjectMembership(ctx, projectID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: bạn không thuộc dự án này", ErrForbidden)
	}
	return nil
}

func (s *service) GetModule(ctx context.Context, moduleID string) (*ModuleResponse, error) {
	return s.repo.GetModule(ctx, moduleID)
}

func (s *service) UpdateModule(ctx context.Context, moduleID, callerID string, req UpdateModuleRequest) (*ModuleResponse, error) {
	module, err := s.repo.UpdateModule(ctx, moduleID, callerID, req)
	if err != nil {
		return nil, err
	}
	if module != nil {
		s.broadcast(ctx, module.ProjectID, "module.updated", module.ID, callerID, module)
	}
	return module, nil
}

func (s *service) ListModuleHistory(ctx context.Context, moduleID string, params pagination.Params) (pagination.Response[ModuleHistoryResponse], error) {
	return s.repo.ListModuleHistory(ctx, moduleID, params)
}

func (s *service) AddAttachment(ctx context.Context, moduleID, uploaderID string, req CreateModuleAttachmentRequest) (*ModuleAttachmentResponse, error) {
	return s.repo.AddAttachment(ctx, moduleID, uploaderID, req)
}

func (s *service) ListAttachments(ctx context.Context, moduleID string) ([]ModuleAttachmentResponse, error) {
	return s.repo.ListAttachments(ctx, moduleID)
}

func (s *service) DeleteAttachment(ctx context.Context, attachmentID string) error {
	return s.repo.DeleteAttachment(ctx, attachmentID)
}

func (s *service) DeleteModule(ctx context.Context, moduleID string) error {
	module, moduleErr := s.repo.GetModule(ctx, moduleID)
	if err := s.repo.DeleteModule(ctx, moduleID); err != nil {
		return err
	}
	if moduleErr == nil && module != nil {
		s.broadcast(ctx, module.ProjectID, "module.deleted", moduleID, "", nil)
	}
	return nil
}

var sprintStatusLabel = map[SprintStatus]string{
	SprintStatusPlanned:   "Lên kế hoạch",
	SprintStatusActive:    "Đang chạy",
	SprintStatusCompleted: "Đã xong",
}

func invalidSprint(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidSprint, fmt.Sprintf(format, args...))
}

func ddmm(date string) string {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return date
	}
	return t.Format("02/01")
}

func sprintLabel(sp ModuleSprintResponse) string {
	return fmt.Sprintf("%s (%s–%s): %s", sp.Name, ddmm(sp.StartDate), ddmm(sp.EndDate), sprintStatusLabel[sp.Status])
}

func validateSprint(sp ModuleSprintResponse, others []ModuleSprintResponse) error {
	start, err := time.Parse(time.DateOnly, sp.StartDate)
	if err != nil {
		return invalidSprint("ngày bắt đầu phải theo định dạng YYYY-MM-DD")
	}
	end, err := time.Parse(time.DateOnly, sp.EndDate)
	if err != nil {
		return invalidSprint("ngày kết thúc phải theo định dạng YYYY-MM-DD")
	}
	if end.Before(start) {
		return invalidSprint("ngày kết thúc phải từ ngày bắt đầu trở đi")
	}
	if days := int(end.Sub(start).Hours()/24) + 1; days > maxSprintDays {
		return invalidSprint("sprint dài %d ngày — theo Scrum một sprint tối đa 1 tháng (%d ngày)", days, maxSprintDays)
	}
	if sp.Name == "" {
		return invalidSprint("tên sprint không được để trống")
	}
	if !sp.Status.Valid() {
		return invalidSprint("trạng thái phải là planned, active hoặc completed")
	}
	for _, o := range others {
		if o.ID == sp.ID {
			continue
		}
		if o.StartDate <= sp.EndDate && sp.StartDate <= o.EndDate {
			return invalidSprint("trùng thời gian với %s (%s–%s)", o.Name, ddmm(o.StartDate), ddmm(o.EndDate))
		}
		if sp.Status == SprintStatusActive && o.Status == SprintStatusActive {
			return invalidSprint("%s đang chạy — kết thúc sprint đó trước khi bắt đầu sprint mới", o.Name)
		}
	}
	return nil
}

func (s *service) moduleForSprint(ctx context.Context, moduleID string) (*ModuleResponse, error) {
	m, err := s.repo.GetModule(ctx, moduleID)
	if err != nil {
		return nil, fmt.Errorf("không tìm thấy module: %w", err)
	}
	return m, nil
}

func (s *service) afterSprintChange(ctx context.Context, moduleID, callerID string, started bool) (*ModuleResponse, error) {
	m, err := s.repo.GetModule(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	if started && m.Status == ModuleStatusBacklog {
		st := ModuleStatusInProgress
		if m, err = s.repo.UpdateModule(ctx, moduleID, callerID, UpdateModuleRequest{Status: &st}); err != nil {
			return nil, err
		}
	}
	s.broadcast(ctx, m.ProjectID, "module.updated", m.ID, callerID, m)
	return m, nil
}

func (s *service) CreateSprint(ctx context.Context, moduleID, callerID string, req CreateModuleSprintRequest) (*ModuleResponse, error) {
	m, err := s.moduleForSprint(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	sp := ModuleSprintResponse{
		Name:      strings.TrimSpace(req.Name),
		Goal:      strings.TrimSpace(req.Goal),
		StartDate: strings.TrimSpace(req.StartDate),
		EndDate:   strings.TrimSpace(req.EndDate),
		Status:    SprintStatusPlanned,
	}
	if sp.Name == "" {
		sp.Name = fmt.Sprintf("Sprint %d", len(m.Sprints)+1)
	}
	if sp.StartDate == "" {
		start := time.Now().In(vietnamLocation())
		if n := len(m.Sprints); n > 0 {
			if last, err := time.Parse(time.DateOnly, m.Sprints[n-1].EndDate); err == nil {
				start = last.AddDate(0, 0, 1)
			}
		}
		sp.StartDate = start.Format(time.DateOnly)
	}
	if sp.EndDate == "" {
		if start, err := time.Parse(time.DateOnly, sp.StartDate); err == nil {
			sp.EndDate = start.AddDate(0, 0, defaultSprintDays-1).Format(time.DateOnly)
		}
	}
	if err := validateSprint(sp, m.Sprints); err != nil {
		return nil, err
	}
	err = s.repo.ExecTx(ctx, func(tx Repository) error {
		created, err := tx.CreateSprint(ctx, moduleID, sp)
		if err != nil {
			return err
		}
		if err := tx.SyncModuleStartFromSprints(ctx, moduleID); err != nil {
			return err
		}
		return tx.InsertHistory(ctx, moduleID, callerID, "sprint", nil, stringPtr(sprintLabel(*created)))
	})
	if err != nil {
		return nil, err
	}
	return s.afterSprintChange(ctx, moduleID, callerID, false)
}

func (s *service) UpdateSprint(ctx context.Context, moduleID, sprintID, callerID string, req UpdateModuleSprintRequest) (*ModuleResponse, error) {
	cur, err := s.repo.GetSprint(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	if cur.ModuleID != moduleID {
		return nil, ErrSprintNotFound
	}
	m, err := s.moduleForSprint(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	next := *cur
	set := func(dst *string, v *string) {
		if v != nil {
			*dst = strings.TrimSpace(*v)
		}
	}
	set(&next.Name, req.Name)
	set(&next.Goal, req.Goal)
	set(&next.StartDate, req.StartDate)
	set(&next.EndDate, req.EndDate)
	set(&next.ReviewNote, req.ReviewNote)
	set(&next.RetroNote, req.RetroNote)
	if req.Status != nil {
		next.Status = *req.Status
	}
	if err := validateSprint(next, m.Sprints); err != nil {
		return nil, err
	}
	err = s.repo.ExecTx(ctx, func(tx Repository) error {
		updated, err := tx.UpdateSprint(ctx, next)
		if err != nil {
			return err
		}
		if err := tx.SyncModuleStartFromSprints(ctx, moduleID); err != nil {
			return err
		}
		return tx.InsertHistory(ctx, moduleID, callerID, "sprint", stringPtr(sprintLabel(*cur)), stringPtr(sprintLabel(*updated)))
	})
	if err != nil {
		return nil, err
	}
	return s.afterSprintChange(ctx, moduleID, callerID, cur.Status != SprintStatusActive && next.Status == SprintStatusActive)
}

func (s *service) DeleteSprint(ctx context.Context, moduleID, sprintID, callerID string) (*ModuleResponse, error) {
	cur, err := s.repo.GetSprint(ctx, sprintID)
	if err != nil {
		return nil, err
	}
	if cur.ModuleID != moduleID {
		return nil, ErrSprintNotFound
	}
	err = s.repo.ExecTx(ctx, func(tx Repository) error {
		if err := tx.DeleteSprint(ctx, sprintID); err != nil {
			return err
		}
		if err := tx.SyncModuleStartFromSprints(ctx, moduleID); err != nil {
			return err
		}
		return tx.InsertHistory(ctx, moduleID, callerID, "sprint", stringPtr(sprintLabel(*cur)), nil)
	})
	if err != nil {
		return nil, err
	}
	return s.afterSprintChange(ctx, moduleID, callerID, false)
}
