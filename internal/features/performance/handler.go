package performance

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/pagination"
)

type Handler struct {
	initiatives InitiativeService
	lateLogs    LateLogService
	feedbacks   FeedbackService
	kpi         KPIService
}

func NewHandler(initiatives InitiativeService, lateLogs LateLogService, feedbacks FeedbackService, kpi KPIService) *Handler {
	return &Handler{initiatives: initiatives, lateLogs: lateLogs, feedbacks: feedbacks, kpi: kpi}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/initiatives", authMid.Handle(h.CreateInitiative))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/initiatives", authMid.Handle(h.ListInitiatives))
	mux.HandleFunc("PATCH /v1/initiatives/{initiativeId}", authMid.Handle(h.UpdateInitiative))
	mux.HandleFunc("DELETE /v1/initiatives/{initiativeId}", authMid.Handle(h.DeleteInitiative))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/late/logs", authMid.Handle(h.CreateLateLog))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/late/logs", authMid.Handle(h.ListLateLogs))
	mux.HandleFunc("PATCH /v1/late/logs/{lateLogId}", authMid.Handle(h.UpdateLateLog))
	mux.HandleFunc("DELETE /v1/late/logs/{lateLogId}", authMid.Handle(h.DeleteLateLog))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/feedbacks", authMid.Handle(h.CreateFeedback))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/feedbacks", authMid.Handle(h.ListFeedbacks))
	mux.HandleFunc("PATCH /v1/feedbacks/{feedbackId}", authMid.Handle(h.UpdateFeedback))
	mux.HandleFunc("DELETE /v1/feedbacks/{feedbackId}", authMid.Handle(h.DeleteFeedback))

	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/kpi", authMid.Handle(h.GetKPI))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/kpi/export", authMid.Handle(h.ExportKPI))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/kpi/me", authMid.Handle(h.GetMyKPI))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/members/{memberId}/breakdown", authMid.Handle(h.GetMemberTaskBreakdown))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/kpi/attitude", authMid.Handle(h.SetAttitude))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/kpi/settings", authMid.Handle(h.GetKPISettings))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/kpi/settings", authMid.Handle(h.UpsertKPISettings))
}

func writeServiceError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("%s: %s", fallback, err.Error()))
	}
}

