package graph

import (
	"fmt"
	"sort"
	"strings"
)

// mermaidShapes maps entity types to Mermaid node shape syntax.
var mermaidShapes = map[string]struct {
	open, close string
}{
	"agent":      {"([", "])"},
	"prompt":     {"[", "]"},
	"skill":      {"[", "]"},
	"mcp_server": {"{{", "}}"},
	"mcp_client": {"{{", "}}"},
	"pipeline":   {"([", "])"},
	"step":       {"((", "))"},
	"secret":     {"{", "}"},
	"policy":     {"[", "]"},
	"guardrail":  {"[", "]"},
	"user":       {"[", "]"},
	"deploy":     {"[", "]"},
	"binding":    {"[", "]"},
	"state":      {"[(", ")]"},
	"env":        {"[/", "/]"},
	"type":       {"[", "]"},
	"plugin":     {"[", "]"},
	"file":       {"[", "]"},
}

// mermaidColors maps entity types to CSS colors for classDef.
var mermaidColors = map[string]string{
	"agent":      "#4A9EFF",
	"prompt":     "#4ADE80",
	"skill":      "#A78BFA",
	"mcp_server": "#FB923C",
	"mcp_client": "#FBBF24",
	"pipeline":   "#22D3EE",
	"step":       "#2DD4BF",
	"secret":     "#F87171",
	"policy":     "#F472B6",
	"guardrail":  "#FB7185",
	"user":       "#818CF8",
	"deploy":     "#94A3B8",
	"binding":    "#94A3B8",
	"state":      "#34D399",
	"env":        "#A3E635",
	"type":       "#9CA3AF",
	"plugin":     "#C084FC",
	"file":       "#E5E7EB",
}

// RenderMermaid produces a valid Mermaid graph LR with deterministic output.
func RenderMermaid(g *Graph) string {
	var b strings.Builder

	b.WriteString("graph LR\n")

	// Group nodes by source file for subgraphs
	fileNodes := map[string][]GraphNode{}
	var noFileNodes []GraphNode
	for _, n := range g.Nodes {
		if n.File != "" && n.Type != "file" {
			fileNodes[n.File] = append(fileNodes[n.File], n)
		} else {
			noFileNodes = append(noFileNodes, n)
		}
	}

	fileKeys := make([]string, 0, len(fileNodes))
	for k := range fileNodes {
		fileKeys = append(fileKeys, k)
	}
	sort.Strings(fileKeys)

	for _, file := range fileKeys {
		nodes := fileNodes[file]
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

		b.WriteString(fmt.Sprintf("  subgraph %s\n", mermaidID(file)))
		for _, n := range nodes {
			writeMermaidNode(&b, n, "    ")
		}
		b.WriteString("  end\n")
	}

	sort.Slice(noFileNodes, func(i, j int) bool { return noFileNodes[i].ID < noFileNodes[j].ID })
	for _, n := range noFileNodes {
		writeMermaidNode(&b, n, "  ")
	}

	// Pipeline subgraphs
	pipelineSteps := map[string][]GraphNode{}
	for _, n := range g.Nodes {
		if n.Type == "step" {
			for _, e := range g.Edges {
				if e.Target == n.ID && e.Label == "contains" {
					pipelineSteps[e.Source] = append(pipelineSteps[e.Source], n)
					break
				}
			}
		}
	}

	pipeKeys := make([]string, 0, len(pipelineSteps))
	for k := range pipelineSteps {
		pipeKeys = append(pipeKeys, k)
	}
	sort.Strings(pipeKeys)

	for _, pipeID := range pipeKeys {
		steps := pipelineSteps[pipeID]
		sort.Slice(steps, func(i, j int) bool { return steps[i].ID < steps[j].ID })
		// Find pipeline name from nodes
		pipeName := pipeID
		for _, n := range g.Nodes {
			if n.ID == pipeID {
				pipeName = n.Name
				break
			}
		}
		b.WriteString(fmt.Sprintf("  subgraph %s[%q]\n", mermaidID(pipeID), pipeName+" pipeline"))
		for _, s := range steps {
			writeMermaidNode(&b, s, "    ")
		}
		b.WriteString("  end\n")
	}

	b.WriteString("\n")

	// Edges sorted for determinism
	sortedEdges := make([]GraphEdge, len(g.Edges))
	copy(sortedEdges, g.Edges)
	sort.Slice(sortedEdges, func(i, j int) bool {
		if sortedEdges[i].Source != sortedEdges[j].Source {
			return sortedEdges[i].Source < sortedEdges[j].Source
		}
		if sortedEdges[i].Target != sortedEdges[j].Target {
			return sortedEdges[i].Target < sortedEdges[j].Target
		}
		return sortedEdges[i].Label < sortedEdges[j].Label
	})

	for _, e := range sortedEdges {
		arrow := "-->"
		if e.Style == "dashed" {
			arrow = "-.->"
		}
		b.WriteString(fmt.Sprintf("  %s %s|%s| %s\n",
			mermaidID(e.Source), arrow, e.Label, mermaidID(e.Target)))
	}

	// classDef and class assignments
	b.WriteString("\n")
	usedTypes := map[string][]string{}
	for _, n := range g.Nodes {
		usedTypes[n.Type] = append(usedTypes[n.Type], mermaidID(n.ID))
	}

	typeKeys := make([]string, 0, len(usedTypes))
	for k := range usedTypes {
		typeKeys = append(typeKeys, k)
	}
	sort.Strings(typeKeys)

	for _, typ := range typeKeys {
		color := mermaidColors[typ]
		if color == "" {
			color = "#9CA3AF"
		}
		b.WriteString(fmt.Sprintf("  classDef %s fill:%s20,stroke:%s,color:#fff\n", typ, color, color))
		ids := usedTypes[typ]
		sort.Strings(ids)
		b.WriteString(fmt.Sprintf("  class %s %s\n", strings.Join(ids, ","), typ))
	}

	return b.String()
}

func writeMermaidNode(b *strings.Builder, n GraphNode, indent string) {
	shape, ok := mermaidShapes[n.Type]
	if !ok {
		shape = mermaidShapes["type"]
	}

	id := mermaidID(n.ID)
	label := n.Name
	if n.Attributes != nil && n.Attributes["missing"] == "true" {
		label = "? " + label
	}

	b.WriteString(fmt.Sprintf("%s%s%s%q%s\n", indent, id, shape.open, label, shape.close))
}

// mermaidID converts a node ID to a valid Mermaid identifier.
func mermaidID(id string) string {
	r := strings.NewReplacer(
		":", "_",
		".", "_",
		"/", "_",
		" ", "_",
		"*", "x",
		"-", "_",
	)
	return r.Replace(id)
}
