package files

import (
	"context"
	"strings"

	"github.com/alexcao194/kairso/internal/database"
	storage "github.com/alexcao194/kairso/pkg/minio"
	"github.com/alexcao194/kairso/pkg/pgutil"
)

func (s *Service) CreateFolder(ctx context.Context, workspaceID, callerID string, req CreateFolderRequest) (*FolderResponse, error) {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, invalid("Thiếu tên thư mục")
	}
	var parentID *string
	if strings.TrimSpace(req.ParentID) != "" {
		parent, err := s.repo.GetFolder(ctx, req.ParentID)
		if err != nil {
			return nil, ErrNotFound
		}
		if pgutil.UUIDToString(parent.WorkspaceID) != workspaceID {
			return nil, ErrForbidden
		}
		p := req.ParentID
		parentID = &p
	}
	cb := callerID
	row, err := s.repo.CreateFolder(ctx, CreateFolderParams{
		WorkspaceID: workspaceID, ParentID: parentID, Name: name,
		Kind: FolderKindCustom, IsPublic: req.IsPublic, CreatedBy: &cb,
	})
	if err != nil {
		return nil, err
	}
	resp := folderResponse(row)
	return &resp, nil
}

func (s *Service) UpdateFolder(ctx context.Context, workspaceID, callerID, folderID string, req UpdateFolderRequest) (*FolderResponse, error) {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	folder, err := s.repo.GetFolder(ctx, folderID)
	if err != nil {
		return nil, ErrNotFound
	}
	if pgutil.UUIDToString(folder.WorkspaceID) != workspaceID {
		return nil, ErrForbidden
	}
	locked := folder.Kind != FolderKindCustom

	if req.IsPublic != nil {
		if _, err := s.repo.SetFolderPublic(ctx, folderID, *req.IsPublic); err != nil {
			return nil, err
		}
	}
	if req.Name != nil {
		if locked {
			return nil, ErrLocked
		}
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, invalid("Thiếu tên thư mục")
		}
		if _, err := s.repo.RenameFolder(ctx, folderID, name); err != nil {
			return nil, err
		}
	}
	if req.ParentID != nil {
		if locked {
			return nil, ErrLocked
		}
		if err := s.validateMoveTarget(ctx, workspaceID, folderID, *req.ParentID); err != nil {
			return nil, err
		}
		var parentID *string
		if strings.TrimSpace(*req.ParentID) != "" {
			p := *req.ParentID
			parentID = &p
		}
		if _, err := s.repo.MoveFolder(ctx, folderID, parentID); err != nil {
			return nil, err
		}
	}
	updated, err := s.repo.GetFolder(ctx, folderID)
	if err != nil {
		return nil, err
	}
	resp := folderResponse(updated)
	return &resp, nil
}

func (s *Service) validateMoveTarget(ctx context.Context, workspaceID, folderID, newParentID string) error {
	newParentID = strings.TrimSpace(newParentID)
	if newParentID == "" {
		return nil
	}
	if newParentID == folderID {
		return invalid("Không thể chuyển thư mục vào chính nó")
	}
	folders, err := s.repo.ListFolders(ctx, workspaceID)
	if err != nil {
		return err
	}
	childrenOf := map[string][]string{}
	inWorkspace := map[string]bool{}
	for _, f := range folders {
		id := pgutil.UUIDToString(f.ID)
		inWorkspace[id] = true
		if f.ParentID.Valid {
			pid := pgutil.UUIDToString(f.ParentID)
			childrenOf[pid] = append(childrenOf[pid], id)
		}
	}
	if !inWorkspace[newParentID] {
		return ErrForbidden
	}
	stack := []string{folderID}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, c := range childrenOf[cur] {
			if c == newParentID {
				return invalid("Không thể chuyển thư mục vào thư mục con của nó")
			}
			stack = append(stack, c)
		}
	}
	return nil
}

