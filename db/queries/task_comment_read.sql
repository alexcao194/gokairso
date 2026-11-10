-- name: MarkTaskCommentsRead :exec
INSERT INTO task_comment_reads (task_id, user_id, last_read_at)
VALUES (sqlc.arg(task_id), sqlc.arg(user_id), NOW())
ON CONFLICT (task_id, user_id) DO UPDATE SET last_read_at = NOW();

-- name: CountUnreadTaskComments :one
SELECT COUNT(*) FROM task_comments c
WHERE c.task_id = sqlc.arg(task_id)
  AND c.author_id != sqlc.arg(user_id)
  AND c.created_at > COALESCE(
    (SELECT r.last_read_at FROM task_comment_reads r WHERE r.task_id = sqlc.arg(task_id) AND r.user_id = sqlc.arg(user_id)),
    '-infinity'::timestamptz
  );

-- name: CountUnreadTaskCommentsByTaskIDs :many
SELECT c.task_id, COUNT(*)::int AS count
FROM task_comments c
LEFT JOIN task_comment_reads r ON r.task_id = c.task_id AND r.user_id = sqlc.arg(user_id)
WHERE c.task_id = ANY(sqlc.arg(task_ids)::uuid[])
  AND c.author_id != sqlc.arg(user_id)
  AND c.created_at > COALESCE(r.last_read_at, '-infinity'::timestamptz)
GROUP BY c.task_id;
