package attendance

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/pagination"
)

type Handler struct {
	settings      AttendanceSettingService
	attendance    AttendanceService
	schedules     WorkScheduleService
	stats         AttendanceStatisticsService
	taskStats     TaskStatisticsService
	workWeek      WorkWeekService
	checkRequests AttendanceCorrectionRequestService
	dailyReports  DailyReportService
}

func NewHandler(settings AttendanceSettingService, attendance AttendanceService, schedules WorkScheduleService, stats AttendanceStatisticsService, taskStats TaskStatisticsService, workWeek WorkWeekService, checkRequests AttendanceCorrectionRequestService, dailyReports DailyReportService) *Handler {
	return &Handler{settings: settings, attendance: attendance, schedules: schedules, stats: stats, taskStats: taskStats, workWeek: workWeek, checkRequests: checkRequests, dailyReports: dailyReports}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/attendance/settings", authMid.Handle(h.GetSettings))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/attendance/settings", authMid.Handle(h.UpsertSettings))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/attendance/status", authMid.Handle(h.CheckStatus))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/attendance/check/in", authMid.Handle(h.CheckIn))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/attendance/check/out", authMid.Handle(h.CheckOut))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/attendance/records", authMid.Handle(h.ListAttendanceRecords))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/attendance/records/{recordId}", authMid.Handle(h.DeleteAttendanceRecord))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/attendance/sessions", authMid.Handle(h.UpsertAttendanceSession))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/attendance/check-requests", authMid.Handle(h.SubmitCheckRequest))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/attendance/check-requests", authMid.Handle(h.ListCheckRequests))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/attendance/check-requests/me", authMid.Handle(h.ListMyCheckRequests))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/attendance/check-requests/{requestId}/approve", authMid.Handle(h.ApproveCheckRequest))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/attendance/check-requests/{requestId}/reject", authMid.Handle(h.RejectCheckRequest))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/work/schedules", authMid.Handle(h.UpsertWorkSchedules))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/work/schedules/{memberId}", authMid.Handle(h.UpsertMemberWorkSchedules))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/work/schedules/me", authMid.Handle(h.ListMyWorkSchedules))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/work/schedules/pending", authMid.Handle(h.ListPendingWorkSchedules))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/work/schedules/{memberId}", authMid.Handle(h.ListMemberWorkSchedules))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/work/schedules/{memberId}/approve", authMid.Handle(h.ApproveWorkSchedules))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/work/schedules/{memberId}/reject", authMid.Handle(h.RejectWorkSchedules))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/work/schedules/{memberId}/approve-all", authMid.Handle(h.ApproveAllWorkSchedules))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/attendance/statistics", authMid.Handle(h.GetAttendanceStatistics))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/attendance/statistics/me", authMid.Handle(h.GetMyAttendanceStatistics))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/task/statistics", authMid.Handle(h.GetTaskStatistics))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/task/statistics/me", authMid.Handle(h.GetMyTaskStatistics))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/work/week", authMid.Handle(h.GetWorkWeek))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/work/week", authMid.Handle(h.UpsertWorkWeek))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/work/week/me", authMid.Handle(h.GetMyWorkWeek))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/work/week/me", authMid.Handle(h.UpsertMyWorkWeek))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/daily-reports/me", authMid.Handle(h.UpsertMyDailyReport))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/daily-reports/me", authMid.Handle(h.ListMyDailyReports))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/daily-reports", authMid.Handle(h.ListDailyReports))
}

