package project

import (
	"context"
	"fmt"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func vietnamLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}

func monthRange(month string) (from, to pgtype.Timestamptz, err error) {
	if month == "" {
		return from, to, nil
	}
	start, err := time.ParseInLocation("2006-01", month, vietnamLocation())
	if err != nil {
		return from, to, fmt.Errorf("month phải theo định dạng YYYY-MM")
	}
	end := start.AddDate(0, 1, 0).Add(-time.Second)
	return pgtype.Timestamptz{Time: start, Valid: true}, pgtype.Timestamptz{Time: end, Valid: true}, nil
}

type Repository interface {
	ExecTx(ctx context.Context, fn func(Repository) error) error
	CreateProject(ctx context.Context, workspaceID, userID string, req CreateProjectRequest) (*ProjectResponse, error)
	ListProjects(ctx context.Context, workspaceID, userID string, params pagination.StatsParams) (pagination.Response[ProjectResponse], error)
	GetProjectOverviewsByIDs(ctx context.Context, projectIDs []string) ([]ProjectOverviewResponse, error)
	UpdateProject(ctx context.Context, projectID, callerID string, req UpdateProjectRequest) (*ProjectResponse, error)
	ListProjectHistory(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectHistoryResponse], error)
	DeleteProject(ctx context.Context, projectID string) error

	GetCallerWorkspaceRole(ctx context.Context, workspaceID, callerID string) (ProjectRole, error)
	GetProjectMemberRole(ctx context.Context, projectID, userID string) (ProjectRole, error)
	GetProjectContext(ctx context.Context, projectID string) (ProjectContext, error)
	ListProjectMembers(ctx context.Context, projectID string, params pagination.Params, includeAvailable bool) (*ListProjectMembersResponse, error)
	AddProjectMember(ctx context.Context, projectID, userID string, role ProjectRole) (*ProjectMemberResponse, error)
	UpdateProjectMemberRole(ctx context.Context, projectID, userID string, role ProjectRole) (*ProjectMemberResponse, error)
	RemoveProjectMember(ctx context.Context, projectID, userID string) error

	CreateProjectLink(ctx context.Context, projectID, userID string, req CreateProjectLinkRequest) (*ProjectLinkResponse, error)
	ListProjectLinks(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectLinkResponse], error)
	UpdateProjectLink(ctx context.Context, linkID string, req UpdateProjectLinkRequest) (*ProjectLinkResponse, error)
	DeleteProjectLink(ctx context.Context, linkID string) (string, error)

	CreateProjectNote(ctx context.Context, projectID string, userID string, req CreateProjectNoteRequest) (*ProjectNoteResponse, error)
	ListProjectNotes(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectNoteResponse], error)
	UpdateProjectNote(ctx context.Context, noteID string, req UpdateProjectNoteRequest) (*ProjectNoteResponse, error)
	DeleteProjectNote(ctx context.Context, noteID string) (string, error)

	CreateProjectFile(ctx context.Context, projectID string, userID string, req CreateProjectFileRequest) (*ProjectFileResponse, error)
	ListProjectFiles(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectFileResponse], error)
	DeleteProjectFile(ctx context.Context, fileID string) (string, error)
}

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

