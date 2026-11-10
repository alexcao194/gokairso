package files

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
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/file-manager/tree", authMid.Handle(h.GetTree))
	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/file-manager/settings", authMid.Handle(h.GetSettings))
	mux.HandleFunc("PUT /v1/workspaces/{workspaceId}/file-manager/settings", authMid.Handle(h.UpsertSettings))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/file-manager/folders", authMid.Handle(h.CreateFolder))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/file-manager/folders/{folderId}", authMid.Handle(h.UpdateFolder))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/file-manager/folders/{folderId}", authMid.Handle(h.DeleteFolder))

	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/file-manager/presign", authMid.Handle(h.Presign))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/file-manager", authMid.Handle(h.CreateFile))
	mux.HandleFunc("PATCH /v1/workspaces/{workspaceId}/file-manager/{fileId}", authMid.Handle(h.UpdateFile))
	mux.HandleFunc("DELETE /v1/workspaces/{workspaceId}/file-manager/{fileId}", authMid.Handle(h.DeleteFile))

	mux.HandleFunc("GET /v1/workspaces/{workspaceId}/file-manager/{fileId}/versions", authMid.Handle(h.ListVersions))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/file-manager/{fileId}/versions", authMid.Handle(h.AddVersion))
	mux.HandleFunc("POST /v1/workspaces/{workspaceId}/file-manager/{fileId}/versions/{versionId}/restore", authMid.Handle(h.RestoreVersion))
}

func (h *Handler) RegisterPublicRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /public/projects/{token}/files", h.PublicTree)
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
	case errors.Is(err, ErrLocked):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNoStorage):
		httpx.Error(w, http.StatusInternalServerError, err.Error())
	default:
		var ve ValidationError
		if errors.As(err, &ve) {
			httpx.Error(w, http.StatusBadRequest, ve.Error())
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "Lỗi máy chủ: "+err.Error())
	}
}

// @Summary Get workspace file tree
// @Tags Files
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} TreeResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/tree [get]
func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.GetTree(r.Context(), r.PathValue("workspaceId"), callerID)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Get workspace file settings
// @Tags Files
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Success 200 {object} FileSettingsResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/settings [get]
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

// @Summary Update workspace file settings
// @Tags Files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param request body UpsertFileSettingsRequest true "Settings"
// @Success 200 {object} FileSettingsResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/settings [put]
func (h *Handler) UpsertSettings(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req UpsertFileSettingsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return
	}
	resp, err := h.svc.UpsertSettings(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create a custom folder
// @Tags Files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateFolderRequest true "Folder"
// @Success 201 {object} FolderResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/folders [post]
func (h *Handler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req CreateFolderRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return
	}
	resp, err := h.svc.CreateFolder(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary Update a folder (rename/move custom; toggle public any)
// @Tags Files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param folderId path string true "Folder ID"
// @Param request body UpdateFolderRequest true "Changes"
// @Success 200 {object} FolderResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/folders/{folderId} [patch]
func (h *Handler) UpdateFolder(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req UpdateFolderRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return
	}
	resp, err := h.svc.UpdateFolder(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("folderId"), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete a custom folder (recursive)
// @Tags Files
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param folderId path string true "Folder ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId}/file-manager/folders/{folderId} [delete]
func (h *Handler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteFolder(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("folderId")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Presign an upload URL (validated against workspace file settings)
// @Tags Files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param request body PresignUploadRequest true "File metadata"
// @Success 200 {object} PresignUploadResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/presign [post]
func (h *Handler) Presign(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req PresignUploadRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return
	}
	resp, err := h.svc.PresignUpload(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create a file record after upload
// @Tags Files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param request body CreateFileRequest true "File"
// @Success 201 {object} FileResponse
// @Router /v1/workspaces/{workspaceId}/file-manager [post]
func (h *Handler) CreateFile(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req CreateFileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return
	}
	resp, err := h.svc.CreateFile(r.Context(), r.PathValue("workspaceId"), callerID, req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary Update a file (rename/move/public)
// @Tags Files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param fileId path string true "File ID"
// @Param request body UpdateFileRequest true "Changes"
// @Success 200 {object} FileResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/{fileId} [patch]
func (h *Handler) UpdateFile(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req UpdateFileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return
	}
	resp, err := h.svc.UpdateFile(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("fileId"), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete a file
// @Tags Files
// @Security BearerAuth
// @Param workspaceId path string true "Workspace ID"
// @Param fileId path string true "File ID"
// @Success 204
// @Router /v1/workspaces/{workspaceId}/file-manager/{fileId} [delete]
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteFile(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("fileId")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary List file versions
// @Tags Files
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param fileId path string true "File ID"
// @Success 200 {array} FileVersionResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/{fileId}/versions [get]
func (h *Handler) ListVersions(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.ListVersions(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("fileId"))
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Add a new version to a file
// @Tags Files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param fileId path string true "File ID"
// @Param request body CreateVersionRequest true "New version"
// @Success 200 {object} FileResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/{fileId}/versions [post]
func (h *Handler) AddVersion(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	var req CreateVersionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "Nội dung yêu cầu không hợp lệ")
		return
	}
	resp, err := h.svc.AddVersion(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("fileId"), req)
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Restore a previous file version
// @Tags Files
// @Security BearerAuth
// @Produce json
// @Param workspaceId path string true "Workspace ID"
// @Param fileId path string true "File ID"
// @Param versionId path string true "Version ID"
// @Success 200 {object} FileResponse
// @Router /v1/workspaces/{workspaceId}/file-manager/{fileId}/versions/{versionId}/restore [post]
func (h *Handler) RestoreVersion(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.caller(w, r)
	if !ok {
		return
	}
	resp, err := h.svc.RestoreVersion(r.Context(), r.PathValue("workspaceId"), callerID, r.PathValue("fileId"), r.PathValue("versionId"))
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Public file tree for a project (token-gated)
// @Tags Files
// @Produce json
// @Param token path string true "Public feedback token"
// @Success 200 {object} TreeResponse
// @Router /public/projects/{token}/files [get]
func (h *Handler) PublicTree(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.GetPublicTree(r.Context(), r.PathValue("token"))
	if err != nil {
		writeErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