// @Summary Get attendance settings
// @Tags Attendance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} AttendanceSettingResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/settings [get]
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.settings.GetSettings(r.Context(), r.PathValue("workspaceId"), callerID)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy cấu hình chấm công")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Configure attendance settings
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body UpsertAttendanceSettingRequest true "Attendance settings"
// @Success 200 {object} AttendanceSettingResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/settings [put]
func (h *Handler) UpsertSettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpsertAttendanceSettingRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeAttendanceSettingRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.settings.UpsertSettings(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể cập nhật cấu hình chấm công")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Check attendance status
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body AttendanceCheckRequest true "Attendance check payload"
// @Success 200 {object} AttendanceStatusResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/status [post]
func (h *Handler) CheckStatus(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req AttendanceCheckRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeAttendanceCheckRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.attendance.CheckStatus(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể kiểm tra trạng thái chấm công")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Check in
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body AttendanceCheckRequest true "Attendance check payload"
// @Success 201 {object} AttendanceRecordResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/check/in [post]
func (h *Handler) CheckIn(w http.ResponseWriter, r *http.Request) {
	h.check(w, r, h.attendance.CheckIn)
}

// @Summary Check out
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body AttendanceCheckRequest true "Attendance check payload"
// @Success 201 {object} AttendanceRecordResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/check/out [post]
func (h *Handler) CheckOut(w http.ResponseWriter, r *http.Request) {
	h.check(w, r, h.attendance.CheckOut)
}

func (h *Handler) check(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, workspaceId, callerID string, req AttendanceCheckRequest) (*AttendanceRecordResponse, error)) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req AttendanceCheckRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeAttendanceCheckRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := fn(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể ghi nhận chấm công")
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List attendance records
// @Tags Attendance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param from query string false "RFC3339 start time"
// @Param to query string false "RFC3339 end time"
// @Param memberId query string false "Member id, holder/admin only"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[AttendanceRecordResponse]
// @Router /v1/workspaces/{workspaceId}/attendance/records [get]
func (h *Handler) ListAttendanceRecords(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := AttendanceRecordFilter{
		From:     r.URL.Query().Get("from"),
		To:       r.URL.Query().Get("to"),
		MemberID: r.URL.Query().Get("memberId"),
	}
	resp, err := h.attendance.ListRecords(r.Context(), r.PathValue("workspaceId"), callerID, filter, pagination.Parse(r, 20, 100))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách bản ghi chấm công")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete a member's check-in/check-out record (holder or admin)
// @Tags Attendance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param recordId path string true "Attendance record ID"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/records/{recordId} [delete]
func (h *Handler) DeleteAttendanceRecord(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.attendance.DeleteRecord(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("recordId")); err != nil {
		writeServiceError(w, err, "Không thể xoá bản ghi chấm công")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Add/correct/clear a member's check-in and check-out together (holder or admin)
// @Description Session-shaped counterpart to the single-record Create/Update/Delete endpoints — both halves are written atomically, so e.g. adding a check-out to an already checked-in session cannot leave the check-in touched but the check-out failed.
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body UpsertAttendanceSessionRequest true "Session edit"
// @Success 200 {object} AttendanceSessionResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/sessions [post]
func (h *Handler) UpsertAttendanceSession(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpsertAttendanceSessionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.attendance.UpsertSession(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể lưu ca chấm công")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Submit a request to add a missing check-in/check-out
// @Tags Attendance Correction Requests
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body SubmitAttendanceCorrectionRequest true "Correction request"
// @Success 201 {object} AttendanceCorrectionRequestResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/check-requests [post]
func (h *Handler) SubmitCheckRequest(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req SubmitAttendanceCorrectionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.checkRequests.Submit(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể gửi yêu cầu bổ sung công")
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List every attendance correction request in the workspace (holder or admin)
// @Tags Attendance Correction Requests
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId query string false "Filter by member id"
// @Param status query string false "Filter by status: pending, approved, or rejected"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[AttendanceCorrectionRequestResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/check-requests [get]
func (h *Handler) ListCheckRequests(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := AttendanceCorrectionRequestFilter{
		MemberID: r.URL.Query().Get("memberId"),
		Status:   r.URL.Query().Get("status"),
	}
	resp, err := h.checkRequests.List(r.Context(), r.PathValue("workspaceId"), callerID, filter, pagination.Parse(r, 20, 100))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách yêu cầu bổ sung công")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List my own attendance correction requests
// @Tags Attendance Correction Requests
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param status query string false "Filter by status: pending, approved, or rejected"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[AttendanceCorrectionRequestResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/check-requests/me [get]
func (h *Handler) ListMyCheckRequests(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := AttendanceCorrectionRequestFilter{
		Status: r.URL.Query().Get("status"),
	}
	resp, err := h.checkRequests.ListMine(r.Context(), r.PathValue("workspaceId"), callerID, filter, pagination.Parse(r, 20, 100))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách yêu cầu bổ sung công của tôi")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Approve an attendance correction request (holder or admin); may be called again to re-approve after a reject
// @Tags Attendance Correction Requests
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param requestId path string true "Correction request ID"
// @Success 200 {object} AttendanceCorrectionRequestResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/check-requests/{requestId}/approve [post]
func (h *Handler) ApproveCheckRequest(w http.ResponseWriter, r *http.Request) {
	h.decideCheckRequest(w, r, h.checkRequests.Approve, "Không thể duyệt yêu cầu bổ sung công")
}

// @Summary Reject an attendance correction request (holder or admin); may be called again to re-reject after an approve
// @Tags Attendance Correction Requests
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param requestId path string true "Correction request ID"
// @Success 200 {object} AttendanceCorrectionRequestResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/check-requests/{requestId}/reject [post]
func (h *Handler) RejectCheckRequest(w http.ResponseWriter, r *http.Request) {
	h.decideCheckRequest(w, r, h.checkRequests.Reject, "Không thể từ chối yêu cầu bổ sung công")
}

func (h *Handler) decideCheckRequest(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, workspaceId, callerID, requestID string) (*AttendanceCorrectionRequestResponse, error), fallback string) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := fn(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("requestId"))
	if err != nil {
		writeServiceError(w, err, fallback)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create or update own work schedules
// @Tags Work Schedules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body UpsertWorkSchedulesRequest true "Work schedules"
// @Success 200 {array} WorkScheduleDayResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/work/schedules [put]
func (h *Handler) UpsertWorkSchedules(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpsertWorkSchedulesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.schedules.UpsertWorkSchedules(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể cập nhật lịch làm việc")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Set a member's work schedule (holder or admin)
// @Tags Work Schedules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId path string true "Member ID"
// @Param request body UpsertMemberWorkSchedulesRequest true "Work schedules"
// @Success 200 {array} WorkScheduleDayResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/work/schedules/{memberId} [put]
func (h *Handler) UpsertMemberWorkSchedules(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpsertMemberWorkSchedulesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.schedules.UpsertMemberWorkSchedules(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("memberId"), req)
	if err != nil {
		writeServiceError(w, err, "Không thể cập nhật lịch làm việc của thành viên")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List my work schedules
// @Tags Work Schedules
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param month query string false "Month YYYY-MM"
// @Param from query string false "Start date YYYY-MM-DD"
// @Param to query string false "End date YYYY-MM-DD"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkScheduleDayResponse]
// @Router /v1/workspaces/{workspaceId}/work/schedules/me [get]
func (h *Handler) ListMyWorkSchedules(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := WorkScheduleFilter{
		Month:    r.URL.Query().Get("month"),
		From:     r.URL.Query().Get("from"),
		To:       r.URL.Query().Get("to"),
		MemberID: callerID,
	}
	resp, err := h.schedules.ListMySchedules(r.Context(), r.PathValue("workspaceId"), callerID, filter, pagination.Parse(r, 20, 100))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách lịch làm việc của tôi")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List pending work schedule days across every member in the workspace (holder or admin)
// @Tags Work Schedules
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param month query string false "Month YYYY-MM"
// @Param from query string false "Start date YYYY-MM-DD"
// @Param to query string false "End date YYYY-MM-DD"
// @Param status query string false "Filter by status: pending, approved, or rejected (default pending)"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkScheduleDayResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/work/schedules/pending [get]
func (h *Handler) ListPendingWorkSchedules(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := WorkScheduleFilter{
		Month:  r.URL.Query().Get("month"),
		From:   r.URL.Query().Get("from"),
		To:     r.URL.Query().Get("to"),
		Status: r.URL.Query().Get("status"),
	}
	resp, err := h.schedules.ListPendingSchedules(r.Context(), r.PathValue("workspaceId"), callerID, filter, pagination.Parse(r, 20, 100))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách lịch làm việc chờ duyệt")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List a member's work schedules, including pending approval (holder or admin)
// @Tags Work Schedules
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId path string true "Member ID"
// @Param month query string false "Month YYYY-MM"
// @Param from query string false "Start date YYYY-MM-DD"
// @Param to query string false "End date YYYY-MM-DD"
// @Param status query string false "Filter by status: pending, approved, or rejected"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkScheduleDayResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/work/schedules/{memberId} [get]
func (h *Handler) ListMemberWorkSchedules(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := WorkScheduleFilter{
		Month:  r.URL.Query().Get("month"),
		From:   r.URL.Query().Get("from"),
		To:     r.URL.Query().Get("to"),
		Status: r.URL.Query().Get("status"),
	}
	resp, err := h.schedules.ListMemberSchedules(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("memberId"), filter, pagination.Parse(r, 20, 100))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách lịch làm việc của thành viên")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Approve a member's pending work schedule days (holder or admin)
// @Tags Work Schedules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId path string true "Member ID"
// @Param request body UpdateWorkScheduleStatusRequest true "Day IDs to approve"
// @Success 200 {array} WorkScheduleDayResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/work/schedules/{memberId}/approve [post]
func (h *Handler) ApproveWorkSchedules(w http.ResponseWriter, r *http.Request) {
	h.updateScheduleStatus(w, r, h.schedules.ApproveSchedules, "Không thể duyệt lịch làm việc")
}

// @Summary Reject a member's pending work schedule days (holder or admin)
// @Tags Work Schedules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId path string true "Member ID"
// @Param request body UpdateWorkScheduleStatusRequest true "Day IDs to reject"
// @Success 200 {array} WorkScheduleDayResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/work/schedules/{memberId}/reject [post]
func (h *Handler) RejectWorkSchedules(w http.ResponseWriter, r *http.Request) {
	h.updateScheduleStatus(w, r, h.schedules.RejectSchedules, "Không thể từ chối lịch làm việc")
}

// @Summary Approve every pending work schedule day for a member in one action (holder or admin)
// @Tags Work Schedules
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId path string true "Member ID"
// @Success 200 {array} WorkScheduleDayResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/work/schedules/{memberId}/approve-all [post]
func (h *Handler) ApproveAllWorkSchedules(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.schedules.ApproveAllPendingSchedules(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("memberId"))
	if err != nil {
		writeServiceError(w, err, "Không thể duyệt toàn bộ lịch làm việc")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) updateScheduleStatus(w http.ResponseWriter, r *http.Request, fn func(ctx context.Context, workspaceId, callerID, memberID string, req UpdateWorkScheduleStatusRequest) ([]WorkScheduleDayResponse, error), fallback string) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateWorkScheduleStatusRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := fn(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("memberId"), req)
	if err != nil {
		writeServiceError(w, err, fallback)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get monthly attendance statistics
// @Tags Attendance Statistics
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param month query string false "Month YYYY-MM"
// @Param memberId query string false "Member id, holder/admin only"
// @Success 200 {object} AttendanceStatisticsSummaryResponse
// @Router /v1/workspaces/{workspaceId}/attendance/statistics [get]
func (h *Handler) GetAttendanceStatistics(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := AttendanceStatisticsFilter{
		Month:    r.URL.Query().Get("month"),
		Week:     r.URL.Query().Get("week"),
		Year:     r.URL.Query().Get("year"),
		MemberID: r.URL.Query().Get("memberId"),
	}
	resp, err := h.stats.GetMonthlySummary(r.Context(), r.PathValue("workspaceId"), callerID, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy thống kê chấm công")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get task statistics by project/module with optional time period
// @Tags Task Statistics
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId query string false "Member id, holder/admin only"
// @Param projectId query string false "Filter by project ID"
// @Param moduleId query string false "Filter by module id"
// @Param month query string false "Month YYYY-MM"
// @Param week query string false "ISO week YYYY-Www or date YYYY-MM-DD"
// @Param year query string false "Year YYYY"
// @Success 200 {object} TaskStatisticsSummaryResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/task/statistics [get]
func (h *Handler) GetTaskStatistics(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := TaskStatisticsFilter{
		MemberID:  r.URL.Query().Get("memberId"),
		ProjectID: r.URL.Query().Get("projectId"),
		ModuleID:  r.URL.Query().Get("moduleId"),
		Month:     r.URL.Query().Get("month"),
		Week:      r.URL.Query().Get("week"),
		Year:      r.URL.Query().Get("year"),
	}
	resp, err := h.taskStats.GetTaskStatistics(r.Context(), r.PathValue("workspaceId"), callerID, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy thống kê task")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get my own attendance statistics
// @Tags Attendance Statistics
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param month query string false "Month YYYY-MM"
// @Param week query string false "ISO week YYYY-Www or date YYYY-MM-DD"
// @Param year query string false "Year YYYY"
// @Success 200 {object} MyAttendanceStatisticsResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/attendance/statistics/me [get]
func (h *Handler) GetMyAttendanceStatistics(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := AttendanceStatisticsFilter{
		Month: r.URL.Query().Get("month"),
		Week:  r.URL.Query().Get("week"),
		Year:  r.URL.Query().Get("year"),
	}
	resp, err := h.stats.GetMyAttendanceSummary(r.Context(), r.PathValue("workspaceId"), callerID, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy thống kê chấm công")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get my own task statistics
// @Tags Task Statistics
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId query string false "Filter by project ID"
// @Param moduleId query string false "Filter by module id"
// @Param month query string false "Month YYYY-MM"
// @Param week query string false "ISO week YYYY-Www or date YYYY-MM-DD"
// @Param year query string false "Year YYYY"
// @Success 200 {object} MyTaskStatisticsResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/task/statistics/me [get]
func (h *Handler) GetMyTaskStatistics(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := TaskStatisticsFilter{
		ProjectID: r.URL.Query().Get("projectId"),
		ModuleID:  r.URL.Query().Get("moduleId"),
		Month:     r.URL.Query().Get("month"),
		Week:      r.URL.Query().Get("week"),
		Year:      r.URL.Query().Get("year"),
	}
	resp, err := h.taskStats.GetMyTaskStatistics(r.Context(), r.PathValue("workspaceId"), callerID, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy thống kê task")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get workspace default work week
// @Tags Attendance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} WorkWeekResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/work/week [get]
func (h *Handler) GetWorkWeek(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.workWeek.GetWorkWeek(r.Context(), r.PathValue("workspaceId"), callerID)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy lịch tuần mặc định")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Set workspace default work week
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body UpsertWorkWeekRequest true "Work week config"
// @Success 200 {object} WorkWeekResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/work/week [put]
func (h *Handler) UpsertWorkWeek(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpsertWorkWeekRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ: "+err.Error())
		return
	}
	resp, err := h.workWeek.UpsertWorkWeek(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể cập nhật lịch tuần mặc định")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get my personal default work week
// @Tags Attendance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} WorkWeekResponse
// @Router /v1/workspaces/{workspaceId}/work/week/me [get]
func (h *Handler) GetMyWorkWeek(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.workWeek.GetMyWorkWeek(r.Context(), r.PathValue("workspaceId"), callerID)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy lịch mặc định của tôi")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Save my personal default work week
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body UpsertWorkWeekRequest true "Default week"
// @Success 200 {object} WorkWeekResponse
// @Router /v1/workspaces/{workspaceId}/work/week/me [put]
func (h *Handler) UpsertMyWorkWeek(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpsertWorkWeekRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ: "+err.Error())
		return
	}
	resp, err := h.workWeek.UpsertMyWorkWeek(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể lưu lịch mặc định của tôi")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Save my end-of-day report for a date
// @Description Creates or updates the caller's daily report for the given date (one per member per day — checking out again the same day reloads and edits the existing one). done = features finished, inProgress = what's in progress, issues = anything to discuss.
// @Tags Attendance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body UpsertDailyReportRequest true "Daily report"
// @Success 200 {object} DailyReportResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/daily-reports/me [put]
func (h *Handler) UpsertMyDailyReport(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpsertDailyReportRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ: "+err.Error())
		return
	}
	resp, err := h.dailyReports.UpsertMyDailyReport(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể lưu báo cáo cuối ngày")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List my end-of-day reports
// @Tags Attendance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param from query string false "From date (YYYY-MM-DD)"
// @Param to query string false "To date (YYYY-MM-DD)"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} pagination.Response[DailyReportResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/daily-reports/me [get]
func (h *Handler) ListMyDailyReports(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := DailyReportFilter{
		From: r.URL.Query().Get("from"),
		To:   r.URL.Query().Get("to"),
	}
	resp, err := h.dailyReports.ListMyDailyReports(r.Context(), r.PathValue("workspaceId"), callerID, filter, pagination.Parse(r, 20, 100))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy báo cáo cuối ngày của tôi")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List members' end-of-day reports (holder or admin)
// @Tags Attendance
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId query string false "Filter by member ID"
// @Param from query string false "From date (YYYY-MM-DD)"
// @Param to query string false "To date (YYYY-MM-DD)"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} pagination.Response[DailyReportResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/daily-reports [get]
func (h *Handler) ListDailyReports(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := DailyReportFilter{
		MemberID: r.URL.Query().Get("memberId"),
		From:     r.URL.Query().Get("from"),
		To:       r.URL.Query().Get("to"),
	}
	resp, err := h.dailyReports.ListDailyReports(r.Context(), r.PathValue("workspaceId"), callerID, filter, pagination.Parse(r, 20, 100))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách báo cáo cuối ngày")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func writeServiceError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrAttendanceValidationFailed), errors.Is(err, ErrInvalidWorkSchedule), errors.Is(err, ErrInvalidDailyReport):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("%s: %s", fallback, err.Error()))
	}
}
