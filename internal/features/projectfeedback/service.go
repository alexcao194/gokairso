package projectfeedback

import (
	"context"
	"errors"
	"time"

	"github.com/alexcao194/kairso/pkg/pagination"
)

var (
	ErrForbidden = errors.New("Bạn không có quyền thực hiện thao tác này")
	ErrDisabled  = errors.New("Dự án này chưa bật trang feedback công khai")
)

type Service interface {
	GetFeedbackSettings(ctx context.Context, projectID, callerID string) (*FeedbackSettingsResponse, error)
	EnableFeedbackSettings(ctx context.Context, projectID, callerID string) (*FeedbackSettingsResponse, error)
	DisableFeedbackSettings(ctx context.Context, projectID, callerID string) (*FeedbackSettingsResponse, error)
	RegenerateFeedbackToken(ctx context.Context, projectID, callerID string) (*FeedbackSettingsResponse, error)
	UpdateShowTimeline(ctx context.Context, projectID, callerID string, show bool) (*FeedbackSettingsResponse, error)

	ListFeedbacks(ctx context.Context, projectID, callerID string, filter ListFeedbacksFilter, params pagination.Params) (ProjectFeedbackListResponse, error)
	ListFeedbacksForTask(ctx context.Context, taskID, callerID string) ([]ProjectFeedbackResponse, error)
	GetFeedbackSummary(ctx context.Context, projectID, callerID string) (*FeedbackSummaryResponse, error)
	CreateFeedback(ctx context.Context, projectID, callerID string, req CreateProjectFeedbackRequest) (*ProjectFeedbackResponse, error)
	UpdateFeedback(ctx context.Context, feedbackID, callerID string, req UpdateProjectFeedbackRequest) (*ProjectFeedbackResponse, error)
	UpdateFeedbackStatus(ctx context.Context, feedbackID, callerID string, status FeedbackStatus) (*ProjectFeedbackResponse, error)
	RespondToFeedback(ctx context.Context, feedbackID, callerID, response string, expectedFixDate *time.Time) (*ProjectFeedbackResponse, error)
	LinkTask(ctx context.Context, feedbackID, taskID, callerID string) error
	UnlinkTask(ctx context.Context, feedbackID, taskID, callerID string) error

	GetPublicProject(ctx context.Context, token string) (*PublicProjectResponse, error)
	ListPublicFeedbacks(ctx context.Context, token string, filter ListFeedbacksFilter, params pagination.Params) (ProjectFeedbackListResponse, error)
	SubmitPublicFeedback(ctx context.Context, token string, req CreateProjectFeedbackRequest) (*ProjectFeedbackResponse, error)
	ListPublicModules(ctx context.Context, token string) ([]PublicModuleResponse, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) GetFeedbackSettings(ctx context.Context, projectID, callerID string) (*FeedbackSettingsResponse, error) {
	allowed, err := s.repo.CheckProjectManageAuth(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.GetFeedbackSettings(ctx, projectID)
}

func (s *service) EnableFeedbackSettings(ctx context.Context, projectID, callerID string) (*FeedbackSettingsResponse, error) {
	allowed, err := s.repo.CheckProjectManageAuth(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.EnableFeedbackSettings(ctx, projectID)
}

func (s *service) DisableFeedbackSettings(ctx context.Context, projectID, callerID string) (*FeedbackSettingsResponse, error) {
	allowed, err := s.repo.CheckProjectManageAuth(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.DisableFeedbackSettings(ctx, projectID)
}

func (s *service) RegenerateFeedbackToken(ctx context.Context, projectID, callerID string) (*FeedbackSettingsResponse, error) {
	allowed, err := s.repo.CheckProjectManageAuth(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.RegenerateFeedbackToken(ctx, projectID)
}

func (s *service) UpdateShowTimeline(ctx context.Context, projectID, callerID string, show bool) (*FeedbackSettingsResponse, error) {
	allowed, err := s.repo.CheckProjectManageAuth(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.UpdateShowTimeline(ctx, projectID, show)
}

func (s *service) ListFeedbacks(ctx context.Context, projectID, callerID string, filter ListFeedbacksFilter, params pagination.Params) (ProjectFeedbackListResponse, error) {
	var empty ProjectFeedbackListResponse
	allowed, err := s.repo.CheckProjectMembership(ctx, projectID, callerID)
	if err != nil {
		return empty, err
	}
	if !allowed {
		return empty, ErrForbidden
	}
	return s.repo.ListFeedbacks(ctx, projectID, filter, params)
}

func (s *service) ListFeedbacksForTask(ctx context.Context, taskID, callerID string) ([]ProjectFeedbackResponse, error) {
	projectID, err := s.repo.GetTaskProjectID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	allowed, err := s.repo.CheckProjectMembership(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.ListFeedbacksForTask(ctx, taskID)
}

func (s *service) GetFeedbackSummary(ctx context.Context, projectID, callerID string) (*FeedbackSummaryResponse, error) {
	allowed, err := s.repo.CheckProjectMembership(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.GetFeedbackSummary(ctx, projectID)
}

func (s *service) CreateFeedback(ctx context.Context, projectID, callerID string, req CreateProjectFeedbackRequest) (*ProjectFeedbackResponse, error) {
	allowed, err := s.repo.CheckProjectMembership(ctx, projectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	if err := normalizeCreateFeedbackRequest(&req); err != nil {
		return nil, err
	}
	return s.repo.CreateFeedback(ctx, projectID, req)
}

func (s *service) UpdateFeedback(ctx context.Context, feedbackID, callerID string, req UpdateProjectFeedbackRequest) (*ProjectFeedbackResponse, error) {
	feedback, err := s.repo.GetFeedback(ctx, feedbackID)
	if err != nil {
		return nil, err
	}
	allowed, err := s.repo.CheckProjectMembership(ctx, feedback.ProjectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.UpdateFeedback(ctx, feedbackID, req)
}

func (s *service) UpdateFeedbackStatus(ctx context.Context, feedbackID, callerID string, status FeedbackStatus) (*ProjectFeedbackResponse, error) {
	feedback, err := s.repo.GetFeedback(ctx, feedbackID)
	if err != nil {
		return nil, err
	}
	allowed, err := s.repo.CheckProjectMembership(ctx, feedback.ProjectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.UpdateFeedbackStatus(ctx, feedbackID, status)
}

func (s *service) RespondToFeedback(ctx context.Context, feedbackID, callerID, response string, expectedFixDate *time.Time) (*ProjectFeedbackResponse, error) {
	feedback, err := s.repo.GetFeedback(ctx, feedbackID)
	if err != nil {
		return nil, err
	}
	allowed, err := s.repo.CheckProjectMembership(ctx, feedback.ProjectID, callerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	return s.repo.UpdateFeedbackResponse(ctx, feedbackID, callerID, response, expectedFixDate)
}

func (s *service) LinkTask(ctx context.Context, feedbackID, taskID, callerID string) error {
	feedback, err := s.repo.GetFeedback(ctx, feedbackID)
	if err != nil {
		return err
	}
	allowed, err := s.repo.CheckProjectMembership(ctx, feedback.ProjectID, callerID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return s.repo.LinkTask(ctx, feedbackID, taskID)
}

func (s *service) UnlinkTask(ctx context.Context, feedbackID, taskID, callerID string) error {
	feedback, err := s.repo.GetFeedback(ctx, feedbackID)
	if err != nil {
		return err
	}
	allowed, err := s.repo.CheckProjectMembership(ctx, feedback.ProjectID, callerID)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrForbidden
	}
	return s.repo.UnlinkTask(ctx, feedbackID, taskID)
}

func (s *service) GetPublicProject(ctx context.Context, token string) (*PublicProjectResponse, error) {
	project, _, err := s.repo.GetPublicProjectByToken(ctx, token)
	return project, err
}

func (s *service) ListPublicFeedbacks(ctx context.Context, token string, filter ListFeedbacksFilter, params pagination.Params) (ProjectFeedbackListResponse, error) {
	var empty ProjectFeedbackListResponse
	project, projectID, err := s.repo.GetPublicProjectByToken(ctx, token)
	if err != nil {
		return empty, err
	}
	if !project.Enabled {
		return empty, ErrDisabled
	}
	return s.repo.ListFeedbacks(ctx, projectID, filter, params)
}

func (s *service) SubmitPublicFeedback(ctx context.Context, token string, req CreateProjectFeedbackRequest) (*ProjectFeedbackResponse, error) {
	project, projectID, err := s.repo.GetPublicProjectByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !project.Enabled {
		return nil, ErrDisabled
	}
	if err := normalizeCreateFeedbackRequest(&req); err != nil {
		return nil, err
	}
	return s.repo.CreateFeedback(ctx, projectID, req)
}

func (s *service) ListPublicModules(ctx context.Context, token string) ([]PublicModuleResponse, error) {
	project, projectID, err := s.repo.GetPublicProjectByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if !project.Enabled {
		return nil, ErrDisabled
	}
	if !project.ShowModuleTimeline {
		return []PublicModuleResponse{}, nil
	}
	return s.repo.ListPublicModules(ctx, projectID)
}
