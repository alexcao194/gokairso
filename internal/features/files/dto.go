package files

import (
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	FolderKindProject = "project"
	FolderKindModule  = "module"
	FolderKindCustom  = "custom"
)

type FolderResponse struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parentId,omitempty"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Locked    bool      `json:"locked"`
	ProjectID *string   `json:"projectId,omitempty"`
	ModuleID  *string   `json:"moduleId,omitempty"`
	IsPublic  bool      `json:"isPublic"`
	CreatedAt time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
}

type FileResponse struct {
	ID          string    `json:"id"`
	FolderID    *string   `json:"folderId,omitempty"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	IsPublic    bool      `json:"isPublic"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
}

type TreeNode struct {
	FolderResponse
	Children []*TreeNode    `json:"children"`
	Files    []FileResponse `json:"files"`
}

type TreeResponse struct {
	Folders   []*TreeNode    `json:"folders"`
	RootFiles []FileResponse `json:"rootFiles"`
}

type FileSettingsResponse struct {
	MaxFileSizeMB int    `json:"maxFileSizeMb"`
	AllowedTypes  string `json:"allowedTypes"`
}

type FileVersionResponse struct {
	ID          string    `json:"id"`
	ContentType string    `json:"contentType"`
	SizeBytes   int64     `json:"sizeBytes"`
	URL         string    `json:"url"`
	CreatedAt   time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
}

type PresignUploadResponse struct {
	UploadURL  string    `json:"uploadUrl"`
	Method     string    `json:"method"`
	ObjectName string    `json:"objectName"`
	FileURL    string    `json:"fileUrl"`
	ExpiresAt  time.Time `json:"expiresAt" swaggertype:"string" format:"date-time"`
}

type CreateFolderRequest struct {
	ParentID string `json:"parentId"`
	Name     string `json:"name"`
	IsPublic bool   `json:"isPublic"`
}

type UpdateFolderRequest struct {
	Name     *string `json:"name"`
	ParentID *string `json:"parentId"`
	IsPublic *bool   `json:"isPublic"`
}

type PresignUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
}

type CreateFileRequest struct {
	FolderID    string `json:"folderId"`
	Name        string `json:"name"`
	ObjectName  string `json:"objectName"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	IsPublic    bool   `json:"isPublic"`
}

type UpdateFileRequest struct {
	Name     *string `json:"name"`
	FolderID *string `json:"folderId"`
	IsPublic *bool   `json:"isPublic"`
}

type CreateVersionRequest struct {
	ObjectName  string `json:"objectName"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
}

type UpsertFileSettingsRequest struct {
	MaxFileSizeMB int    `json:"maxFileSizeMb"`
	AllowedTypes  string `json:"allowedTypes"`
}

func uuidPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := pgutil.UUIDToString(u)
	return &s
}

func folderResponse(f database.FileFolder) FolderResponse {
	return FolderResponse{
		ID:        pgutil.UUIDToString(f.ID),
		ParentID:  uuidPtr(f.ParentID),
		Name:      f.Name,
		Kind:      f.Kind,
		Locked:    f.Kind != FolderKindCustom,
		ProjectID: uuidPtr(f.ProjectID),
		ModuleID:  uuidPtr(f.ModuleID),
		IsPublic:  f.IsPublic,
		CreatedAt: f.CreatedAt.Time,
	}
}

func fileResponse(f database.File, url string) FileResponse {
	return FileResponse{
		ID:          pgutil.UUIDToString(f.ID),
		FolderID:    uuidPtr(f.FolderID),
		Name:        f.Name,
		ContentType: f.ContentType,
		SizeBytes:   f.SizeBytes,
		IsPublic:    f.IsPublic,
		URL:         url,
		CreatedAt:   f.CreatedAt.Time,
		UpdatedAt:   f.UpdatedAt.Time,
	}
}

func fileSettingsResponse(s database.WorkspaceFileSetting) FileSettingsResponse {
	return FileSettingsResponse{
		MaxFileSizeMB: int(s.MaxFileSizeMb),
		AllowedTypes:  s.AllowedTypes,
	}
}

func fileVersionResponse(v database.FileVersion, url string) FileVersionResponse {
	return FileVersionResponse{
		ID:          pgutil.UUIDToString(v.ID),
		ContentType: v.ContentType,
		SizeBytes:   v.SizeBytes,
		URL:         url,
		CreatedAt:   v.CreatedAt.Time,
	}
}
