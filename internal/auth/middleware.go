package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Middleware returns an HTTP middleware that validates API key authentication.
// Requests to skipPaths (e.g., "/healthz") and static asset paths are allowed
// without authentication. If noAuth is true, all requests are allowed.
// If rateLimiter is non-nil, failed auth attempts are tracked and IPs are blocked
// after exceeding the threshold (10 failures/min, 5-min block).
func Middleware(apiKey string, noAuth bool, skipPaths []string, rateLimiter ...*RateLimiter) func(http.Handler) http.Handler {
	skipSet := make(map[string]bool, len(skipPaths))
	for _, p := range skipPaths {
		skipSet[p] = true
	}

	var rl *RateLimiter
	if len(rateLimiter) > 0 {
		rl = rateLimiter[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth if disabled
			if noAuth {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for allowed paths
			if skipSet[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for static assets
			if strings.HasPrefix(r.URL.Path, "/static/") ||
				r.URL.Path == "/favicon.ico" {
				next.ServeHTTP(w, r)
				return
			}

			// Check auth rate limiting before validation
			clientIP := ClientIPKeyFunc(r)
			if rl != nil && rl.IsAuthBlocked(clientIP) {
				retryAfter := rl.AuthBlockRetryAfter(clientIP)
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				writeAuthError(w, http.StatusTooManyRequests, "Too many failed authentication attempts. Try again later.")
				return
			}

			// No API key configured — reject all
			if apiKey == "" {
				writeAuthError(w, http.StatusUnauthorized, "API key not configured")
				return
			}

			// Extract Bearer token
			auth := r.Header.Get("Authorization")
			if auth == "" {
				if rl != nil {
					rl.AuthFailure(clientIP)
				}
				writeAuthError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				if rl != nil {
					rl.AuthFailure(clientIP)
				}
				writeAuthError(w, http.StatusUnauthorized, "invalid Authorization format, expected 'Bearer <key>'")
				return
			}

			key := strings.TrimPrefix(auth, prefix)
			if !ValidateKey(key, apiKey) {
				if rl != nil {
					rl.AuthFailure(clientIP)
				}
				writeAuthError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			// Success — clear failure tracking
			if rl != nil {
				rl.AuthSuccess(clientIP)
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   http.StatusText(status),
		"message": message,
	})
}
