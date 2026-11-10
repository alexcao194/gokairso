package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alexcao194/kairso/pkg/httpx"
)

func GenerateSignature(secret, method, path, timestamp string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("signature: secret must not be empty")
	}

	payload := fmt.Sprintf("%s\n%s\n%s", timestamp, method, path)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func VerifySignature(token, secret, method, path, timestamp string) (bool, error) {
	if secret == "" {
		return false, fmt.Errorf("signature: secret is not configured")
	}
	if token == "" {
		return false, nil
	}

	expected, err := GenerateSignature(secret, method, path, timestamp)
	if err != nil {
		return false, err
	}
	return hmac.Equal([]byte(token), []byte(expected)), nil
}

type AppTokenMiddleware struct {
	secret string
}

func NewAppTokenMiddleware(secret string) *AppTokenMiddleware {
	return &AppTokenMiddleware{secret: secret}
}

func (m *AppTokenMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/firebase/login" {
			next.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get("X-App-Token")
		if token == "" {
			httpx.Error(w, http.StatusForbidden, "Thiếu header X-App-Token")
			return
		}

		now := time.Now().Truncate(time.Minute)
		timesToTest := []time.Time{
			now,
			now.Add(-time.Minute),
			now.Add(time.Minute),
		}

		path := r.URL.EscapedPath()

		valid := false
		for _, t := range timesToTest {
			timestampStr := strconv.FormatInt(t.UnixMilli(), 10)
			matched, err := VerifySignature(token, m.secret, r.Method, path, timestampStr)
			if err == nil && matched {
				valid = true
				break
			}
		}

		if !valid {
			httpx.Error(w, http.StatusForbidden, "X-App-Token không hợp lệ")
			return
		}

		next.ServeHTTP(w, r)
	})
}