// @Summary Create initiative
// @Tags Performance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateInitiativeRequest true "Initiative"
// @Success 201 {object} InitiativeResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/initiatives [post]
func (h *Handler) CreateInitiative(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateInitiativeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.initiatives.CreateInitiative(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể tạo sáng kiến")
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List initiatives
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param status query string false "Filter by status (proposed, in_progress, accepted, rejected). Omit to return all statuses."
// @Param proposerId query string false "Filter by proposer user ID"
// @Param refMemberId query string false "Filter by reference member user ID"
// @Param from query string false "Filter from time (RFC3339 or YYYY-MM-DD)"
// @Param to query string false "Filter to time (RFC3339 or YYYY-MM-DD)"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[InitiativeResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/initiatives [get]
func (h *Handler) ListInitiatives(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	params := pagination.Parse(r, 20, 100)
	var filter ListInitiativesFilter
	if err := httpx.DecodeQuery(r, &filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Tham số truy vấn không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.initiatives.ListInitiatives(r.Context(), r.PathValue("workspaceId"), callerID, params, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách sáng kiến")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update initiative
// @Tags Performance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param initiativeId path string true "Initiative ID"
// @Param request body UpdateInitiativeRequest true "Update"
// @Success 200 {object} InitiativeResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/initiatives/{initiativeId} [patch]
func (h *Handler) UpdateInitiative(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateInitiativeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.initiatives.UpdateInitiative(r.Context(), r.PathValue("initiativeId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể cập nhật sáng kiến")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete initiative
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param initiativeId path string true "Initiative ID"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/initiatives/{initiativeId} [delete]
func (h *Handler) DeleteInitiative(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.initiatives.DeleteInitiative(r.Context(), r.PathValue("initiativeId"), callerID); err != nil {
		writeServiceError(w, err, "Không thể xoá sáng kiến")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Create late log
// @Tags Performance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateLateLogRequest true "Late log"
// @Success 201 {object} LateLogResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/late/logs [post]
func (h *Handler) CreateLateLog(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateLateLogRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.lateLogs.CreateLateLog(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể tạo ghi nhận đi muộn")
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List late logs
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param lateUserId query string false "Filter by late user ID"
// @Param from query string false "Filter from time (RFC3339 or YYYY-MM-DD)"
// @Param to query string false "Filter to time (RFC3339 or YYYY-MM-DD)"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[LateLogResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/late/logs [get]
func (h *Handler) ListLateLogs(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	params := pagination.Parse(r, 20, 100)
	var filter ListLateLogsFilter
	if err := httpx.DecodeQuery(r, &filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Tham số truy vấn không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.lateLogs.ListLateLogs(r.Context(), r.PathValue("workspaceId"), callerID, params, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách ghi nhận đi muộn")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update late log
// @Tags Performance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param lateLogId path string true "Late log ID"
// @Param request body UpdateLateLogRequest true "Update"
// @Success 200 {object} LateLogResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/late/logs/{lateLogId} [patch]
func (h *Handler) UpdateLateLog(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateLateLogRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.lateLogs.UpdateLateLog(r.Context(), r.PathValue("lateLogId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể cập nhật ghi nhận đi muộn")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete late log
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param lateLogId path string true "Late log ID"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/late/logs/{lateLogId} [delete]
func (h *Handler) DeleteLateLog(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.lateLogs.DeleteLateLog(r.Context(), r.PathValue("lateLogId"), callerID); err != nil {
		writeServiceError(w, err, "Không thể xoá ghi nhận đi muộn")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Create customer feedback
// @Tags Performance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateFeedbackRequest true "Feedback"
// @Success 201 {object} FeedbackResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/feedbacks [post]
func (h *Handler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateFeedbackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.feedbacks.CreateFeedback(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể tạo feedback")
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List customer feedbacks
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId query string false "Filter by beneficiary user ID"
// @Param from query string false "Filter from time (RFC3339 or YYYY-MM-DD)"
// @Param to query string false "Filter to time (RFC3339 or YYYY-MM-DD)"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[FeedbackResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/feedbacks [get]
func (h *Handler) ListFeedbacks(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	params := pagination.Parse(r, 20, 100)
	var filter ListFeedbacksFilter
	if err := httpx.DecodeQuery(r, &filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Tham số truy vấn không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.feedbacks.ListFeedbacks(r.Context(), r.PathValue("workspaceId"), callerID, params, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách feedback")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update customer feedback
// @Tags Performance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param feedbackId path string true "Feedback ID"
// @Param request body UpdateFeedbackRequest true "Update"
// @Success 200 {object} FeedbackResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/feedbacks/{feedbackId} [patch]
func (h *Handler) UpdateFeedback(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateFeedbackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.feedbacks.UpdateFeedback(r.Context(), r.PathValue("feedbackId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể cập nhật feedback")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete customer feedback
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param feedbackId path string true "Feedback ID"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/feedbacks/{feedbackId} [delete]
func (h *Handler) DeleteFeedback(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.feedbacks.DeleteFeedback(r.Context(), r.PathValue("feedbackId"), callerID); err != nil {
		writeServiceError(w, err, "Không thể xoá feedback")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Get monthly KPI for workspace members
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param month query string false "Month (YYYY-MM, defaults to current month)"
// @Param memberId query string false "Filter by member ID"
// @Success 200 {array} KPIMemberResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/kpi [get]
func (h *Handler) GetKPI(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var filter KPIFilter
	if err := httpx.DecodeQuery(r, &filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Tham số truy vấn không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.kpi.GetKPI(r.Context(), r.PathValue("workspaceId"), callerID, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy KPI")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Export monthly KPI appraisal sheets as an Excel workbook (admin/holder only)
// @Tags Performance
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param month query string false "Month (YYYY-MM, defaults to current month)"
// @Param memberId query string false "Export one member only; omit for the whole group"
// @Success 200 {file} binary "KPI workbook (.xlsx)"
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/kpi/export [get]
func (h *Handler) ExportKPI(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var filter KPIFilter
	if err := httpx.DecodeQuery(r, &filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Tham số truy vấn không hợp lệ: %s", err.Error()))
		return
	}
	file, err := h.kpi.ExportKPIExcel(r.Context(), r.PathValue("workspaceId"), callerID, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể xuất phiếu KPI")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(file.Data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(file.Data)
}

// @Summary Get my monthly KPI
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param month query string false "Month (YYYY-MM, defaults to current month)"
// @Success 200 {object} KPIMemberResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/kpi/me [get]
func (h *Handler) GetMyKPI(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var filter KPIFilter
	if err := httpx.DecodeQuery(r, &filter); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Tham số truy vấn không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.kpi.GetMyKPI(r.Context(), r.PathValue("workspaceId"), callerID, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy KPI của tôi")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Member monthly task breakdown (bug/late/overdue/rework)
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId path string true "Member ID"
// @Param month query string false "Month YYYY-MM (default: current)"
// @Success 200 {object} MemberTaskBreakdownResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/members/{memberId}/breakdown [get]
func (h *Handler) GetMemberTaskBreakdown(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.kpi.GetMemberTaskBreakdown(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("memberId"), r.URL.Query().Get("month"))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy chi tiết công việc của nhân viên")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Set work attitude score for a member (admin/holder only)
// @Tags Performance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body SetAttitudeRequest true "Attitude score"
// @Success 200 {object} AttitudeResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/kpi/attitude [put]
func (h *Handler) SetAttitude(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req SetAttitudeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.kpi.SetAttitude(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể đặt điểm thái độ")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get workspace KPI weight settings (holder only)
// @Tags Performance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} KPISettingsResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/kpi/settings [get]
func (h *Handler) GetKPISettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.kpi.GetKPISettings(r.Context(), r.PathValue("workspaceId"), callerID)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy cấu hình KPI")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Configure workspace KPI weight settings (holder only)
// @Tags Performance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body UpsertKPISettingsRequest true "KPI weights (must sum to 100)"
// @Success 200 {object} KPISettingsResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/kpi/settings [put]
func (h *Handler) UpsertKPISettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpsertKPISettingsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := validateKPISettingsRequest(req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.kpi.UpsertKPISettings(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể cập nhật cấu hình KPI")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
