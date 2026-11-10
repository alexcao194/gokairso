package task

import (
	"context"
	"fmt"
	"time"
)

const deadlineReminderInterval = 5 * time.Minute

type DeadlineReminderService interface {
	RunDue(ctx context.Context, now time.Time) error
	Start(ctx context.Context)
}

type deadlineReminderService struct {
	repo      Repository
	publisher NotificationPublisher
}

func NewDeadlineReminderService(repo Repository, publisher NotificationPublisher) DeadlineReminderService {
	return &deadlineReminderService{repo: repo, publisher: publisher}
}

func (s *deadlineReminderService) RunDue(ctx context.Context, now time.Time) error {
	windowStart := now.UTC().Add(24 * time.Hour)
	windowEnd := windowStart.Add(deadlineReminderInterval)
	reminders, err := s.repo.ClaimDueTaskDeadlineReminders(ctx, windowStart, windowEnd)
	if err != nil {
		return err
	}
	for _, reminder := range reminders {
		workerIDs, err := s.repo.ListTaskWorkerUserIDs(ctx, reminder.TaskID)
		if err != nil {
			return err
		}
		for _, workerID := range workerIDs {
			if err := s.publisher.PublishTaskNotification(ctx, TaskNotificationEvent{
				Type:        NotificationTypeTaskDeadlineReminder,
				RecipientID: workerID,
				WorkspaceID: reminder.WorkspaceID,
				TaskID:      reminder.TaskID,
				ProjectID:   reminder.ProjectID,
				Title:       reminder.Title,
				DueAt:       reminder.EndDate,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *deadlineReminderService) Start(ctx context.Context) {
	go func() {
		run := func() {
			if err := s.RunDue(ctx, time.Now()); err != nil {
				fmt.Printf("task deadline reminder failed: %v\n", err)
			}
		}
		run()
		ticker := time.NewTicker(deadlineReminderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}
