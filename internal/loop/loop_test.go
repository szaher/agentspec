package loop

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// ---------- mock ToolExecutor ----------

type mockToolExecutor struct {
	results map[string]string // tool name → output
	err     error
}

func newMockToolExecutor(results map[string]string) *mockToolExecutor {
	return &mockToolExecutor{results: results}
}

func (m *mockToolExecutor) Execute(_ context.Context, call llm.ToolCall) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if out, ok := m.results[call.Name]; ok {
		return out, nil
	}
	return "ok", nil
}

func (m *mockToolExecutor) ExecuteConcurrent(_ context.Context, calls []llm.ToolCall) []llm.ToolResult {
	var results []llm.ToolResult
	for _, c := range calls {
		out, err := m.Execute(context.Background(), c)
		if err != nil {
			results = append(results, llm.ToolResult{
				ToolUseID: c.ID,
				Content:   err.Error(),
				IsError:   true,
			})
		} else {
			results = append(results, llm.ToolResult{
				ToolUseID: c.ID,
				Content:   out,
			})
		}
	}
	return results
}

// ---------- Strategy.Name() tests ----------

func TestStrategyNames(t *testing.T) {
	tests := []struct {
		strategy Strategy
		want     string
	}{
		{&ReActStrategy{}, "react"},
		{&PlanExecuteStrategy{}, "plan-and-execute"},
		{&ReflexionStrategy{}, "reflexion"},
		{&MapReduceStrategy{}, "map-reduce"},
		{&RouterStrategy{}, "router"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.strategy.Name(); got != tc.want {
				t.Errorf("Name() = %q, want %q", got, tc.want)
			}
		})
	}
}

// ---------- ReAct strategy ----------

func TestReAct_SimpleCompletion(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "The answer is 42.",
		StopReason: llm.StopEndTurn,
		Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
	})
	tools := newMockToolExecutor(nil)

	s := &ReActStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:    "What is the meaning of life?",
		MaxTurns: 5,
	}, mock, tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Output != "The answer is 42." {
		t.Errorf("Output = %q, want %q", resp.Output, "The answer is 42.")
	}
	if resp.Turns != 1 {
		t.Errorf("Turns = %d, want 1", resp.Turns)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls = %d, want 0", len(resp.ToolCalls))
	}
	if resp.Tokens.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", resp.Tokens.InputTokens)
	}
}

func TestReAct_ToolCallFlow(t *testing.T) {
	mock := llm.NewMockClient(
		// Turn 1: LLM requests a tool call
		llm.MockResponse{
			Content: "I need to look that up.",
			ToolCalls: []llm.ToolCall{
				{ID: "tc1", Name: "search", Input: map[string]interface{}{"q": "weather"}},
			},
			StopReason: llm.StopToolUse,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 15},
		},
		// Turn 2: LLM returns final answer
		llm.MockResponse{
			Content:    "The weather is sunny.",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 20, OutputTokens: 10},
		},
	)

	tools := newMockToolExecutor(map[string]string{
		"search": "sunny, 72F",
	})

	s := &ReActStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:    "What is the weather?",
		MaxTurns: 5,
	}, mock, tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Output != "The weather is sunny." {
		t.Errorf("Output = %q, want %q", resp.Output, "The weather is sunny.")
	}
	if resp.Turns != 2 {
		t.Errorf("Turns = %d, want 2", resp.Turns)
	}
	if len(resp.ToolCalls) != 1 {
		t.Errorf("ToolCalls = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].ToolName != "search" {
		t.Errorf("ToolCalls[0].ToolName = %q, want %q", resp.ToolCalls[0].ToolName, "search")
	}
	if resp.ToolCalls[0].Output != "sunny, 72F" {
		t.Errorf("ToolCalls[0].Output = %q, want %q", resp.ToolCalls[0].Output, "sunny, 72F")
	}
	// Token aggregation
	if resp.Tokens.InputTokens != 30 {
		t.Errorf("InputTokens = %d, want 30", resp.Tokens.InputTokens)
	}
	if resp.Tokens.OutputTokens != 25 {
		t.Errorf("OutputTokens = %d, want 25", resp.Tokens.OutputTokens)
	}
}

func TestReAct_MaxTurnsLimit(t *testing.T) {
	// Every response triggers another tool call, so the loop should stop at MaxTurns
	mock := llm.NewMockClient(llm.MockResponse{
		Content: "calling tool",
		ToolCalls: []llm.ToolCall{
			{ID: "tc", Name: "loop_tool", Input: map[string]interface{}{}},
		},
		StopReason: llm.StopToolUse,
		Usage:      llm.TokenUsage{InputTokens: 5, OutputTokens: 5},
	})

	tools := newMockToolExecutor(map[string]string{"loop_tool": "ok"})

	s := &ReActStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:    "Keep going",
		MaxTurns: 3,
	}, mock, tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Turns != 3 {
		t.Errorf("Turns = %d, want 3", resp.Turns)
	}
	if len(resp.ToolCalls) != 3 {
		t.Errorf("ToolCalls = %d, want 3", len(resp.ToolCalls))
	}
}

