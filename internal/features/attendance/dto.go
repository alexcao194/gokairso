package attendance

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

	NotificationTypeWorkScheduleRegistration   = "work_schedule_registration"
	NotificationTypeWorkScheduleApproved       = "work_schedule_approved"
	NotificationTypeWorkScheduleRejected       = "work_schedule_rejected"
	NotificationTypeAttendanceCheckInReminder  = "attendance_check_in_reminder"
	NotificationTypeAttendanceCheckOutReminder = "attendance_check_out_reminder"

	NotificationTypeCheckRequestSubmitted = "attendance_correction_request_submitted"
	NotificationTypeCheckRequestApproved  = "attendance_correction_request_approved"
	NotificationTypeCheckRequestRejected  = "attendance_correction_request_rejected"

	NotificationTypeDailyReportReminder = "daily_report_reminder"
)

type WorkScheduleStatus string

const (
	WorkScheduleStatusPending  WorkScheduleStatus = "pending"
	WorkScheduleStatusApproved WorkScheduleStatus = "approved"
	WorkScheduleStatusRejected WorkScheduleStatus = "rejected"
)

type AttendanceCorrectionStatus string

const (
	AttendanceCorrectionStatusPending  AttendanceCorrectionStatus = "pending"
	AttendanceCorrectionStatusApproved AttendanceCorrectionStatus = "approved"
	AttendanceCorrectionStatusRejected AttendanceCorrectionStatus = "rejected"
)

