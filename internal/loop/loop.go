// Package loop implements agentic loop strategies for the AgentSpec runtime.
package loop

import (
	"context"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// ToolCallRecord is an audit record of a single tool invocation.
type ToolCallRecord struct {
	ID       string                 `json:"id"`
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
	Output   string                 `json:"output"`
	Duration time.Duration          `json:"duration"`
	Error    string                 `json:"error,omitempty"`
}

// Invocation represents a single agent invocation request.
type Invocation struct {
	AgentName   string            `json:"agent_name"`
	Model       string            `json:"model"`
	System      string            `json:"system"`
	Input       string            `json:"input"`
	Messages    []llm.Message     `json:"messages,omitempty"` // Existing conversation context
	Variables   map[string]string `json:"variables,omitempty"`
	MaxTurns    int               `json:"max_turns"`
	MaxTokens   int               `json:"max_tokens"`
	TokenBudget int               `json:"token_budget"`
	Temperature *float64          `json:"temperature,omitempty"`
	Stream      bool              `json:"stream"`
}

// Response represents the result of an agent invocation.
type Response struct {
	Output    string           `json:"output"`
	ToolCalls []ToolCallRecord `json:"tool_calls,omitempty"`
	Tokens    llm.TokenUsage   `json:"tokens"`
	Turns     int              `json:"turns"`
	Duration  time.Duration    `json:"duration"`
	Error     string           `json:"error,omitempty"`
}

// StreamCallback is called with each streaming event during execution.
type StreamCallback func(event llm.StreamEvent)

// ToolExecutor dispatches tool calls during the agentic loop.
type ToolExecutor interface {
	Execute(ctx context.Context, call llm.ToolCall) (string, error)
	ExecuteConcurrent(ctx context.Context, calls []llm.ToolCall) []llm.ToolResult
}

// Strategy defines the execution strategy for an agentic loop.
type Strategy interface {
	// Execute runs the agentic loop and returns the final response.
	Execute(ctx context.Context, inv Invocation, llmClient llm.Client, tools ToolExecutor, onEvent StreamCallback) (*Response, error)

	// Name returns the strategy identifier.
	Name() string
}
