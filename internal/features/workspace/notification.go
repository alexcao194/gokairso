package workspace

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

func (p notificationPublisher) PublishWorkspaceNotification(ctx context.Context, event WorkspaceNotificationEvent) error {
	return p.svc.Publish(ctx, event.ToNotificationEvent())
}
