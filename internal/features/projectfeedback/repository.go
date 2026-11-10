package projectfeedback

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/task"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/alexcao194/kairso/pkg/randtoken"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const tokenByteLength = 18

type Repository interface {
	GetFeedbackSettings(ctx context.Context, projectID string) (*FeedbackSettingsResponse, error)
	EnableFeedbackSettings(ctx context.Context, projectID string) (*FeedbackSettingsResponse, error)
	DisableFeedbackSettings(ctx context.Context, projectID string) (*FeedbackSettingsResponse, error)
	RegenerateFeedbackToken(ctx context.Context, projectID string) (*FeedbackSettingsResponse, error)
	UpdateShowTimeline(ctx context.Context, projectID string, show bool) (*FeedbackSettingsResponse, error)
	GetPublicProjectByToken(ctx context.Context, token string) (*PublicProjectResponse, string, error)
	ListPublicModules(ctx context.Context, projectID string) ([]PublicModuleResponse, error)

	CreateFeedback(ctx context.Context, projectID string, req CreateProjectFeedbackRequest) (*ProjectFeedbackResponse, error)
	ListFeedbacks(ctx context.Context, projectID string, filter ListFeedbacksFilter, params pagination.Params) (ProjectFeedbackListResponse, error)
	ListFeedbacksForTask(ctx context.Context, taskID string) ([]ProjectFeedbackResponse, error)
	GetTaskProjectID(ctx context.Context, taskID string) (string, error)
	GetFeedback(ctx context.Context, feedbackID string) (*ProjectFeedbackResponse, error)
	UpdateFeedback(ctx context.Context, feedbackID string, req UpdateProjectFeedbackRequest) (*ProjectFeedbackResponse, error)
	UpdateFeedbackStatus(ctx context.Context, feedbackID string, status FeedbackStatus) (*ProjectFeedbackResponse, error)
	UpdateFeedbackResponse(ctx context.Context, feedbackID, responderID, response string, expectedFixDate *time.Time) (*ProjectFeedbackResponse, error)
	GetFeedbackSummary(ctx context.Context, projectID string) (*FeedbackSummaryResponse, error)

	LinkTask(ctx context.Context, feedbackID, taskID string) error
	UnlinkTask(ctx context.Context, feedbackID, taskID string) error

	CheckProjectMembership(ctx context.Context, projectID, userID string) (bool, error)
	CheckProjectManageAuth(ctx context.Context, projectID, userID string) (bool, error)
}

type repository struct {
	queries *database.Queries
	db      *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{queries: database.New(pool), db: pool}
}

func (r *repository) GetFeedbackSettings(ctx context.Context, projectID string) (*FeedbackSettingsResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetProjectFeedbackSettingsByProjectID(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return &FeedbackSettingsResponse{Enabled: false}, nil
	}
	if err != nil {
		return nil, err
	}
	return &FeedbackSettingsResponse{Enabled: row.Enabled, Token: row.Token, ShowModuleTimeline: row.ShowModuleTimeline}, nil
}

func (r *repository) EnableFeedbackSettings(ctx context.Context, projectID string) (*FeedbackSettingsResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	_, err = r.queries.GetProjectFeedbackSettingsByProjectID(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		token, tokenErr := randtoken.Generate(tokenByteLength)
		if tokenErr != nil {
			return nil, tokenErr
		}
		row, createErr := r.queries.CreateProjectFeedbackSettings(ctx, database.CreateProjectFeedbackSettingsParams{
			ProjectID: pid, Token: token,
		})
		if createErr != nil {
			return nil, createErr
		}
		return &FeedbackSettingsResponse{Enabled: row.Enabled, Token: row.Token, ShowModuleTimeline: row.ShowModuleTimeline}, nil
	}
	if err != nil {
		return nil, err
	}
	row, err := r.queries.SetProjectFeedbackSettingsEnabled(ctx, database.SetProjectFeedbackSettingsEnabledParams{
		ProjectID: pid, Enabled: true,
	})
	if err != nil {
		return nil, err
	}
	return &FeedbackSettingsResponse{Enabled: row.Enabled, Token: row.Token, ShowModuleTimeline: row.ShowModuleTimeline}, nil
}

