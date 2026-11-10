-- name: ListPublicModulesByProjectID :many
SELECT id, name, scrum_status, start_date, end_date, released_at
FROM modules
WHERE project_id = $1 AND deleted_at IS NULL
ORDER BY end_date ASC NULLS LAST, created_at, name;

-- name: ListPublicSprintsByModuleIDs :many
SELECT module_id, name, start_date, end_date, status
FROM module_sprints
WHERE module_id = ANY(sqlc.arg(module_ids)::uuid[])
ORDER BY start_date, created_at;
