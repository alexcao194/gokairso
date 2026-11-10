package files

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	storage "github.com/alexcao194/kairso/pkg/minio"
	"github.com/alexcao194/kairso/pkg/pgutil"
)

const (
	defaultMaxFileSizeMB = 50
	presignExpiry        = 15 * time.Minute
)

var (
	ErrForbidden = errors.New("Bạn không có quyền thực hiện thao tác này")
	ErrNotFound  = errors.New("Không tìm thấy mục này")
	ErrLocked    = errors.New("Không thể đổi tên/di chuyển/xoá thư mục mặc định theo dự án hoặc module")
	ErrNoStorage = errors.New("Máy chủ chưa cấu hình kho lưu trữ tệp")
)

type ValidationError struct{ Msg string }

func (e ValidationError) Error() string { return e.Msg }

func invalid(format string, a ...any) error { return ValidationError{Msg: fmt.Sprintf(format, a...)} }

type Storage interface {
	PresignPutObject(ctx context.Context, input storage.PresignPutInput) (storage.PresignPutOutput, error)
	ObjectURL(objectName string) string
}

type Service struct {
	repo    Repository
	storage Storage
}

func NewService(repo Repository, store Storage) *Service {
	return &Service{repo: repo, storage: store}
}

func (s *Service) url(objectName string) string {
	if s.storage == nil || objectName == "" {
		return ""
	}
	return s.storage.ObjectURL(objectName)
}

func (s *Service) requireMember(ctx context.Context, workspaceID, callerID string) error {
	role, err := s.repo.GetCallerRole(ctx, workspaceID, callerID)
	if err != nil {
		return err
	}
	if role == "" {
		return ErrForbidden
	}
	return nil
}

func (s *Service) requireManager(ctx context.Context, workspaceID, callerID string) error {
	role, err := s.repo.GetCallerRole(ctx, workspaceID, callerID)
	if err != nil {
		return err
	}
	if !role.CanManageMembers() {
		return ErrForbidden
	}
	return nil
}

func (s *Service) GetSettings(ctx context.Context, workspaceID, callerID string) (*FileSettingsResponse, error) {
	if err := s.requireMember(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	row, found, err := s.repo.GetSettings(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &FileSettingsResponse{MaxFileSizeMB: defaultMaxFileSizeMB, AllowedTypes: ""}, nil
	}
	resp := fileSettingsResponse(row)
	return &resp, nil
}

func (s *Service) UpsertSettings(ctx context.Context, workspaceID, callerID string, req UpsertFileSettingsRequest) (*FileSettingsResponse, error) {
	if err := s.requireManager(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	if req.MaxFileSizeMB <= 0 || req.MaxFileSizeMB > 5000 {
		return nil, invalid("Dung lượng tối đa phải trong khoảng 1–5000 MB")
	}
	allowed := normalizeAllowedTypes(req.AllowedTypes)
	row, err := s.repo.UpsertSettings(ctx, workspaceID, req.MaxFileSizeMB, allowed)
	if err != nil {
		return nil, err
	}
	resp := fileSettingsResponse(row)
	return &resp, nil
}

func normalizeAllowedTypes(raw string) string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == ';' })
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		ext := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(p), "."))
		if ext == "" || seen[ext] {
			continue
		}
		seen[ext] = true
		out = append(out, ext)
	}
	return strings.Join(out, ",")
}

func (s *Service) effectiveSettings(ctx context.Context, workspaceID string) (maxMB int, allowed string, err error) {
	row, found, err := s.repo.GetSettings(ctx, workspaceID)
	if err != nil {
		return 0, "", err
	}
	if !found {
		return defaultMaxFileSizeMB, "", nil
	}
	return int(row.MaxFileSizeMb), row.AllowedTypes, nil
}

func validateUpload(filename string, size int64, maxMB int, allowed string) error {
	if strings.TrimSpace(filename) == "" {
		return invalid("Thiếu tên tệp")
	}
	if size <= 0 {
		return invalid("Kích thước tệp không hợp lệ")
	}
	if maxMB > 0 && size > int64(maxMB)*1024*1024 {
		return invalid("Tệp vượt quá dung lượng tối đa %d MB", maxMB)
	}
	if allowed != "" {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
		ok := false
		for _, a := range strings.Split(allowed, ",") {
			if a == ext {
				ok = true
				break
			}
		}
		if !ok {
			return invalid("Loại tệp .%s không được phép (cho phép: %s)", ext, allowed)
		}
	}
	return nil
}