func TestReAct_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	mock := llm.NewMockClient(llm.MockResponse{
		Error: ctx.Err(),
	})
	tools := newMockToolExecutor(nil)

	s := &ReActStrategy{}
	_, err := s.Execute(ctx, Invocation{
		Input:    "hello",
		MaxTurns: 5,
	}, mock, tools, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestReAct_StreamCallback(t *testing.T) {
	mock := llm.NewMockClient(
		// Turn 1: tool call
		llm.MockResponse{
			Content: "",
			ToolCalls: []llm.ToolCall{
				{ID: "tc1", Name: "greet", Input: map[string]interface{}{}},
			},
			StopReason: llm.StopToolUse,
			Usage:      llm.TokenUsage{InputTokens: 5, OutputTokens: 5},
		},
		// Turn 2: final
		llm.MockResponse{
			Content:    "done",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 5, OutputTokens: 5},
		},
	)
	tools := newMockToolExecutor(map[string]string{"greet": "hi"})

	var events []llm.StreamEvent
	cb := func(e llm.StreamEvent) {
		events = append(events, e)
	}

	s := &ReActStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:    "test",
		MaxTurns: 5,
	}, mock, tools, cb)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Output != "done" {
		t.Errorf("Output = %q, want %q", resp.Output, "done")
	}
	// Should have received a tool_call_end event from the callback
	found := false
	for _, e := range events {
		if e.Type == "tool_call_end" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a tool_call_end event in stream callbacks")
	}
}

func TestReAct_TokenBudgetExceeded(t *testing.T) {
	// Token budget is very small so the loop should return with an error in the response
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "should not reach here",
		StopReason: llm.StopEndTurn,
		Usage:      llm.TokenUsage{InputTokens: 100, OutputTokens: 100},
	})
	tools := newMockToolExecutor(nil)

	s := &ReActStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:       "test",
		MaxTurns:    5,
		MaxTokens:   4096,
		TokenBudget: 1, // extremely small budget
	}, mock, tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Error == "" {
		t.Error("expected token budget error in response.Error, got empty string")
	}
}

// ---------- PlanExecute strategy ----------

func TestPlanExecute_PlanGeneration(t *testing.T) {
	mock := llm.NewMockClient(
		// Plan phase response
		llm.MockResponse{
			Content:    "STEP 1: Do something\nSTEP 2: Do another thing",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 20, OutputTokens: 20},
		},
		// Step 1 execution
		llm.MockResponse{
			Content:    "Step 1 done",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 10},
		},
		// Step 2 execution
		llm.MockResponse{
			Content:    "Step 2 complete - all done",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 10},
		},
	)
	tools := newMockToolExecutor(nil)

	s := &PlanExecuteStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:    "build a house",
		MaxTurns: 10,
	}, mock, tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Output != "Step 2 complete - all done" {
		t.Errorf("Output = %q, want %q", resp.Output, "Step 2 complete - all done")
	}
	// 1 plan turn + 2 step turns = 3
	if resp.Turns != 3 {
		t.Errorf("Turns = %d, want 3", resp.Turns)
	}
}

// ---------- Reflexion strategy ----------

func TestReflexion_BasicReflectionLoop(t *testing.T) {
	mock := llm.NewMockClient(
		// Initial response
		llm.MockResponse{
			Content:    "Initial answer",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 10},
		},
		// Self-critique: SATISFACTORY, so it stops
		llm.MockResponse{
			Content:    "SATISFACTORY",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
		},
	)
	tools := newMockToolExecutor(nil)

	s := &ReflexionStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:    "Write a poem",
		MaxTurns: 5,
	}, mock, tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Output != "Initial answer" {
		t.Errorf("Output = %q, want %q", resp.Output, "Initial answer")
	}
	// 1 initial + 1 critique = 2
	if resp.Turns != 2 {
		t.Errorf("Turns = %d, want 2", resp.Turns)
	}
}

func TestReflexion_WithImprovement(t *testing.T) {
	mock := llm.NewMockClient(
		// Initial response
		llm.MockResponse{
			Content:    "First draft",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 10},
		},
		// Critique: not satisfactory
		llm.MockResponse{
			Content:    "Needs more detail",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 10},
		},
		// Improved response
		llm.MockResponse{
			Content:    "Improved draft with detail",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 10},
		},
		// Second critique: satisfactory
		llm.MockResponse{
			Content:    "This is SATISFACTORY now",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
		},
	)
	tools := newMockToolExecutor(nil)

	s := &ReflexionStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:    "Write something",
		MaxTurns: 5,
	}, mock, tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Output != "Improved draft with detail" {
		t.Errorf("Output = %q, want %q", resp.Output, "Improved draft with detail")
	}
}

// ---------- DelegationRouter ----------

