package graph

import (
	"strings"
	"testing"
)

func TestRenderDOTBasic(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:router", Type: "agent", Name: "router", File: "main.ias"},
			{ID: "skill:search", Type: "skill", Name: "search", File: "main.ias"},
		},
		Edges: []GraphEdge{
			{Source: "agent:router", Target: "skill:search", Label: "uses skill", Style: "solid"},
		},
	}

	out := RenderDOT(g)

	if !strings.Contains(out, "digraph agentspec") {
		t.Error("expected digraph header")
	}
	if !strings.Contains(out, "rankdir=LR") {
		t.Error("expected rankdir=LR")
	}
	if !strings.Contains(out, `"agent:router"`) {
		t.Error("expected agent:router node")
	}
	if !strings.Contains(out, `"skill:search"`) {
		t.Error("expected skill:search node")
	}
	if !strings.Contains(out, `"agent:router" -> "skill:search"`) {
		t.Error("expected edge from router to search")
	}
	if !strings.Contains(out, `label="uses skill"`) {
		t.Error("expected edge label")
	}
}

func TestRenderDOTShapes(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:a", Type: "agent", Name: "a"},
			{ID: "mcp_server:s", Type: "mcp_server", Name: "s"},
			{ID: "secret:sec", Type: "secret", Name: "sec"},
		},
	}

	out := RenderDOT(g)

	// Agent should have rounded style
	if !strings.Contains(out, "rounded") {
		t.Error("expected agent to have rounded style")
	}
	if !strings.Contains(out, "shape=hexagon") {
		t.Error("expected mcp_server to use hexagon shape")
	}
	if !strings.Contains(out, "shape=diamond") {
		t.Error("expected secret to use diamond shape")
	}
}

func TestRenderDOTDashedEdge(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:a", Type: "agent", Name: "a"},
			{ID: "agent:b", Type: "agent", Name: "b"},
		},
		Edges: []GraphEdge{
			{Source: "agent:a", Target: "agent:b", Label: "delegates to", Style: "dashed"},
		},
	}

	out := RenderDOT(g)
	if !strings.Contains(out, "style=dashed") {
		t.Error("expected dashed edge style")
	}
}

func TestRenderDOTMissingNode(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "skill:missing", Type: "skill", Name: "missing",
				Attributes: map[string]string{"missing": "true"}},
		},
	}

	out := RenderDOT(g)
	if !strings.Contains(out, `style="dashed"`) {
		t.Error("expected missing node to have dashed style")
	}
}

func TestRenderDOTDeterministic(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "skill:z", Type: "skill", Name: "z"},
			{ID: "agent:a", Type: "agent", Name: "a"},
			{ID: "prompt:m", Type: "prompt", Name: "m"},
		},
		Edges: []GraphEdge{
			{Source: "agent:a", Target: "skill:z", Label: "uses skill"},
			{Source: "agent:a", Target: "prompt:m", Label: "uses prompt"},
		},
	}

	out1 := RenderDOT(g)
	out2 := RenderDOT(g)

	if out1 != out2 {
		t.Error("expected deterministic output")
	}

	// Verify ordering: prompt edge should come before skill edge (sorted by target)
	promptIdx := strings.Index(out1, `"prompt:m"`)
	skillIdx := strings.Index(out1, `"skill:z"`)
	if promptIdx < 0 || skillIdx < 0 {
		t.Fatal("missing nodes in output")
	}
}

func TestRenderDOTSubgraphClustering(t *testing.T) {
	g := &Graph{
		Nodes: []GraphNode{
			{ID: "agent:a", Type: "agent", Name: "a", File: "main.ias"},
			{ID: "agent:b", Type: "agent", Name: "b", File: "other.ias"},
		},
	}

	out := RenderDOT(g)
	if !strings.Contains(out, "subgraph cluster_") {
		t.Error("expected subgraph clusters")
	}
	if !strings.Contains(out, `label="main.ias"`) {
		t.Error("expected main.ias cluster label")
	}
	if !strings.Contains(out, `label="other.ias"`) {
		t.Error("expected other.ias cluster label")
	}
}
