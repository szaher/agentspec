package integration_tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

	resp, err := http.Get(ts.URL + "/v1/agents")
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
	req, _ := http.NewRequest("GET", ts.URL+"/v1/agents", nil)
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
	req2, _ := http.NewRequest("GET", ts.URL+"/v1/agents", nil)
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

	req, _ := http.NewRequest("GET", ts.URL+"/v1/agents", nil)
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

// TestNoAuthBypassesAuth verifies that when no API key is configured, all requests succeed.
func TestNoAuthBypassesAuth(t *testing.T) {
	ts := newAuthTestServer("") // No API key = auth disabled
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/v1/agents")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with no auth configured, got %d", resp.StatusCode)
	}
}

// TestHealthzAlwaysAccessible verifies /healthz is always accessible regardless of auth.
func TestHealthzAlwaysAccessible(t *testing.T) {
	ts := newAuthTestServer("my-secret-key")
	defer ts.Close()

	// /healthz without any auth should work
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /healthz without auth, got %d", resp.StatusCode)
	}
}
