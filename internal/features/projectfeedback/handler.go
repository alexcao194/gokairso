package projectfeedback

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/pagination"
)

func parseListFeedbacksFilter(r *http.Request) (ListFeedbacksFilter, error) {
	var filter ListFeedbacksFilter
	query := r.URL.Query()
	if raw := query.Get("priority"); raw != "" {
		priorities := make([]int32, 0, len(strings.Split(raw, ",")))
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			v, err := strconv.Atoi(part)
			if err != nil {
				return filter, fmt.Errorf("priority phải là danh sách số nguyên cách nhau bằng dấu phẩy")
			}
			priorities = append(priorities, int32(v))
		}
		filter.Priorities = priorities
	}
	if raw := query.Get("hasResponse"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			filter.HasResponse = &v
		}
	}
	if raw := query.Get("status"); raw != "" {
		statuses := make([]FeedbackStatus, 0, len(strings.Split(raw, ",")))
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			status := FeedbackStatus(part)
			if !status.Valid() {
				return filter, fmt.Errorf("Trạng thái phải là một trong: pending, discussing, acknowledged, in_progress, resolved, deployed, canceled")
			}
			statuses = append(statuses, status)
		}
		filter.Statuses = statuses
	}
	if raw := query.Get("feedbackType"); raw != "" {
		types := make([]FeedbackType, 0, len(strings.Split(raw, ",")))
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			ft := FeedbackType(part)
			if !ft.Valid() {
				return filter, fmt.Errorf("feedbackType phải là feature, improvement hoặc bug")
			}
			types = append(types, ft)
		}
		filter.FeedbackTypes = types
	}
	filter.Query = query.Get("q")

	sortRules, err := parseFeedbackSortParam(query.Get("sort"))
	if err != nil {
		return filter, err
	}
	filter.SortRules = sortRules
	return filter, nil
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings", authMid.Handle(h.GetSettings))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings/enable", authMid.Handle(h.EnableSettings))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings/disable", authMid.Handle(h.DisableSettings))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings/show-timeline", authMid.Handle(h.UpdateShowTimeline))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings/regenerate", authMid.Handle(h.RegenerateToken))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/project-feedbacks", authMid.Handle(h.ListFeedbacks))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/project-feedbacks", authMid.Handle(h.CreateFeedback))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-summary", authMid.Handle(h.GetSummary))
	mux.HandleFunc("GET /v1/tasks/{taskId}/feedbacks", authMid.Handle(h.ListFeedbacksForTask))
	mux.HandleFunc("PATCH /v1/project-feedbacks/{feedbackId}", authMid.Handle(h.UpdateFeedback))
	mux.HandleFunc("PATCH /v1/project-feedbacks/{feedbackId}/status", authMid.Handle(h.UpdateFeedbackStatus))
	mux.HandleFunc("PATCH /v1/project-feedbacks/{feedbackId}/response", authMid.Handle(h.RespondToFeedback))
	mux.HandleFunc("PUT /v1/project-feedbacks/{feedbackId}/tasks/{taskId}", authMid.Handle(h.LinkTask))
	mux.HandleFunc("DELETE /v1/project-feedbacks/{feedbackId}/tasks/{taskId}", authMid.Handle(h.UnlinkTask))
}

func handleServiceError(w http.ResponseWriter, err error, action string) {
	if errors.Is(err, ErrForbidden) {
		httpx.Error(w, http.StatusForbidden, "Bạn không có quyền thực hiện thao tác này")
		return
	}
	httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể %s: %s", action, err.Error()))
}

