package webhook

import (
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
)

// EventTest is the type of the event sent by the "send test" action.
const EventTest = "webhook.test"

type WebhookRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	// Token authenticates deliveries. Left empty on create, one is generated.
	Token   string `json:"token"`
	Enabled *bool  `json:"enabled,omitempty"`
}

type WebhookResponse struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	URL             string     `json:"url"`
	Token           string     `json:"token"`
	Enabled         bool       `json:"enabled"`
	FailureCount    int        `json:"failureCount"`
	LastStatus      *int       `json:"lastStatus,omitempty"`
	LastError       string     `json:"lastError,omitempty"`
	LastDeliveredAt *time.Time `json:"lastDeliveredAt,omitempty" swaggertype:"string" format:"date-time"`
	CreatedAt       time.Time  `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt       time.Time  `json:"updatedAt" swaggertype:"string" format:"date-time"`
}

type WebhookDeliveryResponse struct {
	ID         string    `json:"id,omitempty"`
	EventID    string    `json:"eventId"`
	EventType  string    `json:"eventType"`
	Success    bool      `json:"success"`
	StatusCode int       `json:"statusCode"`
	Error      string    `json:"error,omitempty"`
	Attempts   int       `json:"attempts"`
	DurationMs int       `json:"durationMs"`
	CreatedAt  time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
}

// Payload is the JSON body POSTed to a webhook URL.
type Payload struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	WorkspaceID string    `json:"workspaceId"`
	EntityID    string    `json:"entityId,omitempty"`
	ActorID     string    `json:"actorId,omitempty"`
	OccurredAt  time.Time `json:"occurredAt" swaggertype:"string" format:"date-time"`
	Data        any       `json:"data,omitempty"`
}

func webhookResponse(w database.Webhook) WebhookResponse {
	resp := WebhookResponse{
		ID:           pgutil.UUIDToString(w.ID),
		Name:         w.Name,
		URL:          w.Url,
		Token:        w.Token,
		Enabled:      w.Enabled,
		FailureCount: int(w.FailureCount),
		LastError:    w.LastError,
		CreatedAt:    w.CreatedAt.Time,
		UpdatedAt:    w.UpdatedAt.Time,
	}
	if w.LastStatus.Valid {
		v := int(w.LastStatus.Int32)
		resp.LastStatus = &v
	}
	if w.LastDeliveredAt.Valid {
		t := w.LastDeliveredAt.Time
		resp.LastDeliveredAt = &t
	}
	return resp
}

func deliveryResponse(d database.ListWebhookDeliveriesRow) WebhookDeliveryResponse {
	return WebhookDeliveryResponse{
		ID:         pgutil.UUIDToString(d.ID),
		EventID:    pgutil.UUIDToString(d.EventID),
		EventType:  d.EventType,
		Success:    d.Success,
		StatusCode: int(d.StatusCode),
		Error:      d.Error,
		Attempts:   int(d.Attempts),
		DurationMs: int(d.DurationMs),
		CreatedAt:  d.CreatedAt.Time,
	}
}
