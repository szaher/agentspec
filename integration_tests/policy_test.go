package integration_tests

import (
	"testing"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/policy"
)

func TestPolicyPinnedImports(t *testing.T) {
	engine := policy.NewDefaultEngine()

	t.Run("unpinned import rejected", func(t *testing.T) {
		policies := []ir.Policy{{
			Name: "security",
			Rules: []ir.Rule{{Action: "require", Resource: "*", Subject: "pinned imports"}},
		}}
		resources := []ir.Resource{{
			Kind:       "agent",
			Name:       "my-agent",
			References: []string{"github.com/org/pkg"},
		}}
		violations := engine.Evaluate(policies, resources)
		if len(violations) == 0 {
			t.Fatal("expected violation for unpinned import")
		}
		if violations[0].Message != "requirement not met: pinned imports" {
			t.Errorf("unexpected message: %s", violations[0].Message)
		}
	})

	t.Run("pinned import accepted", func(t *testing.T) {
		policies := []ir.Policy{{
			Name: "security",
			Rules: []ir.Rule{{Action: "require", Resource: "*", Subject: "pinned imports"}},
		}}
		resources := []ir.Resource{{
			Kind:       "agent",
			Name:       "my-agent",
			References: []string{"github.com/org/pkg@v1.2.3"},
		}}
		violations := engine.Evaluate(policies, resources)
		if len(violations) != 0 {
			t.Fatalf("expected no violations, got %d: %v", len(violations), violations)
		}
	})

	t.Run("@latest is not pinned", func(t *testing.T) {
		policies := []ir.Policy{{
			Name: "security",
			Rules: []ir.Rule{{Action: "require", Resource: "*", Subject: "pinned imports"}},
		}}
		resources := []ir.Resource{{
			Kind:       "agent",
			Name:       "my-agent",
			References: []string{"github.com/org/pkg@latest"},
		}}
		violations := engine.Evaluate(policies, resources)
		if len(violations) == 0 {
			t.Fatal("@latest should not be considered pinned")
		}
	})
}

func TestPolicyDenyCommand(t *testing.T) {
	engine := policy.NewDefaultEngine()

	t.Run("denied command blocked", func(t *testing.T) {
		policies := []ir.Policy{{
			Name: "security",
			Rules: []ir.Rule{{Action: "deny", Resource: "*", Subject: "command rm"}},
		}}
		resources := []ir.Resource{{
			Kind:       "tool",
			Name:       "cleanup",
			Attributes: map[string]interface{}{"binary": "rm"},
		}}
		violations := engine.Evaluate(policies, resources)
		if len(violations) == 0 {
			t.Fatal("expected violation for denied command")
		}
	})

	t.Run("allowed command passes", func(t *testing.T) {
		policies := []ir.Policy{{
			Name: "security",
			Rules: []ir.Rule{{Action: "deny", Resource: "*", Subject: "command rm"}},
		}}
		resources := []ir.Resource{{
			Kind:       "tool",
			Name:       "lister",
			Attributes: map[string]interface{}{"binary": "ls"},
		}}
		violations := engine.Evaluate(policies, resources)
		if len(violations) != 0 {
			t.Fatalf("expected no violations for allowed command, got %d", len(violations))
		}
	})
}

func TestPolicyRequireSecret(t *testing.T) {
	engine := policy.NewDefaultEngine()

	t.Run("missing secret rejected", func(t *testing.T) {
		policies := []ir.Policy{{
			Name: "security",
			Rules: []ir.Rule{{Action: "require", Resource: "*", Subject: "secret api-key"}},
		}}
		resources := []ir.Resource{{
			Kind: "agent",
			Name: "my-agent",
		}}
		violations := engine.Evaluate(policies, resources)
		if len(violations) == 0 {
			t.Fatal("expected violation for missing secret")
		}
	})

	t.Run("present secret accepted", func(t *testing.T) {
		policies := []ir.Policy{{
			Name: "security",
			Rules: []ir.Rule{{Action: "require", Resource: "*", Subject: "secret api-key"}},
		}}
		resources := []ir.Resource{{
			Kind: "agent",
			Name: "my-agent",
			Metadata: map[string]interface{}{
				"secrets": map[string]interface{}{"api-key": "value"},
			},
		}}
		violations := engine.Evaluate(policies, resources)
		if len(violations) != 0 {
			t.Fatalf("expected no violations, got %d", len(violations))
		}
	})
}

func TestPolicySignedPackages(t *testing.T) {
	engine := policy.NewDefaultEngine()

	// Signed packages is a stub that warns but passes
	policies := []ir.Policy{{
		Name: "security",
		Rules: []ir.Rule{{Action: "require", Resource: "*", Subject: "signed packages"}},
	}}
	resources := []ir.Resource{{
		Kind: "agent",
		Name: "my-agent",
	}}
	violations := engine.Evaluate(policies, resources)
	if len(violations) != 0 {
		t.Fatalf("signed packages stub should pass, got %d violations", len(violations))
	}
}

func TestPolicyMultipleViolations(t *testing.T) {
	engine := policy.NewDefaultEngine()

	policies := []ir.Policy{{
		Name: "security",
		Rules: []ir.Rule{
			{Action: "require", Resource: "*", Subject: "pinned imports"},
			{Action: "deny", Resource: "*", Subject: "command rm"},
		},
	}}
	resources := []ir.Resource{
		{Kind: "agent", Name: "my-agent", References: []string{"unpinned/pkg"}},
		{Kind: "tool", Name: "cleanup", Attributes: map[string]interface{}{"binary": "rm"}},
	}

	violations := engine.Evaluate(policies, resources)
	if len(violations) < 2 {
		t.Fatalf("expected at least 2 violations, got %d", len(violations))
	}
}

func TestPolicyWarnMode(t *testing.T) {
	violations := []policy.Violation{{
		Rule:     ir.Rule{Action: "require", Resource: "*", Subject: "pinned imports"},
		Resource: ir.Resource{Kind: "agent", Name: "my-agent"},
		Message:  "requirement not met: pinned imports",
	}}

	output := policy.FormatViolations(violations, policy.ModeWarn)
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	if !contains(output, "WARNING") {
		t.Errorf("warn mode should use WARNING prefix, got: %s", output)
	}

	output2 := policy.FormatViolations(violations, policy.ModeEnforce)
	if !contains(output2, "ERROR") {
		t.Errorf("enforce mode should use ERROR prefix, got: %s", output2)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
