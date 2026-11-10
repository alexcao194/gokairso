-- name: CreateTaskCommentMention :exec
INSERT INTO task_comment_mentions (comment_id, task_id, user_id)
VALUES (sqlc.arg(comment_id), sqlc.arg(task_id), sqlc.arg(user_id))
ON CONFLICT (comment_id, user_id) DO NOTHING;

-- name: DeleteTaskCommentMentionsByComment :exec
DELETE FROM task_comment_mentions WHERE comment_id = sqlc.arg(comment_id);
