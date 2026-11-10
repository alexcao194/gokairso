-- name: ListNotifications :many
SELECT id, created_at, updated_at, workspace_id, user_id, title, body, data, read_at
FROM notifications
WHERE workspace_id = $1 AND user_id = $2
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: ReadAllNotifications :exec
UPDATE notifications
SET read_at = COALESCE(read_at, NOW())
WHERE workspace_id = $1 AND user_id = $2;

-- name: ReadNotificationsByIDs :exec
UPDATE notifications
SET read_at = COALESCE(read_at, NOW())
WHERE workspace_id = $1 AND user_id = $2 AND id = ANY($3::uuid[]);

-- name: CreateNotification :one
INSERT INTO notifications (workspace_id, user_id, title, body, data)
VALUES ($1, $2, $3, $4, $5::jsonb)
RETURNING id, created_at, updated_at, workspace_id, user_id, title, body, data, read_at;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications
WHERE workspace_id = $1 AND user_id = $2 AND read_at IS NULL;
