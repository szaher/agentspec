package targets

import (
	"fmt"
	"strings"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/plugins"
)

func init() {
	Register(&LangGraphTarget{})
}

// LangGraphTarget compiles AgentSpec IR to LangGraph Python projects.
type LangGraphTarget struct{}

func (t *LangGraphTarget) Name() string { return "langgraph" }

func (t *LangGraphTarget) FeatureSupport() plugins.FeatureMap {
	return plugins.FeatureMap{
		"agent":                 plugins.FeatureFull,
		"skill":                 plugins.FeatureFull,
		"prompt":                plugins.FeatureFull,
		"pipeline_sequential":   plugins.FeatureFull,
		"pipeline_hierarchical": plugins.FeatureFull,
		"pipeline_conditional":  plugins.FeatureFull,
		"loop_react":            plugins.FeatureFull,
		"loop_plan_execute":     plugins.FeatureFull,
		"loop_reflexion":        plugins.FeatureFull,
		"loop_router":           plugins.FeatureFull,
		"loop_map_reduce":       plugins.FeaturePartial,
		"validation_rules":      plugins.FeatureEmulated,
		"eval_cases":            plugins.FeatureEmulated,
		"config_params":         plugins.FeatureFull,
		"control_flow_if":       plugins.FeatureFull,
		"control_flow_foreach":  plugins.FeatureEmulated,
		"streaming":             plugins.FeatureFull,
		"sessions":              plugins.FeatureFull,
		"type_definitions":      plugins.FeaturePartial,
		"delegation":            plugins.FeatureFull,
		"inline_tools":          plugins.FeatureNone,
		"mcp_tools":             plugins.FeatureNone,
		"error_handling":        plugins.FeatureFull,
	}
}

func (t *LangGraphTarget) Compile(doc *ir.Document, name string) (*Result, error) {
	agents := extractAgents(doc)
	skills := extractSkills(doc)

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents found in IR document")
	}

	var files []plugins.GeneratedFile

	files = append(files, t.generateRequirements())
	files = append(files, t.generateTools(skills))
	files = append(files, t.generateGraph(doc, agents, skills))
	files = append(files, t.generateMain(name))

	return &Result{
		Files: files,
		Metadata: plugins.CompileMetadata{
			Framework:        "langgraph",
			FrameworkVersion: ">=0.2.0",
			PythonVersion:    ">=3.10",
			RunCommand:       "python main.py",
		},
	}, nil
}

func (t *LangGraphTarget) generateRequirements() plugins.GeneratedFile {
	content := `langgraph>=0.2.0
langchain>=0.3.0
langchain-openai>=0.2.0
langchain-anthropic>=0.3.0
`
	return plugins.GeneratedFile{Path: "requirements.txt", Content: content, Mode: "0644"}
}

