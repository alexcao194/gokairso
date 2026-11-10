package projectfeedback

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexcao194/kairso/pkg/httpx"
	storage "github.com/alexcao194/kairso/pkg/minio"
	"github.com/alexcao194/kairso/pkg/pagination"
)

type uploadPresigner interface {
	PresignPutObject(ctx context.Context, input storage.PresignPutInput) (storage.PresignPutOutput, error)
}

type PublicHandler struct {
	svc     Service
	uploads uploadPresigner
}

func NewPublicHandler(svc Service, uploads uploadPresigner) *PublicHandler {
	return &PublicHandler{svc: svc, uploads: uploads}
}

func (h *PublicHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /public/projects/{token}", h.GetProject)
	mux.HandleFunc("GET /public/projects/{token}/feedbacks", h.ListFeedbacks)
	mux.HandleFunc("POST /public/projects/{token}/feedbacks", h.CreateFeedback)
	mux.HandleFunc("POST /public/projects/{token}/uploads/presigned-url", h.CreatePresignedUploadURL)
	mux.HandleFunc("GET /public/projects/{token}/modules", h.ListModules)
}

func (h *PublicHandler) handleError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrDisabled) {
		httpx.Error(w, http.StatusNotFound, "Không tìm thấy dữ liệu")
		return
	}
	httpx.Error(w, http.StatusNotFound, "Không tìm thấy dữ liệu")
}

// @Summary Get Public Project Info
// @Description No authentication required — gated only by the unguessable token.
// @Tags PublicFeedback
// @Produce json
// @Param token path string true "Public feedback token"
// @Success 200 {object} PublicProjectResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /public/projects/{token} [get]
func (h *PublicHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.GetPublicProject(r.Context(), r.PathValue("token"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary List Public Feedback Board
// @Description No authentication required — gated only by the unguessable token.
// @Tags PublicFeedback
// @Produce json
// @Param token path string true "Public feedback token"
// @Param priority query string false "Comma-separated list of priorities to filter by (1-4), e.g. '2,3'"
// @Param hasResponse query bool false "Filter by whether a Response has been set"
// @Param status query string false "Comma-separated list of triage statuses to filter by: pending, discussing, acknowledged, in_progress, resolved, deployed, canceled"
// @Param feedbackType query string false "Comma-separated list of types to filter by: feature, improvement, bug"
// @Param q query string false "Search submitter name/company, feature name, content, response, and code"
// @Param sort query string false "Comma-separated sort fields, each optionally suffixed with :asc or :desc (default asc), e.g. 'priority:desc,submitted:asc'. Defaults to submitted desc when omitted. Valid fields: code, priority, feature, submitter, company, status, expectedFix, submitted"
// @Param request query pagination.Params false "Pagination"
// @Success 200 {object} ProjectFeedbackListResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /public/projects/{token}/feedbacks [get]
func (h *PublicHandler) ListFeedbacks(w http.ResponseWriter, r *http.Request) {
	params := pagination.Parse(r, 20, 100)
	filter, err := parseListFeedbacksFilter(r)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.ListPublicFeedbacks(r.Context(), r.PathValue("token"), filter, params)
	if err != nil {
		h.handleError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

// @Summary Submit Public Feedback
// @Description No authentication required — gated only by the unguessable token.
// @Tags PublicFeedback
// @Accept json
// @Produce json
// @Param token path string true "Public feedback token"
// @Param request body CreateProjectFeedbackRequest true "Feedback"
// @Success 201 {object} ProjectFeedbackResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /public/projects/{token}/feedbacks [post]
func (h *PublicHandler) CreateFeedback(w http.ResponseWriter, r *http.Request) {
	var req CreateProjectFeedbackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	if err := normalizeCreateFeedbackRequest(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.svc.SubmitPublicFeedback(r.Context(), r.PathValue("token"), req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

const feedbackImageContentTypeAllowed = "image/jpeg,image/png,image/webp,image/gif"

// @Summary Create Presigned Upload URL For A Feedback Photo
// @Description No authentication required — gated only by the unguessable token. Client PUTs the file directly to object storage using the returned URL.
// @Tags PublicFeedback
// @Accept json
// @Produce json
// @Param token path string true "Public feedback token"
// @Param request body PublicUploadRequest true "Upload metadata"
// @Success 200 {object} PublicUploadResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /public/projects/{token}/uploads/presigned-url [post]
func (h *PublicHandler) CreatePresignedUploadURL(w http.ResponseWriter, r *http.Request) {
	if h.uploads == nil {
		httpx.Error(w, http.StatusInternalServerError, "Máy chủ chưa cấu hình kho lưu trữ tệp")
		return
	}
	token := r.PathValue("token")
	project, err := h.svc.GetPublicProject(r.Context(), token)
	if err != nil {
		h.handleError(w, err)
		return
	}
	if !project.Enabled {
		h.handleError(w, ErrDisabled)
		return
	}
	var req PublicUploadRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	req.Filename = strings.TrimSpace(req.Filename)
	req.ContentType = strings.TrimSpace(req.ContentType)
	if req.Filename == "" {
		httpx.Error(w, http.StatusBadRequest, "filename là bắt buộc")
		return
	}
	if !strings.Contains(feedbackImageContentTypeAllowed, req.ContentType) || req.ContentType == "" {
		httpx.Error(w, http.StatusBadRequest, "contentType phải là image/jpeg, image/png, image/webp hoặc image/gif")
		return
	}
	objectName := storage.AvatarObjectName("feedback_attachment", token, req.Filename, req.ContentType)
	out, err := h.uploads.PresignPutObject(r.Context(), storage.PresignPutInput{
		ObjectName:  objectName,
		ContentType: req.ContentType,
		Expires:     10 * time.Minute,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo đường dẫn tải lên: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, PublicUploadResponse{
		Method:    out.Method,
		UploadURL: out.UploadURL,
		FileURL:   out.FileURL,
		ExpiresAt: out.ExpiresAt,
	})
}

type PublicUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
}

type PublicUploadResponse struct {
	Method    string    `json:"method"`
	UploadURL string    `json:"uploadUrl"`
	FileURL   string    `json:"fileUrl"`
	ExpiresAt time.Time `json:"expiresAt" swaggertype:"string" format:"date-time"`
}

// @Summary List Public Modules
// @Description No authentication required. Returns modules if project timeline is enabled.
// @Tags PublicFeedback
// @Produce json
// @Param token path string true "Public feedback token"
// @Success 200 {array} PublicModuleResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /public/projects/{token}/modules [get]
func (h *PublicHandler) ListModules(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListPublicModules(r.Context(), r.PathValue("token"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}