func textString(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

type AttendanceType string

const (
	AttendanceTypeCheckIn  AttendanceType = "check_in"
	AttendanceTypeCheckOut AttendanceType = "check_out"
)

type UpsertAttendanceSettingRequest struct {
	CompanyWifiBSSID    string  `json:"companyWifiBSSID"`
	CompanyWifiSSID     string  `json:"companyWifiSSID,omitempty"`
	CompanyLatitude     float64 `json:"companyLatitude"`
	CompanyLongitude    float64 `json:"companyLongitude"`
	AllowedRadiusMeters int     `json:"allowedRadiusMeters"`
	ValidationEnabled   bool    `json:"validationEnabled"`
}

type AttendanceCheckRequest struct {
	Latitude   float64    `json:"latitude"`
	Longitude  float64    `json:"longitude"`
	WifiBSSID  string     `json:"wifiBSSID"`
	WifiSSID   string     `json:"wifiSSID,omitempty"`
	RecordedAt *time.Time `json:"recordedAt,omitempty" swaggertype:"string" format:"date-time"`
}

type AttendanceStatusResponse struct {
	Valid          bool           `json:"valid"`
	ValidationMsg  string         `json:"validationMsg,omitempty"`
	NextAction     AttendanceType `json:"nextAction"`
	DistanceMeters float64        `json:"distanceMeters"`
	IsConfigured   bool           `json:"isConfigured"`
}

type AttendanceRecordFilter struct {
	From     string `schema:"from"`
	To       string `schema:"to"`
	MemberID string `schema:"memberId"`
}

type AttendanceHalfEdit struct {
	RecordID   string    `json:"recordId,omitempty"`
	Enabled    bool      `json:"enabled"`
	RecordedAt time.Time `json:"recordedAt,omitempty" swaggertype:"string" format:"date-time"`
}

type UpsertAttendanceSessionRequest struct {
	MemberID string             `json:"memberId"`
	CheckIn  AttendanceHalfEdit `json:"checkIn"`
	CheckOut AttendanceHalfEdit `json:"checkOut"`
}

type AttendanceSessionResponse struct {
	CheckIn  *AttendanceRecordResponse `json:"checkIn,omitempty"`
	CheckOut *AttendanceRecordResponse `json:"checkOut,omitempty"`
}

type SubmitAttendanceCorrectionRequest struct {
	CheckInAt              *time.Time `json:"checkInAt,omitempty" swaggertype:"string" format:"date-time"`
	CheckOutAt             *time.Time `json:"checkOutAt,omitempty" swaggertype:"string" format:"date-time"`
	TargetCheckInRecordID  string     `json:"targetCheckInRecordId,omitempty"`
	TargetCheckOutRecordID string     `json:"targetCheckOutRecordId,omitempty"`
	Reason                 string     `json:"reason"`
}

type AttendanceCorrectionRequestFilter struct {
	MemberID string `schema:"memberId"`
	Status   string `schema:"status"`
}

type UpsertMemberWorkSchedulesRequest struct {
	Schedules []UpsertWorkScheduleRequest `json:"schedules"`
	WeekNote  string                      `json:"weekNote,omitempty"`
}

type WorkScheduleSessionRequest struct {
	StartAt time.Time `json:"startAt" swaggertype:"string" format:"date-time"`
	EndAt   time.Time `json:"endAt" swaggertype:"string" format:"date-time"`
}

type UpsertWorkScheduleRequest struct {
	Date     string                       `json:"date"`
	Sessions []WorkScheduleSessionRequest `json:"sessions"`
	Note     string                       `json:"note,omitempty"`
}

type UpsertWorkSchedulesRequest struct {
	Schedules []UpsertWorkScheduleRequest `json:"schedules"`
	WeekNote  string                      `json:"weekNote,omitempty"`
}

type UpdateWorkScheduleStatusRequest struct {
	DayIDs []string `json:"dayIds"`
}

type WorkScheduleFilter struct {
	Month    string `schema:"month"`
	From     string `schema:"from"`
	To       string `schema:"to"`
	MemberID string `schema:"memberId"`
	Status   string `schema:"status"`
}

type AttendanceStatisticsFilter struct {
	Month    string `schema:"month"`
	Week     string `schema:"week"`
	Year     string `schema:"year"`
	MemberID string `schema:"memberId"`
}

type AttendanceSettingResponse struct {
	WorkspaceID         string    `json:"workspaceId"`
	CreatedAt           time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt           time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
	CompanyWifiBSSID    string    `json:"companyWifiBSSID"`
	CompanyWifiSSID     string    `json:"companyWifiSSID,omitempty"`
	CompanyLatitude     float64   `json:"companyLatitude"`
	CompanyLongitude    float64   `json:"companyLongitude"`
	AllowedRadiusMeters int       `json:"allowedRadiusMeters"`
	ValidationEnabled   bool      `json:"validationEnabled"`
}

type AttendanceRecordResponse struct {
	ID             string         `json:"id"`
	CreatedAt      time.Time      `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt      time.Time      `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID    string         `json:"workspaceId"`
	MemberID       string         `json:"memberId"`
	MemberAlias    string         `json:"memberAlias,omitempty"`
	MemberName     string         `json:"memberName,omitempty"`
	MemberAvatar   string         `json:"memberAvatar,omitempty"`
	Type           AttendanceType `json:"type"`
	RecordedAt     time.Time      `json:"recordedAt" swaggertype:"string" format:"date-time"`
	Latitude       float64        `json:"latitude"`
	Longitude      float64        `json:"longitude"`
	WifiBSSID      string         `json:"wifiBSSID"`
	WifiSSID       string         `json:"wifiSSID,omitempty"`
	DistanceMeters float64        `json:"distanceMeters"`
}

type AttendanceCorrectionRequestResponse struct {
	ID           string                     `json:"id"`
	CreatedAt    time.Time                  `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt    time.Time                  `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID  string                     `json:"workspaceId"`
	MemberID     string                     `json:"memberId"`
	MemberAlias  string                     `json:"memberAlias,omitempty"`
	MemberName   string                     `json:"memberName,omitempty"`
	MemberAvatar string                     `json:"memberAvatar,omitempty"`
	Reason       string                     `json:"reason"`
	Status       AttendanceCorrectionStatus `json:"status"`

	CheckInAt  *time.Time `json:"checkInAt,omitempty" swaggertype:"string" format:"date-time"`
	CheckOutAt *time.Time `json:"checkOutAt,omitempty" swaggertype:"string" format:"date-time"`

	TargetCheckInRecordID  string `json:"targetCheckInRecordId,omitempty"`
	TargetCheckOutRecordID string `json:"targetCheckOutRecordId,omitempty"`

	OriginalCheckInAt  *time.Time `json:"originalCheckInAt,omitempty" swaggertype:"string" format:"date-time"`
	OriginalCheckOutAt *time.Time `json:"originalCheckOutAt,omitempty" swaggertype:"string" format:"date-time"`

	ResultCheckInRecordID  string `json:"resultCheckInRecordId,omitempty"`
	ResultCheckOutRecordID string `json:"resultCheckOutRecordId,omitempty"`

	DecidedByID    string     `json:"decidedById,omitempty"`
	DecidedByAlias string     `json:"decidedByAlias,omitempty"`
	DecidedByName  string     `json:"decidedByName,omitempty"`
	DecidedAt      *time.Time `json:"decidedAt,omitempty" swaggertype:"string" format:"date-time"`
}

type WorkScheduleDayResponse struct {
	ID                 string                        `json:"id"`
	CreatedAt          time.Time                     `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt          time.Time                     `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID        string                        `json:"workspaceId"`
	MemberID           string                        `json:"memberId"`
	MemberAlias        string                        `json:"memberAlias,omitempty"`
	MemberName         string                        `json:"memberName,omitempty"`
	MemberAvatar       string                        `json:"memberAvatar,omitempty"`
	Date               string                        `json:"date"`
	Status             WorkScheduleStatus            `json:"status,omitempty"`
	Note               string                        `json:"note,omitempty"`
	WeekNote           string                        `json:"weekNote,omitempty"`
	Sessions           []WorkScheduleSessionResponse `json:"sessions"`
	RegisteredByID     string                        `json:"registeredById,omitempty"`
	RegisteredByAlias  string                        `json:"registeredByAlias,omitempty"`
	RegisteredByName   string                        `json:"registeredByName,omitempty"`
	RegisteredByAvatar string                        `json:"registeredByAvatar,omitempty"`
}

type WorkScheduleSessionResponse struct {
	ID        string    `json:"id,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty" swaggertype:"string" format:"date-time"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" swaggertype:"string" format:"date-time"`
	DayID     string    `json:"dayId,omitempty"`
	StartAt   time.Time `json:"startAt" swaggertype:"string" format:"date-time"`
	EndAt     time.Time `json:"endAt" swaggertype:"string" format:"date-time"`
}

type DailyReportResponse struct {
	ID           string    `json:"id"`
	CreatedAt    time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt    time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID  string    `json:"workspaceId"`
	MemberID     string    `json:"memberId"`
	MemberAlias  string    `json:"memberAlias,omitempty"`
	MemberName   string    `json:"memberName,omitempty"`
	MemberAvatar string    `json:"memberAvatar,omitempty"`
	Date         string    `json:"date"`
	Done         string    `json:"done"`
	InProgress   string    `json:"inProgress"`
	Issues       string    `json:"issues"`
}

type UpsertDailyReportRequest struct {
	Date       string `json:"date"`
	Done       string `json:"done"`
	InProgress string `json:"inProgress"`
	Issues     string `json:"issues"`
}

type DailyReportFilter struct {
	MemberID string
	From     string
	To       string
}

type WorkspaceMemberBriefResponse struct {
	MemberID     string `json:"memberId"`
	MemberAlias  string `json:"memberAlias"`
	MemberName   string `json:"memberName"`
	MemberAvatar string `json:"memberAvatar,omitempty"`
}

type AttendanceStatisticsMemberResponse struct {
	MemberID           string  `json:"memberId"`
	MemberAlias        string  `json:"memberAlias"`
	MemberName         string  `json:"memberName"`
	MemberAvatar       string  `json:"memberAvatar,omitempty"`
	LateDays           int     `json:"lateDays"`
	EarlyLeaveDays     int     `json:"earlyLeaveDays"`
	RegisteredHours    float64 `json:"registeredHours"`
	AttendanceHours    float64 `json:"attendanceHours"`
	EstimatedTaskHours float64 `json:"estimatedTaskHours"`
	InitiativeEstimate float64 `json:"initiativeEstimate"`
	LateLogEstimate    float64 `json:"lateLogEstimate"`
	FeedbackEstimate   float64 `json:"feedbackEstimate"`
}

type AttendanceStatisticsSummaryResponse struct {
	WorkspaceID string                               `json:"workspaceId,omitempty"`
	Period      string                               `json:"period"`
	Label       string                               `json:"label"`
	Month       string                               `json:"month,omitempty"`
	Members     []AttendanceStatisticsMemberResponse `json:"members"`
	Totals      AttendanceStatisticsTotalsResponse   `json:"totals"`
}

type AttendanceStatisticsTotalsResponse struct {
	LateDays           int     `json:"lateDays"`
	EarlyLeaveDays     int     `json:"earlyLeaveDays"`
	RegisteredHours    float64 `json:"registeredHours"`
	AttendanceHours    float64 `json:"attendanceHours"`
	EstimatedTaskHours float64 `json:"estimatedTaskHours"`
	InitiativeEstimate float64 `json:"initiativeEstimate"`
	LateLogEstimate    float64 `json:"lateLogEstimate"`
	FeedbackEstimate   float64 `json:"feedbackEstimate"`
}

type TaskStatisticsFilter struct {
	MemberID  string `schema:"memberId"`
	ProjectID string `schema:"projectId"`
	ModuleID  string `schema:"moduleId"`
	Month     string `schema:"month"`
	Week      string `schema:"week"`
	Year      string `schema:"year"`
}

type TaskStatsMemberResponse struct {
	MemberID                string  `json:"memberId"`
	MemberAlias             string  `json:"memberAlias"`
	MemberName              string  `json:"memberName"`
	MemberAvatar            string  `json:"memberAvatar,omitempty"`
	TotalTasks              int64   `json:"totalTasks"`
	Backlog                 int64   `json:"backlog"`
	Todo                    int64   `json:"todo"`
	InProgress              int64   `json:"inProgress"`
	Review                  int64   `json:"review"`
	Done                    int64   `json:"done"`
	Canceled                int64   `json:"canceled"`
	Priority1               int64   `json:"priority1"`
	Priority2               int64   `json:"priority2"`
	Priority3               int64   `json:"priority3"`
	Priority4               int64   `json:"priority4"`
	TotalEstimation         float64 `json:"totalEstimation"`
	TotalOverdue            int64   `json:"totalOverdue"`
	TotalEstimatedTaskHours float64 `json:"totalEstimatedTaskHours"`
	TotalLate               float64 `json:"totalLate"`
	TotalBug                float64 `json:"totalBug"`
}

type TaskStatsTotalsResponse struct {
	TotalTasks              int64   `json:"totalTasks"`
	Backlog                 int64   `json:"backlog"`
	Todo                    int64   `json:"todo"`
	InProgress              int64   `json:"inProgress"`
	Review                  int64   `json:"review"`
	Done                    int64   `json:"done"`
	Canceled                int64   `json:"canceled"`
	Priority1               int64   `json:"priority1"`
	Priority2               int64   `json:"priority2"`
	Priority3               int64   `json:"priority3"`
	Priority4               int64   `json:"priority4"`
	TotalEstimation         float64 `json:"totalEstimation"`
	TotalOverdue            int64   `json:"totalOverdue"`
	TotalEstimatedTaskHours float64 `json:"totalEstimatedTaskHours"`
	TotalLate               float64 `json:"totalLate"`
	TotalBug                float64 `json:"totalBug"`
	AvgCycleDays            float64 `json:"avgCycleDays"`
	CycleTimeSampleSize     int64   `json:"cycleTimeSampleSize"`
}

type TaskStatisticsSummaryResponse struct {
	Period    string                    `json:"period,omitempty"`
	Label     string                    `json:"label,omitempty"`
	ProjectID string                    `json:"projectId,omitempty"`
	ModuleID  string                    `json:"moduleId,omitempty"`
	Members   []TaskStatsMemberResponse `json:"members"`
	Totals    TaskStatsTotalsResponse   `json:"totals"`
}

type MyAttendanceStatisticsResponse struct {
	WorkspaceID        string  `json:"workspaceId,omitempty"`
	Period             string  `json:"period"`
	Label              string  `json:"label"`
	Month              string  `json:"month,omitempty"`
	MemberID           string  `json:"memberId"`
	MemberAlias        string  `json:"memberAlias"`
	MemberName         string  `json:"memberName"`
	MemberAvatar       string  `json:"memberAvatar,omitempty"`
	LateDays           int     `json:"lateDays"`
	EarlyLeaveDays     int     `json:"earlyLeaveDays"`
	RegisteredHours    float64 `json:"registeredHours"`
	AttendanceHours    float64 `json:"attendanceHours"`
	EstimatedTaskHours float64 `json:"estimatedTaskHours"`
	InitiativeEstimate float64 `json:"initiativeEstimate"`
	LateLogEstimate    float64 `json:"lateLogEstimate"`
	FeedbackEstimate   float64 `json:"feedbackEstimate"`
}

type MyTaskStatisticsResponse struct {
	Period                  string  `json:"period,omitempty"`
	Label                   string  `json:"label,omitempty"`
	ProjectID               string  `json:"projectId,omitempty"`
	ModuleID                string  `json:"moduleId,omitempty"`
	MemberID                string  `json:"memberId"`
	MemberAlias             string  `json:"memberAlias"`
	MemberName              string  `json:"memberName"`
	MemberAvatar            string  `json:"memberAvatar,omitempty"`
	TotalTasks              int64   `json:"totalTasks"`
	Backlog                 int64   `json:"backlog"`
	Todo                    int64   `json:"todo"`
	InProgress              int64   `json:"inProgress"`
	Review                  int64   `json:"review"`
	Done                    int64   `json:"done"`
	Canceled                int64   `json:"canceled"`
	Priority1               int64   `json:"priority1"`
	Priority2               int64   `json:"priority2"`
	Priority3               int64   `json:"priority3"`
	Priority4               int64   `json:"priority4"`
	TotalEstimation         float64 `json:"totalEstimation"`
	TotalOverdue            int64   `json:"totalOverdue"`
	TotalEstimatedTaskHours float64 `json:"totalEstimatedTaskHours"`
	TotalLate               float64 `json:"totalLate"`
	TotalBug                float64 `json:"totalBug"`
}

type WorkWeekSessionRequest struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type WorkWeekDayRequest struct {
	DayOfWeek int                      `json:"dayOfWeek"`
	Sessions  []WorkWeekSessionRequest `json:"sessions"`
}

type UpsertWorkWeekRequest struct {
	Days []WorkWeekDayRequest `json:"days"`
}

type WorkWeekSessionResponse struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

type WorkWeekDayResponse struct {
	DayOfWeek int                       `json:"dayOfWeek"`
	Sessions  []WorkWeekSessionResponse `json:"sessions"`
}

type WorkWeekResponse struct {
	WorkspaceID string                `json:"workspaceId"`
	Days        []WorkWeekDayResponse `json:"days"`
}

type ScheduleNotificationEvent struct {
	Type           string
	RecipientID    string
	WorkspaceID    string
	WorkspaceAlias string
	MemberID       string
	MemberName     string
	FromDate       string
	ToDate         string
}

type CheckRequestNotificationEvent struct {
	Type           string
	RecipientID    string
	WorkspaceID    string
	WorkspaceAlias string
	RequestID      string
	MemberID       string
	MemberName     string
	CheckInAt      string
	CheckOutAt     string
}

type AttendanceReminder struct {
	SessionID    string
	WorkspaceID  string
	MemberID     string
	Date         string
	StartAt      time.Time
	EndAt        time.Time
	ReminderType AttendanceType
}

type AttendanceReminderEvent struct {
	Type         string
	RecipientID  string
	WorkspaceID  string
	SessionID    string
	ScheduleDate string
	SessionTime  string
}

type DailyReportReminder struct {
	WorkspaceID string
	MemberID    string
	Date        string
	LastEndAt   time.Time
}

type DailyReportReminderEvent struct {
	RecipientID string
	WorkspaceID string
	Date        string
	LastEndAt   string
}

type createAttendanceRecordParams struct {
	Type           AttendanceType
	RecordedAt     time.Time
	Latitude       float64
	Longitude      float64
	WifiBSSID      string
	WifiSSID       string
	DistanceMeters float64
}

type createCheckRequestParams struct {
	Reason                 string
	CheckInAt              *time.Time
	CheckOutAt             *time.Time
	TargetCheckInRecordID  string
	TargetCheckOutRecordID string
	OriginalCheckInAt      *time.Time
	OriginalCheckOutAt     *time.Time
}

func normalizeAttendanceSettingRequest(req *UpsertAttendanceSettingRequest) error {
	req.CompanyWifiBSSID = strings.TrimSpace(req.CompanyWifiBSSID)
	req.CompanyWifiSSID = strings.TrimSpace(req.CompanyWifiSSID)
	if req.CompanyWifiBSSID == "" {
		return fmt.Errorf("companyWifiBSSID là bắt buộc")
	}
	if req.AllowedRadiusMeters <= 0 {
		return fmt.Errorf("allowedRadiusMeters phải lớn hơn 0")
	}
	if req.CompanyLatitude < -90 || req.CompanyLatitude > 90 {
		return fmt.Errorf("companyLatitude phải nằm trong khoảng -90 đến 90")
	}
	if req.CompanyLongitude < -180 || req.CompanyLongitude > 180 {
		return fmt.Errorf("companyLongitude phải nằm trong khoảng -180 đến 180")
	}
	return nil
}

func normalizeAttendanceCheckRequest(req *AttendanceCheckRequest) error {
	req.WifiBSSID = strings.TrimSpace(req.WifiBSSID)
	req.WifiSSID = strings.TrimSpace(req.WifiSSID)
	if req.Latitude < -90 || req.Latitude > 90 {
		return fmt.Errorf("latitude phải nằm trong khoảng -90 đến 90")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return fmt.Errorf("longitude phải nằm trong khoảng -180 đến 180")
	}
	return nil
}

func normalizeUpsertAttendanceSessionRequest(req *UpsertAttendanceSessionRequest) error {
	req.MemberID = strings.TrimSpace(req.MemberID)
	if req.MemberID == "" {
		return fmt.Errorf("%w: memberId là bắt buộc", ErrAttendanceValidationFailed)
	}
	req.CheckIn.RecordID = strings.TrimSpace(req.CheckIn.RecordID)
	req.CheckOut.RecordID = strings.TrimSpace(req.CheckOut.RecordID)

	touchesCheckIn := req.CheckIn.Enabled || req.CheckIn.RecordID != ""
	touchesCheckOut := req.CheckOut.Enabled || req.CheckOut.RecordID != ""
	if !touchesCheckIn && !touchesCheckOut {
		return fmt.Errorf("%w: cần thay đổi ít nhất giờ vào hoặc giờ ra", ErrAttendanceValidationFailed)
	}

	now := time.Now()
	if req.CheckIn.Enabled {
		if req.CheckIn.RecordedAt.IsZero() {
			return fmt.Errorf("%w: checkIn.recordedAt là bắt buộc khi bật checkIn", ErrAttendanceValidationFailed)
		}
		if req.CheckIn.RecordedAt.After(now) {
			return fmt.Errorf("%w: checkIn.recordedAt không được ở tương lai", ErrAttendanceValidationFailed)
		}
	}
	if req.CheckOut.Enabled {
		if req.CheckOut.RecordedAt.IsZero() {
			return fmt.Errorf("%w: checkOut.recordedAt là bắt buộc khi bật checkOut", ErrAttendanceValidationFailed)
		}
		if req.CheckOut.RecordedAt.After(now) {
			return fmt.Errorf("%w: checkOut.recordedAt không được ở tương lai", ErrAttendanceValidationFailed)
		}
	}
	if req.CheckIn.Enabled && req.CheckOut.Enabled && !req.CheckIn.RecordedAt.Before(req.CheckOut.RecordedAt) {
		return fmt.Errorf("%w: giờ vào phải trước giờ ra", ErrAttendanceValidationFailed)
	}
	return nil
}

const maxCheckRequestReasonLength = 500

func normalizeSubmitAttendanceCorrectionRequest(req *SubmitAttendanceCorrectionRequest) error {
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" {
		return fmt.Errorf("%w: lý do là bắt buộc", ErrAttendanceValidationFailed)
	}
	if len(req.Reason) > maxCheckRequestReasonLength {
		return fmt.Errorf("%w: lý do tối đa %d ký tự", ErrAttendanceValidationFailed, maxCheckRequestReasonLength)
	}
	if req.CheckInAt == nil && req.CheckOutAt == nil {
		return fmt.Errorf("%w: cần ít nhất checkInAt hoặc checkOutAt", ErrAttendanceValidationFailed)
	}
	now := time.Now()
	if req.CheckInAt != nil && req.CheckInAt.After(now) {
		return fmt.Errorf("%w: checkInAt không được ở tương lai", ErrAttendanceValidationFailed)
	}
	if req.CheckOutAt != nil && req.CheckOutAt.After(now) {
		return fmt.Errorf("%w: checkOutAt không được ở tương lai", ErrAttendanceValidationFailed)
	}
	if req.CheckInAt != nil && req.CheckOutAt != nil && !req.CheckInAt.Before(*req.CheckOutAt) {
		return fmt.Errorf("%w: checkInAt phải trước checkOutAt", ErrAttendanceValidationFailed)
	}
	req.TargetCheckInRecordID = strings.TrimSpace(req.TargetCheckInRecordID)
	req.TargetCheckOutRecordID = strings.TrimSpace(req.TargetCheckOutRecordID)
	if req.TargetCheckInRecordID != "" && req.CheckInAt == nil {
		return fmt.Errorf("%w: checkInAt là bắt buộc khi sửa lần check-in đã có", ErrAttendanceValidationFailed)
	}
	if req.TargetCheckOutRecordID != "" && req.CheckOutAt == nil {
		return fmt.Errorf("%w: checkOutAt là bắt buộc khi sửa lần check-out đã có", ErrAttendanceValidationFailed)
	}
	return nil
}

