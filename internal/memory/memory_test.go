package memory

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/szaher/designs/agentz/internal/llm"
)

// mockLLMClient implements llm.Client for testing the Summary memory store.
type mockLLMClient struct {
	mu        sync.Mutex
	responses []llm.ChatResponse
	callIndex int
	calls     []llm.ChatRequest
}

func newMockLLMClient(responses ...llm.ChatResponse) *mockLLMClient {
	return &mockLLMClient{responses: responses}
}

func (m *mockLLMClient) Chat(_ context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, req)

	if len(m.responses) == 0 {
		return nil, fmt.Errorf("no responses configured")
	}

	idx := m.callIndex
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	} else {
		m.callIndex++
	}

	resp := m.responses[idx]
	return &resp, nil
}

func (m *mockLLMClient) ChatStream(_ context.Context, _ llm.ChatRequest) (<-chan llm.StreamEvent, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockLLMClient) getCalls() []llm.ChatRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]llm.ChatRequest(nil), m.calls...)
}

// --- SlidingWindow Tests ---

func TestNewSlidingWindow(t *testing.T) {
	t.Run("valid size", func(t *testing.T) {
		sw := NewSlidingWindow(10)
		if sw == nil {
			t.Fatal("expected non-nil SlidingWindow")
		}
		if sw.maxMessages != 10 {
			t.Errorf("expected maxMessages=10, got %d", sw.maxMessages)
		}
	})

	t.Run("zero size defaults to 50", func(t *testing.T) {
		sw := NewSlidingWindow(0)
		if sw.maxMessages != 50 {
			t.Errorf("expected maxMessages=50 for zero input, got %d", sw.maxMessages)
		}
	})

	t.Run("negative size defaults to 50", func(t *testing.T) {
		sw := NewSlidingWindow(-5)
		if sw.maxMessages != 50 {
			t.Errorf("expected maxMessages=50 for negative input, got %d", sw.maxMessages)
		}
	})
}

func TestSlidingWindowSaveAndLoad(t *testing.T) {
	ctx := context.Background()
	sw := NewSlidingWindow(10)

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "hello"},
		{Role: llm.RoleAssistant, Content: "hi there"},
	}

	if err := sw.Save(ctx, "session-1", messages); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := sw.Load(ctx, "session-1")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(loaded))
	}
	if loaded[0].Content != "hello" {
		t.Errorf("expected first message content 'hello', got %q", loaded[0].Content)
	}
	if loaded[1].Content != "hi there" {
		t.Errorf("expected second message content 'hi there', got %q", loaded[1].Content)
	}
}

func TestSlidingWindowEviction(t *testing.T) {
	ctx := context.Background()
	sw := NewSlidingWindow(3)

	// Save 5 messages, only last 3 should remain
	messages := make([]llm.Message, 5)
	for i := range messages {
		messages[i] = llm.Message{Role: llm.RoleUser, Content: fmt.Sprintf("msg-%d", i)}
	}

	if err := sw.Save(ctx, "s1", messages); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := sw.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 messages after eviction, got %d", len(loaded))
	}
	// Oldest messages (msg-0, msg-1) should be evicted
	if loaded[0].Content != "msg-2" {
		t.Errorf("expected first remaining message 'msg-2', got %q", loaded[0].Content)
	}
	if loaded[1].Content != "msg-3" {
		t.Errorf("expected second remaining message 'msg-3', got %q", loaded[1].Content)
	}
	if loaded[2].Content != "msg-4" {
		t.Errorf("expected third remaining message 'msg-4', got %q", loaded[2].Content)
	}
}

func TestSlidingWindowClear(t *testing.T) {
	ctx := context.Background()
	sw := NewSlidingWindow(10)

	messages := []llm.Message{
		{Role: llm.RoleUser, Content: "hello"},
	}
	if err := sw.Save(ctx, "s1", messages); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := sw.Clear(ctx, "s1"); err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}

	loaded, err := sw.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(loaded) != 0 {
		t.Errorf("expected 0 messages after Clear, got %d", len(loaded))
	}
}

