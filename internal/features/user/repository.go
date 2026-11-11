package user

import (
	"context"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ExecTx(ctx context.Context, fn func(Repository) error) error
	GetUser(ctx context.Context, id string) (*UserResponse, error)
	GetUserByAlias(ctx context.Context, alias string) (*UserResponse, error)
	UpdateUser(ctx context.Context, id string, req UpdateMeRequest) (*UserResponse, error)
	ListWorkspaceIDsByUser(ctx context.Context, userID string) ([]string, error)
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
	txRepo := &repository{
		queries: r.queries.WithTx(tx),
		db:      r.db,
	}
	if err := fn(txRepo); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

func (r *repository) GetUser(ctx context.Context, id string) (*UserResponse, error) {
	uid, err := pgutil.ParseUUID(id)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.GetUserByID(ctx, uid)
	if err != nil {
		return nil, err
	}
	return userResponsePtrFromDatabaseUser(row), nil
}

func (r *repository) GetUserByAlias(ctx context.Context, alias string) (*UserResponse, error) {
	row, err := r.queries.GetUserByAlias(ctx, alias)
	if err != nil {
		return nil, err
	}
	return userResponsePtrFromDatabaseUser(row), nil
}

func (r *repository) UpdateUser(ctx context.Context, id string, req UpdateMeRequest) (*UserResponse, error) {
	uid, err := pgutil.ParseUUID(id)
	if err != nil {
		return nil, err
	}
	birthdayVal, err := pgtypeDateFromStringPtr(req.Birthday)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.UpdateUserProfile(ctx, database.UpdateUserProfileParams{
		ID:              uid,
		Name:            pgtypeTextFromStringPtr(req.Name),
		Alias:           pgtypeTextFromStringPtr(req.Alias),
		Avatar:          pgtypeTextFromStringPtr(req.Avatar),
		BirthdayPresent: req.Birthday != nil,
		Birthday:        birthdayVal,
		BioPresent:      req.Bio != nil,
		Bio:             pgtypeTextFromStringPtr(req.Bio),
		PhonePresent:    req.Phone != nil,
		Phone:           pgtypeTextFromStringPtr(req.Phone),
		NicknamePresent: req.Nickname != nil,
		Nickname:        pgtypeTextFromStringPtr(req.Nickname),
		Locale:          pgtypeTextFromStringPtr(req.Locale),
	})
	if err != nil {
		return nil, err
	}
	return userResponsePtrFromDatabaseUser(row), nil
}

func (r *repository) ListWorkspaceIDsByUser(ctx context.Context, userID string) ([]string, error) {
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	workspaces, err := r.queries.ListWorkspacesByUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(workspaces))
	for i, ws := range workspaces {
		ids[i] = pgutil.UUIDToString(ws.ID)
	}
	return ids, nil
}
