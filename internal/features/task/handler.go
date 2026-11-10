package task

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/pagination"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) authorizeTask(w http.ResponseWriter, r *http.Request, taskID string) (string, bool) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return "", false
	}
	if err := h.svc.AuthorizeTaskAccess(r.Context(), taskID, userID); err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, err.Error())
		} else {
			httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không kiểm tra được quyền: %s", err.Error()))
		}
		return "", false
	}
	return userID, true
}

func (h *Handler) authorizeProject(w http.ResponseWriter, r *http.Request, projectID string) (string, bool) {
	return h.authorizeWith(w, r, func(userID string) error {
		return h.svc.AuthorizeProjectAccess(r.Context(), projectID, userID)
	})
}

func (h *Handler) authorizeLabel(w http.ResponseWriter, r *http.Request, labelID string) (string, bool) {
	return h.authorizeWith(w, r, func(userID string) error {
		return h.svc.AuthorizeLabelAccess(r.Context(), labelID, userID)
	})
}

func (h *Handler) authorizeWith(w http.ResponseWriter, r *http.Request, check func(userID string) error) (string, bool) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return "", false
	}
	if err := check(userID); err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, err.Error())
		} else {
			httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không kiểm tra được quyền: %s", err.Error()))
		}
		return "", false
	}
	return userID, true
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/my-tasks", authMid.Handle(h.ListMyTasks))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/my-tasks/count", authMid.Handle(h.CountMyActiveTasks))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/threads", authMid.Handle(h.ListThreads))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/threads/count", authMid.Handle(h.CountThreads))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/tasks", authMid.Handle(h.CreateTask))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/tasks", authMid.Handle(h.ListTasks))
	mux.HandleFunc("POST /v1/projects/{projectId}/tasks", authMid.Handle(h.CreateTask))
	mux.HandleFunc("GET /v1/projects/{projectId}/tasks", authMid.Handle(h.ListTasks))
	mux.HandleFunc("GET /v1/tasks/{taskId}", authMid.Handle(h.GetTask))
	mux.HandleFunc("PATCH /v1/tasks/{taskId}", authMid.Handle(h.UpdateTask))
	mux.HandleFunc("GET /v1/tasks/{taskId}/history", authMid.Handle(h.ListTaskHistory))
	mux.HandleFunc("DELETE /v1/tasks/{taskId}", authMid.Handle(h.DeleteTask))

	mux.HandleFunc("POST /v1/tasks/{taskId}/attachments", authMid.Handle(h.AddAttachment))
	mux.HandleFunc("GET /v1/tasks/{taskId}/attachments", authMid.Handle(h.ListAttachments))
	mux.HandleFunc("DELETE /v1/tasks/{taskId}/attachments/{attachmentId}", authMid.Handle(h.DeleteAttachment))

	mux.HandleFunc("POST /v1/tasks/{taskId}/comments", authMid.Handle(h.AddComment))
	mux.HandleFunc("GET /v1/tasks/{taskId}/comments", authMid.Handle(h.ListComments))
	mux.HandleFunc("PATCH /v1/tasks/{taskId}/comments/{commentId}", authMid.Handle(h.UpdateComment))
	mux.HandleFunc("DELETE /v1/tasks/{taskId}/comments/{commentId}", authMid.Handle(h.DeleteComment))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/labels", authMid.Handle(h.CreateTaskLabel))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/labels", authMid.Handle(h.ListTaskLabels))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/labels", authMid.Handle(h.ListTaskLabelsByProjectIDs))
	mux.HandleFunc("POST /v1/projects/{projectId}/labels", authMid.Handle(h.CreateTaskLabel))
	mux.HandleFunc("GET /v1/projects/{projectId}/labels", authMid.Handle(h.ListTaskLabels))
	mux.HandleFunc("PATCH /v1/labels/{labelId}", authMid.Handle(h.UpdateTaskLabel))
	mux.HandleFunc("DELETE /v1/labels/{labelId}", authMid.Handle(h.DeleteTaskLabel))
	mux.HandleFunc("PUT /v1/tasks/{taskId}/labels/{labelId}", authMid.Handle(h.AttachTaskLabel))
	mux.HandleFunc("DELETE /v1/tasks/{taskId}/labels/{labelId}", authMid.Handle(h.DetachTaskLabel))

	mux.HandleFunc("PUT /v1/tasks/{taskId}/workers/{userId}", authMid.Handle(h.AssignTaskWorker))
	mux.HandleFunc("DELETE /v1/tasks/{taskId}/workers/{userId}", authMid.Handle(h.RemoveTaskWorker))

	mux.HandleFunc("PUT /v1/tasks/{taskId}/fault-holders/{userId}", authMid.Handle(h.AssignTaskFaultHolder))
	mux.HandleFunc("DELETE /v1/tasks/{taskId}/fault-holders/{userId}", authMid.Handle(h.RemoveTaskFaultHolder))
}

