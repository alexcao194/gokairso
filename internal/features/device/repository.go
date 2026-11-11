package device

import (
	"context"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	Upsert(ctx context.Context, userID, token, platform, locale string) error
	Delete(ctx context.Context, token string) error
	DeleteMany(ctx context.Context, tokens []string) error
	ListByUser(ctx context.Context, userID string) ([]DeviceToken, error)
}

type repository struct {
	queries *database.Queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{queries: database.New(pool)}
}

func (r *repository) Upsert(ctx context.Context, userID, token, platform, locale string) error {
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return err
	}
	return r.queries.UpsertDeviceToken(ctx, database.UpsertDeviceTokenParams{
		UserID:   uid,
		Token:    token,
		Platform: platform,
		Locale:   locale,
	})
}

func (r *repository) Delete(ctx context.Context, token string) error {
	return r.queries.DeleteDeviceToken(ctx, token)
}

func (r *repository) DeleteMany(ctx context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}
	return r.queries.DeleteDeviceTokens(ctx, tokens)
}

func (r *repository) ListByUser(ctx context.Context, userID string) ([]DeviceToken, error) {
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListDeviceTokensByUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	out := make([]DeviceToken, 0, len(rows))
	for _, row := range rows {
		out = append(out, DeviceToken{Token: row.Token, Platform: row.Platform, Locale: row.Locale})
	}
	return out, nil
}