func normalizeWorkScheduleRequest(req *UpsertWorkScheduleRequest) error {
	req.Date = strings.TrimSpace(req.Date)
	if _, err := parseDate(req.Date); err != nil {
		return fmt.Errorf("%w: ngày phải theo định dạng YYYY-MM-DD", ErrInvalidWorkSchedule)
	}
	if len(req.Sessions) > 3 {
		return fmt.Errorf("%w: mỗi ngày tối đa 3 ca", ErrInvalidWorkSchedule)
	}
	type parsedSession struct{ start, end time.Time }
	parsed := make([]parsedSession, 0, len(req.Sessions))
	for i := range req.Sessions {
		start, end := req.Sessions[i].StartAt, req.Sessions[i].EndAt
		if start.IsZero() || end.IsZero() || !start.Before(end) {
			return fmt.Errorf("%w: cần startAt và endAt hợp lệ", ErrInvalidWorkSchedule)
		}
		parsed = append(parsed, parsedSession{start: start, end: end})
	}
	sort.Slice(parsed, func(i, j int) bool { return parsed[i].start.Before(parsed[j].start) })
	for i := 1; i < len(parsed); i++ {
		if parsed[i].start.Before(parsed[i-1].end) {
			return fmt.Errorf("%w: các ca làm việc không được chồng lấn", ErrInvalidWorkSchedule)
		}
	}
	return nil
}

