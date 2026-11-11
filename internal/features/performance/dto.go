package performance

import (
	"fmt"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/notification"
	"github.com/alexcao194/kairso/internal/i18n"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
)

func textString(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}

type InitiativeStatus string

const (
	InitiativeStatusProposed   InitiativeStatus = "proposed"
	InitiativeStatusInProgress InitiativeStatus = "in_progress"
	InitiativeStatusAccepted   InitiativeStatus = "accepted"
	InitiativeStatusRejected   InitiativeStatus = "rejected"

	NotificationTypeInitiativeCreated = "initiative_created"
	NotificationTypeLateLogCreated    = "late_log_created"
	NotificationTypeFeedbackCreated   = "customer_feedback_created"
)

func (s InitiativeStatus) Valid() bool {
	switch s {
	case InitiativeStatusProposed, InitiativeStatusInProgress, InitiativeStatusAccepted, InitiativeStatusRejected:
		return true
	default:
		return false
	}
}

type CreateInitiativeRequest struct {
	Content              string  `json:"content"`
	Estimate             float64 `json:"estimate"`
	EarliestApprovalDate *string `json:"earliestApprovalDate,omitempty"`
	RefMemberID          string  `json:"refMemberId"`
}

type UpdateInitiativeRequest struct {
	Content              *string           `json:"content,omitempty"`
	Status               *InitiativeStatus `json:"status,omitempty"`
	ChangesAfter         *string           `json:"changesAfter,omitempty"`
	EarliestApprovalDate *string           `json:"earliestApprovalDate,omitempty"`
	Estimate             *float64          `json:"estimate,omitempty"`
	RefMemberID          *string           `json:"refMemberId,omitempty"`
}

type ListInitiativesFilter struct {
	Status      string `schema:"status"`
	ProposerID  string `schema:"proposerId"`
	RefMemberID string `schema:"refMemberId"`
	From        string `schema:"from"`
	To          string `schema:"to"`
}

type ListLateLogsFilter struct {
	LateUserID string `schema:"lateUserId"`
	From       string `schema:"from"`
	To         string `schema:"to"`
}

type ListFeedbacksFilter struct {
	MemberID string `schema:"memberId"`
	From     string `schema:"from"`
	To       string `schema:"to"`
}

