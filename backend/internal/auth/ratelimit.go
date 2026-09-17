package auth

import (
	"net"
	"net/http"
	"sync"
	"time"

	"institution-os/pkg/response"
)

// RateLimiter is a minimal per-IP token-bucket (pilot P0-4). Stdlib only —
// no new dependency. Not distributed: each app replica tracks its own
// buckets, which is fine for a single-instance pilot.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	limit    int
	interval time.Duration
}

type bucket struct {
	tokens   int
	lastFill time.Time
}

// NewRateLimiter allows `limit` requests per `interval` per client IP.
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{buckets: map[string]*bucket{}, limit: limit, interval: interval}
}

func (l *RateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b, ok := l.buckets[ip]
	if !ok {
		l.buckets[ip] = &bucket{tokens: l.limit - 1, lastFill: now}
		return true
	}
	if now.Sub(b.lastFill) >= l.interval {
		b.tokens = l.limit
		b.lastFill = now
	}
	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// Middleware rejects over-limit callers with 429.
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}
		if !l.allow(ip) {
			response.Error(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests — try again later")
			return
		}
		next.ServeHTTP(w, r)
	})
}
