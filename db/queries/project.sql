-- name: CreateProject :one
INSERT INTO projects (workspace_id, name)
VALUES ($1, $2)
RETURNING id, created_at, updated_at, deleted_at, name, workspace_id, avatar, task_sequence_counter, feedback_sequence_counter;

-- name: ListProjectsByWorkspaceAliasWithStats :many
SELECT p.id, p.created_at, p.updated_at, p.deleted_at, p.name, p.workspace_id, p.avatar,
       COUNT(t.id)::int AS task_count,
       COALESCE(SUM(t.estimation), 0)::int AS total_estimation
FROM projects p
INNER JOIN project_users pu ON pu.project_id = p.id AND pu.user_id = sqlc.arg(user_id)
LEFT JOIN tasks t ON t.project_id = p.id AND t.deleted_at IS NULL
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR COALESCE(t.end_date, t.start_date, t.created_at) >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR COALESCE(t.start_date, t.end_date, t.created_at) <= sqlc.narg(to_time)::timestamptz)
WHERE p.workspace_id = sqlc.arg(workspace_id)::uuid AND p.deleted_at IS NULL
GROUP BY p.id, p.created_at, p.updated_at, p.deleted_at, p.name, p.workspace_id, p.avatar
ORDER BY p.name
LIMIT sqlc.arg(row_limit)::int OFFSET sqlc.arg(row_offset)::int;

-- name: GetProjectByID :one
SELECT id, created_at, updated_at, deleted_at, name, workspace_id, avatar, task_sequence_counter, feedback_sequence_counter
FROM projects
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetProjectByIDForUpdate :one
SELECT id, created_at, updated_at, deleted_at, name, workspace_id, avatar, task_sequence_counter, feedback_sequence_counter
FROM projects
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1
FOR UPDATE;

-- name: GetProjectOverviewsByIDs :many
SELECT
    p.id,
    p.created_at,
    p.updated_at,
    p.name,
    p.workspace_id,
    p.avatar,
    COUNT(DISTINCT pu.user_id)::int AS member_count,
    COUNT(DISTINCT m.id)::int AS module_count,
    COUNT(DISTINCT t.id)::int AS task_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'backlog')::int AS backlog_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'todo')::int AS todo_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'in_progress')::int AS in_progress_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'review')::int AS review_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'done')::int AS done_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'canceled')::int AS canceled_count
FROM projects p
LEFT JOIN project_users pu ON pu.project_id = p.id
LEFT JOIN modules m ON m.project_id = p.id AND m.deleted_at IS NULL
LEFT JOIN tasks t ON t.project_id = p.id AND t.deleted_at IS NULL
WHERE p.id = ANY(sqlc.arg(ids)::uuid[]) AND p.deleted_at IS NULL
GROUP BY p.id, p.created_at, p.updated_at, p.name, p.workspace_id, p.avatar;

-- name: UpdateProject :one
UPDATE projects
SET
    name = COALESCE(sqlc.narg(name), name)
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, created_at, updated_at, deleted_at, name, workspace_id, avatar, task_sequence_counter, feedback_sequence_counter;

-- name: UpdateProjectByID :one
UPDATE projects
SET
    name = CASE WHEN sqlc.arg(name_present)::boolean THEN sqlc.arg(name) ELSE name END,
    avatar = CASE WHEN sqlc.arg(avatar_present)::boolean THEN NULLIF(sqlc.arg(avatar), '') ELSE avatar END
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id, created_at, updated_at, deleted_at, name, workspace_id, avatar, task_sequence_counter, feedback_sequence_counter;

-- name: SoftDeleteProjectByID :exec
UPDATE projects
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListProjectHistories :many
SELECT h.id, h.entity_type, h.entity_id, h.changed_by_id, h.field, h.old_value, h.new_value, h.created_at,
       u.alias AS changed_by_alias, u.name AS changed_by_name, COALESCE(u.avatar, '') AS changed_by_avatar
FROM entity_update_histories h
LEFT JOIN users u ON u.id = h.changed_by_id
WHERE h.entity_type = 'project' AND h.entity_id = $1
ORDER BY h.created_at DESC, h.id DESC
LIMIT $2 OFFSET $3;

-- name: GetCallerWorkspaceRoleForProject :one
SELECT wu.role
FROM workspace_users wu
WHERE wu.workspace_id = $1 AND wu.user_id = $2
LIMIT 1;

-- name: ListProjectMembersByAlias :many
SELECT pu.project_id, pu.user_id, u.alias, u.name, COALESCE(u.avatar, ''), pu.role
FROM project_users pu
INNER JOIN users u ON u.id = pu.user_id
WHERE pu.project_id = $1
  AND u.deleted_at IS NULL
ORDER BY pu.role, u.alias
LIMIT $2 OFFSET $3;

-- name: ListAvailableProjectMembers :many
SELECT wu.workspace_id, wu.user_id, u.alias, u.name, COALESCE(u.avatar, ''), wu.role::text
FROM workspace_users wu
INNER JOIN users u ON u.id = wu.user_id
INNER JOIN projects p ON p.workspace_id = wu.workspace_id AND p.id = sqlc.arg(project_id) AND p.deleted_at IS NULL
LEFT JOIN project_users pu ON pu.project_id = p.id AND pu.user_id = wu.user_id
WHERE wu.workspace_id = sqlc.arg(workspace_id)
  AND u.deleted_at IS NULL
  AND pu.user_id IS NULL
ORDER BY wu.role, u.alias;

