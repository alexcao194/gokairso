-- name: CreateLateLog :one
INSERT INTO late_logs (workspace_id, late_user_id, creator_id, content, late_time, estimate)
VALUES (sqlc.arg(workspace_id), sqlc.arg(late_user_id), sqlc.arg(creator_id), sqlc.arg(content), sqlc.arg(late_time), sqlc.arg(estimate))
RETURNING id, created_at, updated_at, workspace_id, late_user_id, creator_id, content, late_time, estimate;

-- name: ListLateLogs :many
SELECT
    ll.id, ll.created_at, ll.updated_at, ll.workspace_id,
    ll.late_user_id, u.alias AS late_user_alias, u.name AS late_user_name, u.avatar AS late_user_avatar,
    ll.creator_id, uc.alias AS creator_alias, uc.name AS creator_name, uc.avatar AS creator_avatar,
    ll.content, ll.late_time, ll.estimate
FROM late_logs ll
JOIN users u ON u.id = ll.late_user_id
JOIN users uc ON uc.id = ll.creator_id
WHERE ll.workspace_id = sqlc.arg(workspace_id)::uuid
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR ll.late_time >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR ll.late_time <= sqlc.narg(to_time)::timestamptz)
  AND (sqlc.narg(late_user_id)::uuid IS NULL OR ll.late_user_id = sqlc.narg(late_user_id)::uuid)
ORDER BY ll.late_time DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetLateLog :one
SELECT
    ll.id, ll.created_at, ll.updated_at, ll.workspace_id,
    ll.late_user_id, u.alias AS late_user_alias, u.name AS late_user_name, u.avatar AS late_user_avatar,
    ll.creator_id, uc.alias AS creator_alias, uc.name AS creator_name, uc.avatar AS creator_avatar,
    ll.content, ll.late_time, ll.estimate
FROM late_logs ll
JOIN users u ON u.id = ll.late_user_id
JOIN users uc ON uc.id = ll.creator_id
WHERE ll.id = $1;

-- name: UpdateLateLog :one
UPDATE late_logs
SET
    updated_at   = now(),
    content      = CASE WHEN sqlc.arg(content_present)::boolean THEN sqlc.arg(content)::text        ELSE content   END,
    late_time    = CASE WHEN sqlc.arg(late_time_present)::boolean THEN sqlc.arg(late_time)::timestamptz ELSE late_time END,
    estimate     = CASE WHEN sqlc.arg(estimate_present)::boolean THEN sqlc.arg(estimate)::float       ELSE estimate  END
WHERE id = $1
RETURNING id, created_at, updated_at, workspace_id, late_user_id, creator_id, content, late_time, estimate;

-- name: DeleteLateLog :exec
DELETE FROM late_logs WHERE id = $1;

-- name: SumLateLogEstimateByUser :many
SELECT ll.late_user_id AS user_id, COALESCE(SUM(ll.estimate), 0)::float AS total
FROM late_logs ll
WHERE ll.workspace_id = sqlc.arg(workspace_id)::uuid
  AND ll.late_time >= sqlc.arg('from')
  AND ll.late_time <= sqlc.arg('to')
  AND (sqlc.narg(member_id)::uuid IS NULL OR ll.late_user_id = sqlc.narg(member_id)::uuid)
GROUP BY ll.late_user_id;
