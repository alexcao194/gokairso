package task

import (
	"context"

	"github.com/alexcao194/kairso/internal/features/notification"
)

type notificationPublisher struct {
	svc notification.Service
}

func NewNotificationPublisher(svc notification.Service) NotificationPublisher {
	return notificationPublisher{svc: svc}
}

func (p notificationPublisher) PublishTaskNotification(ctx context.Context, event TaskNotificationEvent) error {
	return p.svc.Publish(ctx, event.ToNotificationEvent())
}
