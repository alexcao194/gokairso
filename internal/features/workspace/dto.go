package workspace

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

type WorkspaceRole string
type WorkspaceMemberStatus string

const (
	WorkspaceRoleHolder WorkspaceRole = "holder"
	WorkspaceRoleAdmin  WorkspaceRole = "admin"
	WorkspaceRoleMember WorkspaceRole = "member"

	WorkspaceMemberStatusAccepted WorkspaceMemberStatus = "accepted"
	WorkspaceMemberStatusPending  WorkspaceMemberStatus = "pending"

	NotificationTypeWorkspaceInvitation        = "workspace_invitation"
	NotificationTypeWorkspaceMemberRoleUpdated = "workspace_member_role_updated"
	NotificationTypeWorkspaceMemberRemoved     = "workspace_member_removed"
)

func (r WorkspaceRole) Valid() bool {
	switch r {
	case WorkspaceRoleHolder, WorkspaceRoleAdmin, WorkspaceRoleMember:
		return true
	default:
		return false
	}
}

func (r WorkspaceRole) CanManageMembers() bool {
	return r == WorkspaceRoleHolder || r == WorkspaceRoleAdmin
}

func (r WorkspaceRole) rank() int {
	switch r {
	case WorkspaceRoleHolder:
		return 3
	case WorkspaceRoleAdmin:
		return 2
	default:
		return 1
	}
}

func (r WorkspaceRole) CanManageRole(target WorkspaceRole) bool {
	return r.rank() >= target.rank()
}

type CreateWorkspaceRequest struct {
	Name string `json:"name"`
}

type UpdateWorkspaceRequest struct {
	Name      *string `json:"name,omitempty"`
	Avatar    *string `json:"avatar,omitempty"`
	NameSet   bool    `json:"-"`
	AvatarSet bool    `json:"-"`
}

type InviteMemberRequest struct {
	UserAlias string        `json:"userAlias"`
	Role      WorkspaceRole `json:"role"`
}

type UpdateMemberRoleRequest struct {
	Role WorkspaceRole `json:"role"`
}

type WorkspaceNotificationEvent struct {
	Type          string
	RecipientID   string
	WorkspaceID   string
	WorkspaceName string
	ActorAlias    string
	Role          WorkspaceRole
}

type WorkspaceResponse struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
	CreatedByID string    `json:"createdById"`
	Name        string    `json:"name"`
	Avatar      string    `json:"avatar,omitempty"`
}

type HistoryChangedBy struct {
	ID     string `json:"id"`
	Alias  string `json:"alias"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type WorkspaceHistoryResponse struct {
	ID        string           `json:"id"`
	ChangedBy HistoryChangedBy `json:"changedBy"`
	Field     string           `json:"field"`
	OldValue  *string          `json:"oldValue,omitempty"`
	NewValue  *string          `json:"newValue,omitempty"`
	CreatedAt time.Time        `json:"createdAt" swaggertype:"string" format:"date-time"`
}

type WorkspaceMemberResponse struct {
	WorkspaceID  string                `json:"workspaceId"`
	UserID       string                `json:"userId"`
	UserAlias    string                `json:"userAlias"`
	UserName     string                `json:"userName"`
	UserAvatar   string                `json:"userAvatar,omitempty"`
	Role         WorkspaceRole         `json:"role"`
	Status       WorkspaceMemberStatus `json:"status"`
	InvitationID string                `json:"invitationId,omitempty"`
}

type WorkspaceInvitationResponse struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspaceId"`
	InviteeID   string        `json:"inviteeId"`
	InviterID   string        `json:"inviterId"`
	Role        WorkspaceRole `json:"role"`
	Status      string        `json:"status"`
	CreatedAt   time.Time     `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time     `json:"updatedAt" swaggertype:"string" format:"date-time"`
}

type WorkspaceInvitationSummaryResponse struct {
	ID            string        `json:"id"`
	WorkspaceID   string        `json:"workspaceId"`
	WorkspaceName string        `json:"workspaceName"`
	InviterID     string        `json:"inviterId"`
	InviterAlias  string        `json:"inviterAlias"`
	InviterName   string        `json:"inviterName"`
	InviterAvatar string        `json:"inviterAvatar,omitempty"`
	Role          WorkspaceRole `json:"role"`
	Status        string        `json:"status"`
	CreatedAt     time.Time     `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt     time.Time     `json:"updatedAt" swaggertype:"string" format:"date-time"`
}

