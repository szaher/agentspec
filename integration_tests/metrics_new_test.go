package integration_tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/szaher/agentspec/internal/cost"
	"github.com/szaher/agentspec/internal/llm"
	"github.com/szaher/agentspec/internal/loop"
	"github.com/szaher/agentspec/internal/memory"
	"github.com/szaher/agentspec/internal/runtime"
	"github.com/szaher/agentspec/internal/session"
	"github.com/szaher/agentspec/internal/telemetry"
	"github.com/szaher/agentspec/internal/tools"
)

// TestMetricsIncludesNewMetricNames verifies that new metrics are exposed after invocation.
func TestMetricsIncludesNewMetricNames(t *testing.T) {
	metrics := telemetry.NewMetrics()
	config := &runtime.RuntimeConfig{
		PackageName: "test",
		Agents: []runtime.AgentConfig{
			{Name: "metrics-agent", Model: "claude-sonnet-4-20250514", Strategy: "react", MaxTurns: 5},
		},
	}

	mockClient := llm.NewMockClient(
		llm.MockResponse{
			Content:    "Hello metrics!",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 100, OutputTokens: 50},
		},
	)

	// Set up cost tracker with budget
	resetAt := time.Now().UTC().Add(24 * time.Hour)
	budgets := []cost.BudgetEntry{
		{
			AgentName:    "metrics-agent",
			Period:       "daily",
			LimitDollars: 10.0,
			UsedDollars:  0,
			ResetAt:      resetAt.Format(time.RFC3339),
			Paused:       false,
		},
	}
	costTracker := cost.New(budgets)

	registry := tools.NewRegistry()
	mgr := session.NewManager(session.NewMemoryStore(0, 0), memory.NewSlidingWindow(50))
	strategy := &loop.ReActStrategy{}

	srv := runtime.NewServer(config, mockClient, registry, mgr, strategy,
		runtime.WithMetrics(metrics),
		runtime.WithCostTracker(costTracker),
		runtime.WithNoAuth(true))

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	// Invoke the agent to generate metrics
	body := `{"message":"test"}`
	invokeReq, err := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/metrics-agent/invoke", strings.NewReader(body))
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

	// Decode invoke response
	var invokeResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&invokeResp); err != nil {
		t.Fatalf("decode invoke response: %v", err)
	}

	// Update budget usage in metrics (this would normally happen in the server)
	ratio := costTracker.BudgetUsageRatio("metrics-agent", "daily")
	metrics.SetBudgetUsage("metrics-agent", "daily", ratio)

	// Scrape the metrics endpoint
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

	// Verify new metric names are present
	expectedMetrics := []string{
		"agentspec_cost_dollars_total",
		"agentspec_budget_usage_ratio",
		"agentspec_fallback_total",
		"agentspec_guardrail_violations_total",
	}

	for _, metricName := range expectedMetrics {
		if !strings.Contains(metricsBody, metricName) {
			t.Errorf("metrics missing expected metric: %s", metricName)
		}
	}

	// Verify cost metric has data
	if !strings.Contains(metricsBody, `agentspec_cost_dollars_total{agent="metrics-agent"`) {
		t.Error("metrics missing cost data for metrics-agent")
	}

	// Verify budget usage metric has data
	if !strings.Contains(metricsBody, `agentspec_budget_usage_ratio{agent="metrics-agent",period="daily"}`) {
		t.Error("metrics missing budget usage data for metrics-agent")
	}
}
