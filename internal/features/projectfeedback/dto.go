package projectfeedback

import (
	"fmt"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
)

type ListFeedbacksFilter struct {
	Priorities    []int32
	HasResponse   *bool
	Statuses      []FeedbackStatus
	FeedbackTypes []FeedbackType
	Query         string
	SortRules     []FeedbackSortRule
}

type FeedbackListStats struct {
	TotalCount int `json:"totalCount"`
}

type ProjectFeedbackListResponse struct {
	pagination.Response[ProjectFeedbackResponse]
	Stats FeedbackListStats `json:"stats"`
}

type UserSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Alias  string `json:"alias"`
	Avatar string `json:"avatar,omitempty"`
}

type LinkedTaskSummary struct {
	Code   string `json:"code"`
	Status string `json:"status"`
}

type FeedbackStatus string

const (
	FeedbackStatusPending      FeedbackStatus = "pending"
	FeedbackStatusDiscussing   FeedbackStatus = "discussing"
	FeedbackStatusAcknowledged FeedbackStatus = "acknowledged"
	FeedbackStatusInProgress   FeedbackStatus = "in_progress"
	FeedbackStatusResolved     FeedbackStatus = "resolved"
	FeedbackStatusDeployed     FeedbackStatus = "deployed"
	FeedbackStatusCanceled     FeedbackStatus = "canceled"
)

func (s FeedbackStatus) Valid() bool {
	switch s {
	case FeedbackStatusPending, FeedbackStatusDiscussing,
		FeedbackStatusAcknowledged, FeedbackStatusInProgress,
		FeedbackStatusResolved, FeedbackStatusDeployed,
		FeedbackStatusCanceled:
		return true
	default:
		return false
	}
}

type FeedbackType string

const (
	FeedbackTypeFeature     FeedbackType = "feature"
	FeedbackTypeImprovement FeedbackType = "improvement"
	FeedbackTypeBug         FeedbackType = "bug"
)

func (t FeedbackType) Valid() bool {
	switch t {
	case FeedbackTypeFeature, FeedbackTypeImprovement, FeedbackTypeBug:
		return true
	default:
		return false
	}
}

type CreateProjectFeedbackRequest struct {
	SubmitterName    string   `json:"submitterName"`
	SubmitterCompany string   `json:"submitterCompany"`
	FeatureName      string   `json:"featureName"`
	Content          string   `json:"content"`
	Priority         *int     `json:"priority"`
	FeedbackType     string   `json:"feedbackType"`
	Images           []string `json:"images,omitempty"`
}

type UpdateProjectFeedbackRequest struct {
	SubmitterName    string   `json:"submitterName"`
	SubmitterCompany string   `json:"submitterCompany"`
	FeatureName      string   `json:"featureName"`
	Content          string   `json:"content"`
	Priority         *int     `json:"priority"`
	FeedbackType     string   `json:"feedbackType"`
	Images           []string `json:"images,omitempty"`
}

type UpdateFeedbackStatusRequest struct {
	Status FeedbackStatus `json:"status"`
}

type UpdateFeedbackResponseRequest struct {
	Response        string     `json:"response"`
	ExpectedFixDate *time.Time `json:"expectedFixDate,omitempty" swaggertype:"string" format:"date-time"`
}

type FeedbackSummaryResponse struct {
	PendingCount int `json:"pendingCount"`
}

type ProjectFeedbackResponse struct {
	ID               string              `json:"id"`
	CreatedAt        time.Time           `json:"createdAt" swaggertype:"string" format:"date-time"`
	ProjectID        string              `json:"projectId"`
	SubmitterName    string              `json:"submitterName"`
	SubmitterCompany string              `json:"submitterCompany"`
	FeatureName      string              `json:"featureName"`
	Content          string              `json:"content"`
	Priority         int                 `json:"priority"`
	Status           FeedbackStatus      `json:"status"`
	FeedbackType     FeedbackType        `json:"feedbackType"`
	Images           []string            `json:"images"`
	Response         string              `json:"response,omitempty"`
	ExpectedFixDate  *time.Time          `json:"expectedFixDate,omitempty" swaggertype:"string" format:"date-time"`
	RespondedAt      *time.Time          `json:"respondedAt,omitempty" swaggertype:"string" format:"date-time"`
	RespondedBy      *UserSummary        `json:"respondedBy,omitempty"`
	Tasks            []LinkedTaskSummary `json:"tasks"`
	SequenceNo       int                 `json:"-"`
	Code             string              `json:"code"`
}

