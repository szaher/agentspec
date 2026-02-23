package loop

import (
	"context"
	"fmt"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// ReActStrategy implements the Reason-Act-Observe agentic loop.
type ReActStrategy struct{}

// Name returns the strategy identifier.
func (s *ReActStrategy) Name() string { return "react" }

// Execute runs the ReAct loop: reason → act → observe, repeating until
// the model stops requesting tools or limits are reached.
func (s *ReActStrategy) Execute(ctx context.Context, inv Invocation, llmClient llm.Client, tools ToolExecutor, onEvent StreamCallback) (*Response, error) {
	start := time.Now()
	tracker := llm.NewTokenTracker(inv.TokenBudget)

	maxTurns := inv.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 10
	}

	maxTokens := inv.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	// Build initial messages
	messages := make([]llm.Message, len(inv.Messages))
	copy(messages, inv.Messages)
	messages = append(messages, llm.Message{
		Role:    llm.RoleUser,
		Content: inv.Input,
	})

	var allToolRecords []ToolCallRecord
	var finalOutput string
	turns := 0

	for turn := 0; turn < maxTurns; turn++ {
		turns++

		// Check token budget
		if err := tracker.CheckBudget(maxTokens); err != nil {
			return &Response{
				Output:    finalOutput,
				ToolCalls: allToolRecords,
				Tokens:    tracker.Usage(),
				Turns:     turns,
				Duration:  time.Since(start),
				Error:     err.Error(),
			}, nil
		}

		req := llm.ChatRequest{
			Model:       inv.Model,
			Messages:    messages,
			System:      inv.System,
			MaxTokens:   maxTokens,
			Temperature: inv.Temperature,
		}

		// Get tool definitions from executor if it provides them
		if reg, ok := tools.(interface{ Definitions() []llm.ToolDefinition }); ok {
			req.Tools = reg.Definitions()
		}

		var resp *llm.ChatResponse
		var err error

		if inv.Stream && onEvent != nil {
			ch, streamErr := llmClient.ChatStream(ctx, req)
			if streamErr != nil {
				return nil, fmt.Errorf("react: stream turn %d: %w", turn+1, streamErr)
			}
			for event := range ch {
				onEvent(event)
				if event.Response != nil {
					resp = event.Response
				}
				if event.Error != nil {
					return nil, fmt.Errorf("react: stream error turn %d: %w", turn+1, event.Error)
				}
			}
			if resp == nil {
				return nil, fmt.Errorf("react: stream turn %d: no response received", turn+1)
			}
		} else {
			resp, err = llmClient.Chat(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("react: turn %d: %w", turn+1, err)
			}
		}

		tracker.Add(resp.Usage)

		// Accumulate text output
		if resp.Content != "" {
			finalOutput = resp.Content
		}

		// If no tool calls, we're done
		if len(resp.ToolCalls) == 0 || resp.StopReason != llm.StopToolUse {
			break
		}

		// Add assistant message with tool calls
		messages = append(messages, llm.Message{
			Role:      llm.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute tools concurrently
		toolResults := tools.ExecuteConcurrent(ctx, resp.ToolCalls)

		// Record tool calls and add results as messages
		for i, result := range toolResults {
			tc := resp.ToolCalls[i]
			record := ToolCallRecord{
				ID:       tc.ID,
				ToolName: tc.Name,
				Input:    tc.Input,
				Output:   result.Content,
			}
			if result.IsError {
				record.Error = result.Content
			}
			allToolRecords = append(allToolRecords, record)

			if onEvent != nil {
				onEvent(llm.StreamEvent{
					Type: "tool_call_end",
					ToolCall: &llm.ToolCall{
						ID:   tc.ID,
						Name: tc.Name,
					},
				})
			}

			messages = append(messages, llm.Message{
				Role:       llm.RoleUser,
				ToolResult: &result,
			})
		}
	}

	return &Response{
		Output:    finalOutput,
		ToolCalls: allToolRecords,
		Tokens:    tracker.Usage(),
		Turns:     turns,
		Duration:  time.Since(start),
	}, nil
}
