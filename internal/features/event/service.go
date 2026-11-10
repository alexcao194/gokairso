package event

import (
	"context"
	"errors"

	"github.com/alexcao194/kairso/internal/features/workspace"
	"github.com/alexcao194/kairso/pkg/pagination"
)

var ErrForbidden = errors.New("Bạn không có quyền thực hiện thao tác này")

type Service interface {
	ListEvents(ctx context.Context, workspaceAlias, callerID string, filter CalendarEventFilter) ([]CalendarDayResponse, error)
	ListMembersEvents(ctx context.Context, workspaceAlias, callerID string, filter CalendarEventFilter, params pagination.Params) (pagination.Response[MemberCalendarResponse], error)
}

type Repository interface {
	GetCallerRole(ctx context.Context, workspaceAlias, callerID string) (workspace.WorkspaceRole, error)
	ListEvents(ctx context.Context, workspaceAlias string, filter CalendarEventFilter) ([]CalendarDayResponse, error)
	ListMembersEvents(ctx context.Context, workspaceAlias string, filter CalendarEventFilter) ([]MemberCalendarResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) ListEvents(ctx context.Context, workspaceAlias, callerID string, filter CalendarEventFilter) ([]CalendarDayResponse, error) {
	role, err := s.repo.GetCallerRole(ctx, workspaceAlias, callerID)
	if err != nil {
		return nil, err
	}
	if role == "" {
		return nil, ErrForbidden
	}
	if filter.MemberID == "" {
		filter.MemberID = callerID
	}
	if filter.MemberID != callerID {
		if !roleCanViewAll(role) {
			return nil, ErrForbidden
		}
	}
	return s.repo.ListEvents(ctx, workspaceAlias, filter)
}

func (s *service) ListMembersEvents(ctx context.Context, workspaceAlias, callerID string, filter CalendarEventFilter, params pagination.Params) (pagination.Response[MemberCalendarResponse], error) {
	var empty pagination.Response[MemberCalendarResponse]
	role, err := s.repo.GetCallerRole(ctx, workspaceAlias, callerID)
	if err != nil {
		return empty, err
	}
	if !roleCanViewAll(role) {
		return empty, ErrForbidden
	}
	all, err := s.repo.ListMembersEvents(ctx, workspaceAlias, filter)
	if err != nil {
		return empty, err
	}
	offset := params.Offset()
	if offset >= len(all) {
		return pagination.NewResponse([]MemberCalendarResponse{}, false, params.Page, params.Limit), nil
	}
	all = all[offset:]
	hasMore := pagination.Process(&all, params.Limit)
	return pagination.NewResponse(all, hasMore, params.Page, params.Limit), nil
}

func roleCanViewAll(role workspace.WorkspaceRole) bool {
	return role == workspace.WorkspaceRoleHolder || role == workspace.WorkspaceRoleAdmin
}
