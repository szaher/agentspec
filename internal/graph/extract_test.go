package graph

import (
	"testing"

	"github.com/szaher/agentspec/internal/ast"
)

func TestExtractAgent(t *testing.T) {
	files := []*ast.File{{
		Path: "test.ias",
		Statements: []ast.Statement{
			&ast.Agent{
				Name:     "router",
				Model:    "claude-sonnet-4-20250514",
				Strategy: "router",
				MaxTurns: 5,
				Prompt:   &ast.Ref{Name: "sys"},
				Skills:   []*ast.Ref{{Name: "search"}, {Name: "respond"}},
				Delegates: []*ast.Delegate{
					{AgentRef: "specialist"},
				},
				Fallback:      "fallback-agent",
				GuardrailRefs: []string{"content-filter"},
				Client:        &ast.Ref{Name: "my-client"},
				StartPos:      ast.Pos{Line: 10},
			},
		},
	}}

	g := Extract(files)

	// Check agent node
	agentNode := findNode(g, "agent:router")
	if agentNode == nil {
		t.Fatal("expected agent:router node")
	}
	if agentNode.Attributes["model"] != "claude-sonnet-4-20250514" {
		t.Errorf("expected model claude-sonnet-4-20250514, got %s", agentNode.Attributes["model"])
	}
	if agentNode.Attributes["strategy"] != "router" {
		t.Errorf("expected strategy router, got %s", agentNode.Attributes["strategy"])
	}
	if agentNode.Line != 10 {
		t.Errorf("expected line 10, got %d", agentNode.Line)
	}

	// Check edges
	assertEdge(t, g, "agent:router", "prompt:sys", "uses prompt")
	assertEdge(t, g, "agent:router", "skill:search", "uses skill")
	assertEdge(t, g, "agent:router", "skill:respond", "uses skill")
	assertEdge(t, g, "agent:router", "agent:specialist", "delegates to")
	assertEdge(t, g, "agent:router", "agent:fallback-agent", "fallback")
	assertEdge(t, g, "agent:router", "guardrail:content-filter", "uses guardrail")
	assertEdge(t, g, "agent:router", "mcp_client:my-client", "uses client")
}

func TestExtractPrompt(t *testing.T) {
	files := []*ast.File{{
		Path: "test.ias",
		Statements: []ast.Statement{
			&ast.Prompt{
				Name:    "greeting",
				Content: "Hello, {{name}}!",
				Variables: []*ast.Variable{
					{Name: "name", Type: "string"},
				},
			},
		},
	}}
	g := Extract(files)

	node := findNode(g, "prompt:greeting")
	if node == nil {
		t.Fatal("expected prompt:greeting node")
	}
	if node.Attributes["content_preview"] != "Hello, {{name}}!" {
		t.Errorf("unexpected content preview: %s", node.Attributes["content_preview"])
	}
	if node.Attributes["variables"] != "name" {
		t.Errorf("unexpected variables: %s", node.Attributes["variables"])
	}
}

func TestExtractSkillWithMCPTool(t *testing.T) {
	files := []*ast.File{{
		Path: "test.ias",
		Statements: []ast.Statement{
			&ast.Skill{
				Name:        "search",
				Description: "Search docs",
				ToolConfig: &ast.ToolConfig{
					Type:       "mcp",
					ServerTool: "search-server/query",
				},
				Input:  []*ast.Field{{Name: "q", Type: "string"}},
				Output: []*ast.Field{{Name: "results", Type: "list"}},
			},
		},
	}}
	g := Extract(files)

	node := findNode(g, "skill:search")
	if node == nil {
		t.Fatal("expected skill:search node")
	}
	if node.Attributes["tool_type"] != "mcp" {
		t.Errorf("expected tool_type mcp, got %s", node.Attributes["tool_type"])
	}
	assertEdge(t, g, "skill:search", "mcp_server:search-server", "uses tool")
}

func TestExtractMCPClientServers(t *testing.T) {
	files := []*ast.File{{
		Statements: []ast.Statement{
			&ast.MCPServer{Name: "server1", Transport: "stdio"},
			&ast.MCPClient{
				Name:    "client1",
				Servers: []*ast.Ref{{Name: "server1"}, {Name: "server2"}},
			},
		},
	}}
	g := Extract(files)
	assertEdge(t, g, "mcp_client:client1", "mcp_server:server1", "connects to")
	assertEdge(t, g, "mcp_client:client1", "mcp_server:server2", "connects to")
}

