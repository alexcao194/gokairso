package attendance

import (
	"context"
	"fmt"

	"github.com/alexcao194/kairso/internal/features/notification"
)

type notificationPublisher struct {
	svc notification.Service
}

func NewNotificationPublisher(svc notification.Service) ScheduleNotificationPublisher {
	return notificationPublisher{svc: svc}
}

func (p notificationPublisher) PublishWorkScheduleRegistration(ctx context.Context, event ScheduleNotificationEvent) error {
	registrant := event.MemberName
	if registrant == "" {
		registrant = "Một thành viên workspace"
	}
	return p.svc.Publish(ctx, notification.NotificationEvent{
		UserID:      event.RecipientID,
		WorkspaceID: event.WorkspaceID,
		Title:       "Đăng ký lịch làm việc",
		Body:        fmt.Sprintf("%s đã đăng ký lịch làm việc từ %s đến %s", registrant, event.FromDate, event.ToDate),
		Data: map[string]string{
			"type":        event.Type,
			"workspaceId": event.WorkspaceID,
			"memberId":    event.MemberID,
			"fromDate":    event.FromDate,
			"toDate":      event.ToDate,
		},
	})
}

func (p notificationPublisher) PublishWorkScheduleDecision(ctx context.Context, event ScheduleNotificationEvent) error {
	title := "Lịch làm việc đã được duyệt"
	body := fmt.Sprintf("Lịch làm việc từ %s đến %s của bạn đã được duyệt.", event.FromDate, event.ToDate)
	if event.Type == NotificationTypeWorkScheduleRejected {
		title = "Lịch làm việc bị từ chối"
		body = fmt.Sprintf("Lịch làm việc từ %s đến %s của bạn bị từ chối.", event.FromDate, event.ToDate)
	}
	return p.svc.Publish(ctx, notification.NotificationEvent{
		UserID:      event.RecipientID,
		WorkspaceID: event.WorkspaceID,
		Title:       title,
		Body:        body,
		Data: map[string]string{
			"type":        event.Type,
			"workspaceId": event.WorkspaceID,
			"memberId":    event.MemberID,
			"fromDate":    event.FromDate,
			"toDate":      event.ToDate,
		},
	})
}

func describeCheckRequestSession(event CheckRequestNotificationEvent) string {
	switch {
	case event.CheckInAt != "" && event.CheckOutAt != "":
		return fmt.Sprintf("từ %s đến %s", event.CheckInAt, event.CheckOutAt)
	case event.CheckInAt != "":
		return fmt.Sprintf("check-in lúc %s", event.CheckInAt)
	default:
		return fmt.Sprintf("check-out lúc %s", event.CheckOutAt)
	}
}

func (p notificationPublisher) PublishDailyReportReminder(ctx context.Context, event DailyReportReminderEvent) error {
	return p.svc.Publish(ctx, notification.NotificationEvent{
		UserID:      event.RecipientID,
		WorkspaceID: event.WorkspaceID,
		Title:       "Nhắc nộp báo cáo cuối ngày",
		Body:        "Bạn sắp hết giờ làm hôm nay mà chưa nộp báo cáo cuối ngày. Hãy điền báo cáo khi check-out.",
		Data: map[string]string{
			"type":        NotificationTypeDailyReportReminder,
			"workspaceId": event.WorkspaceID,
			"reportDate":  event.Date,
		},
	})
}

func (p notificationPublisher) PublishCheckRequestSubmitted(ctx context.Context, event CheckRequestNotificationEvent) error {
	requester := event.MemberName
	if requester == "" {
		requester = "Một thành viên workspace"
	}
	return p.svc.Publish(ctx, notification.NotificationEvent{
		UserID:      event.RecipientID,
		WorkspaceID: event.WorkspaceID,
		Title:       "Yêu cầu bổ sung công",
		Body:        fmt.Sprintf("%s đã gửi yêu cầu bổ sung công: %s", requester, describeCheckRequestSession(event)),
		Data: map[string]string{
			"type":        event.Type,
			"workspaceId": event.WorkspaceID,
			"memberId":    event.MemberID,
			"targetID":    event.RequestID,
			"checkInAt":   event.CheckInAt,
			"checkOutAt":  event.CheckOutAt,
		},
	})
}

func (p notificationPublisher) PublishCheckRequestDecision(ctx context.Context, event CheckRequestNotificationEvent) error {
	title := "Yêu cầu bổ sung công đã được duyệt"
	body := fmt.Sprintf("Yêu cầu bổ sung công (%s) của bạn đã được duyệt.", describeCheckRequestSession(event))
	if event.Type == NotificationTypeCheckRequestRejected {
		title = "Yêu cầu bổ sung công bị từ chối"
		body = fmt.Sprintf("Yêu cầu bổ sung công (%s) của bạn bị từ chối.", describeCheckRequestSession(event))
	}
	return p.svc.Publish(ctx, notification.NotificationEvent{
		UserID:      event.RecipientID,
		WorkspaceID: event.WorkspaceID,
		Title:       title,
		Body:        body,
		Data: map[string]string{
			"type":        event.Type,
			"workspaceId": event.WorkspaceID,
			"memberId":    event.MemberID,
			"targetID":    event.RequestID,
			"checkInAt":   event.CheckInAt,
			"checkOutAt":  event.CheckOutAt,
		},
	})
}

func (p notificationPublisher) PublishAttendanceReminder(ctx context.Context, event AttendanceReminderEvent) error {
	title := "Nhắc chấm công"
	body := fmt.Sprintf("Ca làm việc của bạn bắt đầu lúc %s. Hãy check-in.", event.SessionTime)
	if event.Type == NotificationTypeAttendanceCheckOutReminder {
		body = fmt.Sprintf("Ca làm việc của bạn kết thúc lúc %s. Hãy check-out.", event.SessionTime)
	}
	return p.svc.Publish(ctx, notification.NotificationEvent{
		UserID: event.RecipientID, WorkspaceID: event.WorkspaceID, Title: title, Body: body,
		Data: map[string]string{
			"type": event.Type, "workspaceId": event.WorkspaceID, "sessionId": event.SessionID,
			"scheduleDate": event.ScheduleDate, "sessionAt": event.SessionTime,
		},
	})
}
