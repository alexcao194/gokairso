-- name: GetAttendanceSettingsByAlias :one
SELECT
    s.workspace_id,
    s.created_at,
    s.updated_at,
    s.company_wifi_bssid,
    s.company_wifi_ssid,
    s.company_latitude,
    s.company_longitude,
    s.allowed_radius_meters,
    s.validation_enabled
FROM workspace_attendance_settings s
WHERE s.workspace_id = $1
LIMIT 1;

-- name: UpsertAttendanceSettingsByAlias :one
INSERT INTO workspace_attendance_settings (
    workspace_id,
    company_wifi_bssid,
    company_wifi_ssid,
    company_latitude,
    company_longitude,
    allowed_radius_meters,
    validation_enabled
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(company_wifi_bssid),
    sqlc.arg(company_wifi_ssid),
    sqlc.arg(company_latitude),
    sqlc.arg(company_longitude),
    sqlc.arg(allowed_radius_meters),
    sqlc.arg(validation_enabled)
)
ON CONFLICT (workspace_id) DO UPDATE SET
    company_wifi_bssid = EXCLUDED.company_wifi_bssid,
    company_wifi_ssid = EXCLUDED.company_wifi_ssid,
    company_latitude = EXCLUDED.company_latitude,
    company_longitude = EXCLUDED.company_longitude,
    allowed_radius_meters = EXCLUDED.allowed_radius_meters,
    validation_enabled = EXCLUDED.validation_enabled
RETURNING
    workspace_id,
    created_at,
    updated_at,
    company_wifi_bssid,
    company_wifi_ssid,
    company_latitude,
    company_longitude,
    allowed_radius_meters,
    validation_enabled;

-- name: CreateAttendanceRecordByAlias :one
INSERT INTO attendance_records (
    workspace_id,
    member_id,
    type,
    recorded_at,
    latitude,
    longitude,
    wifi_bssid,
    wifi_ssid,
    distance_meters
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(member_id),
    sqlc.arg(type),
    sqlc.arg(recorded_at),
    sqlc.arg(latitude),
    sqlc.arg(longitude),
    sqlc.arg(wifi_bssid),
    sqlc.arg(wifi_ssid),
    sqlc.arg(distance_meters)
)
RETURNING
    id,
    created_at,
    updated_at,
    workspace_id,
    member_id,
    type,
    recorded_at,
    latitude,
    longitude,
    wifi_bssid,
    wifi_ssid,
    distance_meters;

-- name: GetAttendanceRecordByID :one
SELECT
    id,
    created_at,
    updated_at,
    workspace_id,
    member_id,
    type,
    recorded_at,
    latitude,
    longitude,
    wifi_bssid,
    wifi_ssid,
    distance_meters
FROM attendance_records
WHERE id = $1;

-- name: UpdateAttendanceRecordByID :one
UPDATE attendance_records
SET recorded_at = sqlc.arg(recorded_at)
WHERE id = sqlc.arg(id)
RETURNING
    id,
    created_at,
    updated_at,
    workspace_id,
    member_id,
    type,
    recorded_at,
    latitude,
    longitude,
    wifi_bssid,
    wifi_ssid,
    distance_meters;

-- name: DeleteAttendanceRecordByID :exec
DELETE FROM attendance_records
WHERE id = $1;

-- name: ListAttendanceRecordsByAlias :many
SELECT
    ar.id,
    ar.created_at,
    ar.updated_at,
    ar.workspace_id,
    ar.member_id,
    u.alias AS member_alias,
    u.name AS member_name,
    u.avatar AS member_avatar,
    ar.type,
    ar.recorded_at,
    ar.latitude,
    ar.longitude,
    ar.wifi_bssid,
    ar.wifi_ssid,
    ar.distance_meters
