package rest

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/api/rest/middleware"
	"github.com/alexcao194/kairso/pkg/httpx"
	storage "github.com/alexcao194/kairso/pkg/minio"
)

type uploadPresigner interface {
	PresignPutObject(ctx context.Context, input storage.PresignPutInput) (storage.PresignPutOutput, error)
}

type UploadHandler struct {
	client uploadPresigner
}

type CreatePresignedUploadURLRequest struct {
	Kind        string `json:"kind"`
	Owner       string `json:"owner,omitempty"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
}

type PresignedUploadURLResponse struct {
	Method     string    `json:"method"`
	UploadURL  string    `json:"uploadUrl"`
	FileURL    string    `json:"fileUrl"`
	ObjectName string    `json:"objectName"`
	ExpiresAt  time.Time `json:"expiresAt" swaggertype:"string" format:"date-time"`
}

func NewUploadHandler(client uploadPresigner) *UploadHandler {
	return &UploadHandler{client: client}
}

func (h *UploadHandler) RegisterRoutes(mux *http.ServeMux, authMid *middleware.AuthMiddleware) {
	mux.HandleFunc("POST /v1/uploads/presigned/url", authMid.Handle(h.CreatePresignedURL))
}

// @Summary Create Presigned Upload URL
// @Description Create a presigned MinIO PUT URL. Client uploads the file directly to MinIO, then sends fileUrl to the target update endpoint.
// @Tags Uploads
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreatePresignedUploadURLRequest true "Upload metadata"
// @Success 200 {object} PresignedUploadURLResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /v1/uploads/presigned/url [post]
func (h *UploadHandler) CreatePresignedURL(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		httpx.Error(w, http.StatusInternalServerError, "Máy chủ chưa cấu hình kho lưu trữ tệp")
		return
	}
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "Chưa xác thực hoặc phiên đăng nhập đã hết hạn")
		return
	}
	var req CreatePresignedUploadURLRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, fmt.Sprintf("Nội dung yêu cầu không hợp lệ: %s", err.Error()))
		return
	}
	owner, err := normalizePresignRequest(&req, userID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	objectName := storage.AvatarObjectName(req.Kind, owner, req.Filename, req.ContentType)
	if resourceKind, ok := fileUploadResourceKind(req.Kind); ok {
		objectName = storage.FileObjectName(resourceKind, owner, req.Filename)
	}
	out, err := h.client.PresignPutObject(r.Context(), storage.PresignPutInput{
		ObjectName:  objectName,
		ContentType: req.ContentType,
		Expires:     15 * time.Minute,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, fmt.Sprintf("Không thể tạo đường dẫn tải lên: %s", err.Error()))
		return
	}
	httpx.JSON(w, http.StatusOK, PresignedUploadURLResponse(out))
}

func normalizePresignRequest(req *CreatePresignedUploadURLRequest, userID string) (string, error) {
	req.Kind = strings.TrimSpace(req.Kind)
	req.Owner = strings.TrimSpace(req.Owner)
	req.Filename = strings.TrimSpace(req.Filename)
	req.ContentType = strings.TrimSpace(req.ContentType)
	if req.Filename == "" {
		return "", fmt.Errorf("filename là bắt buộc")
	}
	if _, ok := fileUploadResourceKind(req.Kind); ok {
		if req.Owner == "" {
			return "", fmt.Errorf("owner là bắt buộc khi tải lên với kind workspace_file, project_file, task_file, task_comment_file, module_file, workflow_task_file và workflow_instance_file")
		}
		if req.ContentType == "" {
			req.ContentType = "application/octet-stream"
		}
		return req.Owner, nil
	}
	if !avatarContentTypeAllowed(req.ContentType) {
		return "", fmt.Errorf("contentType phải là image/jpeg, image/png, image/webp hoặc image/gif")
	}
	switch req.Kind {
	case "user":
		return userID, nil
	case "workspace", "project":
		if req.Owner == "" {
			return "", fmt.Errorf("owner là bắt buộc khi tải lên cho workspace và dự án")
		}
		return req.Owner, nil
	default:
		return "", fmt.Errorf("kind phải là một trong: user, workspace, project, workspace_file, project_file, task_file, task_comment_file, module_file, workflow_task_file, workflow_instance_file")
	}
}

func fileUploadResourceKind(kind string) (string, bool) {
	switch kind {
	case "workspace_file":
		return "workspace", true
	case "project_file":
		return "project", true
	case "task_file":
		return "task", true
	case "task_comment_file":
		return "task_comment", true
	case "module_file":
		return "module", true
	case "workflow_task_file":
		return "workflow_task", true
	case "workflow_instance_file":
		return "workflow_instance", true
	default:
		return "", false
	}
}

func avatarContentTypeAllowed(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}
