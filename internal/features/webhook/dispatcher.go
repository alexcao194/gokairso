package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"syscall"
	"time"

	"github.com/alexcao194/kairso/internal/database"
	"github.com/alexcao194/kairso/pkg/pgutil"
	"github.com/alexcao194/kairso/pkg/realtime"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	eventBuffer    = 2048
	jobBuffer      = 4096
	workerCount    = 4
	maxAttempts    = 3
	requestTimeout = 8 * time.Second
	cacheTTL       = 30 * time.Second
	// disableAfter consecutive failed deliveries, a webhook is switched off so a
	// dead endpoint cannot keep the queue busy.
	disableAfter = 20
	maxErrorLen  = 500
)

var retryBackoff = []time.Duration{2 * time.Second, 10 * time.Second}

var errBlockedAddress = errors.New("địa chỉ đích thuộc mạng nội bộ nên bị chặn")

type target struct {
	id    pgtype.UUID
	url   string
	token string
}

type job struct {
	workspaceID string
	target      target
	eventID     uuid.UUID
	eventType   string
	body        []byte
	attempt     int
}

type result struct {
	success    bool
	statusCode int
	errText    string
	duration   time.Duration
}

type cacheEntry struct {
	targets []target
	expires time.Time
}

// isPublicAddr reports whether ip is a routable public address.
func isPublicAddr(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
		return false
	}
	// Carrier-grade NAT range, also not publicly routable.
	return !netip.MustParsePrefix("100.64.0.0/10").Contains(ip)
}

// newHTTPClient builds the client used for deliveries. Unless allowPrivate, the
// resolved IP is checked at connect time (so DNS tricks cannot reach internal
// hosts), proxies are ignored and redirects are not followed.
func newHTTPClient(allowPrivate bool) *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	if !allowPrivate {
		dialer.Control = func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip, err := netip.ParseAddr(host)
			if err != nil || !isPublicAddr(ip) {
				return errBlockedAddress
			}
			return nil
		}
	}
	return &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: requestTimeout,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       60 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func sign(token string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(token))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func truncate(s string) string {
	if len(s) > maxErrorLen {
		return s[:maxErrorLen]
	}
	return s
}

// send performs one delivery attempt.
func (s *Service) send(ctx context.Context, j job) result {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.target.url, bytes.NewReader(j.body))
	if err != nil {
		return result{errText: truncate(err.Error())}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Kairso-Webhook/1.0")
	req.Header.Set("Authorization", "Bearer "+j.target.token)
	req.Header.Set("X-Kairso-Event", j.eventType)
	req.Header.Set("X-Kairso-Delivery", j.eventID.String())
	req.Header.Set("X-Kairso-Signature", sign(j.target.token, j.body))

	resp, err := s.client.Do(req)
	if err != nil {
		msg := err.Error()
		if errors.Is(err, errBlockedAddress) {
			msg = errBlockedAddress.Error()
		}
		return result{errText: truncate(msg), duration: time.Since(start)}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	res := result{statusCode: resp.StatusCode, duration: time.Since(start)}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		res.success = true
	} else {
		res.errText = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return res
}

// record stores the final outcome of a delivery and updates the webhook's health.
func (s *Service) record(ctx context.Context, j job, res result) {
	if err := s.repo.InsertDelivery(ctx, database.InsertWebhookDeliveryParams{
		WebhookID:  j.target.id,
		EventID:    pgtype.UUID{Bytes: j.eventID, Valid: true},
		EventType:  j.eventType,
		Success:    res.success,
		StatusCode: int32(res.statusCode),
		Error:      res.errText,
		Attempts:   int32(j.attempt),
		DurationMs: int32(res.duration.Milliseconds()),
	}); err != nil {
		slog.Warn("webhook delivery log failed", "err", err)
	}
	if res.success {
		_ = s.repo.RecordSuccess(ctx, j.target.id, int32(res.statusCode))
		return
	}
	enabled, err := s.repo.RecordFailure(ctx, j.target.id, int32(res.statusCode), res.errText, disableAfter)
	if err == nil && !enabled {
		slog.Warn("webhook disabled after repeated failures", "webhook", pgutil.UUIDToString(j.target.id))
		s.invalidate(j.workspaceID)
	}
}

// Start launches the dispatcher and delivery workers; they stop with ctx.
func (s *Service) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-s.events:
				s.fanOut(ctx, ev)
			}
		}
	}()
	for range workerCount {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case j := <-s.jobs:
					s.process(ctx, j)
				}
			}
		}()
	}
}

// Broadcast implements realtime.Broadcaster: every workspace event is queued
// for delivery. It never blocks the caller; when the queue is full the event
// is dropped (and logged).
func (s *Service) Broadcast(event realtime.Event) {
	if event.WorkspaceID == "" || skipEvent(event.Type) {
		return
	}
	select {
	case s.events <- event:
	default:
		slog.Warn("webhook queue full, event dropped", "type", event.Type)
	}
}

// skipEvent filters per-user UI signals (unread counters) that are not
// workspace business events.
func skipEvent(eventType string) bool {
	return strings.HasPrefix(eventType, "notification.") || strings.HasPrefix(eventType, "notifications.")
}

func (s *Service) fanOut(ctx context.Context, ev realtime.Event) {
	targets := s.targets(ctx, ev.WorkspaceID)
	if len(targets) == 0 {
		return
	}
	id := uuid.New()
	body, err := json.Marshal(Payload{
		ID: id.String(), Type: ev.Type, WorkspaceID: ev.WorkspaceID, EntityID: ev.EntityID,
		ActorID: ev.ActorID, OccurredAt: time.Now().UTC(), Data: ev.Entity,
	})
	if err != nil {
		return
	}
	for _, t := range targets {
		select {
		case s.jobs <- job{workspaceID: ev.WorkspaceID, target: t, eventID: id, eventType: ev.Type, body: body, attempt: 1}:
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) process(ctx context.Context, j job) {
	res := s.send(ctx, j)
	if !res.success && j.attempt < maxAttempts && ctx.Err() == nil {
		next := j
		next.attempt++
		time.AfterFunc(retryBackoff[j.attempt-1], func() {
			select {
			case s.jobs <- next:
			default:
				// Queue saturated: give up on the retry and record the failure.
				s.record(context.WithoutCancel(ctx), j, res)
			}
		})
		return
	}
	s.record(context.WithoutCancel(ctx), j, res)
}

// targets returns the enabled webhooks of a workspace, cached briefly so busy
// workspaces do not hit the database for every event.
func (s *Service) targets(ctx context.Context, workspaceID string) []target {
	s.mu.Lock()
	if e, ok := s.cache[workspaceID]; ok && time.Now().Before(e.expires) {
		s.mu.Unlock()
		return e.targets
	}
	s.mu.Unlock()

	ws, err := pgutil.ParseUUID(workspaceID)
	if err != nil {
		return nil
	}
	rows, err := s.repo.ListEnabled(ctx, ws)
	if err != nil {
		slog.Warn("webhook lookup failed", "err", err)
		return nil
	}
	targets := make([]target, 0, len(rows))
	for _, r := range rows {
		targets = append(targets, target{id: r.ID, url: r.Url, token: r.Token})
	}
	s.mu.Lock()
	s.cache[workspaceID] = cacheEntry{targets: targets, expires: time.Now().Add(cacheTTL)}
	s.mu.Unlock()
	return targets
}

func (s *Service) invalidate(workspaceID string) {
	s.mu.Lock()
	delete(s.cache, workspaceID)
	s.mu.Unlock()
}
