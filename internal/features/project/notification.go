package project

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

func (p notificationPublisher) PublishProjectNotification(ctx context.Context, event ProjectNotificationEvent) error {
	return p.svc.Publish(ctx, event.ToNotificationEvent())
}
