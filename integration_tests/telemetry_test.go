package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/szaher/agentspec/internal/auth"
	"github.com/szaher/agentspec/internal/eviction"
	"github.com/szaher/agentspec/internal/llm"
	"github.com/szaher/agentspec/internal/loop"
	"github.com/szaher/agentspec/internal/memory"
	"github.com/szaher/agentspec/internal/runtime"
	"github.com/szaher/agentspec/internal/session"
	"github.com/szaher/agentspec/internal/telemetry"
	"github.com/szaher/agentspec/internal/tools"
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
	mgr := session.NewManager(session.NewMemoryStore(0, 0), memory.NewSlidingWindow(50))
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
	mgr := session.NewManager(session.NewMemoryStore(0, 0), memory.NewSlidingWindow(50))
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

func TestRateLimiterEviction(t *testing.T) {
	// T013A: Verify that rate limiter evicts stale buckets after TTL expires.
	//
	// Strategy: create a RateLimiter with a very short TTL and eviction interval,
	// populate it with many unique clients, wait for the TTL + eviction interval
	// to pass, then verify the buckets have been reclaimed by confirming that
	// Allow() for the same keys creates fresh buckets with a full burst allowance.

	const (
		numClients = 60
		burst      = 3
	)

	policy := eviction.Policy{
		MaxEntries:       100,
		TTL:              100 * time.Millisecond,
		EvictionInterval: 50 * time.Millisecond,
	}

	rl := auth.NewRateLimiter(auth.RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             burst,
	}, policy)

	// Start the background eviction goroutine.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rl.Start(ctx)

	// Phase 1: Exhaust the burst for many unique clients.
	for i := 0; i < numClients; i++ {
		key := fmt.Sprintf("client-%d", i)
		for j := 0; j < burst; j++ {
			if !rl.Allow(key) {
				t.Fatalf("phase 1: expected Allow(%q) attempt %d to succeed", key, j)
			}
		}
		// One more call should be denied — burst is exhausted.
		if rl.Allow(key) {
			t.Fatalf("phase 1: expected Allow(%q) to be denied after exhausting burst", key)
		}
	}

	// Phase 2: Wait long enough for TTL to expire and eviction to run.
	// TTL is 100ms and eviction runs every 50ms, so 250ms gives ample margin.
	time.Sleep(250 * time.Millisecond)

	// Phase 3: Verify that buckets have been evicted.
	// If eviction worked, calling Allow() for the same keys should create brand-new
	// buckets with a full burst, so all calls should succeed.
	for i := 0; i < numClients; i++ {
		key := fmt.Sprintf("client-%d", i)
		if !rl.Allow(key) {
			t.Errorf("phase 3: expected Allow(%q) to succeed after eviction (bucket should be fresh)", key)
		}
	}

	// Phase 4: Double-check that the fresh buckets have full burst available.
	// We already consumed 1 token per key above, so (burst - 1) more should succeed.
	for i := 0; i < numClients; i++ {
		key := fmt.Sprintf("client-%d", i)
		for j := 1; j < burst; j++ {
			if !rl.Allow(key) {
				t.Errorf("phase 4: expected Allow(%q) attempt %d to succeed (fresh burst)", key, j)
			}
		}
	}
}