func normalizeUpdateWorkScheduleStatusRequest(req *UpdateWorkScheduleStatusRequest) error {
	ids := make([]string, 0, len(req.DayIDs))
	for _, id := range req.DayIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	req.DayIDs = ids
	if len(req.DayIDs) == 0 {
		return fmt.Errorf("%w: cần ít nhất một dayId", ErrInvalidWorkSchedule)
	}
	return nil
}

func normalizeWorkSchedulesRequest(req *UpsertWorkSchedulesRequest) error {
	if len(req.Schedules) == 0 {
		return fmt.Errorf("%w: cần ít nhất một lịch", ErrInvalidWorkSchedule)
	}
	for i := range req.Schedules {
		if err := normalizeWorkScheduleRequest(&req.Schedules[i]); err != nil {
			return err
		}
	}
	return nil
}

func attendanceSettingResponseFromDatabase(row database.WorkspaceAttendanceSetting) AttendanceSettingResponse {
	return AttendanceSettingResponse{
		WorkspaceID:         pgutil.UUIDToString(row.WorkspaceID),
		CreatedAt:           row.CreatedAt.Time,
		UpdatedAt:           row.UpdatedAt.Time,
		CompanyWifiBSSID:    row.CompanyWifiBssid,
		CompanyWifiSSID:     row.CompanyWifiSsid,
		CompanyLatitude:     row.CompanyLatitude,
		CompanyLongitude:    row.CompanyLongitude,
		AllowedRadiusMeters: int(row.AllowedRadiusMeters),
		ValidationEnabled:   row.ValidationEnabled,
	}
}