FROM attendance_records ar
INNER JOIN users u ON u.id = ar.member_id
WHERE ar.workspace_id = sqlc.arg(workspace_id)::uuid
  AND (sqlc.narg(member_id)::uuid IS NULL OR ar.member_id = sqlc.narg(member_id)::uuid)
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR ar.recorded_at >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR ar.recorded_at <= sqlc.narg(to_time)::timestamptz)
ORDER BY ar.recorded_at DESC, ar.id DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: UpsertWorkScheduleDayByAlias :one
INSERT INTO work_schedule_days (workspace_id, member_id, schedule_date, status, registered_by, note, week_note)
VALUES (sqlc.arg(workspace_id), sqlc.arg(member_id), sqlc.arg(schedule_date)::date, sqlc.arg(status)::text, sqlc.arg(registered_by), sqlc.arg(note), sqlc.arg(week_note))
ON CONFLICT (workspace_id, member_id, schedule_date) DO UPDATE SET status = EXCLUDED.status, registered_by = EXCLUDED.registered_by, note = EXCLUDED.note, week_note = EXCLUDED.week_note, updated_at = NOW()
RETURNING id, created_at, updated_at, workspace_id, member_id, schedule_date, status, registered_by, note, week_note;

-- name: UpdateWorkScheduleDayStatusByID :one
UPDATE work_schedule_days
SET status = sqlc.arg(status)::text, updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND member_id = sqlc.arg(member_id)
RETURNING id, created_at, updated_at, workspace_id, member_id, schedule_date, status, registered_by, note, week_note;

-- name: ApproveAllPendingWorkScheduleDaysByMember :many
UPDATE work_schedule_days
SET status = 'approved', updated_at = NOW()
WHERE workspace_id = sqlc.arg(workspace_id)
  AND member_id = sqlc.arg(member_id)
  AND status = 'pending'
RETURNING id, created_at, updated_at, workspace_id, member_id, schedule_date, status, registered_by, note, week_note;

-- name: DeleteWorkScheduleSessionsByDayID :exec
DELETE FROM work_schedule_sessions
WHERE day_id = $1;

-- name: DeleteWorkScheduleDayByID :exec
DELETE FROM work_schedule_days
WHERE id = $1;

-- name: CreateWorkScheduleSession :one
INSERT INTO work_schedule_sessions (day_id, start_at, end_at)
VALUES (sqlc.arg(day_id), sqlc.arg(start_at), sqlc.arg(end_at))
RETURNING id, created_at, updated_at, day_id, start_at, end_at;

-- name: ListWorkSchedulesByAlias :many
SELECT
    d.id,
    d.created_at,
    d.updated_at,
    d.workspace_id,
    d.member_id,
    u.alias AS member_alias,
    u.name AS member_name,
    u.avatar AS member_avatar,
    d.schedule_date,
    d.status,
    d.note,
    d.week_note,
    d.registered_by,
    r.alias AS registered_by_alias,
    r.name AS registered_by_name,
    r.avatar AS registered_by_avatar
FROM work_schedule_days d
INNER JOIN users u ON u.id = d.member_id
LEFT JOIN users r ON r.id = d.registered_by
WHERE d.workspace_id = sqlc.arg(workspace_id)::uuid
  AND (sqlc.narg(member_id)::uuid IS NULL OR d.member_id = sqlc.narg(member_id)::uuid)
  AND (sqlc.narg(from_date)::date IS NULL OR d.schedule_date >= sqlc.narg(from_date)::date)
  AND (sqlc.narg(to_date)::date IS NULL OR d.schedule_date <= sqlc.narg(to_date)::date)
  AND (sqlc.narg(status)::text IS NULL OR d.status = sqlc.narg(status)::text)
ORDER BY d.schedule_date DESC, u.alias
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: ListWorkScheduleSessionsByDayID :many
SELECT id, created_at, updated_at, day_id, start_at, end_at
FROM work_schedule_sessions
WHERE day_id = $1
ORDER BY start_at, id;

-- name: ListWorkspaceMembersForStatistics :many
SELECT wu.user_id AS member_id, u.alias AS member_alias, u.name AS member_name, u.avatar AS member_avatar
FROM workspace_users wu
INNER JOIN users u ON u.id = wu.user_id
WHERE wu.workspace_id = sqlc.arg(workspace_id)::uuid
  AND (sqlc.narg(member_id)::uuid IS NULL OR wu.user_id = sqlc.narg(member_id)::uuid)
