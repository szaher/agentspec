package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/frontend"
	"github.com/szaher/designs/agentz/internal/loop"
	"github.com/szaher/designs/agentz/internal/runtime"
	"github.com/szaher/designs/agentz/internal/session"
	"github.com/szaher/designs/agentz/internal/telemetry"
	"github.com/szaher/designs/agentz/internal/tools"
)

// TestFrontendServesIndexHTML verifies the frontend handler serves index.html.
func TestFrontendServesIndexHTML(t *testing.T) {
	h := frontend.NewHandler("/")
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "AgentSpec") {
		t.Error("expected index.html to contain 'AgentSpec'")
	}
	if !strings.Contains(body, "app.js") {
		t.Error("expected index.html to reference app.js")
	}
}

// TestFrontendServesAppJS verifies the frontend handler serves app.js.
func TestFrontendServesAppJS(t *testing.T) {
	h := frontend.NewHandler("/")
	req := httptest.NewRequest("GET", "/app.js", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "AgentSpec Frontend") {
		t.Error("expected app.js content")
	}
}

// TestFrontendSPAFallback verifies unknown paths fall back to index.html.
func TestFrontendSPAFallback(t *testing.T) {
	h := frontend.NewHandler("/")
	req := httptest.NewRequest("GET", "/unknown/route", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for SPA fallback, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "AgentSpec") {
		t.Error("expected SPA fallback to serve index.html")
	}
}

// TestFrontendMountedOnServer verifies the frontend is accessible through the runtime server.
func TestFrontendMountedOnServer(t *testing.T) {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{
				Name:     "test-agent",
				FQN:      "test/test-agent",
				Model:    "test-model",
				System:   "You are a test agent.",
				MaxTurns: 1,
			},
		},
	}

	registry := tools.NewRegistry()
	sessionMgr := session.NewManager(session.NewMemoryStore(0), nil)
	strategy := &loop.ReActStrategy{}
	metrics := telemetry.NewMetrics()

	server := runtime.NewServer(config, nil, registry, sessionMgr, strategy,
		runtime.WithUI(true),
		runtime.WithMetrics(metrics),
	)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// Fetch index.html from root
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /, got %d", resp.StatusCode)
	}

	// Fetch agents API (should also work without auth since no key set)
	resp2, err := http.Get(ts.URL + "/v1/agents")
	if err != nil {
		t.Fatalf("GET /v1/agents error: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /v1/agents, got %d", resp2.StatusCode)
	}

	var agentResp map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&agentResp); err != nil {
		t.Fatalf("decode agents response: %v", err)
	}
	agents, ok := agentResp["agents"].([]interface{})
	if !ok || len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %v", agentResp)
	}
}

// TestFrontendDisabledByDefault verifies the frontend is not served when UI is disabled.
func TestFrontendDisabledByDefault(t *testing.T) {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "test-agent", FQN: "test/test-agent", Model: "test-model"},
		},
	}

	registry := tools.NewRegistry()
	sessionMgr := session.NewManager(session.NewMemoryStore(0), nil)
	strategy := &loop.ReActStrategy{}

	// No WithUI option — frontend should not be mounted
	server := runtime.NewServer(config, nil, registry, sessionMgr, strategy)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Without UI, root should return 404 (no handler registered for /)
	if resp.StatusCode == http.StatusOK {
		t.Error("expected non-200 when UI is disabled, but got 200")
	}
}

// TestFrontendStaticAssetsSkipAuth verifies static assets bypass auth when UI is enabled.
func TestFrontendStaticAssetsSkipAuth(t *testing.T) {
	config := &runtime.RuntimeConfig{
		Agents: []runtime.AgentConfig{
			{Name: "test-agent", FQN: "test/test-agent", Model: "test-model"},
		},
	}

	registry := tools.NewRegistry()
	sessionMgr := session.NewManager(session.NewMemoryStore(0), nil)
	strategy := &loop.ReActStrategy{}

	server := runtime.NewServer(config, nil, registry, sessionMgr, strategy,
		runtime.WithUI(true),
		runtime.WithAPIKey("secret-key"),
	)

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// Static assets should be accessible without auth
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("GET / error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for / without auth (static asset), got %d", resp.StatusCode)
	}

	// API endpoints should require auth
	resp2, err := http.Get(ts.URL + "/v1/agents")
	if err != nil {
		t.Fatalf("GET /v1/agents error: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 for /v1/agents without auth, got %d", resp2.StatusCode)
	}

	// API endpoints should work with auth
	req, _ := http.NewRequest("GET", ts.URL+"/v1/agents", nil)
	req.Header.Set("X-API-Key", "secret-key")
	resp3, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /v1/agents with key error: %v", err)
	}
	defer func() { _ = resp3.Body.Close() }()
	if resp3.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for /v1/agents with valid key, got %d", resp3.StatusCode)
	}
}
