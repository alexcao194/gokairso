package project

import (
	"context"
	"errors"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/realtime"
)

var ErrForbidden = errors.New("Bạn không có quyền thực hiện thao tác này (vai trò không đủ)")

var ErrCannotManageEqualOrHigherRole = errors.New("chỉ được sửa vai trò của người có cấp thấp hơn hoặc bằng mình")

type Service interface {
	CreateProject(ctx context.Context, workspaceID, userID string, req CreateProjectRequest) (*ProjectResponse, error)
	ListProjects(ctx context.Context, workspaceID, userID string, params pagination.StatsParams) (pagination.Response[ProjectResponse], error)
	GetProjectOverviewsByIDs(ctx context.Context, projectIDs []string) ([]ProjectOverviewResponse, error)
	UpdateProject(ctx context.Context, projectID, callerID string, req UpdateProjectRequest) (*ProjectResponse, error)
	ListProjectHistory(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectHistoryResponse], error)
	DeleteProject(ctx context.Context, projectID string) error

	ListProjectMembers(ctx context.Context, projectID string, params pagination.Params, includeAvailable bool) (*ListProjectMembersResponse, error)
	AddProjectMember(ctx context.Context, projectID, callerID, userID string, role ProjectRole) (*ProjectMemberResponse, error)
	UpdateProjectMemberRole(ctx context.Context, projectID, callerID, userID string, role ProjectRole) (*ProjectMemberResponse, error)
	RemoveProjectMember(ctx context.Context, projectID, callerID, userID string) error

	CreateProjectLink(ctx context.Context, projectID, userID string, req CreateProjectLinkRequest) (*ProjectLinkResponse, error)
	ListProjectLinks(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectLinkResponse], error)
	UpdateProjectLink(ctx context.Context, linkID string, req UpdateProjectLinkRequest) (*ProjectLinkResponse, error)
	DeleteProjectLink(ctx context.Context, linkID string) error

	CreateProjectNote(ctx context.Context, projectID string, userID string, req CreateProjectNoteRequest) (*ProjectNoteResponse, error)
	ListProjectNotes(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectNoteResponse], error)
	UpdateProjectNote(ctx context.Context, noteID string, req UpdateProjectNoteRequest) (*ProjectNoteResponse, error)
	DeleteProjectNote(ctx context.Context, noteID string) error

	CreateProjectFile(ctx context.Context, projectID string, userID string, req CreateProjectFileRequest) (*ProjectFileResponse, error)
	ListProjectFiles(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectFileResponse], error)
	DeleteProjectFile(ctx context.Context, fileID string) error
}

type NotificationPublisher interface {
	PublishProjectNotification(ctx context.Context, event ProjectNotificationEvent) error
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

func (s *service) CreateProject(ctx context.Context, workspaceID, userID string, req CreateProjectRequest) (*ProjectResponse, error) {
	project, err := s.repo.CreateProject(ctx, workspaceID, userID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, workspaceID, "project.created", project.ID, userID, project)
	return project, nil
}

func (s *service) ListProjects(ctx context.Context, workspaceID, userID string, params pagination.StatsParams) (pagination.Response[ProjectResponse], error) {
	return s.repo.ListProjects(ctx, workspaceID, userID, params)
}

func (s *service) GetProjectOverviewsByIDs(ctx context.Context, projectIDs []string) ([]ProjectOverviewResponse, error) {
	return s.repo.GetProjectOverviewsByIDs(ctx, projectIDs)
}

func (s *service) UpdateProject(ctx context.Context, projectID, callerID string, req UpdateProjectRequest) (*ProjectResponse, error) {
	project, err := s.repo.UpdateProject(ctx, projectID, callerID, req)
	if err != nil {
		return nil, err
	}
	pctx, err := s.repo.GetProjectContext(ctx, projectID)
	if err == nil {
		s.broadcast(ctx, pctx.WorkspaceID, "project.updated", projectID, callerID, project)
	}
	return project, nil
}

func (s *service) ListProjectHistory(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectHistoryResponse], error) {
	return s.repo.ListProjectHistory(ctx, projectID, params)
}

