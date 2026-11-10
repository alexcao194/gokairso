package workspace

import (
	"context"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ExecTx(ctx context.Context, fn func(Repository) error) error
	CreateWorkspace(ctx context.Context, userID string, req CreateWorkspaceRequest) (*WorkspaceResponse, error)
	ListWorkspaces(ctx context.Context, userID string, params pagination.Params) (pagination.Response[WorkspaceResponse], error)
	GetWorkspace(ctx context.Context, workspaceID string) (*WorkspaceResponse, error)
	UpdateWorkspace(ctx context.Context, workspaceID, callerID string, req UpdateWorkspaceRequest) (*WorkspaceResponse, error)
	ListWorkspaceHistory(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceHistoryResponse], error)
	DeleteWorkspace(ctx context.Context, workspaceID string) error

	GetCallerRole(ctx context.Context, workspaceID, callerID string) (WorkspaceRole, error)
	ListWorkspaceMembers(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceMemberResponse], error)
	ListMyInvitations(ctx context.Context, userID string, params pagination.Params) (pagination.Response[WorkspaceInvitationSummaryResponse], error)
	GetUserIDByAlias(ctx context.Context, userAlias string) (string, error)
	InviteMember(ctx context.Context, workspaceID, inviteeID, inviterID string, role WorkspaceRole) (*WorkspaceInvitationResponse, error)
	GetWorkspaceIDByInvitationID(ctx context.Context, invitationID string) (string, error)
	AcceptInvitation(ctx context.Context, invitationID, callerID string) error
	RejectInvitation(ctx context.Context, invitationID, callerID string) error
	CancelInvitation(ctx context.Context, invitationID string) error
	RemoveMember(ctx context.Context, workspaceID, userID string) error
	UpdateMemberRole(ctx context.Context, workspaceID, userID string, role WorkspaceRole) (*WorkspaceMemberResponse, error)

	CreateWorkspaceLink(ctx context.Context, workspaceID, userID string, req CreateWorkspaceLinkRequest) (*WorkspaceLinkResponse, error)
	ListWorkspaceLinks(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceLinkResponse], error)
	DeleteWorkspaceLink(ctx context.Context, linkID string) (string, error)

	CreateWorkspaceNote(ctx context.Context, workspaceID string, userID string, req CreateWorkspaceNoteRequest) (*WorkspaceNoteResponse, error)
	ListWorkspaceNotes(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceNoteResponse], error)
	UpdateWorkspaceNote(ctx context.Context, noteID string, req UpdateWorkspaceNoteRequest) (*WorkspaceNoteResponse, error)
	DeleteWorkspaceNote(ctx context.Context, noteID string) (string, error)

	CreateWorkspaceFile(ctx context.Context, workspaceID string, userID string, req CreateWorkspaceFileRequest) (*WorkspaceFileResponse, error)
	ListWorkspaceFiles(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceFileResponse], error)
	DeleteWorkspaceFile(ctx context.Context, fileID string) (string, error)
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

