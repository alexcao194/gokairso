package firebase

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// PushMessage is one localized push notification for a single device token.
type PushMessage struct {
	Token string
	Title string
	Body  string
	Data  map[string]string
}

// Messenger delivers push notifications via Firebase Cloud Messaging.
type Messenger interface {
	// Send delivers the messages and returns the subset of tokens that FCM
	// reports as no longer valid, so the caller can purge them.
	Send(ctx context.Context, msgs []PushMessage) (invalid []string, err error)
}

type messenger struct {
	client *messaging.Client
}

// NewMessenger builds a Messenger from a Firebase service-account credentials file.
func NewMessenger(ctx context.Context, credentialsPath string) (Messenger, error) {
	if credentialsPath == "" {
		return nil, fmt.Errorf("credentials path is required")
	}
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsPath))
	if err != nil {
		return nil, fmt.Errorf("initialize firebase app: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize firebase messaging client: %w", err)
	}
	return &messenger{client: client}, nil
}

func (m *messenger) Send(ctx context.Context, msgs []PushMessage) ([]string, error) {
	if len(msgs) == 0 {
		return nil, nil
	}
	fcm := make([]*messaging.Message, 0, len(msgs))
	for _, msg := range msgs {
		fcm = append(fcm, &messaging.Message{
			Token:        msg.Token,
			Notification: &messaging.Notification{Title: msg.Title, Body: msg.Body},
			Data:         msg.Data,
		})
	}
	resp, err := m.client.SendEach(ctx, fcm)
	if err != nil {
		return nil, err
	}
	var invalid []string
	for i, r := range resp.Responses {
		if r.Success {
			continue
		}
		if messaging.IsUnregistered(r.Error) || messaging.IsRegistrationTokenNotRegistered(r.Error) {
			invalid = append(invalid, msgs[i].Token)
		}
	}
	return invalid, nil
}