// @Summary Create Task
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request body CreateTaskRequest true "Task"
// @Success 201 {object} TaskResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/tasks [post]
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorizeProject(w, r, r.PathValue("projectId"))
	if !ok {
		return
	}
	var req CreateTaskRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeCreateTaskRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.CreateTask(r.Context(), r.PathValue("projectId"), userID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List My Tasks
// @Description List tasks assigned to the authenticated user within a workspace
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param status query string false "Task status"
// @Param archived query boolean false "Archived"
// @Param priority query integer false "Priority"
// @Param projectId query string false "Narrow to a single project"
// @Param startDateFrom query string false "Start date lower bound (RFC3339)"
// @Param startDateTo query string false "Start date upper bound (RFC3339)"
// @Param endDateFrom query string false "End date lower bound (RFC3339)"
// @Param endDateTo query string false "End date upper bound (RFC3339)"
// @Param q query string false "Search title"
// @Param sort query string false "Comma-separated sort fields, each optionally suffixed with :asc or :desc (default asc), e.g. 'status:asc,endDate:desc'. Defaults to status asc, endDate asc when omitted. Valid fields: title, project, status, priority, estimation, module, returnCount, startDate, endDate"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} TaskListResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/my-tasks [get]
func (h *Handler) ListMyTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var filter ListMyTasksParams
	if err := httpx.DecodeQuery(r, &filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := normalizeListMyTasksParams(&filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListMyTasks(r.Context(), r.PathValue("workspaceId"), userID, filter, params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách task của tôi: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Count My Active Tasks
// @Description Count non-archived tasks assigned to the authenticated user with todo or in-progress status
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} MyTasksCountResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/my-tasks/count [get]
func (h *Handler) CountMyActiveTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	count, err := h.svc.CountMyActiveTasks(
		r.Context(),
		r.PathValue("workspaceId"),
		userID,
	)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể đếm task đang làm của tôi: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, MyTasksCountResponse{Count: count})
}

// @Summary List Threads
// @Description Tasks with unread comment activity for the authenticated user — assigned to them, or they were @mentioned — ordered by most recently active first. Powers the "Threads" nav destination.
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[ThreadResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/threads [get]
func (h *Handler) ListThreads(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListThreads(r.Context(), r.PathValue("workspaceId"), userID, params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách thảo luận: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Count Threads
// @Description Nav-bar badge figure — how many tasks currently qualify for ListThreads.
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} ThreadsCountResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/threads/count [get]
func (h *Handler) CountThreads(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	count, err := h.svc.CountThreads(r.Context(), r.PathValue("workspaceId"), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể đếm thảo luận: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, ThreadsCountResponse{Count: count})
}

// @Summary List Tasks
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param moduleId query string false "Module ID"
// @Param status query string false "Task status"
// @Param archived query boolean false "Archived"
// @Param parentId query string false "Parent task ID"
// @Param createdById query string false "Creator user ID"
// @Param priority query integer false "Priority"
// @Param estimation query integer false "Estimation"
// @Param startDateFrom query string false "Start date from" format(date-time)
// @Param startDateTo query string false "Start date to" format(date-time)
// @Param endDateFrom query string false "End date from" format(date-time)
// @Param endDateTo query string false "End date to" format(date-time)
// @Param labelId query string false "Label ID"
// @Param workerId query string false "Worker ID"
// @Param q query string false "Search title or description"
// @Param sort query string false "Comma-separated sort fields, each optionally suffixed with :asc or :desc (default asc), e.g. 'status:asc,endDate:desc'. Defaults to status asc, endDate asc when omitted. Valid fields: title, project, status, priority, estimation, module, returnCount, startDate, endDate"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} TaskListResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/tasks [get]
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeProject(w, r, r.PathValue("projectId"))
	if !ok {
		return
	}
	var filter ListTasksParams
	if err := httpx.DecodeQuery(r, &filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := normalizeListTasksParams(&filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListTasks(r.Context(), r.PathValue("projectId"), callerID, filter, params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get Task
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Success 200 {object} TaskResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/tasks/{taskId} [get]
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeTask(w, r, r.PathValue("taskId")); !ok {
		return
	}
	resp, err := h.svc.GetTask(r.Context(), r.PathValue("taskId"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, fmt.Sprintf("Không tìm thấy task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update Task
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param request body UpdateTaskRequest true "Update fields"
// @Success 200 {object} TaskResponse
// @Router /v1/tasks/{taskId} [patch]
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	var req UpdateTaskRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeUpdateTaskRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.UpdateTask(r.Context(), r.PathValue("taskId"), callerID, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List Task Update History
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[TaskHistoryResponse]
// @Router /v1/tasks/{taskId}/history [get]
func (h *Handler) ListTaskHistory(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeTask(w, r, r.PathValue("taskId")); !ok {
		return
	}
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListTaskHistory(r.Context(), r.PathValue("taskId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách lịch sử task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Add Task Attachment
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param request body CreateTaskAttachmentRequest true "File reference (upload it first via POST /v1/uploads/presigned/url)"
// @Success 201 {object} TaskAttachmentResponse
// @Router /v1/tasks/{taskId}/attachments [post]
func (h *Handler) AddAttachment(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	var req CreateTaskAttachmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.FileName = strings.TrimSpace(req.FileName)
	req.FileURL = strings.TrimSpace(req.FileURL)
	if req.FileName == "" || req.FileURL == "" {
		httpx.Error(w, http.StatusBadRequest, "fileName và fileUrl là bắt buộc")
		return
	}
	resp, err := h.svc.AddAttachment(r.Context(), r.PathValue("taskId"), callerID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể thêm tệp đính kèm: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Task Attachments
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Success 200 {array} TaskAttachmentResponse
// @Router /v1/tasks/{taskId}/attachments [get]
func (h *Handler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeTask(w, r, r.PathValue("taskId")); !ok {
		return
	}
	resp, err := h.svc.ListAttachments(r.Context(), r.PathValue("taskId"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách tệp đính kèm: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Task Attachment
// @Tags Tasks
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param attachmentId path string true "Attachment ID"
// @Success 204
// @Router /v1/tasks/{taskId}/attachments/{attachmentId} [delete]
func (h *Handler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeTask(w, r, r.PathValue("taskId")); !ok {
		return
	}
	if err := h.svc.DeleteAttachment(r.Context(), r.PathValue("attachmentId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá tệp đính kèm: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Add Task Comment
// @Description Posts a message to the task's comment thread. Content may embed @[Display Name](userId) mention tokens (inserted by the FE's autocomplete) — mentioned users, task assignees, and prior commenters are all notified.
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param request body CreateTaskCommentRequest true "Comment"
// @Success 201 {object} TaskCommentResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/tasks/{taskId}/comments [post]
func (h *Handler) AddComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	var req CreateTaskCommentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeCreateTaskCommentRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.AddComment(r.Context(), r.PathValue("taskId"), userID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể thêm bình luận: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Task Comments
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Success 200 {array} TaskCommentResponse
// @Router /v1/tasks/{taskId}/comments [get]
func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	resp, err := h.svc.ListComments(r.Context(), r.PathValue("taskId"), callerID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách bình luận: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Edit Task Comment (author only)
// @Description Edits a comment's text — attachments are fixed at creation time. Only the comment's own author may edit it.
// @Tags Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param commentId path string true "Comment ID"
// @Param request body UpdateTaskCommentRequest true "New content"
// @Success 200 {object} TaskCommentResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/tasks/{taskId}/comments/{commentId} [patch]
func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	var req UpdateTaskCommentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeUpdateTaskCommentRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.UpdateComment(r.Context(), r.PathValue("commentId"), callerID, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, err.Error())
			return
		}
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Không thể cập nhật bình luận: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Task Comment (author only)
// @Tags Tasks
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param commentId path string true "Comment ID"
// @Success 204
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/tasks/{taskId}/comments/{commentId} [delete]
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	if err := h.svc.DeleteComment(r.Context(), r.PathValue("commentId"), callerID); err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá bình luận: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Delete Task
// @Tags Tasks
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Success 204
// @Router /v1/tasks/{taskId} [delete]
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeTask(w, r, r.PathValue("taskId")); !ok {
		return
	}
	if err := h.svc.DeleteTask(r.Context(), r.PathValue("taskId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá task: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Create Task Label
// @Tags Labels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request body CreateTaskLabelRequest true "Label"
// @Success 201 {object} TaskLabelResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/labels [post]
func (h *Handler) CreateTaskLabel(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.authorizeProject(w, r, r.PathValue("projectId"))
	if !ok {
		return
	}
	var req CreateTaskLabelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "Tên là bắt buộc")
		return
	}
	req.Color = strings.TrimSpace(req.Color)
	if req.Color == "" {
		req.Color = "#000000"
	}
	resp, err := h.svc.CreateTaskLabel(r.Context(), r.PathValue("projectId"), userID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo nhãn task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Task Labels
// @Tags Labels
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[TaskLabelResponse]
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/labels [get]
func (h *Handler) ListTaskLabels(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeProject(w, r, r.PathValue("projectId")); !ok {
		return
	}
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListTaskLabels(r.Context(), r.PathValue("projectId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách nhãn task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List Task Labels Across Projects
// @Description Every task label across the given projects in one call — for a caller that would otherwise call List Task Labels once per project (e.g. the My Tasks screen).
// @Tags Labels
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectIds query string true "Comma-separated project IDs"
// @Success 200 {object} TaskLabelsByProjectIDsResponse
// @Router /v1/workspaces/{workspaceId}/labels [get]
func (h *Handler) ListTaskLabelsByProjectIDs(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	projectIDs := httpx.ParseCSVParam(r, "projectIds")
	items, err := h.svc.ListTaskLabelsByProjectIDs(r.Context(), projectIDs, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách nhãn task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, TaskLabelsByProjectIDsResponse{Items: items})
}

// @Summary Update Task Label
// @Tags Labels
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param labelId path string true "Label ID"
// @Param request body UpdateTaskLabelRequest true "Update"
// @Success 200 {object} TaskLabelResponse
// @Router /v1/labels/{labelId} [patch]
func (h *Handler) UpdateTaskLabel(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeLabel(w, r, r.PathValue("labelId")); !ok {
		return
	}
	var req UpdateTaskLabelRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "Tên là bắt buộc")
		return
	}
	req.Color = strings.TrimSpace(req.Color)
	if req.Color == "" {
		req.Color = "#000000"
	}
	resp, err := h.svc.UpdateTaskLabel(r.Context(), r.PathValue("labelId"), req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật nhãn task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Task Label
// @Tags Labels
// @Security BearerAuth
// @Param labelId path string true "Label ID"
// @Success 204
// @Router /v1/labels/{labelId} [delete]
func (h *Handler) DeleteTaskLabel(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeLabel(w, r, r.PathValue("labelId")); !ok {
		return
	}
	if err := h.svc.DeleteTaskLabel(r.Context(), r.PathValue("labelId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá nhãn task: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Attach Label to Task
// @Tags Tasks
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param labelId path string true "Label ID"
// @Success 204
// @Router /v1/tasks/{taskId}/labels/{labelId} [put]
func (h *Handler) AttachTaskLabel(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeTask(w, r, r.PathValue("taskId")); !ok {
		return
	}
	if err := h.svc.AttachTaskLabel(r.Context(), r.PathValue("taskId"), r.PathValue("labelId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể gắn nhãn task: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Detach Label from Task
// @Tags Tasks
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param labelId path string true "Label ID"
// @Success 204
// @Router /v1/tasks/{taskId}/labels/{labelId} [delete]
func (h *Handler) DetachTaskLabel(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeTask(w, r, r.PathValue("taskId")); !ok {
		return
	}
	if err := h.svc.DetachTaskLabel(r.Context(), r.PathValue("taskId"), r.PathValue("labelId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể gỡ nhãn task: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Assign Worker to Task
// @Tags Task Workers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param userId path string true "User ID"
// @Param request body AssignTaskWorkerRequest true "Worker weight"
// @Success 200 {object} TaskWorkerResponse
// @Router /v1/tasks/{taskId}/workers/{userId} [put]
func (h *Handler) AssignTaskWorker(w http.ResponseWriter, r *http.Request) {
	var req AssignTaskWorkerRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if req.Weight <= 0 {
		req.Weight = 1
	}
	callerID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	resp, err := h.svc.AssignTaskWorker(r.Context(), r.PathValue("taskId"), callerID, r.PathValue("userId"), req.Weight)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể giao người làm của task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Remove Worker from Task
// @Tags Task Workers
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param userId path string true "User ID"
// @Success 204
// @Router /v1/tasks/{taskId}/workers/{userId} [delete]
func (h *Handler) RemoveTaskWorker(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	if err := h.svc.RemoveTaskWorker(r.Context(), r.PathValue("taskId"), callerID, r.PathValue("userId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể gỡ người làm của task: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Assign Fault Holder to Task
// @Tags Task Fault Holders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param userId path string true "User ID"
// @Param request body AssignTaskFaultHolderRequest true "Worker weight"
// @Success 200 {object} TaskFaultHolderResponse
// @Router /v1/tasks/{taskId}/fault-holders/{userId} [put]
func (h *Handler) AssignTaskFaultHolder(w http.ResponseWriter, r *http.Request) {
	var req AssignTaskFaultHolderRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if req.Weight <= 0 {
		req.Weight = 1
	}
	callerID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	resp, err := h.svc.AssignTaskFaultHolder(r.Context(), r.PathValue("taskId"), callerID, r.PathValue("userId"), req.Weight)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể giao người chịu lỗi của task: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Remove Fault Holder from Task
// @Tags Task Fault Holders
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Param userId path string true "User ID"
// @Success 204
// @Router /v1/tasks/{taskId}/fault-holders/{userId} [delete]
func (h *Handler) RemoveTaskFaultHolder(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeTask(w, r, r.PathValue("taskId"))
	if !ok {
		return
	}
	if err := h.svc.RemoveTaskFaultHolder(r.Context(), r.PathValue("taskId"), callerID, r.PathValue("userId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể gỡ người chịu lỗi của task: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