func TestDelegationRouter_MatchingCondition(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "billing-agent",
		StopReason: llm.StopEndTurn,
		Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
	})

	router := &DelegationRouter{
		Rules: []DelegationRule{
			{TargetAgent: "billing-agent", Condition: "billing questions"},
			{TargetAgent: "support-agent", Condition: "technical support"},
		},
		LLMClient: mock,
		Model:     "test-model",
	}

	result, err := router.Evaluate(context.Background(), "How much is my bill?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ShouldDelegate {
		t.Fatal("expected ShouldDelegate = true")
	}
	if result.TargetAgent != "billing-agent" {
		t.Errorf("TargetAgent = %q, want %q", result.TargetAgent, "billing-agent")
	}
}

func TestDelegationRouter_NoMatchingRules(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "NONE",
		StopReason: llm.StopEndTurn,
		Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 5},
	})

	router := &DelegationRouter{
		Rules: []DelegationRule{
			{TargetAgent: "billing-agent", Condition: "billing questions"},
		},
		LLMClient: mock,
		Model:     "test-model",
	}

	result, err := router.Evaluate(context.Background(), "What time is it?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldDelegate {
		t.Fatal("expected ShouldDelegate = false")
	}
}

func TestDelegationRouter_EmptyRules(t *testing.T) {
	router := &DelegationRouter{
		Rules: nil,
	}
	result, err := router.Evaluate(context.Background(), "anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ShouldDelegate {
		t.Fatal("expected ShouldDelegate = false for empty rules")
	}
}

// ---------- MapReduceStrategy ----------

func TestMapReduce_SplitAndMerge(t *testing.T) {
	// Input with multiple paragraphs separated by double newlines
	input := "Paragraph one content.\n\nParagraph two content.\n\nParagraph three content."

	callCount := 0
	mock := llm.NewMockClient(
		// Chunk 1 (ReAct)
		llm.MockResponse{
			Content:    "Result 1",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 5, OutputTokens: 5},
		},
		// Chunk 2 (ReAct)
		llm.MockResponse{
			Content:    "Result 2",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 5, OutputTokens: 5},
		},
		// Chunk 3 (ReAct)
		llm.MockResponse{
			Content:    "Result 3",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 5, OutputTokens: 5},
		},
		// Reduce phase
		llm.MockResponse{
			Content:    "Merged result",
			StopReason: llm.StopEndTurn,
			Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 10},
		},
	)
	_ = callCount

	tools := newMockToolExecutor(nil)

	s := &MapReduceStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:    input,
		MaxTurns: 10,
	}, mock, tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Output != "Merged result" {
		t.Errorf("Output = %q, want %q", resp.Output, "Merged result")
	}
	// 3 chunks + plan + reduce = 5
	if resp.Turns != 5 {
		t.Errorf("Turns = %d, want 5", resp.Turns)
	}
}

func TestMapReduce_SingleChunkFallsBackToReAct(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Content:    "Direct answer",
		StopReason: llm.StopEndTurn,
		Usage:      llm.TokenUsage{InputTokens: 10, OutputTokens: 10},
	})
	tools := newMockToolExecutor(nil)

	s := &MapReduceStrategy{}
	resp, err := s.Execute(context.Background(), Invocation{
		Input:    "Short single paragraph input",
		MaxTurns: 5,
	}, mock, tools, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Output != "Direct answer" {
		t.Errorf("Output = %q, want %q", resp.Output, "Direct answer")
	}
}

func TestMapReduce_SplitInput(t *testing.T) {
	s := &MapReduceStrategy{}

	// Test paragraph splitting (default)
	chunks := s.splitInput("A\n\nB\n\nC")
	if len(chunks) != 3 {
		t.Errorf("splitInput paragraphs: got %d chunks, want 3", len(chunks))
	}

	// Test chunk size splitting
	s2 := &MapReduceStrategy{ChunkSize: 5}
	chunks2 := s2.splitInput("HelloWorld!")
	if len(chunks2) != 3 {
		t.Errorf("splitInput chunkSize=5: got %d chunks, want 3", len(chunks2))
	}

	// Empty double-newline sections should be filtered
	chunks3 := s.splitInput("\n\n\n\n")
	if len(chunks3) != 1 {
		t.Errorf("splitInput empty: got %d chunks, want 1 (original input)", len(chunks3))
	}
}

// ---------- Invocation and Response structs ----------

func TestResponseDuration(t *testing.T) {
	start := time.Now()
	resp := &Response{
		Output:   "test",
		Turns:    1,
		Duration: time.Since(start),
	}
	if resp.Duration < 0 {
		t.Error("Duration should be non-negative")
	}
}

func TestReAct_LLMError(t *testing.T) {
	mock := llm.NewMockClient(llm.MockResponse{
		Error: fmt.Errorf("API rate limit exceeded"),
	})
	tools := newMockToolExecutor(nil)

	s := &ReActStrategy{}
	_, err := s.Execute(context.Background(), Invocation{
		Input:    "test",
		MaxTurns: 3,
	}, mock, tools, nil)
	if err == nil {
		t.Fatal("expected error from LLM failure, got nil")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error %q should contain 'rate limit'", err.Error())
	}
}