func (r *repository) DisableFeedbackSettings(ctx context.Context, projectID string) (*FeedbackSettingsResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	_, err = r.queries.GetProjectFeedbackSettingsByProjectID(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		return &FeedbackSettingsResponse{Enabled: false}, nil
	}
	if err != nil {
		return nil, err
	}
	row, err := r.queries.SetProjectFeedbackSettingsEnabled(ctx, database.SetProjectFeedbackSettingsEnabledParams{
		ProjectID: pid, Enabled: false,
	})
	if err != nil {
		return nil, err
	}
	return &FeedbackSettingsResponse{Enabled: row.Enabled, Token: row.Token, ShowModuleTimeline: row.ShowModuleTimeline}, nil
}

func (r *repository) RegenerateFeedbackToken(ctx context.Context, projectID string) (*FeedbackSettingsResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	token, err := randtoken.Generate(tokenByteLength)
	if err != nil {
		return nil, err
	}
	_, err = r.queries.GetProjectFeedbackSettingsByProjectID(ctx, pid)
	if errors.Is(err, pgx.ErrNoRows) {
		row, createErr := r.queries.CreateProjectFeedbackSettings(ctx, database.CreateProjectFeedbackSettingsParams{
			ProjectID: pid, Token: token,
		})
		if createErr != nil {
			return nil, createErr
		}
		return &FeedbackSettingsResponse{Enabled: row.Enabled, Token: row.Token, ShowModuleTimeline: row.ShowModuleTimeline}, nil
	}
	if err != nil {
		return nil, err
	}
	row, err := r.queries.RegenerateProjectFeedbackToken(ctx, database.RegenerateProjectFeedbackTokenParams{
		ProjectID: pid, Token: token,
	})
	if err != nil {
		return nil, err
	}
	return &FeedbackSettingsResponse{Enabled: row.Enabled, Token: row.Token, ShowModuleTimeline: row.ShowModuleTimeline}, nil
}

func (r *repository) UpdateShowTimeline(ctx context.Context, projectID string, show bool) (*FeedbackSettingsResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateProjectFeedbackSettingsShowTimeline(ctx, database.UpdateProjectFeedbackSettingsShowTimelineParams{
		ProjectID:          pid,
		ShowModuleTimeline: show,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return &FeedbackSettingsResponse{Enabled: false, ShowModuleTimeline: show}, nil
	}
	if err != nil {
		return nil, err
	}
	return &FeedbackSettingsResponse{Enabled: row.Enabled, Token: row.Token, ShowModuleTimeline: row.ShowModuleTimeline}, nil
}

func (r *repository) GetPublicProjectByToken(ctx context.Context, token string) (*PublicProjectResponse, string, error) {
	row, err := r.queries.GetProjectFeedbackSettingsByToken(ctx, token)
	if err != nil {
		return nil, "", err
	}
	return &PublicProjectResponse{ProjectName: row.ProjectName, Enabled: row.Enabled, ShowModuleTimeline: row.ShowModuleTimeline}, pgutil.UUIDToString(row.ProjectID), nil
}

func (r *repository) CreateFeedback(ctx context.Context, projectID string, req CreateProjectFeedbackRequest) (*ProjectFeedbackResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	sequenceNo, err := r.queries.IncrementProjectFeedbackSequence(ctx, pid)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateProjectFeedback(ctx, database.CreateProjectFeedbackParams{
		ProjectID:        pid,
		SubmitterName:    req.SubmitterName,
		SubmitterCompany: req.SubmitterCompany,
		FeatureName:      req.FeatureName,
		Content:          req.Content,
		Priority:         int32(*req.Priority),
		SequenceNo:       sequenceNo,
		FeedbackType:     req.FeedbackType,
	})
	if err != nil {
		return nil, err
	}
	for _, url := range req.Images {
		if _, err := r.queries.CreateProjectFeedbackImage(ctx, database.CreateProjectFeedbackImageParams{
			FeedbackID: row.ID, Url: url,
		}); err != nil {
			return nil, err
		}
	}
	resp := feedbackResponseFromRow(feedbackRowFromCreate(row))
	return r.hydrate(ctx, &resp, row.RespondedByID)
}

func buildFeedbackListConditions(pid pgtype.UUID, filter ListFeedbacksFilter) (*sqlArgs, []string) {
	a := &sqlArgs{}
	conditions := []string{
		fmt.Sprintf("f.project_id = %s::uuid", a.bind(pid)),
		"f.deleted_at IS NULL",
	}
	if len(filter.Priorities) > 0 {
		conditions = append(conditions, fmt.Sprintf("f.priority = ANY(%s::integer[])", a.bind(filter.Priorities)))
	}
	if filter.HasResponse != nil {
		conditions = append(conditions, fmt.Sprintf("(f.response IS NOT NULL) = %s::boolean", a.bind(*filter.HasResponse)))
	}
	if len(filter.Statuses) > 0 {
		statuses := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			statuses[i] = string(s)
		}
		conditions = append(conditions, fmt.Sprintf("f.status = ANY(%s::text[])", a.bind(statuses)))
	}
	if len(filter.FeedbackTypes) > 0 {
		types := make([]string, len(filter.FeedbackTypes))
		for i, t := range filter.FeedbackTypes {
			types[i] = string(t)
		}
		conditions = append(conditions, fmt.Sprintf("f.feedback_type = ANY(%s::text[])", a.bind(types)))
	}
	if filter.Query != "" {
		queryArg := a.bind(filter.Query)
		conditions = append(conditions, fmt.Sprintf(
			`(unaccent(lower(f.submitter_name)) LIKE '%%' || unaccent(lower(%[1]s)) || '%%' OR
			  unaccent(lower(f.submitter_company)) LIKE '%%' || unaccent(lower(%[1]s)) || '%%' OR
			  unaccent(lower(f.feature_name)) LIKE '%%' || unaccent(lower(%[1]s)) || '%%' OR
			  unaccent(lower(f.content)) LIKE '%%' || unaccent(lower(%[1]s)) || '%%' OR
			  unaccent(lower(COALESCE(f.response, ''))) LIKE '%%' || unaccent(lower(%[1]s)) || '%%' OR
			  (regexp_replace(%[1]s, '\D', '', 'g') != '' AND f.sequence_no::text = regexp_replace(%[1]s, '\D', '', 'g')))`,
			queryArg,
		))
	}
	return a, conditions
}