type ProposerResponse struct {
	ID     string `json:"id"`
	Alias  string `json:"alias"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type InitiativeResponse struct {
	ID                   string           `json:"id"`
	CreatedAt            time.Time        `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt            time.Time        `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID          string           `json:"workspaceId"`
	Proposer             ProposerResponse `json:"proposer"`
	RefMember            ProposerResponse `json:"refMember"`
	Content              string           `json:"content"`
	Status               InitiativeStatus `json:"status"`
	ChangesAfter         *string          `json:"changesAfter,omitempty"`
	EarliestApprovalDate *string          `json:"earliestApprovalDate,omitempty"`
	ApprovedAt           *time.Time       `json:"approvedAt,omitempty" swaggertype:"string" format:"date-time"`
	Estimate             float64          `json:"estimate"`
}

type CreateLateLogRequest struct {
	LateUserID string    `json:"lateUserId"`
	Content    string    `json:"content"`
	LateTime   time.Time `json:"lateTime" swaggertype:"string" format:"date-time"`
	Estimate   float64   `json:"estimate"`
}

type UpdateLateLogRequest struct {
	Content  *string    `json:"content,omitempty"`
	LateTime *time.Time `json:"lateTime,omitempty" swaggertype:"string" format:"date-time"`
	Estimate *float64   `json:"estimate,omitempty"`
}

type LateUserResponse struct {
	ID     string `json:"id"`
	Alias  string `json:"alias"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type LateLogResponse struct {
	ID          string           `json:"id"`
	CreatedAt   time.Time        `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time        `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID string           `json:"workspaceId"`
	Creator     ProposerResponse `json:"creator"`
	LateUser    LateUserResponse `json:"lateUser"`
	Content     string           `json:"content"`
	LateTime    time.Time        `json:"lateTime" swaggertype:"string" format:"date-time"`
	Estimate    float64          `json:"estimate"`
}

type BeneficiaryRequest struct {
	UserID string  `json:"userId"`
	Weight float64 `json:"weight"`
}

type CreateFeedbackRequest struct {
	FeedbackText  string               `json:"feedbackText"`
	Content       string               `json:"content"`
	Estimate      float64              `json:"estimate"`
	Beneficiaries []BeneficiaryRequest `json:"beneficiaries"`
}

type UpdateFeedbackRequest struct {
	FeedbackText  *string               `json:"feedbackText,omitempty"`
	Content       *string               `json:"content,omitempty"`
	Estimate      *float64              `json:"estimate,omitempty"`
	Beneficiaries *[]BeneficiaryRequest `json:"beneficiaries,omitempty"`
}

type BeneficiaryResponse struct {
	ID     string  `json:"id"`
	Alias  string  `json:"alias"`
	Name   string  `json:"name"`
	Avatar string  `json:"avatar,omitempty"`
	Weight float64 `json:"weight"`
}

type FeedbackResponse struct {
	ID            string                `json:"id"`
	CreatedAt     time.Time             `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt     time.Time             `json:"updatedAt" swaggertype:"string" format:"date-time"`
	WorkspaceID   string                `json:"workspaceId"`
	Creator       ProposerResponse      `json:"creator"`
	FeedbackText  string                `json:"feedbackText"`
	Content       string                `json:"content"`
	Estimate      float64               `json:"estimate"`
	Beneficiaries []BeneficiaryResponse `json:"beneficiaries"`
}

type PerformanceNotificationEvent struct {
	Type           string
	RecipientID    string
	WorkspaceID    string
	WorkspaceAlias string
	EntityID       string
}

func (e PerformanceNotificationEvent) ToNotificationEvent() notification.NotificationEvent {
	data := map[string]string{
		"type":           e.Type,
		"targetID":       e.EntityID,
		"workspaceId":    e.WorkspaceID,
		"workspaceAlias": e.WorkspaceAlias,
	}
	title, body := i18n.RenderNotification(i18n.Default, e.Type, nil)
	if title == "" {
		title, body = i18n.RenderNotification(i18n.Default, "workspace_performance_updated", nil)
	}
	return notification.NotificationEvent{
		UserID:      e.RecipientID,
		WorkspaceID: e.WorkspaceID,
		Title:       title,
		Body:        body,
		Data:        data,
	}
}

func initiativeResponseFromRow(row database.ListInitiativesRow) InitiativeResponse {
	resp := InitiativeResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
		Proposer: ProposerResponse{
			ID:     pgutil.UUIDToString(row.ProposerID),
			Alias:  row.ProposerAlias,
			Name:   row.ProposerName,
			Avatar: textString(row.ProposerAvatar),
		},
		RefMember: ProposerResponse{
			ID:     pgutil.UUIDToString(row.RefMemberID),
			Alias:  row.RefMemberAlias,
			Name:   row.RefMemberName,
			Avatar: textString(row.RefMemberAvatar),
		},
		Content:  row.Content,
		Status:   InitiativeStatus(row.Status),
		Estimate: row.Estimate,
	}
	if row.ChangesAfter.Valid {
		resp.ChangesAfter = &row.ChangesAfter.String
	}
	if row.EarliestApprovalDate.Valid {
		s := row.EarliestApprovalDate.Time.Format("2006-01-02")
		resp.EarliestApprovalDate = &s
	}
	if row.ApprovedAt.Valid {
		t := row.ApprovedAt.Time
		resp.ApprovedAt = &t
	}
	return resp
}

func lateLogResponseFromRow(row database.ListLateLogsRow) LateLogResponse {
	return LateLogResponse{
		ID:          pgutil.UUIDToString(row.ID),
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
		WorkspaceID: pgutil.UUIDToString(row.WorkspaceID),
		Creator: ProposerResponse{
			ID:     pgutil.UUIDToString(row.CreatorID),
			Alias:  row.CreatorAlias,
			Name:   row.CreatorName,
			Avatar: textString(row.CreatorAvatar),
		},
		LateUser: LateUserResponse{
			ID:     pgutil.UUIDToString(row.LateUserID),
			Alias:  row.LateUserAlias,
			Name:   row.LateUserName,
			Avatar: textString(row.LateUserAvatar),
		},
		Content:  row.Content,
		LateTime: row.LateTime.Time,
		Estimate: row.Estimate,
	}
}

func feedbackResponseFromFields(
	id pgtype.UUID,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
	workspaceID pgtype.UUID,
	creatorID pgtype.UUID,
	creatorAlias string,
	creatorName string,
	creatorAvatar pgtype.Text,
	feedbackText string,
	content string,
	estimate float64,
	beneficiaries []database.ListFeedbackBeneficiariesRow,
) FeedbackResponse {
	bens := make([]BeneficiaryResponse, 0, len(beneficiaries))
	for _, b := range beneficiaries {
		bens = append(bens, BeneficiaryResponse{
			ID:     pgutil.UUIDToString(b.UserID),
			Alias:  b.UserAlias,
			Name:   b.UserName,
			Avatar: textString(b.UserAvatar),
			Weight: b.Weight,
		})
	}
	return FeedbackResponse{
		ID:          pgutil.UUIDToString(id),
		CreatedAt:   createdAt.Time,
		UpdatedAt:   updatedAt.Time,
		WorkspaceID: pgutil.UUIDToString(workspaceID),
		Creator: ProposerResponse{
			ID:     pgutil.UUIDToString(creatorID),
			Alias:  creatorAlias,
			Name:   creatorName,
			Avatar: textString(creatorAvatar),
		},
		FeedbackText:  feedbackText,
		Content:       content,
		Estimate:      estimate,
		Beneficiaries: bens,
	}
}

func pgtypeDate(s *string) pgtype.Date {
	if s == nil {
		return pgtype.Date{}
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: t, Valid: true}
}

type KPIFilter struct {
	Month    string `schema:"month"`
	MemberID string `schema:"memberId"`
}

type SetAttitudeRequest struct {
	MemberID string  `json:"memberId"`
	Month    string  `json:"month"`
	Score    float64 `json:"score"`
	Comment  string  `json:"comment,omitempty"`
}

const (
	DefaultCustomerWeight    = 10.0
	DefaultProgressWeight    = 35.0
	DefaultQualityWeight     = 30.0
	DefaultInitiativeWeight  = 5.0
	DefaultPunctualityWeight = 15.0
	DefaultAttitudeWeight    = 5.0
)

type UpsertKPISettingsRequest struct {
	CustomerWeight    float64 `json:"customerWeight"`
	ProgressWeight    float64 `json:"progressWeight"`
	QualityWeight     float64 `json:"qualityWeight"`
	InitiativeWeight  float64 `json:"initiativeWeight"`
	PunctualityWeight float64 `json:"punctualityWeight"`
	AttitudeWeight    float64 `json:"attitudeWeight"`
}

func validateKPISettingsRequest(req UpsertKPISettingsRequest) error {
	weights := []float64{
		req.CustomerWeight,
		req.ProgressWeight,
		req.QualityWeight,
		req.InitiativeWeight,
		req.PunctualityWeight,
		req.AttitudeWeight,
	}
	sum := 0.0
	for _, w := range weights {
		if w < 0 {
			return fmt.Errorf("Trọng số KPI không được âm")
		}
		sum += w
	}
	const epsilon = 0.01
	if sum < 100-epsilon || sum > 100+epsilon {
		return fmt.Errorf("Tổng trọng số KPI phải bằng 100 (hiện là %.2f)", sum)
	}
	return nil
}

type CustomerKPIDetail struct {
	RawEstimate float64 `json:"rawEstimate"`
	Score       float64 `json:"score"`
	KPI         float64 `json:"kpi"`
}

type ProgressKPIDetail struct {
	TotalWorkerEstimationReviewedDone float64 `json:"totalWorkerEstimationReviewedDone"`
	LateWorkerEstimation              float64 `json:"lateWorkerEstimation"`
	LateRatio                         float64 `json:"lateRatio"`
	KPI                               float64 `json:"kpi"`
}

type QualityKPIDetail struct {
	TotalWorkerEstimation float64 `json:"totalWorkerEstimation"`
	TotalBugEstimation    float64 `json:"totalBugEstimation"`
	TotalReturnEstimation float64 `json:"totalReturnEstimation"`
	Ratio                 float64 `json:"ratio"`
	KPI                   float64 `json:"kpi"`
}

type InitiativeKPIDetail struct {
	RawEstimate float64 `json:"rawEstimate"`
	Score       float64 `json:"score"`
	KPI         float64 `json:"kpi"`
}

type PunctualityKPIDetail struct {
	RawEstimate    float64 `json:"rawEstimate"`
	LateDays       int     `json:"lateDays"`
	EarlyLeaveDays int     `json:"earlyLeaveDays"`
	Deduction      float64 `json:"deduction"`
	KPI            float64 `json:"kpi"`
}

type AttitudeKPIDetail struct {
	Score   float64 `json:"score"`
	Comment string  `json:"comment,omitempty"`
	KPI     float64 `json:"kpi"`
}

type KPIMemberResponse struct {
	MemberID     string `json:"memberId"`
	MemberAlias  string `json:"memberAlias"`
	MemberName   string `json:"memberName"`
	MemberAvatar string `json:"memberAvatar,omitempty"`
	Period       string `json:"period"`
	Label        string `json:"label"`

	Customer    CustomerKPIDetail    `json:"customer"`
	Progress    ProgressKPIDetail    `json:"progress"`
	Quality     QualityKPIDetail     `json:"quality"`
	Initiative  InitiativeKPIDetail  `json:"initiative"`
	Punctuality PunctualityKPIDetail `json:"punctuality"`
	Attitude    AttitudeKPIDetail    `json:"attitude"`

	TotalKPI float64 `json:"totalKpi"`
}

type KPIExportFile struct {
	Filename string
	Data     []byte
}

type AttitudeResponse struct {
	MemberID     string    `json:"memberId"`
	MemberAlias  string    `json:"memberAlias"`
	MemberName   string    `json:"memberName"`
	MemberAvatar string    `json:"memberAvatar,omitempty"`
	Month        string    `json:"month"`
	Score        float64   `json:"score"`
	Comment      string    `json:"comment,omitempty"`
	UpdatedByID  string    `json:"updatedById,omitempty"`
	UpdatedAt    time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
}

type KPISettingsResponse struct {
	WorkspaceID       string    `json:"workspaceId"`
	CustomerWeight    float64   `json:"customerWeight"`
	ProgressWeight    float64   `json:"progressWeight"`
	QualityWeight     float64   `json:"qualityWeight"`
	InitiativeWeight  float64   `json:"initiativeWeight"`
	PunctualityWeight float64   `json:"punctualityWeight"`
	AttitudeWeight    float64   `json:"attitudeWeight"`
	UpdatedAt         time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
}

func defaultKPISettings(workspaceID string) KPISettingsResponse {
	return KPISettingsResponse{
		WorkspaceID:       workspaceID,
		CustomerWeight:    DefaultCustomerWeight,
		ProgressWeight:    DefaultProgressWeight,
		QualityWeight:     DefaultQualityWeight,
		InitiativeWeight:  DefaultInitiativeWeight,
		PunctualityWeight: DefaultPunctualityWeight,
		AttitudeWeight:    DefaultAttitudeWeight,
	}
}

func kpiSettingsResponseFromDatabase(row database.WorkspaceKpiSetting) KPISettingsResponse {
	return KPISettingsResponse{
		WorkspaceID:       pgutil.UUIDToString(row.WorkspaceID),
		CustomerWeight:    row.CustomerWeight,
		ProgressWeight:    row.ProgressWeight,
		QualityWeight:     row.QualityWeight,
		InitiativeWeight:  row.InitiativeWeight,
		PunctualityWeight: row.PunctualityWeight,
		AttitudeWeight:    row.AttitudeWeight,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}

type kpiTaskStats struct {
	memberID                          string
	memberAlias                       string
	memberName                        string
	totalWorkerEstimationReviewedDone float64
	lateWorkerEstimation              float64
	totalWorkerEstimation             float64
	totalBugEstimation                float64
	totalReturnEstimation             float64
}

type kpiAttitude struct {
	memberID string
	score    float64
	comment  string
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func buildKPIMember(
	memberID, alias, name, avatar, period, label string,
	taskStats *kpiTaskStats,
	feedbackEstimate float64,
	initiativeEstimate float64,
	lateLogEstimate float64,
	attitude kpiAttitude,
	lateDays int,
	earlyLeaveDays int,
	settings KPISettingsResponse,
) KPIMemberResponse {
	customerScore := clamp(feedbackEstimate+settings.CustomerWeight/2, 0, settings.CustomerWeight)
	customerKPI := customerScore

	var totalWorkerEstimationReviewedDone float64
	var lateWorkerEstimation float64
	var lateRatio float64
	if taskStats != nil {
		totalWorkerEstimationReviewedDone = taskStats.totalWorkerEstimationReviewedDone
		lateWorkerEstimation = taskStats.lateWorkerEstimation
		if totalWorkerEstimationReviewedDone > 0 {
			lateRatio = lateWorkerEstimation / totalWorkerEstimationReviewedDone
		}
	}
	progressKPI := (1 - lateRatio) * settings.ProgressWeight

	var totalWorkerEstimation float64
	var totalBugEstimation float64
	var totalReturnEstimation float64
	qualityRatio := 1.0
	if taskStats != nil {
		totalWorkerEstimation = taskStats.totalWorkerEstimation
		totalBugEstimation = taskStats.totalBugEstimation
		totalReturnEstimation = taskStats.totalReturnEstimation
		if totalWorkerEstimation > 0 {
			clean := totalWorkerEstimation - totalBugEstimation - totalReturnEstimation
			qualityRatio = clean / totalWorkerEstimation
			if qualityRatio < 0 {
				qualityRatio = 0
			}
		}
	}
	qualityKPI := qualityRatio * settings.QualityWeight

	initiativeScore := clamp(initiativeEstimate, 0, settings.InitiativeWeight)
	initiativeKPI := initiativeScore

	rawPunctualityDeduction := float64(lateDays+earlyLeaveDays)*1.0 + lateLogEstimate
	punctualityDeduction := clamp(rawPunctualityDeduction, 0, settings.PunctualityWeight)
	punctualityKPI := settings.PunctualityWeight - punctualityDeduction

	attitudeKPI := attitude.score
	if attitudeKPI > settings.AttitudeWeight {
		attitudeKPI = settings.AttitudeWeight
	}

	total := customerKPI + progressKPI + qualityKPI + initiativeKPI + punctualityKPI + attitudeKPI

	resp := KPIMemberResponse{
		MemberID: memberID, MemberAlias: alias, MemberName: name, MemberAvatar: avatar,
		Period: period, Label: label,
		Customer: CustomerKPIDetail{
			RawEstimate: feedbackEstimate,
			Score:       customerScore,
			KPI:         customerKPI,
		},
		Progress: ProgressKPIDetail{
			TotalWorkerEstimationReviewedDone: totalWorkerEstimationReviewedDone,
			LateWorkerEstimation:              lateWorkerEstimation,
			LateRatio:                         lateRatio,
			KPI:                               progressKPI,
		},
		Quality: QualityKPIDetail{
			TotalWorkerEstimation: totalWorkerEstimation,
			TotalBugEstimation:    totalBugEstimation,
			TotalReturnEstimation: totalReturnEstimation,
			Ratio:                 qualityRatio,
			KPI:                   qualityKPI,
		},
		Initiative: InitiativeKPIDetail{
			RawEstimate: initiativeEstimate,
			Score:       initiativeScore,
			KPI:         initiativeKPI,
		},
		Punctuality: PunctualityKPIDetail{
			RawEstimate:    lateLogEstimate,
			LateDays:       lateDays,
			EarlyLeaveDays: earlyLeaveDays,
			Deduction:      punctualityDeduction,
			KPI:            punctualityKPI,
		},
		Attitude: AttitudeKPIDetail{
			Score:   attitude.score,
			Comment: attitude.comment,
			KPI:     attitudeKPI,
		},
		TotalKPI: total,
	}
	return resp
}

type MemberTaskBrief struct {
	ID               string     `json:"id"`
	Code             string     `json:"code"`
	Title            string     `json:"title"`
	ProjectID        string     `json:"projectId"`
	ProjectName      string     `json:"projectName"`
	Status           string     `json:"status"`
	Priority         int        `json:"priority"`
	StartDate        *time.Time `json:"startDate,omitempty" swaggertype:"string" format:"date-time"`
	EndDate          *time.Time `json:"endDate,omitempty" swaggertype:"string" format:"date-time"`
	FirstCompletedAt *time.Time `json:"firstCompletedAt,omitempty" swaggertype:"string" format:"date-time"`
	ReturnCount      int        `json:"returnCount"`
	Estimation       int        `json:"estimation"`
}

type MemberTaskBreakdownResponse struct {
	MemberID           string            `json:"memberId"`
	Month              string            `json:"month"`
	Label              string            `json:"label"`
	BugTasks           []MemberTaskBrief `json:"bugTasks"`
	CompletedLateTasks []MemberTaskBrief `json:"completedLateTasks"`
	OverdueTasks       []MemberTaskBrief `json:"overdueTasks"`
	ReworkTasks        []MemberTaskBrief `json:"reworkTasks"`
}

type memberTaskRow struct {
	brief         MemberTaskBrief
	isWorker      bool
	isFaultHolder bool
}

func timestamptzPtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}
