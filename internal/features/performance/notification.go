package performance

import (
	"context"

	"github.com/alexcao194/kairso/internal/features/notification"
)

type NotificationPublisher interface {
	PublishPerformanceNotification(ctx context.Context, event PerformanceNotificationEvent) error
}

type notificationPublisher struct {
	svc notification.Service
}

func NewNotificationPublisher(svc notification.Service) NotificationPublisher {
	return notificationPublisher{svc: svc}
}

func (p notificationPublisher) PublishPerformanceNotification(ctx context.Context, event PerformanceNotificationEvent) error {
	return p.svc.Publish(ctx, event.ToNotificationEvent())
}
