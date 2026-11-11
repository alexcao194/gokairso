package device

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alexcao194/kairso/internal/i18n"
	"github.com/alexcao194/kairso/pkg/firebase"
)

type Service interface {
	Register(ctx context.Context, userID string, req RegisterDeviceRequest) error
	Unregister(ctx context.Context, userID, token string) error
	// PushToUser implements notification.Pusher.
	PushToUser(ctx context.Context, userID, defaultTitle, defaultBody string, data map[string]string)
}

type service struct {
	repo      Repository
	messenger firebase.Messenger
}

// NewService builds the device service. messenger may be nil, in which case
// push delivery is disabled (tokens are still stored).
func NewService(repo Repository, messenger firebase.Messenger) *service {
	return &service{repo: repo, messenger: messenger}
}

func (s *service) Register(ctx context.Context, userID string, req RegisterDeviceRequest) error {
	if req.Token == "" {
		return fmt.Errorf("token is required")
	}
	platform := req.Platform
	if platform == "" {
		platform = "web"
	}
	return s.repo.Upsert(ctx, userID, req.Token, platform, i18n.Normalize(req.Locale))
}

func (s *service) Unregister(ctx context.Context, userID, token string) error {
	return s.repo.Delete(ctx, token)
}

func (s *service) PushToUser(ctx context.Context, userID, defaultTitle, defaultBody string, data map[string]string) {
	if s.messenger == nil {
		return
	}
	tokens, err := s.repo.ListByUser(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return
	}
	ntype := data["type"]
	msgs := make([]firebase.PushMessage, 0, len(tokens))
	for _, t := range tokens {
		title, body := defaultTitle, defaultBody
		// defaultTitle/Body are already rendered in the default locale; only
		// re-render for devices on a different locale.
		if t.Locale != i18n.Default {
			key := ntype
			if data["mentioned"] == "true" && key == "task_comment_added" {
				key = "task_mentioned"
			}
			if rt, rb := i18n.RenderNotification(t.Locale, key, data); rt != "" {
				title, body = rt, rb
			}
		}
		msgs = append(msgs, firebase.PushMessage{Token: t.Token, Title: title, Body: body, Data: data})
	}
	invalid, err := s.messenger.Send(ctx, msgs)
	if err != nil {
		slog.Warn("push notification send failed", "err", err, "userId", userID)
		return
	}
	if len(invalid) > 0 {
		_ = s.repo.DeleteMany(ctx, invalid)
	}
}