func (r *repository) CreateProject(ctx context.Context, workspaceID, userID string, req CreateProjectRequest) (*ProjectResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	txQueries := database.New(tx)
	defer tx.Rollback(ctx)

	projectRow, err := txQueries.CreateProject(ctx, database.CreateProjectParams{
		WorkspaceID: wid,
		Name:        req.Name,
	})
	if err != nil {
		return nil, err
	}
	project := projectResponseFromDatabaseProject(projectRow)
	if _, err := txQueries.UpsertProjectUser(ctx, database.UpsertProjectUserParams{
		ProjectID: projectRow.ID,
		UserID:    uid,
		Column3:   database.ProjectRole(ProjectRoleHolder),
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &project, nil
}

type ProjectContext struct {
	WorkspaceID string
	Name        string
}

func (r *repository) GetProjectContext(ctx context.Context, projectID string) (ProjectContext, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return ProjectContext{}, err
	}
	project, err := r.queries.GetProjectByID(ctx, pid)
	if err != nil {
		return ProjectContext{}, err
	}
	return ProjectContext{
		WorkspaceID: pgutil.UUIDToString(project.WorkspaceID),
		Name:        project.Name,
	}, nil
}

func (r *repository) ListProjects(ctx context.Context, workspaceID, userID string, params pagination.StatsParams) (pagination.Response[ProjectResponse], error) {
	var empty pagination.Response[ProjectResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()

	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return empty, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return empty, err
	}
	fromTime, toTime, err := monthRange(params.Month)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListProjectsByWorkspaceAliasWithStats(ctx, database.ListProjectsByWorkspaceAliasWithStatsParams{
		UserID:      uid,
		WorkspaceID: wid,
		FromTime:    fromTime,
		ToTime:      toTime,
		RowLimit:    int32(limit),
		RowOffset:   int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]ProjectResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectResponseFromDatabaseProjectWithStats(row))
	}

	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) GetProjectOverviewsByIDs(ctx context.Context, projectIDs []string) ([]ProjectOverviewResponse, error) {
	if len(projectIDs) == 0 {
		return []ProjectOverviewResponse{}, nil
	}
	ids := make([]pgtype.UUID, 0, len(projectIDs))
	for _, id := range projectIDs {
		pid, err := pgutil.ParseUUID(id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, pid)
	}
	rows, err := r.queries.GetProjectOverviewsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	items := make([]ProjectOverviewResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectOverviewResponseFromBatchRow(row))
	}
	return items, nil
}

func (r *repository) UpdateProject(ctx context.Context, projectID, callerID string, req UpdateProjectRequest) (*ProjectResponse, error) {
	callerUID, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return nil, err
	}
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	txQueries := database.New(tx)
	defer tx.Rollback(ctx)
	beforeRow, err := txQueries.GetProjectByIDForUpdate(ctx, pid)
	if err != nil {
		return nil, err
	}
	before := projectResponseFromDatabaseProject(beforeRow)
	afterRow, err := txQueries.UpdateProjectByID(ctx, database.UpdateProjectByIDParams{
		NamePresent:   req.namePresent(),
		Name:          stringValue(req.Name),
		AvatarPresent: req.avatarPresent(),
		Avatar:        stringValue(req.Avatar),
		ID:            pid,
	})
	if err != nil {
		return nil, err
	}
	after := projectResponseFromDatabaseProject(afterRow)
	if req.namePresent() {
		if err := insertHistory(ctx, txQueries, "project", after.ID, callerUID, "name", stringPtr(before.Name), stringPtr(after.Name)); err != nil {
			return nil, err
		}
	}
	if req.avatarPresent() {
		if err := insertHistory(ctx, txQueries, "project", after.ID, callerUID, "avatar", emptyToNil(before.Avatar), emptyToNil(after.Avatar)); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &after, nil
}

func (r *repository) ListProjectHistory(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectHistoryResponse], error) {
	var empty pagination.Response[ProjectHistoryResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListProjectHistories(ctx, database.ListProjectHistoriesParams{
		EntityID: pid,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]ProjectHistoryResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectHistoryResponseFromDatabaseHistory(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) DeleteProject(ctx context.Context, projectID string) error {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return err
	}
	return r.queries.SoftDeleteProjectByID(ctx, pid)
}

func (r *repository) GetCallerWorkspaceRole(ctx context.Context, workspaceID, callerID string) (ProjectRole, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return "", err
	}
	uid, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return "", err
	}
	role, err := r.queries.GetCallerWorkspaceRoleForProject(ctx, database.GetCallerWorkspaceRoleForProjectParams{
		WorkspaceID: wid,
		UserID:      uid,
	})
	if err != nil {
		return "", err
	}
	return ProjectRole(string(role)), nil
}

