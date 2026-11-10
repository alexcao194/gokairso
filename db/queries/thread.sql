-- name: ListThreads :many
SELECT
    t.id, t.created_at, t.updated_at, t.deleted_at, t.project_id, t.module_id, t.created_by_id, t.parent_id,
    t.title, t.description, t.status, t.priority, t.start_date, t.end_date, t.first_completed_at, t.estimation,
    t.archived, t.return_count, t.fault_holder_id, t.sequence_no,
    EXISTS (
        SELECT 1 FROM task_comment_mentions m WHERE m.task_id = t.id AND m.user_id = sqlc.arg(user_id)
    ) AS mentioned,
    lc.author_id AS latest_comment_author_id,
    lc.author_alias AS latest_comment_author_alias,
    lc.author_name AS latest_comment_author_name,
    lc.author_avatar AS latest_comment_author_avatar,
    lc.content AS latest_comment_content,
    lc.created_at AS latest_comment_created_at
FROM tasks t
INNER JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
LEFT JOIN LATERAL (
    SELECT c.author_id, u.alias AS author_alias, u.name AS author_name, u.avatar AS author_avatar,
           c.content, c.created_at
    FROM task_comments c
    INNER JOIN users u ON u.id = c.author_id
    WHERE c.task_id = t.id
    ORDER BY c.created_at DESC
    LIMIT 1
) lc ON true
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND t.deleted_at IS NULL
  AND (
    EXISTS (SELECT 1 FROM task_workers tw WHERE tw.task_id = t.id AND tw.user_id = sqlc.arg(user_id))
    OR EXISTS (SELECT 1 FROM task_comment_mentions m2 WHERE m2.task_id = t.id AND m2.user_id = sqlc.arg(user_id))
  )
  AND EXISTS (
    SELECT 1 FROM task_comments c2
    WHERE c2.task_id = t.id
      AND c2.author_id != sqlc.arg(user_id)
      AND c2.created_at > COALESCE(
        (SELECT r.last_read_at FROM task_comment_reads r WHERE r.task_id = t.id AND r.user_id = sqlc.arg(user_id)),
        '-infinity'::timestamptz
      )
  )
ORDER BY lc.created_at DESC NULLS LAST
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: CountThreads :one
SELECT COUNT(*) FROM tasks t
INNER JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
WHERE p.workspace_id = sqlc.arg(workspace_id)
  AND t.deleted_at IS NULL
  AND (
    EXISTS (SELECT 1 FROM task_workers tw WHERE tw.task_id = t.id AND tw.user_id = sqlc.arg(user_id))
    OR EXISTS (SELECT 1 FROM task_comment_mentions m2 WHERE m2.task_id = t.id AND m2.user_id = sqlc.arg(user_id))
  )
  AND EXISTS (
    SELECT 1 FROM task_comments c2
    WHERE c2.task_id = t.id
      AND c2.author_id != sqlc.arg(user_id)
      AND c2.created_at > COALESCE(
        (SELECT r.last_read_at FROM task_comment_reads r WHERE r.task_id = t.id AND r.user_id = sqlc.arg(user_id)),
        '-infinity'::timestamptz
      )
  );
