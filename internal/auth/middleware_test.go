package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// okHandler is a simple handler that writes 200 OK with body "ok".
func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestMiddleware(t *testing.T) {
	const apiKey = "test-api-key"
	skipPaths := []string{"/healthz"}

	t.Run("valid Bearer token returns 200", func(t *testing.T) {
		mw := Middleware(apiKey, false, skipPaths)
		handler := mw(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "ok" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
		}
	})

	t.Run("invalid Bearer token returns 401", func(t *testing.T) {
		mw := Middleware(apiKey, false, skipPaths)
		handler := mw(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.Header.Set("Authorization", "Bearer wrong-key")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("missing Authorization header returns 401", func(t *testing.T) {
		mw := Middleware(apiKey, false, skipPaths)
		handler := mw(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("noAuth=true always passes through", func(t *testing.T) {
		mw := Middleware(apiKey, true, skipPaths)
		handler := mw(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		// No Authorization header set
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "ok" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
		}
	})

	t.Run("skip path /healthz returns 200 without auth", func(t *testing.T) {
		mw := Middleware(apiKey, false, skipPaths)
		handler := mw(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("skip path /static/app.js returns 200 without auth", func(t *testing.T) {
		mw := Middleware(apiKey, false, skipPaths)
		handler := mw(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("skip path /favicon.ico returns 200 without auth", func(t *testing.T) {
		mw := Middleware(apiKey, false, skipPaths)
		handler := mw(okHandler())

		req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestClientIPKeyFunc(t *testing.T) {
	t.Run("with X-Forwarded-For returns first IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")

		got := ClientIPKeyFunc(req)
		if got != "203.0.113.50" {
			t.Errorf("ClientIPKeyFunc() = %q, want %q", got, "203.0.113.50")
		}
	})

	t.Run("without X-Forwarded-For returns RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.1:12345"

		got := ClientIPKeyFunc(req)
		if got != "192.168.1.1:12345" {
			t.Errorf("ClientIPKeyFunc() = %q, want %q", got, "192.168.1.1:12345")
		}
	})
}
