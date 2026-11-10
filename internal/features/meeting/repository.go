package meeting

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pagination"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetCallerRole(ctx context.Context, workspaceID, callerID string) (string, error)
	ProjectBelongsToWorkspace(ctx context.Context, projectID, workspaceID string) (bool, error)
	Create(ctx context.Context, workspaceID, callerID string, req CreateMeetingMinutesRequest) (*MeetingMinutesResponse, error)
	Get(ctx context.Context, id string) (*MeetingMinutesResponse, error)
	List(ctx context.Context, workspaceID string, filter ListMeetingMinutesFilter, params pagination.Params) (pagination.Response[MeetingMinutesResponse], error)
	Update(ctx context.Context, id string, req UpdateMeetingMinutesRequest) (*MeetingMinutesResponse, error)
	Delete(ctx context.Context, id string) error
}

type repository struct {
	queries *database.Queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{queries: database.New(pool)}
}

func (r *repository) GetCallerRole(ctx context.Context, workspaceID, callerID string) (string, error) {
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
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	s := string(role)
	return s, nil
}

func (r *repository) ProjectBelongsToWorkspace(ctx context.Context, projectID, workspaceID string) (bool, error) {
	pid, err := pgutil.ParseUUID(projectID)
	if err != nil {
		return false, nil
	}
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return false, err
	}
	return r.queries.ProjectBelongsToWorkspace(ctx, database.ProjectBelongsToWorkspaceParams{ProjectID: pid, WorkspaceID: wid})
}

func (r *repository) Create(ctx context.Context, workspaceID, callerID string, req CreateMeetingMinutesRequest) (*MeetingMinutesResponse, error) {
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	uid, err := pgutil.ParseUUID(callerID)
	if err != nil {
		return nil, err
	}
	var pid pgtype.UUID
	if req.ProjectID != "" {
		if pid, err = pgutil.ParseUUID(req.ProjectID); err != nil {
			return nil, err
		}
	}
	date, err := parseDate(req.MeetingDate)
	if err != nil {
		return nil, err
	}
	items, err := json.Marshal(req.Items)
	if err != nil {
		return nil, err
	}
	notes, err := json.Marshal(req.SectionNotes)
	if err != nil {
		return nil, err
	}
	attendees, err := json.Marshal(req.Attendees)
	if err != nil {
		return nil, err
	}
	recorders, err := json.Marshal(req.Recorders)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateMeetingMinutes(ctx, database.CreateMeetingMinutesParams{
		WorkspaceID:  wid,
		ProjectID:    pid,
		Title:        req.Title,
		MeetingDate:  date,
		Location:     req.Location,
		Attendees:    attendees,
		Recorders:    recorders,
		Summary:      req.Summary,
		Items:        items,
		SectionNotes: notes,
		CreatedByID:  uid,
	})
	if err != nil {
		return nil, err
	}
	resp := meetingResponseFromCreateRow(row)
	return &resp, nil
}

func (r *repository) Get(ctx context.Context, id string) (*MeetingMinutesResponse, error) {
	mid, err := pgutil.ParseUUID(id)
	if err != nil {
		return nil, ErrNotFound
	}
	row, err := r.queries.GetMeetingMinutes(ctx, mid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	resp := meetingResponseFromRow(row)
	return &resp, nil
}

func (r *repository) List(ctx context.Context, workspaceID string, filter ListMeetingMinutesFilter, params pagination.Params) (pagination.Response[MeetingMinutesResponse], error) {
	var empty pagination.Response[MeetingMinutesResponse]
	wid, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return empty, err
	}
	var pid pgtype.UUID
	if filter.ProjectID != "" {
		if pid, err = pgutil.ParseUUID(filter.ProjectID); err != nil {
			return empty, err
		}
	}
	q := pgtype.Text{}
	if filter.Query != "" {
		q = pgtype.Text{String: filter.Query, Valid: true}
	}
	rows, err := r.queries.ListMeetingMinutes(ctx, database.ListMeetingMinutesParams{
		WorkspaceID: wid,
		ProjectID:   pid,
		Q:           q,
		RowLimit:    int32(params.LimitWithMore()),
		RowOffset:   int32(params.Offset()),
	})
	if err != nil {
		return empty, err
	}
	items := make([]MeetingMinutesResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, meetingResponseFromListRow(row))
	}
	hasMore := pagination.Process(&items, params.Limit)
	return pagination.NewResponse(items, hasMore, params.Page, params.Limit), nil
}

func (r *repository) Update(ctx context.Context, id string, req UpdateMeetingMinutesRequest) (*MeetingMinutesResponse, error) {
	mid, err := pgutil.ParseUUID(id)
	if err != nil {
		return nil, ErrNotFound
	}
	params := database.UpdateMeetingMinutesParams{
		ID:       mid,
		Title:    textPtr(req.Title),
		Location: textPtr(req.Location),
		Summary:  textPtr(req.Summary),
	}
	if req.MeetingDate != nil {
		if params.MeetingDate, err = parseDate(*req.MeetingDate); err != nil {
			return nil, err
		}
	}
	if req.Items != nil {
		if params.Items, err = json.Marshal(*req.Items); err != nil {
			return nil, err
		}
	}
	if req.Attendees != nil {
		if params.Attendees, err = json.Marshal(*req.Attendees); err != nil {
			return nil, err
		}
	}
	if req.Recorders != nil {
		if params.Recorders, err = json.Marshal(*req.Recorders); err != nil {
			return nil, err
		}
	}
	if req.SectionNotes != nil {
		if params.SectionNotes, err = json.Marshal(*req.SectionNotes); err != nil {
			return nil, err
		}
	}
	row, err := r.queries.UpdateMeetingMinutes(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	resp := meetingResponseFromUpdateRow(row)
	return &resp, nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	mid, err := pgutil.ParseUUID(id)
	if err != nil {
		return ErrNotFound
	}
	return r.queries.DeleteMeetingMinutes(ctx, mid)
}
