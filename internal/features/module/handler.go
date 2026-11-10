package module

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

func (h *Handler) authorizeModule(w http.ResponseWriter, r *http.Request, moduleID string) (string, bool) {
	return h.authorize(w, r, func(userID string) error {
		return h.svc.AuthorizeModuleAccess(r.Context(), moduleID, userID)
	})
}

func (h *Handler) authorizeProject(w http.ResponseWriter, r *http.Request, projectID string) (string, bool) {
	return h.authorize(w, r, func(userID string) error {
		return h.svc.AuthorizeProjectAccess(r.Context(), projectID, userID)
	})
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request, check func(userID string) error) (string, bool) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return "", false
	}
	if err := check(userID); err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, err.Error())
		} else {
			httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không kiểm tra được quyền: %s", err.Error()))
		}
		return "", false
	}
	return userID, true
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("POST /v1/projects/{projectId}/modules", authMid.Handle(h.CreateModule))
	mux.HandleFunc("GET /v1/projects/{projectId}/modules", authMid.Handle(h.ListModules))
	mux.HandleFunc("GET /v1/modules/{moduleId}", authMid.Handle(h.GetModule))
	mux.HandleFunc("PATCH /v1/modules/{moduleId}", authMid.Handle(h.UpdateModule))
	mux.HandleFunc("GET /v1/modules/{moduleId}/history", authMid.Handle(h.ListModuleHistory))
	mux.HandleFunc("DELETE /v1/modules/{moduleId}", authMid.Handle(h.DeleteModule))
	mux.HandleFunc("POST /v1/modules/{moduleId}/attachments", authMid.Handle(h.AddAttachment))
	mux.HandleFunc("GET /v1/modules/{moduleId}/attachments", authMid.Handle(h.ListAttachments))
	mux.HandleFunc("DELETE /v1/modules/{moduleId}/attachments/{attachmentId}", authMid.Handle(h.DeleteAttachment))
	mux.HandleFunc("POST /v1/modules/{moduleId}/sprints", authMid.Handle(h.CreateSprint))
	mux.HandleFunc("PATCH /v1/modules/{moduleId}/sprints/{sprintId}", authMid.Handle(h.UpdateSprint))
	mux.HandleFunc("DELETE /v1/modules/{moduleId}/sprints/{sprintId}", authMid.Handle(h.DeleteSprint))
}

func writeSprintError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrInvalidSprint):
		httpx.Error(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrSprintNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	default:
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("%s: %s", fallback, err.Error()))
	}
}

