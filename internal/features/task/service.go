package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/realtime"
)

type Service interface {
	CreateTask(ctx context.Context, projectAlias, userID string, req CreateTaskRequest) (*TaskResponse, error)
	ListTasks(ctx context.Context, projectAlias, viewerID string, filter ListTasksParams, params pagination.Params) (TaskListResponse, error)
	GetTask(ctx context.Context, taskID string) (*TaskResponse, error)
	AuthorizeTaskAccess(ctx context.Context, taskID, callerID string) error
	AuthorizeProjectAccess(ctx context.Context, projectID, callerID string) error
	UpdateTask(ctx context.Context, taskID, callerID string, req UpdateTaskRequest) (*TaskResponse, error)
	ListTaskHistory(ctx context.Context, taskID string, params pagination.Params) (pagination.Response[TaskHistoryResponse], error)
	DeleteTask(ctx context.Context, taskID string) error

	AddAttachment(ctx context.Context, taskID, uploaderID string, req CreateTaskAttachmentRequest) (*TaskAttachmentResponse, error)
	ListAttachments(ctx context.Context, taskID string) ([]TaskAttachmentResponse, error)
	DeleteAttachment(ctx context.Context, attachmentID string) error

	AddComment(ctx context.Context, taskID, authorID string, req CreateTaskCommentRequest) (*TaskCommentResponse, error)
	ListComments(ctx context.Context, taskID, callerID string) ([]TaskCommentResponse, error)
	UpdateComment(ctx context.Context, commentID, callerID string, req UpdateTaskCommentRequest) (*TaskCommentResponse, error)
	DeleteComment(ctx context.Context, commentID, callerID string) error

	CreateTaskLabel(ctx context.Context, projectAlias, userID string, req CreateTaskLabelRequest) (*TaskLabelResponse, error)
	ListTaskLabels(ctx context.Context, projectAlias string, params pagination.Params) (pagination.Response[TaskLabelResponse], error)
	ListTaskLabelsByProjectIDs(ctx context.Context, projectIDs []string, callerID string) ([]TaskLabelResponse, error)
	AuthorizeLabelAccess(ctx context.Context, labelID, callerID string) error
	UpdateTaskLabel(ctx context.Context, labelID string, req UpdateTaskLabelRequest) (*TaskLabelResponse, error)
	DeleteTaskLabel(ctx context.Context, labelID string) error
	AttachTaskLabel(ctx context.Context, taskID, labelID string) error
	DetachTaskLabel(ctx context.Context, taskID, labelID string) error

	AssignTaskWorker(ctx context.Context, taskID, callerID, userID string, weight int) (*TaskWorkerResponse, error)
	RemoveTaskWorker(ctx context.Context, taskID, callerID, userID string) error

	ListMyTasks(ctx context.Context, workspaceID, userID string, filter ListMyTasksParams, params pagination.Params) (TaskListResponse, error)
	CountMyActiveTasks(ctx context.Context, workspaceID, userID string) (int64, error)
	ListThreads(ctx context.Context, workspaceID, userID string, params pagination.Params) (pagination.Response[ThreadResponse], error)
	CountThreads(ctx context.Context, workspaceID, userID string) (int64, error)

	AssignTaskFaultHolder(ctx context.Context, taskID, callerID, userID string, weight int) (*TaskFaultHolderResponse, error)
	RemoveTaskFaultHolder(ctx context.Context, taskID, callerID, userID string) error
}

type NotificationPublisher interface {
	PublishTaskNotification(ctx context.Context, event TaskNotificationEvent) error
}

type service struct {
	repo      Repository
	publisher NotificationPublisher
	realtime  realtime.Broadcaster
}

func NewService(repo Repository, publishers ...NotificationPublisher) *service {
	var publisher NotificationPublisher
	if len(publishers) > 0 {
		publisher = publishers[0]
	}
	return &service{repo: repo, publisher: publisher}
}

func (s *service) SetRealtime(rt realtime.Broadcaster) {
	s.realtime = rt
}

