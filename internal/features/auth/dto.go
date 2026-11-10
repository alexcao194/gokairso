package auth

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/firebase"
	"github.com/alexcao194/kairso/pkg/pgutil"
)

type FirebaseLoginRequest struct {
	IDToken string `json:"idToken"`
}

var ErrInvalidAlias = errors.New("Alias phải gồm 3–32 ký tự: chữ thường, chữ số hoặc dấu gạch ngang")

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type LoginResponse struct {
	AccessToken  string           `json:"accessToken"`
	RefreshToken string           `json:"refreshToken"`
	User         AuthUserResponse `json:"user"`
}

type FirebaseToken struct {
	UID            string
	Email          string
	EmailVerified  bool
	Name           string
	Picture        string
	SignInProvider string
}

type FirebaseUserParams struct {
	FirebaseUID   string
	Email         string
	EmailVerified bool
	Name          string
	Avatar        string
	Provider      string
}

type AuthUser struct {
	ID            string
	CreatedAt     time.Time `swaggertype:"string" format:"date-time"`
	UpdatedAt     time.Time `swaggertype:"string" format:"date-time"`
	Name          string
	Alias         string
	Avatar        string
	FirebaseUID   string
	Email         string
	EmailVerified bool
	Provider      string
	LastSignInAt  *time.Time `swaggertype:"string" format:"date-time"`
}

type AuthUserResponse struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt     time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
	Name          string    `json:"name"`
	Alias         string    `json:"alias"`
	Avatar        string    `json:"avatar,omitempty"`
	Email         string    `json:"email,omitempty"`
	EmailVerified bool      `json:"emailVerified"`
}

type authUserRow struct {
	ID            pgtype.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Name          string
	Alias         string
	Avatar        pgtype.Text
	FirebaseUID   string
	Email         pgtype.Text
	EmailVerified bool
	Provider      string
	LastSignInAt  pgtype.Timestamptz
}

func mapToTokenResponse(accessToken, refreshToken string) TokenResponse {
	return TokenResponse{AccessToken: accessToken, RefreshToken: refreshToken}
}

func mapToLoginResponse(accessToken, refreshToken string, user *AuthUser) LoginResponse {
	return LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         AuthUserResponseFromModel(user),
	}
}

func AuthUserResponseFromModel(user *AuthUser) AuthUserResponse {
	if user == nil {
		return AuthUserResponse{}
	}
	return AuthUserResponse{
		ID:            user.ID,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
		Name:          user.Name,
		Alias:         user.Alias,
		Avatar:        user.Avatar,
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
	}
}

func authUserFromRow(row authUserRow) *AuthUser {
	userID := ""
	if row.ID.Valid {
		userID = uuid.UUID(row.ID.Bytes).String()
	}
	return &AuthUser{
		ID:            userID,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		Name:          row.Name,
		Alias:         row.Alias,
		Avatar:        textString(row.Avatar),
		FirebaseUID:   row.FirebaseUID,
		Email:         textString(row.Email),
		EmailVerified: row.EmailVerified,
		Provider:      row.Provider,
		LastSignInAt:  timestamptzPtr(row.LastSignInAt),
	}
}

func authUserFromDatabaseUser(row database.User) *AuthUser {
	return authUserFromRow(authUserRow{
		ID:            row.ID,
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
		Name:          row.Name,
		Alias:         row.Alias,
		Avatar:        row.Avatar,
		FirebaseUID:   row.FirebaseUid,
		Email:         row.Email,
		EmailVerified: row.EmailVerified,
		Provider:      row.FirebaseProvider,
		LastSignInAt:  row.LastSignInAt,
	})
}

func firebaseParamsFromToken(token *FirebaseToken) FirebaseUserParams {
	if token == nil {
		return FirebaseUserParams{}
	}
	provider := strings.TrimSpace(token.SignInProvider)
	if provider == "" {
		provider = "firebase"
	}
	return FirebaseUserParams{
		FirebaseUID:   strings.TrimSpace(token.UID),
		Email:         strings.TrimSpace(strings.ToLower(token.Email)),
		EmailVerified: token.EmailVerified,
		Name:          strings.TrimSpace(token.Name),
		Avatar:        strings.TrimSpace(token.Picture),
		Provider:      provider,
	}
}

func aliasBaseFromFirebaseToken(params FirebaseUserParams) string {
	base := ""
	if params.Email != "" {
		base = strings.Split(params.Email, "@")[0]
	}
	if base == "" {
		base = params.Name
	}
	if base == "" {
		base = params.FirebaseUID
	}
	return slugifyAlias(base)
}

func slugifyAlias(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	value = re.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	return value
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func timestamptzPtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func mapFirebaseToken(token *firebase.Token) *FirebaseToken {
	if token == nil {
		return nil
	}
	return &FirebaseToken{
		UID:            token.UID,
		Email:          token.Email,
		EmailVerified:  token.EmailVerified,
		Name:           token.Name,
		Picture:        token.Picture,
		SignInProvider: token.SignInProvider,
	}
}

func normalizeFirebaseUserParams(params FirebaseUserParams) FirebaseUserParams {
	params.FirebaseUID = strings.TrimSpace(params.FirebaseUID)
	params.Email = strings.ToLower(strings.TrimSpace(params.Email))
	params.Name = strings.TrimSpace(params.Name)
	params.Avatar = strings.TrimSpace(params.Avatar)
	params.Provider = strings.TrimSpace(params.Provider)
	if params.Provider == "" {
		params.Provider = "firebase"
	}
	return params
}

func aliasCandidates(params FirebaseUserParams) []string {
	base := normalizeGeneratedAlias(aliasBaseFromFirebaseToken(params))
	suffix := aliasSuffix(params.FirebaseUID)
	if suffix == "" {
		return []string{base}
	}
	prefix := strings.Trim(base, "-")
	maxPrefixLen := 32 - len(suffix) - 1
	if len(prefix) > maxPrefixLen {
		prefix = strings.Trim(prefix[:maxPrefixLen], "-")
	}
	if len(prefix) < 3 {
		prefix = "user"
	}
	return []string{base, normalizeGeneratedAlias(prefix + "-" + suffix)}
}

func normalizeGeneratedAlias(alias string) string {
	alias = strings.Trim(strings.ToLower(strings.TrimSpace(alias)), "-")
	if len(alias) > 32 {
		alias = strings.Trim(alias[:32], "-")
	}
	if len(alias) < 3 {
		alias = "user"
	}
	if len(alias) > 32 {
		alias = alias[:32]
	}
	return alias
}

func aliasSuffix(firebaseUID string) string {
	firebaseUID = strings.TrimSpace(firebaseUID)
	if firebaseUID == "" {
		return ""
	}
	sum := sha1.Sum([]byte(firebaseUID))
	return hex.EncodeToString(sum[:])[:8]
}

func mapUniqueViolation(err error) error {
	if pgutil.IsAliasConflict(err) {
		return ErrAliasAlreadyExists
	}
	return err
}
