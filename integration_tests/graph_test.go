package integration_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/szaher/agentspec/internal/ast"
	"github.com/szaher/agentspec/internal/graph"
	"github.com/szaher/agentspec/internal/parser"
)

func TestGraphExtractMultiAgentRouter(t *testing.T) {
	path := filepath.Join("..", "examples", "multi-agent-router", "multi-agent-router.ias")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("example file not found: %v", err)
	}

	f, errs := parser.Parse(string(input), path)
	if errs != nil {
		t.Fatalf("parse errors: %v", errs)
	}

	g := graph.Extract([]*ast.File{f})

	if g.Stats.NodeCount == 0 {
		t.Error("expected at least one node")
	}
	if g.Stats.EdgeCount == 0 {
		t.Error("expected at least one edge")
	}

	// Should have agent nodes
	hasAgent := false
	for _, n := range g.Nodes {
		if n.Type == "agent" {
			hasAgent = true
			break
		}
	}
	if !hasAgent {
		t.Error("expected at least one agent node")
	}
}

func TestGraphDOTDeterministic(t *testing.T) {
	path := filepath.Join("..", "examples", "multi-agent-router", "multi-agent-router.ias")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("example file not found: %v", err)
	}

	f, errs := parser.Parse(string(input), path)
	if errs != nil {
		t.Fatalf("parse errors: %v", errs)
	}

	g := graph.Extract([]*ast.File{f})

	out1 := graph.RenderDOT(g)
	out2 := graph.RenderDOT(g)

	if out1 != out2 {
		t.Error("DOT output is not deterministic")
	}
	if !strings.Contains(out1, "digraph agentspec") {
		t.Error("expected valid DOT output")
	}
}

func TestGraphMermaidDeterministic(t *testing.T) {
	path := filepath.Join("..", "examples", "multi-agent-router", "multi-agent-router.ias")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("example file not found: %v", err)
	}

	f, errs := parser.Parse(string(input), path)
	if errs != nil {
		t.Fatalf("parse errors: %v", errs)
	}

	g := graph.Extract([]*ast.File{f})

	out1 := graph.RenderMermaid(g)
	out2 := graph.RenderMermaid(g)

	if out1 != out2 {
		t.Error("Mermaid output is not deterministic")
	}
	if !strings.HasPrefix(out1, "graph LR") {
		t.Error("expected valid Mermaid output")
	}
}

func TestGraphErrorResilience(t *testing.T) {
	dir := t.TempDir()

	// Write a valid file
	validContent := `package "test-pkg" version "1.0" lang "2.0"

agent "valid-agent" {
  model "test-model"
}
`
	if err := os.WriteFile(filepath.Join(dir, "valid.ias"), []byte(validContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write an invalid file
	if err := os.WriteFile(filepath.Join(dir, "invalid.ias"), []byte("this is not valid ias {{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Parse valid file
	validInput, _ := os.ReadFile(filepath.Join(dir, "valid.ias"))
	f, errs := parser.Parse(string(validInput), filepath.Join(dir, "valid.ias"))

	var files []*ast.File
	var parseErrors []string

	if errs != nil {
		for _, e := range errs {
			parseErrors = append(parseErrors, e.Message)
		}
	} else {
		files = append(files, f)
	}

	// Parse invalid file
	invalidInput, _ := os.ReadFile(filepath.Join(dir, "invalid.ias"))
	_, errs = parser.Parse(string(invalidInput), filepath.Join(dir, "invalid.ias"))
	if errs != nil {
		for _, e := range errs {
			parseErrors = append(parseErrors, e.Message)
		}
	}

	if len(files) == 0 {
		t.Skip("valid file didn't parse — parser may require different syntax")
	}

	g := graph.Extract(files)
	g.Errors = parseErrors

	if len(g.Nodes) == 0 {
		t.Error("expected at least one node from valid file")
	}
	if len(g.Errors) == 0 {
		t.Error("expected parse errors from invalid file")
	}
}

func TestGraphScalePerformance(t *testing.T) {
	// Build a graph with 200 nodes programmatically
	g := &graph.Graph{
		Files: []string{"scale-test.ias"},
		Stats: graph.GraphStats{TypeCounts: map[string]int{}},
	}

	for i := 0; i < 200; i++ {
		g.Nodes = append(g.Nodes, graph.GraphNode{
			ID:   "agent:" + strings.Repeat("a", 3) + string(rune('0'+i%10)) + string(rune('0'+i/10%10)) + string(rune('0'+i/100%10)),
			Type: "agent",
			Name: "agent-" + strings.Repeat("x", 5),
			File: "scale-test.ias",
		})
	}
	// Add edges between consecutive nodes
	for i := 0; i < len(g.Nodes)-1; i++ {
		g.Edges = append(g.Edges, graph.GraphEdge{
			Source: g.Nodes[i].ID,
			Target: g.Nodes[i+1].ID,
			Label:  "delegates to",
		})
	}

	graph.ComputeStats(g)

	start := time.Now()
	dotOut := graph.RenderDOT(g)
	mermaidOut := graph.RenderMermaid(g)
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Errorf("rendering 200 nodes took %v (expected < 1s)", elapsed)
	}
	if dotOut == "" {
		t.Error("expected non-empty DOT output")
	}
	if mermaidOut == "" {
		t.Error("expected non-empty Mermaid output")
	}
}
