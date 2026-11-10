package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAliasAlreadyExists = errors.New("Alias đã tồn tại")

type Repository interface {
	ExecTx(ctx context.Context, fn func(Repository) error) error
	UpsertFirebaseUser(ctx context.Context, params FirebaseUserParams) (*AuthUser, error)
	UpdateFirebaseUser(ctx context.Context, params FirebaseUserParams) (*AuthUser, error)
	InsertFirebaseUser(ctx context.Context, params FirebaseUserParams, alias string) (*AuthUser, error)
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

func (r *repository) UpdateFirebaseUser(ctx context.Context, params FirebaseUserParams) (*AuthUser, error) {
	row, err := r.queries.UpdateFirebaseUser(ctx, database.UpdateFirebaseUserParams{
		FirebaseUid:      params.FirebaseUID,
		Column2:          params.Name,
		Column3:          params.Avatar,
		Column4:          params.Email,
		EmailVerified:    params.EmailVerified,
		FirebaseProvider: params.Provider,
	})
	if err != nil {
		return nil, err
	}
	return authUserFromDatabaseUser(row), nil
}

func (r *repository) UpsertFirebaseUser(ctx context.Context, params FirebaseUserParams) (*AuthUser, error) {
	params = normalizeFirebaseUserParams(params)
	if params.FirebaseUID == "" {
		return nil, ErrInvalidFirebaseToken
	}

	updated, err := r.UpdateFirebaseUser(ctx, params)
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	for _, alias := range aliasCandidates(params) {
		user, insertErr := r.InsertFirebaseUser(ctx, params, alias)
		if insertErr == nil {
			return user, nil
		}
		if !errors.Is(insertErr, ErrAliasAlreadyExists) {
			return nil, insertErr
		}
	}
	return nil, ErrAliasAlreadyExists
}

func (r *repository) InsertFirebaseUser(ctx context.Context, params FirebaseUserParams, alias string) (*AuthUser, error) {
	name := params.Name
	if name == "" {
		if params.Email != "" {
			name = strings.Split(params.Email, "@")[0]
		} else {
			name = params.FirebaseUID
		}
	}

	row, err := r.queries.InsertFirebaseUser(ctx, database.InsertFirebaseUserParams{
		Name:             name,
		Alias:            alias,
		Column3:          params.Avatar,
		FirebaseUid:      params.FirebaseUID,
		Column5:          params.Email,
		EmailVerified:    params.EmailVerified,
		FirebaseProvider: params.Provider,
	})
	if err != nil {
		return nil, mapUniqueViolation(err)
	}
	return authUserFromDatabaseUser(row), nil
}
