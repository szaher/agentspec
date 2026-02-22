package plan

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/szaher/designs/agentz/internal/adapters"
)

// FormatText produces a human-readable text plan output.
func FormatText(p *Plan) string {
	if !p.HasChanges {
		return "No changes. Infrastructure is up-to-date.\n"
	}

	var creates, updates, deletes, noops int
	for _, a := range p.Actions {
		switch a.Type {
		case adapters.ActionCreate:
			creates++
		case adapters.ActionUpdate:
			updates++
		case adapters.ActionDelete:
			deletes++
		case adapters.ActionNoop:
			noops++
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Plan: %d to create, %d to update, %d to delete\n\n",
		creates, updates, deletes))

	for _, a := range p.Actions {
		switch a.Type {
		case adapters.ActionCreate:
			sb.WriteString(fmt.Sprintf("  + %s\n", fqnToDisplay(a.FQN)))
		case adapters.ActionUpdate:
			sb.WriteString(fmt.Sprintf("  ~ %s\n", fqnToDisplay(a.FQN)))
		case adapters.ActionDelete:
			sb.WriteString(fmt.Sprintf("  - %s\n", fqnToDisplay(a.FQN)))
		}
	}

	if p.TargetBinding != "" {
		sb.WriteString(fmt.Sprintf("\nTarget: %s\n", p.TargetBinding))
	}

	return sb.String()
}

// FormatJSON produces a JSON plan output.
func FormatJSON(p *Plan) (string, error) {
	type jsonAction struct {
		FQN    string `json:"fqn"`
		Action string `json:"action"`
		Reason string `json:"reason,omitempty"`
	}
	type jsonPlan struct {
		HasChanges bool         `json:"has_changes"`
		Actions    []jsonAction `json:"actions"`
		Target     string       `json:"target,omitempty"`
	}

	jp := jsonPlan{
		HasChanges: p.HasChanges,
		Target:     p.TargetBinding,
	}
	for _, a := range p.Actions {
		jp.Actions = append(jp.Actions, jsonAction{
			FQN:    a.FQN,
			Action: string(a.Type),
			Reason: a.Reason,
		})
	}

	data, err := json.MarshalIndent(jp, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

// fqnToDisplay converts "pkg/Kind/name" to "Kind/name" for display.
func fqnToDisplay(fqn string) string {
	parts := strings.SplitN(fqn, "/", 3)
	if len(parts) == 3 {
		return parts[1] + "/" + parts[2]
	}
	return fqn
}
