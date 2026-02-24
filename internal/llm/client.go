// Package llm defines the LLM client abstraction for the AgentSpec runtime.
package llm

import (
	"context"
)

// Role represents a message sender role.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// StopReason indicates why the model stopped generating.
type StopReason string

const (
	StopEndTurn      StopReason = "end_turn"
	StopMaxTokens    StopReason = "max_tokens"
	StopToolUse      StopReason = "tool_use"
	StopStopSequence StopReason = "stop_sequence"
)

// Message represents a single message in a conversation.
type Message struct {
	Role       Role        `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ToolDefinition describes a tool available to the LLM.
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolCall represents the LLM requesting a tool invocation.
type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult represents the result of a tool invocation sent back to the LLM.
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// TokenUsage tracks token consumption for a single LLM call.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheRead    int `json:"cache_read"`
	CacheWrite   int `json:"cache_write"`
}

// Total returns the sum of all token fields.
func (u TokenUsage) Total() int {
	return u.InputTokens + u.OutputTokens
}

// ChatRequest contains parameters for an LLM chat call.
type ChatRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	System      string           `json:"system,omitempty"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	MaxTokens   int              `json:"max_tokens"`
	Temperature *float64         `json:"temperature,omitempty"`
}

// ChatResponse contains the LLM's response to a chat request.
type ChatResponse struct {
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	StopReason StopReason `json:"stop_reason"`
	Usage      TokenUsage `json:"usage"`
}

// StreamEvent represents an incremental event during streaming.
type StreamEvent struct {
	Type string `json:"type"` // "text", "tool_call_start", "tool_call_delta", "tool_call_end", "done", "error"

	// Text events
	Text string `json:"text,omitempty"`

	// Tool call events
	ToolCall *ToolCall `json:"tool_call,omitempty"`

	// Done events
	Response *ChatResponse `json:"response,omitempty"`

	// Error events
	Error error `json:"-"`
}

// Client is the interface for LLM interactions.
type Client interface {
	// Chat sends a request and returns the complete response.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatStream sends a request and returns a channel of streaming events.
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
}
