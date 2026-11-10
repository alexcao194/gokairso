package middleware

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	maxLoggedRequestBodyBytes  = 4096
	maxLoggedResponseBodyBytes = 4096
)

type ErrorLogMiddleware struct{}

func NewErrorLogMiddleware() *ErrorLogMiddleware {
	return &ErrorLogMiddleware{}
}

func (m *ErrorLogMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestBody, requestBodyTruncated := readRequestBody(r)
		recorder := &errorLogResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			if !recorder.wroteHeader {
				recorder.status = http.StatusInternalServerError
			}
			recorder.appendBody(fmt.Sprintf("panic: %v", recovered))
			m.write(r, recorder, requestBody, requestBodyTruncated, time.Since(start))
			panic(recovered)
		}()

		next.ServeHTTP(recorder, r)

		if recorder.status < http.StatusBadRequest {
			return
		}

		m.write(r, recorder, requestBody, requestBodyTruncated, time.Since(start))
	})
}

func (m *ErrorLogMiddleware) write(r *http.Request, recorder *errorLogResponseWriter, requestBody string, requestBodyTruncated bool, duration time.Duration) {
	slog.Error(
		"API request failed",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"params", r.URL.Query(),
		"headers", r.Header,
		"status", recorder.status,
		"duration", duration,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
		"request_body", requestBody,
		"request_body_truncated", requestBodyTruncated,
		"response_body", strings.TrimSpace(recorder.body.String()),
	)
}

func readRequestBody(r *http.Request) (string, bool) {
	if r.Body == nil {
		return "", false
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		r.Body = io.NopCloser(bytes.NewReader(nil))
		return fmt.Sprintf("failed to read request body: %v", err), false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	if len(body) <= maxLoggedRequestBodyBytes {
		return string(body), false
	}
	return string(body[:maxLoggedRequestBodyBytes]), true
}

type errorLogResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        strings.Builder
}

func (w *errorLogResponseWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *errorLogResponseWriter) Write(data []byte) (int, error) {
	w.appendBody(string(data))
	return w.ResponseWriter.Write(data)
}

func (w *errorLogResponseWriter) appendBody(data string) {
	if w.body.Len() < maxLoggedResponseBodyBytes {
		remaining := maxLoggedResponseBodyBytes - w.body.Len()
		loggedData := data
		if len(loggedData) > remaining {
			loggedData = data[:remaining]
		}
		w.body.WriteString(loggedData)
	}
}

func (w *errorLogResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *errorLogResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
	}
	return hijacker.Hijack()
}
