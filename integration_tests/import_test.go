package integration_tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/szaher/designs/agentz/internal/imports"
	"github.com/szaher/designs/agentz/internal/parser"
)

func TestImportResolverLocalFile(t *testing.T) {
	// Create temp directory with multiple .ias files
	tmpDir := t.TempDir()

	// Write skill file
	skillContent := `package "skills" version "1.0.0" lang "3.0"

skill "helper" {
  description "A helper skill"

  input {
    query string required
  }

  output {
    result string required
  }

  tool command {
    binary "echo"
    args "helper result"
  }
}
`
	skillPath := filepath.Join(tmpDir, "helper.ias")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		t.Fatalf("writing skill file: %v", err)
	}

	// Write main file that imports the skill
	mainContent := `package "test-pkg" version "1.0.0" lang "3.0"

import "./helper.ias"

prompt "sys" {
  content "You are a test agent."
}

agent "test-agent" {
  model "claude-sonnet-4-20250514"
  prompt "sys"
  strategy "react"
  max_turns 5

  uses skill "helper"
}
`
	mainPath := filepath.Join(tmpDir, "main.ias")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("writing main file: %v", err)
	}

	// Parse the main file
	f, parseErrs := parser.Parse(mainContent, mainPath)
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	// Resolve imports
	resolver := imports.NewResolver(tmpDir, nil)
	resolved, err := resolver.ResolveAll(f)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved import, got %d", len(resolved))
	}

	ri := resolved[0]
	if ri.Kind != "local" {
		t.Errorf("expected kind 'local', got %q", ri.Kind)
	}
	if ri.Path != skillPath {
		t.Errorf("expected path %q, got %q", skillPath, ri.Path)
	}
	if ri.Hash == "" {
		t.Error("expected non-empty hash")
	}
	if len(ri.Resources) != 1 || ri.Resources[0] != "Skill/helper" {
		t.Errorf("expected resources [Skill/helper], got %v", ri.Resources)
	}
}

func TestImportResolverAutoExtension(t *testing.T) {
	// Test that resolver adds .ias extension automatically
	tmpDir := t.TempDir()

	skillContent := `package "skills" version "1.0.0" lang "3.0"

skill "auto" {
  description "Auto-extension test"

  input {
    x string required
  }

  output {
    y string required
  }

  tool command {
    binary "echo"
    args "auto"
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "auto.ias"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("writing skill file: %v", err)
	}

	// Import without .ias extension
	mainContent := `package "test" version "1.0.0" lang "3.0"

import "./auto"

agent "a" {
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 5
  uses skill "auto"
}
`
	mainPath := filepath.Join(tmpDir, "main.ias")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatalf("writing main file: %v", err)
	}

	f, parseErrs := parser.Parse(mainContent, mainPath)
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	resolver := imports.NewResolver(tmpDir, nil)
	resolved, err := resolver.ResolveAll(f)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved import, got %d", len(resolved))
	}
}

func TestImportCircularDependencyDetection(t *testing.T) {
	// Test cycle detection at the graph level directly
	// (The resolver breaks recursion via its visited set, so we test the graph independently)
	graph := imports.NewGraph()
	graph.AddNode("a.ias", "hash_a", []string{"b.ias"})
	graph.AddNode("b.ias", "hash_b", []string{"a.ias"})

	cycles := graph.DetectCycles()
	if len(cycles) == 0 {
		t.Error("expected circular dependency to be detected")
	}

	// Verify topological sort fails for cyclic graphs
	_, err := graph.TopologicalSort()
	if err == nil {
		t.Error("expected topological sort to fail for cyclic graph")
	}
}

func TestImportCircularDependencyInFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// File A imports B
	fileA := `package "a" version "1.0.0" lang "3.0"

import "./b.ias"

skill "skill_a" {
  description "Skill A"

  input {
    x string required
  }

  output {
    y string required
  }

  tool command {
    binary "echo"
    args "a"
  }
}
`
	// File B imports A (circular!)
	fileB := `package "b" version "1.0.0" lang "3.0"

import "./a.ias"

skill "skill_b" {
  description "Skill B"

  input {
    x string required
  }

  output {
    y string required
  }

  tool command {
    binary "echo"
    args "b"
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "a.ias"), []byte(fileA), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.ias"), []byte(fileB), 0644); err != nil {
		t.Fatal(err)
	}

	f, parseErrs := parser.Parse(fileA, filepath.Join(tmpDir, "a.ias"))
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	// The resolver should handle circular imports without stack overflow
	resolver := imports.NewResolver(tmpDir, nil)
	resolved, err := resolver.ResolveAll(f)
	if err != nil {
		t.Fatalf("resolve error (should not fail — cycles are detected at graph level): %v", err)
	}

	// The resolved list should contain both files (resolver visits both)
	if len(resolved) < 1 {
		t.Errorf("expected at least 1 resolved import, got %d", len(resolved))
	}
}

