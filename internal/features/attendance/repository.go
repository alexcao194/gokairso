package attendance

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/workspace"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	queries *database.Queries
	db      *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{queries: database.New(pool), db: pool}
}

func (r *repository) ExecTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	txRepo := &repository{queries: r.queries.WithTx(tx), db: r.db}
	if err := fn(txRepo); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

func (r *repository) GetCallerRole(ctx context.Context, workspaceAlias, callerID string) (workspace.WorkspaceRole, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return "", err
	}
	uid, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return "", err
	}
	role, err := r.queries.GetCallerWorkspaceRole(ctx, database.GetCallerWorkspaceRoleParams{WorkspaceID: wid, UserID: uid})
	if err != nil {
		return "", err
	}
	return workspace.WorkspaceRole(string(role)), nil
}

func (r *repository) CheckProjectManageAuth(ctx context.Context, projectID, callerID string) (bool, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return false, err
	}
	uid, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return false, err
	}
	return r.queries.CheckProjectManageAuth(ctx, database.CheckProjectManageAuthParams{
		ProjectID: pid,
		UserID:    uid,
	})
}

func (r *repository) GetAttendanceSettings(ctx context.Context, workspaceAlias string) (*AttendanceSettingResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetAttendanceSettingsByAlias(ctx, wid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &AttendanceSettingResponse{}, nil
		}
		return nil, err
	}
	resp := attendanceSettingResponseFromDatabase(row)
	return &resp, nil
}

func (r *repository) UpsertAttendanceSettings(ctx context.Context, workspaceAlias string, req UpsertAttendanceSettingRequest) (*AttendanceSettingResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpsertAttendanceSettingsByAlias(ctx, database.UpsertAttendanceSettingsByAliasParams{
		WorkspaceID:         wid,
		CompanyWifiBssid:    req.CompanyWifiBSSID,
		CompanyWifiSsid:     req.CompanyWifiSSID,
		CompanyLatitude:     req.CompanyLatitude,
		CompanyLongitude:    req.CompanyLongitude,
		AllowedRadiusMeters: int32(req.AllowedRadiusMeters),
		ValidationEnabled:   req.ValidationEnabled,
	})
	if err != nil {
		return nil, err
	}
	resp := attendanceSettingResponseFromDatabase(row)
	return &resp, nil
}

func (r *repository) CreateAttendanceRecord(ctx context.Context, workspaceAlias, memberID string, req createAttendanceRecordParams) (*AttendanceRecordResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateAttendanceRecordByAlias(ctx, database.CreateAttendanceRecordByAliasParams{
		WorkspaceID:    wid,
		MemberID:       uid,
		Type:           string(req.Type),
		RecordedAt:     pgtype.Timestamptz{Time: req.RecordedAt, Valid: true},
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		WifiBssid:      req.WifiBSSID,
		WifiSsid:       req.WifiSSID,
		DistanceMeters: req.DistanceMeters,
	})
	if err != nil {
		return nil, err
	}
	resp := attendanceRecordResponseFromDatabase(row)
	return &resp, nil
}