func (s *Service) DeleteFolder(ctx context.Context, workspaceID, callerID, folderID string) error {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return err
	}
	folder, err := s.repo.GetFolder(ctx, folderID)
	if err != nil {
		return ErrNotFound
	}
	if pgutil.UUIDToString(folder.WorkspaceID) != workspaceID {
		return ErrForbidden
	}
	if folder.Kind != FolderKindCustom {
		return ErrLocked
	}
	folders, err := s.repo.ListFolders(ctx, workspaceID)
	if err != nil {
		return err
	}
	childrenOf := map[string][]string{}
	for _, f := range folders {
		if f.ParentID.Valid {
			pid := pgutil.UUIDToString(f.ParentID)
			childrenOf[pid] = append(childrenOf[pid], pgutil.UUIDToString(f.ID))
		}
	}
	toDelete := map[string]bool{}
	stack := []string{folderID}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if toDelete[cur] {
			continue
		}
		toDelete[cur] = true
		stack = append(stack, childrenOf[cur]...)
	}
	fileRows, err := s.repo.ListFiles(ctx, workspaceID)
	if err != nil {
		return err
	}
	for _, f := range fileRows {
		if f.FolderID.Valid && toDelete[pgutil.UUIDToString(f.FolderID)] {
			if err := s.repo.SoftDeleteFile(ctx, pgutil.UUIDToString(f.ID)); err != nil {
				return err
			}
		}
	}
	for id := range toDelete {
		if err := s.repo.SoftDeleteFolder(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) PresignUpload(ctx context.Context, workspaceID, callerID string, req PresignUploadRequest) (*PresignUploadResponse, error) {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	if s.storage == nil {
		return nil, ErrNoStorage
	}
	maxMB, allowed, err := s.effectiveSettings(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if err := validateUpload(req.Filename, req.SizeBytes, maxMB, allowed); err != nil {
		return nil, err
	}
	objectName := storage.FileObjectName("workspace", workspaceID, req.Filename)
	out, err := s.storage.PresignPutObject(ctx, storage.PresignPutInput{
		ObjectName:  objectName,
		ContentType: req.ContentType,
		Expires:     presignExpiry,
	})
	if err != nil {
		return nil, err
	}
	return &PresignUploadResponse{
		UploadURL:  out.UploadURL,
		Method:     out.Method,
		ObjectName: out.ObjectName,
		FileURL:    out.FileURL,
		ExpiresAt:  out.ExpiresAt,
	}, nil
}

func (s *Service) CreateFile(ctx context.Context, workspaceID, callerID string, req CreateFileRequest) (*FileResponse, error) {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "tệp"
	}
	if strings.TrimSpace(req.ObjectName) == "" {
		return nil, invalid("Thiếu objectName (tải tệp lên trước)")
	}
	maxMB, allowed, err := s.effectiveSettings(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if err := validateUpload(name, req.SizeBytes, maxMB, allowed); err != nil {
		return nil, err
	}
	var folderID *string
	if strings.TrimSpace(req.FolderID) != "" {
		folder, err := s.repo.GetFolder(ctx, req.FolderID)
		if err != nil {
			return nil, ErrNotFound
		}
		if pgutil.UUIDToString(folder.WorkspaceID) != workspaceID {
			return nil, ErrForbidden
		}
		f := req.FolderID
		folderID = &f
	}
	cb := callerID
	row, err := s.repo.CreateFile(ctx, CreateFileParams{
		WorkspaceID: workspaceID, FolderID: folderID, Name: name,
		ObjectName: req.ObjectName, ContentType: req.ContentType, SizeBytes: req.SizeBytes,
		IsPublic: req.IsPublic, CreatedBy: &cb,
	})
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.CreateVersion(ctx, CreateVersionParams{
		FileID: pgutil.UUIDToString(row.ID), ObjectName: req.ObjectName,
		ContentType: req.ContentType, SizeBytes: req.SizeBytes, CreatedBy: &cb,
	}); err != nil {
		return nil, err
	}
	resp := fileResponse(row, s.url(row.ObjectName))
	return &resp, nil
}

func (s *Service) UpdateFile(ctx context.Context, workspaceID, callerID, fileID string, req UpdateFileRequest) (*FileResponse, error) {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	file, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return nil, ErrNotFound
	}
	if pgutil.UUIDToString(file.WorkspaceID) != workspaceID {
		return nil, ErrForbidden
	}
	if req.IsPublic != nil {
		if _, err := s.repo.SetFilePublic(ctx, fileID, *req.IsPublic); err != nil {
			return nil, err
		}
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, invalid("Thiếu tên tệp")
		}
		if _, err := s.repo.RenameFile(ctx, fileID, name); err != nil {
			return nil, err
		}
	}
	if req.FolderID != nil {
		var folderID *string
		if strings.TrimSpace(*req.FolderID) != "" {
			folder, err := s.repo.GetFolder(ctx, *req.FolderID)
			if err != nil {
				return nil, ErrNotFound
			}
			if pgutil.UUIDToString(folder.WorkspaceID) != workspaceID {
				return nil, ErrForbidden
			}
			f := *req.FolderID
			folderID = &f
		}
		if _, err := s.repo.MoveFile(ctx, fileID, folderID); err != nil {
			return nil, err
		}
	}
	updated, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return nil, err
	}
	resp := fileResponse(updated, s.url(updated.ObjectName))
	return &resp, nil
}

func (s *Service) DeleteFile(ctx context.Context, workspaceID, callerID, fileID string) error {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return err
	}
	file, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return ErrNotFound
	}
	if pgutil.UUIDToString(file.WorkspaceID) != workspaceID {
		return ErrForbidden
	}
	return s.repo.SoftDeleteFile(ctx, fileID)
}

