package graph

import (
	"strconv"
	"strings"

	"github.com/szaher/agentspec/internal/ast"
)

// Extract walks AST files and builds a Graph with nodes and edges.
func Extract(files []*ast.File) *Graph {
	g := &Graph{
		Files: []string{},
		Stats: GraphStats{TypeCounts: map[string]int{}},
	}

	// Track all nodes by ID for unresolved reference detection.
	nodeSet := map[string]bool{}

	for _, f := range files {
		if f == nil {
			continue
		}
		if f.Path != "" {
			g.Files = append(g.Files, f.Path)
		}

		// Extract package info from first file that has it.
		if f.Package != nil && g.Package.Name == "" {
			g.Package = PackageInfo{
				Name:        f.Package.Name,
				Version:     f.Package.Version,
				Description: f.Package.Description,
			}
		}

		// Extract plugins from the package header (not in Statements).
		if f.Package != nil {
			extractPlugins(g, f.Package, f.Path, nodeSet)
		}

		for _, stmt := range f.Statements {
			extractStatement(g, stmt, f.Path, nodeSet)
		}
	}

	// Create placeholder nodes for unresolved references.
	resolveUnresolved(g, nodeSet)

	ComputeStats(g)
	return g
}

func extractStatement(g *Graph, stmt ast.Statement, file string, nodeSet map[string]bool) {
	switch s := stmt.(type) {
	case *ast.Agent:
		extractAgent(g, s, file, nodeSet)
	case *ast.Prompt:
		extractPrompt(g, s, file, nodeSet)
	case *ast.Skill:
		extractSkill(g, s, file, nodeSet)
	case *ast.MCPServer:
		extractMCPServer(g, s, file, nodeSet)
	case *ast.MCPClient:
		extractMCPClient(g, s, file, nodeSet)
	case *ast.Pipeline:
		extractPipeline(g, s, file, nodeSet)
	case *ast.Secret:
		extractSecret(g, s, file, nodeSet)
	case *ast.Policy:
		extractPolicy(g, s, file, nodeSet)
	case *ast.Guardrail:
		extractGuardrail(g, s, file, nodeSet)
	case *ast.User:
		extractUser(g, s, file, nodeSet)
	case *ast.Binding:
		extractBinding(g, s, file, nodeSet)
	case *ast.DeployTarget:
		extractDeployTarget(g, s, file, nodeSet)
	case *ast.StateConfig:
		extractStateConfig(g, s, file, nodeSet)
	case *ast.Environment:
		extractEnvironment(g, s, file, nodeSet)
	case *ast.TypeDef:
		extractTypeDef(g, s, file, nodeSet)
	case *ast.Package:
		extractPlugins(g, s, file, nodeSet)
	}
}

func addNode(g *Graph, node GraphNode, nodeSet map[string]bool) {
	g.Nodes = append(g.Nodes, node)
	nodeSet[node.ID] = true
}

func addEdge(g *Graph, source, target, label, style string) {
	if style == "" {
		style = "solid"
	}
	g.Edges = append(g.Edges, GraphEdge{
		Source: source,
		Target: target,
		Label:  label,
		Style:  style,
	})
}

func nodeID(typ, name string) string {
	return typ + ":" + name
}

// --- Entity extractors ---

