package integration_tests

import (
	"context"
	"testing"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
)

func TestDelegationRouterEvaluate(t *testing.T) {
	mockClient := llm.NewMockClient(
		llm.MockResponse{
			Content:    "billing",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 5, OutputTokens: 5},
		},
	)

	router := &loop.DelegationRouter{
		Rules: []loop.DelegationRule{
			{TargetAgent: "billing", Condition: "user asks about billing"},
			{TargetAgent: "support", Condition: "user asks for help"},
		},
		LLMClient: mockClient,
		Model:     "test-model",
	}

	result, err := router.Evaluate(context.Background(), "How much is my bill?")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	if !result.ShouldDelegate {
		t.Fatal("expected delegation to billing agent")
	}
	if result.TargetAgent != "billing" {
		t.Errorf("expected target agent 'billing', got %q", result.TargetAgent)
	}
}

func TestDelegationRouterNoMatch(t *testing.T) {
	mockClient := llm.NewMockClient(
		llm.MockResponse{
			Content:    "NONE",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 5, OutputTokens: 5},
		},
	)

	router := &loop.DelegationRouter{
		Rules: []loop.DelegationRule{
			{TargetAgent: "billing", Condition: "user asks about billing"},
		},
		LLMClient: mockClient,
		Model:     "test-model",
	}

	result, err := router.Evaluate(context.Background(), "What's the weather?")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	if result.ShouldDelegate {
		t.Fatal("expected no delegation")
	}
}

func TestDelegationRouterNoRules(t *testing.T) {
	router := &loop.DelegationRouter{
		Rules: nil,
	}

	result, err := router.Evaluate(context.Background(), "anything")
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	if result.ShouldDelegate {
		t.Fatal("expected no delegation with no rules")
	}
}
