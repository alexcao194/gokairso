package whatsnew

import (
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
)

type WhatsNewNoteType string

const (
	WhatsNewNoteTypeNewFeature  WhatsNewNoteType = "new_feature"
	WhatsNewNoteTypeImprovement WhatsNewNoteType = "improvement"
	WhatsNewNoteTypeBugFix      WhatsNewNoteType = "bug_fix"
)

type WhatsNewNoteResponse struct {
	ID      string           `json:"id"`
	Type    WhatsNewNoteType `json:"type"`
	Content string           `json:"content"`
}

type WhatsNewReleaseResponse struct {
	ID          string                 `json:"id"`
	Version     string                 `json:"version"`
	PublishedAt time.Time              `json:"publishedAt" swaggertype:"string" format:"date-time"`
	Notes       []WhatsNewNoteResponse `json:"notes"`
}

func whatsNewNoteResponseFromDatabaseNote(row database.ListWhatsNewReleaseNotesByReleaseIDsRow) WhatsNewNoteResponse {
	return WhatsNewNoteResponse{
		ID:      pgutil.UUIDToString(row.ID),
		Type:    WhatsNewNoteType(row.Type),
		Content: row.Content,
	}
}

func whatsNewReleaseResponseFromDatabaseRelease(row database.ListWhatsNewReleasesRow, notes []WhatsNewNoteResponse) WhatsNewReleaseResponse {
	if notes == nil {
		notes = []WhatsNewNoteResponse{}
	}
	return WhatsNewReleaseResponse{
		ID:          pgutil.UUIDToString(row.ID),
		Version:     row.Version,
		PublishedAt: row.PublishedAt.Time,
		Notes:       notes,
	}
}
