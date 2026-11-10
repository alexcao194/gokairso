package whatsnew

import (
	"context"

	"github.com/alexcao194/kairso/pkg/pagination"
)

type Service interface {
	ListReleases(ctx context.Context, params pagination.Params) (pagination.Response[WhatsNewReleaseResponse], error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) *service {
	return &service{repo: repo}
}

func (s *service) ListReleases(ctx context.Context, params pagination.Params) (pagination.Response[WhatsNewReleaseResponse], error) {
	return s.repo.ListReleases(ctx, params)
}