type FeedbackSettingsResponse struct {
	Enabled            bool   `json:"enabled"`
	Token              string `json:"token,omitempty"`
	ShowModuleTimeline bool   `json:"showModuleTimeline"`
}

type UpdateFeedbackSettingsRequest struct {
	ShowModuleTimeline *bool `json:"showModuleTimeline,omitempty"`
}

type PublicProjectResponse struct {
	ProjectName        string `json:"projectName"`
	Enabled            bool   `json:"enabled"`
	ShowModuleTimeline bool   `json:"showModuleTimeline"`
}

type PublicModuleResponse struct {
	Name       string                 `json:"name"`
	Status     string                 `json:"status" enums:"backlog,in_progress,done,released"`
	StartDate  *time.Time             `json:"startDate,omitempty" swaggertype:"string" format:"date-time"`
	EndDate    *time.Time             `json:"endDate,omitempty" swaggertype:"string" format:"date-time"`
	ReleasedAt *time.Time             `json:"releasedAt,omitempty" swaggertype:"string" format:"date-time"`
	Sprints    []PublicSprintResponse `json:"sprints"`
}

type PublicSprintResponse struct {
	Name      string `json:"name"`
	StartDate string `json:"startDate" swaggertype:"string" format:"date"`
	EndDate   string `json:"endDate" swaggertype:"string" format:"date"`
	Status    string `json:"status" enums:"planned,active,completed"`
}

func timePtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func publicModulesFromDatabase(modules []database.ListPublicModulesByProjectIDRow, sprints []database.ListPublicSprintsByModuleIDsRow) []PublicModuleResponse {
	byModule := make(map[[16]byte][]PublicSprintResponse, len(modules))
	for _, sp := range sprints {
		byModule[sp.ModuleID.Bytes] = append(byModule[sp.ModuleID.Bytes], PublicSprintResponse{
			Name:      sp.Name,
			StartDate: sp.StartDate.Time.Format(time.DateOnly),
			EndDate:   sp.EndDate.Time.Format(time.DateOnly),
			Status:    string(sp.Status),
		})
	}
	out := make([]PublicModuleResponse, 0, len(modules))
	for _, m := range modules {
		list := byModule[m.ID.Bytes]
		if list == nil {
			list = []PublicSprintResponse{}
		}
		out = append(out, PublicModuleResponse{
			Name:       m.Name,
			Status:     string(m.ScrumStatus),
			StartDate:  timePtr(m.StartDate),
			EndDate:    timePtr(m.EndDate),
			ReleasedAt: timePtr(m.ReleasedAt),
			Sprints:    list,
		})
	}
	return out
}

func normalizeCreateFeedbackRequest(req *CreateProjectFeedbackRequest) error {
	req.SubmitterName = strings.TrimSpace(req.SubmitterName)
	req.SubmitterCompany = strings.TrimSpace(req.SubmitterCompany)
	req.FeatureName = strings.TrimSpace(req.FeatureName)
	req.Content = strings.TrimSpace(req.Content)
	if req.SubmitterName == "" {
		return fmt.Errorf("submitterName là bắt buộc")
	}
	if req.SubmitterCompany == "" {
		return fmt.Errorf("submitterCompany là bắt buộc")
	}
	if req.FeatureName == "" {
		return fmt.Errorf("featureName là bắt buộc")
	}
	if req.Content == "" {
		return fmt.Errorf("Nội dung là bắt buộc")
	}
	if req.Priority == nil {
		return fmt.Errorf("priority là bắt buộc")
	}
	if *req.Priority < 1 || *req.Priority > 4 {
		return fmt.Errorf("priority phải là 1 (thấp), 2 (vừa), 3 (cao) hoặc 4 (gấp)")
	}
	if req.FeedbackType == "" {
		return fmt.Errorf("feedbackType là bắt buộc")
	}
	if !FeedbackType(req.FeedbackType).Valid() {
		return fmt.Errorf("feedbackType phải là feature, improvement hoặc bug")
	}
	return nil
}

