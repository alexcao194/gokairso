-- name: GetKPITaskStats :many
WITH task_workers_total AS (
    SELECT task_id, SUM(weight) as total_weight
    FROM task_workers
    GROUP BY task_id
),
task_fault_holders_total AS (
    SELECT task_id, SUM(weight) as total_weight
    FROM task_fault_holders
    GROUP BY task_id
),
task_members AS (
    SELECT tw.task_id, tw.user_id,
           COALESCE(tw.weight::float8 / NULLIF(twt.total_weight, 0), 0) AS weight,
           'worker' AS role
    FROM task_workers tw
    JOIN task_workers_total twt ON tw.task_id = twt.task_id
    UNION ALL
    SELECT tfh.task_id, tfh.user_id,
           COALESCE(tfh.weight::float8 / NULLIF(tfht.total_weight, 0), 0) AS weight,
           'fault_holder' AS role
    FROM task_fault_holders tfh
    JOIN task_fault_holders_total tfht ON tfh.task_id = tfht.task_id
)
SELECT
    tm.user_id                                                                                             AS member_id,
    u.alias                                                                                                AS member_alias,
    u.name                                                                                                 AS member_name,
    u.avatar                                                                                               AS member_avatar,
    COALESCE(SUM(t.estimation * tm.weight) FILTER (
        WHERE tm.role = 'worker'
          AND (t.status IN ('review'::task_status, 'done'::task_status)
               OR (t.end_date IS NOT NULL AND t.end_date < NOW()))
    ), 0)::float8                                                                                          AS total_worker_estimation_reviewed_done,
    COALESCE(SUM(t.estimation * tm.weight) FILTER (
        WHERE tm.role = 'worker'
          AND (t.status IN ('review'::task_status, 'done'::task_status)
               OR (t.end_date IS NOT NULL AND t.end_date < NOW()))
          AND t.first_completed_at IS NOT NULL
          AND t.first_completed_at > t.end_date
    ), 0)::float8                                                                                          AS late_worker_estimation,
    COALESCE(SUM(t.estimation * tm.weight) FILTER (WHERE tm.role = 'worker'),       0)::float8             AS total_worker_estimation,
    COALESCE(SUM(t.estimation * tm.weight) FILTER (WHERE tm.role = 'fault_holder'), 0)::float8             AS total_bug_estimation,
    COALESCE(SUM(t.estimation * tm.weight * LEAST(t.return_count, 1)) FILTER (WHERE tm.role = 'worker'), 0)::float8  AS total_return_estimation
FROM task_members tm
INNER JOIN tasks      t  ON t.id  = tm.task_id    AND t.deleted_at IS NULL
INNER JOIN projects   p  ON p.id  = t.project_id  AND p.deleted_at IS NULL
INNER JOIN workspaces w  ON w.id  = p.workspace_id AND w.deleted_at IS NULL
INNER JOIN users      u  ON u.id  = tm.user_id    AND u.deleted_at IS NULL
WHERE w.id          = sqlc.arg(workspace_id)::uuid
  AND t.archived    = false
  AND (sqlc.narg(member_id)::uuid          IS NULL OR tm.user_id  = sqlc.narg(member_id)::uuid)
  AND (sqlc.narg('from')::timestamptz   IS NULL OR t.end_date >= sqlc.narg('from')::timestamptz)
  AND (sqlc.narg('to')::timestamptz   IS NULL OR t.end_date <= sqlc.narg('to')::timestamptz)
GROUP BY tm.user_id, u.alias, u.name, u.avatar
ORDER BY u.alias;

-- name: UpsertEmployeeMonthlyAttitude :one
INSERT INTO employee_monthly_attitudes (workspace_id, member_id, month, score, comment, updated_by_id)
VALUES (sqlc.arg(workspace_id), sqlc.arg(member_id)::uuid, sqlc.arg('month')::date, sqlc.arg(score)::float, sqlc.arg(comment), sqlc.arg(updated_by_id)::uuid)
ON CONFLICT (workspace_id, member_id, month) DO UPDATE
    SET score         = EXCLUDED.score,
        comment       = EXCLUDED.comment,
        updated_by_id = EXCLUDED.updated_by_id,
        updated_at    = now()
