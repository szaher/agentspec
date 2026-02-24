package llm

import (
	"context"
	"fmt"
	"sync"
)

// MockResponse configures a single response from the mock client.
type MockResponse struct {
	Content    string
	ToolCalls  []ToolCall
	StopReason StopReason
	Usage      TokenUsage
	Error      error
}

// MockClient is a configurable mock LLM client for testing.
type MockClient struct {
	mu        sync.Mutex
	responses []MockResponse
	callIndex int
	calls     []ChatRequest
}

// NewMockClient creates a mock client with a sequence of responses.
// Responses are returned in order; if exhausted, the last response repeats.
func NewMockClient(responses ...MockResponse) *MockClient {
	return &MockClient{responses: responses}
}

// Chat returns the next configured response.
func (m *MockClient) Chat(_ context.Context, req ChatRequest) (*ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = append(m.calls, req)

	if len(m.responses) == 0 {
		return nil, fmt.Errorf("mock: no responses configured")
	}

	idx := m.callIndex
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	} else {
		m.callIndex++
	}

	resp := m.responses[idx]
	if resp.Error != nil {
		return nil, resp.Error
	}

	return &ChatResponse{
		Content:    resp.Content,
		ToolCalls:  resp.ToolCalls,
		StopReason: resp.StopReason,
		Usage:      resp.Usage,
	}, nil
}

// ChatStream returns streaming events for the next configured response.
func (m *MockClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	resp, err := m.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan StreamEvent, 10)
	go func() {
		defer close(ch)

		if resp.Content != "" {
			ch <- StreamEvent{Type: "text", Text: resp.Content}
		}
		for i := range resp.ToolCalls {
			ch <- StreamEvent{Type: "tool_call_start", ToolCall: &resp.ToolCalls[i]}
		}
		ch <- StreamEvent{Type: "done", Response: resp}
	}()

	return ch, nil
}

// Calls returns all requests made to the mock client.
func (m *MockClient) Calls() []ChatRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]ChatRequest(nil), m.calls...)
}

// Reset clears call history and resets the response index.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callIndex = 0
	m.calls = nil
}
