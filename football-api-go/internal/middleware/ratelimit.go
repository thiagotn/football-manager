package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ipBucket struct {
	count   int
	resetAt time.Time
}

// LoginRateLimiter enforces 5 login attempts per IP per 15 minutes.
type LoginRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*ipBucket
	limit   int
	window  time.Duration
}

func NewLoginRateLimiter() *LoginRateLimiter {
	l := &LoginRateLimiter{
		buckets: make(map[string]*ipBucket),
		limit:   5,
		window:  15 * time.Minute,
	}
	go l.cleanup()
	return l
}

// Middleware wraps a handler with rate limiting.
func (l *LoginRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		if !l.allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "900")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"detail": "too many requests"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *LoginRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[ip]
	if !ok || time.Now().After(b.resetAt) {
		l.buckets[ip] = &ipBucket{count: 1, resetAt: time.Now().Add(l.window)}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	return true
}

func (l *LoginRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for ip, b := range l.buckets {
			if now.After(b.resetAt) {
				delete(l.buckets, ip)
			}
		}
		l.mu.Unlock()
	}
}

func extractIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may contain multiple IPs; take the first (client IP)
		if idx := len(xff); idx > 0 {
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					return strings.TrimSpace(xff[:i])
				}
			}
			return strings.TrimSpace(xff)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
