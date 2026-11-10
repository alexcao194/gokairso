package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/alexcao194/kairso/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("Token không hợp lệ")
	ErrExpiredToken = errors.New("Token đã hết hạn")
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type TokenService interface {
	GenerateToken(cfg *config.AppConfig, userID string) (string, string, error)
	ValidateToken(tokenStr string) (*Claims, error)
}
type jwtService struct {
	secret []byte
}

func NewJWTService(secret string) TokenService {
	return &jwtService{
		secret: []byte(secret),
	}
}

type Claims struct {
	UserID    string `json:"userId"`
	TokenType string `json:"tokenType"`
	jwt.RegisteredClaims
}

func (s *jwtService) GenerateToken(cfg *config.AppConfig, userID string) (string, string, error) {
	if userID == "" {
		return "", "", errors.New("userId không được để trống")
	}

	secret := s.secret
	if cfg.JWTSecret != "" {
		secret = []byte(cfg.JWTSecret)
	}

	accessClaims := &Claims{
		UserID:    userID,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	accessTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessToken, err := accessTokenObj.SignedString(secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	refreshClaims := &Claims{
		UserID:    userID,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	refreshTokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := refreshTokenObj.SignedString(secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}
func (s *jwtService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}
