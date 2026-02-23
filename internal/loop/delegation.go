package loop

import (
	"context"
	"fmt"
	"strings"

	"github.com/szaher/designs/agentz/internal/llm"
)

// DelegationRule describes when an agent should delegate to another agent.
type DelegationRule struct {
	TargetAgent string
	Condition   string
}

// DelegationRouter evaluates delegation rules against user input using the LLM.
type DelegationRouter struct {
	Rules     []DelegationRule
	LLMClient llm.Client
	Model     string
}

// EvaluateResult represents the result of delegation evaluation.
type EvaluateResult struct {
	ShouldDelegate bool
	TargetAgent    string
	Confidence     float64
}

// Evaluate checks if the input matches any delegation rule.
// Uses the LLM to evaluate natural language conditions.
func (d *DelegationRouter) Evaluate(ctx context.Context, input string) (*EvaluateResult, error) {
	if len(d.Rules) == 0 {
		return &EvaluateResult{ShouldDelegate: false}, nil
	}

	// Build a classification prompt
	var sb strings.Builder
	sb.WriteString("Given the following user message, determine which agent (if any) should handle it.\n\n")
	sb.WriteString("User message: ")
	sb.WriteString(input)
	sb.WriteString("\n\nAvailable agents and their conditions:\n")
	for i, rule := range d.Rules {
		fmt.Fprintf(&sb, "%d. Agent \"%s\" - when: \"%s\"\n", i+1, rule.TargetAgent, rule.Condition)
	}
	sb.WriteString("\nRespond with ONLY the agent name if a match is found, or \"NONE\" if no agent matches.")

	resp, err := d.LLMClient.Chat(ctx, llm.ChatRequest{
		Model: d.Model,
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: sb.String()},
		},
		MaxTokens: 50,
	})
	if err != nil {
		return nil, fmt.Errorf("delegation evaluation: %w", err)
	}

	answer := strings.TrimSpace(resp.Content)
	if answer == "NONE" || answer == "" {
		return &EvaluateResult{ShouldDelegate: false}, nil
	}

	// Match against known agents
	for _, rule := range d.Rules {
		if strings.Contains(strings.ToLower(answer), strings.ToLower(rule.TargetAgent)) {
			return &EvaluateResult{
				ShouldDelegate: true,
				TargetAgent:    rule.TargetAgent,
				Confidence:     1.0,
			}, nil
		}
	}

	return &EvaluateResult{ShouldDelegate: false}, nil
}
