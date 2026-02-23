package loop

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// PlanExecuteStrategy implements a two-phase strategy:
// 1. Plan: LLM creates a step-by-step plan
// 2. Execute: Each step is executed sequentially, re-planning on failure
type PlanExecuteStrategy struct{}

// Name returns the strategy identifier.
func (s *PlanExecuteStrategy) Name() string { return "plan-and-execute" }

// Execute runs the Plan-and-Execute loop.
func (s *PlanExecuteStrategy) Execute(ctx context.Context, inv Invocation, llmClient llm.Client, tools ToolExecutor, onEvent StreamCallback) (*Response, error) {
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

	// Phase 1: Generate a plan
	planPrompt := fmt.Sprintf("%s\n\nCreate a step-by-step plan to accomplish the following task. "+
		"Format each step on a new line prefixed with 'STEP N:'. "+
		"After planning, I will execute each step.\n\nTask: %s", inv.System, inv.Input)

	planResp, err := llmClient.Chat(ctx, llm.ChatRequest{
		Model: inv.Model,
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: planPrompt},
		},
		MaxTokens: maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("plan phase: %w", err)
	}
	tracker.Add(planResp.Usage)

	if onEvent != nil {
		onEvent(llm.StreamEvent{Type: "text", Text: "Plan:\n" + planResp.Content + "\n\n"})
	}

	// Phase 2: Execute each step
	messages := []llm.Message{
		{Role: llm.RoleUser, Content: inv.Input},
		{Role: llm.RoleAssistant, Content: planResp.Content},
	}

	var allToolRecords []ToolCallRecord
	var finalOutput string
	turns := 1

	// Extract step count from plan
	steps := countSteps(planResp.Content)
	if steps == 0 {
		steps = 1
	}

	for step := 0; step < steps && step < maxTurns-1; step++ {
		turns++

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

		stepPrompt := fmt.Sprintf("Execute step %d of the plan. Use available tools as needed.", step+1)
		messages = append(messages, llm.Message{
			Role:    llm.RoleUser,
			Content: stepPrompt,
		})

		req := llm.ChatRequest{
			Model:       inv.Model,
			Messages:    messages,
			System:      inv.System,
			MaxTokens:   maxTokens,
			Temperature: inv.Temperature,
		}
		if reg, ok := tools.(interface{ Definitions() []llm.ToolDefinition }); ok {
			req.Tools = reg.Definitions()
		}

		resp, err := llmClient.Chat(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("execute step %d: %w", step+1, err)
		}
		tracker.Add(resp.Usage)

		if resp.Content != "" {
			finalOutput = resp.Content
		}

		messages = append(messages, llm.Message{
			Role:      llm.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Handle tool calls within this step
		if len(resp.ToolCalls) > 0 && resp.StopReason == llm.StopToolUse {
			toolResults := tools.ExecuteConcurrent(ctx, resp.ToolCalls)
			for i, result := range toolResults {
				tc := resp.ToolCalls[i]
				allToolRecords = append(allToolRecords, ToolCallRecord{
					ID:       tc.ID,
					ToolName: tc.Name,
					Input:    tc.Input,
					Output:   result.Content,
				})
				messages = append(messages, llm.Message{
					Role:       llm.RoleUser,
					ToolResult: &result,
				})
			}
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

func countSteps(plan string) int {
	count := 0
	for _, line := range strings.Split(plan, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "STEP ") || strings.HasPrefix(line, "Step ") {
			count++
		}
	}
	return count
}
