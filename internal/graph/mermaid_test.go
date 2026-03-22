package graph

import (
	"strings"
	"testing"
)

func TestRenderMermaidBasic(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:router", Type: "agent", Name: "router", File: "main.ias"},
			{ID: "skill:search", Type: "skill", Name: "search", File: "main.ias"},
		},
		Edges: []GraphEdge{
			{Source: "agent:router", Target: "skill:search", Label: "uses skill", Style: "solid"},
		},
	}

	out := RenderMermaid(g)

	if !strings.HasPrefix(out, "graph LR") {
		t.Error("expected graph LR header")
	}
	if !strings.Contains(out, "agent_router") {
		t.Error("expected agent_router node")
	}
	if !strings.Contains(out, "skill_search") {
		t.Error("expected skill_search node")
	}
	if !strings.Contains(out, "-->|uses skill|") {
		t.Error("expected edge with label")
	}
}

func TestRenderMermaidShapes(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:a", Type: "agent", Name: "a"},
			{ID: "mcp_server:s", Type: "mcp_server", Name: "s"},
			{ID: "step:st", Type: "step", Name: "st"},
			{ID: "state:db", Type: "state", Name: "db"},
		},
	}

	out := RenderMermaid(g)

	// Agent uses ([ ]) stadium shape
	if !strings.Contains(out, "([") {
		t.Error("expected agent to use stadium shape ([...])")
	}
	// MCP server uses {{ }} hexagon
	if !strings.Contains(out, "{{") {
		t.Error("expected mcp_server to use hexagon shape {{...}}")
	}
	// Step uses (( )) circle
	if !strings.Contains(out, "((") {
		t.Error("expected step to use circle shape ((...)))")
	}
	// State uses [( )] cylindrical
	if !strings.Contains(out, "[(") {
		t.Error("expected state to use cylindrical shape [(...)]")
	}
}

func TestRenderMermaidDashedEdge(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:a", Type: "agent", Name: "a"},
			{ID: "agent:b", Type: "agent", Name: "b"},
		},
		Edges: []GraphEdge{
			{Source: "agent:a", Target: "agent:b", Label: "delegates to", Style: "dashed"},
		},
	}

	out := RenderMermaid(g)
	if !strings.Contains(out, "-.->") {
		t.Error("expected dashed arrow -.->")
	}
}

func TestRenderMermaidClassDef(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:a", Type: "agent", Name: "a"},
			{ID: "agent:b", Type: "agent", Name: "b"},
		},
	}

	out := RenderMermaid(g)
	if !strings.Contains(out, "classDef agent") {
		t.Error("expected classDef for agent type")
	}
	if !strings.Contains(out, "class agent_a,agent_b agent") {
		t.Error("expected class assignment for agent nodes")
	}
}

func TestRenderMermaidDeterministic(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "skill:z", Type: "skill", Name: "z"},
			{ID: "agent:a", Type: "agent", Name: "a"},
		},
		Edges: []GraphEdge{
			{Source: "agent:a", Target: "skill:z", Label: "uses skill"},
		},
	}

	out1 := RenderMermaid(g)
	out2 := RenderMermaid(g)

	if out1 != out2 {
		t.Error("expected deterministic output")
	}
}

func TestRenderMermaidSubgraph(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:a", Type: "agent", Name: "a", File: "main.ias"},
			{ID: "agent:b", Type: "agent", Name: "b", File: "other.ias"},
		},
	}

	out := RenderMermaid(g)
	if !strings.Contains(out, "subgraph main_ias") {
		t.Error("expected main.ias subgraph")
	}
	if !strings.Contains(out, "subgraph other_ias") {
		t.Error("expected other.ias subgraph")
	}
}
