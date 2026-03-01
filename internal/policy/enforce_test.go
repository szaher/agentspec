package policy

import (
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/ir"
)

func TestEnforce_ZeroViolations(t *testing.T) {
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

	errs := Enforce(engine, policies, resources)

	if errs != nil {
		t.Fatalf("expected nil errors, got %d", len(errs))
	}
}

func TestEnforce_MultipleViolations(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{
			{
				Action:   "deny",
				Resource: "*",
				Subject:  "something dangerous",
			},
			{
				Action:   "require",
				Resource: "*",
				Subject:  "pinned imports",
			},
		},
	}}
	resources := []ir.Resource{{
		Kind:       "Agent",
		Name:       "my-agent",
		FQN:        "agent/my-agent",
		References: []string{"lib@latest"},
	}}

	errs := Enforce(engine, policies, resources)

	if len(errs) != 2 {
		t.Fatalf("expected 2 validation errors, got %d", len(errs))
	}
}

func TestEnforce_ViolationMessageFormat(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "*",
			Subject:  "something",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Agent",
		Name: "my-agent",
		FQN:  "agent/my-agent",
	}}

	errs := Enforce(engine, policies, resources)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if !strings.HasPrefix(errs[0].Message, "policy violation on ") {
		t.Errorf("expected message to start with %q, got %q", "policy violation on ", errs[0].Message)
	}
	if !strings.Contains(errs[0].Message, "agent/my-agent") {
		t.Errorf("expected message to contain FQN %q, got %q", "agent/my-agent", errs[0].Message)
	}
}

func TestEnforce_ViolationHintFormat(t *testing.T) {
	engine := NewDefaultEngine()
	policies := []ir.Policy{{
		Name: "test",
		Rules: []ir.Rule{{
			Action:   "deny",
			Resource: "Agent",
			Subject:  "command rm",
		}},
	}}
	resources := []ir.Resource{{
		Kind: "Agent",
		Name: "my-agent",
		FQN:  "agent/my-agent",
		Attributes: map[string]interface{}{
			"binary": "rm",
		},
	}}

	errs := Enforce(engine, policies, resources)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if !strings.HasPrefix(errs[0].Hint, "rule: ") {
		t.Errorf("expected hint to start with %q, got %q", "rule: ", errs[0].Hint)
	}
	if !strings.Contains(errs[0].Hint, "deny") {
		t.Errorf("expected hint to contain %q, got %q", "deny", errs[0].Hint)
	}

	// Verify the require variant too
	policies[0].Rules[0].Action = "require"
	policies[0].Rules[0].Subject = "secret API_KEY"
	resources[0].Attributes = nil
	resources[0].Metadata = map[string]interface{}{}

	errs = Enforce(engine, policies, resources)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error for require violation, got %d", len(errs))
	}
	if !strings.Contains(errs[0].Hint, "require") {
		t.Errorf("expected hint to contain %q, got %q", "require", errs[0].Hint)
	}
}
