package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Middleware returns an HTTP middleware that validates API key authentication.
// Requests to skipPaths (e.g., "/healthz") and static asset paths are allowed
// without authentication. If noAuth is true, all requests are allowed.
func Middleware(apiKey string, noAuth bool, skipPaths []string) func(http.Handler) http.Handler {
	skipSet := make(map[string]bool, len(skipPaths))
	for _, p := range skipPaths {
		skipSet[p] = true
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

			// No API key configured — reject all
			if apiKey == "" {
				writeAuthError(w, http.StatusUnauthorized, "API key not configured")
				return
			}

			// Extract Bearer token
			auth := r.Header.Get("Authorization")
			if auth == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(auth, prefix) {
				writeAuthError(w, http.StatusUnauthorized, "invalid Authorization format, expected 'Bearer <key>'")
				return
			}

			key := strings.TrimPrefix(auth, prefix)
			if !ValidateKey(key, apiKey) {
				writeAuthError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   http.StatusText(status),
		"message": message,
	})
}
