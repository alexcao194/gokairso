package workspace

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("POST /v1/workspaces", authMid.Handle(h.CreateWorkspace))
	mux.HandleFunc("GET /v1/workspaces", authMid.Handle(h.ListWorkspaces))
	mux.HandleFunc("GET /v1/invitations", authMid.Handle(h.ListMyInvitations))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}", authMid.Handle(h.UpdateWorkspace))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/history", authMid.Handle(h.ListWorkspaceHistory))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}", authMid.Handle(h.DeleteWorkspace))

	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/members", authMid.Handle(h.ListWorkspaceMembers))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/invitations", authMid.Handle(h.InviteMember))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/invitations/{invitationId}/accept", authMid.Handle(h.AcceptInvitation))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/invitations/{invitationId}/reject", authMid.Handle(h.RejectInvitation))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/invitations/{invitationId}", authMid.Handle(h.CancelInvitation))
	mux.HandleFunc("POST /v1/invitations/{invitationId}/accept", authMid.Handle(h.AcceptInvitation))
	mux.HandleFunc("POST /v1/invitations/{invitationId}/reject", authMid.Handle(h.RejectInvitation))
	mux.HandleFunc("DELETE /v1/invitations/{invitationId}", authMid.Handle(h.CancelInvitation))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/members/{userId}", authMid.Handle(h.RemoveMember))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/members/{userId}", authMid.Handle(h.UpdateMemberRole))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/links", authMid.Handle(h.CreateWorkspaceLink))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/links", authMid.Handle(h.ListWorkspaceLinks))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/links/{linkId}", authMid.Handle(h.DeleteWorkspaceLink))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/notes", authMid.Handle(h.CreateWorkspaceNote))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/notes", authMid.Handle(h.ListWorkspaceNotes))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/notes/{noteId}", authMid.Handle(h.UpdateWorkspaceNote))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/notes/{noteId}", authMid.Handle(h.DeleteWorkspaceNote))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/files", authMid.Handle(h.CreateWorkspaceFile))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/files", authMid.Handle(h.ListWorkspaceFiles))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/files/{fileId}", authMid.Handle(h.DeleteWorkspaceFile))
}

