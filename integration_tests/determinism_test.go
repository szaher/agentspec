package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	adapterlocal "github.com/szaher/designs/agentz/internal/adapters/local"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/plan"
)

// TestDeterminismIR verifies byte-identical IR output across runs.
func TestDeterminismIR(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")

	// Parse and lower twice
	f1, _ := parser.Parse(input, "valid.ias")
	doc1, _ := ir.Lower(f1)
	json1, _ := doc1.MarshalJSON()

	f2, _ := parser.Parse(input, "valid.ias")
	doc2, _ := ir.Lower(f2)
	json2, _ := doc2.MarshalJSON()

	if string(json1) != string(json2) {
		t.Error("IR JSON output is not deterministic across runs")
	}
}

// TestDeterminismPlan verifies byte-identical plan output across runs.
func TestDeterminismPlan(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")

	f, _ := parser.Parse(input, "valid.ias")
	doc, _ := ir.Lower(f)

	// Both plans from empty state
	p1 := plan.ComputePlan(doc.Resources, nil)
	p2 := plan.ComputePlan(doc.Resources, nil)

	text1 := plan.FormatText(p1)
	text2 := plan.FormatText(p2)
	if text1 != text2 {
		t.Error("plan text output is not deterministic")
	}

	json1, _ := plan.FormatJSON(p1)
	json2, _ := plan.FormatJSON(p2)
	if json1 != json2 {
		t.Error("plan JSON output is not deterministic")
	}
}

// TestDeterminismExport verifies byte-identical export output across runs.
func TestDeterminismExport(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")

	f, _ := parser.Parse(input, "valid.ias")
	doc, _ := ir.Lower(f)

	adapter := &adapterlocal.Adapter{}
	dir1 := filepath.Join(t.TempDir(), "export1")
	dir2 := filepath.Join(t.TempDir(), "export2")

	if err := adapter.Export(context.Background(), doc.Resources, dir1); err != nil {
		t.Fatalf("export to dir1 failed: %v", err)
	}
	if err := adapter.Export(context.Background(), doc.Resources, dir2); err != nil {
		t.Fatalf("export to dir2 failed: %v", err)
	}

	files1, _ := os.ReadDir(dir1)
	for _, f := range files1 {
		data1, _ := os.ReadFile(filepath.Join(dir1, f.Name()))
		data2, _ := os.ReadFile(filepath.Join(dir2, f.Name()))
		if string(data1) != string(data2) {
			t.Errorf("export file %s is not deterministic", f.Name())
		}
	}
}

// TestDeterminismHashes verifies identical content hashes for identical inputs.
func TestDeterminismHashes(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")

	f1, _ := parser.Parse(input, "valid.ias")
	doc1, _ := ir.Lower(f1)

	f2, _ := parser.Parse(input, "valid.ias")
	doc2, _ := ir.Lower(f2)

	if len(doc1.Resources) != len(doc2.Resources) {
		t.Fatal("resource count mismatch")
	}

	for i := range doc1.Resources {
		if doc1.Resources[i].Hash != doc2.Resources[i].Hash {
			t.Errorf("hash mismatch for %s: %s vs %s",
				doc1.Resources[i].FQN, doc1.Resources[i].Hash, doc2.Resources[i].Hash)
		}
	}
}
