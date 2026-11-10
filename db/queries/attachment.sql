-- name: CreateEntityAttachment :one
INSERT INTO entity_attachments (entity_type, entity_id, uploader_id, file_name, file_url, content_type, size_bytes)
VALUES (sqlc.arg(entity_type), sqlc.arg(entity_id), sqlc.arg(uploader_id), sqlc.arg(file_name), sqlc.arg(file_url), sqlc.narg(content_type), sqlc.narg(size_bytes))
RETURNING *;

-- name: ListEntityAttachmentsByEntity :many
SELECT a.*, u.alias AS uploader_alias, u.name AS uploader_name
FROM entity_attachments a
INNER JOIN users u ON u.id = a.uploader_id
WHERE a.entity_type = sqlc.arg(entity_type) AND a.entity_id = sqlc.arg(entity_id)
ORDER BY a.created_at;

-- name: DeleteEntityAttachment :exec
DELETE FROM entity_attachments WHERE id = sqlc.arg(id) AND entity_type = sqlc.arg(entity_type);

-- name: DeleteEntityAttachmentsByEntity :exec
DELETE FROM entity_attachments WHERE entity_type = sqlc.arg(entity_type) AND entity_id = sqlc.arg(entity_id);
