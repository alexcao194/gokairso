package workspace

import (
	"context"
	"errors"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/realtime"
)

var ErrForbidden = errors.New("Bạn không có quyền thực hiện thao tác này (vai trò không đủ)")

var ErrCannotManageEqualOrHigherRole = errors.New("chỉ được sửa vai trò của người có cấp thấp hơn hoặc bằng mình")

var ErrNotFound = errors.New("Không tìm thấy dữ liệu")

type Service interface {
	CreateWorkspace(ctx context.Context, userID string, req CreateWorkspaceRequest) (*WorkspaceResponse, error)
	ListWorkspaces(ctx context.Context, userID string, params pagination.Params) (pagination.Response[WorkspaceResponse], error)
	GetWorkspace(ctx context.Context, workspaceID string) (*WorkspaceResponse, error)
	UpdateWorkspace(ctx context.Context, workspaceID, callerID string, req UpdateWorkspaceRequest) (*WorkspaceResponse, error)
	ListWorkspaceHistory(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceHistoryResponse], error)
	DeleteWorkspace(ctx context.Context, workspaceID string) error

	ListWorkspaceMembers(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceMemberResponse], error)
	ListMyInvitations(ctx context.Context, userID string, params pagination.Params) (pagination.Response[WorkspaceInvitationSummaryResponse], error)
	InviteMember(ctx context.Context, workspaceID, callerID string, req InviteMemberRequest) (*WorkspaceInvitationResponse, error)
	AcceptInvitation(ctx context.Context, invitationID, callerID string) error
	RejectInvitation(ctx context.Context, invitationID, callerID string) error
	CancelInvitation(ctx context.Context, invitationID, callerID string) error
	RemoveMember(ctx context.Context, workspaceID, callerID, userID string) error
	UpdateMemberRole(ctx context.Context, workspaceID, callerID, userID string, role WorkspaceRole) (*WorkspaceMemberResponse, error)

	CreateWorkspaceLink(ctx context.Context, workspaceID, userID string, req CreateWorkspaceLinkRequest) (*WorkspaceLinkResponse, error)
	ListWorkspaceLinks(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceLinkResponse], error)
	DeleteWorkspaceLink(ctx context.Context, linkID string) error

	CreateWorkspaceNote(ctx context.Context, workspaceID string, userID string, req CreateWorkspaceNoteRequest) (*WorkspaceNoteResponse, error)
	ListWorkspaceNotes(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceNoteResponse], error)
	UpdateWorkspaceNote(ctx context.Context, noteID string, req UpdateWorkspaceNoteRequest) (*WorkspaceNoteResponse, error)
	DeleteWorkspaceNote(ctx context.Context, noteID string) error

	CreateWorkspaceFile(ctx context.Context, workspaceID string, userID string, req CreateWorkspaceFileRequest) (*WorkspaceFileResponse, error)
	ListWorkspaceFiles(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceFileResponse], error)
	DeleteWorkspaceFile(ctx context.Context, fileID string) error
}

type NotificationPublisher interface {
	PublishWorkspaceNotification(ctx context.Context, event WorkspaceNotificationEvent) error
}

type service struct {
	repo      Repository
	publisher NotificationPublisher
	realtime  realtime.Broadcaster
}

func NewService(repo Repository, publishers ...NotificationPublisher) *service {
	var publisher NotificationPublisher
	if len(publishers) > 0 {
		publisher = publishers[0]
	}
	return &service{repo: repo, publisher: publisher}
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

func (s *service) CreateWorkspace(ctx context.Context, userID string, req CreateWorkspaceRequest) (*WorkspaceResponse, error) {
	return s.repo.CreateWorkspace(ctx, userID, req)
}

func (s *service) ListWorkspaces(ctx context.Context, userID string, params pagination.Params) (pagination.Response[WorkspaceResponse], error) {
	return s.repo.ListWorkspaces(ctx, userID, params)
}

func (s *service) GetWorkspace(ctx context.Context, workspaceID string) (*WorkspaceResponse, error) {
	return s.repo.GetWorkspace(ctx, workspaceID)
}

func (s *service) UpdateWorkspace(ctx context.Context, workspaceID, callerID string, req UpdateWorkspaceRequest) (*WorkspaceResponse, error) {
	ws, err := s.repo.UpdateWorkspace(ctx, workspaceID, callerID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, workspaceID, "workspace.updated", workspaceID, callerID, ws)
	return ws, nil
}

func (s *service) ListWorkspaceHistory(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceHistoryResponse], error) {
	return s.repo.ListWorkspaceHistory(ctx, workspaceID, params)
}

