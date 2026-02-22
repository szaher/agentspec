package integration_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/formatter"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/validate"
)

func TestParseValidateFormatRoundTrip(t *testing.T) {
	input := readTestFile(t, "testdata/valid.az")

	// Parse
	f, parseErrs := parser.Parse(input, "valid.az")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}
	if f == nil {
		t.Fatal("parsed file is nil")
	}
	if f.Package == nil {
		t.Fatal("parsed package is nil")
	}
	if f.Package.Name != "demo" {
		t.Errorf("expected package name 'demo', got %q", f.Package.Name)
	}

	// Structural validation
	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural validation errors: %v", structErrs)
	}

	// Semantic validation
	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("unexpected semantic validation errors: %v", semErrs)
	}

	// Format and verify idempotency
	formatted1 := formatter.Format(f)
	f2, parseErrs2 := parser.Parse(formatted1, "valid.az")
	if parseErrs2 != nil {
		t.Fatalf("re-parse after format failed: %v", parseErrs2)
	}
	formatted2 := formatter.Format(f2)

	if formatted1 != formatted2 {
		t.Errorf("formatter is not idempotent:\nfirst:\n%s\nsecond:\n%s", formatted1, formatted2)
	}
}

func TestParseValidateInvalidRef(t *testing.T) {
	input := readTestFile(t, "testdata/invalid_ref.az")

	// Parse should succeed
	f, parseErrs := parser.Parse(input, "invalid_ref.az")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	// Structural validation should pass
	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("unexpected structural errors: %v", structErrs)
	}

	// Semantic validation should find the bad reference
	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) == 0 {
		t.Fatal("expected semantic validation errors for bad skill reference")
	}

	// Check for "did you mean" suggestion
	found := false
	for _, e := range semErrs {
		if strings.Contains(e.Message, "web-serch") &&
			strings.Contains(e.Hint, "web-search") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'did you mean' hint for 'web-serch' -> 'web-search', got errors: %v", semErrs)
	}
}

func TestIRLowering(t *testing.T) {
	input := readTestFile(t, "testdata/valid.az")

	f, parseErrs := parser.Parse(input, "valid.az")
	if parseErrs != nil {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering failed: %v", err)
	}

	if doc.Package.Name != "demo" {
		t.Errorf("expected package name 'demo', got %q", doc.Package.Name)
	}
	if doc.IRVersion != "1.0" {
		t.Errorf("expected IR version '1.0', got %q", doc.IRVersion)
	}

	// Check resources are sorted by kind then name
	for i := 1; i < len(doc.Resources); i++ {
		prev := doc.Resources[i-1]
		curr := doc.Resources[i]
		if prev.Kind > curr.Kind || (prev.Kind == curr.Kind && prev.Name > curr.Name) {
			t.Errorf("resources not sorted: %s/%s before %s/%s",
				prev.Kind, prev.Name, curr.Kind, curr.Name)
		}
	}

	// Verify hashes are computed
	for _, r := range doc.Resources {
		if r.Hash == "" {
			t.Errorf("resource %s has empty hash", r.FQN)
		}
		if !strings.HasPrefix(r.Hash, "sha256:") {
			t.Errorf("resource %s hash has wrong format: %s", r.FQN, r.Hash)
		}
	}

	// Verify deterministic IR output
	data1, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("first marshal failed: %v", err)
	}
	data2, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("second marshal failed: %v", err)
	}
	if string(data1) != string(data2) {
		t.Error("IR serialization is not deterministic")
	}

	// Verify bindings are present
	if len(doc.Bindings) != 1 {
		t.Errorf("expected 1 binding, got %d", len(doc.Bindings))
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	absPath := filepath.Join(getTestdataDir(t), filepath.Base(path))
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("failed to read test file %s: %v", path, err)
	}
	return string(data)
}

func getTestdataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "testdata")
}
