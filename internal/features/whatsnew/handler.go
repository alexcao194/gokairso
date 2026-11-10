package whatsnew

import (
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
	mux.HandleFunc("GET /v1/whats-new", authMid.Handle(h.ListWhatsNew))
}

// @Summary List What's New releases
// @Description List paginated app release notes ("what's new"), newest first
// @Tags WhatsNew
// @Produce json
// @Security BearerAuth
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WhatsNewReleaseResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/whats-new [get]
func (h *Handler) ListWhatsNew(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.GetUserIDFromContext(r.Context()); !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	resp, err := h.svc.ListReleases(r.Context(), pagination.Parse(r, 20, 100))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "Không thể lấy danh sách các bản phát hành: "+err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