func (t *LangGraphTarget) generateTools(skills []ir.Resource) plugins.GeneratedFile {
	var sb strings.Builder

	sb.WriteString("\"\"\"Tool definitions for the LangGraph agent.\"\"\"\n\n")
	sb.WriteString("import subprocess\n")
	sb.WriteString("import urllib.request\n")
	sb.WriteString("from langchain_core.tools import tool\n\n")

	for _, skill := range skills {
		safeName := pythonSafe(skill.Name)
		desc := getStringAttr(skill, "description")
		if desc == "" {
			desc = fmt.Sprintf("%s tool", skill.Name)
		}

		sb.WriteString("@tool\n")
		sb.WriteString(fmt.Sprintf("def %s(", safeName))

		if inputs, ok := skill.Attributes["input"].([]interface{}); ok {
			var params []string
			for _, inp := range inputs {
				if m, ok := inp.(map[string]interface{}); ok {
					paramName := pythonSafe(fmt.Sprintf("%v", m["name"]))
					paramType := "str"
					if tp, ok := m["type"].(string); ok {
						switch tp {
						case "int":
							paramType = "int"
						case "float":
							paramType = "float"
						case "bool":
							paramType = "bool"
						}
					}
					params = append(params, fmt.Sprintf("%s: %s", paramName, paramType))
				}
			}
			sb.WriteString(strings.Join(params, ", "))
		}

		sb.WriteString(") -> str:\n")
		sb.WriteString(fmt.Sprintf("    \"\"\"%s\"\"\"\n", desc))

		if tool, ok := skill.Attributes["tool"].(map[string]interface{}); ok {
			toolType, _ := tool["type"].(string)
			switch toolType {
			case "command":
				binary, _ := tool["binary"].(string)
				args, _ := tool["args"].(string)
				if binary != "" {
					sb.WriteString(fmt.Sprintf("    result = subprocess.run([%q", binary))
					if args != "" {
						for _, arg := range strings.Fields(args) {
							sb.WriteString(fmt.Sprintf(", %q", arg))
						}
					}
					sb.WriteString("], capture_output=True, text=True)\n")
					sb.WriteString("    return result.stdout\n")
				} else {
					sb.WriteString("    return \"not implemented\"\n")
				}
			case "http":
				url, _ := tool["url"].(string)
				method, _ := tool["method"].(string)
				if url != "" {
					if method == "" {
						method = "GET"
					}
					sb.WriteString(fmt.Sprintf("    req = urllib.request.Request(%q, method=%q)\n", url, method))
					sb.WriteString("    with urllib.request.urlopen(req) as resp:\n")
					sb.WriteString("        return resp.read().decode()\n")
				} else {
					sb.WriteString("    return \"not implemented\"\n")
				}
			default:
				sb.WriteString("    return \"not implemented\"\n")
			}
		} else {
			sb.WriteString("    return \"not implemented\"\n")
		}

		sb.WriteString("\n\n")
	}

	// Export list
	if len(skills) > 0 {
		sb.WriteString("all_tools = [")
		var toolNames []string
		for _, s := range skills {
			toolNames = append(toolNames, pythonSafe(s.Name))
		}
		sb.WriteString(strings.Join(toolNames, ", "))
		sb.WriteString("]\n")
	}

	return plugins.GeneratedFile{Path: "tools.py", Content: sb.String(), Mode: "0644"}
}