-- name: UpsertProjectUser :one
INSERT INTO project_users (project_id, user_id, role)
VALUES ($1, $2, $3::project_role)
ON CONFLICT (project_id, user_id) DO UPDATE
SET role = EXCLUDED.role
RETURNING project_id, user_id, role;

-- name: AddProjectMemberByID :one
INSERT INTO project_users (project_id, user_id, role)
VALUES (sqlc.arg(project_id), sqlc.arg(user_id), sqlc.arg(role)::project_role)
ON CONFLICT (project_id, user_id) DO UPDATE SET role = EXCLUDED.role
RETURNING project_id, user_id,
    (SELECT users.alias FROM users WHERE users.id = project_users.user_id),
    (SELECT users.name FROM users WHERE users.id = project_users.user_id),
    COALESCE((SELECT users.avatar FROM users WHERE users.id = project_users.user_id), ''),
    role;

-- name: GetProjectMemberRole :one
SELECT role FROM project_users WHERE project_id = $1 AND user_id = $2;

-- name: UpdateProjectMemberRoleByID :one
UPDATE project_users pu
SET role = $3::project_role
FROM users u
WHERE pu.user_id = u.id
  AND pu.project_id = $1
  AND pu.user_id = $2
  AND u.deleted_at IS NULL
RETURNING pu.project_id, pu.user_id, u.alias, u.name, COALESCE(u.avatar, ''), pu.role;

-- name: DeleteProjectUser :exec
DELETE FROM project_users
WHERE project_id = $1 AND user_id = $2;

-- name: CreateProjectLink :one
WITH inserted AS (
    INSERT INTO project_links (project_id, title, url, created_by_id)
    VALUES (sqlc.arg(project_id), sqlc.arg(title), sqlc.arg(url), sqlc.arg(created_by_id))
    RETURNING id, created_at, updated_at, project_id, title, url, created_by_id
)
SELECT i.id, i.created_at, i.updated_at, i.project_id, i.title, i.url, i.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM inserted i
JOIN users u ON u.id = i.created_by_id;

-- name: ListProjectLinks :many
SELECT l.id, l.created_at, l.updated_at, l.project_id, l.title, l.url, l.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM project_links l
JOIN users u ON u.id = l.created_by_id
WHERE l.project_id = $1
ORDER BY l.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateProjectLink :one
WITH updated AS (
    UPDATE project_links
    SET title = COALESCE(sqlc.narg(link_title)::text, project_links.title),
        url = COALESCE(sqlc.narg(link_url)::text, project_links.url)
    WHERE project_links.id = sqlc.arg(id)
    RETURNING id, created_at, updated_at, project_id, title, url, created_by_id
)
SELECT u_l.id, u_l.created_at, u_l.updated_at, u_l.project_id, u_l.title, u_l.url, u_l.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM updated u_l
JOIN users u ON u.id = u_l.created_by_id;

-- name: DeleteProjectLink :one
DELETE FROM project_links
WHERE id = $1
RETURNING project_id;

-- name: CreateProjectNote :one
WITH inserted AS (
    INSERT INTO project_notes (project_id, title, content, created_by_id)
    VALUES ($1, $2, $3, $4)
    RETURNING id, created_at, updated_at, project_id, title, content, created_by_id
)
SELECT i.id, i.created_at, i.updated_at, i.project_id, i.title, i.content, i.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM inserted i
JOIN users u ON u.id = i.created_by_id;

-- name: ListProjectNotes :many
SELECT n.id, n.created_at, n.updated_at, n.project_id, n.title, n.content, n.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM project_notes n
JOIN users u ON u.id = n.created_by_id
WHERE n.project_id = $1
ORDER BY n.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateProjectNote :one
WITH updated AS (
    UPDATE project_notes
    SET
        title = COALESCE(sqlc.narg(note_title)::text, project_notes.title),
        content = COALESCE(sqlc.narg(note_content)::text, project_notes.content)
    WHERE project_notes.id = sqlc.arg(id)
    RETURNING id, created_at, updated_at, project_id, title, content, created_by_id
)
SELECT u_n.id, u_n.created_at, u_n.updated_at, u_n.project_id, u_n.title, u_n.content, u_n.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM updated u_n
JOIN users u ON u.id = u_n.created_by_id;

-- name: DeleteProjectNote :one
DELETE FROM project_notes
WHERE id = $1
RETURNING project_id;

-- name: CreateProjectFile :one
WITH inserted AS (
    INSERT INTO project_files (
        project_id,
        title,
        file_url,
        file_name,
        content_type,
        size_bytes,
        created_by_id
    )
    VALUES (
        sqlc.arg(project_id),
        sqlc.arg(title),
        sqlc.arg(file_url),
        sqlc.arg(file_name),
        sqlc.arg(content_type),
        sqlc.arg(size_bytes),
        sqlc.arg(created_by_id)
    )
    RETURNING id, created_at, updated_at, project_id, title, file_url, file_name, content_type, size_bytes, created_by_id
)
SELECT i.id, i.created_at, i.updated_at, i.project_id, i.title, i.file_url, i.file_name, i.content_type, i.size_bytes, i.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM inserted i
JOIN users u ON u.id = i.created_by_id;

-- name: ListProjectFiles :many
SELECT f.id, f.created_at, f.updated_at, f.project_id, f.title, f.file_url, f.file_name, f.content_type, f.size_bytes, f.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM project_files f
JOIN users u ON u.id = f.created_by_id
WHERE f.project_id = $1
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteProjectFile :one
DELETE FROM project_files
WHERE id = $1
RETURNING project_id;