type workspaceRow struct {
	ID          pgtype.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedByID pgtype.UUID
	Name        string
	Avatar      pgtype.Text
}

type workspaceMemberRow struct {
	WorkspaceID  pgtype.UUID
	UserID       pgtype.UUID
	UserAlias    string
	UserName     string
	UserAvatar   string
	Role         WorkspaceRole
	Status       WorkspaceMemberStatus
	InvitationID pgtype.UUID
	JoinedAt     time.Time
}

type workspaceInvitationRow struct {
	ID          pgtype.UUID
	WorkspaceID pgtype.UUID
	InviteeID   pgtype.UUID
	InviterID   pgtype.UUID
	Role        WorkspaceRole
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type workspaceInvitationSummaryRow struct {
	ID            pgtype.UUID
	WorkspaceID   pgtype.UUID
	WorkspaceName string
	InviterID     pgtype.UUID
	InviterAlias  string
	InviterName   string
	InviterAvatar pgtype.Text
	Role          WorkspaceRole
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type workspaceHistoryRow struct {
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

func workspaceResponseFromRow(row workspaceRow) WorkspaceResponse {
	return WorkspaceResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		CreatedByID: pgutil.UUIDToString(row.CreatedByID),
		Name:        row.Name,
		Avatar:      textString(row.Avatar),
	}
}

func (event WorkspaceNotificationEvent) ToNotificationEvent() notification.NotificationEvent {
	params := map[string]string{}
	if event.Role != "" {
		params["role"] = string(event.Role)
	}
	if event.WorkspaceName != "" {
		params["workspaceName"] = event.WorkspaceName
	}
	if event.ActorAlias != "" {
		params["actor"] = event.ActorAlias
	}

	data := map[string]string{
		"type":        event.Type,
		"targetID":    event.WorkspaceID,
		"workspaceId": event.WorkspaceID,
	}
	for k, v := range params {
		data[k] = v
	}

	title, body := i18n.RenderNotification(i18n.Default, event.Type, params)
	if title == "" {
		title, body = i18n.RenderNotification(i18n.Default, "workspace_updated", params)
	}
	return notification.NotificationEvent{
		UserID:      event.RecipientID,
		WorkspaceID: event.WorkspaceID,
		Title:       title,
		Body:        body,
		Data:        data,
	}
}

func workspaceResponseFromDatabaseWorkspace(row database.Workspace) WorkspaceResponse {
	return workspaceResponseFromRow(workspaceRow{
		ID:          row.ID,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		CreatedByID: row.CreatedByID,
		Name:        row.Name,
		Avatar:      row.Avatar,
	})
}

func workspaceMemberResponseFromRow(row workspaceMemberRow) WorkspaceMemberResponse {
	return WorkspaceMemberResponse{
		WorkspaceID:  pgutil.UUIDToString(row.WorkspaceID),
		UserID:       pgutil.UUIDToString(row.UserID),
		UserAlias:    row.UserAlias,
		UserName:     row.UserName,
		UserAvatar:   row.UserAvatar,
		Role:         row.Role,
		Status:       row.Status,
		InvitationID: pgutil.UUIDToString(row.InvitationID),
	}
}

func workspaceMemberResponseFromListRow(row database.ListWorkspaceMembersRow) WorkspaceMemberResponse {
	return workspaceMemberResponseFromRow(workspaceMemberRow{
		WorkspaceID:  row.WorkspaceID,
		UserID:       row.UserID,
		UserAlias:    row.Alias,
		UserName:     row.Name,
		UserAvatar:   row.Avatar,
		Role:         WorkspaceRole(string(row.Role)),
		Status:       WorkspaceMemberStatus(row.Status),
		InvitationID: row.InvitationID,
	})
}

func workspaceMemberResponseFromUpdateRow(row database.UpdateWorkspaceMemberRoleRow) WorkspaceMemberResponse {
	return workspaceMemberResponseFromRow(workspaceMemberRow{
		WorkspaceID:  row.WorkspaceID,
		UserID:       row.UserID,
		UserAlias:    row.Alias,
		UserName:     row.Name,
		UserAvatar:   row.Avatar,
		Role:         WorkspaceRole(string(row.Role)),
		Status:       WorkspaceMemberStatus(row.Column7),
		InvitationID: row.Column8,
	})
}

func workspaceInvitationResponseFromRow(row workspaceInvitationRow) WorkspaceInvitationResponse {
	return WorkspaceInvitationResponse{
		ID:          pgutil.UUIDToString(row.ID),
		WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
		InviteeID:   pgutil.UUIDToString(row.InviteeID),
		InviterID:   pgutil.UUIDToString(row.InviterID),
		Role:        row.Role,
		Status:      row.Status,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func workspaceInvitationResponseFromDatabaseInvitation(row database.WorkspaceInvitation) WorkspaceInvitationResponse {
	return workspaceInvitationResponseFromRow(workspaceInvitationRow{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		InviteeID:   row.InviteeID,
		InviterID:   row.InviterID,
		Role:        WorkspaceRole(string(row.Role)),
		Status:      row.Status,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	})
}

func workspaceInvitationSummaryResponseFromRow(row workspaceInvitationSummaryRow) WorkspaceInvitationSummaryResponse {
	return WorkspaceInvitationSummaryResponse{
		ID:            pgutil.UUIDToString(row.ID),
		WorkspaceID:   pgutil.UUIDToString(row.WorkspaceID),
		WorkspaceName: row.WorkspaceName,
		InviterID:     pgutil.UUIDToString(row.InviterID),
		InviterAlias:  row.InviterAlias,
		InviterName:   row.InviterName,
		InviterAvatar: textString(row.InviterAvatar),
		Role:          row.Role,
		Status:        row.Status,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
}

func workspaceInvitationSummaryResponseFromDatabaseRow(row database.ListMyWorkspaceInvitationsRow) WorkspaceInvitationSummaryResponse {
	return workspaceInvitationSummaryResponseFromRow(workspaceInvitationSummaryRow{
		ID:            row.ID,
		WorkspaceID:   row.WorkspaceID,
		WorkspaceName: row.WorkspaceName,
		InviterID:     row.InviterID,
		InviterAlias:  row.InviterAlias,
		InviterName:   row.InviterName,
		InviterAvatar: row.InviterAvatar,
		Role:          WorkspaceRole(string(row.Role)),
		Status:        row.Status,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	})
}

func workspaceHistoryResponseFromRow(row workspaceHistoryRow) WorkspaceHistoryResponse {
	return WorkspaceHistoryResponse{
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

func workspaceHistoryResponseFromDatabaseHistory(row database.ListWorkspaceHistoriesRow) WorkspaceHistoryResponse {
	return workspaceHistoryResponseFromRow(workspaceHistoryRow{
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

func normalizeUpdateWorkspaceRequest(req *UpdateWorkspaceRequest) error {
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

func (req UpdateWorkspaceRequest) namePresent() bool {
	return req.NameSet || req.Name != nil
}

func (req UpdateWorkspaceRequest) avatarPresent() bool {
	return req.AvatarSet || req.Avatar != nil
}

func (req *UpdateWorkspaceRequest) UnmarshalJSON(data []byte) error {
	type updateWorkspaceRequest UpdateWorkspaceRequest
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded updateWorkspaceRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*req = UpdateWorkspaceRequest(decoded)
	_, req.NameSet = raw["name"]
	_, req.AvatarSet = raw["avatar"]
	return nil
}

type CreateWorkspaceLinkRequest struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type WorkspaceLinkResponse struct {
	ID          string      `json:"id"`
	CreatedAt   time.Time   `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time   `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID string      `json:"workspaceId"`
	Title       string      `json:"title"`
	URL         string      `json:"url"`
	CreatedBy   NoteCreator `json:"createdBy"`
}

type CreateWorkspaceNoteRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type UpdateWorkspaceNoteRequest struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
}

type NoteCreator struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Alias  string `json:"alias"`
	Avatar string `json:"avatar,omitempty"`
}

type WorkspaceNoteResponse struct {
	ID          string      `json:"id"`
	CreatedAt   time.Time   `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time   `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID string      `json:"workspaceId"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	CreatedBy   NoteCreator `json:"createdBy"`
}

type CreateWorkspaceFileRequest struct {
	Title       string `json:"title"`
	FileURL     string `json:"fileUrl"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
}

type WorkspaceFileResponse struct {
	ID          string      `json:"id"`
	CreatedAt   time.Time   `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time   `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID string      `json:"workspaceId"`
	Title       string      `json:"title"`
	FileURL     string      `json:"fileUrl"`
	FileName    string      `json:"fileName"`
	ContentType string      `json:"contentType"`
	SizeBytes   int64       `json:"sizeBytes"`
	CreatedBy   NoteCreator `json:"createdBy"`
}

type workspaceLinkRow struct {
	ID            pgtype.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	WorkspaceID   pgtype.UUID
	Title         string
	URL           string
	CreatedByID   pgtype.UUID
	CreatorName   string
	CreatorAlias  string
	CreatorAvatar string
}

type workspaceNoteRow struct {
	ID            pgtype.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	WorkspaceID   pgtype.UUID
	Title         string
	Content       string
	CreatedByID   pgtype.UUID
	CreatorName   string
	CreatorAlias  string
	CreatorAvatar string
}

type workspaceFileRow struct {
	ID            pgtype.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	WorkspaceID   pgtype.UUID
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

func workspaceLinkResponseFromRow(row workspaceLinkRow) WorkspaceLinkResponse {
	return WorkspaceLinkResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
		Title:       row.Title,
		URL:         row.URL,
		CreatedBy:   NoteCreator{ID: pgutil.UUIDToString(row.CreatedByID), Name: row.CreatorName, Alias: row.CreatorAlias, Avatar: row.CreatorAvatar},
	}
}

func workspaceLinkResponseFromDatabaseCreateLink(row database.CreateWorkspaceLinkRow) WorkspaceLinkResponse {
	return workspaceLinkResponseFromRow(workspaceLinkRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		WorkspaceID:   row.WorkspaceID,
		Title:         row.Title,
		URL:           row.Url,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func workspaceLinkResponseFromDatabaseListLink(row database.ListWorkspaceLinksRow) WorkspaceLinkResponse {
	return workspaceLinkResponseFromRow(workspaceLinkRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		WorkspaceID:   row.WorkspaceID,
		Title:         row.Title,
		URL:           row.Url,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func workspaceNoteResponseFromRow(row workspaceNoteRow) WorkspaceNoteResponse {
	return WorkspaceNoteResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
		Title:       row.Title,
		Content:     row.Content,
		CreatedBy: NoteCreator{
			ID:     pgutil.UUIDToString(row.CreatedByID),
			Name:   row.CreatorName,
			Alias:  row.CreatorAlias,
			Avatar: row.CreatorAvatar,
		},
	}
}

func workspaceNoteResponseFromDatabaseCreateNote(row database.CreateWorkspaceNoteRow) WorkspaceNoteResponse {
	return workspaceNoteResponseFromRow(workspaceNoteRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		WorkspaceID:   row.WorkspaceID,
		Title:         row.Title,
		Content:       row.Content,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func workspaceNoteResponseFromDatabaseListNote(row database.ListWorkspaceNotesRow) WorkspaceNoteResponse {
	return workspaceNoteResponseFromRow(workspaceNoteRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		WorkspaceID:   row.WorkspaceID,
		Title:         row.Title,
		Content:       row.Content,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func workspaceNoteResponseFromDatabaseUpdateNote(row database.UpdateWorkspaceNoteRow) WorkspaceNoteResponse {
	return workspaceNoteResponseFromRow(workspaceNoteRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		WorkspaceID:   row.WorkspaceID,
		Title:         row.Title,
		Content:       row.Content,
		CreatedByID:   row.CreatedByID,
		CreatorName:   row.CreatorName,
		CreatorAlias:  row.CreatorAlias,
		CreatorAvatar: row.CreatorAvatar,
	})
}

func workspaceFileResponseFromRow(row workspaceFileRow) WorkspaceFileResponse {
	return WorkspaceFileResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
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

func workspaceFileResponseFromDatabaseCreateFile(row database.CreateWorkspaceFileRow) WorkspaceFileResponse {
	return workspaceFileResponseFromRow(workspaceFileRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		WorkspaceID:   row.WorkspaceID,
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

func workspaceFileResponseFromDatabaseListFile(row database.ListWorkspaceFilesRow) WorkspaceFileResponse {
	return workspaceFileResponseFromRow(workspaceFileRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		WorkspaceID:   row.WorkspaceID,
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
