package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiterAllow(t *testing.T) {
	t.Run("first N requests within burst are allowed", func(t *testing.T) {
		cfg := RateLimitConfig{RequestsPerSecond: 10, Burst: 5}
		rl := NewRateLimiter(cfg)

		for i := 0; i < 5; i++ {
			if !rl.Allow("client1") {
				t.Errorf("Allow() = false for request %d, want true (within burst)", i+1)
			}
		}
	})

	t.Run("returns false after burst is exhausted", func(t *testing.T) {
		cfg := RateLimitConfig{RequestsPerSecond: 10, Burst: 3}
		rl := NewRateLimiter(cfg)

		// Exhaust the burst
		for i := 0; i < 3; i++ {
			rl.Allow("client1")
		}

		if rl.Allow("client1") {
			t.Error("Allow() = true after burst exhausted, want false")
		}
	})
}

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()

	if cfg.RequestsPerSecond != 10 {
		t.Errorf("RequestsPerSecond = %v, want 10", cfg.RequestsPerSecond)
	}
	if cfg.Burst != 20 {
		t.Errorf("Burst = %d, want 20", cfg.Burst)
	}
}

func TestRateLimitConfigFromEnv(t *testing.T) {
	t.Run("parses valid env var", func(t *testing.T) {
		t.Setenv("AGENTSPEC_RATE_LIMIT", "50:100")

		cfg := RateLimitConfigFromEnv()

		if cfg.RequestsPerSecond != 50 {
			t.Errorf("RequestsPerSecond = %v, want 50", cfg.RequestsPerSecond)
		}
		if cfg.Burst != 100 {
			t.Errorf("Burst = %d, want 100", cfg.Burst)
		}
	})

	t.Run("returns defaults when env is empty", func(t *testing.T) {
		t.Setenv("AGENTSPEC_RATE_LIMIT", "")

		cfg := RateLimitConfigFromEnv()
		defaults := DefaultRateLimitConfig()

		if cfg.RequestsPerSecond != defaults.RequestsPerSecond {
			t.Errorf("RequestsPerSecond = %v, want %v", cfg.RequestsPerSecond, defaults.RequestsPerSecond)
		}
		if cfg.Burst != defaults.Burst {
			t.Errorf("Burst = %d, want %d", cfg.Burst, defaults.Burst)
		}
	})
}

func TestAuthFailure(t *testing.T) {
	t.Run("returns false before reaching threshold", func(t *testing.T) {
		rl := NewRateLimiter(DefaultRateLimitConfig())

		for i := 0; i < 9; i++ {
			blocked := rl.AuthFailure("192.168.1.1")
			if blocked {
				t.Errorf("AuthFailure() = true at attempt %d, want false (below threshold)", i+1)
			}
		}
	})

	t.Run("returns true (blocked) after 10 failures", func(t *testing.T) {
		rl := NewRateLimiter(DefaultRateLimitConfig())

		var blocked bool
		for i := 0; i < 10; i++ {
			blocked = rl.AuthFailure("192.168.1.1")
		}

		if !blocked {
			t.Error("AuthFailure() = false after 10 failures, want true (blocked)")
		}
	})
}

func TestIsAuthBlocked(t *testing.T) {
	t.Run("returns true when IP is blocked", func(t *testing.T) {
		rl := NewRateLimiter(DefaultRateLimitConfig())

		// Trigger block by exceeding failure threshold
		for i := 0; i < 10; i++ {
			rl.AuthFailure("192.168.1.1")
		}

		if !rl.IsAuthBlocked("192.168.1.1") {
			t.Error("IsAuthBlocked() = false, want true (IP should be blocked)")
		}
	})

	t.Run("returns false for unknown IP", func(t *testing.T) {
		rl := NewRateLimiter(DefaultRateLimitConfig())

		if rl.IsAuthBlocked("10.0.0.1") {
			t.Error("IsAuthBlocked() = true for unknown IP, want false")
		}
	})
}

func TestAuthSuccess(t *testing.T) {
	t.Run("clears failure tracking", func(t *testing.T) {
		rl := NewRateLimiter(DefaultRateLimitConfig())

		// Accumulate some failures (but not enough to block)
		for i := 0; i < 5; i++ {
			rl.AuthFailure("192.168.1.1")
		}

		rl.AuthSuccess("192.168.1.1")

		// After clearing, 9 more failures should not trigger a block
		var blocked bool
		for i := 0; i < 9; i++ {
			blocked = rl.AuthFailure("192.168.1.1")
		}
		if blocked {
			t.Error("AuthFailure() = true after AuthSuccess() cleared tracking, want false")
		}
	})
}

func TestAuthBlockRetryAfter(t *testing.T) {
	t.Run("returns positive value when blocked", func(t *testing.T) {
		rl := NewRateLimiter(DefaultRateLimitConfig())

		// Trigger block
		for i := 0; i < 10; i++ {
			rl.AuthFailure("192.168.1.1")
		}

		retryAfter := rl.AuthBlockRetryAfter("192.168.1.1")
		if retryAfter <= 0 {
			t.Errorf("AuthBlockRetryAfter() = %d, want > 0", retryAfter)
		}
	})

	t.Run("returns zero for non-blocked IP", func(t *testing.T) {
		rl := NewRateLimiter(DefaultRateLimitConfig())

		retryAfter := rl.AuthBlockRetryAfter("10.0.0.1")
		if retryAfter != 0 {
			t.Errorf("AuthBlockRetryAfter() = %d, want 0", retryAfter)
		}
	})
}

func TestRateLimiterMiddleware(t *testing.T) {
	t.Run("allows requests within rate limit", func(t *testing.T) {
		cfg := RateLimitConfig{RequestsPerSecond: 10, Burst: 5}
		rl := NewRateLimiter(cfg)

		handler := rl.Middleware(ClientIPKeyFunc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))

		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Body.String() != "ok" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "ok")
		}
	})

	t.Run("returns 429 when rate limit exceeded", func(t *testing.T) {
		cfg := RateLimitConfig{RequestsPerSecond: 10, Burst: 2}
		rl := NewRateLimiter(cfg)

		handler := rl.Middleware(ClientIPKeyFunc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Exhaust the burst
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/data?i=%d", i), nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
		}

		// Next request should be rate limited
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
		}

		retryAfter := rec.Header().Get("Retry-After")
		if retryAfter == "" {
			t.Error("missing Retry-After header on 429 response")
		}
	})
}
