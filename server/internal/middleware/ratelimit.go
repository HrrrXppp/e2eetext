package middleware

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ekhrunov/messenger/server/internal/requestorigin"
	"github.com/ekhrunov/messenger/server/internal/service"
)

const (
	rateLimitSweepInterval = time.Minute
	rateLimitMinIdleTTL    = 10 * time.Minute
)

// RateLimitConfig describes one token bucket per key: it refills at
// RequestsPerMinute and holds at most Burst tokens. A non-positive value in
// either field disables limiting for the class (the middleware is a no-op).
type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

// IPKey keys requests by client IP. Use it for routes that run before any
// user identity exists (login, OAuth callback, refresh, WebSocket upgrade).
func IPKey(r *http.Request) string {
	return "ip:" + requestorigin.ClientIP(r)
}

// UserKey keys requests by authenticated user (provider + subject, from the
// TokenUser that RequireAuth stores in the context) and falls back to the
// client IP when no user is present. It must be installed after RequireAuth.
func UserKey(r *http.Request) string {
	if user, ok := r.Context().Value(service.TokenUserContextKey).(service.TokenUser); ok && user.Subject != "" {
		return "user:" + user.Provider + ":" + user.Subject
	}
	return IPKey(r)
}

// RateLimit returns middleware that rejects requests over the configured rate
// with 429 and a Retry-After header. Buckets live in process memory, which is
// correct only while the server runs as a single instance; scaling out
// horizontally would multiply the effective limit by the instance count and
// need a shared store instead.
func RateLimit(cfg RateLimitConfig, keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return newRateLimiter(cfg, time.Now).middleware(keyFunc)
}

// bucket is a token bucket: tokens refill continuously at the class rate up to
// the burst size, and each admitted request spends one.
type bucket struct {
	tokens   float64
	updated  time.Time
	lastSeen time.Time
}

type rateLimiter struct {
	cfg RateLimitConfig
	now func() time.Time

	mu        sync.Mutex
	buckets   map[string]*bucket
	lastSweep time.Time
}

func newRateLimiter(cfg RateLimitConfig, now func() time.Time) *rateLimiter {
	return &rateLimiter{cfg: cfg, now: now, buckets: make(map[string]*bucket), lastSweep: now()}
}

func (l *rateLimiter) enabled() bool {
	return l.cfg.RequestsPerMinute > 0 && l.cfg.Burst > 0
}

func (l *rateLimiter) middleware(keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !l.enabled() {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if retryAfter, ok := l.allow(keyFunc(r)); !ok {
				writeTooManyRequests(w, retryAfter)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// allow takes one token for key. When none is available it reports how long
// until one is.
func (l *rateLimiter) allow(key string) (time.Duration, bool) {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweepLocked(now)

	perSecond := float64(l.cfg.RequestsPerMinute) / 60
	burst := float64(l.cfg.Burst)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: burst, updated: now}
		l.buckets[key] = b
	}
	b.lastSeen = now

	if elapsed := now.Sub(b.updated).Seconds(); elapsed > 0 {
		b.tokens = min(burst, b.tokens+elapsed*perSecond)
		b.updated = now
	}
	if b.tokens >= 1 {
		b.tokens--
		return 0, true
	}
	return time.Duration((1 - b.tokens) / perSecond * float64(time.Second)), false
}

// sweepLocked bounds memory by dropping buckets idle long enough to have fully
// refilled (an evicted key restarts with a full bucket, so nothing is lost).
// It runs inline on the request path rather than on a background ticker, so
// there is no extra goroutine lifecycle to manage: the cost of walking the
// map is paid at most once per rateLimitSweepInterval, by whichever request
// happens to arrive after it elapses.
func (l *rateLimiter) sweepLocked(now time.Time) {
	if now.Sub(l.lastSweep) < rateLimitSweepInterval {
		return
	}
	l.lastSweep = now

	refill := time.Duration(float64(l.cfg.Burst) / float64(l.cfg.RequestsPerMinute) * float64(time.Minute))
	ttl := max(rateLimitMinIdleTTL, refill)
	for key, b := range l.buckets {
		if now.Sub(b.lastSeen) > ttl {
			delete(l.buckets, key)
		}
	}
}

func writeTooManyRequests(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := max(1, int(math.Ceil(retryAfter.Seconds())))
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests"})
}