func (r *repository) feedbackListCount(ctx context.Context, a *sqlArgs, conditions []string) (int, error) {
	var count int
	query := fmt.Sprintf(`
		SELECT COUNT(*)::int
		FROM project_feedbacks f
		WHERE %s
	`, strings.Join(conditions, " AND "))
	err := r.db.QueryRow(ctx, query, a.values...).Scan(&count)
	return count, err
}

func (r *repository) ListFeedbacks(ctx context.Context, projectID string, filter ListFeedbacksFilter, params pagination.Params) (ProjectFeedbackListResponse, error) {
	var empty ProjectFeedbackListResponse
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return empty, err
	}

	limit := params.LimitWithMore()
	a, conditions := buildFeedbackListConditions(pid, filter)
	orderBy := buildFeedbackOrderByClause(filter.SortRules)
	limitArg := a.bind(int32(limit))
	offsetArg := a.bind(int32(params.Offset()))

	query := fmt.Sprintf(`
		SELECT f.id, f.created_at, f.updated_at, f.project_id, f.submitter_name, f.submitter_company,
		       f.feature_name, f.content, f.priority, f.status, f.response, f.responded_at, f.responded_by_id,
		       f.expected_fix_date, f.sequence_no, f.feedback_type
		FROM project_feedbacks f
		WHERE %s
		%s
		LIMIT %s OFFSET %s
	`, strings.Join(conditions, " AND "), orderBy, limitArg, offsetArg)

	rows, err := r.db.Query(ctx, query, a.values...)
	if err != nil {
		return empty, err
	}

	items := make([]ProjectFeedbackResponse, 0)
	for rows.Next() {
		var row feedbackRow
		var updatedAt time.Time
		if err := rows.Scan(
			&row.ID, &row.CreatedAt, &updatedAt, &row.ProjectID, &row.SubmitterName, &row.SubmitterCompany,
			&row.FeatureName, &row.Content, &row.Priority, &row.Status, &row.Response, &row.RespondedAt, &row.RespondedByID,
			&row.ExpectedFixDate, &row.SequenceNo, &row.FeedbackType,
		); err != nil {
			rows.Close()
			return empty, err
		}
		resp := feedbackResponseFromRow(row)
		hydrated, err := r.hydrate(ctx, &resp, row.RespondedByID)
		if err != nil {
			rows.Close()
			return empty, err
		}
		items = append(items, *hydrated)
	}
	if err := rows.Err(); err != nil {
		return empty, err
	}
	rows.Close()

	statsArgs, statsConditions := buildFeedbackListConditions(pid, filter)
	totalCount, err := r.feedbackListCount(ctx, statsArgs, statsConditions)
	if err != nil {
		return empty, err
	}

	hasMore := pagination.Process(&items, params.Limit)
	return ProjectFeedbackListResponse{
		Response: pagination.NewResponse(items, hasMore, params.Page, params.Limit),
		Stats:    FeedbackListStats{TotalCount: totalCount},
	}, nil
}

