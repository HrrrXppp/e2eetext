package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ekhrunov/messenger/server/internal/service"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

func newTestLimitedHandler(cfg RateLimitConfig, clock *fakeClock, keyFunc func(*http.Request) string) http.Handler {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	return newRateLimiter(cfg, clock.now).middleware(keyFunc)(next)
}

func do(h http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRateLimit_BurstThenReject(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1000, 0)}
	h := newTestLimitedHandler(RateLimitConfig{RequestsPerMinute: 60, Burst: 3}, clock, IPKey)

	for i := 0; i < 3; i++ {
		if rec := do(h, "192.0.2.1:1"); rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i, rec.Code)
		}
	}
	rec := do(h, "192.0.2.1:1")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After = %q, want 1", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	if body := rec.Body.String(); body != "{\"error\":\"too many requests\"}\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestRateLimit_RefillsOverTime(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1000, 0)}
	h := newTestLimitedHandler(RateLimitConfig{RequestsPerMinute: 60, Burst: 1}, clock, IPKey)

	if rec := do(h, "192.0.2.1:1"); rec.Code != http.StatusOK {
		t.Fatalf("first status = %d", rec.Code)
	}
	if rec := do(h, "192.0.2.1:1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", rec.Code)
	}
	clock.t = clock.t.Add(1100 * time.Millisecond)
	if rec := do(h, "192.0.2.1:1"); rec.Code != http.StatusOK {
		t.Fatalf("after refill status = %d, want 200", rec.Code)
	}
}

func TestRateLimit_RetryAfterReflectsRate(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1000, 0)}
	h := newTestLimitedHandler(RateLimitConfig{RequestsPerMinute: 6, Burst: 1}, clock, IPKey)

	do(h, "192.0.2.1:1")
	rec := do(h, "192.0.2.1:1")
	if got := rec.Header().Get("Retry-After"); got != "10" {
		t.Fatalf("Retry-After = %q, want 10", got)
	}
}

func TestRateLimit_KeysAreIsolated(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1000, 0)}
	h := newTestLimitedHandler(RateLimitConfig{RequestsPerMinute: 60, Burst: 1}, clock, IPKey)

	if rec := do(h, "192.0.2.1:1"); rec.Code != http.StatusOK {
		t.Fatalf("a first = %d", rec.Code)
	}
	if rec := do(h, "192.0.2.1:1"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("a second = %d, want 429", rec.Code)
	}
	if rec := do(h, "192.0.2.2:1"); rec.Code != http.StatusOK {
		t.Fatalf("b first = %d, want 200", rec.Code)
	}
}

func TestRateLimit_DisabledWhenNonPositive(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1000, 0)}
	h := newTestLimitedHandler(RateLimitConfig{}, clock, IPKey)
	for i := 0; i < 50; i++ {
		if rec := do(h, "192.0.2.1:1"); rec.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i, rec.Code)
		}
	}
}

func TestUserKey(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "192.0.2.1:1"
	if got := UserKey(req); got != "ip:192.0.2.1" {
		t.Fatalf("anonymous key = %q", got)
	}
	ctx := context.WithValue(req.Context(), service.TokenUserContextKey, service.TokenUser{Provider: "google", Subject: "abc"})
	if got := UserKey(req.WithContext(ctx)); got != "user:google:abc" {
		t.Fatalf("user key = %q", got)
	}
}

func TestRateLimit_PerUserIgnoresIP(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1000, 0)}
	h := newTestLimitedHandler(RateLimitConfig{RequestsPerMinute: 60, Burst: 1}, clock, UserKey)

	send := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.RemoteAddr = ip
		ctx := context.WithValue(req.Context(), service.TokenUserContextKey, service.TokenUser{Provider: "google", Subject: "u1"})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req.WithContext(ctx))
		return rec.Code
	}
	if code := send("192.0.2.1:1"); code != http.StatusOK {
		t.Fatalf("first = %d", code)
	}
	if code := send("192.0.2.9:1"); code != http.StatusTooManyRequests {
		t.Fatalf("same user other ip = %d, want 429", code)
	}
}

func TestRateLimit_SweepsIdleBuckets(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1000, 0)}
	l := newRateLimiter(RateLimitConfig{RequestsPerMinute: 60, Burst: 1}, clock.now)

	l.allow("a")
	l.allow("b")
	clock.t = clock.t.Add(rateLimitMinIdleTTL + 2*rateLimitSweepInterval)
	l.allow("c")

	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.buckets) != 1 {
		t.Fatalf("buckets = %d, want 1 after sweep", len(l.buckets))
	}
}
