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
	"github.com/szaher/agentspec/internal/tools"
)

// TestBudgetExceeded verifies that invocations are blocked with 429 when budget is exceeded.
func TestBudgetExceeded(t *testing.T) {
	config := &runtime.RuntimeConfig{
		PackageName: "test",
		Agents: []runtime.AgentConfig{
			{Name: "budget-agent", Model: "claude-sonnet-4-20250514", Strategy: "react", MaxTurns: 5},
		},
	}

	// Create mock client that returns responses with token usage
	mockClient := llm.NewMockClient(
		llm.MockResponse{
			Content:    "Response 1",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 1000, OutputTokens: 500}, // Will cost ~$0.0105
		},
		llm.MockResponse{
			Content:    "Response 2",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 1000, OutputTokens: 500},
		},
	)

	// Create a cost tracker with a very low daily budget
	resetAt := time.Now().UTC().Add(24 * time.Hour)
	budgets := []cost.BudgetEntry{
		{
			AgentName:    "budget-agent",
			Period:       "daily",
			LimitDollars: 0.001, // Very low limit - first request will exceed it
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
		runtime.WithCostTracker(costTracker),
		runtime.WithNoAuth(true))

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"message":"test"}`

	// First invocation - should succeed but exceed the budget
	req1, err := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/budget-agent/invoke", strings.NewReader(body))
	if err != nil {
		t.Fatalf("creating request 1: %v", err)
	}
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatalf("request 1: %v", err)
	}
	defer func() { _ = resp1.Body.Close() }()

	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for first request, got %d", resp1.StatusCode)
	}

	// Decode response
	var invokeResp map[string]interface{}
	if err := json.NewDecoder(resp1.Body).Decode(&invokeResp); err != nil {
		t.Fatalf("decode response 1: %v", err)
	}

	// Second invocation - should be blocked with 429
	req2, err := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/budget-agent/invoke", strings.NewReader(body))
	if err != nil {
		t.Fatalf("creating request 2: %v", err)
	}
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("request 2: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()

	// Verify 429 status
	if resp2.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 for budget exceeded, got %d", resp2.StatusCode)
	}

	// Verify Retry-After header
	retryAfter := resp2.Header.Get("Retry-After")
	if retryAfter == "" {
		t.Error("expected Retry-After header, got none")
	}

	// Verify error response contains budget_exceeded
	respBody, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("read response 2 body: %v", err)
	}
	respStr := string(respBody)
	if !strings.Contains(respStr, "budget_exceeded") {
		t.Errorf("expected 'budget_exceeded' in response body, got: %s", respStr)
	}
}

// TestBudgetNotExceeded verifies that invocations succeed normally when budget is generous.
func TestBudgetNotExceeded(t *testing.T) {
	config := &runtime.RuntimeConfig{
		PackageName: "test",
		Agents: []runtime.AgentConfig{
			{Name: "budget-agent", Model: "claude-sonnet-4-20250514", Strategy: "react", MaxTurns: 5},
		},
	}

	// Create mock client
	mockClient := llm.NewMockClient(
		llm.MockResponse{
			Content:    "Success",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 100, OutputTokens: 50},
		},
	)

	// Create a cost tracker with a generous budget
	resetAt := time.Now().UTC().Add(24 * time.Hour)
	budgets := []cost.BudgetEntry{
		{
			AgentName:    "budget-agent",
			Period:       "daily",
			LimitDollars: 100.0, // Generous limit
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
		runtime.WithCostTracker(costTracker),
		runtime.WithNoAuth(true))

	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"message":"test"}`
	req, err := http.NewRequestWithContext(context.Background(), "POST", ts.URL+"/v1/agents/budget-agent/invoke", strings.NewReader(body))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with generous budget, got %d", resp.StatusCode)
	}

	var invokeResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&invokeResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if invokeResp["output"] != "Success" {
		t.Errorf("expected output 'Success', got %q", invokeResp["output"])
	}
}
