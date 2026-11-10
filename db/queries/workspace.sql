-- name: CreateWorkspace :one
INSERT INTO workspaces (created_by_id, name)
VALUES ($1, $2)
RETURNING id, created_at, updated_at, deleted_at, created_by_id, name, avatar;

-- name: ListWorkspacesByUser :many
SELECT w.id, w.created_at, w.updated_at, w.deleted_at, w.created_by_id, w.name, w.avatar
FROM workspaces w
INNER JOIN workspace_users wu ON wu.workspace_id = w.id
WHERE wu.user_id = $1 AND w.deleted_at IS NULL
ORDER BY w.name;

-- name: ListWorkspacesByUserPaginated :many
SELECT w.id, w.created_at, w.updated_at, w.deleted_at, w.created_by_id, w.name, w.avatar
FROM workspaces w
INNER JOIN workspace_users wu ON wu.workspace_id = w.id
WHERE wu.user_id = $1 AND w.deleted_at IS NULL
ORDER BY w.name
LIMIT $2 OFFSET $3;

-- name: GetWorkspaceByID :one
SELECT id, created_at, updated_at, deleted_at, created_by_id, name, avatar
FROM workspaces
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetWorkspaceByIDForUpdate :one
SELECT id, created_at, updated_at, deleted_at, created_by_id, name, avatar
FROM workspaces
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1
FOR UPDATE;

-- name: UpdateWorkspace :one
UPDATE workspaces
SET
    name = COALESCE(sqlc.narg(name), name)
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, created_at, updated_at, deleted_at, created_by_id, name, avatar;

-- name: UpdateWorkspaceByID :one
UPDATE workspaces
SET
    name = CASE WHEN sqlc.arg(name_present)::boolean THEN sqlc.arg(name) ELSE name END,
    avatar = CASE WHEN sqlc.arg(avatar_present)::boolean THEN NULLIF(sqlc.arg(avatar), '') ELSE avatar END
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id, created_at, updated_at, deleted_at, created_by_id, name, avatar;

-- name: SoftDeleteWorkspaceByID :exec
UPDATE workspaces
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListWorkspaceHistories :many
SELECT h.id, h.entity_type, h.entity_id, h.changed_by_id, h.field, h.old_value, h.new_value, h.created_at,
       u.alias AS changed_by_alias, u.name AS changed_by_name, COALESCE(u.avatar, '') AS changed_by_avatar
FROM entity_update_histories h
LEFT JOIN users u ON u.id = h.changed_by_id
WHERE h.entity_type = 'workspace' AND h.entity_id = $1
ORDER BY h.created_at DESC, h.id DESC
LIMIT $2 OFFSET $3;

-- name: GetCallerWorkspaceRole :one
SELECT wu.role
FROM workspace_users wu
WHERE wu.workspace_id = $1 AND wu.user_id = $2
LIMIT 1;

-- name: ListWorkspaceMembers :many
SELECT
    wu.workspace_id,
    wu.user_id,
    u.alias,
    u.name,
    COALESCE(u.avatar, '') AS avatar,
    wu.role,
    'accepted'::text AS status,
    NULL::uuid AS invitation_id
FROM workspace_users wu
INNER JOIN users u ON u.id = wu.user_id
WHERE wu.workspace_id = $1
  AND u.deleted_at IS NULL
UNION ALL
SELECT
    wi.workspace_id,
    wi.invitee_id,
    u.alias,
    u.name,
    COALESCE(u.avatar, '') AS avatar,
    wi.role,
    'pending'::text AS status,
    wi.id AS invitation_id
FROM workspace_invitations wi
INNER JOIN users u ON u.id = wi.invitee_id
LEFT JOIN workspace_users wu ON wu.workspace_id = wi.workspace_id AND wu.user_id = wi.invitee_id
WHERE wi.workspace_id = $1
  AND u.deleted_at IS NULL
  AND wi.status = 'pending'
  AND wu.user_id IS NULL
