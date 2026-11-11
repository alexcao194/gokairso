package user

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserResponse struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt" swaggertype:"string" format:"date-time"`
	UpdatedAt     time.Time `json:"updatedAt" swaggertype:"string" format:"date-time"`
	Name          string    `json:"name"`
	Alias         string    `json:"alias"`
	Avatar        string    `json:"avatar,omitempty"`
	Email         string    `json:"email,omitempty"`
	EmailVerified bool      `json:"emailVerified"`
	Birthday      string    `json:"birthday,omitempty" swaggertype:"string" format:"date"`
	Bio           string    `json:"bio,omitempty"`
	Phone         string    `json:"phone,omitempty"`
	Nickname      string    `json:"nickname,omitempty"`
	Locale        string    `json:"locale,omitempty"`
}

type UserSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Alias  string `json:"alias"`
	Avatar string `json:"avatar,omitempty"`
}

type UpdateMeRequest struct {
	Name     *string `json:"name,omitempty"`
	Alias    *string `json:"alias,omitempty"`
	Avatar   *string `json:"avatar,omitempty"`
	Birthday *string `json:"birthday,omitempty"`
	Bio      *string `json:"bio,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Nickname *string `json:"nickname,omitempty"`
	Locale   *string `json:"locale,omitempty"`
}

type userRow struct {
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
	Birthday      pgtype.Date
	Bio           pgtype.Text
	Phone         pgtype.Text
	Nickname      pgtype.Text
	Locale        string
}

func dateString(value pgtype.Date) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.DateOnly)
}

func userResponseFromRow(row userRow) UserResponse {
	userID := ""
	if row.ID.Valid {
		userID = uuid.UUID(row.ID.Bytes).String()
	}
	return UserResponse{
		ID:            userID,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
		Name:          row.Name,
		Alias:         row.Alias,
		Avatar:        textString(row.Avatar),
		Email:         textString(row.Email),
		EmailVerified: row.EmailVerified,
		Birthday:      dateString(row.Birthday),
		Bio:           textString(row.Bio),
		Phone:         textString(row.Phone),
		Nickname:      textString(row.Nickname),
		Locale:        row.Locale,
	}
}

func userResponseFromDatabaseUser(row database.User) UserResponse {
	return userResponseFromRow(userRow{
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
		Birthday:      row.Birthday,
		Bio:           row.Bio,
		Phone:         row.Phone,
		Nickname:      row.Nickname,
		Locale:        row.Locale,
	})
}

func userResponsePtrFromDatabaseUser(row database.User) *UserResponse {
	resp := userResponseFromDatabaseUser(row)
	return &resp
}

func pgtypeTextFromStringPtr(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func pgtypeDateFromStringPtr(value *string) (pgtype.Date, error) {
	if value == nil {
		return pgtype.Date{}, nil
	}
	if *value == "" {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse(time.DateOnly, *value)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

type User struct {
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
	Birthday      *time.Time `swaggertype:"string" format:"date-time"`
	Bio           string
	Phone         string
	Nickname      string
}

func textString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func normalizeUpdateRequest(req *UpdateMeRequest) error {
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			return fmt.Errorf("Tên không được để trống")
		}
		req.Name = &value
	}
	if req.Alias != nil {
		value := strings.TrimSpace(*req.Alias)
		if value == "" {
			return fmt.Errorf("alias không được để trống")
		}
		req.Alias = &value
	}
	if req.Avatar != nil {
		value := strings.TrimSpace(*req.Avatar)
		if parsed, err := url.Parse(value); err == nil {
			parsed.RawQuery = ""
			value = parsed.String()
		}
		req.Avatar = &value
	}
	if req.Birthday != nil {
		value := strings.TrimSpace(*req.Birthday)
		if value != "" {
			_, err := time.Parse(time.DateOnly, value)
			if err != nil {
				return fmt.Errorf("Ngày sinh phải theo định dạng YYYY-MM-DD")
			}
		}
		req.Birthday = &value
	}
	if req.Bio != nil {
		value := strings.TrimSpace(*req.Bio)
		req.Bio = &value
	}
	if req.Phone != nil {
		value := strings.TrimSpace(*req.Phone)
		req.Phone = &value
	}
	if req.Nickname != nil {
		value := strings.TrimSpace(*req.Nickname)
		req.Nickname = &value
	}
	return nil
}
