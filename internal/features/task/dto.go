package task

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/notification"
	"github.com/alexcao194/kairso/internal/features/user"
	"github.com/alexcao194/kairso/internal/i18n"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type TaskStatus string

const (
	TaskStatusBacklog    TaskStatus = "backlog"
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusCanceled   TaskStatus = "canceled"

	NotificationTypeTaskAssigned              = "task_assigned"
	NotificationTypeTaskUnassigned            = "task_unassigned"
	NotificationTypeTaskStatusUpdated         = "task_status_updated"
	NotificationTypeTaskFaultHolderAssigned   = "task_fault_holder_assigned"
	NotificationTypeTaskFaultHolderUnassigned = "task_fault_holder_unassigned"
	NotificationTypeTaskDeadlineReminder      = "task_deadline_reminder"
	NotificationTypeTaskCommentAdded          = "task_comment_added"
)

func (s TaskStatus) Valid() bool {
	switch s {
	case TaskStatusBacklog, TaskStatusTodo, TaskStatusInProgress, TaskStatusReview, TaskStatusDone, TaskStatusCanceled:
		return true
	default:
		return false
	}
}

type CreateTaskRequest struct {
	ModuleID      *string    `json:"moduleId,omitempty"`
	ParentID      *string    `json:"parentId,omitempty"`
	Title         string     `json:"title"`
	Description   string     `json:"description,omitempty"`
	Status        TaskStatus `json:"status"`
	Priority      *int       `json:"priority"`
	StartDate     *time.Time `json:"startDate,omitempty" swaggertype:"string" format:"date-time"`
	EndDate       *time.Time `json:"endDate,omitempty" swaggertype:"string" format:"date-time"`
	Estimation    int        `json:"estimation"`
	Archived      bool       `json:"archived"`
	FaultHolderID *string    `json:"faultHolderId,omitempty"`
}

type UpdateTaskRequest struct {
	ModuleID         *string     `json:"moduleId,omitempty"`
	ParentID         *string     `json:"parentId,omitempty"`
	Title            *string     `json:"title,omitempty"`
	Description      *string     `json:"description,omitempty"`
	Status           *TaskStatus `json:"status,omitempty"`
	Priority         *int        `json:"priority,omitempty"`
	StartDate        *time.Time  `json:"startDate,omitempty" swaggertype:"string" format:"date-time"`
	EndDate          *time.Time  `json:"endDate,omitempty" swaggertype:"string" format:"date-time"`
	Estimation       *int        `json:"estimation,omitempty"`
	Archived         *bool       `json:"archived,omitempty"`
	FaultHolderID    *string     `json:"faultHolderId,omitempty"`
	ReturnCount      *int        `json:"returnCount,omitempty"`
	ModuleIDSet      bool        `json:"-"`
	ParentIDSet      bool        `json:"-"`
	TitleSet         bool        `json:"-"`
	DescriptionSet   bool        `json:"-"`
	StatusSet        bool        `json:"-"`
	PrioritySet      bool        `json:"-"`
	StartDateSet     bool        `json:"-"`
	EndDateSet       bool        `json:"-"`
	EstimationSet    bool        `json:"-"`
	ArchivedSet      bool        `json:"-"`
	FaultHolderIDSet bool        `json:"-"`
	ReturnCountSet   bool        `json:"-"`
}

type ListTasksParams struct {
	ModuleID      string         `schema:"moduleId"`
	Status        string         `schema:"status"`
	Archived      string         `schema:"archived"`
	ParentID      string         `schema:"parentId"`
	CreatedByID   string         `schema:"createdById"`
	Priority      string         `schema:"priority"`
	Estimation    string         `schema:"estimation"`
	StartDateFrom string         `schema:"startDateFrom"`
	StartDateTo   string         `schema:"startDateTo"`
	EndDateFrom   string         `schema:"endDateFrom"`
	EndDateTo     string         `schema:"endDateTo"`
	LabelID       string         `schema:"labelId"`
	WorkerID      string         `schema:"workerId"`
	Query         string         `schema:"q"`
	Sort          string         `schema:"sort"`
	SortRules     []TaskSortRule `schema:"-"`
}

type ListMyTasksParams struct {
	Status        string         `schema:"status"`
	Archived      string         `schema:"archived"`
	Priority      string         `schema:"priority"`
	ProjectID     string         `schema:"projectId"`
	StartDateFrom string         `schema:"startDateFrom"`
	StartDateTo   string         `schema:"startDateTo"`
	EndDateFrom   string         `schema:"endDateFrom"`
	EndDateTo     string         `schema:"endDateTo"`
	Query         string         `schema:"q"`
	Sort          string         `schema:"sort"`
	SortRules     []TaskSortRule `schema:"-"`
}

type MyTasksCountResponse struct {
	Count int64 `json:"count"`
}

type TaskNotificationEvent struct {
	Type        string
	RecipientID string
	WorkspaceID string
	TaskID      string
	ProjectID   string
	Title       string
	Status      TaskStatus
	CallerAlias string
	DueAt       time.Time
	// @mentioned in it (vs. just notified as a task participant), which
	Comment       string
	HasAttachment bool
	Mentioned     bool
}

