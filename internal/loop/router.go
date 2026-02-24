package loop

import (
	"context"
	"fmt"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// RouterStrategy classifies input and dispatches to a specialized sub-agent.
// The router uses the LLM to determine the best agent for each request.
type RouterStrategy struct {
	// AgentInvoker is used to invoke the selected sub-agent.
	AgentInvoker func(ctx context.Context, agentName string, inv Invocation, llmClient llm.Client, tools ToolExecutor, onEvent StreamCallback) (*Response, error)

	// Routes maps agent names to their descriptions for classification.
	Routes map[string]string
}

// Name returns the strategy identifier.
func (s *RouterStrategy) Name() string { return "router" }

// Execute classifies the input and routes to the appropriate agent.
func (s *RouterStrategy) Execute(ctx context.Context, inv Invocation, llmClient llm.Client, tools ToolExecutor, onEvent StreamCallback) (*Response, error) {
	start := time.Now()
	tracker := llm.NewTokenTracker(inv.TokenBudget)

	// If no routes defined, fall back to ReAct
	if len(s.Routes) == 0 {
		react := &ReActStrategy{}
		return react.Execute(ctx, inv, llmClient, tools, onEvent)
	}

	// Classification prompt
	classifyPrompt := "Classify the following user message and determine which agent should handle it.\n\n"
	classifyPrompt += fmt.Sprintf("User message: %s\n\n", inv.Input)
	classifyPrompt += "Available agents:\n"
	for name, desc := range s.Routes {
		classifyPrompt += fmt.Sprintf("- %s: %s\n", name, desc)
	}
	classifyPrompt += "\nRespond with ONLY the agent name."

	classifyResp, err := llmClient.Chat(ctx, llm.ChatRequest{
		Model:     inv.Model,
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: classifyPrompt}},
		MaxTokens: 50,
	})
	if err != nil {
		return nil, fmt.Errorf("router classify: %w", err)
	}
	tracker.Add(classifyResp.Usage)

	if onEvent != nil {
		onEvent(llm.StreamEvent{Type: "text", Text: fmt.Sprintf("[Routing to: %s]\n", classifyResp.Content)})
	}

	// Invoke the selected agent
	if s.AgentInvoker != nil {
		routed, err := s.AgentInvoker(ctx, classifyResp.Content, inv, llmClient, tools, onEvent)
		if err != nil {
			return nil, fmt.Errorf("router invoke %s: %w", classifyResp.Content, err)
		}
		routed.Tokens = llm.TokenUsage{
			InputTokens:  tracker.Usage().InputTokens + routed.Tokens.InputTokens,
			OutputTokens: tracker.Usage().OutputTokens + routed.Tokens.OutputTokens,
		}
		routed.Duration = time.Since(start)
		return routed, nil
	}

	// No invoker, fall back to ReAct with the current agent
	react := &ReActStrategy{}
	resp, err := react.Execute(ctx, inv, llmClient, tools, onEvent)
	if err != nil {
		return nil, err
	}
	resp.Tokens = llm.TokenUsage{
		InputTokens:  tracker.Usage().InputTokens + resp.Tokens.InputTokens,
		OutputTokens: tracker.Usage().OutputTokens + resp.Tokens.OutputTokens,
	}
	resp.Duration = time.Since(start)
	return resp, nil
}
