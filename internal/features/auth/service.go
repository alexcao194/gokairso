package auth

import (
	"context"
	"errors"

	"github.com/alexcao194/kairso/internal/config"
	"github.com/alexcao194/kairso/pkg/jwt"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidFirebaseToken  = errors.New("Firebase token không hợp lệ")
	ErrFirebaseNotConfigured = errors.New("Máy chủ chưa cấu hình xác thực Firebase")
)

type FirebaseVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*FirebaseToken, error)
}

type Service interface {
	LoginWithFirebase(ctx context.Context, idToken string) (*LoginResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error)
}

type service struct {
	repo     Repository
	verifier FirebaseVerifier
	jwtSvc   jwt.TokenService
	cfg      *config.AppConfig
}

func NewService(repo Repository, verifier FirebaseVerifier, jwtSvc jwt.TokenService, cfg *config.AppConfig) Service {
	return &service{repo: repo, verifier: verifier, jwtSvc: jwtSvc, cfg: cfg}
}

func (s *service) LoginWithFirebase(ctx context.Context, idToken string) (*LoginResponse, error) {
	if s.verifier == nil {
		return nil, ErrFirebaseNotConfigured
	}
	token, err := s.verifier.VerifyIDToken(ctx, idToken)
	if err != nil || token == nil || token.UID == "" {
		return nil, ErrInvalidFirebaseToken
	}
	params := firebaseParamsFromToken(token)
	user, err := s.upsertFirebaseUser(ctx, params)
	if err != nil {
		return nil, err
	}
	accessToken, refreshToken, err := s.jwtSvc.GenerateToken(s.cfg, user.ID)
	if err != nil {
		return nil, err
	}
	resp := mapToLoginResponse(accessToken, refreshToken, user)
	return &resp, nil
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	claims, err := s.jwtSvc.ValidateToken(refreshToken)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != jwt.TokenTypeRefresh {
		return nil, jwt.ErrInvalidToken
	}
	newAccess, newRefresh, err := s.jwtSvc.GenerateToken(s.cfg, claims.UserID)
	if err != nil {
		return nil, err
	}
	resp := mapToTokenResponse(newAccess, newRefresh)
	return &resp, nil
}

func (s *service) upsertFirebaseUser(ctx context.Context, params FirebaseUserParams) (*AuthUser, error) {
	params = normalizeFirebaseUserParams(params)
	if params.FirebaseUID == "" {
		return nil, ErrInvalidFirebaseToken
	}

	updated, err := s.repo.UpdateFirebaseUser(ctx, params)
	if err == nil {
		return updated, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	for _, alias := range aliasCandidates(params) {
		user, insertErr := s.repo.InsertFirebaseUser(ctx, params, alias)
		if insertErr == nil {
			return user, nil
		}
		if !errors.Is(insertErr, ErrAliasAlreadyExists) {
			return nil, insertErr
		}
	}
	return nil, ErrAliasAlreadyExists
}