type TaskDeadlineReminder struct {
	TaskID      string
	Title       string
	ProjectID   string
	WorkspaceID string
	EndDate     time.Time
}

type TaskResponse struct {
	ID                  string                    `json:"id"`
	CreatedAt           time.Time                 `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt           time.Time                 `json:"updatedAt" swaggertype:"string" format:"date-time"`
	ProjectID           string                    `json:"projectId"`
	ModuleID            string                    `json:"moduleId,omitempty"`
	CreatedByID         string                    `json:"createdById"`
	ParentID            string                    `json:"parentId,omitempty"`
	Title               string                    `json:"title"`
	Description         string                    `json:"description,omitempty"`
	Status              TaskStatus                `json:"status"`
	Priority            int                       `json:"priority"`
	StartDate           *time.Time                `json:"startDate,omitempty" swaggertype:"string" format:"date-time"`
	EndDate             *time.Time                `json:"endDate,omitempty" swaggertype:"string" format:"date-time"`
	FirstCompletedAt    *time.Time                `json:"firstCompletedAt,omitempty" swaggertype:"string" format:"date-time"`
	Estimation          int                       `json:"estimation"`
	Archived            bool                      `json:"archived"`
	FaultHolderID       string                    `json:"faultHolderId,omitempty"`
	ReturnCount         int                       `json:"returnCount"`
	SequenceNo          int                       `json:"sequenceNo"`
	Code                string                    `json:"code"`
	LinkedFeedbackCount int                       `json:"linkedFeedbackCount"`
	UnreadCommentCount  int                       `json:"unreadCommentCount"`
	Labels              []TaskLabelResponse       `json:"labels"`
	Workers             []TaskWorkerResponse      `json:"workers"`
	FaultHolders        []TaskFaultHolderResponse `json:"faultHolders"`
}

type TaskListStats struct {
	TotalCount      int `json:"totalCount"`
	TotalEstimation int `json:"totalEstimation"`
}

type TaskListResponse struct {
	pagination.Response[TaskResponse]
	Stats TaskListStats `json:"stats"`
}

type ThreadLatestCommentResponse struct {
	AuthorID     string    `json:"authorId"`
	AuthorAlias  string    `json:"authorAlias"`
	AuthorName   string    `json:"authorName"`
	AuthorAvatar string    `json:"authorAvatar,omitempty"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
}

type ThreadResponse struct {
	TaskResponse
	Mentioned     bool                         `json:"mentioned"`
	LatestComment *ThreadLatestCommentResponse `json:"latestComment,omitempty"`
}

type ThreadsCountResponse struct {
	Count int64 `json:"count"`
}

type HistoryChangedBy struct {
	ID     string `json:"id"`
	Alias  string `json:"alias"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type TaskHistoryResponse struct {
	ID        string           `json:"id"`
	ChangedBy HistoryChangedBy `json:"changedBy"`
	Field     string           `json:"field"`
	OldValue  *string          `json:"oldValue,omitempty"`
	NewValue  *string          `json:"newValue,omitempty"`
	CreatedAt time.Time        `json:"createdAt" swaggertype:"string" format:"date-time"`
}

type CreateTaskAttachmentRequest struct {
	FileName    string `json:"fileName"`
	FileURL     string `json:"fileUrl"`
	ContentType string `json:"contentType,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
}

type TaskAttachmentResponse struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	TaskID        string    `json:"taskId"`
	UploaderID    string    `json:"uploaderId"`
	UploaderAlias string    `json:"uploaderAlias,omitempty"`
	UploaderName  string    `json:"uploaderName,omitempty"`
	FileName      string    `json:"fileName"`
	FileURL       string    `json:"fileUrl"`
	ContentType   string    `json:"contentType,omitempty"`
	SizeBytes     int64     `json:"sizeBytes,omitempty"`
}

var mentionPattern = regexp.MustCompile(`@\[[^\]]+\]\(([0-9a-fA-F-]{36})\)`)

func parseMentionedUserIDs(content string) []string {
	matches := mentionPattern.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool, len(matches))
	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		id := m[1]
		if seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

var mentionDisplayPattern = regexp.MustCompile(`@\[([^\]]+)\]\([0-9a-fA-F-]{36}\)`)

func plainTextPreview(content string) string {
	return mentionDisplayPattern.ReplaceAllString(content, "@$1")
}

const maxTaskCommentLength = 2000

const maxTaskCommentAttachments = 10

type CreateTaskCommentAttachment struct {
	FileName    string `json:"fileName"`
	FileURL     string `json:"fileUrl"`
	ContentType string `json:"contentType,omitempty"`
	SizeBytes   int64  `json:"sizeBytes,omitempty"`
}

type CreateTaskCommentRequest struct {
	Content     string                        `json:"content"`
	Attachments []CreateTaskCommentAttachment `json:"attachments,omitempty"`
}

