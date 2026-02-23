package integration_tests

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	adaptercompose "github.com/szaher/designs/agentz/internal/adapters/compose"
	adapterlocal "github.com/szaher/designs/agentz/internal/adapters/local"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
)

func TestExportLocalMCP(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")
	f, parseErrs := parser.Parse(input, "valid.ias")
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering: %v", err)
	}

	adapter := &adapterlocal.Adapter{}
	outDir := filepath.Join(t.TempDir(), "local-export")

	if err := adapter.Export(context.Background(), doc.Resources, outDir); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Verify expected files
	for _, name := range []string{"mcp-servers.json", "mcp-clients.json", "agents.json"} {
		path := filepath.Join(outDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected file %s not found: %v", name, err)
		}
	}

	// Verify determinism: re-export should produce identical files
	outDir2 := filepath.Join(t.TempDir(), "local-export-2")
	if err := adapter.Export(context.Background(), doc.Resources, outDir2); err != nil {
		t.Fatalf("second export failed: %v", err)
	}

	for _, name := range []string{"mcp-servers.json", "mcp-clients.json", "agents.json"} {
		data1, _ := os.ReadFile(filepath.Join(outDir, name))
		data2, _ := os.ReadFile(filepath.Join(outDir2, name))
		if string(data1) != string(data2) {
			t.Errorf("file %s is not deterministic across exports", name)
		}
	}
}

func TestExportDockerCompose(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")
	f, parseErrs := parser.Parse(input, "valid.ias")
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering: %v", err)
	}

	adapter := &adaptercompose.Adapter{}
	outDir := filepath.Join(t.TempDir(), "compose-export")

	if err := adapter.Export(context.Background(), doc.Resources, outDir); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// Verify expected files
	composeFile := filepath.Join(outDir, "docker-compose.yml")
	if _, err := os.Stat(composeFile); err != nil {
		t.Errorf("docker-compose.yml not found: %v", err)
	}

	configDir := filepath.Join(outDir, "config")
	if _, err := os.Stat(configDir); err != nil {
		t.Errorf("config/ directory not found: %v", err)
	}

	// Verify determinism
	outDir2 := filepath.Join(t.TempDir(), "compose-export-2")
	if err := adapter.Export(context.Background(), doc.Resources, outDir2); err != nil {
		t.Fatalf("second export failed: %v", err)
	}

	data1, _ := os.ReadFile(composeFile)
	data2, _ := os.ReadFile(filepath.Join(outDir2, "docker-compose.yml"))
	if string(data1) != string(data2) {
		t.Error("docker-compose.yml is not deterministic across exports")
	}
}

func TestExportBothAdaptersFromSameSource(t *testing.T) {
	input := readTestFile(t, "testdata/valid.ias")
	f, parseErrs := parser.Parse(input, "valid.ias")
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering: %v", err)
	}

	localDir := filepath.Join(t.TempDir(), "local")
	composeDir := filepath.Join(t.TempDir(), "compose")

	localAdapter := &adapterlocal.Adapter{}
	composeAdapter := &adaptercompose.Adapter{}

	if err := localAdapter.Export(context.Background(), doc.Resources, localDir); err != nil {
		t.Fatalf("local export failed: %v", err)
	}
	if err := composeAdapter.Export(context.Background(), doc.Resources, composeDir); err != nil {
		t.Fatalf("compose export failed: %v", err)
	}

	// Both should produce artifacts but they should be different
	localFiles, _ := os.ReadDir(localDir)
	composeFiles, _ := os.ReadDir(composeDir)

	if len(localFiles) == 0 {
		t.Error("local export produced no files")
	}
	if len(composeFiles) == 0 {
		t.Error("compose export produced no files")
	}

	// Local should have .json files, compose should have .yml
	hasJSON := false
	for _, f := range localFiles {
		if filepath.Ext(f.Name()) == ".json" {
			hasJSON = true
		}
	}
	if !hasJSON {
		t.Error("local export should produce .json files")
	}

	hasYML := false
	for _, f := range composeFiles {
		if filepath.Ext(f.Name()) == ".yml" {
			hasYML = true
		}
	}
	if !hasYML {
		t.Error("compose export should produce .yml files")
	}
}