ORDER BY status, role, alias
LIMIT $2 OFFSET $3;

-- name: UpsertWorkspaceUser :one
INSERT INTO workspace_users (workspace_id, user_id, role)
VALUES ($1, $2, $3::workspace_role)
ON CONFLICT (workspace_id, user_id) DO UPDATE
SET role = EXCLUDED.role
RETURNING workspace_id, user_id, role;

-- name: DeleteWorkspaceUser :exec
DELETE FROM workspace_users
WHERE workspace_id = $1 AND user_id = $2;

-- name: UpdateWorkspaceMemberRole :one
UPDATE workspace_users wu
SET role = $3::workspace_role
FROM users u
WHERE wu.user_id = u.id
  AND wu.workspace_id = $1
  AND wu.user_id = $2
  AND u.deleted_at IS NULL
RETURNING wu.workspace_id, wu.user_id, u.alias, u.name, COALESCE(u.avatar, ''), wu.role, 'accepted'::text, NULL::uuid;

-- name: ListMyWorkspaceInvitations :many
SELECT
    wi.id,
    wi.workspace_id,
    w.name AS workspace_name,
    wi.inviter_id,
    u.alias AS inviter_alias,
    u.name AS inviter_name,
    u.avatar AS inviter_avatar,
    wi.role,
    wi.status,
    wi.created_at,
    wi.updated_at
FROM workspace_invitations wi
INNER JOIN workspaces w ON w.id = wi.workspace_id
INNER JOIN users u ON u.id = wi.inviter_id
WHERE wi.invitee_id = $1 AND wi.status = 'pending'
ORDER BY wi.created_at DESC
LIMIT $2 OFFSET $3;

-- name: InviteWorkspaceMember :one
INSERT INTO workspace_invitations (workspace_id, invitee_id, inviter_id, role)
VALUES (sqlc.arg(workspace_id), sqlc.arg(invitee_id), sqlc.arg(inviter_id), sqlc.arg(role)::workspace_role)
ON CONFLICT (workspace_id, invitee_id) DO UPDATE
    SET role = EXCLUDED.role, status = 'pending', updated_at = NOW()
RETURNING id, workspace_id, invitee_id, inviter_id, role, status, created_at, updated_at;

-- name: GetPendingWorkspaceInvitationForAccept :one
SELECT wi.workspace_id, wi.role
FROM workspace_invitations wi
WHERE wi.id = $1
  AND wi.invitee_id = $2
  AND wi.status = 'pending'
LIMIT 1;

-- name: GetWorkspaceIDByInvitationID :one
SELECT wi.workspace_id
FROM workspace_invitations wi
WHERE wi.id = $1
LIMIT 1;

-- name: AcceptWorkspaceInvitation :exec
UPDATE workspace_invitations
SET status = 'accepted', updated_at = NOW()
WHERE id = $1;

-- name: RejectWorkspaceInvitation :exec
UPDATE workspace_invitations
SET status = 'rejected', updated_at = NOW()
WHERE workspace_invitations.id = $1
  AND workspace_invitations.invitee_id = $2
  AND workspace_invitations.status = 'pending';

-- name: CancelWorkspaceInvitation :exec
UPDATE workspace_invitations
SET status = 'canceled', updated_at = NOW()
WHERE workspace_invitations.id = $1
  AND workspace_invitations.status = 'pending';

-- name: CreateWorkspaceLink :one
WITH inserted AS (
    INSERT INTO workspace_links (workspace_id, title, url, created_by_id)
    VALUES (sqlc.arg(workspace_id), sqlc.arg(title), sqlc.arg(url), sqlc.arg(created_by_id))
    RETURNING id, created_at, updated_at, workspace_id, title, url, created_by_id
)
SELECT i.id, i.created_at, i.updated_at, i.workspace_id, i.title, i.url, i.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM inserted i
JOIN users u ON u.id = i.created_by_id;