func normalizeUpdateFeedbackRequest(req *UpdateProjectFeedbackRequest) error {
	req.SubmitterName = strings.TrimSpace(req.SubmitterName)
	req.SubmitterCompany = strings.TrimSpace(req.SubmitterCompany)
	req.FeatureName = strings.TrimSpace(req.FeatureName)
	req.Content = strings.TrimSpace(req.Content)
	if req.SubmitterName == "" {
		return fmt.Errorf("submitterName là bắt buộc")
	}
	if req.SubmitterCompany == "" {
		return fmt.Errorf("submitterCompany là bắt buộc")
	}
	if req.FeatureName == "" {
		return fmt.Errorf("featureName là bắt buộc")
	}
	if req.Content == "" {
		return fmt.Errorf("Nội dung là bắt buộc")
	}
	if req.Priority == nil {
		return fmt.Errorf("priority là bắt buộc")
	}
	if *req.Priority < 1 || *req.Priority > 4 {
		return fmt.Errorf("priority phải là 1 (thấp), 2 (vừa), 3 (cao) hoặc 4 (gấp)")
	}
	if req.FeedbackType == "" {
		return fmt.Errorf("feedbackType là bắt buộc")
	}
	if !FeedbackType(req.FeedbackType).Valid() {
		return fmt.Errorf("feedbackType phải là feature, improvement hoặc bug")
	}
	var cleanImages []string
	for _, img := range req.Images {
		if u := strings.TrimSpace(img); u != "" {
			cleanImages = append(cleanImages, u)
		}
	}
	req.Images = cleanImages
	return nil
}

func feedbackResponseFromRow(row feedbackRow) ProjectFeedbackResponse {
	return ProjectFeedbackResponse{
		ID:               pgutil.UUIDToString(row.ID),
		CreatedAt:        row.CreatedAt,
		ProjectID:        pgutil.UUIDToString(row.ProjectID),
		SubmitterName:    row.SubmitterName,
		SubmitterCompany: row.SubmitterCompany,
		FeatureName:      row.FeatureName,
		Content:          row.Content,
		Priority:         int(row.Priority),
		Status:           FeedbackStatus(row.Status),
		FeedbackType:     FeedbackType(row.FeedbackType),
		Response:         textString(row.Response),
		ExpectedFixDate:  timestamptzPtr(row.ExpectedFixDate),
		RespondedAt:      timestamptzPtr(row.RespondedAt),
		Images:           []string{},
		Tasks:            []LinkedTaskSummary{},
		SequenceNo:       int(row.SequenceNo),
	}
}

type feedbackRow struct {
	ID               pgtype.UUID
	CreatedAt        time.Time
	ProjectID        pgtype.UUID
	SubmitterName    string
	SubmitterCompany string
	FeatureName      string
	Content          string
	Priority         int32
	Status           string
	FeedbackType     string
	Response         pgtype.Text
	ExpectedFixDate  pgtype.Timestamptz
	RespondedAt      pgtype.Timestamptz
	RespondedByID    pgtype.UUID
	SequenceNo       int32
}

