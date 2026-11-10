package meeting

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/internal/features/workspace"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/realtime"
	"github.com/google/uuid"
)

var (
	ErrForbidden = errors.New("Bạn không có quyền thực hiện thao tác này")
	ErrNotFound  = errors.New("Không tìm thấy biên bản họp")
	ErrInvalid   = errors.New("Dữ liệu biên bản họp không hợp lệ")
)

const (
	maxItems        = 500
	maxParticipants = 100
	maxTasksPerItem = 50
)

type Service interface {
	Create(ctx context.Context, workspaceID, callerID string, req CreateMeetingMinutesRequest) (*MeetingMinutesResponse, error)
	Get(ctx context.Context, id, callerID string) (*MeetingMinutesResponse, error)
	List(ctx context.Context, workspaceID, callerID string, filter ListMeetingMinutesFilter, params pagination.Params) (pagination.Response[MeetingMinutesResponse], error)
	Update(ctx context.Context, id, callerID string, req UpdateMeetingMinutesRequest) (*MeetingMinutesResponse, error)
	Delete(ctx context.Context, id, callerID string) error
}

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

func (s *service) broadcast(ctx context.Context, workspaceID, eventType, entityID, actorID string, entity any) {
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

func (s *service) callerRole(ctx context.Context, workspaceID, callerID string) (string, error) {
	role, err := s.repo.GetCallerRole(ctx, workspaceID, callerID)
	if err != nil {
		return "", err
	}
	if role == "" {
		return "", ErrForbidden
	}
	return role, nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

func validDate(v string) bool {
	_, err := time.Parse(time.DateOnly, v)
	return err == nil
}

func normalizeItems(in []MeetingItem) ([]MeetingItem, error) {
	if len(in) > maxItems {
		return nil, invalid("tối đa %d dòng", maxItems)
	}
	out := make([]MeetingItem, 0, len(in))
	for i, it := range in {
		it.Module = strings.TrimSpace(it.Module)
		it.Content = strings.TrimSpace(it.Content)
		it.Note = strings.TrimSpace(it.Note)
		if it.Content == "" && it.Module == "" && it.Note == "" && len(it.Tasks) == 0 {
			continue
		}
		if !it.Section.Valid() {
			return nil, invalid("dòng %d có mục không hợp lệ", i+1)
		}
		if it.Content == "" {
			return nil, invalid("dòng %d chưa có nội dung", i+1)
		}
		if it.Status != ItemStatusDone {
			it.Status = ItemStatusOpen
		}
		if it.ID == "" {
			it.ID = uuid.NewString()
		}
		tasks := make([]MeetingTaskRef, 0, len(it.Tasks))
		seen := map[string]bool{}
		for _, t := range it.Tasks {
			t.ID = strings.TrimSpace(t.ID)
			if t.ID == "" || seen[t.ID] {
				continue
			}
			seen[t.ID] = true
			tasks = append(tasks, MeetingTaskRef{ID: t.ID, Code: strings.TrimSpace(t.Code)})
		}
		if len(tasks) > maxTasksPerItem {
			return nil, invalid("dòng %d gắn quá %d task", i+1, maxTasksPerItem)
		}
		it.Tasks = tasks
		out = append(out, it)
	}
	return out, nil
}

func normalizeParticipants(in []MeetingParticipant) ([]MeetingParticipant, error) {
	if len(in) > maxParticipants {
		return nil, invalid("tối đa %d người/bộ phận", maxParticipants)
	}
	out := make([]MeetingParticipant, 0, len(in))
	seen := map[string]bool{}
	for _, p := range in {
		p.Name = strings.TrimSpace(p.Name)
		p.UserID = strings.TrimSpace(p.UserID)
		if p.Name == "" {
			continue
		}
		p.Kind = ParticipantGroup
		key := "g:" + strings.ToLower(p.Name)
		if p.UserID != "" {
			p.Kind = ParticipantUser
			key = "u:" + p.UserID
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, p)
	}
	return out, nil
}

func normalizeNotes(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if v = strings.TrimSpace(v); v != "" && MeetingSection(k).Valid() {
			out[k] = v
		}
	}
	return out
}

func trimPtr(v *string) {
	if v != nil {
		*v = strings.TrimSpace(*v)
	}
}

func (s *service) Create(ctx context.Context, workspaceID, callerID string, req CreateMeetingMinutesRequest) (*MeetingMinutesResponse, error) {
	if _, err := s.callerRole(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	req.Title = strings.TrimSpace(req.Title)
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.Title == "" {
		return nil, invalid("tiêu đề là bắt buộc")
	}
	if !validDate(req.MeetingDate) {
		return nil, invalid("ngày họp phải theo định dạng YYYY-MM-DD")
	}
	if req.ProjectID != "" {
		ok, err := s.repo.ProjectBelongsToWorkspace(ctx, req.ProjectID, workspaceID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNotFound
		}
	}
	items, err := normalizeItems(req.Items)
	if err != nil {
		return nil, err
	}
	req.Items = items
	if req.Attendees, err = normalizeParticipants(req.Attendees); err != nil {
		return nil, err
	}
	if req.Recorders, err = normalizeParticipants(req.Recorders); err != nil {
		return nil, err
	}
	req.SectionNotes = normalizeNotes(req.SectionNotes)
	req.Location = strings.TrimSpace(req.Location)
	req.Summary = strings.TrimSpace(req.Summary)

	resp, err := s.repo.Create(ctx, workspaceID, callerID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, resp.WorkspaceID, "meeting.created", resp.ID, callerID, resp)
	return resp, nil
}

func (s *service) Get(ctx context.Context, id, callerID string) (*MeetingMinutesResponse, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.callerRole(ctx, m.WorkspaceID, callerID); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *service) List(ctx context.Context, workspaceID, callerID string, filter ListMeetingMinutesFilter, params pagination.Params) (pagination.Response[MeetingMinutesResponse], error) {
	if _, err := s.callerRole(ctx, workspaceID, callerID); err != nil {
		return pagination.Response[MeetingMinutesResponse]{}, err
	}
	filter.Query = strings.TrimSpace(filter.Query)
	return s.repo.List(ctx, workspaceID, filter, params)
}

func (s *service) Update(ctx context.Context, id, callerID string, req UpdateMeetingMinutesRequest) (*MeetingMinutesResponse, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.callerRole(ctx, m.WorkspaceID, callerID); err != nil {
		return nil, err
	}
	trimPtr(req.Title)
	trimPtr(req.Location)
	trimPtr(req.Summary)
	if req.Title != nil && *req.Title == "" {
		return nil, invalid("tiêu đề là bắt buộc")
	}
	if req.MeetingDate != nil && !validDate(*req.MeetingDate) {
		return nil, invalid("ngày họp phải theo định dạng YYYY-MM-DD")
	}
	if req.Items != nil {
		items, err := normalizeItems(*req.Items)
		if err != nil {
			return nil, err
		}
		req.Items = &items
	}
	for _, ps := range []*[]MeetingParticipant{req.Attendees, req.Recorders} {
		if ps == nil {
			continue
		}
		norm, err := normalizeParticipants(*ps)
		if err != nil {
			return nil, err
		}
		*ps = norm
	}
	if req.SectionNotes != nil {
		notes := normalizeNotes(*req.SectionNotes)
		req.SectionNotes = &notes
	}
	resp, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, resp.WorkspaceID, "meeting.updated", resp.ID, callerID, resp)
	return resp, nil
}

func (s *service) Delete(ctx context.Context, id, callerID string) error {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	role, err := s.callerRole(ctx, m.WorkspaceID, callerID)
	if err != nil {
		return err
	}
	if m.CreatedBy.ID != callerID && role != string(workspace.WorkspaceRoleAdmin) && role != string(workspace.WorkspaceRoleHolder) {
		return fmt.Errorf("%w: chỉ người tạo hoặc quản trị workspace mới được xoá biên bản", ErrForbidden)
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.broadcast(ctx, m.WorkspaceID, "meeting.deleted", id, callerID, nil)
	return nil
}