func (s *service) DeleteProject(ctx context.Context, projectID string) error {
	pctx, pctxErr := s.repo.GetProjectContext(ctx, projectID)
	if err := s.repo.DeleteProject(ctx, projectID); err != nil {
		return err
	}
	if pctxErr == nil {
		s.broadcast(ctx, pctx.WorkspaceID, "project.deleted", projectID, "", nil)
	}
	return nil
}

func (s *service) ListProjectMembers(ctx context.Context, projectID string, params pagination.Params, includeAvailable bool) (*ListProjectMembersResponse, error) {
	return s.repo.ListProjectMembers(ctx, projectID, params, includeAvailable)
}

func (s *service) AddProjectMember(ctx context.Context, projectID, callerID, userID string, role ProjectRole) (*ProjectMemberResponse, error) {
	pctx, err := s.repo.GetProjectContext(ctx, projectID)
	if err != nil {
		return nil, err
	}
	callerRole, err := s.repo.GetCallerWorkspaceRole(ctx, pctx.WorkspaceID, callerID)
	if err != nil {
		return nil, err
	}
	if !callerRole.CanManageMembers() {
		return nil, ErrForbidden
	}
	if !callerRole.CanManageRole(role) {
		return nil, ErrCannotManageEqualOrHigherRole
	}
	member, err := s.repo.AddProjectMember(ctx, projectID, userID, role)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, pctx.WorkspaceID, "project.member.added", member.UserID, callerID, member)
	if s.publisher != nil && member.UserID != callerID {
		if err := s.publisher.PublishProjectNotification(ctx, ProjectNotificationEvent{
			Type: NotificationTypeProjectMemberAdded, RecipientID: member.UserID, WorkspaceID: pctx.WorkspaceID,
			ProjectID: member.ProjectID, ProjectName: pctx.Name, Role: member.Role,
		}); err != nil {
			return nil, err
		}
	}
	return member, nil
}

func (s *service) UpdateProjectMemberRole(ctx context.Context, projectID, callerID, userID string, role ProjectRole) (*ProjectMemberResponse, error) {
	pctx, err := s.repo.GetProjectContext(ctx, projectID)
	if err != nil {
		return nil, err
	}
	callerRole, err := s.repo.GetCallerWorkspaceRole(ctx, pctx.WorkspaceID, callerID)
	if err != nil {
		return nil, err
	}
	if !callerRole.CanManageMembers() {
		return nil, ErrForbidden
	}
	targetRole, err := s.repo.GetProjectMemberRole(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !callerRole.CanManageRole(targetRole) || !callerRole.CanManageRole(role) {
		return nil, ErrCannotManageEqualOrHigherRole
	}
	member, err := s.repo.UpdateProjectMemberRole(ctx, projectID, userID, role)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, pctx.WorkspaceID, "project.member.updated", member.UserID, callerID, member)
	if s.publisher != nil && member.UserID != callerID {
		if err := s.publisher.PublishProjectNotification(ctx, ProjectNotificationEvent{
			Type: NotificationTypeProjectMemberRoleUpdated, RecipientID: member.UserID, WorkspaceID: pctx.WorkspaceID,
			ProjectID: member.ProjectID, ProjectName: pctx.Name, Role: member.Role,
		}); err != nil {
			return nil, err
		}
	}
	return member, nil
}

func (s *service) RemoveProjectMember(ctx context.Context, projectID, callerID, userID string) error {
	pctx, err := s.repo.GetProjectContext(ctx, projectID)
	if err != nil {
		return err
	}
	callerRole, err := s.repo.GetCallerWorkspaceRole(ctx, pctx.WorkspaceID, callerID)
	if err != nil {
		return err
	}
	if !callerRole.CanManageMembers() {
		return ErrForbidden
	}
	targetRole, err := s.repo.GetProjectMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !callerRole.CanManageRole(targetRole) {
		return ErrCannotManageEqualOrHigherRole
	}
	if err := s.repo.RemoveProjectMember(ctx, projectID, userID); err != nil {
		return err
	}
	s.broadcast(ctx, pctx.WorkspaceID, "project.member.removed", userID, callerID, nil)
	if s.publisher != nil && userID != callerID {
		return s.publisher.PublishProjectNotification(ctx, ProjectNotificationEvent{
			Type: NotificationTypeProjectMemberRemoved, RecipientID: userID, WorkspaceID: pctx.WorkspaceID, ProjectID: projectID, ProjectName: pctx.Name,
		})
	}
	return nil
}