// @Summary Create Workspace
// @Tags Workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateWorkspaceRequest true "Workspace"
// @Success 201 {object} WorkspaceResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/workspaces [post]
func (h *Handler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateWorkspaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "Tên là bắt buộc")
		return
	}
	resp, err := h.svc.CreateWorkspace(r.Context(), userID, req)
	if err != nil {
		if pgutil.IsAliasConflict(err) {
			httpx.Error(w, http.StatusConflict, "Alias đã tồn tại")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Workspaces
// @Tags Workspaces
// @Produce json
// @Security BearerAuth
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkspaceResponse]
// @Router /v1/workspaces [get]
func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListWorkspaces(r.Context(), userID, params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update Workspace
// @Tags Workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body UpdateWorkspaceRequest true "Update fields"
// @Success 200 {object} WorkspaceResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId} [patch]
func (h *Handler) UpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateWorkspaceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeUpdateWorkspaceRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.UpdateWorkspace(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		if pgutil.IsAliasConflict(err) {
			httpx.Error(w, http.StatusConflict, "Alias đã tồn tại")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List Workspace Update History
// @Tags Workspaces
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkspaceHistoryResponse]
// @Router /v1/workspaces/{workspaceId}/history [get]
func (h *Handler) ListWorkspaceHistory(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListWorkspaceHistory(r.Context(), r.PathValue("workspaceId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách lịch sử workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Workspace
// @Tags Workspaces
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId} [delete]
func (h *Handler) DeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteWorkspace(r.Context(), r.PathValue("workspaceId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá workspace: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary List Workspace Members
// @Tags Workspaces
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkspaceMemberResponse]
// @Router /v1/workspaces/{workspaceId}/members [get]
func (h *Handler) ListWorkspaceMembers(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListWorkspaceMembers(r.Context(), r.PathValue("workspaceId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách thành viên workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Invite Member to Workspace
// @Description Invite a user by alias to join the workspace. Caller must be holder or admin.
// @Tags Workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body InviteMemberRequest true "Invitation"
// @Success 201 {object} WorkspaceInvitationResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/invitations [post]
func (h *Handler) InviteMember(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req InviteMemberRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.UserAlias = strings.TrimSpace(req.UserAlias)
	if req.UserAlias == "" {
		httpx.Error(w, http.StatusBadRequest, "userAlias là bắt buộc")
		return
	}
	if !req.Role.Valid() {
		httpx.Error(w, http.StatusBadRequest, "Vai trò phải là một trong: holder, admin, member")
		return
	}
	resp, err := h.svc.InviteMember(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		if errors.Is(err, ErrCannotManageEqualOrHigherRole) {
			httpx.Error(w, http.StatusForbidden, "Bạn chỉ được mời thành viên với vai trò thấp hơn hoặc bằng mình")
			return
		}
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "Chỉ chủ workspace hoặc quản trị viên mới được mời thành viên")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể mời thành viên: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List My Workspace Invitations
// @Description List pending workspace invitations for the authenticated user.
// @Tags Workspaces
// @Produce json
// @Security BearerAuth
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkspaceInvitationSummaryResponse]
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/invitations [get]
func (h *Handler) ListMyInvitations(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListMyInvitations(r.Context(), userID, params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách lời mời: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Accept Workspace Invitation
// @Tags Workspaces
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param invitationId path string true "Invitation ID"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/invitations/{invitationId}/accept [post]
func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.svc.AcceptInvitation(r.Context(), r.PathValue("invitationId"), callerID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể chấp nhận lời mời: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Reject Workspace Invitation
// @Tags Workspaces
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param invitationId path string true "Invitation ID"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/invitations/{invitationId}/reject [post]
func (h *Handler) RejectInvitation(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.svc.RejectInvitation(r.Context(), r.PathValue("invitationId"), callerID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể từ chối lời mời: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Cancel Workspace Invitation
// @Description Cancel a pending workspace invitation. Caller must be holder or admin.
// @Tags Workspaces
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param invitationId path string true "Invitation ID"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/invitations/{invitationId} [delete]
func (h *Handler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.svc.CancelInvitation(r.Context(), r.PathValue("invitationId"), callerID); err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "Chỉ chủ workspace hoặc quản trị viên mới được huỷ lời mời")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể huỷ lời mời: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Remove Member from Workspace
// @Description Remove a member by user ID. Caller must be holder or admin.
// @Tags Workspaces
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param userId path string true "User ID"
// @Success 204
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/members/{userId} [delete]
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.svc.RemoveMember(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("userId")); err != nil {
		if errors.Is(err, ErrCannotManageEqualOrHigherRole) {
			httpx.Error(w, http.StatusForbidden, "Bạn chỉ được gỡ người có vai trò thấp hơn hoặc bằng mình")
			return
		}
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "Chỉ chủ workspace hoặc quản trị viên mới được xoá thành viên")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể gỡ thành viên: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Update Member Role in Workspace
// @Description Change a member's role. Caller must be holder or admin.
// @Tags Workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param userId path string true "User ID"
// @Param request body UpdateMemberRoleRequest true "Role update"
// @Success 200 {object} WorkspaceMemberResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/members/{userId} [patch]
func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateMemberRoleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if !req.Role.Valid() {
		httpx.Error(w, http.StatusBadRequest, "Vai trò phải là một trong: holder, admin, member")
		return
	}
	resp, err := h.svc.UpdateMemberRole(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("userId"), req.Role)
	if err != nil {
		if errors.Is(err, ErrCannotManageEqualOrHigherRole) {
			httpx.Error(w, http.StatusForbidden, "Bạn chỉ được đổi vai trò của người có cấp thấp hơn hoặc bằng mình")
			return
		}
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "Chỉ chủ workspace hoặc quản trị viên mới được đổi vai trò thành viên")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật vai trò thành viên: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create Workspace Link
// @Tags Workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateWorkspaceLinkRequest true "Link data"
// @Success 201 {object} WorkspaceLinkResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/links [post]
func (h *Handler) CreateWorkspaceLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateWorkspaceLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.URL = strings.TrimSpace(req.URL)
	if req.Title == "" || req.URL == "" {
		httpx.Error(w, http.StatusBadRequest, "Tiêu đề và đường dẫn là bắt buộc")
		return
	}
	resp, err := h.svc.CreateWorkspaceLink(r.Context(), r.PathValue("workspaceId"), userID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo liên kết workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Workspace Links
// @Tags Workspaces
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkspaceLinkResponse]
// @Router /v1/workspaces/{workspaceId}/links [get]
func (h *Handler) ListWorkspaceLinks(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListWorkspaceLinks(r.Context(), r.PathValue("workspaceId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách liên kết workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Workspace Link
// @Tags Workspaces
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param linkId path string true "Link ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId}/links/{linkId} [delete]
func (h *Handler) DeleteWorkspaceLink(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteWorkspaceLink(r.Context(), r.PathValue("linkId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá liên kết workspace: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Create Workspace Note
// @Tags Workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateWorkspaceNoteRequest true "Note data"
// @Success 201 {object} WorkspaceNoteResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/notes [post]
func (h *Handler) CreateWorkspaceNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateWorkspaceNoteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.Content = strings.TrimSpace(req.Content)
	if req.Title == "" {
		httpx.Error(w, http.StatusBadRequest, "Tiêu đề là bắt buộc")
		return
	}
	resp, err := h.svc.CreateWorkspaceNote(r.Context(), r.PathValue("workspaceId"), userID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo ghi chú workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Workspace Notes
// @Tags Workspaces
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkspaceNoteResponse]
// @Router /v1/workspaces/{workspaceId}/notes [get]
func (h *Handler) ListWorkspaceNotes(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListWorkspaceNotes(r.Context(), r.PathValue("workspaceId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách ghi chú workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update Workspace Note
// @Tags Workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param noteId path string true "Note ID"
// @Param request body UpdateWorkspaceNoteRequest true "Note update"
// @Success 200 {object} WorkspaceNoteResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/notes/{noteId} [patch]
func (h *Handler) UpdateWorkspaceNote(w http.ResponseWriter, r *http.Request) {
	var req UpdateWorkspaceNoteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.svc.UpdateWorkspaceNote(r.Context(), r.PathValue("noteId"), req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật ghi chú workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Workspace Note
// @Tags Workspaces
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param noteId path string true "Note ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId}/notes/{noteId} [delete]
func (h *Handler) DeleteWorkspaceNote(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteWorkspaceNote(r.Context(), r.PathValue("noteId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá ghi chú workspace: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Create Workspace File
// @Tags Workspaces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateWorkspaceFileRequest true "File data"
// @Success 201 {object} WorkspaceFileResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/files [post]
func (h *Handler) CreateWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateWorkspaceFileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	req.FileURL = strings.TrimSpace(req.FileURL)
	req.FileName = strings.TrimSpace(req.FileName)
	req.ContentType = strings.TrimSpace(req.ContentType)
	if req.Title == "" || req.FileURL == "" || req.FileName == "" {
		httpx.Error(w, http.StatusBadRequest, "Tiêu đề, fileUrl và fileName là bắt buộc")
		return
	}
	if req.SizeBytes < 0 {
		httpx.Error(w, http.StatusBadRequest, "sizeBytes phải lớn hơn hoặc bằng 0")
		return
	}
	resp, err := h.svc.CreateWorkspaceFile(r.Context(), r.PathValue("workspaceId"), userID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo tệp workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Workspace Files
// @Tags Workspaces
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[WorkspaceFileResponse]
// @Router /v1/workspaces/{workspaceId}/files [get]
func (h *Handler) ListWorkspaceFiles(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListWorkspaceFiles(r.Context(), r.PathValue("workspaceId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách tệp workspace: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Workspace File
// @Tags Workspaces
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param fileId path string true "File ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId}/files/{fileId} [delete]
func (h *Handler) DeleteWorkspaceFile(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteWorkspaceFile(r.Context(), r.PathValue("fileId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá tệp workspace: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
