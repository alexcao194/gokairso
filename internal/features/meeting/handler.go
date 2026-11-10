package meeting

import (
	"errors"
	"fmt"
	"net/http"

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

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/meetings", authMid.Handle(h.ListMeetings))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/meetings", authMid.Handle(h.CreateMeeting))
	mux.HandleFunc("GET /v1/meetings/{meetingId}", authMid.Handle(h.GetMeeting))
	mux.HandleFunc("PATCH /v1/meetings/{meetingId}", authMid.Handle(h.UpdateMeeting))
	mux.HandleFunc("DELETE /v1/meetings/{meetingId}", authMid.Handle(h.DeleteMeeting))
}

func writeServiceError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalid):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("%s: %s", fallback, err.Error()))
	}
}

// @Summary List meeting minutes
// @Description Workspace-level minutes, or one project's when projectId is set. Newest meeting first.
// @Tags Meetings
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId query string false "Project ID (omit for workspace-level minutes)"
// @Param q query string false "Search title and line items"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[MeetingMinutesResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/meetings [get]
func (h *Handler) ListMeetings(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := ListMeetingMinutesFilter{ProjectID: r.URL.Query().Get("projectId"), Query: r.URL.Query().Get("q")}
	resp, err := h.svc.List(r.Context(), r.PathValue("workspaceId"), userID, filter, pagination.Parse(r, 20, 100))
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách biên bản họp")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create meeting minutes
// @Tags Meetings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateMeetingMinutesRequest true "Minutes"
// @Success 201 {object} MeetingMinutesResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/meetings [post]
func (h *Handler) CreateMeeting(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateMeetingMinutesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.svc.Create(r.Context(), r.PathValue("workspaceId"), userID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể tạo biên bản họp")
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary Get meeting minutes
// @Tags Meetings
// @Produce json
// @Security BearerAuth
// @Param meetingId path string true "Meeting minutes ID"
// @Success 200 {object} MeetingMinutesResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/meetings/{meetingId} [get]
func (h *Handler) GetMeeting(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.svc.Get(r.Context(), r.PathValue("meetingId"), userID)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy biên bản họp")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update meeting minutes
// @Description Only the fields sent change; items/sectionNotes replace the whole list/map.
// @Tags Meetings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param meetingId path string true "Meeting minutes ID"
// @Param request body UpdateMeetingMinutesRequest true "Changes"
// @Success 200 {object} MeetingMinutesResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/meetings/{meetingId} [patch]
func (h *Handler) UpdateMeeting(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateMeetingMinutesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.svc.Update(r.Context(), r.PathValue("meetingId"), userID, req)
	if err != nil {
		writeServiceError(w, err, "Không thể cập nhật biên bản họp")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete meeting minutes
// @Tags Meetings
// @Security BearerAuth
// @Param meetingId path string true "Meeting minutes ID"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/meetings/{meetingId} [delete]
func (h *Handler) DeleteMeeting(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.svc.Delete(r.Context(), r.PathValue("meetingId"), userID); err != nil {
		writeServiceError(w, err, "Không thể xoá biên bản họp")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