func TestImportMissingFileError(t *testing.T) {
	tmpDir := t.TempDir()

	mainContent := `package "test" version "1.0.0" lang "3.0"

import "./nonexistent.ias"

agent "a" {
  model "claude-sonnet-4-20250514"
  strategy "react"
  max_turns 5
}
`
	mainPath := filepath.Join(tmpDir, "main.ias")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	f, parseErrs := parser.Parse(mainContent, mainPath)
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	resolver := imports.NewResolver(tmpDir, nil)
	_, err := resolver.ResolveAll(f)
	if err == nil {
		t.Fatal("expected error for missing import file")
	}
}

func TestImportMerge(t *testing.T) {
	tmpDir := t.TempDir()

	skillContent := `package "ext" version "1.0.0" lang "3.0"

prompt "ext-prompt" {
  content "External prompt content"
}

skill "ext_skill" {
  description "External skill"

  input {
    q string required
  }

  output {
    r string required
  }

  tool command {
    binary "echo"
    args "ext result"
  }
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "ext.ias"), []byte(skillContent), 0644); err != nil {
		t.Fatal(err)
	}

	mainContent := `package "main" version "1.0.0" lang "3.0"

import "./ext.ias"

prompt "main-prompt" {
  content "Main prompt content"
}

agent "main-agent" {
  model "claude-sonnet-4-20250514"
  prompt "main-prompt"
  strategy "react"
  max_turns 5
  uses skill "ext_skill"
}
`
	mainPath := filepath.Join(tmpDir, "main.ias")
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	f, parseErrs := parser.Parse(mainContent, mainPath)
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	resolver := imports.NewResolver(tmpDir, nil)
	resolved, err := resolver.ResolveAll(f)
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	// Merge imports
	imports.MergeImports(f, resolved)

	// Count statements (should include merged ones)
	// Original: import, prompt "main-prompt", agent "main-agent" = 3
	// Merged: prompt "ext-prompt", skill "ext_skill" = 2
	// Total: 5
	if len(f.Statements) != 5 {
		t.Errorf("expected 5 statements after merge, got %d", len(f.Statements))
	}
}

func TestDependencyGraphTopologicalSort(t *testing.T) {
	graph := imports.NewGraph()
	graph.AddNode("c.ias", "hash_c", nil)
	graph.AddNode("b.ias", "hash_b", []string{"c.ias"})
	graph.AddNode("a.ias", "hash_a", []string{"b.ias", "c.ias"})

	sorted, err := graph.TopologicalSort()
	if err != nil {
		t.Fatalf("sort error: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 sorted nodes, got %d", len(sorted))
	}

	// c.ias should come before b.ias and a.ias
	cIdx, bIdx, aIdx := -1, -1, -1
	for i, s := range sorted {
		switch s {
		case "c.ias":
			cIdx = i
		case "b.ias":
			bIdx = i
		case "a.ias":
			aIdx = i
		}
	}

	if cIdx > bIdx {
		t.Error("c.ias should come before b.ias")
	}
	if bIdx > aIdx {
		t.Error("b.ias should come before a.ias")
	}
}

func TestLockFileRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	lf := &imports.LockFile{
		Version: "1",
		Dependencies: []imports.LockedDep{
			{Source: "./skills/search.ias", Hash: "sha256:abc123", Path: "/tmp/search.ias"},
			{Source: "./skills/respond.ias", Hash: "sha256:def456", Path: "/tmp/respond.ias"},
		},
	}

	// Write
	if err := imports.WriteLockFile(tmpDir, lf); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Read
	lf2, err := imports.ReadLockFile(tmpDir)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if lf2.Version != "1" {
		t.Errorf("expected version '1', got %q", lf2.Version)
	}
	if len(lf2.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(lf2.Dependencies))
	}
}

func TestMVSResolution(t *testing.T) {
	constraints := []imports.VersionConstraint{
		{Package: "github.com/example/tools", MinVersion: "1.2.0", RequiredBy: "main"},
		{Package: "github.com/example/tools", MinVersion: "1.3.0", RequiredBy: "helper"},
		{Package: "github.com/example/utils", MinVersion: "2.0.0", RequiredBy: "main"},
		{Package: "github.com/example/utils", MinVersion: "1.5.0", RequiredBy: "helper"},
	}

	resolved, err := imports.MVS(constraints)
	if err != nil {
		t.Fatalf("MVS error: %v", err)
	}

	if len(resolved) != 2 {
		t.Fatalf("expected 2 resolved, got %d", len(resolved))
	}

	// tools should resolve to 1.3.0 (higher of 1.2.0 and 1.3.0)
	for _, r := range resolved {
		switch r.Package {
		case "github.com/example/tools":
			if r.Version != "1.3.0" {
				t.Errorf("expected tools 1.3.0, got %s", r.Version)
			}
		case "github.com/example/utils":
			if r.Version != "2.0.0" {
				t.Errorf("expected utils 2.0.0, got %s", r.Version)
			}
		}
	}
}