func TestExtractPipeline(t *testing.T) {
	files := []*ast.File{{
		Statements: []ast.Statement{
			&ast.Pipeline{
				Name: "research",
				Steps: []*ast.PipelineStep{
					{Name: "gather", Agent: "web-agent", Parallel: true},
					{Name: "synthesize", Agent: "synth-agent", DependsOn: []string{"gather"}},
				},
			},
		},
	}}
	g := Extract(files)

	if findNode(g, "pipeline:research") == nil {
		t.Fatal("expected pipeline:research node")
	}
	if findNode(g, "step:research.gather") == nil {
		t.Fatal("expected step:research.gather node")
	}
	assertEdge(t, g, "pipeline:research", "step:research.gather", "contains")
	assertEdge(t, g, "step:research.gather", "agent:web-agent", "invokes")
	assertEdge(t, g, "step:research.synthesize", "step:research.gather", "depends on")
}

func TestExtractPolicyGovernance(t *testing.T) {
	files := []*ast.File{{
		Statements: []ast.Statement{
			&ast.Policy{
				Name: "security",
				Rules: []*ast.Rule{
					{Action: "deny", Resource: "agent/*", Subject: "exec"},
				},
			},
		},
	}}
	g := Extract(files)
	assertEdge(t, g, "policy:security", "policy_target:agent/*", "governs")
}

func TestExtractUserAccess(t *testing.T) {
	files := []*ast.File{{
		Statements: []ast.Statement{
			&ast.User{
				Name:   "alice",
				Role:   "admin",
				Agents: []string{"router", "specialist"},
			},
		},
	}}
	g := Extract(files)
	assertEdge(t, g, "user:alice", "agent:router", "can access")
	assertEdge(t, g, "user:alice", "agent:specialist", "can access")
}

func TestExtractEnvironmentOverrides(t *testing.T) {
	files := []*ast.File{{
		Statements: []ast.Statement{
			&ast.Environment{
				Name: "staging",
				Overrides: []*ast.Override{
					{Resource: "agent/router", Attribute: "model", Value: "gpt-4"},
				},
			},
		},
	}}
	g := Extract(files)
	assertEdge(t, g, "env:staging", "env_target:agent/router", "overrides")
}

