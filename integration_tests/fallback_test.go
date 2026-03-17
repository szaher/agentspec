package integration_tests

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/szaher/agentspec/internal/llm"
)

// TestFallbackSuccessOnSecondClient verifies fallback to a working client.
func TestFallbackSuccessOnSecondClient(t *testing.T) {
	// First client always errors
	client1 := llm.NewMockClient(
		llm.MockResponse{
			Error: errors.New("client 1 failed"),
		},
	)

	// Second client succeeds
	client2 := llm.NewMockClient(
		llm.MockResponse{
			Content:    "Success from client 2",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 100, OutputTokens: 50},
		},
	)

	fallbackClient := llm.NewFallbackClient(
		[]llm.Client{client1, client2},
		[]string{"model-1", "model-2"},
		slog.Default(),
	)

	req := llm.ChatRequest{
		Model:     "model-1",
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "test"}},
		MaxTokens: 100,
	}

	resp, err := fallbackClient.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}

	if resp.Content != "Success from client 2" {
		t.Errorf("expected content from client 2, got: %q", resp.Content)
	}
}

// TestFallbackAllFail verifies error when all clients fail.
func TestFallbackAllFail(t *testing.T) {
	// All clients fail
	client1 := llm.NewMockClient(
		llm.MockResponse{
			Error: errors.New("client 1 failed"),
		},
	)

	client2 := llm.NewMockClient(
		llm.MockResponse{
			Error: errors.New("client 2 failed"),
		},
	)

	client3 := llm.NewMockClient(
		llm.MockResponse{
			Error: errors.New("client 3 failed"),
		},
	)

	fallbackClient := llm.NewFallbackClient(
		[]llm.Client{client1, client2, client3},
		[]string{"model-1", "model-2", "model-3"},
		slog.Default(),
	)

	req := llm.ChatRequest{
		Model:     "model-1",
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "test"}},
		MaxTokens: 100,
	}

	resp, err := fallbackClient.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when all clients fail")
	}

	if resp != nil {
		t.Errorf("expected nil response when all clients fail, got: %v", resp)
	}

	// Verify error message mentions all failures
	errMsg := err.Error()
	if !strings.Contains(errMsg, "all LLM clients failed") {
		t.Errorf("expected error to mention all clients failed, got: %s", errMsg)
	}
}

// TestFallbackFirstSucceeds verifies no fallback when first client works.
func TestFallbackFirstSucceeds(t *testing.T) {
	// First client succeeds
	client1 := llm.NewMockClient(
		llm.MockResponse{
			Content:    "Success from client 1",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 100, OutputTokens: 50},
		},
	)

	// Second client would fail (but shouldn't be called)
	client2 := llm.NewMockClient(
		llm.MockResponse{
			Error: errors.New("client 2 should not be called"),
		},
	)

	fallbackClient := llm.NewFallbackClient(
		[]llm.Client{client1, client2},
		[]string{"model-1", "model-2"},
		slog.Default(),
	)

	req := llm.ChatRequest{
		Model:     "model-1",
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "test"}},
		MaxTokens: 100,
	}

	resp, err := fallbackClient.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if resp.Content != "Success from client 1" {
		t.Errorf("expected content from client 1, got: %q", resp.Content)
	}
}

// TestFallbackOnFallbackCallback verifies the OnFallback callback is invoked.
func TestFallbackOnFallbackCallback(t *testing.T) {
	client1 := llm.NewMockClient(
		llm.MockResponse{
			Error: errors.New("client 1 failed"),
		},
	)

	client2 := llm.NewMockClient(
		llm.MockResponse{
			Content:    "Success from client 2",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 100, OutputTokens: 50},
		},
	)

	var callbackInvoked bool
	var fromModel, toModel string

	fallbackClient := llm.NewFallbackClient(
		[]llm.Client{client1, client2},
		[]string{"model-1", "model-2"},
		slog.Default(),
	)

	fallbackClient.OnFallback = func(from, to string) {
		callbackInvoked = true
		fromModel = from
		toModel = to
	}

	req := llm.ChatRequest{
		Model:     "model-1",
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "test"}},
		MaxTokens: 100,
	}

	_, err := fallbackClient.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if !callbackInvoked {
		t.Error("expected OnFallback to be invoked")
	}

	if fromModel != "model-1" {
		t.Errorf("expected fromModel 'model-1', got %q", fromModel)
	}

	if toModel != "model-2" {
		t.Errorf("expected toModel 'model-2', got %q", toModel)
	}
}

// TestFallbackStream verifies ChatStream fallback behavior.
func TestFallbackStream(t *testing.T) {
	// First client fails on stream
	client1 := llm.NewMockClient(
		llm.MockResponse{
			Error: errors.New("stream failed"),
		},
	)

	// Second client succeeds
	client2 := llm.NewMockClient(
		llm.MockResponse{
			Content:    "Stream success",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 100, OutputTokens: 50},
		},
	)

	fallbackClient := llm.NewFallbackClient(
		[]llm.Client{client1, client2},
		[]string{"model-1", "model-2"},
		slog.Default(),
	)

	req := llm.ChatRequest{
		Model:     "model-1",
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: "test"}},
		MaxTokens: 100,
	}

	ch, err := fallbackClient.ChatStream(context.Background(), req)
	if err != nil {
		t.Fatalf("expected fallback stream to succeed, got error: %v", err)
	}

	// Read all events from channel
	var events []llm.StreamEvent
	for event := range ch {
		events = append(events, event)
	}

	if len(events) == 0 {
		t.Fatal("expected stream events, got none")
	}
}
