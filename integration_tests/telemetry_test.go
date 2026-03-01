package integration_tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
	"github.com/szaher/designs/agentz/internal/memory"
	"github.com/szaher/designs/agentz/internal/runtime"
	"github.com/szaher/designs/agentz/internal/session"
	"github.com/szaher/designs/agentz/internal/telemetry"
	"github.com/szaher/designs/agentz/internal/tools"
)

func TestMetricsEndpoint(t *testing.T) {
	metrics := telemetry.NewMetrics()
	config := &runtime.RuntimeConfig{
		PackageName: "test",
		Agents: []runtime.AgentConfig{
			{Name: "test-agent", Model: "test-model", Strategy: "react", MaxTurns: 5},
		},
	}

	mockClient := llm.NewMockClient(
		llm.MockResponse{
			Content:    "Hello!",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 100, OutputTokens: 50},
		},
	)

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0), memory.NewSlidingWindow(50))
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, mockClient, registry, mgr, strategy,
		runtime.WithMetrics(metrics),
		runtime.WithNoAuth(true))

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Invoke the agent to generate metrics
	body := `{"message":"test"}`
	invokeReq, err := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/test-agent/invoke", strings.NewReader(body))
	if err != nil {
		t.Fatalf("creating invoke request: %v", err)
	}
	invokeReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(invokeReq)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Decode invoke response to verify it worked
	var invokeResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&invokeResp); err != nil {
		t.Fatalf("decode invoke response: %v", err)
	}
	if invokeResp["output"] != "Hello!" {
		t.Errorf("expected output 'Hello!', got %q", invokeResp["output"])
	}

	// Now scrape the metrics endpoint
	metricsReq, err := http.NewRequestWithContext(context.Background(), "GET", ts.URL+"/v1/metrics", nil)
	if err != nil {
		t.Fatalf("creating metrics request: %v", err)
	}
	metricsResp, err := http.DefaultClient.Do(metricsReq)
	if err != nil {
		t.Fatalf("get metrics: %v", err)
	}
	defer func() { _ = metricsResp.Body.Close() }()

	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("metrics: expected 200, got %d", metricsResp.StatusCode)
	}

	// Read metrics body
	metricsBytes, err := io.ReadAll(metricsResp.Body)
	if err != nil {
		t.Fatalf("read metrics body: %v", err)
	}
	metricsBody := string(metricsBytes)

	// Verify metrics contain expected counters
	if !strings.Contains(metricsBody, "agentspec_invocations_total") {
		t.Error("metrics missing agentspec_invocations_total")
	}
	if !strings.Contains(metricsBody, `agent="test-agent"`) {
		t.Error("metrics missing agent label")
	}
	if !strings.Contains(metricsBody, `status="completed"`) {
		t.Error("metrics missing completed status")
	}
	if !strings.Contains(metricsBody, "agentspec_tokens_total") {
		t.Error("metrics missing agentspec_tokens_total")
	}
	if !strings.Contains(metricsBody, "agentspec_invocation_duration_seconds") {
		t.Error("metrics missing duration histogram")
	}
}

func TestMetricsRecordToolCalls(t *testing.T) {
	metrics := telemetry.NewMetrics()

	// Record some tool calls
	metrics.RecordToolCall("bot", "search", "success")
	metrics.RecordToolCall("bot", "search", "success")
	metrics.RecordToolCall("bot", "search", "error")
	metrics.RecordToolCall("bot", "lookup", "success")

	// Verify via handler
	handler := metrics.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `agentspec_tool_calls_total{agent="bot",tool="search",status="success"} 2`) {
		t.Errorf("expected search success=2, got:\n%s", body)
	}
	if !strings.Contains(body, `agentspec_tool_calls_total{agent="bot",tool="search",status="error"} 1`) {
		t.Errorf("expected search error=1, got:\n%s", body)
	}
}

func TestRateLimiting(t *testing.T) {
	config := &runtime.RuntimeConfig{
		PackageName: "test",
		Agents: []runtime.AgentConfig{
			{Name: "rate-agent", Model: "test-model", Strategy: "react", MaxTurns: 5},
		},
	}

	// Create mock client with multiple responses (for multiple requests)
	responses := make([]llm.MockResponse, 20)
	for i := range responses {
		responses[i] = llm.MockResponse{
			Content:    "ok",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
		}
	}
	mockClient := llm.NewMockClient(responses...)

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0), memory.NewSlidingWindow(50))
	strategy := &loop.ReActStrategy{}

	// Set rate limit to 2 requests per second, burst of 2
	srv := runtime.NewServer(config, mockClient, registry, mgr, strategy,
		runtime.WithRateLimit(2, 2),
		runtime.WithNoAuth(true))

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"message":"test"}`

	// First 2 requests should succeed (burst)
	for i := 0; i < 2; i++ {
		req, err := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/rate-agent/invoke", strings.NewReader(body))
		if err != nil {
			t.Fatalf("request %d: creating request: %v", i, err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, resp.StatusCode)
		}
	}

	// 3rd request should be rate limited (burst exhausted)
	rateLimitReq, err := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/rate-agent/invoke", strings.NewReader(body))
	if err != nil {
		t.Fatalf("creating rate-limited request: %v", err)
	}
	rateLimitReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(rateLimitReq)
	if err != nil {
		t.Fatalf("rate-limited request: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 rate limited, got %d", resp.StatusCode)
	}
}
