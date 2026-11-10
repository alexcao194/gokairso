package notification

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
)

const TopicSendNotification = "notifications.send"

type ReadNotificationsRequest struct {
	IDs []string `json:"ids"`
}

type NotificationEvent struct {
	UserID      string            `json:"userId"`
	WorkspaceID string            `json:"workspaceId"`
	Title       string            `json:"title"`
	Body        string            `json:"body,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
}

type UnreadCountResponse struct {
	UnreadCount int64 `json:"unreadCount"`
}

type NotificationResponse struct {
	ID          string            `json:"id"`
	CreatedAt   time.Time         `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time         `json:"updatedAt" swaggertype:"string" format:"date-time"`
	UserID      string            `json:"userId"`
	WorkspaceID string            `json:"workspaceId"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Data        map[string]string `json:"data"`
	ReadAt      *time.Time        `json:"readAt,omitempty" swaggertype:"string" format:"date-time"`
}

type notificationRow struct {
	ID          pgtype.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	WorkspaceID pgtype.UUID
	UserID      pgtype.UUID
	Title       string
	Body        string
	Data        []byte
	ReadAt      pgtype.Timestamptz
}

func normalizeNotificationEvent(event *NotificationEvent) error {
	event.UserID = strings.TrimSpace(event.UserID)
	event.WorkspaceID = strings.TrimSpace(event.WorkspaceID)
	event.Title = strings.TrimSpace(event.Title)
	event.Body = strings.TrimSpace(event.Body)
	if event.Data == nil {
		event.Data = map[string]string{}
	}
	if event.UserID == "" {
		return fmt.Errorf("userId không được để trống")
	}
	if event.WorkspaceID == "" {
		return fmt.Errorf("workspaceId không được để trống")
	}
	if event.Title == "" {
		return fmt.Errorf("Tiêu đề không được để trống")
	}
	return nil
}

func notificationResponseFromDatabaseNotification(row database.Notification) (NotificationResponse, error) {
	return notificationResponseFromRow(notificationRow{
		ID:          row.ID,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		Title:       row.Title,
		Body:        row.Body,
		Data:        row.Data,
		ReadAt:      row.ReadAt,
	})
}

func notificationResponseFromRow(row notificationRow) (NotificationResponse, error) {
	data := map[string]string{}
	if len(row.Data) > 0 {
		if err := json.Unmarshal(row.Data, &data); err != nil {
			return NotificationResponse{}, err
		}
	}
	return NotificationResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		UserID:      pgutil.UUIDToString(row.UserID),
		WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
		Title:       row.Title,
		Body:        row.Body,
		Data:        data,
		ReadAt:      timestamptzPtr(row.ReadAt),
	}, nil
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}
