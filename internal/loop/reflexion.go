package loop

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// ReflexionStrategy implements the Reflexion pattern:
// 1. Execute: Generate initial response
// 2. Reflect: Self-critique the response
// 3. Iterate: Improve based on critique until satisfactory
type ReflexionStrategy struct{}

// Name returns the strategy identifier.
func (s *ReflexionStrategy) Name() string { return "reflexion" }

// Execute runs the Reflexion loop.
func (s *ReflexionStrategy) Execute(ctx context.Context, inv Invocation, llmClient llm.Client, tools ToolExecutor, onEvent StreamCallback) (*Response, error) {
	start := time.Now()
	tracker := llm.NewTokenTracker(inv.TokenBudget)

	maxTurns := inv.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 3
	}
	maxTokens := inv.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	var allToolRecords []ToolCallRecord
	turns := 0

	// Initial execution
	messages := make([]llm.Message, len(inv.Messages))
	copy(messages, inv.Messages)
	messages = append(messages, llm.Message{
		Role:    llm.RoleUser,
		Content: inv.Input,
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
		return nil, fmt.Errorf("reflexion initial: %w", err)
	}
	tracker.Add(resp.Usage)
	turns++

	currentOutput := resp.Content

	// Handle tool calls in initial response
	if len(resp.ToolCalls) > 0 && resp.StopReason == llm.StopToolUse {
		messages = append(messages, llm.Message{
			Role:      llm.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})
		toolResults := tools.ExecuteConcurrent(ctx, resp.ToolCalls)
		for i, result := range toolResults {
			tc := resp.ToolCalls[i]
			allToolRecords = append(allToolRecords, ToolCallRecord{
				ID: tc.ID, ToolName: tc.Name, Input: tc.Input, Output: result.Content,
			})
			messages = append(messages, llm.Message{Role: llm.RoleUser, ToolResult: &result})
		}

		// Get response after tool use
		req.Messages = messages
		resp, err = llmClient.Chat(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("reflexion after tools: %w", err)
		}
		tracker.Add(resp.Usage)
		turns++
		currentOutput = resp.Content
	}

	// Reflection iterations
	for iter := 0; iter < maxTurns-1; iter++ {
		if err := tracker.CheckBudget(maxTokens); err != nil {
			break
		}

		// Self-critique
		critiquePrompt := fmt.Sprintf("Review your previous response and identify any issues, "+
			"inaccuracies, or areas for improvement. If the response is satisfactory, "+
			"respond with exactly 'SATISFACTORY'.\n\nPrevious response:\n%s", currentOutput)

		critiqueResp, err := llmClient.Chat(ctx, llm.ChatRequest{
			Model:     inv.Model,
			Messages:  []llm.Message{{Role: llm.RoleUser, Content: critiquePrompt}},
			System:    inv.System,
			MaxTokens: maxTokens,
		})
		if err != nil {
			break
		}
		tracker.Add(critiqueResp.Usage)
		turns++

		critique := strings.TrimSpace(critiqueResp.Content)
		if strings.Contains(strings.ToUpper(critique), "SATISFACTORY") {
			break
		}

		if onEvent != nil {
			onEvent(llm.StreamEvent{Type: "text", Text: fmt.Sprintf("\n[Reflection %d]: %s\n", iter+1, critique)})
		}

		// Improve based on critique
		improvePrompt := fmt.Sprintf("Based on this critique, provide an improved response:\n\n"+
			"Critique: %s\n\nOriginal task: %s", critique, inv.Input)

		improveResp, err := llmClient.Chat(ctx, llm.ChatRequest{
			Model:     inv.Model,
			Messages:  []llm.Message{{Role: llm.RoleUser, Content: improvePrompt}},
			System:    inv.System,
			MaxTokens: maxTokens,
		})
		if err != nil {
			break
		}
		tracker.Add(improveResp.Usage)
		turns++

		currentOutput = improveResp.Content
	}

	return &Response{
		Output:    currentOutput,
		ToolCalls: allToolRecords,
		Tokens:    tracker.Usage(),
		Turns:     turns,
		Duration:  time.Since(start),
	}, nil
}