func (s *Service) GetTree(ctx context.Context, workspaceID, callerID string) (*TreeResponse, error) {
	if err := s.requireMember(ctx, workspaceID, callerID); err != nil {
		return nil, err
	}
	folders, err := s.reconcileLockedFolders(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	fileRows, err := s.repo.ListFiles(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	tree := s.buildTree(folders, fileRows, nil)
	return &tree, nil
}

func (s *Service) buildTree(folders []database.FileFolder, fileRows []database.File, publicOnly map[string]bool) TreeResponse {
	nodes := make(map[string]*TreeNode, len(folders))
	order := make([]*TreeNode, 0, len(folders))
	for i := range folders {
		id := pgutil.UUIDToString(folders[i].ID)
		if publicOnly != nil && !publicOnly[id] {
			continue
		}
		n := &TreeNode{FolderResponse: folderResponse(folders[i]), Children: []*TreeNode{}, Files: []FileResponse{}}
		nodes[id] = n
		order = append(order, n)
	}
	var roots []*TreeNode
	for _, n := range order {
		if n.ParentID == nil {
			roots = append(roots, n)
			continue
		}
		if p, ok := nodes[*n.ParentID]; ok {
			p.Children = append(p.Children, n)
		} else {
			roots = append(roots, n)
		}
	}
	rootFiles := []FileResponse{}
	for i := range fileRows {
		f := fileRows[i]
		fid := pgutil.UUIDToString(f.ID)
		if publicOnly != nil && !publicOnly["file:"+fid] {
			continue
		}
		fr := fileResponse(f, s.url(f.ObjectName))
		if fr.FolderID == nil {
			rootFiles = append(rootFiles, fr)
			continue
		}
		if p, ok := nodes[*fr.FolderID]; ok {
			p.Files = append(p.Files, fr)
		} else if publicOnly == nil {
			rootFiles = append(rootFiles, fr)
		}
	}
	sortNodes(roots)
	sortFiles(rootFiles)
	return TreeResponse{Folders: roots, RootFiles: rootFiles}
}

func sortNodes(ns []*TreeNode) {
	sort.SliceStable(ns, func(i, j int) bool {
		li, lj := ns[i].Locked, ns[j].Locked
		if li != lj {
			return li
		}
		return strings.ToLower(ns[i].Name) < strings.ToLower(ns[j].Name)
	})
	for _, n := range ns {
		sortNodes(n.Children)
		sortFiles(n.Files)
	}
}

func sortFiles(fs []FileResponse) {
	sort.SliceStable(fs, func(i, j int) bool { return strings.ToLower(fs[i].Name) < strings.ToLower(fs[j].Name) })
}

func (s *Service) reconcileLockedFolders(ctx context.Context, workspaceID string) ([]database.FileFolder, error) {
	folders, err := s.repo.ListFolders(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	projects, err := s.repo.ListProjects(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	modules, err := s.repo.ListModules(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	projectFolder := map[string]*database.FileFolder{}
	moduleFolder := map[string]*database.FileFolder{}
	for i := range folders {
		f := &folders[i]
		switch f.Kind {
		case FolderKindProject:
			if f.ProjectID.Valid {
				projectFolder[pgutil.UUIDToString(f.ProjectID)] = f
			}
		case FolderKindModule:
			if f.ModuleID.Valid {
				moduleFolder[pgutil.UUIDToString(f.ModuleID)] = f
			}
		}
	}

	liveProject := map[string]string{}
	liveModule := map[string]struct {
		name      string
		projectID string
	}{}
	changed := false

	for _, p := range projects {
		pid := pgutil.UUIDToString(p.ID)
		liveProject[pid] = p.Name
		existing := projectFolder[pid]
		if existing == nil {
			pidCopy := pid
			if _, err := s.repo.CreateFolder(ctx, CreateFolderParams{
				WorkspaceID: workspaceID, Name: p.Name, Kind: FolderKindProject, ProjectID: &pidCopy,
			}); err != nil {
				return nil, err
			}
			changed = true
		} else if existing.Name != p.Name {
			if _, err := s.repo.RenameFolder(ctx, pgutil.UUIDToString(existing.ID), p.Name); err != nil {
				return nil, err
			}
			changed = true
		}
	}
	for pid, f := range projectFolder {
		if _, ok := liveProject[pid]; !ok {
			if err := s.repo.SoftDeleteFolder(ctx, pgutil.UUIDToString(f.ID)); err != nil {
				return nil, err
			}
			changed = true
		}
	}

	if changed {
		folders, err = s.repo.ListFolders(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		projectFolder = map[string]*database.FileFolder{}
		for i := range folders {
			f := &folders[i]
			if f.Kind == FolderKindProject && f.ProjectID.Valid {
				projectFolder[pgutil.UUIDToString(f.ProjectID)] = f
			}
		}
		changed = false
	}

	for _, m := range modules {
		mid := pgutil.UUIDToString(m.ID)
		pid := pgutil.UUIDToString(m.ProjectID)
		liveModule[mid] = struct {
			name      string
			projectID string
		}{m.Name, pid}
		parent := projectFolder[pid]
		if parent == nil {
			continue
		}
		existing := moduleFolder[mid]
		if existing == nil {
			midCopy, pidCopy := mid, pid
			parentID := pgutil.UUIDToString(parent.ID)
			if _, err := s.repo.CreateFolder(ctx, CreateFolderParams{
				WorkspaceID: workspaceID, ParentID: &parentID, Name: m.Name,
				Kind: FolderKindModule, ProjectID: &pidCopy, ModuleID: &midCopy,
			}); err != nil {
				return nil, err
			}
			changed = true
		} else if existing.Name != m.Name {
			if _, err := s.repo.RenameFolder(ctx, pgutil.UUIDToString(existing.ID), m.Name); err != nil {
				return nil, err
			}
			changed = true
		}
	}
	for mid, f := range moduleFolder {
		if _, ok := liveModule[mid]; !ok {
			if err := s.repo.SoftDeleteFolder(ctx, pgutil.UUIDToString(f.ID)); err != nil {
				return nil, err
			}
			changed = true
		}
	}

	if changed {
		folders, err = s.repo.ListFolders(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
	}
	return folders, nil
}
