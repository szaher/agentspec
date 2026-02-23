package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/szaher/designs/agentz/internal/adapters"
	adapterlocal "github.com/szaher/designs/agentz/internal/adapters/local"
	"github.com/szaher/designs/agentz/internal/apply"
	"github.com/szaher/designs/agentz/internal/events"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/plan"
	"github.com/szaher/designs/agentz/internal/state"
)

func TestPlanApplyIdempotencyCycle(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")

	f, parseErrs := parser.Parse(input, "valid.ias")
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering: %v", err)
	}

	// Set up temp state file
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, ".agentspec.state.json")
	backend := state.NewLocalBackend(stateFile)

	// Step 1: Plan from fresh state — should show all creates
	current, _ := backend.Load()
	p := plan.ComputePlan(doc.Resources, current)
	if !p.HasChanges {
		t.Fatal("expected changes on first plan")
	}

	creates := 0
	for _, a := range p.Actions {
		if a.Type == adapters.ActionCreate {
			creates++
		}
	}
	if creates == 0 {
		t.Fatal("expected create actions on first plan")
	}

	// Verify plan text output is deterministic
	text1 := plan.FormatText(p)
	text2 := plan.FormatText(p)
	if text1 != text2 {
		t.Error("plan text output is not deterministic")
	}

	// Verify plan JSON output is deterministic
	json1, _ := plan.FormatJSON(p)
	json2, _ := plan.FormatJSON(p)
	if json1 != json2 {
		t.Error("plan JSON output is not deterministic")
	}

	// Step 2: Apply
	adapter := &adapterlocal.Adapter{}
	emitter := &events.CollectorEmitter{}
	result, err := apply.Apply(
		context.Background(),
		adapter,
		p.Actions,
		backend,
		emitter,
		"test-correlation",
	)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Created == 0 {
		t.Error("expected created resources after first apply")
	}
	if result.Failed > 0 {
		t.Errorf("unexpected failures: %d", result.Failed)
	}

	// Step 3: Plan again — should show no changes (idempotency)
	current, _ = backend.Load()
	p2 := plan.ComputePlan(doc.Resources, current)
	if p2.HasChanges {
		t.Error("expected no changes on second plan (idempotency)")
		for _, a := range p2.Actions {
			if a.Type != adapters.ActionNoop {
				t.Logf("  unexpected action: %s %s", a.Type, a.FQN)
			}
		}
	}

	// Step 4: Apply again — should be no-op
	result2, err := apply.Apply(
		context.Background(),
		adapter,
		p2.Actions,
		backend,
		emitter,
		"test-correlation-2",
	)
	if err != nil {
		t.Fatalf("second apply failed: %v", err)
	}
	if result2.Created+result2.Updated+result2.Deleted > 0 {
		t.Error("expected no changes on second apply")
	}

	// Step 5: Verify state file exists and is valid JSON
	_, err = os.Stat(stateFile)
	if err != nil {
		t.Errorf("state file not found: %v", err)
	}

	entries, err := backend.Load()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if len(entries) == 0 {
		t.Error("state file has no entries after apply")
	}

	// Step 6: Verify events were emitted
	if len(emitter.Events) == 0 {
		t.Error("no events emitted during apply")
	}
}
