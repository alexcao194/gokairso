package webhook

import (
	"context"
	"errors"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/internal/features/workspace"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetCallerRole(ctx context.Context, workspaceID, userID pgtype.UUID) (workspace.WorkspaceRole, error)
	List(ctx context.Context, workspaceID pgtype.UUID) ([]database.Webhook, error)
	ListEnabled(ctx context.Context, workspaceID pgtype.UUID) ([]database.ListEnabledWebhooksRow, error)
	Count(ctx context.Context, workspaceID pgtype.UUID) (int64, error)
	Get(ctx context.Context, id, workspaceID pgtype.UUID) (database.Webhook, bool, error)
	Create(ctx context.Context, arg database.CreateWebhookParams) (database.Webhook, error)
	Update(ctx context.Context, arg database.UpdateWebhookParams) (database.Webhook, bool, error)
	Delete(ctx context.Context, id, workspaceID pgtype.UUID) (int64, error)
	RecordSuccess(ctx context.Context, id pgtype.UUID, status int32) error
	RecordFailure(ctx context.Context, id pgtype.UUID, status int32, errText string, disableAfter int32) (stillEnabled bool, err error)
	InsertDelivery(ctx context.Context, arg database.InsertWebhookDeliveryParams) error
	ListDeliveries(ctx context.Context, webhookID pgtype.UUID) ([]database.ListWebhookDeliveriesRow, error)
}

type repository struct {
	queries *database.Queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{queries: database.New(pool)}
}

func found[T any](v T, err error) (T, bool, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		var zero T
		return zero, false, nil
	}
	if err != nil {
		var zero T
		return zero, false, err
	}
	return v, true, nil
}

func (r *repository) GetCallerRole(ctx context.Context, workspaceID, userID pgtype.UUID) (workspace.WorkspaceRole, error) {
	role, ok, err := found(r.queries.GetCallerWorkspaceRole(ctx, database.GetCallerWorkspaceRoleParams{WorkspaceID: workspaceID, UserID: userID}))
	if err != nil || !ok {
		return "", err
	}
	return workspace.WorkspaceRole(string(role)), nil
}

func (r *repository) List(ctx context.Context, workspaceID pgtype.UUID) ([]database.Webhook, error) {
	return r.queries.ListWebhooks(ctx, workspaceID)
}

func (r *repository) ListEnabled(ctx context.Context, workspaceID pgtype.UUID) ([]database.ListEnabledWebhooksRow, error) {
	return r.queries.ListEnabledWebhooks(ctx, workspaceID)
}

func (r *repository) Count(ctx context.Context, workspaceID pgtype.UUID) (int64, error) {
	return r.queries.CountWebhooks(ctx, workspaceID)
}

func (r *repository) Get(ctx context.Context, id, workspaceID pgtype.UUID) (database.Webhook, bool, error) {
	return found(r.queries.GetWebhook(ctx, database.GetWebhookParams{ID: id, WorkspaceID: workspaceID}))
}

func (r *repository) Create(ctx context.Context, arg database.CreateWebhookParams) (database.Webhook, error) {
	return r.queries.CreateWebhook(ctx, arg)
}

func (r *repository) Update(ctx context.Context, arg database.UpdateWebhookParams) (database.Webhook, bool, error) {
	return found(r.queries.UpdateWebhook(ctx, arg))
}

func (r *repository) Delete(ctx context.Context, id, workspaceID pgtype.UUID) (int64, error) {
	return r.queries.DeleteWebhook(ctx, database.DeleteWebhookParams{ID: id, WorkspaceID: workspaceID})
}

func (r *repository) RecordSuccess(ctx context.Context, id pgtype.UUID, status int32) error {
	return r.queries.RecordWebhookSuccess(ctx, database.RecordWebhookSuccessParams{ID: id, LastStatus: pgtype.Int4{Int32: status, Valid: true}})
}

func (r *repository) RecordFailure(ctx context.Context, id pgtype.UUID, status int32, errText string, disableAfter int32) (bool, error) {
	return r.queries.RecordWebhookFailure(ctx, database.RecordWebhookFailureParams{
		ID: id, LastStatus: pgtype.Int4{Int32: status, Valid: true}, LastError: errText, DisableAfter: disableAfter,
	})
}

func (r *repository) InsertDelivery(ctx context.Context, arg database.InsertWebhookDeliveryParams) error {
	if err := r.queries.InsertWebhookDelivery(ctx, arg); err != nil {
		return err
	}
	return r.queries.PruneWebhookDeliveries(ctx, arg.WebhookID)
}

func (r *repository) ListDeliveries(ctx context.Context, webhookID pgtype.UUID) ([]database.ListWebhookDeliveriesRow, error) {
	return r.queries.ListWebhookDeliveries(ctx, webhookID)
}