func feedbackRowFromCreate(row database.CreateProjectFeedbackRow) feedbackRow {
	return feedbackRow{
		ID: row.ID, CreatedAt: row.CreatedAt.Time, ProjectID: row.ProjectID,
		SubmitterName: row.SubmitterName, SubmitterCompany: row.SubmitterCompany,
		FeatureName: row.FeatureName, Content: row.Content, Priority: row.Priority, Status: row.Status,
		FeedbackType: row.FeedbackType,
		Response:     row.Response, ExpectedFixDate: row.ExpectedFixDate, RespondedAt: row.RespondedAt, RespondedByID: row.RespondedByID,
		SequenceNo: row.SequenceNo,
	}
}

func feedbackRowFromListByTask(row database.ListFeedbacksByTaskIDRow) feedbackRow {
	return feedbackRow{
		ID: row.ID, CreatedAt: row.CreatedAt.Time, ProjectID: row.ProjectID,
		SubmitterName: row.SubmitterName, SubmitterCompany: row.SubmitterCompany,
		FeatureName: row.FeatureName, Content: row.Content, Priority: row.Priority, Status: row.Status,
		FeedbackType: row.FeedbackType,
		Response:     row.Response, ExpectedFixDate: row.ExpectedFixDate, RespondedAt: row.RespondedAt, RespondedByID: row.RespondedByID,
		SequenceNo: row.SequenceNo,
	}
}

func feedbackRowFromGet(row database.GetProjectFeedbackByIDRow) feedbackRow {
	return feedbackRow{
		ID: row.ID, CreatedAt: row.CreatedAt.Time, ProjectID: row.ProjectID,
		SubmitterName: row.SubmitterName, SubmitterCompany: row.SubmitterCompany,
		FeatureName: row.FeatureName, Content: row.Content, Priority: row.Priority, Status: row.Status,
		FeedbackType: row.FeedbackType,
		Response:     row.Response, ExpectedFixDate: row.ExpectedFixDate, RespondedAt: row.RespondedAt, RespondedByID: row.RespondedByID,
		SequenceNo: row.SequenceNo,
	}
}

func feedbackRowFromUpdate(row database.UpdateProjectFeedbackResponseRow) feedbackRow {
	return feedbackRow{
		ID: row.ID, CreatedAt: row.CreatedAt.Time, ProjectID: row.ProjectID,
		SubmitterName: row.SubmitterName, SubmitterCompany: row.SubmitterCompany,
		FeatureName: row.FeatureName, Content: row.Content, Priority: row.Priority, Status: row.Status,
		FeedbackType: row.FeedbackType,
		Response:     row.Response, ExpectedFixDate: row.ExpectedFixDate, RespondedAt: row.RespondedAt, RespondedByID: row.RespondedByID,
		SequenceNo: row.SequenceNo,
	}
}

func feedbackRowFromUpdateFeedback(row database.UpdateProjectFeedbackRow) feedbackRow {
	return feedbackRow{
		ID: row.ID, CreatedAt: row.CreatedAt.Time, ProjectID: row.ProjectID,
		SubmitterName: row.SubmitterName, SubmitterCompany: row.SubmitterCompany,
		FeatureName: row.FeatureName, Content: row.Content, Priority: row.Priority, Status: row.Status,
		FeedbackType: row.FeedbackType,
		Response:     row.Response, ExpectedFixDate: row.ExpectedFixDate, RespondedAt: row.RespondedAt, RespondedByID: row.RespondedByID,
		SequenceNo: row.SequenceNo,
	}
}

func feedbackRowFromUpdateStatus(row database.UpdateProjectFeedbackStatusRow) feedbackRow {
	return feedbackRow{
		ID: row.ID, CreatedAt: row.CreatedAt.Time, ProjectID: row.ProjectID,
		SubmitterName: row.SubmitterName, SubmitterCompany: row.SubmitterCompany,
		FeatureName: row.FeatureName, Content: row.Content, Priority: row.Priority, Status: row.Status,
		FeedbackType: row.FeedbackType,
		Response:     row.Response, ExpectedFixDate: row.ExpectedFixDate, RespondedAt: row.RespondedAt, RespondedByID: row.RespondedByID,
		SequenceNo: row.SequenceNo,
	}
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}
