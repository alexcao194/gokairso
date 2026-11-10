-- name: CreateTask :one
INSERT INTO tasks (
    project_id, module_id, created_by_id, parent_id,
    title, description, status, priority, start_date, end_date, first_completed_at, estimation, archived, return_count, fault_holder_id, sequence_no
) VALUES (
    sqlc.arg(project_id), sqlc.narg(module_id), sqlc.arg(created_by_id), sqlc.narg(parent_id),
    sqlc.arg(title), NULLIF(sqlc.arg(description), ''), sqlc.arg(status)::task_status, sqlc.arg(priority), sqlc.narg(start_date), sqlc.narg(end_date),
    CASE WHEN sqlc.arg(status)::task_status IN ('review', 'done') THEN NOW() ELSE NULL END,
    sqlc.arg(estimation), sqlc.arg(archived), 0, sqlc.narg(fault_holder_id), sqlc.arg(sequence_no)
)
RETURNING id, created_at, updated_at, deleted_at, project_id, module_id, created_by_id, parent_id,
          title, description, status, priority, start_date, end_date, first_completed_at, estimation, archived, return_count, fault_holder_id, sequence_no;

-- name: GetTaskByID :one
SELECT id, created_at, updated_at, deleted_at, project_id, module_id, created_by_id, parent_id,
       title, description, status, priority, start_date, end_date, first_completed_at, estimation, archived, return_count, fault_holder_id, sequence_no
FROM tasks
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetTaskByIDForUpdate :one
SELECT id, created_at, updated_at, deleted_at, project_id, module_id, created_by_id, parent_id,
       title, description, status, priority, start_date, end_date, first_completed_at, estimation, archived, return_count, fault_holder_id, sequence_no
FROM tasks
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1
FOR UPDATE;

-- name: CheckTaskAuth :one
SELECT EXISTS (
    SELECT 1
    FROM tasks t
    LEFT JOIN projects p ON p.id = t.project_id
    LEFT JOIN project_users pu ON pu.project_id = p.id AND pu.user_id = sqlc.arg(user_id) AND pu.role IN ('admin', 'holder')
    LEFT JOIN workspace_users wu ON wu.workspace_id = p.workspace_id AND wu.user_id = sqlc.arg(user_id) AND wu.role IN ('admin', 'holder')
    WHERE t.id = sqlc.arg(task_id) AND (pu.user_id IS NOT NULL OR wu.user_id IS NOT NULL)
)::boolean;

-- name: UpdateTask :one
UPDATE tasks
SET
    module_id = CASE WHEN sqlc.arg(module_id_present)::boolean THEN sqlc.narg(module_id) ELSE module_id END,
    parent_id = CASE WHEN sqlc.arg(parent_id_present)::boolean THEN sqlc.narg(parent_id) ELSE parent_id END,
    title = CASE WHEN sqlc.arg(title_present)::boolean THEN sqlc.arg(title) ELSE title END,
    description = CASE WHEN sqlc.arg(description_present)::boolean THEN NULLIF(sqlc.arg(description), '') ELSE description END,
    status = CASE WHEN sqlc.arg(status_present)::boolean THEN NULLIF(sqlc.arg(status), '')::task_status ELSE status END,
    priority = CASE WHEN sqlc.arg(priority_present)::boolean THEN sqlc.arg(priority) ELSE priority END,
    start_date = CASE WHEN sqlc.arg(start_date_present)::boolean THEN sqlc.narg(start_date) ELSE start_date END,
    end_date = CASE WHEN sqlc.arg(end_date_present)::boolean THEN sqlc.narg(end_date) ELSE end_date END,
    first_completed_at = CASE
        WHEN first_completed_at IS NULL
         AND sqlc.arg(status_present)::boolean
         AND NULLIF(sqlc.arg(status), '')::task_status IN ('review', 'done')
         AND status NOT IN ('review', 'done')
        THEN NOW()
        ELSE first_completed_at
    END,
    estimation = CASE WHEN sqlc.arg(estimation_present)::boolean THEN sqlc.arg(estimation) ELSE estimation END,
    archived = CASE WHEN sqlc.arg(archived_present)::boolean THEN sqlc.arg(archived) ELSE archived END,
    return_count = CASE WHEN sqlc.arg(return_count_present)::boolean THEN sqlc.arg(return_count) ELSE return_count END,
    fault_holder_id = CASE WHEN sqlc.arg(fault_holder_id_present)::boolean THEN sqlc.narg(fault_holder_id) ELSE fault_holder_id END
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id, created_at, updated_at, deleted_at, project_id, module_id, created_by_id, parent_id,
          title, description, status, priority, start_date, end_date, first_completed_at, estimation, archived, return_count, fault_holder_id, sequence_no;

