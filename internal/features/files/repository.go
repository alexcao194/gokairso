package files

import (
	"context"
	"errors"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/workspace"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateFolderParams struct {
	WorkspaceID string
	ParentID    *string
	Name        string
	Kind        string
	ProjectID   *string
	ModuleID    *string
	IsPublic    bool
	CreatedBy   *string
}

type CreateFileParams struct {
	WorkspaceID string
	FolderID    *string
	Name        string
	ObjectName  string
	ContentType string
	SizeBytes   int64
	IsPublic    bool
	CreatedBy   *string
}

type CreateVersionParams struct {
	FileID      string
	ObjectName  string
	ContentType string
	SizeBytes   int64
	CreatedBy   *string
}

type Repository interface {
	GetCallerRole(ctx context.Context, workspaceID, callerID string) (workspace.WorkspaceRole, error)

	GetSettings(ctx context.Context, workspaceID string) (database.WorkspaceFileSetting, bool, error)
	UpsertSettings(ctx context.Context, workspaceID string, maxMB int, allowedTypes string) (database.WorkspaceFileSetting, error)

	ListFolders(ctx context.Context, workspaceID string) ([]database.FileFolder, error)
	GetFolder(ctx context.Context, folderID string) (database.FileFolder, error)
	CreateFolder(ctx context.Context, p CreateFolderParams) (database.FileFolder, error)
	RenameFolder(ctx context.Context, folderID, name string) (database.FileFolder, error)
	MoveFolder(ctx context.Context, folderID string, parentID *string) (database.FileFolder, error)
	SetFolderPublic(ctx context.Context, folderID string, public bool) (database.FileFolder, error)
	SoftDeleteFolder(ctx context.Context, folderID string) error

	ListFiles(ctx context.Context, workspaceID string) ([]database.File, error)
	GetFile(ctx context.Context, fileID string) (database.File, error)
	CreateFile(ctx context.Context, p CreateFileParams) (database.File, error)
	RenameFile(ctx context.Context, fileID, name string) (database.File, error)
	MoveFile(ctx context.Context, fileID string, folderID *string) (database.File, error)
	SetFilePublic(ctx context.Context, fileID string, public bool) (database.File, error)
	SetFileObject(ctx context.Context, fileID, objectName, contentType string, size int64) (database.File, error)
	SoftDeleteFile(ctx context.Context, fileID string) error

	CreateVersion(ctx context.Context, p CreateVersionParams) (database.FileVersion, error)
	ListVersions(ctx context.Context, fileID string) ([]database.FileVersion, error)
	GetVersion(ctx context.Context, versionID string) (database.FileVersion, error)

	ListProjects(ctx context.Context, workspaceID string) ([]database.ListProjectsForFileSyncRow, error)
	ListModules(ctx context.Context, workspaceID string) ([]database.ListModulesForFileSyncRow, error)

	ResolveFeedbackToken(ctx context.Context, token string) (projectID, workspaceID, projectName string, enabled bool, err error)
}

type repository struct {
	queries *database.Queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{queries: database.New(pool)}
}

func nullableUUID(s *string) (pgtype.UUID, error) {
	if s == nil || *s == "" {
		return pgtype.UUID{}, nil
	}
	return pgutil.ParseUUID(*s)
}

func (r *repository) GetCallerRole(ctx context.Context, workspaceID, callerID string) (workspace.WorkspaceRole, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return "", err
	}
	uid, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return "", err
	}
	role, err := r.queries.GetCallerWorkspaceRole(ctx, database.GetCallerWorkspaceRoleParams{WorkspaceID: wid, UserID: uid})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return workspace.WorkspaceRole(string(role)), nil
}

func (r *repository) GetSettings(ctx context.Context, workspaceID string) (database.WorkspaceFileSetting, bool, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return database.WorkspaceFileSetting{}, false, err
	}
	row, err := r.queries.GetWorkspaceFileSettings(ctx, wid)
	if errors.Is(err, pgx.ErrNoRows) {
		return database.WorkspaceFileSetting{}, false, nil
	}
	if err != nil {
		return database.WorkspaceFileSetting{}, false, err
	}
	return row, true, nil
}

