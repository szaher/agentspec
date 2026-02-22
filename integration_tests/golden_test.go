package integration_tests

import (
	"context"
	"path/filepath"
	"testing"

	adapterlocal "github.com/szaher/designs/agentz/internal/adapters/local"
	"github.com/szaher/designs/agentz/internal/apply"
	"github.com/szaher/designs/agentz/internal/events"
	"github.com/szaher/designs/agentz/internal/formatter"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/plan"
	"github.com/szaher/designs/agentz/internal/state"
	"github.com/szaher/designs/agentz/internal/validate"
)

// TestGoldenPathLifecycle tests the complete lifecycle:
// parse → validate → format → plan → apply → apply(idempotency) → export
func TestGoldenPathLifecycle(t *testing.T) {
	input := readTestFile(t, "testdata/valid.az")

	// Step 1: Parse
	f, parseErrs := parser.Parse(input, "valid.az")
	if parseErrs != nil {
		t.Fatalf("parse failed: %v", parseErrs)
	}

	// Step 2: Validate (structural + semantic)
	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("structural validation failed: %v", structErrs)
	}
	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("semantic validation failed: %v", semErrs)
	}

	// Step 3: Format and verify idempotency
	formatted1 := formatter.Format(f)
	f2, _ := parser.Parse(formatted1, "valid.az")
	formatted2 := formatter.Format(f2)
	if formatted1 != formatted2 {
		t.Error("formatter is not idempotent")
	}

	// Step 4: Lower to IR
	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	// Step 5: Plan from fresh state
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, ".agentz.state.json")
	backend := state.NewLocalBackend(stateFile)
	current, _ := backend.Load()

	p := plan.ComputePlan(doc.Resources, current)
	if !p.HasChanges {
		t.Fatal("expected changes on first plan")
	}

	// Verify plan text is deterministic
	text1 := plan.FormatText(p)
	text2 := plan.FormatText(p)
	if text1 != text2 {
		t.Error("plan text is not deterministic")
	}

	// Step 6: Apply
	adapter := &adapterlocal.Adapter{}
	emitter := &events.CollectorEmitter{}
	result, err := apply.Apply(
		context.Background(), adapter, p.Actions, backend, emitter, "golden-test",
	)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Failed > 0 {
		t.Errorf("unexpected failures: %d", result.Failed)
	}

	// Step 7: Apply again (idempotency)
	current, _ = backend.Load()
	p2 := plan.ComputePlan(doc.Resources, current)
	if p2.HasChanges {
		t.Error("expected no changes on second plan")
	}

	// Step 8: Export
	exportDir := filepath.Join(tmpDir, "export")
	if err := adapter.Export(context.Background(), doc.Resources, exportDir); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Step 9: Verify events were emitted
	if len(emitter.Events) == 0 {
		t.Error("no events emitted")
	}

	t.Logf("Golden path lifecycle complete: %d resources created, %d events emitted",
		result.Created, len(emitter.Events))
}