func extractAgent(g *Graph, a *ast.Agent, file string, nodeSet map[string]bool) {
	id := nodeID("agent", a.Name)
	attrs := map[string]string{}

	if a.Model != "" {
		attrs["model"] = a.Model
	}
	if len(a.Models) > 0 {
		attrs["models"] = strings.Join(a.Models, ", ")
	}
	if a.Strategy != "" {
		attrs["strategy"] = a.Strategy
	}
	if a.MaxTurns > 0 {
		attrs["max_turns"] = strconv.Itoa(a.MaxTurns)
	}
	if a.Timeout != "" {
		attrs["timeout"] = a.Timeout
	}
	if a.TokenBudget > 0 {
		attrs["token_budget"] = strconv.Itoa(a.TokenBudget)
	}
	if a.HasTemp {
		attrs["temperature"] = strconv.FormatFloat(a.Temperature, 'f', -1, 64)
	}
	if a.OnError != "" {
		attrs["on_error"] = a.OnError
	}
	if a.BudgetDaily > 0 {
		attrs["budget_daily"] = strconv.FormatFloat(a.BudgetDaily, 'f', 2, 64)
	}
	if a.BudgetMonthly > 0 {
		attrs["budget_monthly"] = strconv.FormatFloat(a.BudgetMonthly, 'f', 2, 64)
	}

	addNode(g, GraphNode{
		ID: id, Type: "agent", Name: a.Name,
		File: file, Line: a.StartPos.Line, Attributes: attrs,
	}, nodeSet)

	// Edges
	if a.Prompt != nil {
		addEdge(g, id, nodeID("prompt", a.Prompt.Name), "uses prompt", "solid")
	}
	for _, sk := range a.Skills {
		addEdge(g, id, nodeID("skill", sk.Name), "uses skill", "solid")
	}
	for _, gr := range a.GuardrailRefs {
		addEdge(g, id, nodeID("guardrail", gr), "uses guardrail", "solid")
	}
	if a.Client != nil {
		addEdge(g, id, nodeID("mcp_client", a.Client.Name), "uses client", "solid")
	}
	for _, d := range a.Delegates {
		addEdge(g, id, nodeID("agent", d.AgentRef), "delegates to", "dashed")
	}
	if a.Fallback != "" {
		addEdge(g, id, nodeID("agent", a.Fallback), "fallback", "dashed")
	}
}

func extractPrompt(g *Graph, p *ast.Prompt, file string, nodeSet map[string]bool) {
	attrs := map[string]string{}
	if p.Content != "" {
		preview := p.Content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		attrs["content_preview"] = preview
	}
	if p.Version != "" {
		attrs["version"] = p.Version
	}
	if len(p.Variables) > 0 {
		names := make([]string, len(p.Variables))
		for i, v := range p.Variables {
			names[i] = v.Name
		}
		attrs["variables"] = strings.Join(names, ", ")
	}

	addNode(g, GraphNode{
		ID: nodeID("prompt", p.Name), Type: "prompt", Name: p.Name,
		File: file, Line: p.StartPos.Line, Attributes: attrs,
	}, nodeSet)
}

func extractSkill(g *Graph, s *ast.Skill, file string, nodeSet map[string]bool) {
	id := nodeID("skill", s.Name)
	attrs := map[string]string{}
	if s.Description != "" {
		attrs["description"] = s.Description
	}
	if s.ToolConfig != nil {
		attrs["tool_type"] = s.ToolConfig.Type
		if s.ToolConfig.ServerTool != "" {
			attrs["server_tool"] = s.ToolConfig.ServerTool
		}
	}
	if len(s.Input) > 0 {
		names := make([]string, len(s.Input))
		for i, f := range s.Input {
			names[i] = f.Name + ":" + f.Type
		}
		attrs["input"] = strings.Join(names, ", ")
	}
	if len(s.Output) > 0 {
		names := make([]string, len(s.Output))
		for i, f := range s.Output {
			names[i] = f.Name + ":" + f.Type
		}
		attrs["output"] = strings.Join(names, ", ")
	}

	addNode(g, GraphNode{
		ID: id, Type: "skill", Name: s.Name,
		File: file, Line: s.StartPos.Line, Attributes: attrs,
	}, nodeSet)

	// Skill → MCP server tool
	if s.ToolConfig != nil && s.ToolConfig.ServerTool != "" {
		parts := strings.SplitN(s.ToolConfig.ServerTool, "/", 2)
		if len(parts) > 0 {
			addEdge(g, id, nodeID("mcp_server", parts[0]), "uses tool", "solid")
		}
	}
}

func extractMCPServer(g *Graph, m *ast.MCPServer, file string, nodeSet map[string]bool) {
	attrs := map[string]string{}
	if m.Transport != "" {
		attrs["transport"] = m.Transport
	}
	if m.Command != "" {
		attrs["command"] = m.Command
	}
	if m.URL != "" {
		attrs["url"] = m.URL
	}

	addNode(g, GraphNode{
		ID: nodeID("mcp_server", m.Name), Type: "mcp_server", Name: m.Name,
		File: file, Line: m.StartPos.Line, Attributes: attrs,
	}, nodeSet)
}

