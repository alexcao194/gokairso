package user

import (
	"fmt"
	"net/http"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/pgutil"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("GET /v1/users/me", authMid.Handle(h.GetMe))
	mux.HandleFunc("PATCH /v1/users/me", authMid.Handle(h.UpdateMe))
}

// @Summary Get Current User
// @Description Get the authenticated user profile
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/users/me [get]
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	user, err := h.svc.GetMe(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, fmt.Sprintf("Không tìm thấy người dùng: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

// @Summary Update Current User
// @Description Update current user display profile
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateMeRequest true "Profile fields"
// @Success 200 {object} UserResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/users/me [patch]
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateMeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeUpdateRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := h.svc.UpdateMe(r.Context(), userID, req)
	if err != nil {
		if pgutil.IsAliasConflict(err) {
			httpx.Error(w, http.StatusConflict, "Alias đã tồn tại")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật người dùng: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}
