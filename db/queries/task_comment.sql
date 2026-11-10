-- name: CreateTaskComment :one
INSERT INTO task_comments (task_id, author_id, content)
VALUES (sqlc.arg(task_id), sqlc.arg(author_id), sqlc.arg(content))
RETURNING *;

-- name: ListTaskCommentsByTask :many
SELECT c.*, u.alias AS author_alias, u.name AS author_name, u.avatar AS author_avatar
FROM task_comments c
INNER JOIN users u ON u.id = c.author_id
WHERE c.task_id = sqlc.arg(task_id)
ORDER BY c.created_at;

-- name: ListTaskCommentAuthorIDs :many
SELECT DISTINCT author_id FROM task_comments WHERE task_id = sqlc.arg(task_id);

-- name: GetTaskCommentByID :one
SELECT c.*, u.alias AS author_alias, u.name AS author_name, u.avatar AS author_avatar
FROM task_comments c
INNER JOIN users u ON u.id = c.author_id
WHERE c.id = sqlc.arg(id);

-- name: UpdateTaskCommentByID :one
UPDATE task_comments SET content = sqlc.arg(content) WHERE id = sqlc.arg(id) RETURNING *;

-- name: DeleteTaskCommentByID :exec
DELETE FROM task_comments WHERE id = sqlc.arg(id);