ORDER BY u.alias;

-- name: SumAssignedTaskHoursByWorkspace :one
WITH task_workers_total AS (
    SELECT task_id, SUM(weight) AS total_weight
    FROM task_workers
    GROUP BY task_id
)
SELECT COALESCE(SUM(t.estimation * tw.weight / NULLIF(twt.total_weight, 0)), 0)::float8
FROM tasks t
INNER JOIN task_workers tw ON tw.task_id = t.id
INNER JOIN task_workers_total twt ON twt.task_id = t.id
INNER JOIN projects p ON p.id = t.project_id
WHERE p.workspace_id = sqlc.arg(workspace_id)::uuid
  AND t.deleted_at IS NULL
  AND tw.user_id = sqlc.arg(member_id)
  AND (t.start_date IS NULL OR t.start_date < sqlc.arg(month_end)::timestamptz)
  AND (t.end_date IS NULL OR t.end_date >= sqlc.arg(month_start)::timestamptz)
  AND (t.status IN ('review'::task_status, 'done'::task_status) OR (t.end_date IS NOT NULL AND t.end_date < NOW()));

-- name: ListScheduleNotificationRecipients :many
SELECT wu.user_id
FROM workspace_users wu
WHERE wu.workspace_id = sqlc.arg(workspace_id)::uuid
  AND wu.role IN ('holder', 'admin')
  AND wu.user_id <> sqlc.arg(member_id);

-- name: GetWorkWeekByAlias :many
SELECT d.id, d.created_at, d.updated_at, d.workspace_id, d.day_of_week
FROM workspace_work_days d
WHERE d.workspace_id = $1
ORDER BY d.day_of_week;

-- name: GetWorkWeekSessionsByDayID :many
SELECT id, created_at, updated_at, day_id, start_time::text AS start_time, end_time::text AS end_time
FROM workspace_work_sessions
WHERE day_id = $1
ORDER BY start_time, id;

-- name: UpsertWorkWeekDay :one
INSERT INTO workspace_work_days (workspace_id, day_of_week)
VALUES ($1, $2::smallint)
ON CONFLICT (workspace_id, day_of_week) DO UPDATE SET updated_at = NOW()
RETURNING id, created_at, updated_at, workspace_id, day_of_week;

-- name: DeleteWorkWeekDaysNotIn :exec
DELETE FROM workspace_work_days
WHERE workspace_id = $1
  AND NOT (day_of_week = ANY($2::smallint[]));

-- name: DeleteWorkWeekSessionsByDayID :exec
DELETE FROM workspace_work_sessions WHERE day_id = $1;

-- name: InsertWorkWeekSession :exec
INSERT INTO workspace_work_sessions (day_id, start_time, end_time)
VALUES ($1, $2, $3);

-- name: ClaimDueAttendanceReminders :many
WITH due AS (
    SELECT s.id AS session_id, d.workspace_id, d.member_id, d.schedule_date,
           s.start_at, s.end_at,
           CASE
               WHEN s.start_at BETWEEN sqlc.arg(window_start)::timestamptz AND sqlc.arg(window_end)::timestamptz
                   THEN 'check_in'
               ELSE 'check_out'
           END AS reminder_type
    FROM work_schedule_sessions s
    JOIN work_schedule_days d ON d.id = s.day_id
    WHERE d.status = 'approved'
      AND (s.start_at BETWEEN sqlc.arg(window_start)::timestamptz AND sqlc.arg(window_end)::timestamptz
       OR s.end_at BETWEEN sqlc.arg(window_start)::timestamptz - INTERVAL '10 minutes'
                       AND sqlc.arg(window_end)::timestamptz - INTERVAL '10 minutes')
),
claimed AS (
    INSERT INTO attendance_reminders (session_id, reminder_type)
    SELECT session_id, reminder_type FROM due
    ON CONFLICT (session_id, reminder_type) DO NOTHING
    RETURNING session_id, reminder_type
)
SELECT due.session_id, due.workspace_id, due.member_id, due.schedule_date,
       due.start_at, due.end_at, due.reminder_type