func TestSlidingWindowSessionIsolation(t *testing.T) {
	ctx := context.Background()
	sw := NewSlidingWindow(10)

	msgs1 := []llm.Message{{Role: llm.RoleUser, Content: "session-1-msg"}}
	msgs2 := []llm.Message{{Role: llm.RoleUser, Content: "session-2-msg"}}

	if err := sw.Save(ctx, "s1", msgs1); err != nil {
		t.Fatalf("Save s1 error: %v", err)
	}
	if err := sw.Save(ctx, "s2", msgs2); err != nil {
		t.Fatalf("Save s2 error: %v", err)
	}

	loaded1, _ := sw.Load(ctx, "s1")
	loaded2, _ := sw.Load(ctx, "s2")

	if len(loaded1) != 1 || loaded1[0].Content != "session-1-msg" {
		t.Errorf("session 1 messages corrupted: %v", loaded1)
	}
	if len(loaded2) != 1 || loaded2[0].Content != "session-2-msg" {
		t.Errorf("session 2 messages corrupted: %v", loaded2)
	}

	// Clear one session should not affect the other
	if err := sw.Clear(ctx, "s1"); err != nil {
		t.Fatalf("Clear s1 error: %v", err)
	}

	loaded1After, _ := sw.Load(ctx, "s1")
	loaded2After, _ := sw.Load(ctx, "s2")

	if len(loaded1After) != 0 {
		t.Errorf("expected 0 messages for cleared s1, got %d", len(loaded1After))
	}
	if len(loaded2After) != 1 {
		t.Errorf("expected 1 message for s2 after clearing s1, got %d", len(loaded2After))
	}
}

func TestSlidingWindowMultipleSaves(t *testing.T) {
	ctx := context.Background()
	sw := NewSlidingWindow(5)

	// First save: 2 messages
	if err := sw.Save(ctx, "s1", []llm.Message{
		{Role: llm.RoleUser, Content: "a"},
		{Role: llm.RoleAssistant, Content: "b"},
	}); err != nil {
		t.Fatalf("first Save error: %v", err)
	}

	// Second save: 2 more messages
	if err := sw.Save(ctx, "s1", []llm.Message{
		{Role: llm.RoleUser, Content: "c"},
		{Role: llm.RoleAssistant, Content: "d"},
	}); err != nil {
		t.Fatalf("second Save error: %v", err)
	}

	loaded, _ := sw.Load(ctx, "s1")
	if len(loaded) != 4 {
		t.Fatalf("expected 4 accumulated messages, got %d", len(loaded))
	}

	// Third save: push over limit
	if err := sw.Save(ctx, "s1", []llm.Message{
		{Role: llm.RoleUser, Content: "e"},
		{Role: llm.RoleAssistant, Content: "f"},
	}); err != nil {
		t.Fatalf("third Save error: %v", err)
	}

	loaded, _ = sw.Load(ctx, "s1")
	if len(loaded) != 5 {
		t.Fatalf("expected 5 messages (window max), got %d", len(loaded))
	}
	// "a" should be evicted
	if loaded[0].Content != "b" {
		t.Errorf("expected first message 'b' after eviction, got %q", loaded[0].Content)
	}
}

func TestSlidingWindowLoadEmptySession(t *testing.T) {
	ctx := context.Background()
	sw := NewSlidingWindow(10)

	loaded, err := sw.Load(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 messages for nonexistent session, got %d", len(loaded))
	}
}

// --- Summary Tests ---

func TestNewSummary(t *testing.T) {
	mock := newMockLLMClient()

	t.Run("valid threshold", func(t *testing.T) {
		s := NewSummary(10, mock, "test-model")
		if s == nil {
			t.Fatal("expected non-nil Summary")
		}
		if s.threshold != 10 {
			t.Errorf("expected threshold=10, got %d", s.threshold)
		}
		if s.model != "test-model" {
			t.Errorf("expected model='test-model', got %q", s.model)
		}
	})

	t.Run("zero threshold defaults to 20", func(t *testing.T) {
		s := NewSummary(0, mock, "test-model")
		if s.threshold != 20 {
			t.Errorf("expected threshold=20 for zero input, got %d", s.threshold)
		}
	})
}

