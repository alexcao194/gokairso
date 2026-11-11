package attendance

import (
	"context"
	"fmt"

	"github.com/alexcao194/kairso/internal/features/notification"
	"github.com/alexcao194/kairso/internal/i18n"
)

type notificationPublisher struct {
	svc notification.Service
}

func NewNotificationPublisher(svc notification.Service) ScheduleNotificationPublisher {
	return notificationPublisher{svc: svc}
}

// emit renders the title/body in the default locale from the catalog and
// publishes the notification. params feed the message template; extra carries
// non-template data (ids, raw values) merged into the stored data map.
func (p notificationPublisher) emit(ctx context.Context, recipientID, workspaceID, ntype string, params, extra map[string]string) error {
	data := map[string]string{"type": ntype, "workspaceId": workspaceID}
	for k, v := range params {
		data[k] = v
	}
	for k, v := range extra {
		data[k] = v
	}
	title, body := i18n.RenderNotification(i18n.Default, ntype, params)
	return p.svc.Publish(ctx, notification.NotificationEvent{
		UserID:      recipientID,
		WorkspaceID: workspaceID,
		Title:       title,
		Body:        body,
		Data:        data,
	})
}

func (p notificationPublisher) PublishWorkScheduleRegistration(ctx context.Context, event ScheduleNotificationEvent) error {
	registrant := event.MemberName
	if registrant == "" {
		registrant = "Một thành viên workspace"
	}
	return p.emit(ctx, event.RecipientID, event.WorkspaceID, event.Type,
		map[string]string{"registrant": registrant, "fromDate": event.FromDate, "toDate": event.ToDate},
		map[string]string{"memberId": event.MemberID})
}

func (p notificationPublisher) PublishWorkScheduleDecision(ctx context.Context, event ScheduleNotificationEvent) error {
	return p.emit(ctx, event.RecipientID, event.WorkspaceID, event.Type,
		map[string]string{"fromDate": event.FromDate, "toDate": event.ToDate},
		map[string]string{"memberId": event.MemberID})
}

// describeCheckRequestSession builds a short, localized phrase for the session a
// correction request refers to, from its check-in/out times.
func describeCheckRequestSession(locale string, event CheckRequestNotificationEvent) string {
	en := i18n.Normalize(locale) == "en"
	switch {
	case event.CheckInAt != "" && event.CheckOutAt != "":
		if en {
			return fmt.Sprintf("from %s to %s", event.CheckInAt, event.CheckOutAt)
		}
		return fmt.Sprintf("từ %s đến %s", event.CheckInAt, event.CheckOutAt)
	case event.CheckInAt != "":
		if en {
			return fmt.Sprintf("check-in at %s", event.CheckInAt)
		}
		return fmt.Sprintf("check-in lúc %s", event.CheckInAt)
	default:
		if en {
			return fmt.Sprintf("check-out at %s", event.CheckOutAt)
		}
		return fmt.Sprintf("check-out lúc %s", event.CheckOutAt)
	}
}

func (p notificationPublisher) PublishDailyReportReminder(ctx context.Context, event DailyReportReminderEvent) error {
	return p.emit(ctx, event.RecipientID, event.WorkspaceID, NotificationTypeDailyReportReminder,
		nil, map[string]string{"reportDate": event.Date})
}

func (p notificationPublisher) PublishCheckRequestSubmitted(ctx context.Context, event CheckRequestNotificationEvent) error {
	requester := event.MemberName
	if requester == "" {
		requester = "Một thành viên workspace"
	}
	return p.emit(ctx, event.RecipientID, event.WorkspaceID, event.Type,
		map[string]string{"requester": requester, "session": describeCheckRequestSession(i18n.Default, event)},
		map[string]string{"memberId": event.MemberID, "targetID": event.RequestID, "checkInAt": event.CheckInAt, "checkOutAt": event.CheckOutAt})
}

func (p notificationPublisher) PublishCheckRequestDecision(ctx context.Context, event CheckRequestNotificationEvent) error {
	return p.emit(ctx, event.RecipientID, event.WorkspaceID, event.Type,
		map[string]string{"session": describeCheckRequestSession(i18n.Default, event)},
		map[string]string{"memberId": event.MemberID, "targetID": event.RequestID, "checkInAt": event.CheckInAt, "checkOutAt": event.CheckOutAt})
}

func (p notificationPublisher) PublishAttendanceReminder(ctx context.Context, event AttendanceReminderEvent) error {
	return p.emit(ctx, event.RecipientID, event.WorkspaceID, event.Type,
		map[string]string{"sessionTime": event.SessionTime},
		map[string]string{"sessionId": event.SessionID, "scheduleDate": event.ScheduleDate, "sessionAt": event.SessionTime})
}