-- name: SoftDeleteTask :exec
UPDATE tasks
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: InsertEntityHistory :exec
INSERT INTO entity_update_histories (entity_type, entity_id, changed_by_id, field, old_value, new_value)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListTaskHistories :many
SELECT h.id, h.entity_id, h.changed_by_id, h.field, h.old_value, h.new_value, h.created_at,
       u.alias AS changed_by_alias, u.name AS changed_by_name, COALESCE(u.avatar, '') AS changed_by_avatar
FROM entity_update_histories h
INNER JOIN tasks t ON t.id = h.entity_id
LEFT JOIN users u ON u.id = h.changed_by_id
WHERE h.entity_type = 'task' AND t.id = $1 AND t.deleted_at IS NULL
ORDER BY h.created_at DESC, h.id DESC
LIMIT $2 OFFSET $3;

-- name: CreateTaskLabel :one
INSERT INTO task_labels (project_id, created_by_id, name, color)
VALUES ($1, $2, $3, $4)
RETURNING id, created_at, updated_at, deleted_at, created_by_id, name, color, project_id;

-- name: ListTaskLabelsByProject :many
SELECT l.id, l.created_at, l.updated_at, l.deleted_at, l.created_by_id, l.name, l.color, l.project_id
FROM task_labels l
WHERE l.project_id = $1 AND l.deleted_at IS NULL
ORDER BY l.name;

-- name: ListTaskLabelsByProjectIDs :many
SELECT l.id, l.created_at, l.updated_at, l.deleted_at, l.created_by_id, l.name, l.color, l.project_id
FROM task_labels l
WHERE l.project_id = ANY($1::uuid[]) AND l.deleted_at IS NULL
ORDER BY l.name;

-- name: ListTaskLabelsByTask :many
SELECT l.id, l.created_at, l.updated_at, l.deleted_at, l.created_by_id, l.name, l.color, l.project_id
FROM task_label_tasks tlt
INNER JOIN task_labels l ON l.id = tlt.label_id AND l.deleted_at IS NULL
WHERE tlt.task_id = $1
ORDER BY l.name;

-- name: ListTaskLabelsByTaskIDs :many
SELECT tlt.task_id, l.id, l.created_at, l.updated_at, l.deleted_at, l.created_by_id, l.name, l.color, l.project_id
FROM task_label_tasks tlt
INNER JOIN task_labels l ON l.id = tlt.label_id AND l.deleted_at IS NULL
WHERE tlt.task_id = ANY($1::uuid[])
ORDER BY l.name;

-- name: UpdateTaskLabel :one
UPDATE task_labels
SET name = $2, color = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING id, created_at, updated_at, deleted_at, created_by_id, name, color, project_id;

-- name: SoftDeleteTaskLabel :one
UPDATE task_labels
SET deleted_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
RETURNING project_id;

-- name: AttachTaskLabel :exec
INSERT INTO task_label_tasks (task_id, label_id)
VALUES ($1, $2)
ON CONFLICT (task_id, label_id) DO NOTHING;

-- name: DetachTaskLabel :exec
DELETE FROM task_label_tasks
WHERE task_id = $1 AND label_id = $2;

-- name: ListTaskWorkersWithUser :many
SELECT tw.task_id, tw.weight,
       u.id AS user_id, u.name, u.alias, u.avatar
FROM task_workers tw
INNER JOIN users u ON u.id = tw.user_id AND u.deleted_at IS NULL
WHERE tw.task_id = $1
ORDER BY u.alias ASC
LIMIT $2 OFFSET $3;

-- name: ListTaskWorkersWithUserByTaskIDs :many
SELECT tw.task_id, tw.weight,
       u.id AS user_id, u.name, u.alias, u.avatar
FROM task_workers tw
INNER JOIN users u ON u.id = tw.user_id AND u.deleted_at IS NULL
WHERE tw.task_id = ANY($1::uuid[])
ORDER BY u.alias ASC;

-- name: AssignTaskWorker :one
WITH inserted AS (
    INSERT INTO task_workers (task_id, user_id, weight)
    VALUES ($1, $2, $3)
    ON CONFLICT (task_id, user_id) DO UPDATE SET weight = EXCLUDED.weight
    RETURNING task_id, weight, user_id
)
SELECT i.task_id, i.weight,
       u.id AS user_id, u.name, u.alias, u.avatar
FROM inserted i
JOIN users u ON i.user_id = u.id;

-- name: RemoveTaskWorker :exec
DELETE FROM task_workers
WHERE task_id = $1 AND user_id = $2;

-- name: ListTaskFaultHoldersWithUser :many
SELECT tw.task_id, tw.weight,
       u.id AS user_id, u.name, u.alias, u.avatar
