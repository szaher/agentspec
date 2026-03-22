package graph

import (
	"fmt"
	"sort"
	"strings"
)

// dotShapes maps entity types to Graphviz DOT node shapes.
var dotShapes = map[string]string{
	"agent":      "box",
	"prompt":     "note",
	"skill":      "component",
	"mcp_server": "hexagon",
	"mcp_client": "hexagon",
	"pipeline":   "tab",
	"step":       "circle",
	"secret":     "diamond",
	"policy":     "house",
	"guardrail":  "octagon",
	"user":       "invhouse",
	"deploy":     "folder",
	"binding":    "folder",
	"state":      "cylinder",
	"env":        "parallelogram",
	"type":       "rect",
	"plugin":     "pentagon",
	"file":       "folder",
}

// dotColors maps entity types to colors for DOT output.
var dotColors = map[string]string{
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

// RenderDOT produces a valid Graphviz DOT digraph with deterministic output.
func RenderDOT(g *Graph) string {
	var b strings.Builder

	b.WriteString("digraph agentspec {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [fontname=\"Helvetica\" fontsize=10];\n")
	b.WriteString("  edge [fontname=\"Helvetica\" fontsize=8];\n\n")

	// Group nodes by source file for subgraph clusters
	fileNodes := map[string][]GraphNode{}
	var noFileNodes []GraphNode
	for _, n := range g.Nodes {
		if n.File != "" && n.Type != "file" {
			fileNodes[n.File] = append(fileNodes[n.File], n)
		} else {
			noFileNodes = append(noFileNodes, n)
		}
	}

	// Sort file keys for determinism
	fileKeys := make([]string, 0, len(fileNodes))
	for k := range fileNodes {
		fileKeys = append(fileKeys, k)
	}
	sort.Strings(fileKeys)

	clusterIdx := 0
	for _, file := range fileKeys {
		nodes := fileNodes[file]
		sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })

		b.WriteString(fmt.Sprintf("  subgraph cluster_%d {\n", clusterIdx))
		b.WriteString(fmt.Sprintf("    label=%q;\n", file))
		b.WriteString("    style=dashed;\n")
		b.WriteString("    color=\"#475569\";\n")
		for _, n := range nodes {
			writeNode(&b, n, "    ")
		}
		b.WriteString("  }\n\n")
		clusterIdx++
	}

	// Nodes without a file
	sort.Slice(noFileNodes, func(i, j int) bool { return noFileNodes[i].ID < noFileNodes[j].ID })
	for _, n := range noFileNodes {
		writeNode(&b, n, "  ")
	}
	if len(noFileNodes) > 0 {
		b.WriteString("\n")
	}

	// Pipeline subgraphs with rank=same for parallel steps
	pipelineSteps := map[string][]GraphNode{} // pipeline name -> steps
	stepDeps := map[string][]string{}         // step ID -> dependency step IDs
	for _, n := range g.Nodes {
		if n.Type == "step" {
			// Find the pipeline this step belongs to via "contains" edge
			for _, e := range g.Edges {
				if e.Target == n.ID && e.Label == "contains" {
					pipelineName := e.Source
					pipelineSteps[pipelineName] = append(pipelineSteps[pipelineName], n)
					break
				}
			}
		}
	}
	for _, e := range g.Edges {
		if e.Label == "depends on" {
			stepDeps[e.Source] = append(stepDeps[e.Source], e.Target)
		}
	}

	pipeKeys := make([]string, 0, len(pipelineSteps))
	for k := range pipelineSteps {
		pipeKeys = append(pipeKeys, k)
	}
	sort.Strings(pipeKeys)

	for _, pipeID := range pipeKeys {
		steps := pipelineSteps[pipeID]
		// Group by dependency level for rank=same
		levels := groupByDepLevel(steps, stepDeps)
		for _, level := range levels {
			if len(level) > 1 {
				b.WriteString("  { rank=same;")
				sort.Slice(level, func(i, j int) bool { return level[i].ID < level[j].ID })
				for _, n := range level {
					b.WriteString(fmt.Sprintf(" %q;", n.ID))
				}
				b.WriteString(" }\n")
			}
		}
	}
	if len(pipelineSteps) > 0 {
		b.WriteString("\n")
	}

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
		style := ""
		if e.Style == "dashed" {
			style = " style=dashed"
		}
		b.WriteString(fmt.Sprintf("  %q -> %q [label=%q%s];\n",
			e.Source, e.Target, e.Label, style))
	}

	b.WriteString("}\n")
	return b.String()
}

// groupByDepLevel groups steps into levels by their dependency depth.
// Steps with no dependencies are level 0, steps depending on level-0 are level 1, etc.
func groupByDepLevel(steps []GraphNode, deps map[string][]string) [][]GraphNode {
	stepSet := map[string]bool{}
	for _, s := range steps {
		stepSet[s.ID] = true
	}

	levels := map[string]int{}
	var computeLevel func(id string) int
	computeLevel = func(id string) int {
		if l, ok := levels[id]; ok {
			return l
		}
		maxDep := -1
		for _, dep := range deps[id] {
			if stepSet[dep] {
				l := computeLevel(dep)
				if l > maxDep {
					maxDep = l
				}
			}
		}
		levels[id] = maxDep + 1
		return maxDep + 1
	}

	for _, s := range steps {
		computeLevel(s.ID)
	}

	maxLevel := 0
	for _, l := range levels {
		if l > maxLevel {
			maxLevel = l
		}
	}

	result := make([][]GraphNode, maxLevel+1)
	for _, s := range steps {
		l := levels[s.ID]
		result[l] = append(result[l], s)
	}
	return result
}

func writeNode(b *strings.Builder, n GraphNode, indent string) {
	shape := dotShapes[n.Type]
	if shape == "" {
		shape = "box"
	}
	color := dotColors[n.Type]
	if color == "" {
		color = "#9CA3AF"
	}

	style := "filled"
	if n.Attributes != nil && n.Attributes["missing"] == "true" {
		style = "dashed"
	}
	if n.Type == "agent" {
		style += ",rounded"
	}

	b.WriteString(fmt.Sprintf("%s%q [label=%q shape=%s style=%q fillcolor=%q color=%q];\n",
		indent, n.ID, n.Name, shape, style, color+"20", color))
}
