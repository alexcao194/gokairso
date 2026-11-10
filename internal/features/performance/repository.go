package performance

import (
	"context"
	"errors"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/attendance"
	"github.com/alexcao194/kairso/internal/features/task"
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
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return workspace.WorkspaceRole(string(role)), nil
}

func (r *repository) ListWorkspaceAdminHolderIDs(ctx context.Context, workspaceAlias string) ([]string, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uuids, err := r.queries.ListWorkspaceAdminHolderIDs(ctx, wid)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(uuids))
	for _, u := range uuids {
		ids = append(ids, pgutil.UUIDToString(u))
	}
	return ids, nil
}

func (r *repository) GetWorkspaceID(ctx context.Context, workspaceAlias string) (string, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(wid), nil
}

func (r *repository) GetWorkspaceAliasByID(ctx context.Context, workspaceID string) (string, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(wid), nil
}

func (r *repository) GetWorkspaceHolderName(ctx context.Context, workspaceAlias string) (string, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return "", err
	}
	name, err := r.queries.GetWorkspaceHolderName(ctx, wid)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return name, nil
}

func (r *repository) ListWorkspaceMembersForStatistics(ctx context.Context, workspaceAlias, memberID string) ([]database.ListWorkspaceMembersForStatisticsRow, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgUUIDOrNullStr(memberID)
	if err != nil {
		return nil, err
	}
	return r.queries.ListWorkspaceMembersForStatistics(ctx, database.ListWorkspaceMembersForStatisticsParams{
		WorkspaceID: wid,
		MemberID:    uid,
	})
}

