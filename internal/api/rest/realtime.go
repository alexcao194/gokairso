package rest

import (
	"net/http"
	"strings"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/jwt"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/alexcao194/kairso/pkg/realtime"
)

type RealtimeHandler struct {
	hub     *realtime.Hub
	jwtSvc  jwt.TokenService
	queries *database.Queries
}

func NewRealtimeHandler(hub *realtime.Hub, jwtSvc jwt.TokenService, queries *database.Queries) *RealtimeHandler {
	return &RealtimeHandler{hub: hub, jwtSvc: jwtSvc, queries: queries}
}

func (h *RealtimeHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ws", h.Serve)
}

func (h *RealtimeHandler) Serve(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if token == "" {
		httpx.Error(w, http.StatusUnauthorized, "token là bắt buộc")
		return
	}
	claims, err := h.jwtSvc.ValidateToken(token)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	userID, err := pgutil.ParseUUID(claims.UserID)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "Người dùng không hợp lệ")
		return
	}
	workspaces, err := h.queries.ListWorkspacesByUser(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "Không thể xác định workspace của bạn")
		return
	}
	workspaceIDs := make([]string, len(workspaces))
	for i, ws := range workspaces {
		workspaceIDs[i] = pgutil.UUIDToString(ws.ID)
	}
	connID := r.URL.Query().Get("connId")

	_ = h.hub.ServeWS(w, r, workspaceIDs, connID)
}
