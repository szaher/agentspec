package auth

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
}

// DefaultRateLimitConfig returns the default rate limit settings.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 10,
		Burst:             20,
	}
}

// RateLimitConfigFromEnv reads rate limit config from the AGENTSPEC_RATE_LIMIT env var.
// Format: "rate:burst" (e.g., "10:20" means 10 req/s with burst of 20).
func RateLimitConfigFromEnv() RateLimitConfig {
	cfg := DefaultRateLimitConfig()

	val := os.Getenv("AGENTSPEC_RATE_LIMIT")
	if val == "" {
		return cfg
	}

	parts := strings.SplitN(val, ":", 2)
	if rate, err := strconv.ParseFloat(parts[0], 64); err == nil && rate > 0 {
		cfg.RequestsPerSecond = rate
	}
	if len(parts) > 1 {
		if burst, err := strconv.Atoi(parts[1]); err == nil && burst > 0 {
			cfg.Burst = burst
		}
	}

	return cfg
}

// RateLimiter implements per-client token bucket rate limiting.
type RateLimiter struct {
	mu      sync.Mutex
	config  RateLimitConfig
	buckets map[string]*bucket

	authMu       sync.Mutex
	authFailures map[string]*authBucket
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// authBucket tracks failed authentication attempts per IP.
type authBucket struct {
	failures    int
	windowStart time.Time
	blockedUntil time.Time
}

const (
	authMaxFailures   = 10
	authWindowDur     = 1 * time.Minute
	authBlockDur      = 5 * time.Minute
	authEvictInterval = 10 * time.Minute
)

// NewRateLimiter creates a rate limiter with the given configuration.
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		config:       config,
		buckets:      make(map[string]*bucket),
		authFailures: make(map[string]*authBucket),
	}
}

// Allow checks if a request from the given key is allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{
			tokens:     float64(rl.config.Burst),
			lastRefill: time.Now(),
		}
		rl.buckets[key] = b
	}

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * rl.config.RequestsPerSecond
	if b.tokens > float64(rl.config.Burst) {
		b.tokens = float64(rl.config.Burst)
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// IsAuthBlocked checks if an IP is blocked due to too many auth failures.
func (rl *RateLimiter) IsAuthBlocked(ip string) bool {
	rl.authMu.Lock()
	defer rl.authMu.Unlock()

	b, ok := rl.authFailures[ip]
	if !ok {
		return false
	}

	now := time.Now()
	if now.Before(b.blockedUntil) {
		return true
	}

	// Block expired — reset
	if !b.blockedUntil.IsZero() {
		delete(rl.authFailures, ip)
		return false
	}

	return false
}

// AuthBlockRetryAfter returns the number of seconds until the block expires.
func (rl *RateLimiter) AuthBlockRetryAfter(ip string) int {
	rl.authMu.Lock()
	defer rl.authMu.Unlock()

	b, ok := rl.authFailures[ip]
	if !ok {
		return 0
	}
	remaining := time.Until(b.blockedUntil).Seconds()
	if remaining <= 0 {
		return 0
	}
	return int(remaining) + 1
}

// AuthFailure records a failed authentication attempt from an IP.
// Returns true if the IP is now blocked.
func (rl *RateLimiter) AuthFailure(ip string) bool {
	rl.authMu.Lock()
	defer rl.authMu.Unlock()

	now := time.Now()
	b, ok := rl.authFailures[ip]
	if !ok {
		b = &authBucket{
			failures:    0,
			windowStart: now,
		}
		rl.authFailures[ip] = b
	}

	// Reset window if expired
	if now.Sub(b.windowStart) > authWindowDur {
		b.failures = 0
		b.windowStart = now
	}

	b.failures++

	if b.failures >= authMaxFailures {
		b.blockedUntil = now.Add(authBlockDur)
		return true
	}

	// Evict stale entries periodically
	if len(rl.authFailures) > 1000 {
		rl.evictStaleAuthEntries(now)
	}

	return false
}

// AuthSuccess clears auth failure tracking for an IP.
func (rl *RateLimiter) AuthSuccess(ip string) {
	rl.authMu.Lock()
	defer rl.authMu.Unlock()
	delete(rl.authFailures, ip)
}

func (rl *RateLimiter) evictStaleAuthEntries(now time.Time) {
	for ip, b := range rl.authFailures {
		if !b.blockedUntil.IsZero() && now.After(b.blockedUntil) {
			delete(rl.authFailures, ip)
		} else if now.Sub(b.windowStart) > authEvictInterval {
			delete(rl.authFailures, ip)
		}
	}
}

// Middleware returns HTTP middleware that applies rate limiting.
// The key function extracts a rate limit key from the request (e.g., client IP or agent name).
func (rl *RateLimiter) Middleware(keyFunc func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !rl.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", 1.0/rl.config.RequestsPerSecond))
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = fmt.Fprintf(w, `{"error":"rate_limited","message":"Rate limit exceeded. Try again later."}`)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ClientIPKeyFunc extracts the client IP from the request for rate limiting.
func ClientIPKeyFunc(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.SplitN(forwarded, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr
}