func (r *repository) UpsertSettings(ctx context.Context, workspaceID string, maxMB int, allowedTypes string) (database.WorkspaceFileSetting, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return database.WorkspaceFileSetting{}, err
	}
	return r.queries.UpsertWorkspaceFileSettings(ctx, database.UpsertWorkspaceFileSettingsParams{
		WorkspaceID:   wid,
		MaxFileSizeMb: int32(maxMB),
		AllowedTypes:  allowedTypes,
	})
}

func (r *repository) ListFolders(ctx context.Context, workspaceID string) ([]database.FileFolder, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	return r.queries.ListFileFoldersByWorkspace(ctx, wid)
}

func (r *repository) GetFolder(ctx context.Context, folderID string) (database.FileFolder, error) {
	id, err := pgutil.ParseUUID(folderID)
	if err != nil {
		return database.FileFolder{}, err
	}
	return r.queries.GetFileFolderByID(ctx, id)
}

func (r *repository) CreateFolder(ctx context.Context, p CreateFolderParams) (database.FileFolder, error) {
	wid, err := pgutil.ParseUUID(p.WorkspaceID)
	if err != nil {
		return database.FileFolder{}, err
	}
	parent, err := nullableUUID(p.ParentID)
	if err != nil {
		return database.FileFolder{}, err
	}
	project, err := nullableUUID(p.ProjectID)
	if err != nil {
		return database.FileFolder{}, err
	}
	module, err := nullableUUID(p.ModuleID)
	if err != nil {
		return database.FileFolder{}, err
	}
	createdBy, err := nullableUUID(p.CreatedBy)
	if err != nil {
		return database.FileFolder{}, err
	}
	return r.queries.CreateFileFolder(ctx, database.CreateFileFolderParams{
		WorkspaceID: wid,
		ParentID:    parent,
		Name:        p.Name,
		Kind:        p.Kind,
		ProjectID:   project,
		ModuleID:    module,
		IsPublic:    p.IsPublic,
		CreatedBy:   createdBy,
	})
}

func (r *repository) RenameFolder(ctx context.Context, folderID, name string) (database.FileFolder, error) {
	id, err := pgutil.ParseUUID(folderID)
	if err != nil {
		return database.FileFolder{}, err
	}
	return r.queries.UpdateFileFolderName(ctx, database.UpdateFileFolderNameParams{ID: id, Name: name})
}

func (r *repository) MoveFolder(ctx context.Context, folderID string, parentID *string) (database.FileFolder, error) {
	id, err := pgutil.ParseUUID(folderID)
	if err != nil {
		return database.FileFolder{}, err
	}
	parent, err := nullableUUID(parentID)
	if err != nil {
		return database.FileFolder{}, err
	}
	return r.queries.UpdateFileFolderParent(ctx, database.UpdateFileFolderParentParams{ID: id, ParentID: parent})
}

func (r *repository) SetFolderPublic(ctx context.Context, folderID string, public bool) (database.FileFolder, error) {
	id, err := pgutil.ParseUUID(folderID)
	if err != nil {
		return database.FileFolder{}, err
	}
	return r.queries.UpdateFileFolderPublic(ctx, database.UpdateFileFolderPublicParams{ID: id, IsPublic: public})
}

func (r *repository) SoftDeleteFolder(ctx context.Context, folderID string) error {
	id, err := pgutil.ParseUUID(folderID)
	if err != nil {
		return err
	}
	return r.queries.SoftDeleteFileFolder(ctx, id)
}

func (r *repository) ListFiles(ctx context.Context, workspaceID string) ([]database.File, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	return r.queries.ListFilesByWorkspace(ctx, wid)
}

func (r *repository) GetFile(ctx context.Context, fileID string) (database.File, error) {
	id, err := pgutil.ParseUUID(fileID)
	if err != nil {
		return database.File{}, err
	}
	return r.queries.GetFileByID(ctx, id)
}