func extractMCPClient(g *Graph, m *ast.MCPClient, file string, nodeSet map[string]bool) {
	id := nodeID("mcp_client", m.Name)
	addNode(g, GraphNode{
		ID: id, Type: "mcp_client", Name: m.Name,
		File: file, Line: m.StartPos.Line,
	}, nodeSet)

	for _, srv := range m.Servers {
		addEdge(g, id, nodeID("mcp_server", srv.Name), "connects to", "solid")
	}
}

func extractPipeline(g *Graph, p *ast.Pipeline, file string, nodeSet map[string]bool) {
	pipeID := nodeID("pipeline", p.Name)
	addNode(g, GraphNode{
		ID: pipeID, Type: "pipeline", Name: p.Name,
		File: file, Line: p.StartPos.Line,
	}, nodeSet)

	for _, step := range p.Steps {
		stepID := nodeID("step", p.Name+"."+step.Name)
		attrs := map[string]string{}
		if step.Agent != "" {
			attrs["agent"] = step.Agent
		}
		if step.Parallel {
			attrs["parallel"] = "true"
		}
		if step.When != "" {
			attrs["when"] = step.When
		}

		addNode(g, GraphNode{
			ID: stepID, Type: "step", Name: step.Name,
			File: file, Line: step.StartPos.Line, Attributes: attrs,
		}, nodeSet)

		addEdge(g, pipeID, stepID, "contains", "solid")

		if step.Agent != "" {
			addEdge(g, stepID, nodeID("agent", step.Agent), "invokes", "solid")
		}
		for _, dep := range step.DependsOn {
			addEdge(g, stepID, nodeID("step", p.Name+"."+dep), "depends on", "solid")
		}
	}
}

func extractSecret(g *Graph, s *ast.Secret, file string, nodeSet map[string]bool) {
	attrs := map[string]string{}
	if s.Source != "" {
		attrs["source"] = s.Source
	}

	addNode(g, GraphNode{
		ID: nodeID("secret", s.Name), Type: "secret", Name: s.Name,
		File: file, Line: s.StartPos.Line, Attributes: attrs,
	}, nodeSet)
}

func extractPolicy(g *Graph, p *ast.Policy, file string, nodeSet map[string]bool) {
	id := nodeID("policy", p.Name)
	attrs := map[string]string{}
	if len(p.Rules) > 0 {
		attrs["rule_count"] = strconv.Itoa(len(p.Rules))
	}

	addNode(g, GraphNode{
		ID: id, Type: "policy", Name: p.Name,
		File: file, Line: p.StartPos.Line, Attributes: attrs,
	}, nodeSet)

	for _, rule := range p.Rules {
		if rule.Resource != "" {
			addEdge(g, id, nodeID("policy_target", rule.Resource), "governs", "solid")
		}
	}
}

func extractGuardrail(g *Graph, gr *ast.Guardrail, file string, nodeSet map[string]bool) {
	attrs := map[string]string{}
	if gr.Mode != "" {
		attrs["mode"] = gr.Mode
	}
	if len(gr.Keywords) > 0 {
		attrs["keywords"] = strings.Join(gr.Keywords, ", ")
	}
	if gr.FallbackMsg != "" {
		attrs["fallback_msg"] = gr.FallbackMsg
	}

	addNode(g, GraphNode{
		ID: nodeID("guardrail", gr.Name), Type: "guardrail", Name: gr.Name,
		File: file, Line: gr.StartPos.Line, Attributes: attrs,
	}, nodeSet)
}

func extractUser(g *Graph, u *ast.User, file string, nodeSet map[string]bool) {
	id := nodeID("user", u.Name)
	attrs := map[string]string{}
	if u.Role != "" {
		attrs["role"] = u.Role
	}

	addNode(g, GraphNode{
		ID: id, Type: "user", Name: u.Name,
		File: file, Line: u.StartPos.Line, Attributes: attrs,
	}, nodeSet)

	for _, agent := range u.Agents {
		addEdge(g, id, nodeID("agent", agent), "can access", "solid")
	}
}