func (s *service) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	if err := s.repo.DeleteWorkspace(ctx, workspaceID); err != nil {
		return err
	}
	s.broadcast(ctx, workspaceID, "workspace.deleted", workspaceID, "", nil)
	return nil
}

func (s *service) ListWorkspaceMembers(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceMemberResponse], error) {
	return s.repo.ListWorkspaceMembers(ctx, workspaceID, params)
}

func (s *service) ListMyInvitations(ctx context.Context, userID string, params pagination.Params) (pagination.Response[WorkspaceInvitationSummaryResponse], error) {
	return s.repo.ListMyInvitations(ctx, userID, params)
}

func (s *service) InviteMember(ctx context.Context, workspaceID, callerID string, req InviteMemberRequest) (*WorkspaceInvitationResponse, error) {
	role, err := s.repo.GetCallerRole(ctx, workspaceID, callerID)
	if err != nil {
		return nil, err
	}
	if !role.CanManageMembers() {
		return nil, ErrForbidden
	}
	if !role.CanManageRole(req.Role) {
		return nil, ErrCannotManageEqualOrHigherRole
	}
	inviteeID, err := s.repo.GetUserIDByAlias(ctx, req.UserAlias)
	if err != nil {
		return nil, ErrNotFound
	}
	invitation, err := s.repo.InviteMember(ctx, workspaceID, inviteeID, callerID, req.Role)
	if err != nil {
		return nil, err
	}
	if s.publisher != nil && invitation.InviteeID != callerID {
		workspace, err := s.repo.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		if err := s.publisher.PublishWorkspaceNotification(ctx, WorkspaceNotificationEvent{
			Type: NotificationTypeWorkspaceInvitation, RecipientID: invitation.InviteeID, WorkspaceID: invitation.WorkspaceID,
			WorkspaceName: workspace.Name, Role: invitation.Role,
		}); err != nil {
			return nil, err
		}
	}
	return invitation, nil
}

func (s *service) AcceptInvitation(ctx context.Context, invitationID, callerID string) error {
	return s.repo.AcceptInvitation(ctx, invitationID, callerID)
}

func (s *service) RejectInvitation(ctx context.Context, invitationID, callerID string) error {
	return s.repo.RejectInvitation(ctx, invitationID, callerID)
}

func (s *service) CancelInvitation(ctx context.Context, invitationID, callerID string) error {
	workspaceID, err := s.repo.GetWorkspaceIDByInvitationID(ctx, invitationID)
	if err != nil {
		return err
	}
	role, err := s.repo.GetCallerRole(ctx, workspaceID, callerID)
	if err != nil {
		return err
	}
	if !role.CanManageMembers() {
		return ErrForbidden
	}
	return s.repo.CancelInvitation(ctx, invitationID)
}

func (s *service) RemoveMember(ctx context.Context, workspaceID, callerID, userID string) error {
	role, err := s.repo.GetCallerRole(ctx, workspaceID, callerID)
	if err != nil {
		return err
	}
	if !role.CanManageMembers() {
		return ErrForbidden
	}
	targetRole, err := s.repo.GetCallerRole(ctx, workspaceID, userID)
	if err != nil {
		return err
	}
	if !role.CanManageRole(targetRole) {
		return ErrCannotManageEqualOrHigherRole
	}
	workspace, err := s.repo.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	if err := s.repo.RemoveMember(ctx, workspaceID, userID); err != nil {
		return err
	}
	s.broadcast(ctx, workspace.ID, "workspace.member.removed", userID, callerID, nil)
	if s.publisher != nil && userID != callerID {
		return s.publisher.PublishWorkspaceNotification(ctx, WorkspaceNotificationEvent{
			Type: NotificationTypeWorkspaceMemberRemoved, RecipientID: userID, WorkspaceID: workspace.ID,
			WorkspaceName: workspace.Name,
		})
	}
	return nil
}

