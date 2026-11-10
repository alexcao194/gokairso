package notification

import (
	"context"
	"encoding/json"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ExecTx(ctx context.Context, fn func(Repository) error) error
	ListNotifications(ctx context.Context, workspaceID string, userID string, params pagination.Params) (pagination.Response[NotificationResponse], error)
	ReadNotifications(ctx context.Context, workspaceID string, userID string, ids []string) error
	CountUnreadNotifications(ctx context.Context, workspaceID string, userID string) (int64, error)
	CreateNotification(ctx context.Context, event NotificationEvent) (*NotificationResponse, error)
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

func (r *repository) ListNotifications(ctx context.Context, workspaceID string, userID string, params pagination.Params) (pagination.Response[NotificationResponse], error) {
	var empty pagination.Response[NotificationResponse]
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return empty, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return empty, err
	}
	rows, err := r.queries.ListNotifications(ctx, database.ListNotificationsParams{
		WorkspaceID: wid,
		UserID:      uid,
		Limit:       int32(params.LimitWithMore()),
		Offset:      int32(params.Offset()),
	})
	if err != nil {
		return empty, err
	}
	items := make([]NotificationResponse, 0)
	for _, row := range rows {
		item, err := notificationResponseFromDatabaseNotification(row)
		if err != nil {
			return empty, err
		}
		items = append(items, item)
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) ReadNotifications(ctx context.Context, workspaceID string, userID string, ids []string) error {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return r.queries.ReadAllNotifications(ctx, database.ReadAllNotificationsParams{
			WorkspaceID: wid,
			UserID:      uid,
		})
	}
	parsed := make([]pgtype.UUID, 0, len(ids))
	for _, id := range ids {
		value, err := pgutil.ParseUUID(id)
		if err != nil {
			return err
		}
		parsed = append(parsed, value)
	}
	return r.queries.ReadNotificationsByIDs(ctx, database.ReadNotificationsByIDsParams{WorkspaceID: wid, UserID: uid, Column3: parsed})
}

func (r *repository) CountUnreadNotifications(ctx context.Context, workspaceID string, userID string) (int64, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return 0, err
	}
	uid, err := pgutil.ParseUUID(userID)
	if err != nil {
		return 0, err
	}
	return r.queries.CountUnreadNotifications(ctx, database.CountUnreadNotificationsParams{
		WorkspaceID: wid,
		UserID:      uid,
	})
}

func (r *repository) CreateNotification(ctx context.Context, event NotificationEvent) (*NotificationResponse, error) {
	wid, err := pgutil.ParseUUID(event.WorkspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(event.UserID)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(event.Data)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateNotification(ctx, database.CreateNotificationParams{
		WorkspaceID: wid,
		UserID:      uid,
		Title:       event.Title,
		Body:        event.Body,
		Column5:     payload,
	})
	if err != nil {
		return nil, err
	}
	resp, err := notificationResponseFromDatabaseNotification(row)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}