func attendanceReminderFromClaimRow(row database.ClaimDueAttendanceRemindersRow) AttendanceReminder {
	return AttendanceReminder{
		SessionID:    pgutil.UUIDToString(row.SessionID),
		WorkspaceID:  pgutil.UUIDToString(row.WorkspaceID),
		MemberID:     pgutil.UUIDToString(row.MemberID),
		Date:         row.ScheduleDate.Time.Format(time.DateOnly),
		StartAt:      row.StartAt.Time,
		EndAt:        row.EndAt.Time,
		ReminderType: AttendanceType(row.ReminderType),
	}
}

func dailyReportResponseFromDatabase(row database.DailyReport) DailyReportResponse {
	return DailyReportResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
		MemberID:    pgutil.UUIDToString(row.MemberID),
		Date:        row.ReportDate.Time.Format(time.DateOnly),
		Done:        row.Done,
		InProgress:  row.InProgress,
		Issues:      row.Issues,
	}
}

func dailyReportResponseFromListRow(row database.ListDailyReportsByAliasRow) DailyReportResponse {
	resp := dailyReportResponseFromDatabase(database.DailyReport{
		ID:          row.ID,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
		WorkspaceID: row.WorkspaceID,
		MemberID:    row.MemberID,
		ReportDate:  row.ReportDate,
		Done:        row.Done,
		InProgress:  row.InProgress,
		Issues:      row.Issues,
	})
	resp.MemberAlias = row.MemberAlias
	resp.MemberName = row.MemberName
	resp.MemberAvatar = textString(row.MemberAvatar)
	return resp
}