type UpdateTaskCommentRequest struct {
	Content string `json:"content"`
}

func normalizeUpdateTaskCommentRequest(req *UpdateTaskCommentRequest) error {
	req.Content = strings.TrimSpace(req.Content)
	if len(req.Content) > maxTaskCommentLength {
		return fmt.Errorf("Nội dung tối đa %d ký tự", maxTaskCommentLength)
	}
	return nil
}

func normalizeCreateTaskCommentRequest(req *CreateTaskCommentRequest) error {
	req.Content = strings.TrimSpace(req.Content)
	if len(req.Attachments) > maxTaskCommentAttachments {
		return fmt.Errorf("Mỗi bình luận tối đa %d tệp đính kèm", maxTaskCommentAttachments)
	}
	for i := range req.Attachments {
		req.Attachments[i].FileName = strings.TrimSpace(req.Attachments[i].FileName)
		req.Attachments[i].FileURL = strings.TrimSpace(req.Attachments[i].FileURL)
		if req.Attachments[i].FileName == "" || req.Attachments[i].FileURL == "" {
			return fmt.Errorf("Mỗi tệp đính kèm cần có fileName và fileUrl")
		}
	}
	if req.Content == "" && len(req.Attachments) == 0 {
		return fmt.Errorf("Cần nhập nội dung hoặc đính kèm tệp")
	}
	if len(req.Content) > maxTaskCommentLength {
		return fmt.Errorf("Nội dung tối đa %d ký tự", maxTaskCommentLength)
	}
	return nil
}

type TaskCommentAttachmentResponse struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	CommentID   string    `json:"commentId"`
	UploaderID  string    `json:"uploaderId"`
	FileName    string    `json:"fileName"`
	FileURL     string    `json:"fileUrl"`
	ContentType string    `json:"contentType,omitempty"`
	SizeBytes   int64     `json:"sizeBytes,omitempty"`
}

type TaskCommentResponse struct {
	ID               string                          `json:"id"`
	CreatedAt        time.Time                       `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt        time.Time                       `json:"updatedAt" swaggertype:"string" format:"date-time"`
	TaskID           string                          `json:"taskId"`
	AuthorID         string                          `json:"authorId"`
	AuthorAlias      string                          `json:"authorAlias,omitempty"`
	AuthorName       string                          `json:"authorName,omitempty"`
	AuthorAvatar     string                          `json:"authorAvatar,omitempty"`
	Content          string                          `json:"content"`
	MentionedUserIDs []string                        `json:"mentionedUserIds,omitempty"`
	Attachments      []TaskCommentAttachmentResponse `json:"attachments,omitempty"`
}

type CreateTaskLabelRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type UpdateTaskLabelRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type TaskLabelResponse struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
	CreatedByID string    `json:"createdById"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	ProjectID   string    `json:"projectId"`
}

type TaskLabelsByProjectIDsResponse struct {
	Items []TaskLabelResponse `json:"items"`
}

type AssignTaskWorkerRequest struct {
	Weight int `json:"weight"`
}

type TaskWorkerResponse struct {
	TaskID string           `json:"taskId"`
	Weight int              `json:"weight"`
	User   user.UserSummary `json:"user"`
}

type AssignTaskFaultHolderRequest struct {
	Weight int `json:"weight"`
}

type TaskFaultHolderResponse struct {
	TaskID string           `json:"taskId"`
	Weight int              `json:"weight"`
	User   user.UserSummary `json:"user"`
}

