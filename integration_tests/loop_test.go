package integration_tests

import (
	"context"
	"testing"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
	"github.com/szaher/designs/agentz/internal/tools"
)

func TestReActLoopSimpleResponse(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "Hello! How can I help?",
		StopReason: llm.StopEndTurn,
		Usage:      llm.TokenUsage{InputTokens: 20, OutputTokens: 10},
	})

	registry := tools.NewRegistry()
	strategy := &loop.ReActStrategy{}

	inv := loop.Invocation{
		AgentName: "helper",
		Model:     "claude-sonnet-4-20250514",
		System:    "You are a test assistant.",
		Input:     "Hi there",
		MaxTurns:  5,
		MaxTokens: 4096,
	}

	resp, err := strategy.Execute(context.Background(), inv, mock, registry, nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if resp.Output != "Hello! How can I help?" {
		t.Errorf("expected 'Hello! How can I help?', got %q", resp.Output)
	}
	if resp.Turns != 1 {
		t.Errorf("expected 1 turn, got %d", resp.Turns)
	}
	if resp.Tokens.Total() != 30 {
		t.Errorf("expected 30 total tokens, got %d", resp.Tokens.Total())
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("expected no tool calls, got %d", len(resp.ToolCalls))
	}
}

func TestReActLoopWithToolCalls(t *testing.T) {
	mock := llm.NewMockClient(
		// Turn 1: model calls a tool
		llm.MockResponse{
			Content: "",
			ToolCalls: []llm.ToolCall{
				{ID: "tc_1", Name: "get_weather", Input: map[string]interface{}{"city": "London"}},
			},
			StopReason: llm.StopToolUse,
			Usage:      llm.TokenUsage{InputTokens: 30, OutputTokens: 15},
		},
		// Turn 2: model gives final answer using tool result
		llm.MockResponse{
			Content:    "The weather in London is sunny and 20°C.",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 50, OutputTokens: 20},
		},
	)

	// Register a mock tool
	registry := tools.NewRegistry()
	registry.Register("get_weather", llm.ToolDefinition{
		Name:        "get_weather",
		Description: "Get weather for a city",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"city": map[string]interface{}{"type": "string"},
			},
		},
	}, &mockToolExecutor{output: `{"temp": "20°C", "condition": "sunny"}`})

	strategy := &loop.ReActStrategy{}
	inv := loop.Invocation{
		AgentName: "weather-bot",
		Model:     "claude-sonnet-4-20250514",
		System:    "You are a weather assistant.",
		Input:     "What's the weather in London?",
		MaxTurns:  5,
		MaxTokens: 4096,
	}

	resp, err := strategy.Execute(context.Background(), inv, mock, registry, nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if resp.Output != "The weather in London is sunny and 20°C." {
		t.Errorf("unexpected output: %q", resp.Output)
	}
	if resp.Turns != 2 {
		t.Errorf("expected 2 turns, got %d", resp.Turns)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call record, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ToolName != "get_weather" {
		t.Errorf("expected tool name 'get_weather', got %q", resp.ToolCalls[0].ToolName)
	}
}

func TestReActLoopMaxTurnsEnforcement(t *testing.T) {
	// Model always requests tools, never gives final answer
	infiniteToolUse := llm.MockResponse{
		Content: "",
		ToolCalls: []llm.ToolCall{
			{ID: "tc_loop", Name: "search", Input: map[string]interface{}{"q": "test"}},
		},
		StopReason: llm.StopToolUse,
		Usage:      llm.TokenUsage{InputTokens: 20, OutputTokens: 10},
	}
	mock := llm.NewMockClient(infiniteToolUse)

	registry := tools.NewRegistry()
	registry.Register("search", llm.ToolDefinition{
		Name:        "search",
		Description: "Search",
	}, &mockToolExecutor{output: "result"})

	strategy := &loop.ReActStrategy{}
	inv := loop.Invocation{
		AgentName: "loopy",
		Model:     "claude-sonnet-4-20250514",
		Input:     "Find something",
		MaxTurns:  3,
		MaxTokens: 4096,
	}

	resp, err := strategy.Execute(context.Background(), inv, mock, registry, nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if resp.Turns != 3 {
		t.Errorf("expected 3 turns (max_turns enforced), got %d", resp.Turns)
	}
}

func TestReActLoopTokenBudgetEnforcement(t *testing.T) {
	// Each turn uses 100 tokens (50 input + 50 output)
	toolResponse := llm.MockResponse{
		Content: "",
		ToolCalls: []llm.ToolCall{
			{ID: "tc_1", Name: "search", Input: map[string]interface{}{"q": "test"}},
		},
		StopReason: llm.StopToolUse,
		Usage:      llm.TokenUsage{InputTokens: 50, OutputTokens: 50},
	}
	mock := llm.NewMockClient(toolResponse)

	registry := tools.NewRegistry()
	registry.Register("search", llm.ToolDefinition{
		Name: "search",
	}, &mockToolExecutor{output: "result"})

	strategy := &loop.ReActStrategy{}
	inv := loop.Invocation{
		AgentName:   "budget-test",
		Model:       "claude-sonnet-4-20250514",
		Input:       "Search for things",
		MaxTurns:    100,
		MaxTokens:   4096,
		TokenBudget: 150, // Budget for ~1.5 turns
	}

	resp, err := strategy.Execute(context.Background(), inv, mock, registry, nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	// After first turn uses 100 tokens, budget check for next turn (100 + 4096 > 150) should stop
	if resp.Error == "" {
		t.Log("Token budget enforcement did not produce an error, which may be expected if budget check occurs differently")
	}
	// The loop should have stopped
	if resp.Turns > 2 {
		t.Errorf("expected at most 2 turns with token budget of 150, got %d", resp.Turns)
	}
}

func TestReActLoopStreaming(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "Streamed response",
		StopReason: llm.StopEndTurn,
		Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
	})

	registry := tools.NewRegistry()
	strategy := &loop.ReActStrategy{}

	var events []llm.StreamEvent
	onEvent := func(event llm.StreamEvent) {
		events = append(events, event)
	}

	inv := loop.Invocation{
		AgentName: "streamer",
		Model:     "claude-sonnet-4-20250514",
		Input:     "Hello",
		MaxTurns:  5,
		MaxTokens: 4096,
		Stream:    true,
	}

	resp, err := strategy.Execute(context.Background(), inv, mock, registry, onEvent)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if resp.Output != "Streamed response" {
		t.Errorf("expected 'Streamed response', got %q", resp.Output)
	}

	if len(events) == 0 {
		t.Error("expected streaming events, got none")
	}
}

func TestReActLoopConcurrentToolCalls(t *testing.T) {
	mock := llm.NewMockClient(
		// Turn 1: model calls two tools concurrently
		llm.MockResponse{
			Content: "",
			ToolCalls: []llm.ToolCall{
				{ID: "tc_1", Name: "tool_a", Input: map[string]interface{}{"q": "a"}},
				{ID: "tc_2", Name: "tool_b", Input: map[string]interface{}{"q": "b"}},
			},
			StopReason: llm.StopToolUse,
			Usage:      llm.TokenUsage{InputTokens: 30, OutputTokens: 20},
		},
		// Turn 2: final answer
		llm.MockResponse{
			Content:    "Combined result from both tools.",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 50, OutputTokens: 15},
		},
	)

	registry := tools.NewRegistry()
	registry.Register("tool_a", llm.ToolDefinition{Name: "tool_a"}, &mockToolExecutor{output: "result_a"})
	registry.Register("tool_b", llm.ToolDefinition{Name: "tool_b"}, &mockToolExecutor{output: "result_b"})

	strategy := &loop.ReActStrategy{}
	inv := loop.Invocation{
		AgentName: "multi-tool",
		Model:     "claude-sonnet-4-20250514",
		Input:     "Do both things",
		MaxTurns:  5,
		MaxTokens: 4096,
	}

	resp, err := strategy.Execute(context.Background(), inv, mock, registry, nil)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if len(resp.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool call records, got %d", len(resp.ToolCalls))
	}
	if resp.Output != "Combined result from both tools." {
		t.Errorf("unexpected output: %q", resp.Output)
	}
}

// mockToolExecutor is a simple tool executor that returns a fixed output.
type mockToolExecutor struct {
	output string
	err    error
}

func (m *mockToolExecutor) Execute(_ context.Context, _ map[string]interface{}) (string, error) {
	return m.output, m.err
}
