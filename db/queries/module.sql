-- name: CreateModule :one
INSERT INTO modules (project_id, name, start_date, end_date)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, updated_at, deleted_at, name, project_id, start_date, end_date, scrum_status, released_at;

-- name: ListModulesByProjectAliasWithStats :many
SELECT m.id, m.created_at, m.updated_at, m.deleted_at, m.name, m.project_id, m.start_date, m.end_date, m.scrum_status, m.released_at,
       COUNT(t.id)::int AS task_count,
       COALESCE(SUM(t.estimation), 0)::int AS total_estimation,
       (SELECT COUNT(*) FROM entity_attachments a WHERE a.entity_type = 'module' AND a.entity_id = m.id)::int AS attachment_count
FROM modules m
LEFT JOIN tasks t ON t.module_id = m.id AND t.deleted_at IS NULL
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR COALESCE(t.end_date, t.start_date, t.created_at) >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR COALESCE(t.start_date, t.end_date, t.created_at) <= sqlc.narg(to_time)::timestamptz)
WHERE m.project_id = $1 AND m.deleted_at IS NULL
GROUP BY m.id, m.created_at, m.updated_at, m.deleted_at, m.name, m.project_id, m.start_date, m.end_date, m.scrum_status, m.released_at
ORDER BY m.end_date ASC NULLS LAST, m.created_at, m.name
LIMIT $2 OFFSET $3;

-- name: GetModuleByID :one
SELECT id, created_at, updated_at, deleted_at, name, project_id, start_date, end_date, scrum_status, released_at
FROM modules
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetModuleByIDForUpdate :one
SELECT id, created_at, updated_at, deleted_at, name, project_id, start_date, end_date, scrum_status, released_at
FROM modules
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1
FOR UPDATE;

-- name: UpdateModuleByID :one
UPDATE modules
SET
    name = CASE WHEN sqlc.arg(name_present)::boolean THEN sqlc.arg(name) ELSE name END,
    start_date = CASE WHEN sqlc.arg(start_date_present)::boolean THEN sqlc.narg(start_date) ELSE start_date END,
    end_date = CASE WHEN sqlc.arg(end_date_present)::boolean THEN sqlc.narg(end_date) ELSE end_date END,
    scrum_status = CASE WHEN sqlc.arg(scrum_status_present)::boolean THEN NULLIF(sqlc.arg(scrum_status), '')::module_scrum_status ELSE scrum_status END,
    released_at = CASE WHEN sqlc.arg(released_at_present)::boolean THEN sqlc.narg(released_at) ELSE released_at END
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id, created_at, updated_at, deleted_at, name, project_id, start_date, end_date, scrum_status, released_at;

-- name: SoftDeleteModule :exec
UPDATE modules
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListModuleHistoriesByID :many
SELECT h.id, h.entity_type, h.entity_id, h.changed_by_id, h.field, h.old_value, h.new_value, h.created_at,
       u.alias AS changed_by_alias, u.name AS changed_by_name, COALESCE(u.avatar, '') AS changed_by_avatar
FROM entity_update_histories h
LEFT JOIN users u ON u.id = h.changed_by_id
WHERE h.entity_type = 'module' AND h.entity_id = $1
ORDER BY h.created_at DESC, h.id DESC
LIMIT $2 OFFSET $3;

-- name: ListModuleSprintsByModuleIDs :many
SELECT * FROM module_sprints
WHERE module_id = ANY(sqlc.arg(module_ids)::uuid[])
ORDER BY start_date, created_at;

-- name: GetModuleSprint :one
SELECT * FROM module_sprints WHERE id = $1;

-- name: CreateModuleSprint :one
INSERT INTO module_sprints (module_id, name, goal, start_date, end_date, status, review_note, retro_note)
VALUES (sqlc.arg(module_id), sqlc.arg(name), sqlc.arg(goal), sqlc.arg(start_date), sqlc.arg(end_date),
        sqlc.arg(status)::module_sprint_status, sqlc.arg(review_note), sqlc.arg(retro_note))
RETURNING *;

-- name: UpdateModuleSprint :one
UPDATE module_sprints
SET name = sqlc.arg(name), goal = sqlc.arg(goal), start_date = sqlc.arg(start_date), end_date = sqlc.arg(end_date),
    status = sqlc.arg(status)::module_sprint_status, review_note = sqlc.arg(review_note), retro_note = sqlc.arg(retro_note)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteModuleSprint :exec
DELETE FROM module_sprints WHERE id = $1;

-- name: SyncModuleStartFromSprints :exec
UPDATE modules
SET start_date = (SELECT (MIN(s.start_date)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh') FROM module_sprints s WHERE s.module_id = modules.id)
WHERE modules.id = $1 AND EXISTS (SELECT 1 FROM module_sprints s WHERE s.module_id = modules.id);

-- name: CheckModuleMembership :one
SELECT EXISTS (
    SELECT 1
    FROM modules m
    JOIN projects p ON p.id = m.project_id
    LEFT JOIN project_users pu ON pu.project_id = p.id AND pu.user_id = sqlc.arg(user_id)
    LEFT JOIN workspace_users wu ON wu.workspace_id = p.workspace_id AND wu.user_id = sqlc.arg(user_id)
    WHERE m.id = sqlc.arg(module_id) AND (pu.user_id IS NOT NULL OR wu.user_id IS NOT NULL)
)::boolean;