func extractBinding(g *Graph, b *ast.Binding, file string, nodeSet map[string]bool) {
	attrs := map[string]string{}
	if b.Adapter != "" {
		attrs["adapter"] = b.Adapter
	}
	if b.Default {
		attrs["default"] = "true"
	}

	addNode(g, GraphNode{
		ID: nodeID("binding", b.Name), Type: "binding", Name: b.Name,
		File: file, Line: b.StartPos.Line, Attributes: attrs,
	}, nodeSet)
}

func extractDeployTarget(g *Graph, d *ast.DeployTarget, file string, nodeSet map[string]bool) {
	attrs := map[string]string{}
	if d.Target != "" {
		attrs["target"] = d.Target
	}
	if d.Default {
		attrs["default"] = "true"
	}
	if d.Port > 0 {
		attrs["port"] = strconv.Itoa(d.Port)
	}
	if d.Namespace != "" {
		attrs["namespace"] = d.Namespace
	}
	if d.Replicas > 0 {
		attrs["replicas"] = strconv.Itoa(d.Replicas)
	}
	if d.Image != "" {
		attrs["image"] = d.Image
	}

	addNode(g, GraphNode{
		ID: nodeID("deploy", d.Name), Type: "deploy", Name: d.Name,
		File: file, Line: d.StartPos.Line, Attributes: attrs,
	}, nodeSet)
}

func extractStateConfig(g *Graph, s *ast.StateConfig, file string, nodeSet map[string]bool) {
	attrs := map[string]string{}
	if s.Type != "" {
		attrs["backend_type"] = s.Type
	}
	for k, v := range s.Properties {
		attrs[k] = v
	}

	name := s.Name
	if name == "" {
		name = s.Type
	}
	addNode(g, GraphNode{
		ID: nodeID("state", name), Type: "state", Name: name,
		File: file, Line: s.StartPos.Line, Attributes: attrs,
	}, nodeSet)
}

func extractEnvironment(g *Graph, e *ast.Environment, file string, nodeSet map[string]bool) {
	id := nodeID("env", e.Name)
	attrs := map[string]string{}
	if len(e.Overrides) > 0 {
		attrs["override_count"] = strconv.Itoa(len(e.Overrides))
	}

	addNode(g, GraphNode{
		ID: id, Type: "env", Name: e.Name,
		File: file, Line: e.StartPos.Line, Attributes: attrs,
	}, nodeSet)

	for _, ov := range e.Overrides {
		if ov.Resource != "" {
			addEdge(g, id, nodeID("env_target", ov.Resource), "overrides", "dashed")
		}
	}
}

func extractTypeDef(g *Graph, t *ast.TypeDef, file string, nodeSet map[string]bool) {
	attrs := map[string]string{}
	if len(t.Fields) > 0 {
		names := make([]string, len(t.Fields))
		for i, f := range t.Fields {
			names[i] = f.Name + ":" + f.Type
		}
		attrs["fields"] = strings.Join(names, ", ")
	}
	if len(t.EnumVals) > 0 {
		attrs["enum_values"] = strings.Join(t.EnumVals, ", ")
	}
	if t.ListOf != "" {
		attrs["list_of"] = t.ListOf
	}

	addNode(g, GraphNode{
		ID: nodeID("type", t.Name), Type: "type", Name: t.Name,
		File: file, Line: t.StartPos.Line, Attributes: attrs,
	}, nodeSet)
}

func extractPlugins(g *Graph, pkg *ast.Package, file string, nodeSet map[string]bool) {
	for _, p := range pkg.Plugins {
		attrs := map[string]string{}
		if p.Version != "" {
			attrs["version"] = p.Version
		}

		addNode(g, GraphNode{
			ID: nodeID("plugin", p.Name), Type: "plugin", Name: p.Name,
			File: file, Line: p.StartPos.Line, Attributes: attrs,
		}, nodeSet)
	}
}

// resolveUnresolved creates placeholder nodes for edge targets that don't exist.
func resolveUnresolved(g *Graph, nodeSet map[string]bool) {
	for _, edge := range g.Edges {
		if !nodeSet[edge.Target] {
			parts := strings.SplitN(edge.Target, ":", 2)
			typ := "unknown"
			name := edge.Target
			if len(parts) == 2 {
				typ = parts[0]
				name = parts[1]
			}
			g.Nodes = append(g.Nodes, GraphNode{
				ID:         edge.Target,
				Type:       typ,
				Name:       name,
				Attributes: map[string]string{"missing": "true"},
			})
			nodeSet[edge.Target] = true
		}
	}
}