RETURNING id, workspace_id, member_id, month, score, comment, updated_by_id, created_at, updated_at;

-- name: GetEmployeeMonthlyAttitudes :many
SELECT ema.member_id, ema.month, ema.score, ema.comment, ema.updated_by_id
FROM employee_monthly_attitudes ema
WHERE ema.workspace_id = sqlc.arg(workspace_id)::uuid
  AND (sqlc.narg(member_id)::uuid IS NULL OR ema.member_id = sqlc.narg(member_id)::uuid)
  AND ema.month >= sqlc.arg('from')::date
  AND ema.month <= sqlc.arg('to')::date;

-- name: GetKPISettingsByAlias :one
SELECT
    s.workspace_id,
    s.created_at,
    s.updated_at,
    s.customer_weight,
    s.progress_weight,
    s.quality_weight,
    s.initiative_weight,
    s.punctuality_weight,
    s.attitude_weight
FROM workspace_kpi_settings s
WHERE s.workspace_id = $1
LIMIT 1;

-- name: UpsertKPISettingsByAlias :one
INSERT INTO workspace_kpi_settings (
    workspace_id,
    customer_weight,
    progress_weight,
    quality_weight,
    initiative_weight,
    punctuality_weight,
    attitude_weight
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(customer_weight),
    sqlc.arg(progress_weight),
    sqlc.arg(quality_weight),
    sqlc.arg(initiative_weight),
    sqlc.arg(punctuality_weight),
    sqlc.arg(attitude_weight)
)
ON CONFLICT (workspace_id) DO UPDATE SET
    customer_weight = EXCLUDED.customer_weight,
    progress_weight = EXCLUDED.progress_weight,
    quality_weight = EXCLUDED.quality_weight,
    initiative_weight = EXCLUDED.initiative_weight,
    punctuality_weight = EXCLUDED.punctuality_weight,
    attitude_weight = EXCLUDED.attitude_weight
RETURNING
    workspace_id,
    created_at,
    updated_at,
    customer_weight,
    progress_weight,
    quality_weight,
    initiative_weight,
    punctuality_weight,
    attitude_weight;

-- name: ListMemberTaskBreakdown :many
SELECT t.id, t.title, t.project_id, p.name AS project_name, t.sequence_no,
       t.status, t.priority, t.start_date, t.end_date, t.first_completed_at,
       t.return_count, t.estimation,
       EXISTS (SELECT 1 FROM task_workers tw WHERE tw.task_id = t.id AND tw.user_id = sqlc.arg(member_id)) AS is_worker,
       EXISTS (SELECT 1 FROM task_fault_holders tf WHERE tf.task_id = t.id AND tf.user_id = sqlc.arg(member_id)) AS is_fault_holder
FROM tasks t
JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
JOIN workspaces w ON w.id = p.workspace_id AND w.deleted_at IS NULL
WHERE w.id = sqlc.arg(workspace_id)::uuid
  AND t.deleted_at IS NULL
  AND (EXISTS (SELECT 1 FROM task_workers tw WHERE tw.task_id = t.id AND tw.user_id = sqlc.arg(member_id))
       OR EXISTS (SELECT 1 FROM task_fault_holders tf WHERE tf.task_id = t.id AND tf.user_id = sqlc.arg(member_id)))
  AND COALESCE(t.end_date, t.start_date, t.created_at) >= sqlc.arg(from_time)::timestamptz
  AND COALESCE(t.start_date, t.end_date, t.created_at) <= sqlc.arg(to_time)::timestamptz
ORDER BY t.end_date DESC NULLS LAST, t.created_at DESC;

-- name: GetWorkspaceHolderName :one
SELECT u.name
FROM workspace_users wu
JOIN users u ON u.id = wu.user_id
WHERE wu.workspace_id = sqlc.arg(workspace_id)::uuid
  AND wu.role = 'holder'
LIMIT 1;
