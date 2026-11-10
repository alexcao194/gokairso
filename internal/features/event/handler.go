package event

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
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/calendar/events", authMid.Handle(h.ListEvents))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/calendar/events/me", authMid.Handle(h.ListMyEvents))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/calendar/members/events", authMid.Handle(h.ListMembersEvents))
}

// @Summary List calendar events (schedule, attendance + all members' birthdays + holidays) for a member or workspace
// @Tags Calendar Events
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param memberId query string false "Member ID (optional, default to caller)"
// @Param from query string false "RFC3339 start time or YYYY-MM-DD"
// @Param to query string false "RFC3339 end time or YYYY-MM-DD"
// @Success 200 {array} CalendarDayResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/calendar/events [get]
func (h *Handler) ListEvents(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := CalendarEventFilter{
		From:     r.URL.Query().Get("from"),
		To:       r.URL.Query().Get("to"),
		MemberID: r.URL.Query().Get("memberId"),
	}
	resp, err := h.svc.ListEvents(r.Context(), r.PathValue("workspaceId"), callerID, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách sự kiện lịch")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List my calendar events (own schedule, attendance + all members' birthdays + holidays)
// @Tags Calendar Events
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param from query string false "RFC3339 start time or YYYY-MM-DD"
// @Param to query string false "RFC3339 end time or YYYY-MM-DD"
// @Success 200 {array} CalendarDayResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/calendar/events/me [get]
func (h *Handler) ListMyEvents(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	filter := CalendarEventFilter{
		From: r.URL.Query().Get("from"),
		To:   r.URL.Query().Get("to"),
	}
	resp, err := h.svc.ListEvents(r.Context(), r.PathValue("workspaceId"), callerID, filter)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách sự kiện lịch")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List work schedule and attendance for all workspace members (admin/holder only)
// @Tags Calendar Events
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param from query string false "RFC3339 start time or YYYY-MM-DD"
// @Param to query string false "RFC3339 end time or YYYY-MM-DD"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[MemberCalendarResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/calendar/members/events [get]
func (h *Handler) ListMembersEvents(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	params := pagination.Parse(r, 20, 100)
	filter := CalendarEventFilter{
		From: r.URL.Query().Get("from"),
		To:   r.URL.Query().Get("to"),
	}
	resp, err := h.svc.ListMembersEvents(r.Context(), r.PathValue("workspaceId"), callerID, filter, params)
	if err != nil {
		writeServiceError(w, err, "Không thể lấy danh sách lịch của các thành viên")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func writeServiceError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("%s: %s", fallback, err.Error()))
	}
}
