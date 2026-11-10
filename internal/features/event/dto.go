package event

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	vietnamTimezone = "Asia/Ho_Chi_Minh"

	attendanceTypeCheckIn  = "check_in"
	attendanceTypeCheckOut = "check_out"
	EventTypeBirthday      = "birthday"
)

func textString(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

type birthdayEventSource struct {
	WorkspaceID string
	MemberID    string
	MemberAlias string
	MemberName  string
	Birthday    time.Time
}

type CalendarEventFilter struct {
	From     string `schema:"from"`
	To       string `schema:"to"`
	MemberID string `schema:"-"`
}

type CalendarEventCheckInfo struct {
	ID             string    `json:"id"`
	RecordedAt     time.Time `json:"recordedAt" swaggertype:"string" format:"date-time"`
	DistanceMeters float64   `json:"distanceMeters"`
}

type CalendarScheduleSession struct {
	ID      string    `json:"id"`
	StartAt time.Time `json:"startAt" swaggertype:"string" format:"date-time"`
	EndAt   time.Time `json:"endAt" swaggertype:"string" format:"date-time"`
}

type CalendarScheduleInfo struct {
	ID                 string                    `json:"id"`
	Status             string                    `json:"status,omitempty"`
	Sessions           []CalendarScheduleSession `json:"sessions"`
	RegisteredByID     string                    `json:"registeredById,omitempty"`
	RegisteredByAlias  string                    `json:"registeredByAlias,omitempty"`
	RegisteredByName   string                    `json:"registeredByName,omitempty"`
	RegisteredByAvatar string                    `json:"registeredByAvatar,omitempty"`
}

type CalendarSessionInfo struct {
	CheckIn  *CalendarEventCheckInfo `json:"checkIn,omitempty"`
	CheckOut *CalendarEventCheckInfo `json:"checkOut,omitempty"`
}

type CalendarEventResponse struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"`
	Title   string                 `json:"title"`
	StartAt time.Time              `json:"startAt" swaggertype:"string" format:"date-time"`
	EndAt   *time.Time             `json:"endAt,omitempty" swaggertype:"string" format:"date-time"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

type CalendarDayResponse struct {
	Date         string                  `json:"date"`
	WorkspaceID  string                  `json:"workspaceId"`
	MemberID     string                  `json:"memberId"`
	MemberAlias  string                  `json:"memberAlias,omitempty"`
	MemberName   string                  `json:"memberName,omitempty"`
	MemberAvatar string                  `json:"memberAvatar,omitempty"`
	Schedule     *CalendarScheduleInfo   `json:"schedule,omitempty"`
	Sessions     []CalendarSessionInfo   `json:"sessions,omitempty"`
	Events       []CalendarEventResponse `json:"events"`
}

type MemberCalendarDay struct {
	Date     string                `json:"date"`
	Schedule *CalendarScheduleInfo `json:"schedule,omitempty"`
	Sessions []CalendarSessionInfo `json:"sessions,omitempty"`
}

type MemberCalendarResponse struct {
	MemberID     string              `json:"memberId"`
	MemberAlias  string              `json:"memberAlias"`
	MemberName   string              `json:"memberName"`
	MemberAvatar string              `json:"memberAvatar,omitempty"`
	Days         []MemberCalendarDay `json:"days"`
}

type workScheduleDay struct {
	ID                 string
	WorkspaceID        string
	MemberID           string
	MemberAlias        string
	MemberName         string
	MemberAvatar       string
	Date               string
	Status             string
	Sessions           []workScheduleSession
	RegisteredByID     string
	RegisteredByAlias  string
	RegisteredByName   string
	RegisteredByAvatar string
}

type workScheduleSession struct {
	ID      string
	StartAt time.Time
	EndAt   time.Time
}

type attendanceRecord struct {
	ID             string
	WorkspaceID    string
	MemberID       string
	MemberAlias    string
	MemberName     string
	MemberAvatar   string
	Type           string
	RecordedAt     time.Time
	DistanceMeters float64
}

func getDaySessions(dateStr string, memberID string, records []attendanceRecord) []CalendarSessionInfo {
	var dayRecords []*attendanceRecord
	for i := range records {
		rec := &records[i]
		if rec.MemberID != memberID {
			continue
		}
		if rec.RecordedAt.In(vietnamLocation()).Format(time.DateOnly) != dateStr {
			continue
		}
		dayRecords = append(dayRecords, rec)
	}
	sort.Slice(dayRecords, func(i, j int) bool {
		return dayRecords[i].RecordedAt.Before(dayRecords[j].RecordedAt)
	})

	type pair struct{ in, out *CalendarEventCheckInfo }
	var pairs []pair
	var sessions []CalendarSessionInfo
	var openIns []*CalendarEventCheckInfo
	for _, rec := range dayRecords {
		info := &CalendarEventCheckInfo{
			ID:             rec.ID,
			RecordedAt:     rec.RecordedAt,
			DistanceMeters: rec.DistanceMeters,
		}
		switch rec.Type {
		case attendanceTypeCheckIn:
			openIns = append(openIns, info)
		case attendanceTypeCheckOut:
			if len(openIns) > 0 {
				in := openIns[0]
				openIns = openIns[1:]
				pairs = append(pairs, pair{in, info})
			} else {
				sessions = append(sessions, CalendarSessionInfo{CheckOut: info})
			}
		}
	}
	for _, in := range openIns {
		sessions = append(sessions, CalendarSessionInfo{CheckIn: in})
	}

	sort.SliceStable(pairs, func(i, j int) bool { return pairs[i].in.RecordedAt.Before(pairs[j].in.RecordedAt) })
	for i := 0; i < len(pairs); {
		groupIn, groupOut := pairs[i].in, pairs[i].out
		j := i + 1
		for j < len(pairs) && !pairs[j].in.RecordedAt.After(groupOut.RecordedAt) {
			if pairs[j].out.RecordedAt.After(groupOut.RecordedAt) {
				groupOut = pairs[j].out
			}
			j++
		}
		sessions = append(sessions, CalendarSessionInfo{CheckIn: groupIn, CheckOut: groupOut})
		i = j
	}

	sessionStart := func(s CalendarSessionInfo) time.Time {
		if s.CheckIn != nil {
			return s.CheckIn.RecordedAt
		}
		return s.CheckOut.RecordedAt
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessionStart(sessions[i]).Before(sessionStart(sessions[j]))
	})
	return sessions
}

func eventsFromSources(
	schedules []workScheduleDay,
	records []attendanceRecord,
	birthdays []birthdayEventSource,
	holidays []database.Holiday,
	fromDate, toDate time.Time,
	hasFrom, hasTo bool,
) []CalendarDayResponse {
	daysMap := make(map[string]*CalendarDayResponse)

	getOrCreateDay := func(dateStr, workspaceID, memberID, memberAlias, memberName, memberAvatar string) *CalendarDayResponse {
		if day, ok := daysMap[dateStr]; ok {
			if day.WorkspaceID == "" {
				day.WorkspaceID = workspaceID
			}
			if day.MemberID == "" {
				day.MemberID = memberID
			}
			if day.MemberAlias == "" {
				day.MemberAlias = memberAlias
			}
			if day.MemberName == "" {
				day.MemberName = memberName
			}
			if day.MemberAvatar == "" {
				day.MemberAvatar = memberAvatar
			}
			return day
		}
		day := &CalendarDayResponse{
			Date:         dateStr,
			WorkspaceID:  workspaceID,
			MemberID:     memberID,
			MemberAlias:  memberAlias,
			MemberName:   memberName,
			MemberAvatar: memberAvatar,
			Events:       make([]CalendarEventResponse, 0),
		}
		daysMap[dateStr] = day
		return day
	}

	for _, schedule := range schedules {
		day := getOrCreateDay(schedule.Date, schedule.WorkspaceID, schedule.MemberID, schedule.MemberAlias, schedule.MemberName, schedule.MemberAvatar)
		schInfo := &CalendarScheduleInfo{
			ID:                 schedule.ID,
			Status:             schedule.Status,
			Sessions:           make([]CalendarScheduleSession, 0, len(schedule.Sessions)),
			RegisteredByID:     schedule.RegisteredByID,
			RegisteredByAlias:  schedule.RegisteredByAlias,
			RegisteredByName:   schedule.RegisteredByName,
			RegisteredByAvatar: schedule.RegisteredByAvatar,
		}
		for _, session := range schedule.Sessions {
			schInfo.Sessions = append(schInfo.Sessions, CalendarScheduleSession{
				ID:      session.ID,
				StartAt: session.StartAt,
				EndAt:   session.EndAt,
			})
		}
		day.Schedule = schInfo
		if sessions := getDaySessions(schedule.Date, schedule.MemberID, records); len(sessions) > 0 {
			day.Sessions = sessions
		}
	}

	for _, record := range records {
		dateStr := record.RecordedAt.In(vietnamLocation()).Format(time.DateOnly)
		day := getOrCreateDay(dateStr, record.WorkspaceID, record.MemberID, record.MemberAlias, record.MemberName, record.MemberAvatar)
		if len(day.Sessions) == 0 {
			if sessions := getDaySessions(dateStr, record.MemberID, records); len(sessions) > 0 {
				day.Sessions = sessions
			}
		}
	}

	if hasFrom && hasTo {
		for _, b := range birthdays {
			birthMonth := b.Birthday.Month()
			birthDay := b.Birthday.Day()

			for y := fromDate.Year(); y <= toDate.Year(); y++ {
				candidate := time.Date(y, birthMonth, birthDay, 0, 0, 0, 0, vietnamLocation())
				if !candidate.Before(fromDate) && !candidate.After(toDate) {
					dateStr := candidate.Format(time.DateOnly)
					endAt := candidate.Add(24*time.Hour - 1*time.Second)

					if _, exists := daysMap[dateStr]; !exists {
						daysMap[dateStr] = &CalendarDayResponse{
							Date:   dateStr,
							Events: make([]CalendarEventResponse, 0),
						}
					}
					day := daysMap[dateStr]
					day.Events = append(day.Events, CalendarEventResponse{
						ID:      fmt.Sprintf("birthday:%s:%s", b.MemberID, dateStr),
						Type:    EventTypeBirthday,
						Title:   fmt.Sprintf("Sinh nhật của %s", b.MemberName),
						StartAt: candidate,
						EndAt:   &endAt,
						Data: map[string]interface{}{
							"memberId":    b.MemberID,
							"memberAlias": b.MemberAlias,
							"memberName":  b.MemberName,
							"birthday":    b.Birthday.Format(time.DateOnly),
						},
					})
				}
			}
		}

		for _, h := range holidays {
			var dates []time.Time
			if !h.Year.Valid {
				if h.DateType == "solar" {
					for y := fromDate.Year(); y <= toDate.Year(); y++ {
						candidate := time.Date(y, time.Month(h.Month), int(h.Day), 0, 0, 0, 0, vietnamLocation())
						if !candidate.Before(fromDate) && !candidate.After(toDate) {
							dates = append(dates, candidate)
						}
					}
				}
			} else {
				candidate := time.Date(int(h.Year.Int32), time.Month(h.Month), int(h.Day), 0, 0, 0, 0, vietnamLocation())
				if !candidate.Before(fromDate) && !candidate.After(toDate) {
					dates = append(dates, candidate)
				}
			}

			for _, date := range dates {
				dateStr := date.Format(time.DateOnly)
				endAt := date.Add(24*time.Hour - 1*time.Second)

				if _, exists := daysMap[dateStr]; !exists {
					daysMap[dateStr] = &CalendarDayResponse{
						Date:   dateStr,
						Events: make([]CalendarEventResponse, 0),
					}
				}
				day := daysMap[dateStr]

				nameEn := ""
				if h.NameEn.Valid {
					nameEn = h.NameEn.String
				}
				desc := ""
				if h.Description.Valid {
					desc = h.Description.String
				}

				day.Events = append(day.Events, CalendarEventResponse{
					ID:      fmt.Sprintf("holiday:%s:%s", pgutil.UUIDToString(h.ID), dateStr),
					Type:    "holiday",
					Title:   h.Name,
					StartAt: date,
					EndAt:   &endAt,
					Data: map[string]interface{}{
						"nameEn":          nameEn,
						"type":            h.Type,
						"dateType":        h.DateType,
						"isPublicHoliday": h.IsPublicHoliday,
						"description":     desc,
					},
				})
			}
		}
	}

	result := make([]CalendarDayResponse, 0, len(daysMap))
	for _, day := range daysMap {
		sort.Slice(day.Events, func(i, j int) bool { return day.Events[i].StartAt.Before(day.Events[j].StartAt) })
		result = append(result, *day)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Date < result[j].Date })

	return result
}

func workScheduleDayFromRow(row database.ListWorkSchedulesByAliasRow, sessions []database.WorkScheduleSession) workScheduleDay {
	item := workScheduleDay{
		ID:                 pgutil.UUIDToString(row.ID),
		WorkspaceID:        pgutil.UUIDToString(row.WorkspaceID),
		MemberID:           pgutil.UUIDToString(row.MemberID),
		MemberAlias:        row.MemberAlias,
		MemberName:         row.MemberName,
		MemberAvatar:       textString(row.MemberAvatar),
		Date:               row.ScheduleDate.Time.Format(time.DateOnly),
		Status:             row.Status,
		RegisteredByID:     pgutil.UUIDToString(row.RegisteredBy),
		RegisteredByAlias:  textString(row.RegisteredByAlias),
		RegisteredByName:   textString(row.RegisteredByName),
		RegisteredByAvatar: textString(row.RegisteredByAvatar),
	}
	for _, session := range sessions {
		item.Sessions = append(item.Sessions, workScheduleSession{
			ID:      pgutil.UUIDToString(session.ID),
			StartAt: session.StartAt.Time,
			EndAt:   session.EndAt.Time,
		})
	}
	return item
}

func attendanceRecordFromRow(row database.ListAttendanceRecordsByAliasRow) attendanceRecord {
	return attendanceRecord{
		ID:             pgutil.UUIDToString(row.ID),
		WorkspaceID:    pgutil.UUIDToString(row.WorkspaceID),
		MemberID:       pgutil.UUIDToString(row.MemberID),
		MemberAlias:    row.MemberAlias,
		MemberName:     row.MemberName,
		MemberAvatar:   textString(row.MemberAvatar),
		Type:           row.Type,
		RecordedAt:     row.RecordedAt.Time,
		DistanceMeters: row.DistanceMeters,
	}
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(time.DateOnly, value, vietnamLocation())
}

func vietnamLocation() *time.Location {
	loc, err := time.LoadLocation(vietnamTimezone)
	if err != nil {
		return time.FixedZone(vietnamTimezone, 7*60*60)
	}
	return loc
}

func pgUUIDOrNull(value string) (pgtype.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return pgtype.UUID{}, nil
	}
	return pgutil.ParseUUID(value)
}

func parseOptionalTime(value string, endOfDay bool) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, true, nil
	}
	if parsed, err := parseDate(value); err == nil {
		if endOfDay {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
		}
		return parsed, true, nil
	}
	return time.Time{}, false, fmt.Errorf("Thời gian phải theo định dạng RFC3339 hoặc YYYY-MM-DD")
}

func parseOptionalDate(value string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, nil
	}
	if parsed, err := parseDate(value); err == nil {
		return parsed, true, nil
	}
	parsed, ok, err := parseOptionalTime(value, false)
	if err != nil || !ok {
		return time.Time{}, ok, err
	}
	return time.Date(parsed.In(vietnamLocation()).Year(), parsed.In(vietnamLocation()).Month(), parsed.In(vietnamLocation()).Day(), 0, 0, 0, 0, vietnamLocation()), true, nil
}

func pgTimeOrNull(value time.Time, valid bool) pgtype.Timestamptz {
	if !valid {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func pgDateOrNull(value time.Time, valid bool) pgtype.Date {
	if !valid {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: value, Valid: true}
}

func memberEventsFromSources(
	members []database.ListWorkspaceMembersForStatisticsRow,
	schedules []workScheduleDay,
	records []attendanceRecord,
) []MemberCalendarResponse {
	schedulesByMember := make(map[string][]workScheduleDay)
	for _, s := range schedules {
		schedulesByMember[s.MemberID] = append(schedulesByMember[s.MemberID], s)
	}
	recordsByMember := make(map[string][]attendanceRecord)
	for _, r := range records {
		recordsByMember[r.MemberID] = append(recordsByMember[r.MemberID], r)
	}

	result := make([]MemberCalendarResponse, 0, len(members))
	for _, m := range members {
		memberID := pgutil.UUIDToString(m.MemberID)
		memberSchedules := schedulesByMember[memberID]
		memberRecords := recordsByMember[memberID]

		daysMap := make(map[string]*MemberCalendarDay)
		getOrCreate := func(date string) *MemberCalendarDay {
			if d, ok := daysMap[date]; ok {
				return d
			}
			d := &MemberCalendarDay{Date: date}
			daysMap[date] = d
			return d
		}

		for _, sch := range memberSchedules {
			day := getOrCreate(sch.Date)
			schInfo := &CalendarScheduleInfo{
				ID:                 sch.ID,
				Status:             sch.Status,
				Sessions:           make([]CalendarScheduleSession, 0, len(sch.Sessions)),
				RegisteredByID:     sch.RegisteredByID,
				RegisteredByAlias:  sch.RegisteredByAlias,
				RegisteredByName:   sch.RegisteredByName,
				RegisteredByAvatar: sch.RegisteredByAvatar,
			}
			for _, sess := range sch.Sessions {
				schInfo.Sessions = append(schInfo.Sessions, CalendarScheduleSession{
					ID:      sess.ID,
					StartAt: sess.StartAt,
					EndAt:   sess.EndAt,
				})
			}
			day.Schedule = schInfo
			if sessions := getDaySessions(sch.Date, memberID, memberRecords); len(sessions) > 0 {
				day.Sessions = sessions
			}
		}

		for _, rec := range memberRecords {
			dateStr := rec.RecordedAt.In(vietnamLocation()).Format(time.DateOnly)
			day := getOrCreate(dateStr)
			if len(day.Sessions) == 0 {
				if sessions := getDaySessions(dateStr, memberID, memberRecords); len(sessions) > 0 {
					day.Sessions = sessions
				}
			}
		}

		days := make([]MemberCalendarDay, 0, len(daysMap))
		for _, d := range daysMap {
			days = append(days, *d)
		}
		sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })

		result = append(result, MemberCalendarResponse{
			MemberID:     memberID,
			MemberAlias:  m.MemberAlias,
			MemberName:   m.MemberName,
			MemberAvatar: m.MemberAvatar.String,
			Days:         days,
		})
	}
	return result
}