func (s *service) broadcast(ctx context.Context, projectID, eventType, entityID, actorID string, entity any) {
	if s.realtime == nil {
		return
	}
	workspaceID, err := s.repo.GetWorkspaceIDByProjectID(ctx, projectID)
	if err != nil || workspaceID == "" {
		return
	}
	s.realtime.Broadcast(realtime.Event{
		Type:        eventType,
		WorkspaceID: workspaceID,
		EntityID:    entityID,
		ActorID:     actorID,
		ActorConnID: middleware.GetConnIDFromContext(ctx),
		Entity:      entity,
	})
}

func (s *service) CreateTask(ctx context.Context, projectAlias, userID string, req CreateTaskRequest) (*TaskResponse, error) {
	task, err := s.repo.CreateTask(ctx, projectAlias, userID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, task.ProjectID, "task.created", task.ID, userID, task)
	return task, nil
}

func (s *service) ListTasks(ctx context.Context, projectAlias, viewerID string, filter ListTasksParams, params pagination.Params) (TaskListResponse, error) {
	return s.repo.ListTasks(ctx, projectAlias, viewerID, filter, params)
}

func (s *service) AuthorizeTaskAccess(ctx context.Context, taskID, callerID string) error {
	ok, err := s.repo.CheckTaskMembership(ctx, taskID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: bạn không thuộc dự án của task này", ErrForbidden)
	}
	return nil
}

func (s *service) AuthorizeProjectAccess(ctx context.Context, projectID, callerID string) error {
	ok, err := s.repo.CheckProjectMembership(ctx, projectID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: bạn không thuộc dự án này", ErrForbidden)
	}
	return nil
}

func (s *service) GetTask(ctx context.Context, taskID string) (*TaskResponse, error) {
	return s.repo.GetTask(ctx, taskID)
}

func (s *service) UpdateTask(ctx context.Context, taskID, callerID string, req UpdateTaskRequest) (*TaskResponse, error) {
	task, err := s.repo.UpdateTask(ctx, taskID, callerID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, task.ProjectID, "task.updated", task.ID, callerID, task)
	if s.publisher != nil && req.statusPresent() {
		workspaceID, _ := s.repo.GetWorkspaceIDByProjectID(ctx, task.ProjectID)
		alias := s.callerAlias(ctx, callerID)
		for _, worker := range task.Workers {
			if worker.User.ID == "" || worker.User.ID == callerID {
				continue
			}
			if err := s.publisher.PublishTaskNotification(ctx, TaskNotificationEvent{
				Type: NotificationTypeTaskStatusUpdated, RecipientID: worker.User.ID, WorkspaceID: workspaceID, TaskID: task.ID,
				ProjectID: task.ProjectID, Title: task.Title, Status: task.Status, CallerAlias: alias,
			}); err != nil {
				return nil, err
			}
		}
	}
	if s.publisher != nil && req.faultHolderIDPresent() && task.FaultHolderID != "" && task.FaultHolderID != callerID {
		workspaceID, _ := s.repo.GetWorkspaceIDByProjectID(ctx, task.ProjectID)
		if err := s.publisher.PublishTaskNotification(ctx, TaskNotificationEvent{
			Type: NotificationTypeTaskFaultHolderAssigned, RecipientID: task.FaultHolderID, WorkspaceID: workspaceID, TaskID: task.ID,
			ProjectID: task.ProjectID, Title: task.Title, Status: task.Status, CallerAlias: s.callerAlias(ctx, callerID),
		}); err != nil {
			return nil, err
		}
	}
	return task, nil
}

func (s *service) ListTaskHistory(ctx context.Context, taskID string, params pagination.Params) (pagination.Response[TaskHistoryResponse], error) {
	return s.repo.ListTaskHistory(ctx, taskID, params)
}

func (s *service) AddAttachment(ctx context.Context, taskID, uploaderID string, req CreateTaskAttachmentRequest) (*TaskAttachmentResponse, error) {
	return s.repo.AddAttachment(ctx, taskID, uploaderID, req)
}

func (s *service) ListAttachments(ctx context.Context, taskID string) ([]TaskAttachmentResponse, error) {
	return s.repo.ListAttachments(ctx, taskID)
}

func (s *service) DeleteAttachment(ctx context.Context, attachmentID string) error {
	return s.repo.DeleteAttachment(ctx, attachmentID)
}