FROM due
JOIN claimed USING (session_id, reminder_type);

-- name: CreateAttendanceCorrectionRequestByAlias :one
INSERT INTO attendance_correction_requests (
    workspace_id,
    member_id,
    reason,
    check_in_at,
    check_out_at,
    target_check_in_record_id,
    target_check_out_record_id,
    original_check_in_at,
    original_check_out_at
)
VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(member_id),
    sqlc.arg(reason),
    sqlc.narg(check_in_at),
    sqlc.narg(check_out_at),
    sqlc.narg(target_check_in_record_id),
    sqlc.narg(target_check_out_record_id),
    sqlc.narg(original_check_in_at),
    sqlc.narg(original_check_out_at)
)
RETURNING id, created_at, updated_at, workspace_id, member_id, reason, status,
    check_in_at, check_out_at,
    target_check_in_record_id, target_check_out_record_id,
    original_check_in_at, original_check_out_at,
    result_check_in_record_id, result_check_out_record_id,
    decided_by, decided_at;

-- name: GetAttendanceCorrectionRequestByID :one
SELECT id, created_at, updated_at, workspace_id, member_id, reason, status,
    check_in_at, check_out_at,
    target_check_in_record_id, target_check_out_record_id,
    original_check_in_at, original_check_out_at,
    result_check_in_record_id, result_check_out_record_id,
    decided_by, decided_at
FROM attendance_correction_requests
WHERE id = $1;

-- name: ListAttendanceCorrectionRequestsByAlias :many
SELECT
    c.id,
    c.created_at,
    c.updated_at,
    c.workspace_id,
    c.member_id,
    u.alias AS member_alias,
    u.name AS member_name,
    u.avatar AS member_avatar,
    c.reason,
    c.status,
    c.check_in_at,
    c.check_out_at,
    c.target_check_in_record_id,
    c.target_check_out_record_id,
    c.original_check_in_at,
    c.original_check_out_at,
    c.result_check_in_record_id,
    c.result_check_out_record_id,
    c.decided_by,
    d.alias AS decided_by_alias,
    d.name AS decided_by_name,
    c.decided_at
FROM attendance_correction_requests c
INNER JOIN users u ON u.id = c.member_id
LEFT JOIN users d ON d.id = c.decided_by
WHERE c.workspace_id = sqlc.arg(workspace_id)::uuid
  AND (sqlc.narg(member_id)::uuid IS NULL OR c.member_id = sqlc.narg(member_id)::uuid)
  AND (sqlc.narg(status)::text IS NULL OR c.status = sqlc.narg(status)::text)
ORDER BY c.created_at DESC, c.id DESC
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: UpdateAttendanceCorrectionRequestDecisionByID :one
UPDATE attendance_correction_requests
SET status = sqlc.arg(status)::text,
    decided_by = sqlc.arg(decided_by),
    decided_at = NOW(),
    result_check_in_record_id = sqlc.narg(result_check_in_record_id)::uuid,
    result_check_out_record_id = sqlc.narg(result_check_out_record_id)::uuid,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING id, created_at, updated_at, workspace_id, member_id, reason, status,
    check_in_at, check_out_at,
    target_check_in_record_id, target_check_out_record_id,
    original_check_in_at, original_check_out_at,
    result_check_in_record_id, result_check_out_record_id,
    decided_by, decided_at;

-- name: ListCheckRequestNotificationRecipients :many
SELECT wu.user_id
FROM workspace_users wu
WHERE wu.workspace_id = sqlc.arg(workspace_id)::uuid
  AND wu.role = 'holder'
  AND wu.user_id <> sqlc.arg(member_id);

-- name: GetMemberWorkWeek :many
SELECT id, workspace_id, member_id, day_of_week
FROM member_work_days
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id)
ORDER BY day_of_week;

