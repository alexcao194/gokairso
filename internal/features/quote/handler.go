package quote

import (
	"errors"
	"net/http"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/currency"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/pagination"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("GET /v1/currencies", authMid.Handle(h.ListCurrencies))

	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/customers", authMid.Handle(h.ListCustomers))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/customers", authMid.Handle(h.CreateCustomer))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/customers/{customerId}", authMid.Handle(h.UpdateCustomer))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/customers/{customerId}", authMid.Handle(h.DeleteCustomer))

	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/quote-settings", authMid.Handle(h.GetSettings))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/quote-settings", authMid.Handle(h.UpdateSettings))

	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/quotes", authMid.Handle(h.ListQuotes))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/quotes", authMid.Handle(h.CreateQuote))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/quotes/{quoteId}", authMid.Handle(h.GetQuote))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/quotes/{quoteId}", authMid.Handle(h.UpdateQuote))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/quotes/{quoteId}", authMid.Handle(h.DeleteQuote))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/quotes/{quoteId}/send", authMid.Handle(h.SendQuote))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/quotes/{quoteId}/duplicate", authMid.Handle(h.DuplicateQuote))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/quotes/{quoteId}/status", authMid.Handle(h.SetStatus))
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
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrCustomerAbsent):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrNotEditable), errors.Is(err, ErrBadTransition):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		var ve ValidationError
		if errors.As(err, &ve) {
			httpx.Error(w, http.StatusBadRequest, ve.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "Lỗi máy chủ: "+err.Error())
	}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := httpx.DecodeJSON(r, v); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return false
	}
	return true
}

// @Summary List currencies
// @Description Every supported ISO 4217 currency with its minor-unit digits
// @Tags Quotes
// @Security BearerAuth
// @Produce json
// @Success 200 {array} currency.Currency
// @Router /v1/currencies [get]
func (h *Handler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.caller(w, r); !ok {
		return
	}
	var list []currency.Currency = h.svc.Currencies()
	httpx.JSON(w, http.StatusOK, list)
}

// @Summary List customers
// @Tags Quotes
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {array} CustomerResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/customers [get]
func (h *Handler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.ListCustomers(r.Context(), r.PathValue("workspaceId"), callerID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create customer
// @Tags Quotes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param body body CustomerRequest true "Customer"
// @Success 201 {object} CustomerResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/customers [post]
func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req CustomerRequest
	if !decode(w, r, &req) {
		return
	}
	resp, err := h.svc.CreateCustomer(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary Update customer
// @Tags Quotes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param customerId path string true "Customer ID"
// @Param body body CustomerRequest true "Customer"
// @Success 200 {object} CustomerResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/customers/{customerId} [put]
func (h *Handler) UpdateCustomer(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req CustomerRequest
	if !decode(w, r, &req) {
		return
	}
	resp, err := h.svc.UpdateCustomer(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("customerId"), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete customer
// @Description Quotes keep their customer snapshot; drafts lose the link
// @Tags Quotes
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param customerId path string true "Customer ID"
// @Success 204
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/customers/{customerId} [delete]
func (h *Handler) DeleteCustomer(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteCustomer(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("customerId")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Get quote settings
// @Tags Quotes
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} QuoteSettingsResponse
// @Router /v1/workspaces/{workspaceId}/quote-settings [get]
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.GetSettings(r.Context(), r.PathValue("workspaceId"), callerID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update quote settings
// @Tags Quotes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param body body QuoteSettingsRequest true "Settings"
// @Success 200 {object} QuoteSettingsResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/quote-settings [put]
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req QuoteSettingsRequest
	if !decode(w, r, &req) {
		return
	}
	resp, err := h.svc.UpdateSettings(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List quotes
// @Tags Quotes
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param moduleId query string false "Filter by module"
// @Param projectId query string false "Filter by project"
// @Param status query string false "Filter by stored status"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[QuoteSummaryResponse]
// @Router /v1/workspaces/{workspaceId}/quotes [get]
func (h *Handler) ListQuotes(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	resp, err := h.svc.ListQuotes(r.Context(), r.PathValue("workspaceId"), callerID,
		ListFilter{ModuleID: q.Get("moduleId"), ProjectID: q.Get("projectId"), Status: q.Get("status")},
		pagination.Parse(r, 30, 100))
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create a draft quote for a module
// @Tags Quotes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param body body CreateQuoteRequest true "Module (and optional customer)"
// @Success 201 {object} QuoteResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/quotes [post]
func (h *Handler) CreateQuote(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req CreateQuoteRequest
	if !decode(w, r, &req) {
		return
	}
	resp, err := h.svc.CreateQuote(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary Get quote
// @Tags Quotes
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param quoteId path string true "Quote ID"
// @Success 200 {object} QuoteResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/quotes/{quoteId} [get]
func (h *Handler) GetQuote(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.GetQuote(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("quoteId"))
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update a draft quote
// @Description Replaces the header and all lines; totals are recomputed server-side
// @Tags Quotes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param quoteId path string true "Quote ID"
// @Param body body UpdateQuoteRequest true "Quote"
// @Success 200 {object} QuoteResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/quotes/{quoteId} [put]
func (h *Handler) UpdateQuote(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req UpdateQuoteRequest
	if !decode(w, r, &req) {
		return
	}
	resp, err := h.svc.UpdateQuote(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("quoteId"), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete a draft quote
// @Tags Quotes
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param quoteId path string true "Quote ID"
// @Success 204
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/quotes/{quoteId} [delete]
func (h *Handler) DeleteQuote(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteQuote(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("quoteId")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Send a quote
// @Description Locks the draft and snapshots customer + issuer details
// @Tags Quotes
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param quoteId path string true "Quote ID"
// @Success 200 {object} QuoteResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/quotes/{quoteId}/send [post]
func (h *Handler) SendQuote(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.SendQuote(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("quoteId"))
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Duplicate a quote as a new draft
// @Tags Quotes
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param quoteId path string true "Quote ID"
// @Success 201 {object} QuoteResponse
// @Router /v1/workspaces/{workspaceId}/quotes/{quoteId}/duplicate [post]
func (h *Handler) DuplicateQuote(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.DuplicateQuote(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("quoteId"))
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary Set quote status
// @Description accepted/rejected (from sent) or cancelled (from draft/sent)
// @Tags Quotes
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param quoteId path string true "Quote ID"
// @Param body body SetQuoteStatusRequest true "Status"
// @Success 200 {object} QuoteResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/quotes/{quoteId}/status [post]
func (h *Handler) SetStatus(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req SetQuoteStatusRequest
	if !decode(w, r, &req) {
		return
	}
	resp, err := h.svc.SetStatus(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("quoteId"), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