func dailyReportReminderFromClaimRow(row database.ClaimDueDailyReportRemindersRow) DailyReportReminder {
	return DailyReportReminder{
		WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
		MemberID:    pgutil.UUIDToString(row.MemberID),
		Date:        row.ScheduleDate.Time.Format(time.DateOnly),
		LastEndAt:   row.LastEnd.Time,
	}
}

func attendanceRecordResponseFromDatabase(row database.AttendanceRecord) AttendanceRecordResponse {
	return AttendanceRecordResponse{
		ID:             pgutil.UUIDToString(row.ID),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
		WorkspaceID:    pgutil.UUIDToString(row.WorkspaceID),
		MemberID:       pgutil.UUIDToString(row.MemberID),
		Type:           AttendanceType(row.Type),
		RecordedAt:     row.RecordedAt.Time,
		Latitude:       row.Latitude,
		Longitude:      row.Longitude,
		WifiBSSID:      row.WifiBssid,
		WifiSSID:       row.WifiSsid,
		DistanceMeters: row.DistanceMeters,
	}
}

func attendanceRecordResponseFromListRow(row database.ListAttendanceRecordsByAliasRow) AttendanceRecordResponse {
	resp := attendanceRecordResponseFromDatabase(database.AttendanceRecord{
		ID:             row.ID,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		WorkspaceID:    row.WorkspaceID,
		MemberID:       row.MemberID,
		Type:           row.Type,
		RecordedAt:     row.RecordedAt,
		Latitude:       row.Latitude,
		Longitude:      row.Longitude,
		WifiBssid:      row.WifiBssid,
		WifiSsid:       row.WifiSsid,
		DistanceMeters: row.DistanceMeters,
	})
	resp.MemberAlias = row.MemberAlias
	resp.MemberName = row.MemberName
	resp.MemberAvatar = textString(row.MemberAvatar)
	return resp
}