func (r *repository) GetProjectMemberRole(ctx context.Context, projectID, userID string) (ProjectRole, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return "", err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return "", err
	}
	role, err := r.queries.GetProjectMemberRole(ctx, database.GetProjectMemberRoleParams{ProjectID: pid, UserID: uid})
	if err != nil {
		return "", err
	}
	return ProjectRole(string(role)), nil
}

func (r *repository) ListProjectMembers(ctx context.Context, projectID string, params pagination.Params, includeAvailable bool) (*ListProjectMembersResponse, error) {
	limit := params.LimitWithMore()
	offset := params.Offset()

	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListProjectMembersByAlias(ctx, database.ListProjectMembersByAliasParams{
		ProjectID: pid,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return nil, err
	}

	resp := &ListProjectMembersResponse{
		Items: make([]ProjectMemberResponse, 0),
	}
	for _, row := range rows {
		resp.Items = append(resp.Items, projectMemberResponseFromListRow(row))
	}

	resp.HasMore = pagination.Process(&resp.Items, params.Limit)
	resp.Page = params.Page
	resp.Limit = params.Limit

	if !includeAvailable {
		return resp, nil
	}
	resp.AvailableMembers = make([]ProjectAvailableMemberResponse, 0)

	project, err := r.queries.GetProjectByID(ctx, pid)
	if err != nil {
		return nil, err
	}
	availableRows, err := r.queries.ListAvailableProjectMembers(ctx, database.ListAvailableProjectMembersParams{
		ProjectID:   pid,
		WorkspaceID: project.WorkspaceID,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range availableRows {
		resp.AvailableMembers = append(resp.AvailableMembers, projectAvailableMemberResponseFromDatabaseRow(row))
	}
	return resp, nil
}

func (r *repository) AddProjectMember(ctx context.Context, projectID, userID string, role ProjectRole) (*ProjectMemberResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.AddProjectMemberByID(ctx, database.AddProjectMemberByIDParams{
		ProjectID: pid,
		UserID:    uid,
		Role:      database.ProjectRole(role),
	})
	if err != nil {
		return nil, err
	}
	resp := projectMemberResponseFromAddRow(row)
	return &resp, nil
}

func (r *repository) UpdateProjectMemberRole(ctx context.Context, projectID, userID string, role ProjectRole) (*ProjectMemberResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateProjectMemberRoleByID(ctx, database.UpdateProjectMemberRoleByIDParams{
		ProjectID: pid,
		UserID:    uid,
		Column3:   database.ProjectRole(role),
	})
	if err != nil {
		return nil, err
	}
	resp := projectMemberResponseFromUpdateRow(row)
	return &resp, nil
}

func (r *repository) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.DeleteProjectUser(ctx, database.DeleteProjectUserParams{ProjectID: pid, UserID: uid})
}

func (r *repository) CreateProjectLink(ctx context.Context, projectID, userID string, req CreateProjectLinkRequest) (*ProjectLinkResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateProjectLink(ctx, database.CreateProjectLinkParams{ProjectID: pid, Title: req.Title, Url: req.URL, CreatedByID: uid})
	if err != nil {
		return nil, err
	}
	resp := projectLinkResponseFromDatabaseCreateLink(row)
	return &resp, nil
}

func (r *repository) ListProjectLinks(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectLinkResponse], error) {
	var empty pagination.Response[ProjectLinkResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()

	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListProjectLinks(ctx, database.ListProjectLinksParams{
		ProjectID: pid,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]ProjectLinkResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectLinkResponseFromDatabaseListLink(row))
	}

	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) UpdateProjectLink(ctx context.Context, linkID string, req UpdateProjectLinkRequest) (*ProjectLinkResponse, error) {
	lid, err := pgutil.ParseUUID(linkID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateProjectLink(ctx, database.UpdateProjectLinkParams{
		ID:        lid,
		LinkTitle: pgtypeText(req.Title),
		LinkUrl:   pgtypeText(req.URL),
	})
	if err != nil {
		return nil, err
	}
	resp := projectLinkResponseFromDatabaseUpdateLink(row)
	return &resp, nil
}

func (r *repository) DeleteProjectLink(ctx context.Context, linkID string) (string, error) {
	lid, err := pgutil.ParseUUID(linkID)
	if err != nil {
		return "", err
	}
	pid, err := r.queries.DeleteProjectLink(ctx, lid)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(pid), nil
}

func (r *repository) CreateProjectNote(ctx context.Context, projectID string, userID string, req CreateProjectNoteRequest) (*ProjectNoteResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateProjectNote(ctx, database.CreateProjectNoteParams{
		ProjectID:   pid,
		Title:       req.Title,
		Content:     req.Content,
		CreatedByID: uid,
	})
	if err != nil {
		return nil, err
	}
	resp := projectNoteResponseFromDatabaseCreateNote(row)
	return &resp, nil
}

func (r *repository) ListProjectNotes(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectNoteResponse], error) {
	var empty pagination.Response[ProjectNoteResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()

	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListProjectNotes(ctx, database.ListProjectNotesParams{
		ProjectID: pid,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]ProjectNoteResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectNoteResponseFromDatabaseListNote(row))
	}

	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) UpdateProjectNote(ctx context.Context, noteID string, req UpdateProjectNoteRequest) (*ProjectNoteResponse, error) {
	nid, err := pgutil.ParseUUID(noteID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateProjectNote(ctx, database.UpdateProjectNoteParams{
		ID:          nid,
		NoteTitle:   pgtypeText(req.Title),
		NoteContent: pgtypeText(req.Content),
	})
	if err != nil {
		return nil, err
	}
	resp := projectNoteResponseFromDatabaseUpdateNote(row)
	return &resp, nil
}

func (r *repository) DeleteProjectNote(ctx context.Context, noteID string) (string, error) {
	nid, err := pgutil.ParseUUID(noteID)
	if err != nil {
		return "", err
	}
	pid, err := r.queries.DeleteProjectNote(ctx, nid)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(pid), nil
}

func (r *repository) CreateProjectFile(ctx context.Context, projectID string, userID string, req CreateProjectFileRequest) (*ProjectFileResponse, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateProjectFile(ctx, database.CreateProjectFileParams{
		ProjectID:   pid,
		Title:       req.Title,
		FileUrl:     req.FileURL,
		FileName:    req.FileName,
		ContentType: req.ContentType,
		SizeBytes:   req.SizeBytes,
		CreatedByID: uid,
	})
	if err != nil {
		return nil, err
	}
	resp := projectFileResponseFromDatabaseCreateFile(row)
	return &resp, nil
}

func (r *repository) ListProjectFiles(ctx context.Context, projectID string, params pagination.Params) (pagination.Response[ProjectFileResponse], error) {
	var empty pagination.Response[ProjectFileResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()

	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListProjectFiles(ctx, database.ListProjectFilesParams{
		ProjectID: pid,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]ProjectFileResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectFileResponseFromDatabaseListFile(row))
	}

	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) DeleteProjectFile(ctx context.Context, fileID string) (string, error) {
	fid, err := pgutil.ParseUUID(fileID)
	if err != nil {
		return "", err
	}
	pid, err := r.queries.DeleteProjectFile(ctx, fid)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(pid), nil
}

func insertHistory(ctx context.Context, txQueries *database.Queries, entityType, entityID string, changedByID pgtype.UUID, field string, oldValue, newValue *string) error {
	if equalStringPtr(oldValue, newValue) {
		return nil
	}
	eid, err := pgutil.ParseUUID(entityID)
	if err != nil {
		return err
	}
	return txQueries.InsertEntityHistory(ctx, database.InsertEntityHistoryParams{
		EntityType:  entityType,
		EntityID:    eid,
		ChangedByID: changedByID,
		Field:       field,
		OldValue:    pgtypeText(oldValue),
		NewValue:    pgtypeText(newValue),
	})
}

func emptyToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func equalStringPtr(left, right *string) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}