-- name: ListWorkspaceLinks :many
SELECT l.id, l.created_at, l.updated_at, l.workspace_id, l.title, l.url, l.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM workspace_links l
JOIN users u ON u.id = l.created_by_id
WHERE l.workspace_id = $1
ORDER BY l.created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteWorkspaceLink :one
DELETE FROM workspace_links
WHERE workspace_links.id = $1
RETURNING workspace_id;

-- name: CreateWorkspaceNote :one
WITH inserted AS (
    INSERT INTO workspace_notes (workspace_id, title, content, created_by_id)
    VALUES (sqlc.arg(workspace_id), sqlc.arg(title), sqlc.arg(content), sqlc.arg(created_by_id))
    RETURNING id, created_at, updated_at, workspace_id, title, content, created_by_id
)
SELECT i.id, i.created_at, i.updated_at, i.workspace_id, i.title, i.content, i.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM inserted i
JOIN users u ON u.id = i.created_by_id;

-- name: ListWorkspaceNotes :many
SELECT n.id, n.created_at, n.updated_at, n.workspace_id, n.title, n.content, n.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM workspace_notes n
JOIN users u ON u.id = n.created_by_id
WHERE n.workspace_id = $1
ORDER BY n.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWorkspaceNote :one
WITH updated AS (
    UPDATE workspace_notes
    SET
        title = COALESCE(sqlc.narg(note_title)::text, workspace_notes.title),
        content = COALESCE(sqlc.narg(note_content)::text, workspace_notes.content)
    WHERE workspace_notes.id = sqlc.arg(id)
    RETURNING id, created_at, updated_at, workspace_id, title, content, created_by_id
)
SELECT u_n.id, u_n.created_at, u_n.updated_at, u_n.workspace_id, u_n.title, u_n.content, u_n.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM updated u_n
JOIN users u ON u.id = u_n.created_by_id;

-- name: DeleteWorkspaceNote :one
DELETE FROM workspace_notes
WHERE workspace_notes.id = $1
RETURNING workspace_id;

-- name: CreateWorkspaceFile :one
WITH inserted AS (
    INSERT INTO workspace_files (
        workspace_id,
        title,
        file_url,
        file_name,
        content_type,
        size_bytes,
        created_by_id
    )
    VALUES (
        sqlc.arg(workspace_id),
        sqlc.arg(title),
        sqlc.arg(file_url),
        sqlc.arg(file_name),
        sqlc.arg(content_type),
        sqlc.arg(size_bytes),
        sqlc.arg(created_by_id)
    )
    RETURNING id, created_at, updated_at, workspace_id, title, file_url, file_name, content_type, size_bytes, created_by_id
)
SELECT i.id, i.created_at, i.updated_at, i.workspace_id, i.title, i.file_url, i.file_name, i.content_type, i.size_bytes, i.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM inserted i
JOIN users u ON u.id = i.created_by_id;

-- name: ListWorkspaceFiles :many
SELECT f.id, f.created_at, f.updated_at, f.workspace_id, f.title, f.file_url, f.file_name, f.content_type, f.size_bytes, f.created_by_id,
       u.name AS creator_name, u.alias AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM workspace_files f
JOIN users u ON u.id = f.created_by_id
WHERE f.workspace_id = $1
ORDER BY f.created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteWorkspaceFile :one
DELETE FROM workspace_files
WHERE workspace_files.id = $1
RETURNING workspace_id;

-- name: ListWorkspaceMembersBirthdays :many
SELECT
    wu.workspace_id,
    u.id AS member_id,
    u.alias AS member_alias,
    u.name AS member_name,
    u.birthday
FROM workspace_users wu
INNER JOIN users u ON u.id = wu.user_id
WHERE wu.workspace_id = sqlc.arg(workspace_id)::uuid
  AND u.deleted_at IS NULL
  AND u.birthday IS NOT NULL
  AND (sqlc.narg(member_id)::uuid IS NULL OR u.id = sqlc.narg(member_id)::uuid);