func (r *repository) ListFeedbacksForTask(ctx context.Context, taskID string) ([]ProjectFeedbackResponse, error) {
	tid, err := pgutil.ParseUUID(taskID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListFeedbacksByTaskID(ctx, tid)
	if err != nil {
		return nil, err
	}
	items := make([]ProjectFeedbackResponse, 0, len(rows))
	for _, row := range rows {
		resp := feedbackResponseFromRow(feedbackRowFromListByTask(row))
		hydrated, err := r.hydrate(ctx, &resp, row.RespondedByID)
		if err != nil {
			return nil, err
		}
		items = append(items, *hydrated)
	}
	return items, nil
}

func (r *repository) GetTaskProjectID(ctx context.Context, taskID string) (string, error) {
	tid, err := pgutil.ParseUUID(taskID)
	if err != nil {
		return "", err
	}
	pid, err := r.queries.GetTaskProjectID(ctx, tid)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(pid), nil
}

func (r *repository) GetFeedback(ctx context.Context, feedbackID string) (*ProjectFeedbackResponse, error) {
	fid, err := pgutil.ParseUUID(feedbackID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetProjectFeedbackByID(ctx, fid)
	if err != nil {
		return nil, err
	}
	resp := feedbackResponseFromRow(feedbackRowFromGet(row))
	return r.hydrate(ctx, &resp, row.RespondedByID)
}

func (r *repository) UpdateFeedbackResponse(ctx context.Context, feedbackID, responderID, response string, expectedFixDate *time.Time) (*ProjectFeedbackResponse, error) {
	fid, err := pgutil.ParseUUID(feedbackID)
	if err != nil {
		return nil, err
	}
	rid, err := pgutil.ParseUUID(responderID)
	if err != nil {
		return nil, err
	}

	expectedFixDateArg := pgtype.Timestamptz{}
	if expectedFixDate != nil {
		expectedFixDateArg = pgtype.Timestamptz{Time: *expectedFixDate, Valid: true}
	}

	row, err := r.queries.UpdateProjectFeedbackResponse(ctx, database.UpdateProjectFeedbackResponseParams{
		ID:              fid,
		Response:        pgtype.Text{String: response, Valid: response != ""},
		ExpectedFixDate: expectedFixDateArg,
		RespondedByID:   rid,
	})
	if err != nil {
		return nil, err
	}
	resp := feedbackResponseFromRow(feedbackRowFromUpdate(row))
	return r.hydrate(ctx, &resp, row.RespondedByID)
}

func (r *repository) UpdateFeedback(ctx context.Context, feedbackID string, req UpdateProjectFeedbackRequest) (*ProjectFeedbackResponse, error) {
	fid, err := pgutil.ParseUUID(feedbackID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateProjectFeedback(ctx, database.UpdateProjectFeedbackParams{
		ID:               fid,
		SubmitterName:    req.SubmitterName,
		SubmitterCompany: req.SubmitterCompany,
		FeatureName:      req.FeatureName,
		Content:          req.Content,
		Priority:         int32(*req.Priority),
		FeedbackType:     req.FeedbackType,
	})
	if err != nil {
		return nil, err
	}
	if err := r.queries.DeleteProjectFeedbackImages(ctx, row.ID); err != nil {
		return nil, err
	}
	for _, url := range req.Images {
		if _, err := r.queries.CreateProjectFeedbackImage(ctx, database.CreateProjectFeedbackImageParams{
			FeedbackID: row.ID, Url: url,
		}); err != nil {
			return nil, err
		}
	}
	resp := feedbackResponseFromRow(feedbackRowFromUpdateFeedback(row))
	return r.hydrate(ctx, &resp, row.RespondedByID)
}

func (r *repository) UpdateFeedbackStatus(ctx context.Context, feedbackID string, status FeedbackStatus) (*ProjectFeedbackResponse, error) {
	fid, err := pgutil.ParseUUID(feedbackID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateProjectFeedbackStatus(ctx, database.UpdateProjectFeedbackStatusParams{
		ID:     fid,
		Status: string(status),
	})
	if err != nil {
		return nil, err
	}
	resp := feedbackResponseFromRow(feedbackRowFromUpdateStatus(row))
	return r.hydrate(ctx, &resp, row.RespondedByID)
}

func (r *repository) GetFeedbackSummary(ctx context.Context, projectID string) (*FeedbackSummaryResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	count, err := r.queries.CountPendingProjectFeedbacks(ctx, pid)
	if err != nil {
		return nil, err
	}
	return &FeedbackSummaryResponse{PendingCount: int(count)}, nil
}

func (r *repository) LinkTask(ctx context.Context, feedbackID, taskID string) error {
	fid, err := pgutil.ParseUUID(feedbackID)
	if err != nil {
		return err
	}
	tid, err := pgutil.ParseUUID(taskID)
	if err != nil {
		return err
	}
	feedback, err := r.queries.GetProjectFeedbackByID(ctx, fid)
	if err != nil {
		return err
	}
	taskProjectID, err := r.queries.GetTaskProjectID(ctx, tid)
	if err != nil {
		return err
	}
	if taskProjectID != feedback.ProjectID {
		return errors.New("Task không thuộc dự án của feedback này")
	}
	return r.queries.LinkFeedbackTask(ctx, database.LinkFeedbackTaskParams{FeedbackID: fid, TaskID: tid})
}

func (r *repository) UnlinkTask(ctx context.Context, feedbackID, taskID string) error {
	fid, err := pgutil.ParseUUID(feedbackID)
	if err != nil {
		return err
	}
	tid, err := pgutil.ParseUUID(taskID)
	if err != nil {
		return err
	}
	return r.queries.UnlinkFeedbackTask(ctx, database.UnlinkFeedbackTaskParams{FeedbackID: fid, TaskID: tid})
}

func (r *repository) CheckProjectMembership(ctx context.Context, projectID, userID string) (bool, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return false, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return false, err
	}
	return r.queries.CheckProjectMembership(ctx, database.CheckProjectMembershipParams{ProjectID: pid, UserID: uid})
}

func (r *repository) CheckProjectManageAuth(ctx context.Context, projectID, userID string) (bool, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return false, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return false, err
	}
	return r.queries.CheckProjectManageAuth(ctx, database.CheckProjectManageAuthParams{ProjectID: pid, UserID: uid})
}

func (r *repository) hydrate(ctx context.Context, resp *ProjectFeedbackResponse, respondedByID pgtype.UUID) (*ProjectFeedbackResponse, error) {
	fid, err := pgutil.ParseUUID(resp.ID)
	if err != nil {
		return nil, err
	}
	images, err := r.queries.ListProjectFeedbackImagesByFeedbackID(ctx, fid)
	if err != nil {
		return nil, err
	}
	imageURLs := make([]string, 0, len(images))
	for _, img := range images {
		imageURLs = append(imageURLs, img.Url)
	}
	resp.Images = imageURLs

	projectName, err := r.queries.GetProjectNameByID(ctx, pgutil.MustUUID(resp.ProjectID))
	if err != nil {
		return nil, err
	}
	resp.Code = task.FeedbackCode(projectName, resp.SequenceNo)

	taskRows, err := r.queries.ListFeedbackTaskSummaries(ctx, fid)
	if err != nil {
		return nil, err
	}
	if len(taskRows) > 0 {
		tasks := make([]LinkedTaskSummary, 0, len(taskRows))
		for _, t := range taskRows {
			status := string(t.Status)
			tasks = append(tasks, LinkedTaskSummary{
				Code:   task.Code(projectName, int(t.SequenceNo)),
				Status: status,
			})
		}
		resp.Tasks = tasks
	}

	if respondedByID.Valid {
		user, err := r.queries.GetUserByID(ctx, respondedByID)
		if err == nil {
			resp.RespondedBy = &UserSummary{
				ID: pgutil.UUIDToString(user.ID), Name: user.Name, Alias: user.Alias, Avatar: textString(user.Avatar),
			}
		}
	}
	return resp, nil
}

func (r *repository) ListPublicModules(ctx context.Context, projectID string) ([]PublicModuleResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	modules, err := r.queries.ListPublicModulesByProjectID(ctx, pid)
	if err != nil {
		return nil, err
	}
	if len(modules) == 0 {
		return []PublicModuleResponse{}, nil
	}
	ids := make([]pgtype.UUID, 0, len(modules))
	for _, m := range modules {
		ids = append(ids, m.ID)
	}
	sprints, err := r.queries.ListPublicSprintsByModuleIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return publicModulesFromDatabase(modules, sprints), nil
}
