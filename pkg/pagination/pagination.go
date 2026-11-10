package pagination

import (
	"net/http"
	"strconv"
)

type Params struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.Limit
}

func (p Params) LimitWithMore() int {
	return p.Limit + 1
}

type StatsParams struct {
	Params
	Month string `json:"month"`
}

func ParseStats(r *http.Request, defaultLimit, maxLimit int) StatsParams {
	return StatsParams{
		Params: Parse(r, defaultLimit, maxLimit),
		Month:  r.URL.Query().Get("month"),
	}
}

func Parse(r *http.Request, defaultLimit, maxLimit int) Params {
	query := r.URL.Query()

	page := 1
	if p, err := strconv.Atoi(query.Get("page")); err == nil && p > 0 {
		page = p
	}

	limit := defaultLimit
	if l, err := strconv.Atoi(query.Get("limit")); err == nil && l > 0 {
		if maxLimit > 0 && l > maxLimit {
			limit = maxLimit
		} else {
			limit = l
		}
	}

	return Params{
		Page:  page,
		Limit: limit,
	}
}

func Process[T any](items *[]T, limit int) bool {
	hasMore := false
	if len(*items) > limit {
		hasMore = true
		*items = (*items)[:limit]
	}
	return hasMore
}

type Response[T any] struct {
	Items   []T  `json:"items"`
	HasMore bool `json:"hasMore"`
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
}

func NewResponse[T any](items []T, hasMore bool, page, limit int) Response[T] {
	return Response[T]{
		Items:   items,
		HasMore: hasMore,
		Page:    page,
		Limit:   limit,
	}
}
