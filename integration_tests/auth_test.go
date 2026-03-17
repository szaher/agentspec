package integration_tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/szaher/agentspec/internal/auth"
	"github.com/szaher/agentspec/internal/eviction"
	"github.com/szaher/agentspec/internal/llm"
	"github.com/szaher/agentspec/internal/loop"
	"github.com/szaher/agentspec/internal/memory"
	"github.com/szaher/agentspec/internal/runtime"
	"github.com/szaher/agentspec/internal/session"
	"github.com/szaher/agentspec/internal/tools"
)

func newAuthTestServer(apiKey string) *httptest.Server {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "test-agent", FQN: "test/test-agent", Model: "test-model", System: "test"},
		},
	}
	registry := tools.NewRegistry()
	sessionMgr := session.NewManager(session.NewMemoryStore(0, 0), nil)
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
	sessionMgr := session.NewManager(session.NewMemoryStore(0, 0), nil)
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
	rl := auth.NewRateLimiter(auth.DefaultRateLimitConfig(), eviction.Policy{})

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

// TestMultiUserAuthResolvesUser verifies multi-user auth resolves user from API key.
func TestMultiUserAuthResolvesUser(t *testing.T) {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "agent-a", FQN: "test/agent-a", Model: "test-model", System: "test", MaxTurns: 5},
			{Name: "agent-b", FQN: "test/agent-b", Model: "test-model", System: "test", MaxTurns: 5},
		},
	}

	mockClient := llm.NewMockClient(
		llm.MockResponse{Content: "Hello!", StopReason: llm.StopEndTurn, Usage: llm.TokenUsage{InputTokens: 10, OutputTokens: 5}},
		llm.MockResponse{Content: "Hello!", StopReason: llm.StopEndTurn, Usage: llm.TokenUsage{InputTokens: 10, OutputTokens: 5}},
	)

	userStore := auth.NewUserStore([]auth.UserDef{
		{Name: "alice", ResolvedKey: "alice-key", Agents: []string{"agent-a"}, Role: "invoke"},
		{Name: "bob", ResolvedKey: "bob-key", Agents: []string{"agent-b"}, Role: "invoke"},
		{Name: "admin", ResolvedKey: "admin-key", Agents: nil, Role: "admin"},
	})

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0, 0), memory.NewSlidingWindow(50))
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, mockClient, registry, mgr, strategy,
		runtime.WithUserStore(userStore))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Alice can access agent-a
	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/agent-a/invoke", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "alice-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 200 for alice on agent-a, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestMultiUserAuthForbidden verifies user cannot access unauthorized agent.
func TestMultiUserAuthForbidden(t *testing.T) {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "agent-a", FQN: "test/agent-a", Model: "test-model", System: "test", MaxTurns: 5},
			{Name: "agent-b", FQN: "test/agent-b", Model: "test-model", System: "test", MaxTurns: 5},
		},
	}

	userStore := auth.NewUserStore([]auth.UserDef{
		{Name: "alice", ResolvedKey: "alice-key", Agents: []string{"agent-a"}, Role: "invoke"},
	})

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0, 0), nil)
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, nil, registry, mgr, strategy,
		runtime.WithUserStore(userStore))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Alice cannot access agent-b
	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/agent-b/invoke", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "alice-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for alice on agent-b, got %d", resp.StatusCode)
	}

	var body map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "forbidden" {
		t.Errorf("expected error 'forbidden', got %q", body["error"])
	}
}

// TestMultiUserAuthAdminBypass verifies admin role can access any agent.
func TestMultiUserAuthAdminBypass(t *testing.T) {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "agent-a", FQN: "test/agent-a", Model: "test-model", System: "test", MaxTurns: 5},
		},
	}

	mockClient := llm.NewMockClient(llm.MockResponse{
		Content: "OK", StopReason: llm.StopEndTurn, Usage: llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
	})

	userStore := auth.NewUserStore([]auth.UserDef{
		{Name: "admin", ResolvedKey: "admin-key", Agents: nil, Role: "admin"},
	})

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0, 0), memory.NewSlidingWindow(50))
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, mockClient, registry, mgr, strategy,
		runtime.WithUserStore(userStore))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/agent-a/invoke", strings.NewReader(`{"message":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "admin-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for admin, got %d", resp.StatusCode)
	}
}

// TestMultiUserInvalidKey verifies unknown API key returns 401.
func TestMultiUserInvalidKey(t *testing.T) {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "agent-a", FQN: "test/agent-a", Model: "test-model", System: "test"},
		},
	}

	userStore := auth.NewUserStore([]auth.UserDef{
		{Name: "alice", ResolvedKey: "alice-key", Agents: nil, Role: "invoke"},
	})

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0, 0), nil)
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, nil, registry, mgr, strategy,
		runtime.WithUserStore(userStore))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/agents", nil)
	req.Header.Set("X-API-Key", "unknown-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for unknown key in multi-user mode, got %d", resp.StatusCode)
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