func (r *repository) CreateWorkspace(ctx context.Context, userID string, req CreateWorkspaceRequest) (*WorkspaceResponse, error) {
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

	workspaceRow, err := txQueries.CreateWorkspace(ctx, database.CreateWorkspaceParams{CreatedByID: uid, Name: req.Name})
	if err != nil {
		return nil, err
	}
	workspace := workspaceResponseFromDatabaseWorkspace(workspaceRow)
	if _, err := txQueries.UpsertWorkspaceUser(ctx, database.UpsertWorkspaceUserParams{
		WorkspaceID: workspaceRow.ID,
		UserID:      uid,
		Column3:     database.WorkspaceRole(WorkspaceRoleHolder),
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (r *repository) ListWorkspaces(ctx context.Context, userID string, params pagination.Params) (pagination.Response[WorkspaceResponse], error) {
	var empty pagination.Response[WorkspaceResponse]
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return empty, err
	}

	limit := params.LimitWithMore()
	offset := params.Offset()

	rows, err := r.queries.ListWorkspacesByUserPaginated(ctx, database.ListWorkspacesByUserPaginatedParams{
		UserID: uid,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]WorkspaceResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, workspaceResponseFromDatabaseWorkspace(row))
	}

	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) GetWorkspace(ctx context.Context, workspaceID string) (*WorkspaceResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetWorkspaceByID(ctx, wid)
	if err != nil {
		return nil, err
	}
	resp := workspaceResponseFromDatabaseWorkspace(row)
	return &resp, nil
}

func (r *repository) UpdateWorkspace(ctx context.Context, workspaceID, callerID string, req UpdateWorkspaceRequest) (*WorkspaceResponse, error) {
	callerUID, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return nil, err
	}
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	txQueries := database.New(tx)
	defer tx.Rollback(ctx)

	beforeRow, err := txQueries.GetWorkspaceByIDForUpdate(ctx, wid)
	if err != nil {
		return nil, err
	}
	before := workspaceResponseFromDatabaseWorkspace(beforeRow)
	afterRow, err := txQueries.UpdateWorkspaceByID(ctx, database.UpdateWorkspaceByIDParams{
		NamePresent:   req.namePresent(),
		Name:          stringValue(req.Name),
		AvatarPresent: req.avatarPresent(),
		Avatar:        stringValue(req.Avatar),
		ID:            wid,
	})
	if err != nil {
		return nil, err
	}
	after := workspaceResponseFromDatabaseWorkspace(afterRow)
	if req.namePresent() {
		if err := insertHistory(ctx, txQueries, "workspace", after.ID, callerUID, "name", stringPtr(before.Name), stringPtr(after.Name)); err != nil {
			return nil, err
		}
	}
	if req.avatarPresent() {
		if err := insertHistory(ctx, txQueries, "workspace", after.ID, callerUID, "avatar", emptyToNil(before.Avatar), emptyToNil(after.Avatar)); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &after, nil
}

func (r *repository) ListWorkspaceHistory(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceHistoryResponse], error) {
	var empty pagination.Response[WorkspaceHistoryResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListWorkspaceHistories(ctx, database.ListWorkspaceHistoriesParams{
		EntityID: wid,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]WorkspaceHistoryResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, workspaceHistoryResponseFromDatabaseHistory(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return err
	}
	return r.queries.SoftDeleteWorkspaceByID(ctx, wid)
}

func (r *repository) GetCallerRole(ctx context.Context, workspaceID, callerID string) (WorkspaceRole, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
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
	return WorkspaceRole(string(role)), nil
}

func (r *repository) ListWorkspaceMembers(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceMemberResponse], error) {
	var empty pagination.Response[WorkspaceMemberResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()

	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListWorkspaceMembers(ctx, database.ListWorkspaceMembersParams{
		WorkspaceID: wid,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return empty, err
	}

	members := make([]WorkspaceMemberResponse, 0)
	for _, row := range rows {
		members = append(members, workspaceMemberResponseFromListRow(row))
	}

	hasMore := pagination.Process(&members, params.Limit)
	return pagination.NewResponse(members, hasMore, params.Page, params.Limit), nil
}

func (r *repository) ListMyInvitations(ctx context.Context, userID string, params pagination.Params) (pagination.Response[WorkspaceInvitationSummaryResponse], error) {
	var empty pagination.Response[WorkspaceInvitationSummaryResponse]
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return empty, err
	}
	limit := params.LimitWithMore()
	offset := params.Offset()

	rows, err := r.queries.ListMyWorkspaceInvitations(ctx, database.ListMyWorkspaceInvitationsParams{
		InviteeID: uid,
		Limit:     int32(limit),
		Offset:    int32(offset),
	})
	if err != nil {
		return empty, err
	}

	items := make([]WorkspaceInvitationSummaryResponse, 0)
	for _, row := range rows {
		items = append(items, workspaceInvitationSummaryResponseFromDatabaseRow(row))
	}

	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) InviteMember(ctx context.Context, workspaceID, inviteeID, inviterID string, role WorkspaceRole) (*WorkspaceInvitationResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	inviteeUID, err := pgutil.ParseUUID(inviteeID)
	if err != nil {
		return nil, err
	}
	inviterUID, err := pgutil.ParseUUID(inviterID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.InviteWorkspaceMember(ctx, database.InviteWorkspaceMemberParams{
		WorkspaceID: wid,
		InviteeID:   inviteeUID,
		InviterID:   inviterUID,
		Role:        database.WorkspaceRole(role),
	})
	if err != nil {
		return nil, err
	}
	resp := workspaceInvitationResponseFromDatabaseInvitation(row)
	return &resp, nil
}

func (r *repository) GetWorkspaceIDByInvitationID(ctx context.Context, invitationID string) (string, error) {
	invUID, err := pgutil.ParseUUID(invitationID)
	if err != nil {
		return "", err
	}
	wid, err := r.queries.GetWorkspaceIDByInvitationID(ctx, invUID)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(wid), nil
}

func (r *repository) AcceptInvitation(ctx context.Context, invitationID, callerID string) error {
	callerUID, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return err
	}
	invUID, err := pgutil.ParseUUID(invitationID)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	txQueries := database.New(tx)
	defer tx.Rollback(ctx)

	invitation, err := txQueries.GetPendingWorkspaceInvitationForAccept(ctx, database.GetPendingWorkspaceInvitationForAcceptParams{
		ID:        invUID,
		InviteeID: callerUID,
	})
	if err != nil {
		return err
	}

	if err := txQueries.AcceptWorkspaceInvitation(ctx, invUID); err != nil {
		return err
	}
	if _, err := txQueries.UpsertWorkspaceUser(ctx, database.UpsertWorkspaceUserParams{
		WorkspaceID: invitation.WorkspaceID,
		UserID:      callerUID,
		Column3:     invitation.Role,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *repository) RejectInvitation(ctx context.Context, invitationID, callerID string) error {
	callerUID, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return err
	}
	invUID, err := pgutil.ParseUUID(invitationID)
	if err != nil {
		return err
	}
	return r.queries.RejectWorkspaceInvitation(ctx, database.RejectWorkspaceInvitationParams{ID: invUID, InviteeID: callerUID})
}

func (r *repository) CancelInvitation(ctx context.Context, invitationID string) error {
	invUID, err := pgutil.ParseUUID(invitationID)
	if err != nil {
		return err
	}
	return r.queries.CancelWorkspaceInvitation(ctx, invUID)
}

func (r *repository) RemoveMember(ctx context.Context, workspaceID, userID string) error {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.DeleteWorkspaceUser(ctx, database.DeleteWorkspaceUserParams{WorkspaceID: wid, UserID: uid})
}

func (r *repository) UpdateMemberRole(ctx context.Context, workspaceID, userID string, role WorkspaceRole) (*WorkspaceMemberResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateWorkspaceMemberRole(ctx, database.UpdateWorkspaceMemberRoleParams{
		WorkspaceID: wid,
		UserID:      uid,
		Column3:     database.WorkspaceRole(role),
	})
	if err != nil {
		return nil, err
	}
	resp := workspaceMemberResponseFromUpdateRow(row)
	return &resp, nil
}

func (r *repository) CreateWorkspaceLink(ctx context.Context, workspaceID, userID string, req CreateWorkspaceLinkRequest) (*WorkspaceLinkResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateWorkspaceLink(ctx, database.CreateWorkspaceLinkParams{WorkspaceID: wid, Title: req.Title, Url: req.URL, CreatedByID: uid})
	if err != nil {
		return nil, err
	}
	resp := workspaceLinkResponseFromDatabaseCreateLink(row)
	return &resp, nil
}

func (r *repository) ListWorkspaceLinks(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceLinkResponse], error) {
	var empty pagination.Response[WorkspaceLinkResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListWorkspaceLinks(ctx, database.ListWorkspaceLinksParams{
		WorkspaceID: wid,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]WorkspaceLinkResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, workspaceLinkResponseFromDatabaseListLink(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) DeleteWorkspaceLink(ctx context.Context, linkID string) (string, error) {
	linkUID, err := pgutil.ParseUUID(linkID)
	if err != nil {
		return "", err
	}
	wid, err := r.queries.DeleteWorkspaceLink(ctx, linkUID)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(wid), nil
}

func (r *repository) CreateWorkspaceNote(ctx context.Context, workspaceID string, userID string, req CreateWorkspaceNoteRequest) (*WorkspaceNoteResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateWorkspaceNote(ctx, database.CreateWorkspaceNoteParams{
		WorkspaceID: wid,
		Title:       req.Title,
		Content:     req.Content,
		CreatedByID: uid,
	})
	if err != nil {
		return nil, err
	}
	resp := workspaceNoteResponseFromDatabaseCreateNote(row)
	return &resp, nil
}

func (r *repository) ListWorkspaceNotes(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceNoteResponse], error) {
	var empty pagination.Response[WorkspaceNoteResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListWorkspaceNotes(ctx, database.ListWorkspaceNotesParams{
		WorkspaceID: wid,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]WorkspaceNoteResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, workspaceNoteResponseFromDatabaseListNote(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) UpdateWorkspaceNote(ctx context.Context, noteID string, req UpdateWorkspaceNoteRequest) (*WorkspaceNoteResponse, error) {
	noteUID, err := pgutil.ParseUUID(noteID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateWorkspaceNote(ctx, database.UpdateWorkspaceNoteParams{
		ID:          noteUID,
		NoteTitle:   pgtypeText(req.Title),
		NoteContent: pgtypeText(req.Content),
	})
	if err != nil {
		return nil, err
	}
	resp := workspaceNoteResponseFromDatabaseUpdateNote(row)
	return &resp, nil
}

func (r *repository) DeleteWorkspaceNote(ctx context.Context, noteID string) (string, error) {
	noteUID, err := pgutil.ParseUUID(noteID)
	if err != nil {
		return "", err
	}
	wid, err := r.queries.DeleteWorkspaceNote(ctx, noteUID)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(wid), nil
}

func (r *repository) CreateWorkspaceFile(ctx context.Context, workspaceID string, userID string, req CreateWorkspaceFileRequest) (*WorkspaceFileResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateWorkspaceFile(ctx, database.CreateWorkspaceFileParams{
		WorkspaceID: wid,
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
	resp := workspaceFileResponseFromDatabaseCreateFile(row)
	return &resp, nil
}

func (r *repository) ListWorkspaceFiles(ctx context.Context, workspaceID string, params pagination.Params) (pagination.Response[WorkspaceFileResponse], error) {
	var empty pagination.Response[WorkspaceFileResponse]
	limit := params.LimitWithMore()
	offset := params.Offset()
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListWorkspaceFiles(ctx, database.ListWorkspaceFilesParams{
		WorkspaceID: wid,
		Limit:       int32(limit),
		Offset:      int32(offset),
	})
	if err != nil {
		return empty, err
	}
	items := make([]WorkspaceFileResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, workspaceFileResponseFromDatabaseListFile(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) DeleteWorkspaceFile(ctx context.Context, fileID string) (string, error) {
	fileUID, err := pgutil.ParseUUID(fileID)
	if err != nil {
		return "", err
	}
	wid, err := r.queries.DeleteWorkspaceFile(ctx, fileUID)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(wid), nil
}

func (r *repository) GetUserIDByAlias(ctx context.Context, userAlias string) (string, error) {
	row, err := r.queries.GetUserByAlias(ctx, userAlias)
	if err != nil {
		return "", err
	}
	return pgutil.UUIDToString(row.ID), nil
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