func (s *service) UpdateMemberRole(ctx context.Context, workspaceID, callerID, userID string, role WorkspaceRole) (*WorkspaceMemberResponse, error) {
	callerRole, err := s.repo.GetCallerRole(ctx, workspaceID, callerID)
	if err != nil {
		return nil, err
	}
	if !callerRole.CanManageMembers() {
		return nil, ErrForbidden
	}
	targetRole, err := s.repo.GetCallerRole(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	if !callerRole.CanManageRole(targetRole) || !callerRole.CanManageRole(role) {
		return nil, ErrCannotManageEqualOrHigherRole
	}
	member, err := s.repo.UpdateMemberRole(ctx, workspaceID, userID, role)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, member.WorkspaceID, "workspace.member.updated", member.UserID, callerID, member)
	if s.publisher != nil && member.UserID != callerID {
		workspace, err := s.repo.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		if err := s.publisher.PublishWorkspaceNotification(ctx, WorkspaceNotificationEvent{
			Type: NotificationTypeWorkspaceMemberRoleUpdated, RecipientID: member.UserID, WorkspaceID: member.WorkspaceID,
			WorkspaceName: workspace.Name, Role: member.Role,
		}); err != nil {
			return nil, err
		}
	}
	return member, nil
}

func (s *service) CreateWorkspaceLink(ctx context.Context, workspaceID, userID string, req CreateWorkspaceLinkRequest) (*WorkspaceLinkResponse, error) {
	link, err := s.repo.CreateWorkspaceLink(ctx, workspaceID, userID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, workspaceID, "workspace.link.created", link.ID, "", link)
	return link, nil
}

func (s *service) ListWorkspaceLinks(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceLinkResponse], error) {
	return s.repo.ListWorkspaceLinks(ctx, workspaceID, params)
}

func (s *service) DeleteWorkspaceLink(ctx context.Context, linkID string) error {
	workspaceID, err := s.repo.DeleteWorkspaceLink(ctx, linkID)
	if err != nil {
		return err
	}
	s.broadcast(ctx, workspaceID, "workspace.link.deleted", linkID, "", nil)
	return nil
}

func (s *service) CreateWorkspaceNote(ctx context.Context, workspaceID string, userID string, req CreateWorkspaceNoteRequest) (*WorkspaceNoteResponse, error) {
	note, err := s.repo.CreateWorkspaceNote(ctx, workspaceID, userID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, workspaceID, "workspace.note.created", note.ID, "", note)
	return note, nil
}

func (s *service) ListWorkspaceNotes(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceNoteResponse], error) {
	return s.repo.ListWorkspaceNotes(ctx, workspaceID, params)
}

func (s *service) UpdateWorkspaceNote(ctx context.Context, noteID string, req UpdateWorkspaceNoteRequest) (*WorkspaceNoteResponse, error) {
	note, err := s.repo.UpdateWorkspaceNote(ctx, noteID, req)
	if err != nil {
		return nil, err
	}
	if note != nil {
		s.broadcast(ctx, note.WorkspaceID, "workspace.note.updated", note.ID, "", note)
	}
	return note, nil
}

func (s *service) DeleteWorkspaceNote(ctx context.Context, noteID string) error {
	workspaceID, err := s.repo.DeleteWorkspaceNote(ctx, noteID)
	if err != nil {
		return err
	}
	s.broadcast(ctx, workspaceID, "workspace.note.deleted", noteID, "", nil)
	return nil
}

func (s *service) CreateWorkspaceFile(ctx context.Context, workspaceID string, userID string, req CreateWorkspaceFileRequest) (*WorkspaceFileResponse, error) {
	file, err := s.repo.CreateWorkspaceFile(ctx, workspaceID, userID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, workspaceID, "workspace.file.created", file.ID, "", file)
	return file, nil
}

func (s *service) ListWorkspaceFiles(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceFileResponse], error) {
	return s.repo.ListWorkspaceFiles(ctx, workspaceID, params)
}

func (s *service) DeleteWorkspaceFile(ctx context.Context, fileID string) error {
	workspaceID, err := s.repo.DeleteWorkspaceFile(ctx, fileID)
	if err != nil {
		return err
	}
	s.broadcast(ctx, workspaceID, "workspace.file.deleted", fileID, "", nil)
	return nil
}