func (s *service) CreateProjectLink(ctx context.Context, projectID, userID string, req CreateProjectLinkRequest) (*ProjectLinkResponse, error) {
	link, err := s.repo.CreateProjectLink(ctx, projectID, userID, req)
	if err != nil {
		return nil, err
	}
	if pctx, err := s.repo.GetProjectContext(ctx, projectID); err == nil {
		s.broadcast(ctx, pctx.WorkspaceID, "project.link.created", link.ID, "", link)
	}
	return link, nil
}

func (s *service) ListProjectLinks(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectLinkResponse], error) {
	return s.repo.ListProjectLinks(ctx, projectID, params)
}

func (s *service) UpdateProjectLink(ctx context.Context, linkID string, req UpdateProjectLinkRequest) (*ProjectLinkResponse, error) {
	link, err := s.repo.UpdateProjectLink(ctx, linkID, req)
	if err != nil {
		return nil, err
	}
	if pctx, err := s.repo.GetProjectContext(ctx, link.ProjectID); err == nil {
		s.broadcast(ctx, pctx.WorkspaceID, "project.link.updated", link.ID, "", link)
	}
	return link, nil
}

func (s *service) DeleteProjectLink(ctx context.Context, linkID string) error {
	projectID, err := s.repo.DeleteProjectLink(ctx, linkID)
	if err != nil {
		return err
	}
	if pctx, err := s.repo.GetProjectContext(ctx, projectID); err == nil {
		s.broadcast(ctx, pctx.WorkspaceID, "project.link.deleted", linkID, "", nil)
	}
	return nil
}

func (s *service) CreateProjectNote(ctx context.Context, projectID string, userID string, req CreateProjectNoteRequest) (*ProjectNoteResponse, error) {
	note, err := s.repo.CreateProjectNote(ctx, projectID, userID, req)
	if err != nil {
		return nil, err
	}
	if pctx, err := s.repo.GetProjectContext(ctx, projectID); err == nil {
		s.broadcast(ctx, pctx.WorkspaceID, "project.note.created", note.ID, "", note)
	}
	return note, nil
}

func (s *service) ListProjectNotes(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectNoteResponse], error) {
	return s.repo.ListProjectNotes(ctx, projectID, params)
}

func (s *service) UpdateProjectNote(ctx context.Context, noteID string, req UpdateProjectNoteRequest) (*ProjectNoteResponse, error) {
	note, err := s.repo.UpdateProjectNote(ctx, noteID, req)
	if err != nil {
		return nil, err
	}
	if note != nil {
		if pctx, err := s.repo.GetProjectContext(ctx, note.ProjectID); err == nil {
			s.broadcast(ctx, pctx.WorkspaceID, "project.note.updated", note.ID, "", note)
		}
	}
	return note, nil
}

func (s *service) DeleteProjectNote(ctx context.Context, noteID string) error {
	projectID, err := s.repo.DeleteProjectNote(ctx, noteID)
	if err != nil {
		return err
	}
	if pctx, err := s.repo.GetProjectContext(ctx, projectID); err == nil {
		s.broadcast(ctx, pctx.WorkspaceID, "project.note.deleted", noteID, "", nil)
	}
	return nil
}

func (s *service) CreateProjectFile(ctx context.Context, projectID string, userID string, req CreateProjectFileRequest) (*ProjectFileResponse, error) {
	file, err := s.repo.CreateProjectFile(ctx, projectID, userID, req)
	if err != nil {
		return nil, err
	}
	if pctx, err := s.repo.GetProjectContext(ctx, projectID); err == nil {
		s.broadcast(ctx, pctx.WorkspaceID, "project.file.created", file.ID, "", file)
	}
	return file, nil
}

func (s *service) ListProjectFiles(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectFileResponse], error) {
	return s.repo.ListProjectFiles(ctx, projectID, params)
}

func (s *service) DeleteProjectFile(ctx context.Context, fileID string) error {
	projectID, err := s.repo.DeleteProjectFile(ctx, fileID)
	if err != nil {
		return err
	}
	if pctx, err := s.repo.GetProjectContext(ctx, projectID); err == nil {
		s.broadcast(ctx, pctx.WorkspaceID, "project.file.deleted", fileID, "", nil)
	}
	return nil
}
