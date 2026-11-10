package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/realtime"
)

type Service interface {
	ListNotifications(ctx context.Context, workspaceID string, userID string, params pagination.Params) (pagination.Response[NotificationResponse], error)
	ReadNotifications(ctx context.Context, workspaceID string, userID string, req ReadNotificationsRequest) error
	GetUnreadCount(ctx context.Context, workspaceID string, userID string) (int64, error)
	Publish(ctx context.Context, event NotificationEvent) error
	HandleNotificationEvent(ctx context.Context, event NotificationEvent) (*NotificationResponse, error)
}

type service struct {
	repo      Repository
	publisher message.Publisher
	rt        realtime.Broadcaster
}

func NewService(repo Repository, publisher message.Publisher) *service {
	return &service{repo: repo, publisher: publisher}
}

func (s *service) SetRealtime(rt realtime.Broadcaster) {
	s.rt = rt
}

func (s *service) broadcastUnreadCount(ctx context.Context, workspaceID, userID, eventType string) {
	if s.rt == nil {
		return
	}
	count, err := s.repo.CountUnreadNotifications(ctx, workspaceID, userID)
	if err != nil {
		return
	}
	s.rt.Broadcast(realtime.Event{
		Type:        eventType,
		WorkspaceID: workspaceID,
		EntityID:    userID,
		ActorConnID: middleware.GetConnIDFromContext(ctx),
		Entity:      map[string]any{"unreadCount": count},
	})
}

func (s *service) ListNotifications(ctx context.Context, workspaceID string, userID string, params pagination.Params) (pagination.Response[NotificationResponse], error) {
	return s.repo.ListNotifications(ctx, workspaceID, userID, params)
}

func (s *service) ReadNotifications(ctx context.Context, workspaceID string, userID string, req ReadNotificationsRequest) error {
	err := s.repo.ReadNotifications(ctx, workspaceID, userID, req.IDs)
	if err == nil {
		s.broadcastUnreadCount(ctx, workspaceID, userID, "notification.read")
	}
	return err
}

func (s *service) GetUnreadCount(ctx context.Context, workspaceID string, userID string) (int64, error) {
	return s.repo.CountUnreadNotifications(ctx, workspaceID, userID)
}

func (s *service) Publish(ctx context.Context, event NotificationEvent) error {
	if err := normalizeNotificationEvent(&event); err != nil {
		return err
	}
	if s.publisher == nil {
		return fmt.Errorf("notification publisher is not configured")
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	msg := message.NewMessage(watermill.NewUUID(), payload)
	go func() {
		_ = s.publisher.Publish(TopicSendNotification, msg)
	}()
	return nil
}

func (s *service) StartConsumer(ctx context.Context, subscriber message.Subscriber) error {
	messages, err := subscriber.Subscribe(ctx, TopicSendNotification)
	if err != nil {
		return err
	}
	go func() {
		for msg := range messages {
			var event NotificationEvent
			if err := json.Unmarshal(msg.Payload, &event); err != nil {
				msg.Nack()
				continue
			}
			if _, err := s.HandleNotificationEvent(ctx, event); err != nil {
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}()
	return nil
}

func (s *service) HandleNotificationEvent(ctx context.Context, event NotificationEvent) (*NotificationResponse, error) {
	if err := normalizeNotificationEvent(&event); err != nil {
		return nil, err
	}
	resp, err := s.repo.CreateNotification(ctx, event)
	if err == nil {
		s.broadcastUnreadCount(ctx, event.WorkspaceID, event.UserID, "notification.created")
	}
	return resp, err
}
