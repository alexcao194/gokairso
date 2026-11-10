-- name: GetProjectFeedbackSettingsByProjectID :one
SELECT project_id, token, enabled, show_module_timeline
FROM project_feedback_settings
WHERE project_id = $1;

-- name: CreateProjectFeedbackSettings :one
INSERT INTO project_feedback_settings (project_id, token, enabled, show_module_timeline)
VALUES ($1, $2, TRUE, FALSE)
RETURNING project_id, token, enabled, show_module_timeline;

-- name: SetProjectFeedbackSettingsEnabled :one
UPDATE project_feedback_settings
SET enabled = sqlc.arg(enabled)
WHERE project_id = sqlc.arg(project_id)
RETURNING project_id, token, enabled, show_module_timeline;

-- name: RegenerateProjectFeedbackToken :one
UPDATE project_feedback_settings
SET token = sqlc.arg(token)
WHERE project_id = sqlc.arg(project_id)
RETURNING project_id, token, enabled, show_module_timeline;

-- name: UpdateProjectFeedbackSettingsShowTimeline :one
UPDATE project_feedback_settings
SET show_module_timeline = sqlc.arg(show_module_timeline)
WHERE project_id = sqlc.arg(project_id)
RETURNING project_id, token, enabled, show_module_timeline;

-- name: GetProjectFeedbackSettingsByToken :one
SELECT s.project_id, s.enabled, s.show_module_timeline, p.name AS project_name, w.name AS workspace_name
FROM project_feedback_settings s
JOIN projects p ON p.id = s.project_id AND p.deleted_at IS NULL
JOIN workspaces w ON w.id = p.workspace_id AND w.deleted_at IS NULL
WHERE s.token = sqlc.arg(token);

-- name: IncrementProjectFeedbackSequence :one
UPDATE projects
SET feedback_sequence_counter = feedback_sequence_counter + 1
WHERE id = sqlc.arg(id)
RETURNING feedback_sequence_counter;

-- name: CreateProjectFeedback :one
INSERT INTO project_feedbacks (
    project_id, submitter_name, submitter_company, feature_name, content, priority, sequence_no, feedback_type
) VALUES (
    sqlc.arg(project_id), sqlc.arg(submitter_name), sqlc.arg(submitter_company),
    sqlc.arg(feature_name), sqlc.arg(content), sqlc.arg(priority), sqlc.arg(sequence_no), sqlc.arg(feedback_type)
)
RETURNING id, created_at, updated_at, project_id, submitter_name, submitter_company,
          feature_name, content, priority, status, response, responded_at, responded_by_id,
          expected_fix_date, sequence_no, feedback_type;

-- name: GetProjectFeedbackByID :one
SELECT id, created_at, updated_at, project_id, submitter_name, submitter_company,
       feature_name, content, priority, status, response, responded_at, responded_by_id,
       expected_fix_date, sequence_no, feedback_type
FROM project_feedbacks
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: UpdateProjectFeedbackResponse :one
UPDATE project_feedbacks
SET response = sqlc.narg(response),
    expected_fix_date = sqlc.narg(expected_fix_date),
    responded_at = NOW(),
    responded_by_id = sqlc.arg(responded_by_id)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id, created_at, updated_at, project_id, submitter_name, submitter_company,
          feature_name, content, priority, status, response, responded_at, responded_by_id,
          expected_fix_date, sequence_no, feedback_type;

-- name: UpdateProjectFeedback :one
UPDATE project_feedbacks
SET submitter_name = sqlc.arg(submitter_name),
    submitter_company = sqlc.arg(submitter_company),
    feature_name = sqlc.arg(feature_name),
    content = sqlc.arg(content),
    priority = sqlc.arg(priority),
    feedback_type = sqlc.arg(feedback_type)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id, created_at, updated_at, project_id, submitter_name, submitter_company,
          feature_name, content, priority, status, response, responded_at, responded_by_id,
          expected_fix_date, sequence_no, feedback_type;

-- name: UpdateProjectFeedbackStatus :one
UPDATE project_feedbacks
SET status = sqlc.arg(status)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id, created_at, updated_at, project_id, submitter_name, submitter_company,
          feature_name, content, priority, status, response, responded_at, responded_by_id,
          expected_fix_date, sequence_no, feedback_type;

-- name: CreateProjectFeedbackImage :one
INSERT INTO project_feedback_images (feedback_id, url)
VALUES ($1, $2)
RETURNING id, feedback_id, url, created_at;

-- name: ListProjectFeedbackImagesByFeedbackID :many
SELECT id, feedback_id, url, created_at
FROM project_feedback_images
WHERE feedback_id = $1
ORDER BY created_at;

-- name: DeleteProjectFeedbackImages :exec
DELETE FROM project_feedback_images
WHERE feedback_id = $1;

-- name: LinkFeedbackTask :exec
INSERT INTO project_feedback_tasks (feedback_id, task_id)
VALUES (sqlc.arg(feedback_id), sqlc.arg(task_id))
ON CONFLICT (feedback_id, task_id) DO NOTHING;

-- name: UnlinkFeedbackTask :exec
DELETE FROM project_feedback_tasks
WHERE feedback_id = sqlc.arg(feedback_id) AND task_id = sqlc.arg(task_id);

-- name: ListFeedbackTaskSummaries :many
SELECT t.id, t.sequence_no, t.status
FROM project_feedback_tasks pft
JOIN tasks t ON t.id = pft.task_id AND t.deleted_at IS NULL
WHERE pft.feedback_id = sqlc.arg(feedback_id)
ORDER BY pft.created_at;

-- name: ListFeedbacksByTaskID :many
SELECT f.id, f.created_at, f.updated_at, f.project_id, f.submitter_name, f.submitter_company,
       f.feature_name, f.content, f.priority, f.status, f.response, f.responded_at, f.responded_by_id,
       f.expected_fix_date, f.sequence_no, f.feedback_type
FROM project_feedbacks f
JOIN project_feedback_tasks pft ON pft.feedback_id = f.id
WHERE pft.task_id = sqlc.arg(task_id) AND f.deleted_at IS NULL
ORDER BY f.created_at DESC;

-- name: GetTaskProjectID :one
SELECT project_id
FROM tasks
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: CheckProjectMembership :one
SELECT EXISTS (
    SELECT 1
    FROM projects p
    LEFT JOIN project_users pu ON pu.project_id = p.id AND pu.user_id = sqlc.arg(user_id)
    LEFT JOIN workspace_users wu ON wu.workspace_id = p.workspace_id AND wu.user_id = sqlc.arg(user_id)
    WHERE p.id = sqlc.arg(project_id) AND (pu.user_id IS NOT NULL OR wu.user_id IS NOT NULL)
)::boolean;

-- name: CheckProjectManageAuth :one
SELECT EXISTS (
    SELECT 1
    FROM projects p
    LEFT JOIN project_users pu ON pu.project_id = p.id AND pu.user_id = sqlc.arg(user_id) AND pu.role IN ('admin', 'holder')
    LEFT JOIN workspace_users wu ON wu.workspace_id = p.workspace_id AND wu.user_id = sqlc.arg(user_id) AND wu.role IN ('admin', 'holder')
    WHERE p.id = sqlc.arg(project_id) AND (pu.user_id IS NOT NULL OR wu.user_id IS NOT NULL)
)::boolean;

-- name: CountPendingProjectFeedbacks :one
SELECT COUNT(*)::int
FROM project_feedbacks f
WHERE f.project_id = sqlc.arg(project_id) AND f.deleted_at IS NULL
  AND f.status = 'pending';