func (s *Service) AddVersion(ctx context.Context, workspaceID, callerID, fileID string, req CreateVersionRequest) (*FileResponse, error) {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	file, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return nil, ErrNotFound
	}
	if pgutil.UUIDToString(file.WorkspaceID) != workspaceID {
		return nil, ErrForbidden
	}
	if strings.TrimSpace(req.ObjectName) == "" {
		return nil, invalid("Thiếu objectName (tải tệp lên trước)")
	}
	maxMB, allowed, err := s.effectiveSettings(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if err := validateUpload(file.Name, req.SizeBytes, maxMB, allowed); err != nil {
		return nil, err
	}
	cb := callerID
	if _, err := s.repo.CreateVersion(ctx, CreateVersionParams{
		FileID: fileID, ObjectName: req.ObjectName, ContentType: req.ContentType,
		SizeBytes: req.SizeBytes, CreatedBy: &cb,
	}); err != nil {
		return nil, err
	}
	updated, err := s.repo.SetFileObject(ctx, fileID, req.ObjectName, req.ContentType, req.SizeBytes)
	if err != nil {
		return nil, err
	}
	resp := fileResponse(updated, s.url(updated.ObjectName))
	return &resp, nil
}

func (s *Service) ListVersions(ctx context.Context, workspaceID, callerID, fileID string) ([]FileVersionResponse, error) {
	if err := s.requireMember(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	file, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return nil, ErrNotFound
	}
	if pgutil.UUIDToString(file.WorkspaceID) != workspaceID {
		return nil, ErrForbidden
	}
	rows, err := s.repo.ListVersions(ctx, fileID)
	if err != nil {
		return nil, err
	}
	out := make([]FileVersionResponse, 0, len(rows))
	for _, v := range rows {
		out = append(out, fileVersionResponse(v, s.url(v.ObjectName)))
	}
	return out, nil
}

func (s *Service) RestoreVersion(ctx context.Context, workspaceID, callerID, fileID, versionID string) (*FileResponse, error) {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	file, err := s.repo.GetFile(ctx, fileID)
	if err != nil {
		return nil, ErrNotFound
	}
	if pgutil.UUIDToString(file.WorkspaceID) != workspaceID {
		return nil, ErrForbidden
	}
	version, err := s.repo.GetVersion(ctx, versionID)
	if err != nil {
		return nil, ErrNotFound
	}
	if pgutil.UUIDToString(version.FileID) != fileID {
		return nil, ErrForbidden
	}
	updated, err := s.repo.SetFileObject(ctx, fileID, version.ObjectName, version.ContentType, version.SizeBytes)
	if err != nil {
		return nil, err
	}
	resp := fileResponse(updated, s.url(updated.ObjectName))
	return &resp, nil
}

func (s *Service) GetPublicTree(ctx context.Context, token string) (*TreeResponse, error) {
	projectID, workspaceID, _, enabled, err := s.repo.ResolveFeedbackToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if projectID == "" || !enabled {
		return nil, ErrNotFound
	}
	folders, err := s.repo.ListFolders(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	fileRows, err := s.repo.ListFiles(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	var projectFolderID string
	for _, f := range folders {
		if f.Kind == FolderKindProject && f.ProjectID.Valid && pgutil.UUIDToString(f.ProjectID) == projectID {
			projectFolderID = pgutil.UUIDToString(f.ID)
			break
		}
	}
	empty := &TreeResponse{Folders: []*TreeNode{}, RootFiles: []FileResponse{}}
	if projectFolderID == "" {
		return empty, nil
	}

	effective := effectivePublicMap(folders)
	if !effective[projectFolderID] {
		return empty, nil
	}

	childrenOf := map[string][]string{}
	for _, f := range folders {
		if f.ParentID.Valid {
			childrenOf[pgutil.UUIDToString(f.ParentID)] = append(childrenOf[pgutil.UUIDToString(f.ParentID)], pgutil.UUIDToString(f.ID))
		}
	}
	inProject := map[string]bool{}
	stack := []string{projectFolderID}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		inProject[cur] = true
		stack = append(stack, childrenOf[cur]...)
	}

	publicOnly := map[string]bool{}
	for id := range inProject {
		if effective[id] {
			publicOnly[id] = true
		}
	}
	for _, f := range fileRows {
		if !f.IsPublic || !f.FolderID.Valid {
			continue
		}
		fid := pgutil.UUIDToString(f.FolderID)
		if inProject[fid] && effective[fid] {
			publicOnly["file:"+pgutil.UUIDToString(f.ID)] = true
		}
	}

	tree := s.buildTree(folders, fileRows, publicOnly)
	for _, root := range tree.Folders {
		if root.ID == projectFolderID {
			return &TreeResponse{Folders: root.Children, RootFiles: root.Files}, nil
		}
	}
	return empty, nil
}

func effectivePublicMap(folders []database.FileFolder) map[string]bool {
	byID := make(map[string]database.FileFolder, len(folders))
	for _, f := range folders {
		byID[pgutil.UUIDToString(f.ID)] = f
	}
	memo := map[string]bool{}
	var eval func(id string, seen map[string]bool) bool
	eval = func(id string, seen map[string]bool) bool {
		if v, ok := memo[id]; ok {
			return v
		}
		if seen[id] {
			return false
		}
		seen[id] = true
		f, ok := byID[id]
		if !ok {
			memo[id] = false
			return false
		}
		res := f.IsPublic
		if res && f.ParentID.Valid {
			res = eval(pgutil.UUIDToString(f.ParentID), seen)
		}
		memo[id] = res
		return res
	}
	for id := range byID {
		eval(id, map[string]bool{})
	}
	return memo
}
