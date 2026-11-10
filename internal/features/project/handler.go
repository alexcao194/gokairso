package project

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
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects", authMid.Handle(h.CreateProject))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects", authMid.Handle(h.ListProjects))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/overviews", authMid.Handle(h.GetProjectOverviewsByIDs))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/projects/{projectId}", authMid.Handle(h.UpdateProject))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/history", authMid.Handle(h.ListProjectHistory))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/projects/{projectId}", authMid.Handle(h.DeleteProject))
	mux.HandleFunc("PATCH /v1/projects/{projectId}", authMid.Handle(h.UpdateProject))
	mux.HandleFunc("GET /v1/projects/{projectId}/history", authMid.Handle(h.ListProjectHistory))
	mux.HandleFunc("DELETE /v1/projects/{projectId}", authMid.Handle(h.DeleteProject))

	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/members", authMid.Handle(h.ListProjectMembers))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/members", authMid.Handle(h.AddProjectMember))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/projects/{projectId}/members/{userId}", authMid.Handle(h.UpdateProjectMemberRole))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/projects/{projectId}/members/{userId}", authMid.Handle(h.RemoveProjectMember))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/links", authMid.Handle(h.CreateProjectLink))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/links", authMid.Handle(h.ListProjectLinks))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/projects/{projectId}/links/{linkId}", authMid.Handle(h.UpdateProjectLink))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/projects/{projectId}/links/{linkId}", authMid.Handle(h.DeleteProjectLink))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/notes", authMid.Handle(h.CreateProjectNote))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/notes", authMid.Handle(h.ListProjectNotes))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/projects/{projectId}/notes/{noteId}", authMid.Handle(h.UpdateProjectNote))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/projects/{projectId}/notes/{noteId}", authMid.Handle(h.DeleteProjectNote))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/projects/{projectId}/files", authMid.Handle(h.CreateProjectFile))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/projects/{projectId}/files", authMid.Handle(h.ListProjectFiles))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/projects/{projectId}/files/{fileId}", authMid.Handle(h.DeleteProjectFile))
}