// @Summary Add a sprint to a module
// @Description Without dates the sprint starts the day after the module's last sprint (or today) and lasts 2 weeks. Sprints are at most one month and never overlap.
// @Tags Modules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module ID"
// @Param request body CreateModuleSprintRequest true "Sprint"
// @Success 201 {object} ModuleResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Router /v1/modules/{moduleId}/sprints [post]
func (h *Handler) CreateSprint(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeModule(w, r, r.PathValue("moduleId"))
	if !ok {
		return
	}
	var req CreateModuleSprintRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.svc.CreateSprint(r.Context(), r.PathValue("moduleId"), callerID, req)
	if err != nil {
		writeSprintError(w, err, "Không thể thêm sprint")
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary Update a module sprint
// @Description Change dates, goal, status (planned → active → completed) or the Review/Retrospective notes. Starting a sprint moves a backlog module to in_progress.
// @Tags Modules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module ID"
// @Param sprintId path string true "Sprint ID"
// @Param request body UpdateModuleSprintRequest true "Changes"
// @Success 200 {object} ModuleResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/modules/{moduleId}/sprints/{sprintId} [patch]
func (h *Handler) UpdateSprint(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeModule(w, r, r.PathValue("moduleId"))
	if !ok {
		return
	}
	var req UpdateModuleSprintRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	resp, err := h.svc.UpdateSprint(r.Context(), r.PathValue("moduleId"), r.PathValue("sprintId"), callerID, req)
	if err != nil {
		writeSprintError(w, err, "Không thể cập nhật sprint")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete a module sprint
// @Tags Modules
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module ID"
// @Param sprintId path string true "Sprint ID"
// @Success 200 {object} ModuleResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/modules/{moduleId}/sprints/{sprintId} [delete]
func (h *Handler) DeleteSprint(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeModule(w, r, r.PathValue("moduleId"))
	if !ok {
		return
	}
	resp, err := h.svc.DeleteSprint(r.Context(), r.PathValue("moduleId"), r.PathValue("sprintId"), callerID)
	if err != nil {
		writeSprintError(w, err, "Không thể xoá sprint")
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Create Module
// @Tags Modules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param projectId path string true "Project ID"
// @Param request body CreateModuleRequest true "Module"
// @Success 201 {object} ModuleResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/projects/{projectId}/modules [post]
func (h *Handler) CreateModule(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeProject(w, r, r.PathValue("projectId")); !ok {
		return
	}
	var req CreateModuleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "Tên là bắt buộc")
		return
	}
	resp, err := h.svc.CreateModule(r.Context(), r.PathValue("projectId"), req)
	if err != nil {
		if pgutil.IsAliasConflict(err) {
			httpx.Error(w, http.StatusConflict, "Alias đã tồn tại")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo module: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Modules
// @Tags Modules
// @Produce json
// @Security BearerAuth
// @Param projectId path string true "Project ID"
// @Param request query pagination.StatsParams false "Pagination and month filter"
// @Success 200 {object} pagination.Response[ModuleResponse]
// @Router /v1/projects/{projectId}/modules [get]
func (h *Handler) ListModules(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeProject(w, r, r.PathValue("projectId")); !ok {
		return
	}
	params := pagination.ParseStats(r, 20, 100)
	resp, err := h.svc.ListModules(r.Context(), r.PathValue("projectId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách module: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) GetModule(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeModule(w, r, r.PathValue("moduleId")); !ok {
		return
	}
	resp, err := h.svc.GetModule(r.Context(), r.PathValue("moduleId"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, fmt.Sprintf("Không tìm thấy module: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Update Module
// @Tags Modules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module ID"
// @Param request body UpdateModuleRequest true "Module"
// @Success 200 {object} ModuleResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Router /v1/modules/{moduleId} [patch]
func (h *Handler) UpdateModule(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeModule(w, r, r.PathValue("moduleId"))
	if !ok {
		return
	}
	var req UpdateModuleRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeUpdateModuleRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.UpdateModule(r.Context(), r.PathValue("moduleId"), callerID, req)
	if err != nil {
		if pgutil.IsAliasConflict(err) {
			httpx.Error(w, http.StatusConflict, "Alias đã tồn tại")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể cập nhật module: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List Module Update History
// @Tags Modules
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module ID"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} pagination.Response[ModuleHistoryResponse]
// @Router /v1/modules/{moduleId}/history [get]
func (h *Handler) ListModuleHistory(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeModule(w, r, r.PathValue("moduleId")); !ok {
		return
	}
	params := pagination.Parse(r, 20, 100)
	resp, err := h.svc.ListModuleHistory(r.Context(), r.PathValue("moduleId"), params)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách lịch sử module: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Add Module Attachment
// @Tags Modules
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module ID"
// @Param request body CreateModuleAttachmentRequest true "File reference (upload it first via POST /v1/uploads/presigned/url)"
// @Success 201 {object} ModuleAttachmentResponse
// @Router /v1/modules/{moduleId}/attachments [post]
func (h *Handler) AddAttachment(w http.ResponseWriter, r *http.Request) {
	callerID, ok := h.authorizeModule(w, r, r.PathValue("moduleId"))
	if !ok {
		return
	}
	var req CreateModuleAttachmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.FileName = strings.TrimSpace(req.FileName)
	req.FileURL = strings.TrimSpace(req.FileURL)
	if req.FileName == "" || req.FileURL == "" {
		httpx.Error(w, http.StatusBadRequest, "fileName và fileUrl là bắt buộc")
		return
	}
	resp, err := h.svc.AddAttachment(r.Context(), r.PathValue("moduleId"), callerID, req)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể thêm tệp đính kèm: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

// @Summary List Module Attachments
// @Tags Modules
// @Produce json
// @Security BearerAuth
// @Param moduleId path string true "Module ID"
// @Success 200 {array} ModuleAttachmentResponse
// @Router /v1/modules/{moduleId}/attachments [get]
func (h *Handler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeModule(w, r, r.PathValue("moduleId")); !ok {
		return
	}
	resp, err := h.svc.ListAttachments(r.Context(), r.PathValue("moduleId"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể lấy danh sách tệp đính kèm: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Delete Module Attachment
// @Tags Modules
// @Security BearerAuth
// @Param moduleId path string true "Module ID"
// @Param attachmentId path string true "Attachment ID"
// @Success 204
// @Router /v1/modules/{moduleId}/attachments/{attachmentId} [delete]
func (h *Handler) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeModule(w, r, r.PathValue("moduleId")); !ok {
		return
	}
	if err := h.svc.DeleteAttachment(r.Context(), r.PathValue("attachmentId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá tệp đính kèm: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Delete Module
// @Tags Modules
// @Security BearerAuth
// @Param moduleId path string true "Module ID"
// @Success 204
// @Failure 404 {object} httpx.ErrorResponse
// @Router /v1/modules/{moduleId} [delete]
func (h *Handler) DeleteModule(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.authorizeModule(w, r, r.PathValue("moduleId")); !ok {
		return
	}
	if err := h.svc.DeleteModule(r.Context(), r.PathValue("moduleId")); err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể xoá module: %s", err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