func attendanceCorrectionRequestResponseFromDatabase(row database.AttendanceCorrectionRequest) AttendanceCorrectionRequestResponse {
	return AttendanceCorrectionRequestResponse{
		ID:                     pgutil.UUIDToString(row.ID),
		CreatedAt:              row.CreatedAt.Time,
		UpdatedAt:              row.UpdatedAt.Time,
		WorkspaceID:            pgutil.UUIDToString(row.WorkspaceID),
		MemberID:               pgutil.UUIDToString(row.MemberID),
		Reason:                 row.Reason,
		Status:                 AttendanceCorrectionStatus(row.Status),
		CheckInAt:              timestamptzPtr(row.CheckInAt),
		CheckOutAt:             timestamptzPtr(row.CheckOutAt),
		TargetCheckInRecordID:  pgutil.UUIDToString(row.TargetCheckInRecordID),
		TargetCheckOutRecordID: pgutil.UUIDToString(row.TargetCheckOutRecordID),
		OriginalCheckInAt:      timestamptzPtr(row.OriginalCheckInAt),
		OriginalCheckOutAt:     timestamptzPtr(row.OriginalCheckOutAt),
		ResultCheckInRecordID:  pgutil.UUIDToString(row.ResultCheckInRecordID),
		ResultCheckOutRecordID: pgutil.UUIDToString(row.ResultCheckOutRecordID),
		DecidedByID:            pgutil.UUIDToString(row.DecidedBy),
		DecidedAt:              timestamptzPtr(row.DecidedAt),
	}
}

func attendanceCorrectionRequestResponseFromListRow(row database.ListAttendanceCorrectionRequestsByAliasRow) AttendanceCorrectionRequestResponse {
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
		ResultCheckInRecordID:  row.ResultCheckInRecordID,
		ResultCheckOutRecordID: row.ResultCheckOutRecordID,
		DecidedBy:              row.DecidedBy,
		DecidedAt:              row.DecidedAt,
	})
	resp.MemberAlias = row.MemberAlias
	resp.MemberName = row.MemberName
	resp.MemberAvatar = textString(row.MemberAvatar)
	resp.DecidedByAlias = textString(row.DecidedByAlias)
	resp.DecidedByName = textString(row.DecidedByName)
	return resp
}

func workScheduleDayResponseFromDatabase(row database.WorkScheduleDay) WorkScheduleDayResponse {
	return WorkScheduleDayResponse{
		ID:             pgutil.UUIDToString(row.ID),
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
		WorkspaceID:    pgutil.UUIDToString(row.WorkspaceID),
		MemberID:       pgutil.UUIDToString(row.MemberID),
		Date:           row.ScheduleDate.Time.Format(time.DateOnly),
		Status:         WorkScheduleStatus(row.Status),
		Note:           row.Note,
		WeekNote:       row.WeekNote,
		Sessions:       []WorkScheduleSessionResponse{},
		RegisteredByID: pgutil.UUIDToString(row.RegisteredBy),
	}
}