func (r *repository) ListAttendanceRecords(ctx context.Context, workspaceAlias, callerID string, filter AttendanceRecordFilter, params pagination.Params) (pagination.Response[AttendanceRecordResponse], error) {
	var empty pagination.Response[AttendanceRecordResponse]
	memberID, err := pgUUIDOrNull(filter.MemberID)
	if err != nil {
		return empty, err
	}
	from, fromOK, err := parseOptionalTime(filter.From, false)
	if err != nil {
		return empty, err
	}
	to, toOK, err := parseOptionalTime(filter.To, true)
	if err != nil {
		return empty, err
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListAttendanceRecordsByAlias(ctx, database.ListAttendanceRecordsByAliasParams{
		WorkspaceID: wid,
		MemberID:    memberID,
		FromTime:    pgTimeOrNull(from, fromOK),
		ToTime:      pgTimeOrNull(to, toOK),
		OffsetCount: int32(params.Offset()),
		LimitCount:  int32(params.LimitWithMore()),
	})
	if err != nil {
		return empty, err
	}
	items := make([]AttendanceRecordResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, attendanceRecordResponseFromListRow(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) GetAttendanceRecord(ctx context.Context, recordID string) (*AttendanceRecordResponse, error) {
	rid, err := pgutil.ParseUUID(recordID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetAttendanceRecordByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	resp := attendanceRecordResponseFromDatabase(row)
	return &resp, nil
}

func (r *repository) UpdateAttendanceRecord(ctx context.Context, recordID string, recordedAt time.Time) (*AttendanceRecordResponse, error) {
	rid, err := pgutil.ParseUUID(recordID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateAttendanceRecordByID(ctx, database.UpdateAttendanceRecordByIDParams{
		ID:         rid,
		RecordedAt: pgtype.Timestamptz{Time: recordedAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	resp := attendanceRecordResponseFromDatabase(row)
	return &resp, nil
}

func (r *repository) DeleteAttendanceRecord(ctx context.Context, recordID string) error {
	rid, err := pgutil.ParseUUID(recordID)
	if err != nil {
		return err
	}
	return r.queries.DeleteAttendanceRecordByID(ctx, rid)
}

func (r *repository) CreateCheckRequest(ctx context.Context, workspaceAlias, memberID string, params createCheckRequestParams) (*AttendanceCorrectionRequestResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	targetCheckIn, err := pgUUIDOrNull(params.TargetCheckInRecordID)
	if err != nil {
		return nil, err
	}
	targetCheckOut, err := pgUUIDOrNull(params.TargetCheckOutRecordID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateAttendanceCorrectionRequestByAlias(ctx, database.CreateAttendanceCorrectionRequestByAliasParams{
		WorkspaceID:            wid,
		MemberID:               uid,
		Reason:                 params.Reason,
		CheckInAt:              pgTimePtr(params.CheckInAt),
		CheckOutAt:             pgTimePtr(params.CheckOutAt),
		TargetCheckInRecordID:  targetCheckIn,
		TargetCheckOutRecordID: targetCheckOut,
		OriginalCheckInAt:      pgTimePtr(params.OriginalCheckInAt),
		OriginalCheckOutAt:     pgTimePtr(params.OriginalCheckOutAt),
	})
	if err != nil {
		return nil, err
	}
	resp := attendanceCorrectionRequestResponseFromDatabase(database.AttendanceCorrectionRequest{
		ID:                     row.ID,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
		WorkspaceID:            row.WorkspaceID,
		MemberID:               row.MemberID,
		Reason:                 row.Reason,
		Status:                 row.Status,
		CheckInAt:              row.CheckInAt,
		CheckOutAt:             row.CheckOutAt,
		TargetCheckInRecordID:  row.TargetCheckInRecordID,
		TargetCheckOutRecordID: row.TargetCheckOutRecordID,
		OriginalCheckInAt:      row.OriginalCheckInAt,
		OriginalCheckOutAt:     row.OriginalCheckOutAt,
		ResultCheckInRecordID:  row.ResultCheckInRecordID,
		ResultCheckOutRecordID: row.ResultCheckOutRecordID,
		DecidedBy:              row.DecidedBy,
		DecidedAt:              row.DecidedAt,
	})
	return &resp, nil
}

func (r *repository) GetCheckRequest(ctx context.Context, requestID string) (*AttendanceCorrectionRequestResponse, error) {
	rid, err := pgutil.ParseUUID(requestID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetAttendanceCorrectionRequestByID(ctx, rid)
	if err != nil {
		return nil, err
	}
	resp := attendanceCorrectionRequestResponseFromDatabase(database.AttendanceCorrectionRequest{
		ID:                     row.ID,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
		WorkspaceID:            row.WorkspaceID,
		MemberID:               row.MemberID,
		Reason:                 row.Reason,
		Status:                 row.Status,
		CheckInAt:              row.CheckInAt,
		CheckOutAt:             row.CheckOutAt,
		TargetCheckInRecordID:  row.TargetCheckInRecordID,
		TargetCheckOutRecordID: row.TargetCheckOutRecordID,
		OriginalCheckInAt:      row.OriginalCheckInAt,
		OriginalCheckOutAt:     row.OriginalCheckOutAt,
		ResultCheckInRecordID:  row.ResultCheckInRecordID,
		ResultCheckOutRecordID: row.ResultCheckOutRecordID,
		DecidedBy:              row.DecidedBy,
		DecidedAt:              row.DecidedAt,
	})
	return &resp, nil
}

func (r *repository) ListCheckRequests(ctx context.Context, workspaceAlias string, filter AttendanceCorrectionRequestFilter, params pagination.Params) (pagination.Response[AttendanceCorrectionRequestResponse], error) {
	var empty pagination.Response[AttendanceCorrectionRequestResponse]
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return empty, err
	}
	memberID, err := pgUUIDOrNull(filter.MemberID)
	if err != nil {
		return empty, err
	}
	var status pgtype.Text
	if filter.Status != "" {
		status = pgtype.Text{String: filter.Status, Valid: true}
	}
	rows, err := r.queries.ListAttendanceCorrectionRequestsByAlias(ctx, database.ListAttendanceCorrectionRequestsByAliasParams{
		WorkspaceID: wid,
		MemberID:    memberID,
		Status:      status,
		OffsetCount: int32(params.Offset()),
		LimitCount:  int32(params.LimitWithMore()),
	})
	if err != nil {
		return empty, err
	}
	items := make([]AttendanceCorrectionRequestResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, attendanceCorrectionRequestResponseFromListRow(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) DecideCheckRequest(ctx context.Context, requestID string, status AttendanceCorrectionStatus, decidedBy string, resultCheckInRecordID, resultCheckOutRecordID *string) (*AttendanceCorrectionRequestResponse, error) {
	rid, err := pgutil.ParseUUID(requestID)
	if err != nil {
		return nil, err
	}
	decidedByID, err := pgutil.ParseUUID(decidedBy)
	if err != nil {
		return nil, err
	}
	resultCheckInArg, err := pgUUIDPtrOrNull(resultCheckInRecordID)
	if err != nil {
		return nil, err
	}
	resultCheckOutArg, err := pgUUIDPtrOrNull(resultCheckOutRecordID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateAttendanceCorrectionRequestDecisionByID(ctx, database.UpdateAttendanceCorrectionRequestDecisionByIDParams{
		Status:                 string(status),
		DecidedBy:              decidedByID,
		ResultCheckInRecordID:  resultCheckInArg,
		ResultCheckOutRecordID: resultCheckOutArg,
		ID:                     rid,
	})
	if err != nil {
		return nil, err
	}
	resp := attendanceCorrectionRequestResponseFromDatabase(database.AttendanceCorrectionRequest{
		ID:                     row.ID,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
		WorkspaceID:            row.WorkspaceID,
		MemberID:               row.MemberID,
		Reason:                 row.Reason,
		Status:                 row.Status,
		CheckInAt:              row.CheckInAt,
		CheckOutAt:             row.CheckOutAt,
		TargetCheckInRecordID:  row.TargetCheckInRecordID,
		TargetCheckOutRecordID: row.TargetCheckOutRecordID,
		OriginalCheckInAt:      row.OriginalCheckInAt,
		OriginalCheckOutAt:     row.OriginalCheckOutAt,
		ResultCheckInRecordID:  row.ResultCheckInRecordID,
		ResultCheckOutRecordID: row.ResultCheckOutRecordID,
		DecidedBy:              row.DecidedBy,
		DecidedAt:              row.DecidedAt,
	})
	return &resp, nil
}

func (r *repository) ListCheckRequestNotificationRecipients(ctx context.Context, workspaceAlias, memberID string) ([]string, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListCheckRequestNotificationRecipients(ctx, database.ListCheckRequestNotificationRecipientsParams{WorkspaceID: wid, MemberID: uid})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, pgutil.UUIDToString(row))
	}
	return ids, nil
}

func (r *repository) ReplaceWorkSchedule(ctx context.Context, workspaceAlias, memberID, registeredBy string, req UpsertWorkScheduleRequest, status WorkScheduleStatus, weekNote string) (*WorkScheduleDayResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	registeredByID, err := pgUUIDOrNull(registeredBy)
	if err != nil {
		return nil, err
	}
	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	day, err := r.queries.UpsertWorkScheduleDayByAlias(ctx, database.UpsertWorkScheduleDayByAliasParams{
		WorkspaceID:  wid,
		MemberID:     uid,
		ScheduleDate: pgDate(date),
		Status:       string(status),
		RegisteredBy: registeredByID,
		Note:         req.Note,
		WeekNote:     weekNote,
	})
	if err != nil {
		return nil, err
	}
	if err := r.queries.DeleteWorkScheduleSessionsByDayID(ctx, day.ID); err != nil {
		return nil, err
	}
	if len(req.Sessions) == 0 {
		if err := r.queries.DeleteWorkScheduleDayByID(ctx, day.ID); err != nil {
			return nil, err
		}
		resp := workScheduleDayResponseFromDatabase(day)
		resp.ID = ""
		return &resp, nil
	}
	resp := workScheduleDayResponseFromDatabase(day)
	for _, session := range req.Sessions {
		row, err := r.queries.CreateWorkScheduleSession(ctx, database.CreateWorkScheduleSessionParams{
			DayID:   day.ID,
			StartAt: pgtype.Timestamptz{Time: session.StartAt, Valid: true},
			EndAt:   pgtype.Timestamptz{Time: session.EndAt, Valid: true},
		})
		if err != nil {
			return nil, err
		}
		resp.Sessions = append(resp.Sessions, workScheduleSessionResponseFromDatabase(row))
	}
	return &resp, nil
}

func (r *repository) ClaimDueAttendanceReminders(ctx context.Context, windowStart, windowEnd time.Time) ([]AttendanceReminder, error) {
	rows, err := r.queries.ClaimDueAttendanceReminders(ctx, database.ClaimDueAttendanceRemindersParams{
		WindowStart: pgtype.Timestamptz{Time: windowStart, Valid: true},
		WindowEnd:   pgtype.Timestamptz{Time: windowEnd, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	items := make([]AttendanceReminder, 0, len(rows))
	for _, row := range rows {
		items = append(items, attendanceReminderFromClaimRow(row))
	}
	return items, nil
}

func (r *repository) ListWorkSchedules(ctx context.Context, workspaceAlias, callerID string, filter WorkScheduleFilter, params pagination.Params) (pagination.Response[WorkScheduleDayResponse], error) {
	var empty pagination.Response[WorkScheduleDayResponse]
	dbFilter, err := workScheduleDBFilter(filter)
	if err != nil {
		return empty, err
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListWorkSchedulesByAlias(ctx, database.ListWorkSchedulesByAliasParams{
		WorkspaceID: wid,
		MemberID:    dbFilter.memberID,
		FromDate:    dbFilter.fromDate,
		ToDate:      dbFilter.toDate,
		Status:      dbFilter.status,
		OffsetCount: int32(params.Offset()),
		LimitCount:  int32(params.LimitWithMore()),
	})
	if err != nil {
		return empty, err
	}
	items, err := r.hydrateScheduleRows(ctx, rows)
	if err != nil {
		return empty, err
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) UpdateWorkScheduleDayStatus(ctx context.Context, workspaceAlias, memberID string, dayIDs []string, status WorkScheduleStatus) ([]WorkScheduleDayResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	responses := make([]WorkScheduleDayResponse, 0, len(dayIDs))
	for _, dayID := range dayIDs {
		did, err := pgutil.ParseUUID(dayID)
		if err != nil {
			return nil, err
		}
		day, err := r.queries.UpdateWorkScheduleDayStatusByID(ctx, database.UpdateWorkScheduleDayStatusByIDParams{
			Status:      string(status),
			ID:          did,
			WorkspaceID: wid,
			MemberID:    uid,
		})
		if err != nil {
			return nil, err
		}
		resp := workScheduleDayResponseFromDatabase(day)
		sessions, err := r.queries.ListWorkScheduleSessionsByDayID(ctx, day.ID)
		if err != nil {
			return nil, err
		}
		for _, session := range sessions {
			resp.Sessions = append(resp.Sessions, workScheduleSessionResponseFromDatabase(session))
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

func (r *repository) ApproveAllPendingSchedules(ctx context.Context, workspaceAlias, memberID string) ([]WorkScheduleDayResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	days, err := r.queries.ApproveAllPendingWorkScheduleDaysByMember(ctx, database.ApproveAllPendingWorkScheduleDaysByMemberParams{
		WorkspaceID: wid,
		MemberID:    uid,
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].ScheduleDate.Time.Before(days[j].ScheduleDate.Time)
	})
	responses := make([]WorkScheduleDayResponse, 0, len(days))
	for _, day := range days {
		resp := workScheduleDayResponseFromDatabase(day)
		sessions, err := r.queries.ListWorkScheduleSessionsByDayID(ctx, day.ID)
		if err != nil {
			return nil, err
		}
		for _, session := range sessions {
			resp.Sessions = append(resp.Sessions, workScheduleSessionResponseFromDatabase(session))
		}
		responses = append(responses, resp)
	}
	return responses, nil
}

func (r *repository) ListWorkspaceMembersForStatistics(ctx context.Context, workspaceAlias string, memberID string) ([]WorkspaceMemberBriefResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgUUIDOrNull(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListWorkspaceMembersForStatistics(ctx, database.ListWorkspaceMembersForStatisticsParams{WorkspaceID: wid, MemberID: uid})
	if err != nil {
		return nil, err
	}
	items := make([]WorkspaceMemberBriefResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, workspaceMemberBriefFromRow(row))
	}
	return items, nil
}

func (r *repository) ListWorkspaceScheduleStats(ctx context.Context, workspaceAlias string, from, to time.Time, memberID string) ([]WorkScheduleDayResponse, error) {
	resp, err := r.ListWorkSchedules(ctx, workspaceAlias, "", WorkScheduleFilter{
		From:     from.Format(time.DateOnly),
		To:       to.Format(time.DateOnly),
		MemberID: memberID,
	}, pagination.Params{Page: 1, Limit: 10000})
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (r *repository) ListWorkspaceAttendanceStats(ctx context.Context, workspaceAlias string, from, to time.Time, memberID string) ([]AttendanceRecordResponse, error) {
	resp, err := r.ListAttendanceRecords(ctx, workspaceAlias, "", AttendanceRecordFilter{
		From:     from.Format(time.RFC3339),
		To:       to.Format(time.RFC3339),
		MemberID: memberID,
	}, pagination.Params{Page: 1, Limit: 10000})
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (r *repository) SumAssignedTaskHours(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (float64, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return 0, err
	}
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return 0, err
	}
	return r.queries.SumAssignedTaskHoursByWorkspace(ctx, database.SumAssignedTaskHoursByWorkspaceParams{
		WorkspaceID: wid,
		MemberID:    uid,
		MonthStart:  pgtype.Timestamptz{Time: from, Valid: true},
		MonthEnd:    pgtype.Timestamptz{Time: to, Valid: true},
	})
}

func (r *repository) GetMemberName(ctx context.Context, memberID string) (string, error) {
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return "", err
	}
	user, err := r.queries.GetUserByID(ctx, uid)
	if err != nil {
		return "", err
	}
	return user.Name, nil
}

func (r *repository) ListScheduleNotificationRecipients(ctx context.Context, workspaceAlias, memberID string) ([]string, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListScheduleNotificationRecipients(ctx, database.ListScheduleNotificationRecipientsParams{WorkspaceID: wid, MemberID: uid})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, pgutil.UUIDToString(row))
	}
	return ids, nil
}

type scheduleDBFilter struct {
	memberID pgtype.UUID
	fromDate pgtype.Date
	toDate   pgtype.Date
	status   pgtype.Text
}

func workScheduleDBFilter(filter WorkScheduleFilter) (scheduleDBFilter, error) {
	memberID, err := pgUUIDOrNull(filter.MemberID)
	if err != nil {
		return scheduleDBFilter{}, err
	}
	var fromDate, toDate pgtype.Date
	if filter.Month != "" {
		month, err := parseMonth(filter.Month)
		if err != nil {
			return scheduleDBFilter{}, err
		}
		fromDate = pgDate(month)
		toDate = pgDate(month.AddDate(0, 1, -1))
	}
	if filter.From != "" {
		from, err := parseDate(filter.From)
		if err != nil {
			return scheduleDBFilter{}, err
		}
		fromDate = pgDate(from)
	}
	if filter.To != "" {
		to, err := parseDate(filter.To)
		if err != nil {
			return scheduleDBFilter{}, err
		}
		toDate = pgDate(to)
	}
	var status pgtype.Text
	if filter.Status != "" {
		status = pgtype.Text{String: filter.Status, Valid: true}
	}
	return scheduleDBFilter{memberID: memberID, fromDate: fromDate, toDate: toDate, status: status}, nil
}

func (r *repository) GetTaskStatsByWorkspace(ctx context.Context, workspaceAlias string, filter TaskStatisticsFilter, from, to time.Time) ([]TaskStatsMemberResponse, error) {
	memberID, err := pgUUIDOrNull(filter.MemberID)
	if err != nil {
		return nil, err
	}
	moduleID, err := pgUUIDOrNull(filter.ModuleID)
	if err != nil {
		return nil, err
	}
	projectID, err := pgUUIDOrNull(filter.ProjectID)
	if err != nil {
		return nil, err
	}
	var fromTime, toTime pgtype.Timestamptz
	if !from.IsZero() {
		fromTime = pgtype.Timestamptz{Time: from, Valid: true}
	}
	if !to.IsZero() {
		toTime = pgtype.Timestamptz{Time: to, Valid: true}
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.GetTaskStatsByWorkspaceAlias(ctx, database.GetTaskStatsByWorkspaceAliasParams{
		WorkspaceID: wid,
		MemberID:    memberID,
		ProjectID:   projectID,
		ModuleID:    moduleID,
		FromTime:    fromTime,
		ToTime:      toTime,
	})
	if err != nil {
		return nil, err
	}
	items := make([]TaskStatsMemberResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, TaskStatsMemberResponse{
			MemberID:                pgutil.UUIDToString(row.MemberID),
			MemberAlias:             row.MemberAlias,
			MemberName:              row.MemberName,
			MemberAvatar:            textString(row.MemberAvatar),
			TotalTasks:              row.TotalTasks,
			Backlog:                 row.BacklogCount,
			Todo:                    row.TodoCount,
			InProgress:              row.InProgressCount,
			Review:                  row.ReviewCount,
			Done:                    row.DoneCount,
			Canceled:                row.CanceledCount,
			Priority1:               row.Priority1Count,
			Priority2:               row.Priority2Count,
			Priority3:               row.Priority3Count,
			Priority4:               row.Priority4Count,
			TotalEstimation:         row.TotalEstimation,
			TotalOverdue:            row.TotalOverdue,
			TotalEstimatedTaskHours: row.TotalEstimation,
			TotalLate:               row.TotalLate,
			TotalBug:                row.TotalBug,
		})
	}
	return items, nil
}

func (r *repository) GetTaskCycleTimeByWorkspace(ctx context.Context, workspaceAlias string, filter TaskStatisticsFilter, from, to time.Time) (float64, int64, error) {
	memberID, err := pgUUIDOrNull(filter.MemberID)
	if err != nil {
		return 0, 0, err
	}
	moduleID, err := pgUUIDOrNull(filter.ModuleID)
	if err != nil {
		return 0, 0, err
	}
	projectID, err := pgUUIDOrNull(filter.ProjectID)
	if err != nil {
		return 0, 0, err
	}
	var fromTime, toTime pgtype.Timestamptz
	if !from.IsZero() {
		fromTime = pgtype.Timestamptz{Time: from, Valid: true}
	}
	if !to.IsZero() {
		toTime = pgtype.Timestamptz{Time: to, Valid: true}
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return 0, 0, err
	}
	row, err := r.queries.GetTaskCycleTimeByWorkspaceAlias(ctx, database.GetTaskCycleTimeByWorkspaceAliasParams{
		WorkspaceID: wid,
		MemberID:    memberID,
		ProjectID:   projectID,
		ModuleID:    moduleID,
		FromTime:    fromTime,
		ToTime:      toTime,
	})
	if err != nil {
		return 0, 0, err
	}
	return row.AvgCycleDays, row.SampleSize, nil
}

func (r *repository) GetWorkWeek(ctx context.Context, workspaceAlias string) (*WorkWeekResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.GetWorkWeekByAlias(ctx, wid)
	if err != nil {
		return nil, err
	}
	resp := &WorkWeekResponse{
		Days: make([]WorkWeekDayResponse, 0, len(rows)),
	}
	if len(rows) > 0 {
		resp.WorkspaceID = pgutil.UUIDToString(rows[0].WorkspaceID)
	}
	for _, row := range rows {
		day := WorkWeekDayResponse{
			DayOfWeek: int(row.DayOfWeek),
			Sessions:  []WorkWeekSessionResponse{},
		}
		sessions, err := r.queries.GetWorkWeekSessionsByDayID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		for _, s := range sessions {
			day.Sessions = append(day.Sessions, WorkWeekSessionResponse{
				StartTime: normalizeSQLTime(s.StartTime),
				EndTime:   normalizeSQLTime(s.EndTime),
			})
		}
		resp.Days = append(resp.Days, day)
	}
	return resp, nil
}

func (r *repository) ReplaceWorkWeek(ctx context.Context, workspaceAlias string, req UpsertWorkWeekRequest) (*WorkWeekResponse, error) {
	keepDays := make([]int16, 0, len(req.Days))
	for _, d := range req.Days {
		keepDays = append(keepDays, int16(d.DayOfWeek))
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	if err := r.queries.DeleteWorkWeekDaysNotIn(ctx, database.DeleteWorkWeekDaysNotInParams{
		WorkspaceID: wid,
		Column2:     keepDays,
	}); err != nil {
		return nil, err
	}
	for _, d := range req.Days {
		row, err := r.queries.UpsertWorkWeekDay(ctx, database.UpsertWorkWeekDayParams{
			WorkspaceID: wid,
			Column2:     int16(d.DayOfWeek),
		})
		if err != nil {
			return nil, err
		}
		if err := r.queries.DeleteWorkWeekSessionsByDayID(ctx, row.ID); err != nil {
			return nil, err
		}
		for _, s := range d.Sessions {
			startClock, err := pgClock(s.StartTime)
			if err != nil {
				return nil, err
			}
			endClock, err := pgClock(s.EndTime)
			if err != nil {
				return nil, err
			}
			if err := r.queries.InsertWorkWeekSession(ctx, database.InsertWorkWeekSessionParams{
				DayID:     row.ID,
				StartTime: startClock,
				EndTime:   endClock,
			}); err != nil {
				return nil, err
			}
		}
	}
	return r.GetWorkWeek(ctx, workspaceAlias)
}

func (r *repository) GetMemberWorkWeek(ctx context.Context, workspaceAlias, memberID string) (*WorkWeekResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	mid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.GetMemberWorkWeek(ctx, database.GetMemberWorkWeekParams{WorkspaceID: wid, MemberID: mid})
	if err != nil {
		return nil, err
	}
	resp := &WorkWeekResponse{WorkspaceID: workspaceAlias, Days: make([]WorkWeekDayResponse, 0, len(rows))}
	for _, row := range rows {
		day := WorkWeekDayResponse{DayOfWeek: int(row.DayOfWeek), Sessions: []WorkWeekSessionResponse{}}
		sessions, err := r.queries.GetMemberWorkSessionsByDayID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		for _, s := range sessions {
			day.Sessions = append(day.Sessions, WorkWeekSessionResponse{StartTime: normalizeSQLTime(s.StartTime), EndTime: normalizeSQLTime(s.EndTime)})
		}
		resp.Days = append(resp.Days, day)
	}
	return resp, nil
}

func (r *repository) ReplaceMemberWorkWeek(ctx context.Context, workspaceAlias, memberID string, req UpsertWorkWeekRequest) (*WorkWeekResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	mid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	keepDays := make([]int16, 0, len(req.Days))
	for _, d := range req.Days {
		keepDays = append(keepDays, int16(d.DayOfWeek))
	}
	if err := r.queries.DeleteMemberWorkDaysNotIn(ctx, database.DeleteMemberWorkDaysNotInParams{WorkspaceID: wid, MemberID: mid, KeepDays: keepDays}); err != nil {
		return nil, err
	}
	for _, d := range req.Days {
		dayID, err := r.queries.UpsertMemberWorkDay(ctx, database.UpsertMemberWorkDayParams{WorkspaceID: wid, MemberID: mid, DayOfWeek: int16(d.DayOfWeek)})
		if err != nil {
			return nil, err
		}
		if err := r.queries.DeleteMemberWorkSessionsByDayID(ctx, dayID); err != nil {
			return nil, err
		}
		for _, s := range d.Sessions {
			startClock, err := pgClock(s.StartTime)
			if err != nil {
				return nil, err
			}
			endClock, err := pgClock(s.EndTime)
			if err != nil {
				return nil, err
			}
			if err := r.queries.InsertMemberWorkSession(ctx, database.InsertMemberWorkSessionParams{DayID: dayID, StartTime: startClock, EndTime: endClock}); err != nil {
				return nil, err
			}
		}
	}
	return r.GetMemberWorkWeek(ctx, workspaceAlias, memberID)
}

func (r *repository) UpsertDailyReport(ctx context.Context, workspaceAlias, memberID string, req UpsertDailyReportRequest) (*DailyReportResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpsertDailyReport(ctx, database.UpsertDailyReportParams{
		WorkspaceID: wid,
		MemberID:    uid,
		ReportDate:  pgDate(date),
		Done:        req.Done,
		InProgress:  req.InProgress,
		Issues:      req.Issues,
	})
	if err != nil {
		return nil, err
	}
	resp := dailyReportResponseFromDatabase(row)
	return &resp, nil
}

func (r *repository) ListDailyReports(ctx context.Context, workspaceAlias string, filter DailyReportFilter, params pagination.Params) (pagination.Response[DailyReportResponse], error) {
	var empty pagination.Response[DailyReportResponse]
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return empty, err
	}
	memberID, err := pgUUIDOrNull(filter.MemberID)
	if err != nil {
		return empty, err
	}
	var fromDate, toDate pgtype.Date
	if filter.From != "" {
		from, err := parseDate(filter.From)
		if err != nil {
			return empty, err
		}
		fromDate = pgDate(from)
	}
	if filter.To != "" {
		to, err := parseDate(filter.To)
		if err != nil {
			return empty, err
		}
		toDate = pgDate(to)
	}
	rows, err := r.queries.ListDailyReportsByAlias(ctx, database.ListDailyReportsByAliasParams{
		WorkspaceID: wid,
		MemberID:    memberID,
		FromDate:    fromDate,
		ToDate:      toDate,
		OffsetCount: int32(params.Offset()),
		LimitCount:  int32(params.LimitWithMore()),
	})
	if err != nil {
		return empty, err
	}
	items := make([]DailyReportResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, dailyReportResponseFromListRow(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) ClaimDueDailyReportReminders(ctx context.Context, windowStart, windowEnd time.Time) ([]DailyReportReminder, error) {
	rows, err := r.queries.ClaimDueDailyReportReminders(ctx, database.ClaimDueDailyReportRemindersParams{
		WindowStart: pgtype.Timestamptz{Time: windowStart, Valid: true},
		WindowEnd:   pgtype.Timestamptz{Time: windowEnd, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	items := make([]DailyReportReminder, 0, len(rows))
	for _, row := range rows {
		items = append(items, dailyReportReminderFromClaimRow(row))
	}
	return items, nil
}

func (r *repository) SumInitiativeEstimates(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]float64, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	memberUID, err := pgUUIDOrNull(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.SumInitiativeEstimateByRefMember(ctx, database.SumInitiativeEstimateByRefMemberParams{
		WorkspaceID: wid,
		From:        pgtype.Timestamptz{Time: from, Valid: true},
		To:          pgtype.Timestamptz{Time: to, Valid: true},
		MemberID:    memberUID,
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string]float64, len(rows))
	for _, row := range rows {
		result[pgutil.UUIDToString(row.UserID)] = row.Total
	}
	return result, nil
}

func (r *repository) SumLateLogEstimates(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]float64, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	memberUID, err := pgUUIDOrNull(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.SumLateLogEstimateByUser(ctx, database.SumLateLogEstimateByUserParams{
		WorkspaceID: wid,
		From:        pgtype.Timestamptz{Time: from, Valid: true},
		To:          pgtype.Timestamptz{Time: to, Valid: true},
		MemberID:    memberUID,
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string]float64, len(rows))
	for _, row := range rows {
		result[pgutil.UUIDToString(row.UserID)] = row.Total
	}
	return result, nil
}

func (r *repository) SumFeedbackEstimates(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]float64, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	memberUID, err := pgUUIDOrNull(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.SumFeedbackEstimateByBeneficiary(ctx, database.SumFeedbackEstimateByBeneficiaryParams{
		WorkspaceID: wid,
		From:        pgtype.Timestamptz{Time: from, Valid: true},
		To:          pgtype.Timestamptz{Time: to, Valid: true},
		MemberID:    memberUID,
	})
	if err != nil {
		return nil, err
	}
	result := make(map[string]float64, len(rows))
	for _, row := range rows {
		result[pgutil.UUIDToString(row.UserID)] = row.Total
	}
	return result, nil
}

func (r *repository) hydrateScheduleRows(ctx context.Context, rows []database.ListWorkSchedulesByAliasRow) ([]WorkScheduleDayResponse, error) {
	items := make([]WorkScheduleDayResponse, 0, len(rows))
	for _, row := range rows {
		item := workScheduleDayResponseFromListRow(row)
		sessions, err := r.queries.ListWorkScheduleSessionsByDayID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		for _, session := range sessions {
			item.Sessions = append(item.Sessions, workScheduleSessionResponseFromDatabase(session))
		}
		items = append(items, item)
	}
	return items, nil
}