func (s *service) AddComment(ctx context.Context, taskID, authorID string, req CreateTaskCommentRequest) (*TaskCommentResponse, error) {
	comment, err := s.repo.CreateTaskComment(ctx, taskID, authorID, req.Content, req.Attachments)
	if err != nil {
		return nil, err
	}

	task, taskErr := s.repo.GetTask(ctx, taskID)
	if taskErr != nil || task == nil {
		return comment, nil
	}
	s.broadcast(ctx, task.ProjectID, "task.comment_added", comment.ID, authorID, comment)

	if s.publisher == nil {
		return comment, nil
	}
	workspaceID, err := s.repo.GetWorkspaceIDByProjectID(ctx, task.ProjectID)
	if err != nil || workspaceID == "" {
		return comment, nil
	}

	workerIDs, _ := s.repo.ListTaskWorkerUserIDs(ctx, taskID)
	commenterIDs, _ := s.repo.ListTaskCommentAuthorIDs(ctx, taskID)
	mentioned := make(map[string]bool, len(comment.MentionedUserIDs))
	for _, id := range comment.MentionedUserIDs {
		mentioned[id] = true
	}
	recipients := make(map[string]bool)
	for _, id := range workerIDs {
		recipients[id] = true
	}
	for _, id := range commenterIDs {
		recipients[id] = true
	}
	for id := range mentioned {
		recipients[id] = true
	}
	delete(recipients, authorID)

	alias := s.callerAlias(ctx, authorID)
	for recipientID := range recipients {
		_ = s.publisher.PublishTaskNotification(ctx, TaskNotificationEvent{
			Type: NotificationTypeTaskCommentAdded, RecipientID: recipientID, WorkspaceID: workspaceID,
			TaskID: task.ID, ProjectID: task.ProjectID, Title: task.Title, CallerAlias: alias,
			Comment: comment.Content, HasAttachment: len(comment.Attachments) > 0, Mentioned: mentioned[recipientID],
		})
	}
	return comment, nil
}

func (s *service) ListComments(ctx context.Context, taskID, callerID string) ([]TaskCommentResponse, error) {
	comments, err := s.repo.ListTaskComments(ctx, taskID)
	if err != nil {
		return nil, err
	}
	_ = s.repo.MarkTaskCommentsRead(ctx, taskID, callerID)
	return comments, nil
}

var ErrForbidden = errors.New("Bạn không có quyền thực hiện thao tác này")

func (s *service) UpdateComment(ctx context.Context, commentID, callerID string, req UpdateTaskCommentRequest) (*TaskCommentResponse, error) {
	existing, err := s.repo.GetTaskComment(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if existing.AuthorID != callerID {
		return nil, fmt.Errorf("%w: chỉ tác giả mới được sửa bình luận", ErrForbidden)
	}
	if req.Content == "" && len(existing.Attachments) == 0 {
		return nil, fmt.Errorf("Nội dung là bắt buộc")
	}
	comment, err := s.repo.UpdateTaskComment(ctx, commentID, req.Content)
	if err != nil {
		return nil, err
	}
	if task, err := s.repo.GetTask(ctx, comment.TaskID); err == nil && task != nil {
		s.broadcast(ctx, task.ProjectID, "task.comment_updated", comment.ID, callerID, comment)
	}
	return comment, nil
}

func (s *service) DeleteComment(ctx context.Context, commentID, callerID string) error {
	existing, err := s.repo.GetTaskComment(ctx, commentID)
	if err != nil {
		return err
	}
	if existing.AuthorID != callerID {
		return fmt.Errorf("%w: chỉ tác giả mới được xoá bình luận", ErrForbidden)
	}
	if err := s.repo.DeleteTaskComment(ctx, commentID); err != nil {
		return err
	}
	if task, err := s.repo.GetTask(ctx, existing.TaskID); err == nil && task != nil {
		s.broadcast(ctx, task.ProjectID, "task.comment_deleted", commentID, callerID, map[string]string{
			"id":     commentID,
			"taskId": existing.TaskID,
		})
	}
	return nil
}

func (s *service) DeleteTask(ctx context.Context, taskID string) error {
	task, taskErr := s.repo.GetTask(ctx, taskID)
	if err := s.repo.DeleteTask(ctx, taskID); err != nil {
		return err
	}
	if taskErr == nil && task != nil {
		s.broadcast(ctx, task.ProjectID, "task.deleted", taskID, "", nil)
	}
	return nil
}

func (s *service) CreateTaskLabel(ctx context.Context, projectAlias, userID string, req CreateTaskLabelRequest) (*TaskLabelResponse, error) {
	label, err := s.repo.CreateTaskLabel(ctx, projectAlias, userID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, label.ProjectID, "task_label.created", label.ID, userID, label)
	return label, nil
}

func (s *service) ListTaskLabels(ctx context.Context, projectAlias string, params pagination.Params) (pagination.Response[TaskLabelResponse], error) {
	return s.repo.ListTaskLabels(ctx, projectAlias, params)
}

func (s *service) ListTaskLabelsByProjectIDs(ctx context.Context, projectIDs []string, callerID string) ([]TaskLabelResponse, error) {
	allowed := make([]string, 0, len(projectIDs))
	for _, pid := range projectIDs {
		ok, err := s.repo.CheckProjectMembership(ctx, pid, callerID)
		if err != nil {
			return nil, err
		}
		if ok {
			allowed = append(allowed, pid)
		}
	}
	if len(allowed) == 0 {
		return []TaskLabelResponse{}, nil
	}
	return s.repo.ListTaskLabelsByProjectIDs(ctx, allowed)
}

func (s *service) AuthorizeLabelAccess(ctx context.Context, labelID, callerID string) error {
	ok, err := s.repo.CheckLabelMembership(ctx, labelID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: bạn không thuộc dự án của nhãn này", ErrForbidden)
	}
	return nil
}

func (s *service) UpdateTaskLabel(ctx context.Context, labelID string, req UpdateTaskLabelRequest) (*TaskLabelResponse, error) {
	label, err := s.repo.UpdateTaskLabel(ctx, labelID, req)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, label.ProjectID, "task_label.updated", label.ID, "", label)
	return label, nil
}

