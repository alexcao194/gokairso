package notification

import (
	"context"
	"fmt"
	"net/http"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/pagination"
)

type WorkspaceIDResolver interface {
	GetWorkspaceID(ctx context.Context, workspaceAlias string) (string, error)
}

type Handler struct {
	svc          Service
	workspaceSvc WorkspaceIDResolver
}

func NewHandler(svc Service, workspaceSvc WorkspaceIDResolver) *Handler {
	return &Handler{svc: svc, workspaceSvc: workspaceSvc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/notifications", authMid.Handle(h.ListNotifications))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/notifications/unread-count", authMid.Handle(h.GetUnreadCount))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/notifications/read", authMid.Handle(h.ReadNotifications))
}

// @Summary List Notifications
// @Description List paginated notifications for the authenticated user
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[NotificationResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/notifications [get]
func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	workspaceID := r.PathValue("workspaceId")
	resp, err := h.svc.ListNotifications(r.Context(), workspaceID, userID, pagination.Parse(r, 20, 100))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách thông báo: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get Unread Notification Count
// @Description Get the number of unread notifications for the authenticated user
// @Tags Notifications
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} UnreadCountResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/notifications/unread-count [get]
func (h *Handler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	workspaceID := r.PathValue("workspaceId")
	count, err := h.svc.GetUnreadCount(r.Context(), workspaceID, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy số thông báo chưa đọc: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, UnreadCountResponse{UnreadCount: count})
}

// @Summary Read Notifications
// @Description Mark selected notifications as read, or all notifications when ids is empty
// @Tags Notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body ReadNotificationsRequest true "Notification ids. Empty ids reads all."
// @Success 204
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/notifications/read [patch]
func (h *Handler) ReadNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	workspaceID := r.PathValue("workspaceId")
	var req ReadNotificationsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := h.svc.ReadNotifications(r.Context(), workspaceID, userID, req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