-- name: GetMemberWorkSessionsByDayID :many
SELECT start_time::text AS start_time, end_time::text AS end_time FROM member_work_sessions WHERE day_id = $1 ORDER BY start_time;

-- name: UpsertMemberWorkDay :one
INSERT INTO member_work_days (workspace_id, member_id, day_of_week)
VALUES (sqlc.arg(workspace_id), sqlc.arg(member_id), sqlc.arg(day_of_week)::smallint)
ON CONFLICT (workspace_id, member_id, day_of_week) DO UPDATE SET updated_at = NOW()
RETURNING id;

-- name: DeleteMemberWorkDaysNotIn :exec
DELETE FROM member_work_days
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id)
  AND day_of_week <> ALL(sqlc.arg(keep_days)::smallint[]);

-- name: DeleteMemberWorkSessionsByDayID :exec
DELETE FROM member_work_sessions WHERE day_id = $1;

-- name: InsertMemberWorkSession :exec
INSERT INTO member_work_sessions (day_id, start_time, end_time) VALUES ($1, $2, $3);

-- name: UpsertDailyReport :one
INSERT INTO daily_reports (workspace_id, member_id, report_date, done, in_progress, issues)
VALUES (sqlc.arg(workspace_id), sqlc.arg(member_id), sqlc.arg(report_date)::date, sqlc.arg(done), sqlc.arg(in_progress), sqlc.arg(issues))
ON CONFLICT (workspace_id, member_id, report_date) DO UPDATE
    SET done = EXCLUDED.done, in_progress = EXCLUDED.in_progress, issues = EXCLUDED.issues, updated_at = NOW()
RETURNING id, created_at, updated_at, workspace_id, member_id, report_date, done, in_progress, issues;

-- name: ListDailyReportsByAlias :many
SELECT
    dr.id,
    dr.created_at,
    dr.updated_at,
    dr.workspace_id,
    dr.member_id,
    u.alias AS member_alias,
    u.name AS member_name,
    u.avatar AS member_avatar,
    dr.report_date,
    dr.done,
    dr.in_progress,
    dr.issues
FROM daily_reports dr
INNER JOIN users u ON u.id = dr.member_id
WHERE dr.workspace_id = sqlc.arg(workspace_id)::uuid
  AND (sqlc.narg(member_id)::uuid IS NULL OR dr.member_id = sqlc.narg(member_id)::uuid)
  AND (sqlc.narg(from_date)::date IS NULL OR dr.report_date >= sqlc.narg(from_date)::date)
  AND (sqlc.narg(to_date)::date IS NULL OR dr.report_date <= sqlc.narg(to_date)::date)
ORDER BY dr.report_date DESC, u.alias
LIMIT sqlc.arg(limit_count) OFFSET sqlc.arg(offset_count);

-- name: ClaimDueDailyReportReminders :many
WITH due AS (
    SELECT d.workspace_id, d.member_id, d.schedule_date, MAX(s.end_at)::timestamptz AS last_end
    FROM work_schedule_days d
    JOIN work_schedule_sessions s ON s.day_id = d.id
    WHERE d.status = 'approved'
    GROUP BY d.workspace_id, d.member_id, d.schedule_date
    HAVING MAX(s.end_at) - INTERVAL '5 minutes'
               BETWEEN sqlc.arg(window_start)::timestamptz AND sqlc.arg(window_end)::timestamptz
       AND NOT EXISTS (
           SELECT 1 FROM daily_reports r
           WHERE r.workspace_id = d.workspace_id
             AND r.member_id = d.member_id
             AND r.report_date = d.schedule_date
       )
),
claimed AS (
    INSERT INTO daily_report_reminders (workspace_id, member_id, report_date)
    SELECT workspace_id, member_id, schedule_date FROM due
    ON CONFLICT (workspace_id, member_id, report_date) DO NOTHING
    RETURNING workspace_id, member_id, report_date AS schedule_date
)
SELECT due.workspace_id, due.member_id, due.schedule_date, due.last_end
FROM due
JOIN claimed USING (workspace_id, member_id, schedule_date);