// ComputeStats populates the Stats field of the graph.
func ComputeStats(g *Graph) {
	g.Stats.NodeCount = len(g.Nodes)
	g.Stats.EdgeCount = len(g.Edges)
	g.Stats.FileCount = len(g.Files)
	g.Stats.TypeCounts = map[string]int{}
	for _, n := range g.Nodes {
		g.Stats.TypeCounts[n.Type]++
	}
}

// FilterFiles removes file-type nodes and their edges.
func FilterFiles(g *Graph) {
	filterNodes(g, func(n GraphNode) bool {
		return n.Type != "file"
	})
}

// FilterOrphans removes nodes with zero edges.
func FilterOrphans(g *Graph) {
	connected := map[string]bool{}
	for _, e := range g.Edges {
		connected[e.Source] = true
		connected[e.Target] = true
	}
	filterNodes(g, func(n GraphNode) bool {
		return connected[n.ID]
	})
}

func filterNodes(g *Graph, keep func(GraphNode) bool) {
	removed := map[string]bool{}
	var nodes []GraphNode
	for _, n := range g.Nodes {
		if keep(n) {
			nodes = append(nodes, n)
		} else {
			removed[n.ID] = true
		}
	}
	g.Nodes = nodes

	var edges []GraphEdge
	for _, e := range g.Edges {
		if !removed[e.Source] && !removed[e.Target] {
			edges = append(edges, e)
		}
	}
	g.Edges = edges

	ComputeStats(g)
}

// AddFileNodes creates synthetic file nodes and defines/imports edges.
// Called for multi-file projects (US3).
func AddFileNodes(g *Graph, files []*ast.File) {
	nodeSet := map[string]bool{}
	for _, n := range g.Nodes {
		nodeSet[n.ID] = true
	}

	fileSet := map[string]bool{}
	for _, f := range files {
		if f == nil || f.Path == "" {
			continue
		}
		fid := nodeID("file", f.Path)
		if fileSet[fid] {
			continue
		}
		fileSet[fid] = true

		addNode(g, GraphNode{
			ID: fid, Type: "file", Name: f.Path,
			File: f.Path, Line: 1,
		}, nodeSet)

		// defines edges
		for _, stmt := range f.Statements {
			targetID := stmtNodeID(stmt)
			if targetID != "" && nodeSet[targetID] {
				addEdge(g, fid, targetID, "defines", "solid")
			}
		}

		// imports edges
		if f.Package != nil {
			for _, imp := range f.Package.Imports {
				importedFID := nodeID("file", imp.Path)
				if !nodeSet[importedFID] {
					addNode(g, GraphNode{
						ID: importedFID, Type: "file", Name: imp.Path,
						File: imp.Path, Line: 1,
					}, nodeSet)
				}
				addEdge(g, fid, importedFID, "imports", "solid")
			}
		}
	}

	ComputeStats(g)
}

func stmtNodeID(stmt ast.Statement) string {
	switch s := stmt.(type) {
	case *ast.Agent:
		return nodeID("agent", s.Name)
	case *ast.Prompt:
		return nodeID("prompt", s.Name)
	case *ast.Skill:
		return nodeID("skill", s.Name)
	case *ast.MCPServer:
		return nodeID("mcp_server", s.Name)
	case *ast.MCPClient:
		return nodeID("mcp_client", s.Name)
	case *ast.Pipeline:
		return nodeID("pipeline", s.Name)
	case *ast.Secret:
		return nodeID("secret", s.Name)
	case *ast.Policy:
		return nodeID("policy", s.Name)
	case *ast.Guardrail:
		return nodeID("guardrail", s.Name)
	case *ast.User:
		return nodeID("user", s.Name)
	case *ast.Binding:
		return nodeID("binding", s.Name)
	case *ast.DeployTarget:
		return nodeID("deploy", s.Name)
	case *ast.StateConfig:
		name := s.Name
		if name == "" {
			name = s.Type
		}
		return nodeID("state", name)
	case *ast.Environment:
		return nodeID("env", s.Name)
	case *ast.TypeDef:
		return nodeID("type", s.Name)
	default:
		return ""
	}
}
