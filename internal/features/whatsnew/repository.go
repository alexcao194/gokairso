package whatsnew

import (
	"context"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	ListReleases(ctx context.Context, params pagination.Params) (pagination.Response[WhatsNewReleaseResponse], error)
}

type repository struct {
	queries *database.Queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{queries: database.New(pool)}
}

func (r *repository) ListReleases(ctx context.Context, params pagination.Params) (pagination.Response[WhatsNewReleaseResponse], error) {
	var empty pagination.Response[WhatsNewReleaseResponse]

	releaseRows, err := r.queries.ListWhatsNewReleases(ctx, database.ListWhatsNewReleasesParams{
		Limit:  int32(params.LimitWithMore()),
		Offset: int32(params.Offset()),
	})
	if err != nil {
		return empty, err
	}

	releaseIDs := make([]pgtype.UUID, 0, len(releaseRows))
	for _, release := range releaseRows {
		releaseIDs = append(releaseIDs, release.ID)
	}

	notesByRelease := make(map[pgtype.UUID][]WhatsNewNoteResponse, len(releaseIDs))
	if len(releaseIDs) > 0 {
		noteRows, err := r.queries.ListWhatsNewReleaseNotesByReleaseIDs(ctx, releaseIDs)
		if err != nil {
			return empty, err
		}
		for _, note := range noteRows {
			notesByRelease[note.ReleaseID] = append(notesByRelease[note.ReleaseID], whatsNewNoteResponseFromDatabaseNote(note))
		}
	}

	items := make([]WhatsNewReleaseResponse, 0, len(releaseRows))
	for _, release := range releaseRows {
		items = append(items, whatsNewReleaseResponseFromDatabaseRelease(release, notesByRelease[release.ID]))
	}

	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}