func (r *repository) CreateInitiative(ctx context.Context, workspaceAlias, proposerID, refMemberID string, req CreateInitiativeRequest) (*InitiativeResponse, error) {
	pUID, err := pgutil.ParseUUID(proposerID)
	if err != nil {
		return nil, err
	}
	rUID, err := pgutil.ParseUUID(refMemberID)
	if err != nil {
		return nil, err
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateInitiative(ctx, database.CreateInitiativeParams{
		WorkspaceID: wid,
		ProposerID:  pUID,
		RefMemberID: rUID,
		Content:     req.Content,
		Estimate:    req.Estimate,
	})
	if err != nil {
		return nil, err
	}
	full, err := r.queries.GetInitiative(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	resp := initiativeResponseFromRow(database.ListInitiativesRow(full))
	if req.EarliestApprovalDate != nil {
		updateRow, err := r.queries.UpdateInitiative(ctx, database.UpdateInitiativeParams{
			ID:                   row.ID,
			EarliestDatePresent:  true,
			EarliestApprovalDate: pgtypeDate(req.EarliestApprovalDate),
		})
		if err != nil {
			return nil, err
		}
		_ = updateRow
		full2, err := r.queries.GetInitiative(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		full2Converted := database.ListInitiativesRow(full2)
		return &[]InitiativeResponse{initiativeResponseFromRow(full2Converted)}[0], nil
	}
	return &resp, nil
}

func (r *repository) ListInitiatives(ctx context.Context, workspaceAlias string, params pagination.Params, filter ListInitiativesFilter) (pagination.Response[InitiativeResponse], error) {
	var empty pagination.Response[InitiativeResponse]
	proposerUID, _ := pgUUIDOrNullStr(filter.ProposerID)
	refMemberUID, _ := pgUUIDOrNullStr(filter.RefMemberID)
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListInitiatives(ctx, database.ListInitiativesParams{
		WorkspaceID: wid,
		Status:      filter.Status,
		FromTime:    parseOptionalTimestamptz(filter.From),
		ToTime:      parseOptionalTimestamptz(filter.To),
		ProposerID:  proposerUID,
		RefMemberID: refMemberUID,
		Limit:       int32(params.LimitWithMore()),
		Offset:      int32(params.Offset()),
	})
	if err != nil {
		return empty, err
	}
	items := make([]InitiativeResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, initiativeResponseFromRow(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) GetInitiative(ctx context.Context, initiativeID string) (*InitiativeResponse, error) {
	uid, err := pgutil.ParseUUID(initiativeID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetInitiative(ctx, uid)
	if err != nil {
		return nil, err
	}
	resp := initiativeResponseFromRow(database.ListInitiativesRow(row))
	return &resp, nil
}

func (r *repository) UpdateInitiative(ctx context.Context, initiativeID string, refMemberID *string, req UpdateInitiativeRequest, setApprovedAt bool) (*InitiativeResponse, error) {
	uid, err := pgutil.ParseUUID(initiativeID)
	if err != nil {
		return nil, err
	}
	params := database.UpdateInitiativeParams{ID: uid}
	if req.Content != nil {
		params.ContentPresent = true
		params.Content = *req.Content
	}
	if req.Status != nil {
		params.StatusPresent = true
		params.Status = string(*req.Status)
	}
	if req.ChangesAfter != nil {
		params.ChangesAfterPresent = true
		params.ChangesAfter = *req.ChangesAfter
	}
	if req.EarliestApprovalDate != nil {
		params.EarliestDatePresent = true
		params.EarliestApprovalDate = pgtypeDate(req.EarliestApprovalDate)
	}
	if setApprovedAt {
		params.ApprovedAtPresent = true
		params.ApprovedAt = pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	}
	if req.Estimate != nil {
		params.EstimatePresent = true
		params.Estimate = *req.Estimate
	}
	if refMemberID != nil {
		params.RefMemberPresent = true
		refUID, err := pgutil.ParseUUID(*refMemberID)
		if err != nil {
			return nil, err
		}
		params.RefMemberID = refUID
	}
	if _, err := r.queries.UpdateInitiative(ctx, params); err != nil {
		return nil, err
	}
	return r.GetInitiative(ctx, initiativeID)
}

func (r *repository) DeleteInitiative(ctx context.Context, initiativeID string) error {
	uid, err := pgutil.ParseUUID(initiativeID)
	if err != nil {
		return err
	}
	return r.queries.DeleteInitiative(ctx, uid)
}

func (r *repository) CreateLateLog(ctx context.Context, workspaceAlias string, creatorID string, req CreateLateLogRequest, lateUserID string) (*LateLogResponse, error) {
	cUID, err := pgutil.ParseUUID(creatorID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(lateUserID)
	if err != nil {
		return nil, err
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateLateLog(ctx, database.CreateLateLogParams{
		WorkspaceID: wid,
		LateUserID:  uid,
		CreatorID:   cUID,
		Content:     req.Content,
		LateTime:    pgtype.Timestamptz{Time: req.LateTime.UTC(), Valid: true},
		Estimate:    req.Estimate,
	})
	if err != nil {
		return nil, err
	}
	full, err := r.queries.GetLateLog(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	resp := lateLogResponseFromRow(database.ListLateLogsRow(full))
	return &resp, nil
}

func (r *repository) ListLateLogs(ctx context.Context, workspaceAlias string, params pagination.Params, filter ListLateLogsFilter) (pagination.Response[LateLogResponse], error) {
	var empty pagination.Response[LateLogResponse]
	lateUserUID, _ := pgUUIDOrNullStr(filter.LateUserID)
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListLateLogs(ctx, database.ListLateLogsParams{
		WorkspaceID: wid,
		FromTime:    parseOptionalTimestamptz(filter.From),
		ToTime:      parseOptionalTimestamptz(filter.To),
		LateUserID:  lateUserUID,
		Limit:       int32(params.LimitWithMore()),
		Offset:      int32(params.Offset()),
	})
	if err != nil {
		return empty, err
	}
	items := make([]LateLogResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, lateLogResponseFromRow(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) GetLateLog(ctx context.Context, lateLogID string) (*LateLogResponse, error) {
	uid, err := pgutil.ParseUUID(lateLogID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetLateLog(ctx, uid)
	if err != nil {
		return nil, err
	}
	resp := lateLogResponseFromRow(database.ListLateLogsRow(row))
	return &resp, nil
}

func (r *repository) UpdateLateLog(ctx context.Context, lateLogID string, req UpdateLateLogRequest) (*LateLogResponse, error) {
	uid, err := pgutil.ParseUUID(lateLogID)
	if err != nil {
		return nil, err
	}
	params := database.UpdateLateLogParams{ID: uid}
	if req.Content != nil {
		params.ContentPresent = true
		params.Content = *req.Content
	}
	if req.LateTime != nil {
		params.LateTimePresent = true
		params.LateTime = pgtype.Timestamptz{Time: req.LateTime.UTC(), Valid: true}
	}
	if req.Estimate != nil {
		params.EstimatePresent = true
		params.Estimate = *req.Estimate
	}
	if _, err := r.queries.UpdateLateLog(ctx, params); err != nil {
		return nil, err
	}
	return r.GetLateLog(ctx, lateLogID)
}

func (r *repository) DeleteLateLog(ctx context.Context, lateLogID string) error {
	uid, err := pgutil.ParseUUID(lateLogID)
	if err != nil {
		return err
	}
	return r.queries.DeleteLateLog(ctx, uid)
}

func (r *repository) CreateFeedback(ctx context.Context, workspaceAlias string, creatorID string, req CreateFeedbackRequest) (*FeedbackResponse, error) {
	cUID, err := pgutil.ParseUUID(creatorID)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	txQ := r.queries.WithTx(tx)
	defer tx.Rollback(ctx)

	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	cf, err := txQ.CreateCustomerFeedback(ctx, database.CreateCustomerFeedbackParams{
		WorkspaceID:  wid,
		CreatorID:    cUID,
		FeedbackText: req.FeedbackText,
		Content:      req.Content,
		Estimate:     req.Estimate,
	})
	if err != nil {
		return nil, err
	}
	for _, b := range req.Beneficiaries {
		userID, err := pgutil.ParseUUID(b.UserID)
		if err != nil {
			return nil, err
		}
		if _, err := txQ.UpsertFeedbackBeneficiary(ctx, database.UpsertFeedbackBeneficiaryParams{
			FeedbackID: cf.ID,
			UserID:     userID,
			Weight:     b.Weight,
		}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetFeedback(ctx, pgutil.UUIDToString(cf.ID))
}

func (r *repository) ListFeedbacks(ctx context.Context, workspaceAlias string, params pagination.Params, filter ListFeedbacksFilter) (pagination.Response[FeedbackResponse], error) {
	var empty pagination.Response[FeedbackResponse]
	memberUID, _ := pgUUIDOrNullStr(filter.MemberID)
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListCustomerFeedbacks(ctx, database.ListCustomerFeedbacksParams{
		WorkspaceID: wid,
		FromTime:    parseOptionalTimestamptz(filter.From),
		ToTime:      parseOptionalTimestamptz(filter.To),
		MemberID:    memberUID,
		Limit:       int32(params.LimitWithMore()),
		Offset:      int32(params.Offset()),
	})
	if err != nil {
		return empty, err
	}
	items := make([]FeedbackResponse, 0, len(rows))
	for _, row := range rows {
		bens, err := r.queries.ListFeedbackBeneficiaries(ctx, row.ID)
		if err != nil {
			return empty, err
		}
		items = append(items, feedbackResponseFromFields(
			row.ID,
			row.CreatedAt,
			row.UpdatedAt,
			row.WorkspaceID,
			row.CreatorID,
			row.CreatorAlias,
			row.CreatorName,
			row.CreatorAvatar,
			row.FeedbackText,
			row.Content,
			row.Estimate,
			bens,
		))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) GetFeedback(ctx context.Context, feedbackID string) (*FeedbackResponse, error) {
	uid, err := pgutil.ParseUUID(feedbackID)
	if err != nil {
		return nil, err
	}
	cf, err := r.queries.GetCustomerFeedback(ctx, uid)
	if err != nil {
		return nil, err
	}
	bens, err := r.queries.ListFeedbackBeneficiaries(ctx, cf.ID)
	if err != nil {
		return nil, err
	}
	resp := feedbackResponseFromFields(
		cf.ID,
		cf.CreatedAt,
		cf.UpdatedAt,
		cf.WorkspaceID,
		cf.CreatorID,
		cf.CreatorAlias,
		cf.CreatorName,
		cf.CreatorAvatar,
		cf.FeedbackText,
		cf.Content,
		cf.Estimate,
		bens,
	)
	return &resp, nil
}

func (r *repository) UpdateFeedback(ctx context.Context, feedbackID string, req UpdateFeedbackRequest) (*FeedbackResponse, error) {
	uid, err := pgutil.ParseUUID(feedbackID)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	txQ := r.queries.WithTx(tx)
	defer tx.Rollback(ctx)

	params := database.UpdateCustomerFeedbackParams{ID: uid}
	if req.FeedbackText != nil {
		params.FeedbackTextPresent = true
		params.FeedbackText = *req.FeedbackText
	}
	if req.Content != nil {
		params.ContentPresent = true
		params.Content = *req.Content
	}
	if req.Estimate != nil {
		params.EstimatePresent = true
		params.Estimate = *req.Estimate
	}
	if _, err := txQ.UpdateCustomerFeedback(ctx, params); err != nil {
		return nil, err
	}
	if req.Beneficiaries != nil {
		if err := txQ.DeleteFeedbackBeneficiaries(ctx, uid); err != nil {
			return nil, err
		}
		for _, b := range *req.Beneficiaries {
			userID, err := pgutil.ParseUUID(b.UserID)
			if err != nil {
				return nil, err
			}
			if _, err := txQ.UpsertFeedbackBeneficiary(ctx, database.UpsertFeedbackBeneficiaryParams{
				FeedbackID: uid,
				UserID:     userID,
				Weight:     b.Weight,
			}); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetFeedback(ctx, feedbackID)
}

func (r *repository) DeleteFeedback(ctx context.Context, feedbackID string) error {
	uid, err := pgutil.ParseUUID(feedbackID)
	if err != nil {
		return err
	}
	return r.queries.DeleteCustomerFeedback(ctx, uid)
}

func (r *repository) SumInitiativeEstimates(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]float64, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgUUIDOrNullStr(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.SumInitiativeEstimateByRefMember(ctx, database.SumInitiativeEstimateByRefMemberParams{
		WorkspaceID: wid,
		From:        pgtype.Timestamptz{Time: from, Valid: true},
		To:          pgtype.Timestamptz{Time: to, Valid: true},
		MemberID:    uid,
	})
	if err != nil {
		return nil, err
	}
	m := make(map[string]float64, len(rows))
	for _, r := range rows {
		m[pgutil.UUIDToString(r.UserID)] = r.Total
	}
	return m, nil
}

func (r *repository) SumLateLogEstimates(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]float64, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgUUIDOrNullStr(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.SumLateLogEstimateByUser(ctx, database.SumLateLogEstimateByUserParams{
		WorkspaceID: wid,
		From:        pgtype.Timestamptz{Time: from, Valid: true},
		To:          pgtype.Timestamptz{Time: to, Valid: true},
		MemberID:    uid,
	})
	if err != nil {
		return nil, err
	}
	m := make(map[string]float64, len(rows))
	for _, r := range rows {
		m[pgutil.UUIDToString(r.UserID)] = r.Total
	}
	return m, nil
}

func (r *repository) SumFeedbackEstimates(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]float64, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	uid, err := pgUUIDOrNullStr(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.SumFeedbackEstimateByBeneficiary(ctx, database.SumFeedbackEstimateByBeneficiaryParams{
		WorkspaceID: wid,
		From:        pgtype.Timestamptz{Time: from, Valid: true},
		To:          pgtype.Timestamptz{Time: to, Valid: true},
		MemberID:    uid,
	})
	if err != nil {
		return nil, err
	}
	m := make(map[string]float64, len(rows))
	for _, r := range rows {
		m[pgutil.UUIDToString(r.UserID)] = r.Total
	}
	return m, nil
}

func (r *repository) GetKPITaskStats(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) ([]kpiTaskStats, error) {
	uid, err := pgUUIDOrNullStr(memberID)
	if err != nil {
		return nil, err
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.GetKPITaskStats(ctx, database.GetKPITaskStatsParams{
		WorkspaceID: wid,
		MemberID:    uid,
		From:        pgtype.Timestamptz{Time: from, Valid: true},
		To:          pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	items := make([]kpiTaskStats, 0, len(rows))
	for _, row := range rows {
		items = append(items, kpiTaskStats{
			memberID:                          pgutil.UUIDToString(row.MemberID),
			memberAlias:                       row.MemberAlias,
			memberName:                        row.MemberName,
			totalWorkerEstimationReviewedDone: row.TotalWorkerEstimationReviewedDone,
			lateWorkerEstimation:              row.LateWorkerEstimation,
			totalWorkerEstimation:             row.TotalWorkerEstimation,
			totalBugEstimation:                row.TotalBugEstimation,
			totalReturnEstimation:             row.TotalReturnEstimation,
		})
	}
	return items, nil
}

func (r *repository) GetKPIAttitudes(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) (map[string]kpiAttitude, error) {
	uid, err := pgUUIDOrNullStr(memberID)
	if err != nil {
		return nil, err
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.GetEmployeeMonthlyAttitudes(ctx, database.GetEmployeeMonthlyAttitudesParams{
		WorkspaceID: wid,
		MemberID:    uid,
		From:        pgDateOf(from),
		To:          pgDateOf(to),
	})
	if err != nil {
		return nil, err
	}
	m := make(map[string]kpiAttitude, len(rows))
	for _, row := range rows {
		id := pgutil.UUIDToString(row.MemberID)
		m[id] = kpiAttitude{
			memberID: id,
			score:    row.Score,
			comment:  textString(row.Comment),
		}
	}
	return m, nil
}

func (r *repository) GetKPISettings(ctx context.Context, workspaceAlias string) (*KPISettingsResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetKPISettingsByAlias(ctx, wid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			resp := defaultKPISettings(workspaceAlias)
			return &resp, nil
		}
		return nil, err
	}
	resp := kpiSettingsResponseFromDatabase(row)
	return &resp, nil
}

func (r *repository) UpsertKPISettings(ctx context.Context, workspaceAlias string, req UpsertKPISettingsRequest) (*KPISettingsResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpsertKPISettingsByAlias(ctx, database.UpsertKPISettingsByAliasParams{
		WorkspaceID:       wid,
		CustomerWeight:    req.CustomerWeight,
		ProgressWeight:    req.ProgressWeight,
		QualityWeight:     req.QualityWeight,
		InitiativeWeight:  req.InitiativeWeight,
		PunctualityWeight: req.PunctualityWeight,
		AttitudeWeight:    req.AttitudeWeight,
	})
	if err != nil {
		return nil, err
	}
	resp := kpiSettingsResponseFromDatabase(row)
	return &resp, nil
}

func (r *repository) SetAttitude(ctx context.Context, workspaceAlias, callerID string, req SetAttitudeRequest) (*AttitudeResponse, error) {
	memberUID, err := pgutil.ParseUUID(req.MemberID)
	if err != nil {
		return nil, err
	}
	callerUID, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return nil, err
	}
	monthTime, err := time.Parse("2006-01", req.Month)
	if err != nil {
		return nil, errors.New("month phải theo định dạng YYYY-MM")
	}
	var comment pgtype.Text
	if req.Comment != "" {
		comment = pgtype.Text{String: req.Comment, Valid: true}
	}
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpsertEmployeeMonthlyAttitude(ctx, database.UpsertEmployeeMonthlyAttitudeParams{
		WorkspaceID: wid,
		MemberID:    memberUID,
		Month:       pgDateOf(monthTime),
		Score:       req.Score,
		Comment:     comment,
		UpdatedByID: callerUID,
	})
	if err != nil {
		return nil, err
	}
	resp := &AttitudeResponse{
		MemberID:    pgutil.UUIDToString(row.MemberID),
		Month:       row.Month.Time.Format("2006-01"),
		Score:       row.Score,
		Comment:     textString(row.Comment),
		UpdatedByID: pgutil.UUIDToString(row.UpdatedByID),
		UpdatedAt:   row.UpdatedAt.Time,
	}
	userRow, err := r.queries.GetUserByID(ctx, row.MemberID)
	if err == nil {
		resp.MemberAlias = userRow.Alias
		resp.MemberName = userRow.Name
		resp.MemberAvatar = textString(userRow.Avatar)
	}
	return resp, nil
}

func parseOptionalTimestamptz(s string) pgtype.Timestamptz {
	if s == "" {
		return pgtype.Timestamptz{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02", s)
		if err != nil {
			return pgtype.Timestamptz{}
		}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func pgUUIDOrNullStr(id string) (pgtype.UUID, error) {
	if id == "" {
		return pgtype.UUID{}, nil
	}
	return pgutil.ParseUUID(id)
}

func pgDateOf(t time.Time) pgtype.Date {
	return pgtype.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

func (r *repository) ListWorkspaceScheduleStats(ctx context.Context, workspaceAlias string, from, to time.Time, memberID string) ([]attendance.WorkScheduleDayResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	memberUID, err := pgUUIDOrNullStr(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListWorkSchedulesByAlias(ctx, database.ListWorkSchedulesByAliasParams{
		WorkspaceID: wid,
		MemberID:    memberUID,
		FromDate:    pgtype.Date{Time: from, Valid: true},
		ToDate:      pgtype.Date{Time: to, Valid: true},
		OffsetCount: 0,
		LimitCount:  10000,
	})
	if err != nil {
		return nil, err
	}
	items := make([]attendance.WorkScheduleDayResponse, 0, len(rows))
	for _, row := range rows {
		item := attendance.WorkScheduleDayResponse{
			ID:           pgutil.UUIDToString(row.ID),
			CreatedAt:    row.CreatedAt.Time,
			UpdatedAt:    row.UpdatedAt.Time,
			WorkspaceID:  pgutil.UUIDToString(row.WorkspaceID),
			MemberID:     pgutil.UUIDToString(row.MemberID),
			MemberAlias:  row.MemberAlias,
			MemberName:   row.MemberName,
			MemberAvatar: textString(row.MemberAvatar),
			Date:         row.ScheduleDate.Time.Format(time.DateOnly),
			Status:       attendance.WorkScheduleStatus(row.Status),
		}
		sessions, err := r.queries.ListWorkScheduleSessionsByDayID(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		for _, s := range sessions {
			item.Sessions = append(item.Sessions, attendance.WorkScheduleSessionResponse{
				ID:        pgutil.UUIDToString(s.ID),
				CreatedAt: s.CreatedAt.Time,
				UpdatedAt: s.UpdatedAt.Time,
				DayID:     pgutil.UUIDToString(s.DayID),
				StartAt:   s.StartAt.Time,
				EndAt:     s.EndAt.Time,
			})
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *repository) ListWorkspaceAttendanceStats(ctx context.Context, workspaceAlias string, from, to time.Time, memberID string) ([]attendance.AttendanceRecordResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	memberUID, err := pgUUIDOrNullStr(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListAttendanceRecordsByAlias(ctx, database.ListAttendanceRecordsByAliasParams{
		WorkspaceID: wid,
		MemberID:    memberUID,
		FromTime:    pgtype.Timestamptz{Time: from, Valid: true},
		ToTime:      pgtype.Timestamptz{Time: to, Valid: true},
		OffsetCount: 0,
		LimitCount:  10000,
	})
	if err != nil {
		return nil, err
	}
	items := make([]attendance.AttendanceRecordResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, attendance.AttendanceRecordResponse{
			ID:             pgutil.UUIDToString(row.ID),
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
			WorkspaceID:    pgutil.UUIDToString(row.WorkspaceID),
			MemberID:       pgutil.UUIDToString(row.MemberID),
			MemberAlias:    row.MemberAlias,
			MemberName:     row.MemberName,
			MemberAvatar:   textString(row.MemberAvatar),
			Type:           attendance.AttendanceType(row.Type),
			RecordedAt:     row.RecordedAt.Time,
			Latitude:       row.Latitude,
			Longitude:      row.Longitude,
			WifiBSSID:      row.WifiBssid,
			WifiSSID:       row.WifiSsid,
			DistanceMeters: row.DistanceMeters,
		})
	}
	return items, nil
}

func (r *repository) ListMemberTaskBreakdown(ctx context.Context, workspaceAlias, memberID string, from, to time.Time) ([]memberTaskRow, error) {
	wid, err := pgutil.ParseUUID(workspaceAlias)
	if err != nil {
		return nil, err
	}
	mid, err := pgutil.ParseUUID(memberID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListMemberTaskBreakdown(ctx, database.ListMemberTaskBreakdownParams{
		WorkspaceID: wid,
		MemberID:    mid,
		FromTime:    pgtype.Timestamptz{Time: from, Valid: true},
		ToTime:      pgtype.Timestamptz{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]memberTaskRow, 0, len(rows))
	for _, row := range rows {
		status := string(row.Status)
		out = append(out, memberTaskRow{
			brief: MemberTaskBrief{
				ID:               pgutil.UUIDToString(row.ID),
				Code:             task.Code(row.ProjectName, int(row.SequenceNo)),
				Title:            row.Title,
				ProjectID:        pgutil.UUIDToString(row.ProjectID),
				ProjectName:      row.ProjectName,
				Status:           status,
				Priority:         int(row.Priority),
				StartDate:        timestamptzPtr(row.StartDate),
				EndDate:          timestamptzPtr(row.EndDate),
				FirstCompletedAt: timestamptzPtr(row.FirstCompletedAt),
				ReturnCount:      int(row.ReturnCount),
				Estimation:       int(row.Estimation),
			},
			isWorker:      row.IsWorker,
			isFaultHolder: row.IsFaultHolder,
		})
	}
	return out, nil
}