func TestSummarySaveBelowThreshold(t *testing.T) {
	ctx := context.Background()
	mock := newMockLLMClient()
	s := NewSummary(10, mock, "test-model")

	// Save fewer messages than threshold -- no summarization should occur
	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "hello"},
		{Role: llm.RoleAssistant, Content: "hi"},
	}

	if err := s.Save(ctx, "s1", msgs); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// LLM should not have been called
	if len(mock.getCalls()) != 0 {
		t.Errorf("expected no LLM calls below threshold, got %d", len(mock.getCalls()))
	}

	loaded, err := s.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("expected 2 messages, got %d", len(loaded))
	}
}

func TestSummarySaveAboveThreshold(t *testing.T) {
	ctx := context.Background()
	mock := newMockLLMClient(llm.ChatResponse{
		Content:    "Summary of earlier conversation.",
		StopReason: llm.StopEndTurn,
	})
	s := NewSummary(4, mock, "test-model")

	// Save 6 messages (above threshold of 4)
	msgs := make([]llm.Message, 6)
	for i := range msgs {
		if i%2 == 0 {
			msgs[i] = llm.Message{Role: llm.RoleUser, Content: fmt.Sprintf("user-msg-%d", i)}
		} else {
			msgs[i] = llm.Message{Role: llm.RoleAssistant, Content: fmt.Sprintf("assistant-msg-%d", i)}
		}
	}

	if err := s.Save(ctx, "s1", msgs); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// LLM should have been called for summarization
	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 LLM call for summarization, got %d", len(calls))
	}

	// After summarization: 1 summary + keepCount (threshold/2 = 2) recent messages
	loaded, err := s.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	// Expected: 1 summary message + 2 kept messages = 3
	if len(loaded) != 3 {
		t.Fatalf("expected 3 messages after summarization, got %d", len(loaded))
	}

	// First message should be the summary
	if loaded[0].Role != llm.RoleAssistant {
		t.Errorf("expected summary message role=assistant, got %q", loaded[0].Role)
	}
	if loaded[0].Content == "" {
		t.Error("expected non-empty summary content")
	}
}

func TestSummaryLoad(t *testing.T) {
	ctx := context.Background()
	mock := newMockLLMClient()
	s := NewSummary(20, mock, "test-model")

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "q1"},
		{Role: llm.RoleAssistant, Content: "a1"},
		{Role: llm.RoleUser, Content: "q2"},
	}

	if err := s.Save(ctx, "s1", msgs); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	loaded, err := s.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(loaded) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(loaded))
	}

	// Verify returned slice is a copy (modifying it doesn't affect internal state)
	loaded[0].Content = "modified"
	reloaded, _ := s.Load(ctx, "s1")
	if reloaded[0].Content != "q1" {
		t.Error("Load should return a copy, not the internal slice")
	}
}

func TestSummaryClear(t *testing.T) {
	ctx := context.Background()
	mock := newMockLLMClient()
	s := NewSummary(20, mock, "test-model")

	msgs := []llm.Message{
		{Role: llm.RoleUser, Content: "hello"},
	}
	if err := s.Save(ctx, "s1", msgs); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if err := s.Clear(ctx, "s1"); err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}

	loaded, err := s.Load(ctx, "s1")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(loaded) != 0 {
		t.Errorf("expected 0 messages after Clear, got %d", len(loaded))
	}
}

func TestStrategyConstants(t *testing.T) {
	if StrategySlidingWindow != "sliding_window" {
		t.Errorf("expected StrategySlidingWindow='sliding_window', got %q", StrategySlidingWindow)
	}
	if StrategySummary != "summary" {
		t.Errorf("expected StrategySummary='summary', got %q", StrategySummary)
	}
}

func TestSlidingWindowImplementsStore(t *testing.T) {
	var _ Store = (*SlidingWindow)(nil)
}

func TestSummaryImplementsStore(t *testing.T) {
	mock := newMockLLMClient()
	var _ Store = NewSummary(10, mock, "m")
}
