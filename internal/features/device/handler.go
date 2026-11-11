package device

import (
	"net/http"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("POST /v1/devices", authMid.Handle(h.Register))
	mux.HandleFunc("DELETE /v1/devices", authMid.Handle(h.Unregister))
}

// @Summary Register a push device token
// @Description Register or refresh the caller's FCM device token for push notifications
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body RegisterDeviceRequest true "Device token"
// @Success 204
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/devices [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req RegisterDeviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Dữ liệu không hợp lệ")
		return
	}
	if err := h.svc.Register(r.Context(), userID, req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Unregister a push device token
// @Description Remove the caller's FCM device token (e.g. on logout)
// @Tags Devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UnregisterDeviceRequest true "Device token"
// @Success 204
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/devices [delete]
func (h *Handler) Unregister(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UnregisterDeviceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Dữ liệu không hợp lệ")
		return
	}
	if err := h.svc.Unregister(r.Context(), userID, req.Token); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