func workScheduleDayResponseFromListRow(row database.ListWorkSchedulesByAliasRow) WorkScheduleDayResponse {
	resp := workScheduleDayResponseFromDatabase(database.WorkScheduleDay{
		ID:           row.ID,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		WorkspaceID:  row.WorkspaceID,
		MemberID:     row.MemberID,
		ScheduleDate: row.ScheduleDate,
		Status:       row.Status,
		Note:         row.Note,
		WeekNote:     row.WeekNote,
		RegisteredBy: row.RegisteredBy,
	})
	resp.MemberAlias = row.MemberAlias
	resp.MemberName = row.MemberName
	resp.MemberAvatar = textString(row.MemberAvatar)
	resp.RegisteredByAlias = textString(row.RegisteredByAlias)
	resp.RegisteredByName = textString(row.RegisteredByName)
	resp.RegisteredByAvatar = textString(row.RegisteredByAvatar)
	return resp
}

func workScheduleSessionResponseFromDatabase(row database.WorkScheduleSession) WorkScheduleSessionResponse {
	return WorkScheduleSessionResponse{
		ID:        pgutil.UUIDToString(row.ID),
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
		DayID:     pgutil.UUIDToString(row.DayID),
		StartAt:   row.StartAt.Time,
		EndAt:     row.EndAt.Time,
	}
}

func workspaceMemberBriefFromRow(row database.ListWorkspaceMembersForStatisticsRow) WorkspaceMemberBriefResponse {
	return WorkspaceMemberBriefResponse{
		MemberID:     pgutil.UUIDToString(row.MemberID),
		MemberAlias:  row.MemberAlias,
		MemberName:   row.MemberName,
		MemberAvatar: textString(row.MemberAvatar),
	}
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(time.DateOnly, value, vietnamLocation())
}

func parseMonth(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01", value, vietnamLocation())
}

func parseOptionalTime(value string, endOfDay bool) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, true, nil
	}
	parsed, err := parseDate(value)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("Thời gian phải theo định dạng RFC3339 hoặc YYYY-MM-DD")
	}
	if endOfDay {
		parsed = parsed.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	return parsed, true, nil
}

func parseClockMinutes(value string) (int, error) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}

func vietnamLocation() *time.Location {
	loc, err := time.LoadLocation(vietnamTimezone)
	if err != nil {
		return time.FixedZone(vietnamTimezone, 7*60*60)
	}
	return loc
}

func vietnamTime(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, vietnamLocation())
}

func normalizeSQLTime(value string) string {
	if len(value) >= 5 {
		return value[:5]
	}
	return value
}

func pgUUIDOrNull(value string) (pgtype.UUID, error) {
	if strings.TrimSpace(value) == "" {
		return pgtype.UUID{}, nil
	}
	return pgutil.ParseUUID(value)
}

func pgUUIDPtrOrNull(value *string) (pgtype.UUID, error) {
	if value == nil {
		return pgtype.UUID{}, nil
	}
	return pgUUIDOrNull(*value)
}

func pgTimeOrNull(value time.Time, valid bool) pgtype.Timestamptz {
	if !valid {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func pgTimePtr(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func pgDate(value time.Time) pgtype.Date {
	return pgtype.Date{Time: value, Valid: true}
}

func pgClock(value string) (pgtype.Time, error) {
	minutes, err := parseClockMinutes(value)
	if err != nil {
		return pgtype.Time{}, err
	}
	return pgtype.Time{Microseconds: int64(minutes) * int64(time.Minute/time.Microsecond), Valid: true}, nil
}

func normalizeWorkWeekRequest(req *UpsertWorkWeekRequest) error {
	seen := make(map[int]bool)
	for i := range req.Days {
		d := &req.Days[i]
		if d.DayOfWeek < 0 || d.DayOfWeek > 6 {
			return fmt.Errorf("dayOfWeek phải từ 0 (Chủ nhật) đến 6 (Thứ bảy)")
		}
		if seen[d.DayOfWeek] {
			return fmt.Errorf("dayOfWeek bị trùng: %d", d.DayOfWeek)
		}
		seen[d.DayOfWeek] = true
		if len(d.Sessions) == 0 {
			return fmt.Errorf("Ngày %d phải có ít nhất một ca", d.DayOfWeek)
		}
		if len(d.Sessions) > 3 {
			return fmt.Errorf("Ngày %d: tối đa 3 ca", d.DayOfWeek)
		}
		type ps struct{ start, end int }
		parsed := make([]ps, 0, len(d.Sessions))
		for j := range d.Sessions {
			s := &d.Sessions[j]
			s.StartTime = strings.TrimSpace(s.StartTime)
			s.EndTime = strings.TrimSpace(s.EndTime)
			start, err := parseClockMinutes(s.StartTime)
			if err != nil {
				return fmt.Errorf("Ngày %d, ca %d: startTime không hợp lệ", d.DayOfWeek, j)
			}
			end, err := parseClockMinutes(s.EndTime)
			if err != nil {
				return fmt.Errorf("Ngày %d, ca %d: endTime không hợp lệ", d.DayOfWeek, j)
			}
			if start >= end {
				return fmt.Errorf("Ngày %d, ca %d: startTime phải trước endTime", d.DayOfWeek, j)
			}
			parsed = append(parsed, ps{start, end})
		}
		sort.Slice(parsed, func(a, b int) bool { return parsed[a].start < parsed[b].start })
		for j := 1; j < len(parsed); j++ {
			if parsed[j].start < parsed[j-1].end {
				return fmt.Errorf("Ngày %d: các ca không được chồng lấn", d.DayOfWeek)
			}
		}
	}
	return nil
}
