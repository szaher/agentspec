package targets

import (
	"fmt"
	"strings"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/plugins"
)

func init() {
	Register(&LlamaIndexTarget{})
}

// LlamaIndexTarget compiles AgentSpec IR to LlamaIndex Python projects.
type LlamaIndexTarget struct{}

func (t *LlamaIndexTarget) Name() string { return "llamaindex" }

func (t *LlamaIndexTarget) FeatureSupport() plugins.FeatureMap {
	return plugins.FeatureMap{
		"agent":                 plugins.FeatureFull,
		"skill":                 plugins.FeatureFull,
		"prompt":                plugins.FeatureFull,
		"pipeline_sequential":   plugins.FeatureFull,
		"pipeline_hierarchical": plugins.FeaturePartial,
		"pipeline_conditional":  plugins.FeaturePartial,
		"loop_react":            plugins.FeatureFull,
		"loop_plan_execute":     plugins.FeaturePartial,
		"loop_reflexion":        plugins.FeatureNone,
		"loop_router":           plugins.FeaturePartial,
		"loop_map_reduce":       plugins.FeatureNone,
		"validation_rules":      plugins.FeatureEmulated,
		"eval_cases":            plugins.FeatureEmulated,
		"config_params":         plugins.FeatureFull,
		"control_flow_if":       plugins.FeatureEmulated,
		"control_flow_foreach":  plugins.FeatureEmulated,
		"streaming":             plugins.FeatureFull,
		"sessions":              plugins.FeaturePartial,
		"type_definitions":      plugins.FeaturePartial,
		"delegation":            plugins.FeaturePartial,
		"inline_tools":          plugins.FeatureNone,
		"mcp_tools":             plugins.FeatureNone,
		"error_handling":        plugins.FeaturePartial,
	}
}

func (t *LlamaIndexTarget) Compile(doc *ir.Document, name string) (*Result, error) {
	agents := extractAgents(doc)
	skills := extractSkills(doc)

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents found in IR document")
	}

	var files []plugins.GeneratedFile

	files = append(files, t.generateRequirements())
	files = append(files, t.generateTools(skills))
	files = append(files, t.generateAgent(doc, agents, skills))
	files = append(files, t.generateMain(name))

	return &Result{
		Files: files,
		Metadata: plugins.CompileMetadata{
			Framework:        "llamaindex",
			FrameworkVersion: ">=0.11.0",
			PythonVersion:    ">=3.10",
			RunCommand:       "python main.py",
		},
	}, nil
}

func (t *LlamaIndexTarget) generateRequirements() plugins.GeneratedFile {
	content := `llama-index>=0.11.0
llama-index-llms-openai>=0.3.0
llama-index-llms-anthropic>=0.4.0
llama-index-agent-openai>=0.4.0
`
	return plugins.GeneratedFile{Path: "requirements.txt", Content: content, Mode: "0644"}
}

func (t *LlamaIndexTarget) generateTools(skills []ir.Resource) plugins.GeneratedFile {
	var sb strings.Builder

	sb.WriteString("\"\"\"Tool definitions for the LlamaIndex agent.\"\"\"\n\n")
	sb.WriteString("import subprocess\n")
	sb.WriteString("import urllib.request\n")
	sb.WriteString("from llama_index.core.tools import FunctionTool\n\n")

	// Generate raw functions
	for _, skill := range skills {
		safeName := pythonSafe(skill.Name)
		desc := getStringAttr(skill, "description")
		if desc == "" {
			desc = fmt.Sprintf("%s tool", skill.Name)
		}

		sb.WriteString(fmt.Sprintf("def _%s_fn(", safeName))

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

	// Create FunctionTool wrappers
	for _, skill := range skills {
		safeName := pythonSafe(skill.Name)
		desc := getStringAttr(skill, "description")
		if desc == "" {
			desc = fmt.Sprintf("%s tool", skill.Name)
		}
		sb.WriteString(fmt.Sprintf("%s = FunctionTool.from_defaults(\n", safeName))
		sb.WriteString(fmt.Sprintf("    fn=_%s_fn,\n", safeName))
		sb.WriteString(fmt.Sprintf("    name=%q,\n", safeName))
		sb.WriteString(fmt.Sprintf("    description=%q,\n", desc))
		sb.WriteString(")\n\n")
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

func (t *LlamaIndexTarget) generateAgent(doc *ir.Document, agents []ir.Resource, skills []ir.Resource) plugins.GeneratedFile {
	var sb strings.Builder

	sb.WriteString("\"\"\"Agent definitions for the LlamaIndex project.\"\"\"\n\n")
	sb.WriteString("from llama_index.core.agent import ReActAgent\n")
	sb.WriteString("from llama_index.llms.openai import OpenAI\n\n")

	if len(skills) > 0 {
		sb.WriteString("from tools import all_tools\n\n")
	}

	for _, agent := range agents {
		safeName := pythonSafe(agent.Name)
		model := getStringAttr(agent, "model")
		if model == "" {
			model = "gpt-4"
		}

		promptName := getStringAttr(agent, "prompt")
		prompt := getPromptContent(doc, promptName)
		if prompt == "" {
			prompt = fmt.Sprintf("You are a %s agent.", agent.Name)
		}

		sb.WriteString(fmt.Sprintf("def create_%s():\n", safeName))
		sb.WriteString(fmt.Sprintf("    \"\"\"Create the %s agent.\"\"\"\n", agent.Name))
		sb.WriteString(fmt.Sprintf("    llm = OpenAI(model=%q)\n", model))

		maxTurns := getIntAttr(agent, "max_turns")

		if len(skills) > 0 {
			sb.WriteString("    agent = ReActAgent.from_tools(\n")
			sb.WriteString("        tools=all_tools,\n")
			sb.WriteString("        llm=llm,\n")
			sb.WriteString(fmt.Sprintf("        system_prompt=%q,\n", prompt))
			sb.WriteString("        verbose=True,\n")
			if maxTurns > 0 {
				sb.WriteString(fmt.Sprintf("        max_iterations=%d,\n", maxTurns))
			}
			sb.WriteString("    )\n")
		} else {
			sb.WriteString("    agent = ReActAgent.from_tools(\n")
			sb.WriteString("        tools=[],\n")
			sb.WriteString("        llm=llm,\n")
			sb.WriteString(fmt.Sprintf("        system_prompt=%q,\n", prompt))
			sb.WriteString("        verbose=True,\n")
			if maxTurns > 0 {
				sb.WriteString(fmt.Sprintf("        max_iterations=%d,\n", maxTurns))
			}
			sb.WriteString("    )\n")
		}

		sb.WriteString("    return agent\n\n\n")
	}

	return plugins.GeneratedFile{Path: "agent.py", Content: sb.String(), Mode: "0644"}
}

func (t *LlamaIndexTarget) generateMain(name string) plugins.GeneratedFile {
	content := fmt.Sprintf(`#!/usr/bin/env python3
"""Entry point for the %s LlamaIndex project."""

import sys
from agent import create_%s


def main():
    agent = create_%s()
    user_input = " ".join(sys.argv[1:]) if len(sys.argv) > 1 else "Hello"
    response = agent.chat(user_input)
    print(response)


if __name__ == "__main__":
    main()
`, name, pythonSafe(name), pythonSafe(name))

	return plugins.GeneratedFile{Path: "main.py", Content: content, Mode: "0755"}
}