FROM task_fault_holders tw
INNER JOIN users u ON u.id = tw.user_id AND u.deleted_at IS NULL
WHERE tw.task_id = $1
ORDER BY u.alias ASC
LIMIT $2 OFFSET $3;

-- name: ListTaskFaultHoldersWithUserByTaskIDs :many
SELECT tw.task_id, tw.weight,
       u.id AS user_id, u.name, u.alias, u.avatar
FROM task_fault_holders tw
INNER JOIN users u ON u.id = tw.user_id AND u.deleted_at IS NULL
WHERE tw.task_id = ANY($1::uuid[])
ORDER BY u.alias ASC;

-- name: AssignTaskFaultHolder :one
WITH inserted AS (
    INSERT INTO task_fault_holders (task_id, user_id, weight)
    VALUES ($1, $2, $3)
    ON CONFLICT (task_id, user_id) DO UPDATE SET weight = EXCLUDED.weight
    RETURNING task_id, weight, user_id
)
SELECT i.task_id, i.weight,
       u.id AS user_id, u.name, u.alias, u.avatar
FROM inserted i
JOIN users u ON i.user_id = u.id;

-- name: RemoveTaskFaultHolder :exec
DELETE FROM task_fault_holders
WHERE task_id = $1 AND user_id = $2;

-- name: CountMyActiveTasksByWorkspace :one
SELECT COUNT(*)::bigint
FROM tasks t
INNER JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
INNER JOIN task_workers tw ON tw.task_id = t.id AND tw.user_id = sqlc.arg(user_id)::uuid
WHERE p.workspace_id = sqlc.arg(workspace_id)::uuid
  AND t.deleted_at IS NULL
  AND t.archived = FALSE
  AND t.status IN ('todo'::task_status, 'in_progress'::task_status);

-- name: GetWorkspaceIDByProjectID :one
SELECT workspace_id
FROM projects
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1;

-- name: GetProjectNameByID :one
SELECT name
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
LIMIT 1;

-- name: GetProjectNamesByIDs :many
SELECT id, name
FROM projects
WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;

-- name: CountFeedbacksByTaskID :one
SELECT COUNT(*)::int
FROM project_feedback_tasks
WHERE task_id = sqlc.arg(task_id);

-- name: CountFeedbacksByTaskIDs :many
SELECT task_id, COUNT(*)::int AS count
FROM project_feedback_tasks
WHERE task_id = ANY(sqlc.arg(task_ids)::uuid[])
GROUP BY task_id;

-- name: IncrementProjectTaskSequence :one
UPDATE projects
SET task_sequence_counter = task_sequence_counter + 1
WHERE id = sqlc.arg(id)
RETURNING task_sequence_counter;

-- name: GetTaskStatsByWorkspaceAlias :many
WITH task_workers_total AS (
    SELECT task_id, SUM(weight) AS total_weight
    FROM task_workers
    GROUP BY task_id
),
task_fault_holders_total AS (
    SELECT task_id, SUM(weight) AS total_weight
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
    tm.user_id AS member_id,
    u.alias AS member_alias,
    u.name AS member_name,
    u.avatar AS member_avatar,
    COUNT(DISTINCT t.id) AS total_tasks,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'backlog'::task_status) AS backlog_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'todo'::task_status) AS todo_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'in_progress'::task_status) AS in_progress_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'review'::task_status) AS review_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'done'::task_status) AS done_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.status = 'canceled'::task_status) AS canceled_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.priority = 1) AS priority1_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.priority = 2) AS priority2_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.priority = 3) AS priority3_count,
    COUNT(DISTINCT t.id) FILTER (WHERE t.priority = 4) AS priority4_count,
    COALESCE(SUM(t.estimation * tm.weight) FILTER (WHERE tm.role = 'worker' AND (t.status IN ('review'::task_status, 'done'::task_status) OR (t.end_date IS NOT NULL AND t.end_date < NOW()))), 0)::float8 AS total_estimation,
    COUNT(DISTINCT t.id) FILTER (WHERE t.first_completed_at IS NOT NULL AND t.end_date IS NOT NULL AND t.first_completed_at > t.end_date) AS total_overdue,
    COALESCE(SUM(t.estimation * tm.weight) FILTER (WHERE tm.role = 'worker' AND t.first_completed_at IS NOT NULL AND t.end_date IS NOT NULL AND t.first_completed_at > t.end_date), 0)::float8 AS total_late,
    COALESCE(SUM(t.estimation * tm.weight) FILTER (WHERE tm.role = 'fault_holder'), 0)::float8 AS total_bug
