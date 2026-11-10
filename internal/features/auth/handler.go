package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/jwt"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/firebase/login", h.LoginWithFirebase)
	mux.HandleFunc("POST /v1/auth/refresh", h.Refresh)
}

// @Summary Login With Firebase
// @Description Verify a Firebase ID token from web, iOS, or Android sign-in providers such as Google and Apple, then return backend JWT tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body FirebaseLoginRequest true "Firebase ID token and alias"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/auth/firebase/login [post]
func (h *Handler) LoginWithFirebase(w http.ResponseWriter, r *http.Request) {
	var req FirebaseLoginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.IDToken = strings.TrimSpace(req.IDToken)
	if req.IDToken == "" {
		httpx.Error(w, http.StatusBadRequest, "idToken là bắt buộc")
		return
	}
	resp, err := h.svc.LoginWithFirebase(r.Context(), req.IDToken)
	if err != nil {
		if errors.Is(err, ErrInvalidFirebaseToken) {
			httpx.Error(w, http.StatusUnauthorized, "Firebase token không hợp lệ")
			return
		}
		if errors.Is(err, ErrAliasAlreadyExists) {
			httpx.Error(w, http.StatusConflict, "Alias đã tồn tại")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Đăng nhập Firebase thất bại: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Refresh Token
// @Description Exchange a valid backend refresh token for a new access token and refresh token pair
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token"
// @Success 200 {object} TokenResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		httpx.Error(w, http.StatusBadRequest, "refreshToken là bắt buộc")
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrInvalidToken) || errors.Is(err, jwt.ErrExpiredToken) {
			httpx.Error(w, http.StatusUnauthorized, err.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Làm mới phiên đăng nhập thất bại: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
