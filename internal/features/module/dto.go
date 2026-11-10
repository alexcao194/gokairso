package module

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
)

type ModuleStatus string

const (
	ModuleStatusBacklog    ModuleStatus = "backlog"
	ModuleStatusInProgress ModuleStatus = "in_progress"
	ModuleStatusDone       ModuleStatus = "done"
	ModuleStatusReleased   ModuleStatus = "released"
)

func (s ModuleStatus) Valid() bool {
	switch s {
	case ModuleStatusBacklog, ModuleStatusInProgress, ModuleStatusDone, ModuleStatusReleased:
		return true
	}
	return false
}

type SprintStatus string

const (
	SprintStatusPlanned   SprintStatus = "planned"
	SprintStatusActive    SprintStatus = "active"
	SprintStatusCompleted SprintStatus = "completed"
)

func (s SprintStatus) Valid() bool {
	return s == SprintStatusPlanned || s == SprintStatusActive || s == SprintStatusCompleted
}

const maxSprintDays = 31

const defaultSprintDays = 14

type ModuleSprintResponse struct {
	ID         string       `json:"id"`
	ModuleID   string       `json:"moduleId"`
	Name       string       `json:"name"`
	Goal       string       `json:"goal"`
	StartDate  string       `json:"startDate" swaggertype:"string" format:"date"`
	EndDate    string       `json:"endDate" swaggertype:"string" format:"date"`
	Status     SprintStatus `json:"status" enums:"planned,active,completed"`
	ReviewNote string       `json:"reviewNote"`
	RetroNote  string       `json:"retroNote"`
	CreatedAt  time.Time    `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt  time.Time    `json:"updatedAt" swaggertype:"string" format:"date-time"`
}

type CreateModuleSprintRequest struct {
	Name      string `json:"name,omitempty"`
	Goal      string `json:"goal,omitempty"`
	StartDate string `json:"startDate,omitempty" swaggertype:"string" format:"date"`
	EndDate   string `json:"endDate,omitempty" swaggertype:"string" format:"date"`
}

type UpdateModuleSprintRequest struct {
	Name       *string       `json:"name,omitempty"`
	Goal       *string       `json:"goal,omitempty"`
	StartDate  *string       `json:"startDate,omitempty" swaggertype:"string" format:"date"`
	EndDate    *string       `json:"endDate,omitempty" swaggertype:"string" format:"date"`
	Status     *SprintStatus `json:"status,omitempty" enums:"planned,active,completed"`
	ReviewNote *string       `json:"reviewNote,omitempty"`
	RetroNote  *string       `json:"retroNote,omitempty"`
}

func dateString(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format(time.DateOnly)
}

func sprintResponseFromDatabase(row database.ModuleSprint) ModuleSprintResponse {
	return ModuleSprintResponse{
		ID:         pgutil.UUIDToString(row.ID),
		ModuleID:   pgutil.UUIDToString(row.ModuleID),
		Name:       row.Name,
		Goal:       row.Goal,
		StartDate:  dateString(row.StartDate),
		EndDate:    dateString(row.EndDate),
		Status:     SprintStatus(row.Status),
		ReviewNote: row.ReviewNote,
		RetroNote:  row.RetroNote,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
	}
}

type CreateModuleRequest struct {
	Name      string     `json:"name"`
	StartDate *time.Time `json:"startDate,omitempty" swaggertype:"string" format:"date-time"`
	EndDate   *time.Time `json:"endDate,omitempty" swaggertype:"string" format:"date-time"`
}

type UpdateModuleRequest struct {
	Name         *string       `json:"name,omitempty"`
	StartDate    *time.Time    `json:"startDate,omitempty" swaggertype:"string" format:"date-time"`
	EndDate      *time.Time    `json:"endDate,omitempty" swaggertype:"string" format:"date-time"`
	Status       *ModuleStatus `json:"status,omitempty" enums:"backlog,in_progress,done,released"`
	NameSet      bool          `json:"-"`
	StartDateSet bool          `json:"-"`
	EndDateSet   bool          `json:"-"`
	StatusSet    bool          `json:"-"`
}

type ModuleResponse struct {
	ID              string                 `json:"id"`
	CreatedAt       time.Time              `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt       time.Time              `json:"updatedAt" swaggertype:"string" format:"date-time"`
	Name            string                 `json:"name"`
	ProjectID       string                 `json:"projectId"`
	StartDate       *time.Time             `json:"startDate,omitempty" swaggertype:"string" format:"date-time"`
	EndDate         *time.Time             `json:"endDate,omitempty" swaggertype:"string" format:"date-time"`
	Status          ModuleStatus           `json:"status" enums:"backlog,in_progress,done,released"`
	ReleasedAt      *time.Time             `json:"releasedAt,omitempty" swaggertype:"string" format:"date-time"`
	Sprints         []ModuleSprintResponse `json:"sprints"`
	TaskCount       int                    `json:"taskCount"`
	TotalEstimation int                    `json:"totalEstimation"`
	AttachmentCount int                    `json:"attachmentCount"`
}