func TestExtractAllEntityTypes(t *testing.T) {
	files := []*ast.File{{
		Path: "test.ias",
		Package: &ast.Package{
			Name:    "test-pkg",
			Version: "1.0",
			Plugins: []*ast.PluginRef{{Name: "custom-plugin", Version: "0.1"}},
		},
		Statements: []ast.Statement{
			&ast.Agent{Name: "a1"},
			&ast.Prompt{Name: "p1"},
			&ast.Skill{Name: "s1"},
			&ast.MCPServer{Name: "ms1"},
			&ast.MCPClient{Name: "mc1"},
			&ast.Pipeline{Name: "pipe1", Steps: []*ast.PipelineStep{{Name: "step1"}}},
			&ast.Secret{Name: "sec1"},
			&ast.Policy{Name: "pol1"},
			&ast.Guardrail{Name: "guard1"},
			&ast.User{Name: "user1"},
			&ast.Binding{Name: "bind1"},
			&ast.DeployTarget{Name: "dep1"},
			&ast.StateConfig{Name: "st1", Type: "local"},
			&ast.Environment{Name: "env1"},
			&ast.TypeDef{Name: "type1"},
		},
	}}
	g := Extract(files)

	expectedTypes := []string{
		"agent", "prompt", "skill", "mcp_server", "mcp_client",
		"pipeline", "step", "secret", "policy", "guardrail",
		"user", "binding", "deploy", "state", "env", "type", "plugin",
	}
	for _, typ := range expectedTypes {
		found := false
		for _, n := range g.Nodes {
			if n.Type == typ {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing entity type: %s", typ)
		}
	}

	if g.Package.Name != "test-pkg" {
		t.Errorf("expected package name test-pkg, got %s", g.Package.Name)
	}
}

func TestUnresolvedReferences(t *testing.T) {
	files := []*ast.File{{
		Statements: []ast.Statement{
			&ast.Agent{
				Name:   "a1",
				Prompt: &ast.Ref{Name: "missing-prompt"},
			},
		},
	}}
	g := Extract(files)

	node := findNode(g, "prompt:missing-prompt")
	if node == nil {
		t.Fatal("expected placeholder node for missing-prompt")
	}
	if node.Attributes["missing"] != "true" {
		t.Error("expected missing=true attribute on placeholder node")
	}
}

func TestComputeStats(t *testing.T) {
	files := []*ast.File{{
		Path: "test.ias",
		Statements: []ast.Statement{
			&ast.Agent{Name: "a1"},
			&ast.Agent{Name: "a2"},
			&ast.Prompt{Name: "p1"},
		},
	}}
	g := Extract(files)

	if g.Stats.NodeCount != 3 {
		t.Errorf("expected 3 nodes, got %d", g.Stats.NodeCount)
	}
	if g.Stats.TypeCounts["agent"] != 2 {
		t.Errorf("expected 2 agents, got %d", g.Stats.TypeCounts["agent"])
	}
	if g.Stats.FileCount != 1 {
		t.Errorf("expected 1 file, got %d", g.Stats.FileCount)
	}
}

func TestFilterOrphans(t *testing.T) {
	files := []*ast.File{{
		Statements: []ast.Statement{
			&ast.Agent{Name: "connected", Prompt: &ast.Ref{Name: "p1"}},
			&ast.Prompt{Name: "p1"},
			&ast.Secret{Name: "orphan"},
		},
	}}
	g := Extract(files)
	FilterOrphans(g)

	if findNode(g, "secret:orphan") != nil {
		t.Error("expected orphan node to be removed")
	}
	if findNode(g, "agent:connected") == nil {
		t.Error("expected connected agent to remain")
	}
}

func TestPromptContentTruncation(t *testing.T) {
	longContent := ""
	for i := 0; i < 250; i++ {
		longContent += "x"
	}
	files := []*ast.File{{
		Statements: []ast.Statement{
			&ast.Prompt{Name: "long", Content: longContent},
		},
	}}
	g := Extract(files)
	node := findNode(g, "prompt:long")
	if node == nil {
		t.Fatal("expected prompt:long node")
	}
	preview := node.Attributes["content_preview"]
	if len(preview) != 203 { // 200 + "..."
		t.Errorf("expected truncated preview of 203 chars, got %d", len(preview))
	}
}

func TestAddFileNodes(t *testing.T) {
	files := []*ast.File{
		{
			Path: "main.ias",
			Package: &ast.Package{
				Imports: []*ast.Import{{Path: "skills.ias"}},
			},
			Statements: []ast.Statement{
				&ast.Agent{Name: "router"},
			},
		},
		{
			Path: "skills.ias",
			Statements: []ast.Statement{
				&ast.Skill{Name: "search"},
			},
		},
	}

	g := Extract(files)
	AddFileNodes(g, files)

	// File nodes
	if findNode(g, "file:main.ias") == nil {
		t.Error("expected file:main.ias node")
	}
	if findNode(g, "file:skills.ias") == nil {
		t.Error("expected file:skills.ias node")
	}

	// Defines edges
	assertEdge(t, g, "file:main.ias", "agent:router", "defines")
	assertEdge(t, g, "file:skills.ias", "skill:search", "defines")

	// Imports edge
	assertEdge(t, g, "file:main.ias", "file:skills.ias", "imports")
}

func TestAddFileNodesDeduplicate(t *testing.T) {
	files := []*ast.File{
		{Path: "main.ias", Statements: []ast.Statement{&ast.Agent{Name: "a"}}},
		{Path: "main.ias", Statements: []ast.Statement{&ast.Agent{Name: "a"}}},
	}

	g := Extract(files)
	AddFileNodes(g, files)

	count := 0
	for _, n := range g.Nodes {
		if n.ID == "file:main.ias" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 file:main.ias node, got %d", count)
	}
}

// --- Helpers ---

func findNode(g *Graph, id string) *GraphNode {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i]
		}
	}
	return nil
}

func assertEdge(t *testing.T, g *Graph, source, target, label string) {
	t.Helper()
	for _, e := range g.Edges {
		if e.Source == source && e.Target == target && e.Label == label {
			return
		}
	}
	t.Errorf("expected edge %s -[%s]-> %s", source, label, target)
}