func (r *repository) CreateFile(ctx context.Context, p CreateFileParams) (database.File, error) {
	wid, err := pgutil.ParseUUID(p.WorkspaceID)
	if err != nil {
		return database.File{}, err
	}
	folder, err := nullableUUID(p.FolderID)
	if err != nil {
		return database.File{}, err
	}
	createdBy, err := nullableUUID(p.CreatedBy)
	if err != nil {
		return database.File{}, err
	}
	return r.queries.CreateFile(ctx, database.CreateFileParams{
		WorkspaceID: wid,
		FolderID:    folder,
		Name:        p.Name,
		ObjectName:  p.ObjectName,
		ContentType: p.ContentType,
		SizeBytes:   p.SizeBytes,
		IsPublic:    p.IsPublic,
		CreatedBy:   createdBy,
	})
}

func (r *repository) RenameFile(ctx context.Context, fileID, name string) (database.File, error) {
	id, err := pgutil.ParseUUID(fileID)
	if err != nil {
		return database.File{}, err
	}
	return r.queries.UpdateFileName(ctx, database.UpdateFileNameParams{ID: id, Name: name})
}

func (r *repository) MoveFile(ctx context.Context, fileID string, folderID *string) (database.File, error) {
	id, err := pgutil.ParseUUID(fileID)
	if err != nil {
		return database.File{}, err
	}
	folder, err := nullableUUID(folderID)
	if err != nil {
		return database.File{}, err
	}
	return r.queries.UpdateFileFolder(ctx, database.UpdateFileFolderParams{ID: id, FolderID: folder})
}

func (r *repository) SetFilePublic(ctx context.Context, fileID string, public bool) (database.File, error) {
	id, err := pgutil.ParseUUID(fileID)
	if err != nil {
		return database.File{}, err
	}
	return r.queries.UpdateFilePublic(ctx, database.UpdateFilePublicParams{ID: id, IsPublic: public})
}

func (r *repository) SetFileObject(ctx context.Context, fileID, objectName, contentType string, size int64) (database.File, error) {
	id, err := pgutil.ParseUUID(fileID)
	if err != nil {
		return database.File{}, err
	}
	return r.queries.UpdateFileObject(ctx, database.UpdateFileObjectParams{
		ID:          id,
		ObjectName:  objectName,
		ContentType: contentType,
		SizeBytes:   size,
	})
}

func (r *repository) SoftDeleteFile(ctx context.Context, fileID string) error {
	id, err := pgutil.ParseUUID(fileID)
	if err != nil {
		return err
	}
	return r.queries.SoftDeleteFile(ctx, id)
}

func (r *repository) CreateVersion(ctx context.Context, p CreateVersionParams) (database.FileVersion, error) {
	fid, err := pgutil.ParseUUID(p.FileID)
	if err != nil {
		return database.FileVersion{}, err
	}
	createdBy, err := nullableUUID(p.CreatedBy)
	if err != nil {
		return database.FileVersion{}, err
	}
	return r.queries.CreateFileVersion(ctx, database.CreateFileVersionParams{
		FileID:      fid,
		ObjectName:  p.ObjectName,
		ContentType: p.ContentType,
		SizeBytes:   p.SizeBytes,
		CreatedBy:   createdBy,
	})
}

func (r *repository) ListVersions(ctx context.Context, fileID string) ([]database.FileVersion, error) {
	fid, err := pgutil.ParseUUID(fileID)
	if err != nil {
		return nil, err
	}
	return r.queries.ListFileVersionsByFile(ctx, fid)
}

func (r *repository) GetVersion(ctx context.Context, versionID string) (database.FileVersion, error) {
	id, err := pgutil.ParseUUID(versionID)
	if err != nil {
		return database.FileVersion{}, err
	}
	return r.queries.GetFileVersionByID(ctx, id)
}

func (r *repository) ListProjects(ctx context.Context, workspaceID string) ([]database.ListProjectsForFileSyncRow, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	return r.queries.ListProjectsForFileSync(ctx, wid)
}

func (r *repository) ListModules(ctx context.Context, workspaceID string) ([]database.ListModulesForFileSyncRow, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	return r.queries.ListModulesForFileSync(ctx, wid)
}

func (r *repository) ResolveFeedbackToken(ctx context.Context, token string) (string, string, string, bool, error) {
	row, err := r.queries.GetProjectByFeedbackToken(ctx, token)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", "", false, nil
	}
	if err != nil {
		return "", "", "", false, err
	}
	return pgutil.UUIDToString(row.ProjectID), pgutil.UUIDToString(row.WorkspaceID), row.ProjectName, row.Enabled, nil
}