type CreateModuleAttachmentRequest struct {
	FileName    string `json:"fileName"`
	FileURL     string `json:"fileUrl"`
	ContentType string `json:"contentType,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
}

type ModuleAttachmentResponse struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	ModuleID      string    `json:"moduleId"`
	UploaderID    string    `json:"uploaderId"`
	UploaderAlias string    `json:"uploaderAlias,omitempty"`
	UploaderName  string    `json:"uploaderName,omitempty"`
	FileName      string    `json:"fileName"`
	FileURL       string    `json:"fileUrl"`
	ContentType   string    `json:"contentType,omitempty"`
	SizeBytes     int64     `json:"sizeBytes,omitempty"`
}

type HistoryChangedBy struct {
	ID     string `json:"id"`
	Alias  string `json:"alias"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type ModuleHistoryResponse struct {
	ID          string           `json:"id"`
	ModuleID    string           `json:"moduleId"`
	ChangedByID string           `json:"changedById"`
	ChangedBy   HistoryChangedBy `json:"changedBy"`
	Field       string           `json:"field"`
	OldValue    *string          `json:"oldValue,omitempty"`
	NewValue    *string          `json:"newValue,omitempty"`
	CreatedAt   time.Time        `json:"createdAt" swaggertype:"string" format:"date-time"`
}

type moduleRow struct {
	ID              pgtype.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Name            string
	ProjectID       pgtype.UUID
	StartDate       pgtype.Timestamptz
	EndDate         pgtype.Timestamptz
	Status          ModuleStatus
	ReleasedAt      pgtype.Timestamptz
	TaskCount       int
	TotalEstimation int
	AttachmentCount int
}

type moduleHistoryRow struct {
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

func moduleResponseFromRow(row moduleRow) ModuleResponse {
	return ModuleResponse{
		ID:              pgutil.UUIDToString(row.ID),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
		Name:            row.Name,
		ProjectID:       pgutil.UUIDToString(row.ProjectID),
		StartDate:       timestamptzPtr(row.StartDate),
		EndDate:         timestamptzPtr(row.EndDate),
		Status:          row.Status,
		ReleasedAt:      timestamptzPtr(row.ReleasedAt),
		Sprints:         []ModuleSprintResponse{},
		TaskCount:       row.TaskCount,
		TotalEstimation: row.TotalEstimation,
		AttachmentCount: row.AttachmentCount,
	}
}

func moduleResponseFromDatabaseModule(row database.Module) ModuleResponse {
	return moduleResponseFromRow(moduleRow{
		ID:         row.ID,
		CreatedAt:  row.CreatedAt.Time,
		UpdatedAt:  row.UpdatedAt.Time,
		Name:       row.Name,
		ProjectID:  row.ProjectID,
		StartDate:  row.StartDate,
		EndDate:    row.EndDate,
		Status:     ModuleStatus(row.ScrumStatus),
		ReleasedAt: row.ReleasedAt,
	})
}

func moduleResponseFromDatabaseModuleWithStats(row database.ListModulesByProjectAliasWithStatsRow) ModuleResponse {
	return moduleResponseFromRow(moduleRow{
		ID:              row.ID,
		CreatedAt:       row.CreatedAt.Time,
		UpdatedAt:       row.UpdatedAt.Time,
		Name:            row.Name,
		ProjectID:       row.ProjectID,
		StartDate:       row.StartDate,
		EndDate:         row.EndDate,
		Status:          ModuleStatus(row.ScrumStatus),
		ReleasedAt:      row.ReleasedAt,
		TaskCount:       int(row.TaskCount),
		TotalEstimation: int(row.TotalEstimation),
		AttachmentCount: int(row.AttachmentCount),
	})
}

func moduleHistoryResponseFromRow(row moduleHistoryRow) ModuleHistoryResponse {
	changedByID := pgutil.UUIDToString(row.ChangedByID)
	return ModuleHistoryResponse{
		ID:          pgutil.UUIDToString(row.ID),
		ModuleID:    pgutil.UUIDToString(row.EntityID),
		ChangedByID: changedByID,
		ChangedBy: HistoryChangedBy{
			ID:     changedByID,
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

func moduleHistoryResponseFromDatabaseHistory(row database.ListModuleHistoriesByIDRow) ModuleHistoryResponse {
	return moduleHistoryResponseFromRow(moduleHistoryRow{
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

func normalizeUpdateModuleRequest(req *UpdateModuleRequest) error {
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
	return nil
}

func (req UpdateModuleRequest) namePresent() bool {
	return req.NameSet || req.Name != nil
}

func (req UpdateModuleRequest) startDatePresent() bool {
	return req.StartDateSet || req.StartDate != nil
}

func (req UpdateModuleRequest) endDatePresent() bool {
	return req.EndDateSet || req.EndDate != nil
}

func (req *UpdateModuleRequest) UnmarshalJSON(data []byte) error {
	type updateModuleRequest UpdateModuleRequest
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded updateModuleRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*req = UpdateModuleRequest(decoded)
	_, req.NameSet = raw["name"]
	_, req.StartDateSet = raw["startDate"]
	_, req.EndDateSet = raw["endDate"]
	_, req.StatusSet = raw["status"]
	return nil
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func nullableTimeValue(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
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
