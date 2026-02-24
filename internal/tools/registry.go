// Package tools implements the tool execution registry for the AgentSpec runtime.
package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/szaher/designs/agentz/internal/llm"
)

// Executor executes a tool call and returns the result as a string.
type Executor interface {
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
}

// Registry manages tool executors and dispatches tool calls.
type Registry struct {
	mu        sync.RWMutex
	executors map[string]Executor
	tools     map[string]llm.ToolDefinition
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		executors: make(map[string]Executor),
		tools:     make(map[string]llm.ToolDefinition),
	}
}

// Register adds a tool executor to the registry.
func (r *Registry) Register(name string, def llm.ToolDefinition, executor Executor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors[name] = executor
	r.tools[name] = def
}

// Execute dispatches a tool call to its registered executor.
func (r *Registry) Execute(ctx context.Context, call llm.ToolCall) (string, error) {
	r.mu.RLock()
	executor, ok := r.executors[call.Name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("tool %q not registered", call.Name)
	}

	return executor.Execute(ctx, call.Input)
}

// ExecuteConcurrent dispatches multiple tool calls concurrently and returns results.
func (r *Registry) ExecuteConcurrent(ctx context.Context, calls []llm.ToolCall) []llm.ToolResult {
	results := make([]llm.ToolResult, len(calls))
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(idx int, tc llm.ToolCall) {
			defer wg.Done()
			output, err := r.Execute(ctx, tc)
			if err != nil {
				results[idx] = llm.ToolResult{
					ToolUseID: tc.ID,
					Content:   err.Error(),
					IsError:   true,
				}
			} else {
				results[idx] = llm.ToolResult{
					ToolUseID: tc.ID,
					Content:   output,
				}
			}
		}(i, call)
	}

	wg.Wait()
	return results
}

// Definitions returns all registered tool definitions.
func (r *Registry) Definitions() []llm.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]llm.ToolDefinition, 0, len(r.tools))
	for _, d := range r.tools {
		defs = append(defs, d)
	}
	return defs
}