func (t *LangGraphTarget) generateGraph(doc *ir.Document, agents []ir.Resource, skills []ir.Resource) plugins.GeneratedFile {
	var sb strings.Builder

	sb.WriteString("\"\"\"LangGraph agent graph definition.\"\"\"\n\n")
	sb.WriteString("import operator\n")
	sb.WriteString("from typing import Annotated, TypedDict\n\n")
	sb.WriteString("from langchain_core.messages import AnyMessage, HumanMessage, SystemMessage\n")
	sb.WriteString("from langchain_openai import ChatOpenAI\n")
	sb.WriteString("from langgraph.graph import StateGraph, END\n")
	sb.WriteString("from langgraph.prebuilt import ToolNode\n\n")

	if len(skills) > 0 {
		sb.WriteString("from tools import all_tools\n\n")
	}

	// State definition
	sb.WriteString("\nclass AgentState(TypedDict):\n")
	sb.WriteString("    messages: Annotated[list[AnyMessage], operator.add]\n\n\n")

	// Node functions for each agent
	for _, agent := range agents {
		safeName := pythonSafe(agent.Name)
		model := getStringAttr(agent, "model")
		if model == "" {
			model = "gpt-4"
		}

		sb.WriteString(fmt.Sprintf("def %s_node(state: AgentState) -> dict:\n", safeName))
		sb.WriteString(fmt.Sprintf("    \"\"\"Node for the %s agent.\"\"\"\n", agent.Name))
		sb.WriteString(fmt.Sprintf("    llm = ChatOpenAI(model=%q)\n", model))

		if len(skills) > 0 {
			sb.WriteString("    llm_with_tools = llm.bind_tools(all_tools)\n")
			sb.WriteString("    response = llm_with_tools.invoke(state[\"messages\"])\n")
		} else {
			sb.WriteString("    response = llm.invoke(state[\"messages\"])\n")
		}

		sb.WriteString("    return {\"messages\": [response]}\n\n\n")
	}

	// Should continue function for tool calling
	if len(skills) > 0 {
		sb.WriteString("def should_continue(state: AgentState) -> str:\n")
		sb.WriteString("    \"\"\"Determine if the agent should continue with tools or end.\"\"\"\n")
		sb.WriteString("    last_message = state[\"messages\"][-1]\n")
		sb.WriteString("    if hasattr(last_message, \"tool_calls\") and last_message.tool_calls:\n")
		sb.WriteString("        return \"tools\"\n")
		sb.WriteString("    return \"end\"\n\n\n")
	}

	// Build graph function
	sb.WriteString("def build_graph():\n")
	sb.WriteString("    \"\"\"Build and compile the agent graph.\"\"\"\n")
	sb.WriteString("    graph = StateGraph(AgentState)\n\n")

	// Add system prompts
	for _, agent := range agents {
		promptName := getStringAttr(agent, "prompt")
		prompt := getPromptContent(doc, promptName)
		if prompt != "" {
			safeName := pythonSafe(agent.Name)
			sb.WriteString(fmt.Sprintf("    # System prompt for %s\n", agent.Name))
			sb.WriteString(fmt.Sprintf("    %s_system = %q\n\n", safeName, prompt))
		}
	}

	// Add nodes
	for _, agent := range agents {
		safeName := pythonSafe(agent.Name)
		sb.WriteString(fmt.Sprintf("    graph.add_node(%q, %s_node)\n", safeName, safeName))
	}

	if len(skills) > 0 {
		sb.WriteString("    graph.add_node(\"tools\", ToolNode(all_tools))\n")
	}

	sb.WriteString("\n")

	// Add edges
	if len(agents) == 1 {
		safeName := pythonSafe(agents[0].Name)
		sb.WriteString(fmt.Sprintf("    graph.set_entry_point(%q)\n", safeName))
		if len(skills) > 0 {
			sb.WriteString(fmt.Sprintf("    graph.add_conditional_edges(%q, should_continue, {\"tools\": \"tools\", \"end\": END})\n", safeName))
			sb.WriteString(fmt.Sprintf("    graph.add_edge(\"tools\", %q)\n", safeName))
		} else {
			sb.WriteString(fmt.Sprintf("    graph.add_edge(%q, END)\n", safeName))
		}
	} else {
		// Multi-agent: sequential chain
		sb.WriteString(fmt.Sprintf("    graph.set_entry_point(%q)\n", pythonSafe(agents[0].Name)))
		for i := 0; i < len(agents)-1; i++ {
			current := pythonSafe(agents[i].Name)
			next := pythonSafe(agents[i+1].Name)
			if len(skills) > 0 {
				sb.WriteString(fmt.Sprintf("    graph.add_conditional_edges(%q, should_continue, {\"tools\": \"tools\", \"end\": %q})\n", current, next))
			} else {
				sb.WriteString(fmt.Sprintf("    graph.add_edge(%q, %q)\n", current, next))
			}
		}
		lastAgent := pythonSafe(agents[len(agents)-1].Name)
		if len(skills) > 0 {
			sb.WriteString(fmt.Sprintf("    graph.add_conditional_edges(%q, should_continue, {\"tools\": \"tools\", \"end\": END})\n", lastAgent))
			sb.WriteString(fmt.Sprintf("    graph.add_edge(\"tools\", %q)\n", lastAgent))
		} else {
			sb.WriteString(fmt.Sprintf("    graph.add_edge(%q, END)\n", lastAgent))
		}
	}

	sb.WriteString("\n    return graph.compile()\n")

	return plugins.GeneratedFile{Path: "graph.py", Content: sb.String(), Mode: "0644"}
}

func (t *LangGraphTarget) generateMain(name string) plugins.GeneratedFile {
	content := fmt.Sprintf(`#!/usr/bin/env python3
"""Entry point for the %s LangGraph project."""

import sys
from langchain_core.messages import HumanMessage
from graph import build_graph


def main():
    user_input = " ".join(sys.argv[1:]) if len(sys.argv) > 1 else "Hello"
    graph = build_graph()
    result = graph.invoke({"messages": [HumanMessage(content=user_input)]})
    print(result["messages"][-1].content)


if __name__ == "__main__":
    main()
`, name)

	return plugins.GeneratedFile{Path: "main.py", Content: content, Mode: "0755"}
}
