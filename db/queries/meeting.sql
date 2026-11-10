-- name: CreateMeetingMinutes :one
WITH inserted AS (
    INSERT INTO meeting_minutes (
        workspace_id, project_id, title, meeting_date, location, attendees,
        recorders, summary, items, section_notes, created_by_id
    )
    VALUES (
        sqlc.arg(workspace_id), sqlc.narg(project_id), sqlc.arg(title), sqlc.arg(meeting_date),
        sqlc.arg(location), sqlc.arg(attendees), sqlc.arg(recorders), sqlc.arg(summary),
        sqlc.arg(items), sqlc.arg(section_notes), sqlc.arg(created_by_id)
    )
    RETURNING *
)
SELECT i.id, i.created_at, i.updated_at, i.workspace_id, i.project_id, i.title, i.meeting_date,
       i.location, i.attendees, i.recorders, i.summary, i.items, i.section_notes, i.created_by_id,
       COALESCE(u.name, '') AS creator_name, COALESCE(u.alias, '') AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM inserted i
LEFT JOIN users u ON u.id = i.created_by_id;

-- name: GetMeetingMinutes :one
SELECT m.id, m.created_at, m.updated_at, m.workspace_id, m.project_id, m.title, m.meeting_date,
       m.location, m.attendees, m.recorders, m.summary, m.items, m.section_notes, m.created_by_id,
       COALESCE(u.name, '') AS creator_name, COALESCE(u.alias, '') AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM meeting_minutes m
LEFT JOIN users u ON u.id = m.created_by_id
WHERE m.id = $1;

-- name: ListMeetingMinutes :many
SELECT m.id, m.created_at, m.updated_at, m.workspace_id, m.project_id, m.title, m.meeting_date,
       m.location, m.attendees, m.recorders, m.summary, m.items, m.section_notes, m.created_by_id,
       COALESCE(u.name, '') AS creator_name, COALESCE(u.alias, '') AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM meeting_minutes m
LEFT JOIN users u ON u.id = m.created_by_id
WHERE m.workspace_id = sqlc.arg(workspace_id)
  AND m.project_id IS NOT DISTINCT FROM sqlc.narg(project_id)::uuid
  AND (sqlc.narg(q)::text IS NULL OR m.title ILIKE '%' || sqlc.narg(q)::text || '%' OR m.items::text ILIKE '%' || sqlc.narg(q)::text || '%')
ORDER BY m.meeting_date DESC, m.created_at DESC
LIMIT sqlc.arg(row_limit) OFFSET sqlc.arg(row_offset);

-- name: UpdateMeetingMinutes :one
WITH updated AS (
    UPDATE meeting_minutes
    SET
        title = COALESCE(sqlc.narg(title)::text, meeting_minutes.title),
        meeting_date = COALESCE(sqlc.narg(meeting_date)::date, meeting_minutes.meeting_date),
        location = COALESCE(sqlc.narg(location)::text, meeting_minutes.location),
        attendees = COALESCE(sqlc.narg(attendees)::jsonb, meeting_minutes.attendees),
        recorders = COALESCE(sqlc.narg(recorders)::jsonb, meeting_minutes.recorders),
        summary = COALESCE(sqlc.narg(summary)::text, meeting_minutes.summary),
        items = COALESCE(sqlc.narg(items)::jsonb, meeting_minutes.items),
        section_notes = COALESCE(sqlc.narg(section_notes)::jsonb, meeting_minutes.section_notes)
    WHERE meeting_minutes.id = sqlc.arg(id)
    RETURNING *
)
SELECT m.id, m.created_at, m.updated_at, m.workspace_id, m.project_id, m.title, m.meeting_date,
       m.location, m.attendees, m.recorders, m.summary, m.items, m.section_notes, m.created_by_id,
       COALESCE(u.name, '') AS creator_name, COALESCE(u.alias, '') AS creator_alias, COALESCE(u.avatar, '') AS creator_avatar
FROM updated m
LEFT JOIN users u ON u.id = m.created_by_id;

-- name: DeleteMeetingMinutes :exec
DELETE FROM meeting_minutes WHERE id = $1;

-- name: ProjectBelongsToWorkspace :one
SELECT EXISTS (SELECT 1 FROM projects WHERE id = sqlc.arg(project_id) AND workspace_id = sqlc.arg(workspace_id));
