package event

import (
	"context"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/workspace"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

type repository struct {
	queries *database.Queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{queries: database.New(pool)}
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

func (r *repository) ListEvents(ctx context.Context, workspaceAlias string, filter CalendarEventFilter) ([]CalendarDayResponse, error) {
	memberID, err := pgUUIDOrNull(filter.MemberID)
	if err != nil {
		return nil, err
	}
	fromTime, hasFromTime, err := parseOptionalTime(filter.From, false)
	if err != nil {
		return nil, err
	}
	toTime, hasToTime, err := parseOptionalTime(filter.To, true)
	if err != nil {
		return nil, err
	}
	fromDate, hasFromDate, err := parseOptionalDate(filter.From)
	if err != nil {
		return nil, err
	}
	toDate, hasToDate, err := parseOptionalDate(filter.To)
	if err != nil {
		return nil, err
	}

	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	scheduleRows, err := r.queries.ListWorkSchedulesByAlias(ctx, database.ListWorkSchedulesByAliasParams{
		WorkspaceID: wid,
		MemberID:    memberID,
		FromDate:    pgDateOrNull(fromDate, hasFromDate),
		ToDate:      pgDateOrNull(toDate, hasToDate),
		OffsetCount: 0,
		LimitCount:  10000,
	})
	if err != nil {
		return nil, err
	}
	schedules := make([]workScheduleDay, 0, len(scheduleRows))
	for _, row := range scheduleRows {
		sessions, err := r.queries.ListWorkScheduleSessionsByDayID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, workScheduleDayFromRow(row, sessions))
	}

	attendanceRows, err := r.queries.ListAttendanceRecordsByAlias(ctx, database.ListAttendanceRecordsByAliasParams{
		WorkspaceID: wid,
		MemberID:    memberID,
		FromTime:    pgTimeOrNull(fromTime, hasFromTime),
		ToTime:      pgTimeOrNull(toTime, hasToTime),
		OffsetCount: 0,
		LimitCount:  10000,
	})
	if err != nil {
		return nil, err
	}
	records := make([]attendanceRecord, 0, len(attendanceRows))
	for _, row := range attendanceRows {
		records = append(records, attendanceRecordFromRow(row))
	}

	birthdayRows, err := r.queries.ListWorkspaceMembersBirthdays(ctx, database.ListWorkspaceMembersBirthdaysParams{
		WorkspaceID: wid,
	})
	if err != nil {
		return nil, err
	}
	birthdays := make([]birthdayEventSource, 0, len(birthdayRows))
	for _, row := range birthdayRows {
		if row.Birthday.Valid {
			birthdays = append(birthdays, birthdayEventSource{
				WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
				MemberID:    pgutil.UUIDToString(row.MemberID),
				MemberAlias: row.MemberAlias,
				MemberName:  row.MemberName,
				Birthday:    row.Birthday.Time,
			})
		}
	}

	startYear := int32(fromDate.Year())
	endYear := int32(toDate.Year())
	if !hasFromDate {
		startYear = 1970
	}
	if !hasToDate {
		endYear = 2100
	}
	holidayRows, err := r.queries.ListHolidays(ctx, database.ListHolidaysParams{
		StartYear: startYear,
		EndYear:   endYear,
	})
	if err != nil {
		return nil, err
	}

	return eventsFromSources(schedules, records, birthdays, holidayRows, fromDate, toDate, hasFromDate, hasToDate), nil
}

func (r *repository) ListMembersEvents(ctx context.Context, workspaceAlias string, filter CalendarEventFilter) ([]MemberCalendarResponse, error) {
	fromTime, hasFromTime, err := parseOptionalTime(filter.From, false)
	if err != nil {
		return nil, err
	}
	toTime, hasToTime, err := parseOptionalTime(filter.To, true)
	if err != nil {
		return nil, err
	}
	fromDate, hasFromDate, err := parseOptionalDate(filter.From)
	if err != nil {
		return nil, err
	}
	toDate, hasToDate, err := parseOptionalDate(filter.To)
	if err != nil {
		return nil, err
	}

	wid2, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	members, err := r.queries.ListWorkspaceMembersForStatistics(ctx, database.ListWorkspaceMembersForStatisticsParams{
		WorkspaceID: wid2,
	})
	if err != nil {
		return nil, err
	}

	scheduleRows, err := r.queries.ListWorkSchedulesByAlias(ctx, database.ListWorkSchedulesByAliasParams{
		WorkspaceID: wid2,
		FromDate:    pgDateOrNull(fromDate, hasFromDate),
		ToDate:      pgDateOrNull(toDate, hasToDate),
		OffsetCount: 0,
		LimitCount:  100000,
	})
	if err != nil {
		return nil, err
	}
	schedules := make([]workScheduleDay, 0, len(scheduleRows))
	for _, row := range scheduleRows {
		sessions, err := r.queries.ListWorkScheduleSessionsByDayID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, workScheduleDayFromRow(row, sessions))
	}

	attendanceRows, err := r.queries.ListAttendanceRecordsByAlias(ctx, database.ListAttendanceRecordsByAliasParams{
		WorkspaceID: wid2,
		FromTime:    pgTimeOrNull(fromTime, hasFromTime),
		ToTime:      pgTimeOrNull(toTime, hasToTime),
		OffsetCount: 0,
		LimitCount:  100000,
	})
	if err != nil {
		return nil, err
	}
	records := make([]attendanceRecord, 0, len(attendanceRows))
	for _, row := range attendanceRows {
		records = append(records, attendanceRecordFromRow(row))
	}

	return memberEventsFromSources(members, schedules, records), nil
}