FROM task_members tm
INNER JOIN tasks t ON t.id = tm.task_id AND t.deleted_at IS NULL
INNER JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
INNER JOIN workspaces w ON w.id = p.workspace_id AND w.deleted_at IS NULL
INNER JOIN users u ON u.id = tm.user_id AND u.deleted_at IS NULL
WHERE w.id = sqlc.arg(workspace_id)::uuid
  AND (sqlc.narg(member_id)::uuid IS NULL OR tm.user_id = sqlc.narg(member_id)::uuid)
  AND (sqlc.narg(project_id)::uuid IS NULL OR p.id = sqlc.narg(project_id)::uuid)
  AND (sqlc.narg(module_id)::uuid IS NULL OR t.module_id = sqlc.narg(module_id)::uuid)
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR COALESCE(t.end_date, t.start_date, t.created_at) >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR COALESCE(t.start_date, t.end_date, t.created_at) <= sqlc.narg(to_time)::timestamptz)
GROUP BY tm.user_id, u.alias, u.name, u.avatar
ORDER BY u.alias;

-- name: GetTaskCycleTimeByWorkspaceAlias :one
SELECT
    COALESCE(AVG(EXTRACT(EPOCH FROM (t.first_completed_at - t.created_at)) / 86400.0), 0)::float8 AS avg_cycle_days,
    COUNT(*) AS sample_size
FROM tasks t
INNER JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
INNER JOIN workspaces w ON w.id = p.workspace_id AND w.deleted_at IS NULL
WHERE w.id = sqlc.arg(workspace_id)::uuid
  AND t.deleted_at IS NULL
  AND t.first_completed_at IS NOT NULL
  AND t.first_completed_at >= t.created_at
  AND (sqlc.narg(project_id)::uuid IS NULL OR p.id = sqlc.narg(project_id)::uuid)
  AND (sqlc.narg(module_id)::uuid IS NULL OR t.module_id = sqlc.narg(module_id)::uuid)
  AND (sqlc.narg(member_id)::uuid IS NULL OR EXISTS (
        SELECT 1 FROM task_workers tw WHERE tw.task_id = t.id AND tw.user_id = sqlc.narg(member_id)::uuid
        UNION ALL
        SELECT 1 FROM task_fault_holders tfh WHERE tfh.task_id = t.id AND tfh.user_id = sqlc.narg(member_id)::uuid
      ))
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR COALESCE(t.end_date, t.start_date, t.created_at) >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR COALESCE(t.start_date, t.end_date, t.created_at) <= sqlc.narg(to_time)::timestamptz);

-- name: ClaimDueTaskDeadlineReminders :many
WITH due AS (
    SELECT t.id AS task_id, t.end_date, t.title, t.project_id, p.workspace_id
    FROM tasks t
    INNER JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
    WHERE t.deleted_at IS NULL
      AND t.archived = FALSE
      AND t.status NOT IN ('done'::task_status, 'canceled'::task_status)
      AND t.end_date BETWEEN sqlc.arg(window_start)::timestamptz AND sqlc.arg(window_end)::timestamptz
),
claimed AS (
    INSERT INTO task_deadline_reminders (task_id, end_date)
    SELECT task_id, end_date FROM due
    ON CONFLICT (task_id, end_date) DO NOTHING
    RETURNING task_id, end_date
)
SELECT due.task_id, due.end_date, due.title, due.project_id, due.workspace_id
FROM due
JOIN claimed USING (task_id, end_date);

-- name: ListTaskWorkerUserIDs :many
SELECT user_id
FROM task_workers
WHERE task_id = $1;

-- name: CheckTaskMembership :one
SELECT EXISTS (
    SELECT 1
    FROM tasks t
    JOIN projects p ON p.id = t.project_id
    LEFT JOIN project_users pu ON pu.project_id = p.id AND pu.user_id = sqlc.arg(user_id)
    LEFT JOIN workspace_users wu ON wu.workspace_id = p.workspace_id AND wu.user_id = sqlc.arg(user_id)
    WHERE t.id = sqlc.arg(task_id) AND (pu.user_id IS NOT NULL OR wu.user_id IS NOT NULL)
)::boolean;

-- name: CheckLabelMembership :one
SELECT EXISTS (
    SELECT 1
    FROM task_labels l
    JOIN projects p ON p.id = l.project_id
    LEFT JOIN project_users pu ON pu.project_id = p.id AND pu.user_id = sqlc.arg(user_id)
    LEFT JOIN workspace_users wu ON wu.workspace_id = p.workspace_id AND wu.user_id = sqlc.arg(user_id)
    WHERE l.id = sqlc.arg(label_id) AND (pu.user_id IS NOT NULL OR wu.user_id IS NOT NULL)
)::boolean;
