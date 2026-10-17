package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ekhrunov/messenger/server/internal/config"
	"github.com/ekhrunov/messenger/server/internal/node"
)

func newRateLimitedTestHandler(t *testing.T, limits config.RateLimits) http.Handler {
	t.Helper()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	application, err := New(
		context.Background(),
		config.Config{RateLimits: limits},
		logger,
		db,
		node.Registry{ID: "99999999-9999-9999-9999-999999999999"},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return application.Handler()
}

// TestRateLimit_AuthRouteRejectsOverBurst drives POST /api/v1/auth/refresh
// through the real app.New(...) handler to confirm the auth-class rate
// limiter is actually wired in (not just unit-tested in isolation): once the
// configured burst is exhausted, further requests from the same IP get 429
// before the handler runs, and /health stays unaffected since it never gets
// a limiter at all.
func TestRateLimit_AuthRouteRejectsOverBurst(t *testing.T) {
	handler := newRateLimitedTestHandler(t, config.RateLimits{
		Auth: config.RateLimitConfig{RequestsPerMinute: 60, Burst: 1},
	})

	refresh := func() int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := refresh(); code == http.StatusTooManyRequests {
		t.Fatalf("first request status = %d, want anything but 429", code)
	}
	if code := refresh(); code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", code)
	}

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthReq.RemoteAddr = "192.0.2.1:1234"
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("/health status = %d, want 200 (unaffected by auth rate limit)", healthRec.Code)
	}
}

// TestRateLimit_ProvidersRouteRejectsOverBurst confirms the public provider
// list shares the IP-keyed auth-class limiter.
func TestRateLimit_ProvidersRouteRejectsOverBurst(t *testing.T) {
	handler := newRateLimitedTestHandler(t, config.RateLimits{
		Auth: config.RateLimitConfig{RequestsPerMinute: 60, Burst: 1},
	})

	providers := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/providers", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := providers(); code == http.StatusTooManyRequests {
		t.Fatalf("first request status = %d, want anything but 429", code)
	}
	if code := providers(); code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", code)
	}
}

// TestRateLimit_HealthNeverLimited confirms /health has no limiter wired in
// at all, even when every route class is configured aggressively enough to
// reject a second request outright — Docker/ALB health checks must not be
// able to trip it.
func TestRateLimit_HealthNeverLimited(t *testing.T) {
	tight := config.RateLimitConfig{RequestsPerMinute: 1, Burst: 1}
	handler := newRateLimitedTestHandler(t, config.RateLimits{
		Auth: tight, Search: tight, Message: tight, Default: tight, WS: tight,
	})

	const (
		method     = http.MethodGet
		path       = "/health"
		remoteAddr = "192.0.2.1:1234"
	)
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, rec.Code)
		}
	}
}

// TestRateLimit_DisabledByDefaultInTests confirms the zero-value
// config.RateLimits used by the other app tests (which build Config{}
// directly rather than going through config.Load) leaves rate limiting off,
// so those tests aren't coupled to this change.
func TestRateLimit_DisabledByDefaultInTests(t *testing.T) {
	handler := newRateLimitedTestHandler(t, config.RateLimits{})

	const (
		method     = http.MethodPost
		path       = "/api/v1/auth/refresh"
		remoteAddr = "192.0.2.1:1234"
	)
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(method, path, nil)
		req.RemoteAddr = remoteAddr
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d got 429 with zero-value RateLimits, want rate limiting disabled", i)
		}
	}
}
