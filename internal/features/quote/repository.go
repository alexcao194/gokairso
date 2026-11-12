package quote

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
	ExecTx(ctx context.Context, fn func(Repository) error) error
	GetCallerRole(ctx context.Context, workspaceID, userID pgtype.UUID) (workspace.WorkspaceRole, error)

	ListCustomers(ctx context.Context, workspaceID pgtype.UUID) ([]database.Customer, error)
	GetCustomer(ctx context.Context, id, workspaceID pgtype.UUID) (database.Customer, bool, error)
	CreateCustomer(ctx context.Context, arg database.CreateCustomerParams) (database.Customer, error)
	UpdateCustomer(ctx context.Context, arg database.UpdateCustomerParams) (database.Customer, bool, error)
	DeleteCustomer(ctx context.Context, id, workspaceID pgtype.UUID) (int64, error)

	GetSettings(ctx context.Context, workspaceID pgtype.UUID) (database.QuoteSetting, bool, error)
	UpsertSettings(ctx context.Context, arg database.UpsertQuoteSettingsParams) (database.QuoteSetting, error)
	NextNumber(ctx context.Context, workspaceID pgtype.UUID, year int32) (int32, error)

	GetModuleContext(ctx context.Context, moduleID pgtype.UUID) (database.GetModuleContextRow, bool, error)
	CreateQuote(ctx context.Context, arg database.CreateQuoteParams) (pgtype.UUID, error)
	GetQuote(ctx context.Context, id, workspaceID pgtype.UUID) (database.GetQuoteRow, bool, error)
	ListQuotes(ctx context.Context, arg database.ListQuotesParams) ([]database.ListQuotesRow, error)
	UpdateQuoteDraft(ctx context.Context, arg database.UpdateQuoteDraftParams) error
	MarkQuoteSent(ctx context.Context, arg database.MarkQuoteSentParams) error
	SetQuoteStatus(ctx context.Context, arg database.SetQuoteStatusParams) error
	DeleteQuote(ctx context.Context, id, workspaceID pgtype.UUID) (int64, error)

	ListItems(ctx context.Context, quoteID pgtype.UUID) ([]database.QuoteItem, error)
	ReplaceItems(ctx context.Context, quoteID pgtype.UUID, items []database.InsertQuoteItemParams) error

	InsertEvent(ctx context.Context, quoteID pgtype.UUID, eventType string, actorID pgtype.UUID, note string) error
	ListEvents(ctx context.Context, quoteID pgtype.UUID) ([]database.ListQuoteEventsRow, error)
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
	if err := fn(&repository{queries: r.queries.WithTx(tx), db: r.db}); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
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

func (r *repository) ListCustomers(ctx context.Context, workspaceID pgtype.UUID) ([]database.Customer, error) {
	return r.queries.ListCustomers(ctx, workspaceID)
}

func (r *repository) GetCustomer(ctx context.Context, id, workspaceID pgtype.UUID) (database.Customer, bool, error) {
	return found(r.queries.GetCustomer(ctx, database.GetCustomerParams{ID: id, WorkspaceID: workspaceID}))
}

func (r *repository) CreateCustomer(ctx context.Context, arg database.CreateCustomerParams) (database.Customer, error) {
	return r.queries.CreateCustomer(ctx, arg)
}

func (r *repository) UpdateCustomer(ctx context.Context, arg database.UpdateCustomerParams) (database.Customer, bool, error) {
	return found(r.queries.UpdateCustomer(ctx, arg))
}

func (r *repository) DeleteCustomer(ctx context.Context, id, workspaceID pgtype.UUID) (int64, error) {
	return r.queries.DeleteCustomer(ctx, database.DeleteCustomerParams{ID: id, WorkspaceID: workspaceID})
}

func (r *repository) GetSettings(ctx context.Context, workspaceID pgtype.UUID) (database.QuoteSetting, bool, error) {
	return found(r.queries.GetQuoteSettings(ctx, workspaceID))
}

func (r *repository) UpsertSettings(ctx context.Context, arg database.UpsertQuoteSettingsParams) (database.QuoteSetting, error) {
	return r.queries.UpsertQuoteSettings(ctx, arg)
}

func (r *repository) NextNumber(ctx context.Context, workspaceID pgtype.UUID, year int32) (int32, error) {
	return r.queries.NextQuoteNumber(ctx, database.NextQuoteNumberParams{WorkspaceID: workspaceID, Year: year})
}

func (r *repository) GetModuleContext(ctx context.Context, moduleID pgtype.UUID) (database.GetModuleContextRow, bool, error) {
	return found(r.queries.GetModuleContext(ctx, moduleID))
}

func (r *repository) CreateQuote(ctx context.Context, arg database.CreateQuoteParams) (pgtype.UUID, error) {
	return r.queries.CreateQuote(ctx, arg)
}

func (r *repository) GetQuote(ctx context.Context, id, workspaceID pgtype.UUID) (database.GetQuoteRow, bool, error) {
	return found(r.queries.GetQuote(ctx, database.GetQuoteParams{ID: id, WorkspaceID: workspaceID}))
}

func (r *repository) ListQuotes(ctx context.Context, arg database.ListQuotesParams) ([]database.ListQuotesRow, error) {
	return r.queries.ListQuotes(ctx, arg)
}

func (r *repository) UpdateQuoteDraft(ctx context.Context, arg database.UpdateQuoteDraftParams) error {
	return r.queries.UpdateQuoteDraft(ctx, arg)
}

func (r *repository) MarkQuoteSent(ctx context.Context, arg database.MarkQuoteSentParams) error {
	return r.queries.MarkQuoteSent(ctx, arg)
}

func (r *repository) SetQuoteStatus(ctx context.Context, arg database.SetQuoteStatusParams) error {
	return r.queries.SetQuoteStatus(ctx, arg)
}

func (r *repository) DeleteQuote(ctx context.Context, id, workspaceID pgtype.UUID) (int64, error) {
	return r.queries.DeleteQuote(ctx, database.DeleteQuoteParams{ID: id, WorkspaceID: workspaceID})
}

func (r *repository) ListItems(ctx context.Context, quoteID pgtype.UUID) ([]database.QuoteItem, error) {
	return r.queries.ListQuoteItems(ctx, quoteID)
}

func (r *repository) ReplaceItems(ctx context.Context, quoteID pgtype.UUID, items []database.InsertQuoteItemParams) error {
	if err := r.queries.DeleteQuoteItems(ctx, quoteID); err != nil {
		return err
	}
	for _, it := range items {
		it.QuoteID = quoteID
		if err := r.queries.InsertQuoteItem(ctx, it); err != nil {
			return err
		}
	}
	return nil
}

func (r *repository) InsertEvent(ctx context.Context, quoteID pgtype.UUID, eventType string, actorID pgtype.UUID, note string) error {
	return r.queries.InsertQuoteEvent(ctx, database.InsertQuoteEventParams{QuoteID: quoteID, Type: eventType, ActorID: actorID, Note: note})
}

func (r *repository) ListEvents(ctx context.Context, quoteID pgtype.UUID) ([]database.ListQuoteEventsRow, error) {
	return r.queries.ListQuoteEvents(ctx, quoteID)
}
