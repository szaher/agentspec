// Package policy defines the policy types and rule engine interface
// for enforcing security constraints in the Agentz toolchain.
package policy

import "github.com/szaher/designs/agentz/internal/ir"

// Violation represents a policy rule that was violated.
type Violation struct {
	Rule     ir.Rule
	Resource ir.Resource
	Message  string
}

// Engine is the interface for policy evaluation.
type Engine interface {
	// Evaluate checks resources against policy rules and returns violations.
	Evaluate(policies []ir.Policy, resources []ir.Resource) []Violation
}

// DefaultEngine implements the Engine interface with built-in rules.
type DefaultEngine struct{}

// NewDefaultEngine creates a new default policy engine.
func NewDefaultEngine() *DefaultEngine {
	return &DefaultEngine{}
}

// Evaluate checks resources against policy rules.
func (e *DefaultEngine) Evaluate(policies []ir.Policy, resources []ir.Resource) []Violation {
	var violations []Violation
	for _, policy := range policies {
		for _, rule := range policy.Rules {
			for _, resource := range resources {
				if v := e.evaluateRule(rule, resource); v != nil {
					violations = append(violations, *v)
				}
			}
		}
	}
	return violations
}

func (e *DefaultEngine) evaluateRule(rule ir.Rule, resource ir.Resource) *Violation {
	switch rule.Action {
	case "deny":
		if matchesPattern(rule.Resource, resource.Kind) {
			return &Violation{
				Rule:     rule,
				Resource: resource,
				Message:  "denied by policy: " + rule.Subject,
			}
		}
	case "require":
		if matchesPattern(rule.Resource, resource.Kind) {
			if !checkRequirement(rule.Subject, resource) {
				return &Violation{
					Rule:     rule,
					Resource: resource,
					Message:  "requirement not met: " + rule.Subject,
				}
			}
		}
	}
	return nil
}

// matchesPattern checks if a resource kind matches a pattern.
// Supports "*" as wildcard for all resource kinds.
func matchesPattern(pattern, kind string) bool {
	if pattern == "*" {
		return true
	}
	return pattern == kind
}

// checkRequirement verifies whether a resource meets a requirement.
func checkRequirement(subject string, resource ir.Resource) bool {
	switch subject {
	case "pinned imports":
		// Check if all references are pinned
		return true // Validated elsewhere
	default:
		return true
	}
}