type taskHistoryRow struct {
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

func (event TaskNotificationEvent) ToNotificationEvent() notification.NotificationEvent {
	data := map[string]string{
		"type":        event.Type,
		"targetID":    event.TaskID,
		"taskId":      event.TaskID,
		"projectId":   event.ProjectID,
		"workspaceId": event.WorkspaceID,
	}
	params := map[string]string{}
	if event.Title != "" {
		params["taskTitle"] = event.Title
	}
	if event.CallerAlias != "" {
		params["caller"] = event.CallerAlias
	}
	if event.Status != "" {
		data["status"] = string(event.Status)
		params["status"] = string(event.Status)
	}

	renderType := event.Type
	switch event.Type {
	case NotificationTypeTaskDeadlineReminder:
		data["dueAt"] = event.DueAt.Format(time.RFC3339)
		params["dueAt"] = event.DueAt.Format("15:04 02/01/2006")
	case NotificationTypeTaskCommentAdded:
		snippet := plainTextPreview(event.Comment)
		if len(snippet) > 120 {
			snippet = snippet[:120] + "…"
		}
		if snippet == "" && event.HasAttachment {
			snippet = "đã gửi một tệp đính kèm"
		}
		params["snippet"] = snippet
		data["mentioned"] = strconv.FormatBool(event.Mentioned)
		if event.Mentioned {
			renderType = "task_mentioned"
		}
	}
	for k, v := range params {
		data[k] = v
	}

	title, body := i18n.RenderNotification(i18n.Default, renderType, params)
	if title == "" {
		title, body = i18n.RenderNotification(i18n.Default, "task_updated", params)
	}
	return notification.NotificationEvent{
		UserID:      event.RecipientID,
		WorkspaceID: event.WorkspaceID,
		Title:       title,
		Body:        body,
		Data:        data,
	}
}

func taskHistoryResponseFromRow(row taskHistoryRow) TaskHistoryResponse {
	return TaskHistoryResponse{
		ID: pgutil.UUIDToString(row.ID),
		ChangedBy: HistoryChangedBy{
			ID:     pgutil.UUIDToString(row.ChangedByID),
			Alias:  row.ChangedByAlias,
			Name:   row.ChangedByName,
			Avatar: row.ChangedByAvatar,
		},
		Field:     row.Field,
		OldValue:  textPtrValue(row.OldValue),
		NewValue:  textPtrValue(row.NewValue),
		CreatedAt: row.CreatedAt,
	}
}

var vietnameseFold = map[rune]rune{
	'à': 'a', 'á': 'a', 'ạ': 'a', 'ả': 'a', 'ã': 'a',
	'â': 'a', 'ầ': 'a', 'ấ': 'a', 'ậ': 'a', 'ẩ': 'a', 'ẫ': 'a',
	'ă': 'a', 'ằ': 'a', 'ắ': 'a', 'ặ': 'a', 'ẳ': 'a', 'ẵ': 'a',
	'è': 'e', 'é': 'e', 'ẹ': 'e', 'ẻ': 'e', 'ẽ': 'e',
	'ê': 'e', 'ề': 'e', 'ế': 'e', 'ệ': 'e', 'ể': 'e', 'ễ': 'e',
	'ì': 'i', 'í': 'i', 'ị': 'i', 'ỉ': 'i', 'ĩ': 'i',
	'ò': 'o', 'ó': 'o', 'ọ': 'o', 'ỏ': 'o', 'õ': 'o',
	'ô': 'o', 'ồ': 'o', 'ố': 'o', 'ộ': 'o', 'ổ': 'o', 'ỗ': 'o',
	'ơ': 'o', 'ờ': 'o', 'ớ': 'o', 'ợ': 'o', 'ở': 'o', 'ỡ': 'o',
	'ù': 'u', 'ú': 'u', 'ụ': 'u', 'ủ': 'u', 'ũ': 'u',
	'ư': 'u', 'ừ': 'u', 'ứ': 'u', 'ự': 'u', 'ử': 'u', 'ữ': 'u',
	'ỳ': 'y', 'ý': 'y', 'ỵ': 'y', 'ỷ': 'y', 'ỹ': 'y',
	'đ': 'd',
}

func taskCodePrefix(projectName string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(projectName) {
		if folded, ok := vietnameseFold[r]; ok {
			r = folded
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	prefix := strings.ToUpper(b.String())
	if prefix == "" {
		return "PRJ"
	}
	return prefix
}

func Code(projectName string, sequenceNo int) string {
	return fmt.Sprintf("#%s-%d", taskCodePrefix(projectName), sequenceNo)
}

func FeedbackCode(projectName string, sequenceNo int) string {
	return fmt.Sprintf("#%s-F%d", taskCodePrefix(projectName), sequenceNo)
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func textPtrValue(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func nullableUUID(id *string) (*pgtype.UUID, error) {
	if id == nil || *id == "" {
		return nil, nil
	}
	parsed, err := pgutil.ParseUUID(*id)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func pgtypeUUIDValue(value *pgtype.UUID) pgtype.UUID {
	if value == nil {
		return pgtype.UUID{}
	}
	return *value
}

func nullableTimeValue(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func intValue(value *int) int32 {
	if value == nil {
		return 0
	}
	return int32(*value)
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizeCreateTaskRequest(req *CreateTaskRequest) error {
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	if req.Status == "" {
		if req.EndDate == nil {
			req.Status = TaskStatusBacklog
		} else {
			req.Status = TaskStatusTodo
		}
	}
	if req.Title == "" {
		return fmt.Errorf("Tiêu đề là bắt buộc")
	}
	if !req.Status.Valid() {
		return fmt.Errorf("Trạng thái phải là một trong: backlog, todo, in_progress, review, done, canceled")
	}
	if req.Priority == nil {
		return fmt.Errorf("priority là bắt buộc")
	}
	if *req.Priority < 1 || *req.Priority > 4 {
		return fmt.Errorf("priority phải là 1 (thấp), 2 (vừa), 3 (cao) hoặc 4 (gấp)")
	}
	if req.Estimation < 0 {
		return fmt.Errorf("estimation phải lớn hơn hoặc bằng 0")
	}
	trimStringPtr(req.ModuleID)
	trimStringPtr(req.ParentID)
	return nil
}

func normalizeUpdateTaskRequest(req *UpdateTaskRequest) error {
	if req.TitleSet && req.Title == nil {
		return fmt.Errorf("Tiêu đề không được null")
	}
	if req.titlePresent() {
		value := strings.TrimSpace(*req.Title)
		if value == "" {
			return fmt.Errorf("Tiêu đề không được để trống")
		}
		req.Title = &value
	}
	if req.descriptionPresent() && req.Description != nil {
		value := strings.TrimSpace(*req.Description)
		req.Description = &value
	}
	if req.StatusSet && req.Status == nil {
		return fmt.Errorf("Trạng thái không được null")
	}
	if req.statusPresent() && !req.Status.Valid() {
		return fmt.Errorf("Trạng thái phải là một trong: backlog, todo, in_progress, review, done, canceled")
	}
	if req.PrioritySet && req.Priority == nil {
		return fmt.Errorf("priority không được null")
	}
	if req.priorityPresent() && (*req.Priority < 1 || *req.Priority > 4) {
		return fmt.Errorf("priority phải là 1 (thấp), 2 (vừa), 3 (cao) hoặc 4 (gấp)")
	}
	if req.EstimationSet && req.Estimation == nil {
		return fmt.Errorf("estimation không được null")
	}
	if req.estimationPresent() && *req.Estimation < 0 {
		return fmt.Errorf("estimation phải lớn hơn hoặc bằng 0")
	}
	if req.ArchivedSet && req.Archived == nil {
		return fmt.Errorf("archived không được null")
	}
	if req.ReturnCountSet && req.ReturnCount == nil {
		return fmt.Errorf("Số lần trả lại không được null")
	}
	if req.returnCountPresent() && *req.ReturnCount < 0 {
		return fmt.Errorf("Số lần trả lại phải lớn hơn hoặc bằng 0")
	}
	trimStringPtr(req.ModuleID)
	trimStringPtr(req.ParentID)
	trimStringPtr(req.FaultHolderID)
	return nil
}

func (req UpdateTaskRequest) moduleIDPresent() bool {
	return req.ModuleIDSet || req.ModuleID != nil
}

func (req UpdateTaskRequest) parentIDPresent() bool {
	return req.ParentIDSet || req.ParentID != nil
}

func (req UpdateTaskRequest) titlePresent() bool {
	return req.TitleSet || req.Title != nil
}

func (req UpdateTaskRequest) descriptionPresent() bool {
	return req.DescriptionSet || req.Description != nil
}

func (req UpdateTaskRequest) statusPresent() bool {
	return req.StatusSet || req.Status != nil
}

func (req UpdateTaskRequest) priorityPresent() bool {
	return req.PrioritySet || req.Priority != nil
}

func (req UpdateTaskRequest) startDatePresent() bool {
	return req.StartDateSet || req.StartDate != nil
}

func (req UpdateTaskRequest) endDatePresent() bool {
	return req.EndDateSet || req.EndDate != nil
}

func (req UpdateTaskRequest) estimationPresent() bool {
	return req.EstimationSet || req.Estimation != nil
}

func (req UpdateTaskRequest) archivedPresent() bool {
	return req.ArchivedSet || req.Archived != nil
}

func (req UpdateTaskRequest) faultHolderIDPresent() bool {
	return req.FaultHolderIDSet || req.FaultHolderID != nil
}

func (req UpdateTaskRequest) returnCountPresent() bool {
	return req.ReturnCountSet || req.ReturnCount != nil
}

func (req *UpdateTaskRequest) UnmarshalJSON(data []byte) error {
	type updateTaskRequest UpdateTaskRequest
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var decoded updateTaskRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*req = UpdateTaskRequest(decoded)
	_, req.ModuleIDSet = raw["moduleId"]
	_, req.ParentIDSet = raw["parentId"]
	_, req.TitleSet = raw["title"]
	_, req.DescriptionSet = raw["description"]
	_, req.StatusSet = raw["status"]
	_, req.PrioritySet = raw["priority"]
	_, req.StartDateSet = raw["startDate"]
	_, req.EndDateSet = raw["endDate"]
	_, req.EstimationSet = raw["estimation"]
	_, req.ArchivedSet = raw["archived"]
	_, req.FaultHolderIDSet = raw["faultHolderId"]
	_, req.ReturnCountSet = raw["returnCount"]
	return nil
}

func normalizeListTasksParams(params *ListTasksParams) error {
	params.ModuleID = strings.TrimSpace(params.ModuleID)
	params.Status = strings.TrimSpace(params.Status)
	params.Archived = strings.TrimSpace(params.Archived)
	params.ParentID = strings.TrimSpace(params.ParentID)
	params.CreatedByID = strings.TrimSpace(params.CreatedByID)
	params.Priority = strings.TrimSpace(params.Priority)
	params.Estimation = strings.TrimSpace(params.Estimation)
	params.StartDateFrom = strings.TrimSpace(params.StartDateFrom)
	params.StartDateTo = strings.TrimSpace(params.StartDateTo)
	params.EndDateFrom = strings.TrimSpace(params.EndDateFrom)
	params.EndDateTo = strings.TrimSpace(params.EndDateTo)
	params.LabelID = strings.TrimSpace(params.LabelID)
	params.WorkerID = strings.TrimSpace(params.WorkerID)
	params.Query = strings.TrimSpace(params.Query)

	sortRules, err := parseTaskSortParam(params.Sort)
	if err != nil {
		return err
	}
	params.SortRules = sortRules

	if params.Status != "" && !TaskStatus(params.Status).Valid() {
		return fmt.Errorf("Trạng thái phải là một trong: todo, in_progress, review, done, canceled")
	}
	if params.Archived != "" {
		if _, err := strconv.ParseBool(params.Archived); err != nil {
			return fmt.Errorf("archived phải là true hoặc false")
		}
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{name: "moduleId", value: params.ModuleID},
		{name: "parentId", value: params.ParentID},
		{name: "createdById", value: params.CreatedByID},
		{name: "labelId", value: params.LabelID},
	} {
		if item.value != "" {
			if _, err := uuid.Parse(item.value); err != nil {
				return fmt.Errorf("%s phải là UUID hợp lệ", item.name)
			}
		}
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{name: "priority", value: params.Priority},
		{name: "estimation", value: params.Estimation},
	} {
		if item.value != "" {
			parsed, err := strconv.Atoi(item.value)
			if err != nil {
				return fmt.Errorf("%s phải là số nguyên", item.name)
			}
			if parsed < 0 {
				return fmt.Errorf("%s phải lớn hơn hoặc bằng 0", item.name)
			}
		}
	}
	for _, item := range []struct {
		name  string
		value string
	}{
		{name: "startDateFrom", value: params.StartDateFrom},
		{name: "startDateTo", value: params.StartDateTo},
		{name: "endDateFrom", value: params.EndDateFrom},
		{name: "endDateTo", value: params.EndDateTo},
	} {
		if item.value != "" {
			if _, err := time.Parse(time.RFC3339, item.value); err != nil {
				return fmt.Errorf("%s phải là thời điểm theo chuẩn RFC3339", item.name)
			}
		}
	}
	return nil
}

func normalizeListMyTasksParams(params *ListMyTasksParams) error {
	params.Status = strings.TrimSpace(params.Status)
	params.Archived = strings.TrimSpace(params.Archived)
	params.Priority = strings.TrimSpace(params.Priority)
	params.Query = strings.TrimSpace(params.Query)

	sortRules, err := parseTaskSortParam(params.Sort)
	if err != nil {
		return err
	}
	params.SortRules = sortRules
	return nil
}

func trimStringPtr(value *string) {
	if value == nil {
		return
	}
	trimmed := strings.TrimSpace(*value)
	*value = trimmed
}

func taskResponseFromCreateTaskRow(row database.CreateTaskRow) TaskResponse {
	return TaskResponse{
		SequenceNo:       int(row.SequenceNo),
		ID:               pgutil.UUIDToString(row.ID),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		ProjectID:        pgutil.UUIDToString(row.ProjectID),
		ModuleID:         pgutil.UUIDToString(row.ModuleID),
		CreatedByID:      pgutil.UUIDToString(row.CreatedByID),
		ParentID:         pgutil.UUIDToString(row.ParentID),
		Title:            row.Title,
		Description:      textString(row.Description),
		Status:           TaskStatus(string(row.Status)),
		Priority:         int(row.Priority),
		StartDate:        timestamptzPtr(row.StartDate),
		EndDate:          timestamptzPtr(row.EndDate),
		FirstCompletedAt: timestamptzPtr(row.FirstCompletedAt),
		Estimation:       int(row.Estimation),
		Archived:         row.Archived,
		FaultHolderID:    pgutil.UUIDToString(row.FaultHolderID),
		ReturnCount:      int(row.ReturnCount),
		Labels:           []TaskLabelResponse{},
		Workers:          []TaskWorkerResponse{},
		FaultHolders:     []TaskFaultHolderResponse{},
	}
}

func taskResponseFromGetTaskByIDRow(row database.GetTaskByIDRow) TaskResponse {
	return TaskResponse{
		SequenceNo:       int(row.SequenceNo),
		ID:               pgutil.UUIDToString(row.ID),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		ProjectID:        pgutil.UUIDToString(row.ProjectID),
		ModuleID:         pgutil.UUIDToString(row.ModuleID),
		CreatedByID:      pgutil.UUIDToString(row.CreatedByID),
		ParentID:         pgutil.UUIDToString(row.ParentID),
		Title:            row.Title,
		Description:      textString(row.Description),
		Status:           TaskStatus(string(row.Status)),
		Priority:         int(row.Priority),
		StartDate:        timestamptzPtr(row.StartDate),
		EndDate:          timestamptzPtr(row.EndDate),
		FirstCompletedAt: timestamptzPtr(row.FirstCompletedAt),
		Estimation:       int(row.Estimation),
		Archived:         row.Archived,
		FaultHolderID:    pgutil.UUIDToString(row.FaultHolderID),
		ReturnCount:      int(row.ReturnCount),
		Labels:           []TaskLabelResponse{},
		Workers:          []TaskWorkerResponse{},
		FaultHolders:     []TaskFaultHolderResponse{},
	}
}

func taskResponseFromGetTaskByIDForUpdateRow(row database.GetTaskByIDForUpdateRow) TaskResponse {
	return TaskResponse{
		SequenceNo:       int(row.SequenceNo),
		ID:               pgutil.UUIDToString(row.ID),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		ProjectID:        pgutil.UUIDToString(row.ProjectID),
		ModuleID:         pgutil.UUIDToString(row.ModuleID),
		CreatedByID:      pgutil.UUIDToString(row.CreatedByID),
		ParentID:         pgutil.UUIDToString(row.ParentID),
		Title:            row.Title,
		Description:      textString(row.Description),
		Status:           TaskStatus(string(row.Status)),
		Priority:         int(row.Priority),
		StartDate:        timestamptzPtr(row.StartDate),
		EndDate:          timestamptzPtr(row.EndDate),
		FirstCompletedAt: timestamptzPtr(row.FirstCompletedAt),
		Estimation:       int(row.Estimation),
		Archived:         row.Archived,
		FaultHolderID:    pgutil.UUIDToString(row.FaultHolderID),
		ReturnCount:      int(row.ReturnCount),
		Labels:           []TaskLabelResponse{},
		Workers:          []TaskWorkerResponse{},
		FaultHolders:     []TaskFaultHolderResponse{},
	}
}

func taskResponseFromUpdateTaskRow(row database.UpdateTaskRow) TaskResponse {
	return TaskResponse{
		SequenceNo:       int(row.SequenceNo),
		ID:               pgutil.UUIDToString(row.ID),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		ProjectID:        pgutil.UUIDToString(row.ProjectID),
		ModuleID:         pgutil.UUIDToString(row.ModuleID),
		CreatedByID:      pgutil.UUIDToString(row.CreatedByID),
		ParentID:         pgutil.UUIDToString(row.ParentID),
		Title:            row.Title,
		Description:      textString(row.Description),
		Status:           TaskStatus(string(row.Status)),
		Priority:         int(row.Priority),
		StartDate:        timestamptzPtr(row.StartDate),
		EndDate:          timestamptzPtr(row.EndDate),
		FirstCompletedAt: timestamptzPtr(row.FirstCompletedAt),
		Estimation:       int(row.Estimation),
		Archived:         row.Archived,
		FaultHolderID:    pgutil.UUIDToString(row.FaultHolderID),
		ReturnCount:      int(row.ReturnCount),
		Labels:           []TaskLabelResponse{},
		Workers:          []TaskWorkerResponse{},
		FaultHolders:     []TaskFaultHolderResponse{},
	}
}

type taskListRow struct {
	ID               pgtype.UUID
	CreatedAt        pgtype.Timestamptz
	UpdatedAt        pgtype.Timestamptz
	DeletedAt        pgtype.Timestamptz
	ProjectID        pgtype.UUID
	ModuleID         pgtype.UUID
	CreatedByID      pgtype.UUID
	ParentID         pgtype.UUID
	Title            string
	Description      pgtype.Text
	Status           interface{}
	Priority         int32
	StartDate        pgtype.Timestamptz
	EndDate          pgtype.Timestamptz
	FirstCompletedAt pgtype.Timestamptz
	Estimation       int32
	Archived         bool
	ReturnCount      int32
	FaultHolderID    pgtype.UUID
	SequenceNo       int32
}

func taskResponseFromListRow(row taskListRow) TaskResponse {
	return TaskResponse{
		SequenceNo:       int(row.SequenceNo),
		ID:               pgutil.UUIDToString(row.ID),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
		ProjectID:        pgutil.UUIDToString(row.ProjectID),
		ModuleID:         pgutil.UUIDToString(row.ModuleID),
		CreatedByID:      pgutil.UUIDToString(row.CreatedByID),
		ParentID:         pgutil.UUIDToString(row.ParentID),
		Title:            row.Title,
		Description:      textString(row.Description),
		Status:           TaskStatus(row.Status.(string)),
		Priority:         int(row.Priority),
		StartDate:        timestamptzPtr(row.StartDate),
		EndDate:          timestamptzPtr(row.EndDate),
		FirstCompletedAt: timestamptzPtr(row.FirstCompletedAt),
		Estimation:       int(row.Estimation),
		Archived:         row.Archived,
		FaultHolderID:    pgutil.UUIDToString(row.FaultHolderID),
		ReturnCount:      int(row.ReturnCount),
		Labels:           []TaskLabelResponse{},
		Workers:          []TaskWorkerResponse{},
		FaultHolders:     []TaskFaultHolderResponse{},
	}
}

func taskLabelResponseFromCreateRow(row database.TaskLabel) TaskLabelResponse {
	return TaskLabelResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		CreatedByID: pgutil.UUIDToString(row.CreatedByID),
		Name:        row.Name,
		Color:       row.Color,
		ProjectID:   pgutil.UUIDToString(row.ProjectID),
	}
}

func taskLabelResponseFromUpdateRow(row database.TaskLabel) TaskLabelResponse {
	return TaskLabelResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		CreatedByID: pgutil.UUIDToString(row.CreatedByID),
		Name:        row.Name,
		Color:       row.Color,
		ProjectID:   pgutil.UUIDToString(row.ProjectID),
	}
}

func taskLabelResponseFromListTaskRow(row database.TaskLabel) TaskLabelResponse {
	return TaskLabelResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		CreatedByID: pgutil.UUIDToString(row.CreatedByID),
		Name:        row.Name,
		Color:       row.Color,
		ProjectID:   pgutil.UUIDToString(row.ProjectID),
	}
}

func taskLabelResponseFromBatchRow(row database.ListTaskLabelsByTaskIDsRow) TaskLabelResponse {
	return TaskLabelResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		CreatedByID: pgutil.UUIDToString(row.CreatedByID),
		Name:        row.Name,
		Color:       row.Color,
		ProjectID:   pgutil.UUIDToString(row.ProjectID),
	}
}

func taskLabelResponseFromListProjectRow(row database.TaskLabel) TaskLabelResponse {
	return TaskLabelResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		CreatedByID: pgutil.UUIDToString(row.CreatedByID),
		Name:        row.Name,
		Color:       row.Color,
		ProjectID:   pgutil.UUIDToString(row.ProjectID),
	}
}

func taskWorkerResponseFromListRow(row database.ListTaskWorkersWithUserRow) TaskWorkerResponse {
	userID := ""
	if row.UserID.Valid {
		userID = uuid.UUID(row.UserID.Bytes).String()
	}
	return TaskWorkerResponse{
		TaskID: pgutil.UUIDToString(row.TaskID),
		Weight: int(row.Weight),
		User: user.UserSummary{
			ID:     userID,
			Name:   row.Name,
			Alias:  row.Alias,
			Avatar: textString(row.Avatar),
		},
	}
}

func taskWorkerResponseFromBatchRow(row database.ListTaskWorkersWithUserByTaskIDsRow) TaskWorkerResponse {
	userID := ""
	if row.UserID.Valid {
		userID = uuid.UUID(row.UserID.Bytes).String()
	}
	return TaskWorkerResponse{
		TaskID: pgutil.UUIDToString(row.TaskID),
		Weight: int(row.Weight),
		User: user.UserSummary{
			ID:     userID,
			Name:   row.Name,
			Alias:  row.Alias,
			Avatar: textString(row.Avatar),
		},
	}
}

func taskWorkerResponseFromAssignRow(row database.AssignTaskWorkerRow) TaskWorkerResponse {
	userID := ""
	if row.UserID.Valid {
		userID = uuid.UUID(row.UserID.Bytes).String()
	}
	return TaskWorkerResponse{
		TaskID: pgutil.UUIDToString(row.TaskID),
		Weight: int(row.Weight),
		User: user.UserSummary{
			ID:     userID,
			Name:   row.Name,
			Alias:  row.Alias,
			Avatar: textString(row.Avatar),
		},
	}
}

func taskFaultHolderResponseFromListRow(row database.ListTaskFaultHoldersWithUserRow) TaskFaultHolderResponse {
	userID := ""
	if row.UserID.Valid {
		userID = uuid.UUID(row.UserID.Bytes).String()
	}
	return TaskFaultHolderResponse{
		TaskID: pgutil.UUIDToString(row.TaskID),
		Weight: int(row.Weight),
		User: user.UserSummary{
			ID:     userID,
			Name:   row.Name,
			Alias:  row.Alias,
			Avatar: textString(row.Avatar),
		},
	}
}

func taskFaultHolderResponseFromBatchRow(row database.ListTaskFaultHoldersWithUserByTaskIDsRow) TaskFaultHolderResponse {
	userID := ""
	if row.UserID.Valid {
		userID = uuid.UUID(row.UserID.Bytes).String()
	}
	return TaskFaultHolderResponse{
		TaskID: pgutil.UUIDToString(row.TaskID),
		Weight: int(row.Weight),
		User: user.UserSummary{
			ID:     userID,
			Name:   row.Name,
			Alias:  row.Alias,
			Avatar: textString(row.Avatar),
		},
	}
}

func taskFaultHolderResponseFromAssignRow(row database.AssignTaskFaultHolderRow) TaskFaultHolderResponse {
	userID := ""
	if row.UserID.Valid {
		userID = uuid.UUID(row.UserID.Bytes).String()
	}
	return TaskFaultHolderResponse{
		TaskID: pgutil.UUIDToString(row.TaskID),
		Weight: int(row.Weight),
		User: user.UserSummary{
			ID:     userID,
			Name:   row.Name,
			Alias:  row.Alias,
			Avatar: textString(row.Avatar),
		},
	}
}

func taskHistoryResponseFromListRow(row database.ListTaskHistoriesRow) TaskHistoryResponse {
	return taskHistoryResponseFromRow(taskHistoryRow{
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

func pgtypeText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}