// @Summary Get Project Feedback Settings
// @Tags ProjectFeedback
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Success 200 {object} FeedbackSettingsResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings [get]
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.svc.GetFeedbackSettings(r.Context(), r.PathValue("projectId"), callerID)
	if err != nil {
		handleServiceError(w, err, "lấy cấu hình trang feedback")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get Project Feedback Summary
// @Description Lightweight count of feedback still in the "pending" triage status — used to badge the "Feedback" feature entry.
// @Tags ProjectFeedback
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Success 200 {object} FeedbackSummaryResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-summary [get]
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.svc.GetFeedbackSummary(r.Context(), r.PathValue("projectId"), callerID)
	if err != nil {
		handleServiceError(w, err, "lấy tóm tắt feedback")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Enable Project Public Feedback
// @Tags ProjectFeedback
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Success 200 {object} FeedbackSettingsResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings/enable [post]
func (h *Handler) EnableSettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.svc.EnableFeedbackSettings(r.Context(), r.PathValue("projectId"), callerID)
	if err != nil {
		handleServiceError(w, err, "bật trang feedback công khai")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Disable Project Public Feedback
// @Tags ProjectFeedback
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Success 200 {object} FeedbackSettingsResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings/disable [post]
func (h *Handler) DisableSettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.svc.DisableFeedbackSettings(r.Context(), r.PathValue("projectId"), callerID)
	if err != nil {
		handleServiceError(w, err, "tắt trang feedback công khai")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Regenerate Project Public Feedback Link
// @Tags ProjectFeedback
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Success 200 {object} FeedbackSettingsResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings/regenerate [post]
func (h *Handler) RegenerateToken(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.svc.RegenerateFeedbackToken(r.Context(), r.PathValue("projectId"), callerID)
	if err != nil {
		handleServiceError(w, err, "tạo lại liên kết feedback")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update Show Module Timeline Setting
// @Tags ProjectFeedback
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request body UpdateFeedbackSettingsRequest true "Settings fields"
// @Success 200 {object} FeedbackSettingsResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/feedback-settings/show-timeline [patch]
func (h *Handler) UpdateShowTimeline(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateFeedbackSettingsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if req.ShowModuleTimeline == nil {
		httpx.Error(w, http.StatusBadRequest, "showModuleTimeline là bắt buộc")
		return
	}
	resp, err := h.svc.UpdateShowTimeline(r.Context(), r.PathValue("projectId"), callerID, *req.ShowModuleTimeline)
	if err != nil {
		handleServiceError(w, err, "cập nhật cài đặt hiển thị timeline")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List Project Feedback (member view)
// @Tags ProjectFeedback
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param priority query string false "Comma-separated list of priorities to filter by (1-4), e.g. '2,3'"
// @Param hasResponse query bool false "Filter by whether a Response has been set"
// @Param status query string false "Comma-separated list of triage statuses to filter by: pending, acknowledged, resolved, canceled"
// @Param feedbackType query string false "Comma-separated list of types to filter by: feature, improvement, bug"
// @Param q query string false "Search submitter name/company, feature name, content, response, and code"
// @Param sort query string false "Comma-separated sort fields, each optionally suffixed with :asc or :desc (default asc), e.g. 'priority:desc,submitted:asc'. Defaults to submitted desc when omitted. Valid fields: code, priority, feature, submitter, company, status, expectedFix, submitted"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} ProjectFeedbackListResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/project-feedbacks [get]
func (h *Handler) ListFeedbacks(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	params := pagination.Parse(r, 20, 100)
	filter, err := parseListFeedbacksFilter(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.ListFeedbacks(r.Context(), r.PathValue("projectId"), callerID, filter, params)
	if err != nil {
		handleServiceError(w, err, "lấy danh sách feedback")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create Project Feedback (member view)
// @Description Lets any project member log a feedback item directly — e.g. one relayed by phone/email from a customer — same shape as the public submission, just authenticated.
// @Tags ProjectFeedback
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request body CreateProjectFeedbackRequest true "Feedback"
// @Success 201 {object} ProjectFeedbackResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/project-feedbacks [post]
func (h *Handler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateProjectFeedbackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeCreateFeedbackRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.CreateFeedback(r.Context(), r.PathValue("projectId"), callerID, req)
	if err != nil {
		handleServiceError(w, err, "tạo feedback")
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Feedback Linked To A Task
// @Description Reverse lookup — the public feedback item(s) that link to this task, so the project's task list can offer a "view feedback" preview.
// @Tags ProjectFeedback
// @Produce json
// @Security BearerAuth
// @Param taskId path string true "Task ID"
// @Success 200 {array} ProjectFeedbackResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/tasks/{taskId}/feedbacks [get]
func (h *Handler) ListFeedbacksForTask(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.svc.ListFeedbacksForTask(r.Context(), r.PathValue("taskId"), callerID)
	if err != nil {
		handleServiceError(w, err, "lấy feedback của task")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Edit A Feedback's Own Fields
// @Description Free-form edit (submitter/feature/content/priority) — any project member, not just admin/holder.
// @Tags ProjectFeedback
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param feedbackId path string true "Feedback ID"
// @Param request body UpdateProjectFeedbackRequest true "Feedback fields"
// @Success 200 {object} ProjectFeedbackResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/project-feedbacks/{feedbackId} [patch]
func (h *Handler) UpdateFeedback(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateProjectFeedbackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeUpdateFeedbackRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.UpdateFeedback(r.Context(), r.PathValue("feedbackId"), callerID, req)
	if err != nil {
		handleServiceError(w, err, "cập nhật feedback")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Set A Feedback's Triage Status
// @Tags ProjectFeedback
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param feedbackId path string true "Feedback ID"
// @Param request body UpdateFeedbackStatusRequest true "Status"
// @Success 200 {object} ProjectFeedbackResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/project-feedbacks/{feedbackId}/status [patch]
func (h *Handler) UpdateFeedbackStatus(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateFeedbackStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if !req.Status.Valid() {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Trạng thái %q không hợp lệ", req.Status))
		return
	}
	resp, err := h.svc.UpdateFeedbackStatus(r.Context(), r.PathValue("feedbackId"), callerID, req.Status)
	if err != nil {
		handleServiceError(w, err, "cập nhật trạng thái feedback")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Set The Response Shown To The Public For A Feedback
// @Tags ProjectFeedback
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param feedbackId path string true "Feedback ID"
// @Param request body UpdateFeedbackResponseRequest true "Response"
// @Success 200 {object} ProjectFeedbackResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/project-feedbacks/{feedbackId}/response [patch]
func (h *Handler) RespondToFeedback(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateFeedbackResponseRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.svc.RespondToFeedback(r.Context(), r.PathValue("feedbackId"), callerID, req.Response, req.ExpectedFixDate)
	if err != nil {
		handleServiceError(w, err, "phản hồi feedback")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Link An Existing Task To A Feedback
// @Tags ProjectFeedback
// @Security BearerAuth
// @Param feedbackId path string true "Feedback ID"
// @Param taskId path string true "Task ID"
// @Success 204
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/project-feedbacks/{feedbackId}/tasks/{taskId} [put]
func (h *Handler) LinkTask(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.svc.LinkTask(r.Context(), r.PathValue("feedbackId"), r.PathValue("taskId"), callerID); err != nil {
		handleServiceError(w, err, "liên kết task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Unlink A Task From A Feedback
// @Tags ProjectFeedback
// @Security BearerAuth
// @Param feedbackId path string true "Feedback ID"
// @Param taskId path string true "Task ID"
// @Success 204
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/project-feedbacks/{feedbackId}/tasks/{taskId} [delete]
func (h *Handler) UnlinkTask(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.svc.UnlinkTask(r.Context(), r.PathValue("feedbackId"), r.PathValue("taskId"), callerID); err != nil {
		handleServiceError(w, err, "bỏ liên kết task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
