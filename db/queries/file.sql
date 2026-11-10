
-- name: GetWorkspaceFileSettings :one
SELECT * FROM workspace_file_settings WHERE workspace_id = $1 LIMIT 1;

-- name: UpsertWorkspaceFileSettings :one
INSERT INTO workspace_file_settings (workspace_id, max_file_size_mb, allowed_types)
VALUES ($1, $2, $3)
ON CONFLICT (workspace_id) DO UPDATE
    SET max_file_size_mb = EXCLUDED.max_file_size_mb,
        allowed_types = EXCLUDED.allowed_types,
        updated_at = NOW()
RETURNING *;

-- name: ListFileFoldersByWorkspace :many
SELECT * FROM file_folders
WHERE workspace_id = $1 AND deleted_at IS NULL
ORDER BY name;

-- name: GetFileFolderByID :one
SELECT * FROM file_folders WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: CreateFileFolder :one
INSERT INTO file_folders (workspace_id, parent_id, name, kind, project_id, module_id, is_public, created_by)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.narg(parent_id),
    sqlc.arg(name),
    sqlc.arg(kind),
    sqlc.narg(project_id),
    sqlc.narg(module_id),
    sqlc.arg(is_public),
    sqlc.narg(created_by)
)
RETURNING *;

-- name: UpdateFileFolderName :one
UPDATE file_folders SET name = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: UpdateFileFolderParent :one
UPDATE file_folders SET parent_id = sqlc.narg(parent_id)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL RETURNING *;

-- name: UpdateFileFolderPublic :one
UPDATE file_folders SET is_public = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: SoftDeleteFileFolder :exec
UPDATE file_folders SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL;

-- name: ListFilesByWorkspace :many
SELECT * FROM files WHERE workspace_id = $1 AND deleted_at IS NULL ORDER BY name;

-- name: GetFileByID :one
SELECT * FROM files WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: CreateFile :one
INSERT INTO files (workspace_id, folder_id, name, object_name, content_type, size_bytes, is_public, created_by)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.narg(folder_id),
    sqlc.arg(name),
    sqlc.arg(object_name),
    sqlc.arg(content_type),
    sqlc.arg(size_bytes),
    sqlc.arg(is_public),
    sqlc.narg(created_by)
)
RETURNING *;

-- name: UpdateFileName :one
UPDATE files SET name = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: UpdateFileFolder :one
UPDATE files SET folder_id = sqlc.narg(folder_id)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL RETURNING *;

-- name: UpdateFilePublic :one
UPDATE files SET is_public = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: UpdateFileObject :one
UPDATE files SET object_name = $2, content_type = $3, size_bytes = $4
WHERE id = $1 AND deleted_at IS NULL RETURNING *;

-- name: SoftDeleteFile :exec
UPDATE files SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateFileVersion :one
INSERT INTO file_versions (file_id, object_name, content_type, size_bytes, created_by)
VALUES ($1, $2, $3, $4, sqlc.narg(created_by))
RETURNING *;

-- name: ListFileVersionsByFile :many
SELECT * FROM file_versions WHERE file_id = $1 ORDER BY created_at DESC;

-- name: GetFileVersionByID :one
SELECT * FROM file_versions WHERE id = $1 LIMIT 1;

-- name: ListProjectsForFileSync :many
SELECT id, name FROM projects
WHERE workspace_id = $1 AND deleted_at IS NULL ORDER BY created_at;

-- name: ListModulesForFileSync :many
SELECT m.id, m.name, m.project_id
FROM modules m
JOIN projects p ON p.id = m.project_id
WHERE p.workspace_id = $1 AND m.deleted_at IS NULL AND p.deleted_at IS NULL
ORDER BY m.created_at;

-- name: GetProjectByFeedbackToken :one
SELECT p.id AS project_id, p.workspace_id, p.name AS project_name, s.enabled
FROM project_feedback_settings s
JOIN projects p ON p.id = s.project_id
WHERE s.token = $1 AND p.deleted_at IS NULL
LIMIT 1;