func (s *service) DeleteTaskLabel(ctx context.Context, labelID string) error {
	projectID, err := s.repo.DeleteTaskLabel(ctx, labelID)
	if err != nil {
		return err
	}
	s.broadcast(ctx, projectID, "task_label.deleted", labelID, "", nil)
	return nil
}

func (s *service) AttachTaskLabel(ctx context.Context, taskID, labelID string) error {
	if err := s.repo.AttachTaskLabel(ctx, taskID, labelID); err != nil {
		return err
	}
	if task, err := s.repo.GetTask(ctx, taskID); err == nil && task != nil {
		s.broadcast(ctx, task.ProjectID, "task.label.attached", taskID, "", task)
	}
	return nil
}

func (s *service) DetachTaskLabel(ctx context.Context, taskID, labelID string) error {
	if err := s.repo.DetachTaskLabel(ctx, taskID, labelID); err != nil {
		return err
	}
	if task, err := s.repo.GetTask(ctx, taskID); err == nil && task != nil {
		s.broadcast(ctx, task.ProjectID, "task.label.detached", taskID, "", task)
	}
	return nil
}

func (s *service) AssignTaskWorker(ctx context.Context, taskID, callerID, userID string, weight int) (*TaskWorkerResponse, error) {
	worker, err := s.repo.AssignTaskWorker(ctx, taskID, userID, weight)
	if err != nil {
		return nil, err
	}
	_ = s.repo.InsertEntityHistory(ctx, "task", taskID, callerID, "workers", nil, stringPtr(userID))
	if task, err := s.repo.GetTask(ctx, taskID); err == nil && task != nil {
		s.broadcast(ctx, task.ProjectID, "task.worker.assigned", taskID, callerID, task)
		if s.publisher != nil && worker.User.ID != "" && worker.User.ID != callerID {
			workspaceID, _ := s.repo.GetWorkspaceIDByProjectID(ctx, task.ProjectID)
			if err := s.publisher.PublishTaskNotification(ctx, TaskNotificationEvent{
				Type: NotificationTypeTaskAssigned, RecipientID: worker.User.ID, WorkspaceID: workspaceID, TaskID: worker.TaskID,
				ProjectID: task.ProjectID, Title: task.Title, CallerAlias: s.callerAlias(ctx, callerID),
			}); err != nil {
				return nil, err
			}
		}
	}
	return worker, nil
}