// @Summary Create Project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateProjectRequest true "Project"
// @Success 201 {object} ProjectResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects [post]
func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateProjectRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "Tên là bắt buộc")
		return
	}
	resp, err := h.svc.CreateProject(r.Context(), r.PathValue("workspaceId"), userID, req)
	if err != nil {
		if pgutil.IsAliasConflict(err) {
			httpx.Error(w, http.StatusConflict, "Alias đã tồn tại")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Projects
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param request query pagination.StatsParams false "Pagination and month filter"
// @Success 200 {object} pagination.Response[ProjectResponse]
// @Router /v1/workspaces/{workspaceId}/projects [get]
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	params := pagination.ParseStats(r, 20, 100)
	resp, err := h.svc.ListProjects(r.Context(), r.PathValue("workspaceId"), userID, params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get Project Overviews Across Projects
// @Description Every project overview across the given projects in one call — for a caller that would otherwise call Get Project Overview once per project (e.g. a workspace dashboard).
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectIds query string true "Comma-separated project IDs"
// @Success 200 {object} ProjectOverviewsByIDsResponse
// @Router /v1/workspaces/{workspaceId}/projects/overviews [get]
func (h *Handler) GetProjectOverviewsByIDs(w http.ResponseWriter, r *http.Request) {
	projectIDs := httpx.ParseCSVParam(r, "projectIds")
	items, err := h.svc.GetProjectOverviewsByIDs(r.Context(), projectIDs)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy tổng quan dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, ProjectOverviewsByIDsResponse{Items: items})
}

// @Summary Update Project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request body UpdateProjectRequest true "Update fields"
// @Success 200 {object} ProjectResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId} [patch]
func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateProjectRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeUpdateProjectRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.UpdateProject(r.Context(), r.PathValue("projectId"), callerID, req)
	if err != nil {
		if pgutil.IsAliasConflict(err) {
			httpx.Error(w, http.StatusConflict, "Alias đã tồn tại")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List Project Update History
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[ProjectHistoryResponse]
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/history [get]
func (h *Handler) ListProjectHistory(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListProjectHistory(r.Context(), r.PathValue("projectId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách lịch sử dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Project
// @Tags Projects
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId}/projects/{projectId} [delete]
func (h *Handler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteProject(r.Context(), r.PathValue("projectId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá dự án: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary List Project Members
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request query pagination.Params false "Pagination"
// @Param includeAvailable query bool false "Also compute workspace members not yet on this project (for the invite picker) — costs 2 extra queries, so it's opt-in"
// @Success 200 {object} ListProjectMembersResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/members [get]
func (h *Handler) ListProjectMembers(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	includeAvailable := r.URL.Query().Get("includeAvailable") == "true"
	resp, err := h.svc.ListProjectMembers(r.Context(), r.PathValue("projectId"), params, includeAvailable)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách thành viên dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Add Member to Project
// @Description Add a user (by ID) to the project. Caller must be workspace holder or admin.
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request body AddProjectMemberRequest true "Member"
// @Success 201 {object} ProjectMemberResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/members [post]
func (h *Handler) AddProjectMember(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req AddProjectMemberRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.UserID = strings.TrimSpace(req.UserID)
	if req.UserID == "" {
		httpx.Error(w, http.StatusBadRequest, "userId là bắt buộc")
		return
	}
	if !req.Role.Valid() {
		httpx.Error(w, http.StatusBadRequest, "Vai trò phải là một trong: holder, admin, member")
		return
	}
	resp, err := h.svc.AddProjectMember(r.Context(), r.PathValue("projectId"), callerID, req.UserID, req.Role)
	if err != nil {
		if errors.Is(err, ErrCannotManageEqualOrHigherRole) {
			httpx.Error(w, http.StatusForbidden, "Bạn chỉ được thêm thành viên với vai trò thấp hơn hoặc bằng mình")
			return
		}
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "Chỉ chủ workspace hoặc quản trị viên mới được quản lý thành viên dự án")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể thêm thành viên dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary Update Project Member Role
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param userId path string true "User ID"
// @Param request body UpdateProjectMemberRoleRequest true "Role"
// @Success 200 {object} ProjectMemberResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/members/{userId} [patch]
func (h *Handler) UpdateProjectMemberRole(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req UpdateProjectMemberRoleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if !req.Role.Valid() {
		httpx.Error(w, http.StatusBadRequest, "Vai trò phải là một trong: holder, admin, member")
		return
	}
	resp, err := h.svc.UpdateProjectMemberRole(r.Context(), r.PathValue("projectId"), callerID, r.PathValue("userId"), req.Role)
	if err != nil {
		if errors.Is(err, ErrCannotManageEqualOrHigherRole) {
			httpx.Error(w, http.StatusForbidden, "Bạn chỉ được đổi vai trò của người có cấp thấp hơn hoặc bằng mình")
			return
		}
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "Chỉ chủ workspace hoặc quản trị viên mới được đổi vai trò thành viên")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật vai trò thành viên dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Remove Member from Project
// @Tags Projects
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param userId path string true "User ID"
// @Success 204
// @Failure 403 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/members/{userId} [delete]
func (h *Handler) RemoveProjectMember(w http.ResponseWriter, r *http.Request) {
	callerID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	if err := h.svc.RemoveProjectMember(r.Context(), r.PathValue("projectId"), callerID, r.PathValue("userId")); err != nil {
		if errors.Is(err, ErrCannotManageEqualOrHigherRole) {
			httpx.Error(w, http.StatusForbidden, "Bạn chỉ được gỡ người có vai trò thấp hơn hoặc bằng mình")
			return
		}
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "Chỉ chủ workspace hoặc quản trị viên mới được xoá thành viên khỏi dự án")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể gỡ thành viên dự án: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Create Project Link
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request body CreateProjectLinkRequest true "Link"
// @Success 201 {object} ProjectLinkResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/links [post]
func (h *Handler) CreateProjectLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateProjectLinkRequest
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
	resp, err := h.svc.CreateProjectLink(r.Context(), r.PathValue("projectId"), userID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo liên kết dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Project Links
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[ProjectLinkResponse]
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/links [get]
func (h *Handler) ListProjectLinks(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListProjectLinks(r.Context(), r.PathValue("projectId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách liên kết dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update Project Link
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param linkId path string true "Link ID"
// @Param request body UpdateProjectLinkRequest true "Link update"
// @Success 200 {object} ProjectLinkResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/links/{linkId} [patch]
func (h *Handler) UpdateProjectLink(w http.ResponseWriter, r *http.Request) {
	var req UpdateProjectLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.svc.UpdateProjectLink(r.Context(), r.PathValue("linkId"), req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật liên kết dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Project Link
// @Tags Projects
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param linkId path string true "Link ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/links/{linkId} [delete]
func (h *Handler) DeleteProjectLink(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteProjectLink(r.Context(), r.PathValue("linkId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá liên kết dự án: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Create Project Note
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request body CreateProjectNoteRequest true "Note"
// @Success 201 {object} ProjectNoteResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/notes [post]
func (h *Handler) CreateProjectNote(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateProjectNoteRequest
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
	resp, err := h.svc.CreateProjectNote(r.Context(), r.PathValue("projectId"), userID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo ghi chú dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Project Notes
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[ProjectNoteResponse]
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/notes [get]
func (h *Handler) ListProjectNotes(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListProjectNotes(r.Context(), r.PathValue("projectId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách ghi chú dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update Project Note
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param noteId path string true "Note ID"
// @Param request body UpdateProjectNoteRequest true "Note update"
// @Success 200 {object} ProjectNoteResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/notes/{noteId} [patch]
func (h *Handler) UpdateProjectNote(w http.ResponseWriter, r *http.Request) {
	var req UpdateProjectNoteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.svc.UpdateProjectNote(r.Context(), r.PathValue("noteId"), req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật ghi chú dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Project Note
// @Tags Projects
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param noteId path string true "Note ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/notes/{noteId} [delete]
func (h *Handler) DeleteProjectNote(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteProjectNote(r.Context(), r.PathValue("noteId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá ghi chú dự án: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Create Project File
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request body CreateProjectFileRequest true "File"
// @Success 201 {object} ProjectFileResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/files [post]
func (h *Handler) CreateProjectFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreateProjectFileRequest
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
	resp, err := h.svc.CreateProjectFile(r.Context(), r.PathValue("projectId"), userID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo tệp dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Project Files
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[ProjectFileResponse]
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/files [get]
func (h *Handler) ListProjectFiles(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListProjectFiles(r.Context(), r.PathValue("projectId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách tệp dự án: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Project File
// @Tags Projects
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param projectId path string true "Project ID"
// @Param fileId path string true "File ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId}/projects/{projectId}/files/{fileId} [delete]
func (h *Handler) DeleteProjectFile(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteProjectFile(r.Context(), r.PathValue("fileId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá tệp dự án: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
