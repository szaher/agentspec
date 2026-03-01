package policy

import (
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/ir"
)

func TestEvaluate_DenyMatchingKind(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "Agent",
			Subject:  "something",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Agent",
		Name: "my-agent",
		FQN:  "agent/my-agent",
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Resource.Name != "my-agent" {
		t.Errorf("expected resource name %q, got %q", "my-agent", violations[0].Resource.Name)
	}
}

func TestEvaluate_DenyNotMatchingKind(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "Tool",
			Subject:  "something",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Agent",
		Name: "my-agent",
		FQN:  "agent/my-agent",
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

func TestEvaluate_DenyCommandMatchingBinary(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "*",
			Subject:  "command rm",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Tool",
		Name: "cleanup",
		FQN:  "tool/cleanup",
		Attributes: map[string]interface{}{
			"binary": "rm",
		},
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "rm") {
		t.Errorf("expected violation message to mention %q, got %q", "rm", violations[0].Message)
	}
}

func TestEvaluate_DenyCommandNonMatchingBinary(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "*",
			Subject:  "command rm",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Tool",
		Name: "greeter",
		FQN:  "tool/greeter",
		Attributes: map[string]interface{}{
			"binary": "echo",
		},
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

func TestEvaluate_RequirePinnedImportsWithVersionedRefs(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "require",
			Resource: "*",
			Subject:  "pinned imports",
		}},
	}}
	resources := []ir.Resource{{
		Kind:       "Agent",
		Name:       "my-agent",
		FQN:        "agent/my-agent",
		References: []string{"lib@1.0.0"},
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

func TestEvaluate_RequirePinnedImportsWithLatestRef(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "require",
			Resource: "*",
			Subject:  "pinned imports",
		}},
	}}
	resources := []ir.Resource{{
		Kind:       "Agent",
		Name:       "my-agent",
		FQN:        "agent/my-agent",
		References: []string{"lib@latest"},
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "pinned imports") {
		t.Errorf("expected violation message to mention %q, got %q", "pinned imports", violations[0].Message)
	}
}

func TestEvaluate_RequirePinnedImportsWithNoVersion(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "require",
			Resource: "*",
			Subject:  "pinned imports",
		}},
	}}
	resources := []ir.Resource{{
		Kind:       "Agent",
		Name:       "my-agent",
		FQN:        "agent/my-agent",
		References: []string{"lib"},
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
}

func TestEvaluate_RequireSecretPresent(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "require",
			Resource: "*",
			Subject:  "secret API_KEY",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Agent",
		Name: "my-agent",
		FQN:  "agent/my-agent",
		Metadata: map[string]interface{}{
			"secrets": map[string]interface{}{
				"API_KEY": "x",
			},
		},
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations, got %d", len(violations))
	}
}

func TestEvaluate_RequireSecretMissing(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "require",
			Resource: "*",
			Subject:  "secret API_KEY",
		}},
	}}
	resources := []ir.Resource{{
		Kind:     "Agent",
		Name:     "my-agent",
		FQN:      "agent/my-agent",
		Metadata: map[string]interface{}{},
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "secret API_KEY") {
		t.Errorf("expected violation message to mention %q, got %q", "secret API_KEY", violations[0].Message)
	}
}

func TestEvaluate_RequireSignedPackagesStubReturnsTrue(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "require",
			Resource: "*",
			Subject:  "signed packages",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Agent",
		Name: "my-agent",
		FQN:  "agent/my-agent",
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations (stub returns true), got %d", len(violations))
	}
}

func TestEvaluate_MatchesPatternWildcard(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "*",
			Subject:  "everything",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Agent",
		Name: "any-agent",
		FQN:  "agent/any-agent",
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 1 {
		t.Fatalf("expected wildcard pattern to match, got %d violations", len(violations))
	}
}

func TestEvaluate_MatchesPatternExactMatch(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "Agent",
			Subject:  "everything",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Agent",
		Name: "my-agent",
		FQN:  "agent/my-agent",
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 1 {
		t.Fatalf("expected exact pattern to match, got %d violations", len(violations))
	}
}

func TestEvaluate_MatchesPatternNoMatch(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "Agent",
			Subject:  "everything",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Tool",
		Name: "my-tool",
		FQN:  "tool/my-tool",
	}}

	violations := engine.Evaluate(policies, resources)

	if len(violations) != 0 {
		t.Fatalf("expected pattern not to match, got %d violations", len(violations))
	}
}

func TestEvaluate_EmptyPolicies(t *testing.T) {
	engine := NewDefaultEngine()
	resources := []ir.Resource{{
		Kind: "Agent",
		Name: "my-agent",
		FQN:  "agent/my-agent",
	}}

	violations := engine.Evaluate(nil, resources)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations with empty policies, got %d", len(violations))
	}
}

func TestEvaluate_EmptyResources(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "*",
			Subject:  "everything",
		}},
	}}

	violations := engine.Evaluate(policies, nil)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations with empty resources, got %d", len(violations))
	}
}

func TestFormatViolations_ModeEnforce(t *testing.T) {
	violations := []Violation{{
		Rule: ir.Rule{
			Action:   "deny",
			Resource: "*",
			Subject:  "something",
		},
		Resource: ir.Resource{
			Kind: "Agent",
			Name: "my-agent",
			FQN:  "agent/my-agent",
		},
		Message: "denied by policy: something",
	}}

	result := FormatViolations(violations, ModeEnforce)

	if !strings.Contains(result, "ERROR") {
		t.Errorf("expected output to contain %q, got %q", "ERROR", result)
	}
}

func TestFormatViolations_ModeWarn(t *testing.T) {
	violations := []Violation{{
		Rule: ir.Rule{
			Action:   "deny",
			Resource: "*",
			Subject:  "something",
		},
		Resource: ir.Resource{
			Kind: "Agent",
			Name: "my-agent",
			FQN:  "agent/my-agent",
		},
		Message: "denied by policy: something",
	}}

	result := FormatViolations(violations, ModeWarn)

	if !strings.Contains(result, "WARNING") {
		t.Errorf("expected output to contain %q, got %q", "WARNING", result)
	}
}

func TestFormatViolations_EmptyViolations(t *testing.T) {
	result := FormatViolations(nil, ModeEnforce)

	if result != "" {
		t.Errorf("expected empty string for no violations, got %q", result)
	}
}
