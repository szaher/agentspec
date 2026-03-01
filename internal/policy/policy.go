// Package policy defines the policy types and rule engine interface
// for enforcing security constraints in the AgentSpec toolchain.
package policy

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/szaher/designs/agentz/internal/ir"
)

// EvalMode determines how violations are handled.
type EvalMode int

const (
	// ModeEnforce blocks on violations (default).
	ModeEnforce EvalMode = iota
	// ModeWarn reports violations without blocking.
	ModeWarn
)

// Violation represents a policy rule that was violated.
type Violation struct {
	Rule     ir.Rule
	Resource ir.Resource
	Message  string
	Details  string
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
// Collects ALL violations (not just the first).
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
		return e.evaluateDeny(rule, resource)
	case "require":
		return e.evaluateRequire(rule, resource)
	default:
		return nil
	}
}

func (e *DefaultEngine) evaluateDeny(rule ir.Rule, resource ir.Resource) *Violation {
	if !matchesPattern(rule.Resource, resource.Kind) {
		return nil
	}

	switch {
	case strings.HasPrefix(rule.Subject, "command "):
		// deny command <binary>
		deniedBinary := strings.TrimPrefix(rule.Subject, "command ")
		if binary, ok := resource.Attributes["binary"].(string); ok && binary == deniedBinary {
			return &Violation{
				Rule:     rule,
				Resource: resource,
				Message:  fmt.Sprintf("denied by policy: command %q is blocked", deniedBinary),
				Details:  fmt.Sprintf("Resource %q uses denied command %q", resource.Name, deniedBinary),
			}
		}
	default:
		// Generic deny
		return &Violation{
			Rule:     rule,
			Resource: resource,
			Message:  "denied by policy: " + rule.Subject,
		}
	}
	return nil
}

func (e *DefaultEngine) evaluateRequire(rule ir.Rule, resource ir.Resource) *Violation {
	if !matchesPattern(rule.Resource, resource.Kind) {
		return nil
	}

	if !checkRequirement(rule.Subject, resource) {
		return &Violation{
			Rule:     rule,
			Resource: resource,
			Message:  "requirement not met: " + rule.Subject,
			Details:  detailsForRequirement(rule.Subject, resource),
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
// Implements 4 requirement types per FR-006.
func checkRequirement(subject string, resource ir.Resource) bool {
	switch subject {
	case "pinned imports":
		return checkPinnedImports(resource)
	case "signed packages":
		return checkSignedPackages(resource)
	default:
		if strings.HasPrefix(subject, "secret ") {
			secretName := strings.TrimPrefix(subject, "secret ")
			return checkSecret(secretName, resource)
		}
		// Unknown requirement type — fail closed
		return false
	}
}

// checkPinnedImports verifies all references have version pins (semver or SHA).
func checkPinnedImports(resource ir.Resource) bool {
	if len(resource.References) == 0 {
		return true // No references to pin
	}
	for _, ref := range resource.References {
		if !hasVersionPin(ref) {
			return false
		}
	}
	return true
}

// hasVersionPin checks if a reference string contains a version pin.
// Accepts: @1.2.3, @v1.2.3, @sha256:..., @latest is NOT pinned.
func hasVersionPin(ref string) bool {
	atIdx := strings.LastIndex(ref, "@")
	if atIdx < 0 {
		return false // No version specified
	}
	version := ref[atIdx+1:]
	if version == "latest" || version == "" {
		return false
	}
	return true
}

// checkSecret verifies a named secret is available in resource metadata.
func checkSecret(secretName string, resource ir.Resource) bool {
	secrets, ok := resource.Metadata["secrets"].(map[string]interface{})
	if !ok {
		return false
	}
	_, exists := secrets[secretName]
	return exists
}

// checkSignedPackages checks if imported packages have valid signatures.
// Currently a stub that logs a warning per research R4.
func checkSignedPackages(resource ir.Resource) bool {
	slog.Warn("signed packages requirement: signature verification not yet implemented, allowing",
		"resource", resource.Name)
	return true
}

func detailsForRequirement(subject string, resource ir.Resource) string {
	switch subject {
	case "pinned imports":
		var unpinned []string
		for _, ref := range resource.References {
			if !hasVersionPin(ref) {
				unpinned = append(unpinned, ref)
			}
		}
		return fmt.Sprintf("Unpinned imports: %s", strings.Join(unpinned, ", "))
	case "signed packages":
		return "Package signature verification not yet implemented"
	default:
		if strings.HasPrefix(subject, "secret ") {
			secretName := strings.TrimPrefix(subject, "secret ")
			return fmt.Sprintf("Secret %q not found in configured secrets", secretName)
		}
		return fmt.Sprintf("Unknown requirement type: %q", subject)
	}
}

// FormatViolations formats violations for user output, grouped by resource.
func FormatViolations(violations []Violation, mode EvalMode) string {
	if len(violations) == 0 {
		return ""
	}

	prefix := "ERROR"
	if mode == ModeWarn {
		prefix = "WARNING"
	}

	var sb strings.Builder
	// Group by resource
	grouped := make(map[string][]Violation)
	var order []string
	for _, v := range violations {
		key := v.Resource.Name
		if _, seen := grouped[key]; !seen {
			order = append(order, key)
		}
		grouped[key] = append(grouped[key], v)
	}

	for _, key := range order {
		sb.WriteString(fmt.Sprintf("\n%s [%s]:\n", prefix, key))
		for _, v := range grouped[key] {
			sb.WriteString(fmt.Sprintf("  [%s] %s\n", v.Rule.Action, v.Message))
			if v.Details != "" {
				sb.WriteString(fmt.Sprintf("    %s\n", v.Details))
			}
		}
	}

	return sb.String()
}
