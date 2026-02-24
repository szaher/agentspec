package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adapterlocal "github.com/szaher/designs/agentz/internal/adapters/local"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/plan"
	"github.com/szaher/designs/agentz/internal/validate"
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

// TestDeterminismV2Resources verifies deterministic IR output for all
// IntentLang 2.0 resource types: tool variants, deploy targets, pipelines,
// type definitions, delegation, and error handling.
func TestDeterminismV2Resources(t *testing.T) {
	files := []string{
		"basic-agent.ias",
		"tool-variants.ias",
		"deploy-targets.ias",
		"pipeline.ias",
		"type-definitions.ias",
		"delegation.ias",
		"error-handling.ias",
		"prompt-variables.ias",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			input := readV2TestFile(t, file)

			f1, errs1 := parser.Parse(input, file)
			if len(errs1) > 0 {
				t.Fatalf("parse errors run 1: %v", errs1)
			}
			doc1, err := ir.Lower(f1)
			if err != nil {
				t.Fatalf("lower run 1: %v", err)
			}
			json1, _ := doc1.MarshalJSON()

			f2, errs2 := parser.Parse(input, file)
			if len(errs2) > 0 {
				t.Fatalf("parse errors run 2: %v", errs2)
			}
			doc2, err := ir.Lower(f2)
			if err != nil {
				t.Fatalf("lower run 2: %v", err)
			}
			json2, _ := doc2.MarshalJSON()

			if string(json1) != string(json2) {
				t.Error("IR JSON output is not deterministic across runs")
			}

			// Verify hash determinism
			if len(doc1.Resources) != len(doc2.Resources) {
				t.Fatalf("resource count mismatch: %d vs %d", len(doc1.Resources), len(doc2.Resources))
			}
			for i := range doc1.Resources {
				if doc1.Resources[i].Hash != doc2.Resources[i].Hash {
					t.Errorf("hash mismatch for %s: %s vs %s",
						doc1.Resources[i].FQN, doc1.Resources[i].Hash, doc2.Resources[i].Hash)
				}
			}
		})
	}
}

// TestDeterminismV2Plan verifies deterministic plan output for v2 resources.
func TestDeterminismV2Plan(t *testing.T) {
	files := []string{
		"deploy-targets.ias",
		"pipeline.ias",
		"type-definitions.ias",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			input := readV2TestFile(t, file)

			f, errs := parser.Parse(input, file)
			if len(errs) > 0 {
				t.Fatalf("parse errors: %v", errs)
			}
			doc, err := ir.Lower(f)
			if err != nil {
				t.Fatalf("lower: %v", err)
			}

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
		})
	}
}

// TestExamplesValidate verifies all example .ias files parse, validate, and
// lower to IR without errors.
func TestExamplesValidate(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	examplesDir := filepath.Join(dir, "..", "examples")

	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("read examples dir: %v", err)
	}

	found := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Find .ias files in each example directory
		exDir := filepath.Join(examplesDir, entry.Name())
		files, err := os.ReadDir(exDir)
		if err != nil {
			t.Fatalf("read example dir %s: %v", entry.Name(), err)
		}

		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".ias") {
				continue
			}
			found++
			iasPath := filepath.Join(exDir, f.Name())

			t.Run(entry.Name()+"/"+f.Name(), func(t *testing.T) {
				data, err := os.ReadFile(iasPath)
				if err != nil {
					t.Fatalf("read file: %v", err)
				}

				// Parse
				ast, errs := parser.Parse(string(data), f.Name())
				if len(errs) > 0 {
					t.Fatalf("parse errors: %v", errs)
				}

				// Validate
				structErrs := validate.ValidateStructural(ast)
				semErrs := validate.ValidateSemantic(ast)
				allErrs := append(structErrs, semErrs...)
				if len(allErrs) > 0 {
					for _, e := range allErrs {
						t.Errorf("validation error: %s", e.Error())
					}
					return
				}

				// Lower to IR
				doc, err := ir.Lower(ast)
				if err != nil {
					t.Fatalf("lower to IR: %v", err)
				}

				if len(doc.Resources) == 0 {
					t.Error("expected at least one IR resource")
				}
			})
		}
	}

	if found < 10 {
		t.Errorf("expected at least 10 example .ias files, found %d", found)
	}
}