func (s *service) RemoveTaskWorker(ctx context.Context, taskID, callerID, userID string) error {
	if err := s.repo.RemoveTaskWorker(ctx, taskID, userID); err != nil {
		return err
	}
	_ = s.repo.InsertEntityHistory(ctx, "task", taskID, callerID, "workers", stringPtr(userID), nil)
	if task, err := s.repo.GetTask(ctx, taskID); err == nil && task != nil {
		s.broadcast(ctx, task.ProjectID, "task.worker.removed", taskID, callerID, task)
		if s.publisher != nil && userID != callerID {
			workspaceID, _ := s.repo.GetWorkspaceIDByProjectID(ctx, task.ProjectID)
			return s.publisher.PublishTaskNotification(ctx, TaskNotificationEvent{
				Type: NotificationTypeTaskUnassigned, RecipientID: userID, WorkspaceID: workspaceID, TaskID: taskID,
				ProjectID: task.ProjectID, Title: task.Title, CallerAlias: s.callerAlias(ctx, callerID),
			})
		}
	}
	return nil
}

func (s *service) AssignTaskFaultHolder(ctx context.Context, taskID, callerID, userID string, weight int) (*TaskFaultHolderResponse, error) {
	worker, err := s.repo.AssignTaskFaultHolder(ctx, taskID, userID, weight)
	if err != nil {
		return nil, err
	}
	_ = s.repo.InsertEntityHistory(ctx, "task", taskID, callerID, "fault_holders", nil, stringPtr(userID))
	if task, err := s.repo.GetTask(ctx, taskID); err == nil && task != nil {
		s.broadcast(ctx, task.ProjectID, "task.fault_holder.assigned", taskID, callerID, task)
		if s.publisher != nil && worker.User.ID != "" && worker.User.ID != callerID {
			workspaceID, _ := s.repo.GetWorkspaceIDByProjectID(ctx, task.ProjectID)
			if err := s.publisher.PublishTaskNotification(ctx, TaskNotificationEvent{
				Type: NotificationTypeTaskFaultHolderAssigned, RecipientID: worker.User.ID, WorkspaceID: workspaceID, TaskID: worker.TaskID,
				ProjectID: task.ProjectID, Title: task.Title, CallerAlias: s.callerAlias(ctx, callerID),
			}); err != nil {
				return nil, err
			}
		}
	}
	return worker, nil
}

func (s *service) ListMyTasks(ctx context.Context, workspaceID, userID string, filter ListMyTasksParams, params pagination.Params) (TaskListResponse, error) {
	return s.repo.ListMyTasks(ctx, workspaceID, userID, filter, params)
}

func (s *service) CountMyActiveTasks(ctx context.Context, workspaceID, userID string) (int64, error) {
	return s.repo.CountMyActiveTasks(ctx, workspaceID, userID)
}

func (s *service) ListThreads(ctx context.Context, workspaceID, userID string, params pagination.Params) (pagination.Response[ThreadResponse], error) {
	return s.repo.ListThreads(ctx, workspaceID, userID, params)
}

func (s *service) CountThreads(ctx context.Context, workspaceID, userID string) (int64, error) {
	return s.repo.CountThreads(ctx, workspaceID, userID)
}

func (s *service) callerAlias(ctx context.Context, callerID string) string {
	alias, _ := s.repo.GetUserAliasByID(ctx, callerID)
	return alias
}

func (s *service) RemoveTaskFaultHolder(ctx context.Context, taskID, callerID, userID string) error {
	if err := s.repo.RemoveTaskFaultHolder(ctx, taskID, userID); err != nil {
		return err
	}
	_ = s.repo.InsertEntityHistory(ctx, "task", taskID, callerID, "fault_holders", stringPtr(userID), nil)
	if task, err := s.repo.GetTask(ctx, taskID); err == nil && task != nil {
		s.broadcast(ctx, task.ProjectID, "task.fault_holder.removed", taskID, callerID, task)
		if s.publisher != nil && userID != callerID {
			workspaceID, _ := s.repo.GetWorkspaceIDByProjectID(ctx, task.ProjectID)
			return s.publisher.PublishTaskNotification(ctx, TaskNotificationEvent{
				Type: NotificationTypeTaskFaultHolderUnassigned, RecipientID: userID, WorkspaceID: workspaceID, TaskID: taskID,
				ProjectID: task.ProjectID, Title: task.Title, CallerAlias: s.callerAlias(ctx, callerID),
			})
		}
	}
	return nil
}
