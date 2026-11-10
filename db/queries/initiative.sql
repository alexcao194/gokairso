-- name: CreateInitiative :one
INSERT INTO initiatives (workspace_id, proposer_id, ref_member_id, content, estimate)
VALUES (sqlc.arg(workspace_id), sqlc.arg(proposer_id), sqlc.arg(ref_member_id), sqlc.arg(content), sqlc.arg(estimate))
RETURNING id, created_at, updated_at, workspace_id, proposer_id, ref_member_id, content, status, changes_after, earliest_approval_date, approved_at, estimate;

-- name: ListInitiatives :many
SELECT
    i.id, i.created_at, i.updated_at, i.workspace_id,
    i.proposer_id, u.alias AS proposer_alias, u.name AS proposer_name, u.avatar AS proposer_avatar,
    i.ref_member_id, ur.alias AS ref_member_alias, ur.name AS ref_member_name, ur.avatar AS ref_member_avatar,
    i.content, i.status, i.changes_after, i.earliest_approval_date, i.approved_at, i.estimate
FROM initiatives i
JOIN users u ON u.id = i.proposer_id
JOIN users ur ON ur.id = i.ref_member_id
WHERE i.workspace_id = sqlc.arg(workspace_id)::uuid
  AND (NULLIF(sqlc.arg(status)::text, '') IS NULL OR i.status = NULLIF(sqlc.arg(status)::text, ''))
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR i.created_at >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR i.created_at <= sqlc.narg(to_time)::timestamptz)
  AND (sqlc.narg(proposer_id)::uuid IS NULL OR i.proposer_id = sqlc.narg(proposer_id)::uuid)
  AND (sqlc.narg(ref_member_id)::uuid IS NULL OR i.ref_member_id = sqlc.narg(ref_member_id)::uuid)
ORDER BY i.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: GetInitiative :one
SELECT
    i.id, i.created_at, i.updated_at, i.workspace_id,
    i.proposer_id, u.alias AS proposer_alias, u.name AS proposer_name, u.avatar AS proposer_avatar,
    i.ref_member_id, ur.alias AS ref_member_alias, ur.name AS ref_member_name, ur.avatar AS ref_member_avatar,
    i.content, i.status, i.changes_after, i.earliest_approval_date, i.approved_at, i.estimate
FROM initiatives i
JOIN users u ON u.id = i.proposer_id
JOIN users ur ON ur.id = i.ref_member_id
WHERE i.id = $1;

-- name: UpdateInitiative :one
UPDATE initiatives
SET
    updated_at             = now(),
    content                = CASE WHEN sqlc.arg(content_present)::boolean THEN sqlc.arg(content)::text    ELSE content                END,
    status                 = CASE WHEN sqlc.arg(status_present)::boolean THEN sqlc.arg(status)::text    ELSE status                 END,
    changes_after          = CASE WHEN sqlc.arg(changes_after_present)::boolean THEN sqlc.arg(changes_after)::text    ELSE changes_after          END,
    earliest_approval_date = CASE WHEN sqlc.arg(earliest_date_present)::boolean THEN sqlc.arg(earliest_approval_date)::date    ELSE earliest_approval_date END,
    approved_at            = CASE WHEN sqlc.arg(approved_at_present)::boolean THEN sqlc.arg(approved_at)::timestamptz ELSE approved_at      END,
    estimate               = CASE WHEN sqlc.arg(estimate_present)::boolean THEN sqlc.arg(estimate)::float  ELSE estimate              END,
    ref_member_id          = CASE WHEN sqlc.arg(ref_member_present)::boolean THEN sqlc.arg(ref_member_id)::uuid  ELSE ref_member_id         END
WHERE id = $1
RETURNING id, created_at, updated_at, workspace_id, proposer_id, ref_member_id, content, status, changes_after, earliest_approval_date, approved_at, estimate;

-- name: DeleteInitiative :exec
DELETE FROM initiatives WHERE id = $1;

-- name: ListWorkspaceAdminHolderIDs :many
SELECT wu.user_id
FROM workspace_users wu
WHERE wu.workspace_id = sqlc.arg(workspace_id)::uuid AND wu.role IN ('admin', 'holder');

-- name: SumInitiativeEstimateByRefMember :many
SELECT i.ref_member_id AS user_id, COALESCE(SUM(i.estimate), 0)::float AS total
FROM initiatives i
WHERE i.workspace_id = sqlc.arg(workspace_id)::uuid
  AND i.approved_at >= sqlc.arg('from')
  AND i.approved_at <= sqlc.arg('to')
  AND (sqlc.narg(member_id)::uuid IS NULL OR i.ref_member_id = sqlc.narg(member_id)::uuid)
GROUP BY i.ref_member_id;
