package project

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/notification"
	"github.com/alexcao194/kairso/internal/i18n"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
)

type ProjectRole string

const (
	ProjectRoleHolder ProjectRole = "holder"
	ProjectRoleAdmin  ProjectRole = "admin"
	ProjectRoleMember ProjectRole = "member"

	NotificationTypeProjectMemberAdded       = "project_member_added"
	NotificationTypeProjectMemberRoleUpdated = "project_member_role_updated"
	NotificationTypeProjectMemberRemoved     = "project_member_removed"
)

func (r ProjectRole) Valid() bool {
	switch r {
	case ProjectRoleHolder, ProjectRoleAdmin, ProjectRoleMember:
		return true
	default:
		return false
	}
}

func (r ProjectRole) CanManageMembers() bool {
	return r == ProjectRoleHolder || r == ProjectRoleAdmin
}

func (r ProjectRole) rank() int {
	switch r {
	case ProjectRoleHolder:
		return 3
	case ProjectRoleAdmin:
		return 2
	default:
		return 1
	}
}

func (r ProjectRole) CanManageRole(target ProjectRole) bool {
	return r.rank() >= target.rank()
}

type CreateProjectRequest struct {
	Name string `json:"name"`
}

type UpdateProjectRequest struct {
	Name      *string `json:"name,omitempty"`
	Avatar    *string `json:"avatar,omitempty"`
	NameSet   bool    `json:"-"`
	AvatarSet bool    `json:"-"`
}

type AddProjectMemberRequest struct {
	UserID string      `json:"userId"`
	Role   ProjectRole `json:"role"`
}

type UpdateProjectMemberRoleRequest struct {
	Role ProjectRole `json:"role"`
}

type ProjectNotificationEvent struct {
	Type        string
	RecipientID string
	WorkspaceID string
	ProjectID   string
	ProjectName string
	Role        ProjectRole
}

type ProjectResponse struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt       time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
	Name            string    `json:"name"`
	WorkspaceID     string    `json:"workspaceId"`
	Avatar          string    `json:"avatar,omitempty"`
	TaskCount       int       `json:"taskCount"`
	TotalEstimation int       `json:"totalEstimation"`
}

