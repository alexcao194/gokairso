package webhook

import (
	"errors"
	"net/http"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/webhooks", authMid.Handle(h.List))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/webhooks", authMid.Handle(h.Create))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/webhooks/{webhookId}", authMid.Handle(h.Update))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/webhooks/{webhookId}", authMid.Handle(h.Delete))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/webhooks/{webhookId}/test", authMid.Handle(h.Test))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/webhooks/{webhookId}/deliveries", authMid.Handle(h.ListDeliveries))
}

func (h *Handler) caller(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return "", false
	}
	return id, true
}

func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpx.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	default:
		var ve ValidationError
		if errors.As(err, &ve) {
			httpx.Error(w, http.StatusBadRequest, ve.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "Lỗi máy chủ: "+err.Error())
	}
}

// @Summary List webhooks
// @Description Outgoing webhooks of a workspace (holder/admin only)
// @Tags Webhooks
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {array} WebhookResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/webhooks [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.List(r.Context(), r.PathValue("workspaceId"), callerID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create webhook
// @Description Every workspace event is POSTed to the URL as a Payload, with "Authorization: Bearer <token>" and an "X-Kairso-Signature: sha256=<HMAC-SHA256(token, body)>" header
// @Tags Webhooks
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param body body WebhookRequest true "Webhook"
// @Success 201 {object} WebhookResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/webhooks [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req WebhookRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return
	}
	resp, err := h.svc.Create(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary Update webhook
// @Description An empty token keeps the current one; re-enabling resets the failure counter
// @Tags Webhooks
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param webhookId path string true "Webhook ID"
// @Param body body WebhookRequest true "Webhook"
// @Success 200 {object} WebhookResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/webhooks/{webhookId} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req WebhookRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return
	}
	resp, err := h.svc.Update(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("webhookId"), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete webhook
// @Tags Webhooks
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param webhookId path string true "Webhook ID"
// @Success 204
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/webhooks/{webhookId} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("webhookId")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Send a test event
// @Description Sends one "webhook.test" event immediately and returns the endpoint's answer
// @Tags Webhooks
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param webhookId path string true "Webhook ID"
// @Success 200 {object} WebhookDeliveryResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/webhooks/{webhookId}/test [post]
func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.Test(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("webhookId"))
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Recent deliveries
// @Description The latest 50 deliveries of a webhook (kept for 7 days)
// @Tags Webhooks
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param webhookId path string true "Webhook ID"
// @Success 200 {array} WebhookDeliveryResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/webhooks/{webhookId}/deliveries [get]
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.ListDeliveries(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("webhookId"))
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
