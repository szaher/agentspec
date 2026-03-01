package integration_tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/szaher/designs/agentz/internal/auth"
	"github.com/szaher/designs/agentz/internal/loop"
	"github.com/szaher/designs/agentz/internal/runtime"
	"github.com/szaher/designs/agentz/internal/session"
	"github.com/szaher/designs/agentz/internal/tools"
)

func newAuthTestServer(apiKey string) *httptest.Server {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "test-agent", FQN: "test/test-agent", Model: "test-model", System: "test"},
		},
	}
	registry := tools.NewRegistry()
	sessionMgr := session.NewManager(session.NewMemoryStore(0), nil)
	strategy := &loop.ReActStrategy{}

	var opts []runtime.ServerOption
	if apiKey != "" {
		opts = append(opts, runtime.WithAPIKey(apiKey))
	}

	server := runtime.NewServer(config, nil, registry, sessionMgr, strategy, opts...)
	return httptest.NewServer(server.Handler())
}

// TestAuthRejectsWithoutKey verifies requests without API key return 401.
func TestAuthRejectsWithoutKey(t *testing.T) {
	ts := newAuthTestServer("my-secret-key")
	defer ts.Close()

	req, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/agents", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without key, got %d", resp.StatusCode)
	}
}

// TestAuthSucceedsWithValidKey verifies requests with valid API key succeed.
func TestAuthSucceedsWithValidKey(t *testing.T) {
	ts := newAuthTestServer("my-secret-key")
	defer ts.Close()

	// Test X-API-Key header
	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/agents", nil)
	req.Header.Set("X-API-Key", "my-secret-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with valid X-API-Key, got %d", resp.StatusCode)
	}

	// Test Bearer token
	req2, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/agents", nil)
	req2.Header.Set("Authorization", "Bearer my-secret-key")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with valid Bearer token, got %d", resp2.StatusCode)
	}
}

// TestAuthRejectsInvalidKey verifies requests with wrong API key return 401.
func TestAuthRejectsInvalidKey(t *testing.T) {
	ts := newAuthTestServer("my-secret-key")
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/agents", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with invalid key, got %d", resp.StatusCode)
	}
}

// TestNoAuthWithoutFlagRejects verifies that no API key WITHOUT --no-auth rejects all requests.
func TestNoAuthWithoutFlagRejects(t *testing.T) {
	// No API key, no --no-auth flag = reject all
	ts := newAuthTestServer("")
	defer ts.Close()

	req, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/agents", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with no auth and no --no-auth flag, got %d", resp.StatusCode)
	}
}

// TestNoAuthWithFlagAllows verifies that --no-auth flag allows unauthenticated access.
func TestNoAuthWithFlagAllows(t *testing.T) {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "test-agent", FQN: "test/test-agent", Model: "test-model", System: "test"},
		},
	}
	registry := tools.NewRegistry()
	sessionMgr := session.NewManager(session.NewMemoryStore(0), nil)
	strategy := &loop.ReActStrategy{}

	server := runtime.NewServer(config, nil, registry, sessionMgr, strategy, runtime.WithNoAuth(true))
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	req, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/agents", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with --no-auth, got %d", resp.StatusCode)
	}
}

// TestAuthRateLimiting verifies brute-force protection blocks IPs after 10 failures.
func TestAuthRateLimiting(t *testing.T) {
	rl := auth.NewRateLimiter(auth.DefaultRateLimitConfig())

	// Simulate 10 failures
	for i := 0; i < 10; i++ {
		blocked := rl.AuthFailure("192.168.1.100")
		if i < 9 && blocked {
			t.Errorf("should not be blocked after %d failures", i+1)
		}
		if i == 9 && !blocked {
			t.Error("should be blocked after 10 failures")
		}
	}

	// Should be blocked
	if !rl.IsAuthBlocked("192.168.1.100") {
		t.Error("IP should be blocked")
	}

	// Correct key should still be blocked
	if !rl.IsAuthBlocked("192.168.1.100") {
		t.Error("IP should still be blocked even with correct key")
	}

	// Different IP should not be blocked
	if rl.IsAuthBlocked("192.168.1.200") {
		t.Error("different IP should not be blocked")
	}

	// Success clears tracking for unblocked IPs
	rl.AuthFailure("10.0.0.1")
	rl.AuthSuccess("10.0.0.1")
	if rl.IsAuthBlocked("10.0.0.1") {
		t.Error("IP should not be blocked after success")
	}
}

// TestHealthzAlwaysAccessible verifies /healthz is always accessible regardless of auth.
func TestHealthzAlwaysAccessible(t *testing.T) {
	ts := newAuthTestServer("my-secret-key")
	defer ts.Close()

	// /healthz without any auth should work
	req, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/healthz", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /healthz without auth, got %d", resp.StatusCode)
	}
}