type HistoryChangedBy struct {
	ID     string `json:"id"`
	Alias  string `json:"alias"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type ProjectHistoryResponse struct {
	ID        string           `json:"id"`
	ChangedBy HistoryChangedBy `json:"changedBy"`
	Field     string           `json:"field"`
	OldValue  *string          `json:"oldValue,omitempty"`
	NewValue  *string          `json:"newValue,omitempty"`
	CreatedAt time.Time        `json:"createdAt" swaggertype:"string" format:"date-time"`
}

type ProjectTaskStatusCounts struct {
	Backlog    int `json:"backlog"`
	Todo       int `json:"todo"`
	InProgress int `json:"inProgress"`
	Review     int `json:"review"`
	Done       int `json:"done"`
	Canceled   int `json:"canceled"`
}

type ProjectOverviewResponse struct {
	Project          ProjectResponse         `json:"project"`
	MemberCount      int                     `json:"memberCount"`
	ModuleCount      int                     `json:"moduleCount"`
	TaskCount        int                     `json:"taskCount"`
	TaskStatusCounts ProjectTaskStatusCounts `json:"taskStatusCounts"`
}

type ProjectOverviewsByIDsResponse struct {
	Items []ProjectOverviewResponse `json:"items"`
}

type ProjectMemberResponse struct {
	ProjectID  string      `json:"projectId"`
	UserID     string      `json:"userId"`
	UserAlias  string      `json:"userAlias"`
	UserName   string      `json:"userName"`
	UserAvatar string      `json:"userAvatar,omitempty"`
	Role       ProjectRole `json:"role"`
}

type ProjectAvailableMemberResponse struct {
	WorkspaceID   string      `json:"workspaceId"`
	UserID        string      `json:"userId"`
	UserAlias     string      `json:"userAlias"`
	UserName      string      `json:"userName"`
	UserAvatar    string      `json:"userAvatar,omitempty"`
	WorkspaceRole ProjectRole `json:"workspaceRole"`
}

type ListProjectMembersResponse struct {
	Items            []ProjectMemberResponse          `json:"items"`
	HasMore          bool                             `json:"hasMore"`
	Page             int                              `json:"page"`
	Limit            int                              `json:"limit"`
	AvailableMembers []ProjectAvailableMemberResponse `json:"availableMembers,omitempty"`
}

type ProjectMembersByProjectIDsResponse struct {
	Items []ProjectMemberResponse `json:"items"`
}

type projectRow struct {
	ID              pgtype.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Name            string
	WorkspaceID     pgtype.UUID
	Avatar          pgtype.Text
	TaskCount       int
	TotalEstimation int
}

type projectMemberRow struct {
	ProjectID  pgtype.UUID
	UserID     pgtype.UUID
	UserAlias  string
	UserName   string
	UserAvatar string
	Role       ProjectRole
}

type projectAvailableMemberRow struct {
	WorkspaceID   pgtype.UUID
	UserID        pgtype.UUID
	UserAlias     string
	UserName      string
	UserAvatar    string
	WorkspaceRole ProjectRole
}

type projectOverviewRow struct {
	Project          projectRow
	MemberCount      int
	ModuleCount      int
	TaskCount        int
	TaskStatusCounts ProjectTaskStatusCounts
}

type projectHistoryRow struct {
	ID              pgtype.UUID
	EntityID        pgtype.UUID
	ChangedByID     pgtype.UUID
	ChangedByAlias  string
	ChangedByName   string
	ChangedByAvatar string
	Field           string
	OldValue        pgtype.Text
	NewValue        pgtype.Text
	CreatedAt       time.Time
}

func projectResponseFromRow(row projectRow) ProjectResponse {
	return ProjectResponse{
		ID:              pgutil.UUIDToString(row.ID),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		Name:            row.Name,
		WorkspaceID:     pgutil.UUIDToString(row.WorkspaceID),
		Avatar:          textString(row.Avatar),
		TaskCount:       row.TaskCount,
		TotalEstimation: row.TotalEstimation,
	}
}

func (event ProjectNotificationEvent) ToNotificationEvent() notification.NotificationEvent {
	params := map[string]string{}
	if event.Role != "" {
		params["role"] = string(event.Role)
	}
	if event.ProjectName != "" {
		params["projectName"] = event.ProjectName
	}

	data := map[string]string{
		"type":        event.Type,
		"targetID":    event.ProjectID,
		"projectId":   event.ProjectID,
		"workspaceId": event.WorkspaceID,
	}
	for k, v := range params {
		data[k] = v
	}

	title, body := i18n.RenderNotification(i18n.Default, event.Type, params)
	if title == "" {
		title, body = i18n.RenderNotification(i18n.Default, "project_updated", params)
	}
	return notification.NotificationEvent{
		UserID:      event.RecipientID,
		WorkspaceID: event.WorkspaceID,
		Title:       title,
		Body:        body,
		Data:        data,
	}
}

func projectResponseFromDatabaseProject(row database.Project) ProjectResponse {
	return projectResponseFromRow(projectRow{
		ID:          row.ID,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		Name:        row.Name,
		WorkspaceID: row.WorkspaceID,
		Avatar:      row.Avatar,
	})
}

func projectResponseFromDatabaseProjectWithStats(row database.ListProjectsByWorkspaceAliasWithStatsRow) ProjectResponse {
	return projectResponseFromRow(projectRow{
		ID:              row.ID,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
		Name:            row.Name,
		WorkspaceID:     row.WorkspaceID,
		Avatar:          row.Avatar,
		TaskCount:       int(row.TaskCount),
		TotalEstimation: int(row.TotalEstimation),
	})
}

func projectOverviewResponseFromBatchRow(row database.GetProjectOverviewsByIDsRow) ProjectOverviewResponse {
	return projectOverviewResponseFromRow(projectOverviewRow{
		Project: projectRow{
			ID:          row.ID,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
			Name:        row.Name,
			WorkspaceID: row.WorkspaceID,
			Avatar:      row.Avatar,
		},
		MemberCount: int(row.MemberCount),
		ModuleCount: int(row.ModuleCount),
		TaskCount:   int(row.TaskCount),
		TaskStatusCounts: ProjectTaskStatusCounts{
			Backlog:    int(row.BacklogCount),
			Todo:       int(row.TodoCount),
			InProgress: int(row.InProgressCount),
			Review:     int(row.ReviewCount),
			Done:       int(row.DoneCount),
			Canceled:   int(row.CanceledCount),
		},
	})
}

func projectMemberResponseFromRow(row projectMemberRow) ProjectMemberResponse {
	return ProjectMemberResponse{
		ProjectID:  pgutil.UUIDToString(row.ProjectID),
		UserID:     pgutil.UUIDToString(row.UserID),
		UserAlias:  row.UserAlias,
		UserName:   row.UserName,
		UserAvatar: row.UserAvatar,
		Role:       row.Role,
	}
}

func projectMemberResponseFromListRow(row database.ListProjectMembersByAliasRow) ProjectMemberResponse {
	return projectMemberResponseFromRow(projectMemberRow{
		ProjectID:  row.ProjectID,
		UserID:     row.UserID,
		UserAlias:  row.Alias,
		UserName:   row.Name,
		UserAvatar: row.Avatar,
		Role:       ProjectRole(string(row.Role)),
	})
}

func projectMemberResponseFromAddRow(row database.AddProjectMemberByIDRow) ProjectMemberResponse {
	return projectMemberResponseFromRow(projectMemberRow{
		ProjectID:  row.ProjectID,
		UserID:     row.UserID,
		UserAlias:  row.Alias,
		UserName:   row.Name,
		UserAvatar: fmt.Sprint(row.Coalesce),
		Role:       ProjectRole(string(row.Role)),
	})
}

func projectMemberResponseFromUpdateRow(row database.UpdateProjectMemberRoleByIDRow) ProjectMemberResponse {
	return projectMemberResponseFromRow(projectMemberRow{
		ProjectID:  row.ProjectID,
		UserID:     row.UserID,
		UserAlias:  row.Alias,
		UserName:   row.Name,
		UserAvatar: row.Avatar,
		Role:       ProjectRole(string(row.Role)),
	})
}

func projectAvailableMemberResponseFromRow(row projectAvailableMemberRow) ProjectAvailableMemberResponse {
	return ProjectAvailableMemberResponse{
		WorkspaceID:   pgutil.UUIDToString(row.WorkspaceID),
		UserID:        pgutil.UUIDToString(row.UserID),
		UserAlias:     row.UserAlias,
		UserName:      row.UserName,
		UserAvatar:    row.UserAvatar,
		WorkspaceRole: row.WorkspaceRole,
	}
}

func projectAvailableMemberResponseFromDatabaseRow(row database.ListAvailableProjectMembersRow) ProjectAvailableMemberResponse {
	return projectAvailableMemberResponseFromRow(projectAvailableMemberRow{
		WorkspaceID:   row.WorkspaceID,
		UserID:        row.UserID,
		UserAlias:     row.Alias,
		UserName:      row.Name,
		UserAvatar:    row.Avatar,
		WorkspaceRole: ProjectRole(row.WuRole),
	})
}

func projectOverviewResponseFromRow(row projectOverviewRow) ProjectOverviewResponse {
	return ProjectOverviewResponse{
		Project:          projectResponseFromRow(row.Project),
		MemberCount:      row.MemberCount,
		ModuleCount:      row.ModuleCount,
		TaskCount:        row.TaskCount,
		TaskStatusCounts: row.TaskStatusCounts,
	}
}

func projectHistoryResponseFromRow(row projectHistoryRow) ProjectHistoryResponse {
	return ProjectHistoryResponse{
		ID: pgutil.UUIDToString(row.ID),
		ChangedBy: HistoryChangedBy{
			ID:     pgutil.UUIDToString(row.ChangedByID),
			Alias:  row.ChangedByAlias,
			Name:   row.ChangedByName,
			Avatar: row.ChangedByAvatar,
		},
		Field:     row.Field,
		OldValue:  textPtr(row.OldValue),
		NewValue:  textPtr(row.NewValue),
		CreatedAt: row.CreatedAt,
	}
}

func projectHistoryResponseFromDatabaseHistory(row database.ListProjectHistoriesRow) ProjectHistoryResponse {
	return projectHistoryResponseFromRow(projectHistoryRow{
		ID:              row.ID,
		EntityID:        row.EntityID,
		ChangedByID:     row.ChangedByID,
		ChangedByAlias:  textString(row.ChangedByAlias),
		ChangedByName:   textString(row.ChangedByName),
		ChangedByAvatar: row.ChangedByAvatar,
		Field:           row.Field,
		OldValue:        row.OldValue,
		NewValue:        row.NewValue,
		CreatedAt:       row.CreatedAt.Time,
	})
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func normalizeUpdateProjectRequest(req *UpdateProjectRequest) error {
	if req.NameSet && req.Name == nil {
		return fmt.Errorf("Tên không được null")
	}
	if req.namePresent() {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			return fmt.Errorf("Tên không được để trống")
		}
		req.Name = &value
	}
	if req.AvatarSet && req.Avatar == nil {
		return fmt.Errorf("avatar không được null")
	}
	if req.avatarPresent() {
		value := strings.TrimSpace(*req.Avatar)
		req.Avatar = &value
	}
	return nil
}

func (req UpdateProjectRequest) namePresent() bool {
	return req.NameSet || req.Name != nil
}

func (req UpdateProjectRequest) avatarPresent() bool {
	return req.AvatarSet || req.Avatar != nil
}

func (req *UpdateProjectRequest) UnmarshalJSON(data []byte) error {
	type updateProjectRequest UpdateProjectRequest
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded updateProjectRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*req = UpdateProjectRequest(decoded)
	_, req.NameSet = raw["name"]
	_, req.AvatarSet = raw["avatar"]
	return nil
}

type CreateProjectLinkRequest struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type UpdateProjectLinkRequest struct {
	Title *string `json:"title,omitempty"`
	URL   *string `json:"url,omitempty"`
}

type ProjectLinkResponse struct {
	ID        string      `json:"id"`
	CreatedAt time.Time   `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt time.Time   `json:"updatedAt" swaggertype:"string" format:"date-time"`
	ProjectID string      `json:"projectId"`
	Title     string      `json:"title"`
	URL       string      `json:"url"`
	CreatedBy NoteCreator `json:"createdBy"`
}

type CreateProjectNoteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type UpdateProjectNoteRequest struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
}

type NoteCreator struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Alias  string `json:"alias"`
	Avatar string `json:"avatar,omitempty"`
}

type ProjectNoteResponse struct {
	ID        string      `json:"id"`
	CreatedAt time.Time   `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt time.Time   `json:"updatedAt" swaggertype:"string" format:"date-time"`
	ProjectID string      `json:"projectId"`
	Title     string      `json:"title"`
	Content   string      `json:"content"`
	CreatedBy NoteCreator `json:"createdBy"`
}

type CreateProjectFileRequest struct {
	Title       string `json:"title"`
	FileURL     string `json:"fileUrl"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
}

type ProjectFileResponse struct {
	ID          string      `json:"id"`
	CreatedAt   time.Time   `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time   `json:"updatedAt" swaggertype:"string" format:"date-time"`
	ProjectID   string      `json:"projectId"`
	Title       string      `json:"title"`
	FileURL     string      `json:"fileUrl"`
	FileName    string      `json:"fileName"`
	ContentType string      `json:"contentType"`
	SizeBytes   int64       `json:"sizeBytes"`
	CreatedBy   NoteCreator `json:"createdBy"`
}

type projectLinkRow struct {
	ID            pgtype.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ProjectID     pgtype.UUID
	Title         string
	URL           string
	CreatedByID   pgtype.UUID
	CreatorName   string
	CreatorAlias  string
	CreatorAvatar string
}

type projectNoteRow struct {
	ID            pgtype.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ProjectID     pgtype.UUID
	Title         string
	Content       string
	CreatedByID   pgtype.UUID
	CreatorName   string
	CreatorAlias  string
	CreatorAvatar string
}

type projectFileRow struct {
	ID            pgtype.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ProjectID     pgtype.UUID
	Title         string
	FileURL       string
	FileName      string
	ContentType   string
	SizeBytes     int64
	CreatedByID   pgtype.UUID
	CreatorName   string
	CreatorAlias  string
	CreatorAvatar string
}

func projectLinkResponseFromRow(row projectLinkRow) ProjectLinkResponse {
	return ProjectLinkResponse{
		ID:        pgutil.UUIDToString(row.ID),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		ProjectID: pgutil.UUIDToString(row.ProjectID),
		Title:     row.Title,
		URL:       row.URL,
		CreatedBy: NoteCreator{ID: pgutil.UUIDToString(row.CreatedByID), Name: row.CreatorName, Alias: row.CreatorAlias, Avatar: row.CreatorAvatar},
	}
}

func projectLinkResponseFromDatabaseCreateLink(row database.CreateProjectLinkRow) ProjectLinkResponse {
	return projectLinkResponseFromRow(projectLinkRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		ProjectID:     row.ProjectID,
		Title:         row.Title,
		URL:           row.Url,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func projectLinkResponseFromDatabaseListLink(row database.ListProjectLinksRow) ProjectLinkResponse {
	return projectLinkResponseFromRow(projectLinkRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		ProjectID:     row.ProjectID,
		Title:         row.Title,
		URL:           row.Url,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func projectLinkResponseFromDatabaseUpdateLink(row database.UpdateProjectLinkRow) ProjectLinkResponse {
	return projectLinkResponseFromRow(projectLinkRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		ProjectID:     row.ProjectID,
		Title:         row.Title,
		URL:           row.Url,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func projectNoteResponseFromRow(row projectNoteRow) ProjectNoteResponse {
	return ProjectNoteResponse{
		ID:        pgutil.UUIDToString(row.ID),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		ProjectID: pgutil.UUIDToString(row.ProjectID),
		Title:     row.Title,
		Content:   row.Content,
		CreatedBy: NoteCreator{
			ID:     pgutil.UUIDToString(row.CreatedByID),
			Name:   row.CreatorName,
			Alias:  row.CreatorAlias,
			Avatar: row.CreatorAvatar,
		},
	}
}

func projectNoteResponseFromDatabaseCreateNote(row database.CreateProjectNoteRow) ProjectNoteResponse {
	return projectNoteResponseFromRow(projectNoteRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		ProjectID:     row.ProjectID,
		Title:         row.Title,
		Content:       row.Content,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func projectNoteResponseFromDatabaseListNote(row database.ListProjectNotesRow) ProjectNoteResponse {
	return projectNoteResponseFromRow(projectNoteRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		ProjectID:     row.ProjectID,
		Title:         row.Title,
		Content:       row.Content,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func projectNoteResponseFromDatabaseUpdateNote(row database.UpdateProjectNoteRow) ProjectNoteResponse {
	return projectNoteResponseFromRow(projectNoteRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		ProjectID:     row.ProjectID,
		Title:         row.Title,
		Content:       row.Content,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func projectFileResponseFromRow(row projectFileRow) ProjectFileResponse {
	return ProjectFileResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		ProjectID:   pgutil.UUIDToString(row.ProjectID),
		Title:       row.Title,
		FileURL:     row.FileURL,
		FileName:    row.FileName,
		ContentType: row.ContentType,
		SizeBytes:   row.SizeBytes,
		CreatedBy: NoteCreator{
			ID:     pgutil.UUIDToString(row.CreatedByID),
			Name:   row.CreatorName,
			Alias:  row.CreatorAlias,
			Avatar: row.CreatorAvatar,
		},
	}
}

func projectFileResponseFromDatabaseCreateFile(row database.CreateProjectFileRow) ProjectFileResponse {
	return projectFileResponseFromRow(projectFileRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		ProjectID:     row.ProjectID,
		Title:         row.Title,
		FileURL:       row.FileUrl,
		FileName:      row.FileName,
		ContentType:   row.ContentType,
		SizeBytes:     row.SizeBytes,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func projectFileResponseFromDatabaseListFile(row database.ListProjectFilesRow) ProjectFileResponse {
	return projectFileResponseFromRow(projectFileRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		ProjectID:     row.ProjectID,
		Title:         row.Title,
		FileURL:       row.FileUrl,
		FileName:      row.FileName,
		ContentType:   row.ContentType,
		SizeBytes:     row.SizeBytes,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func pgtypeText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}
