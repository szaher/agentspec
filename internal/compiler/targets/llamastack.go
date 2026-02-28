package targets

import (
	"fmt"
	"strings"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/plugins"
)

func init() {
	Register(&LlamaStackTarget{})
}

// LlamaStackTarget compiles AgentSpec IR to LlamaStack Python projects.
type LlamaStackTarget struct{}

func (t *LlamaStackTarget) Name() string { return "llamastack" }

func (t *LlamaStackTarget) FeatureSupport() plugins.FeatureMap {
	return plugins.FeatureMap{
		"agent":                 plugins.FeatureFull,
		"skill":                 plugins.FeatureFull,
		"prompt":                plugins.FeatureFull,
		"pipeline_sequential":   plugins.FeatureNone,
		"pipeline_hierarchical": plugins.FeatureNone,
		"pipeline_conditional":  plugins.FeatureNone,
		"loop_react":            plugins.FeatureFull,
		"loop_plan_execute":     plugins.FeatureNone,
		"loop_reflexion":        plugins.FeatureNone,
		"loop_router":           plugins.FeatureNone,
		"loop_map_reduce":       plugins.FeatureNone,
		"validation_rules":      plugins.FeatureEmulated,
		"eval_cases":            plugins.FeatureEmulated,
		"config_params":         plugins.FeatureFull,
		"control_flow_if":       plugins.FeatureNone,
		"control_flow_foreach":  plugins.FeatureNone,
		"streaming":             plugins.FeatureFull,
		"sessions":              plugins.FeaturePartial,
		"type_definitions":      plugins.FeatureNone,
		"delegation":            plugins.FeatureNone,
		"inline_tools":          plugins.FeatureNone,
		"mcp_tools":             plugins.FeatureNone,
		"error_handling":        plugins.FeaturePartial,
	}
}

func (t *LlamaStackTarget) Compile(doc *ir.Document, name string) (*Result, error) {
	agents := extractAgents(doc)
	skills := extractSkills(doc)

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents found in IR document")
	}

	var files []plugins.GeneratedFile

	files = append(files, t.generateRequirements())
	files = append(files, t.generateAgent(doc, agents, skills))

	return &Result{
		Files: files,
		Metadata: plugins.CompileMetadata{
			Framework:        "llamastack",
			FrameworkVersion: ">=0.1.0",
			PythonVersion:    ">=3.10",
			RunCommand:       "python agent.py",
		},
	}, nil
}

func (t *LlamaStackTarget) generateRequirements() plugins.GeneratedFile {
	content := `llama-stack-client>=0.1.0
`
	return plugins.GeneratedFile{Path: "requirements.txt", Content: content, Mode: "0644"}
}

func (t *LlamaStackTarget) generateAgent(doc *ir.Document, agents []ir.Resource, skills []ir.Resource) plugins.GeneratedFile {
	var sb strings.Builder

	sb.WriteString("#!/usr/bin/env python3\n")
	sb.WriteString("\"\"\"LlamaStack agent generated from AgentSpec.\"\"\"\n\n")
	sb.WriteString("import asyncio\n")
	sb.WriteString("import subprocess\n")
	sb.WriteString("import sys\n\n")
	sb.WriteString("from llama_stack_client import LlamaStackClient\n")
	sb.WriteString("from llama_stack_client.types.agent_create_params import AgentConfig\n\n")

	// Generate tool functions
	for _, skill := range skills {
		safeName := pythonSafe(skill.Name)
		desc := getStringAttr(skill, "description")
		if desc == "" {
			desc = fmt.Sprintf("%s tool", skill.Name)
		}

		fmt.Fprintf(&sb, "def %s(", safeName)

		if inputs, ok := skill.Attributes["input"].([]interface{}); ok {
			var params []string
			for _, inp := range inputs {
				if m, ok := inp.(map[string]interface{}); ok {
					paramName := pythonSafe(fmt.Sprintf("%v", m["name"]))
					params = append(params, fmt.Sprintf("%s: str", paramName))
				}
			}
			sb.WriteString(strings.Join(params, ", "))
		}

		sb.WriteString(") -> str:\n")
		fmt.Fprintf(&sb, "    \"\"\"%s\"\"\"\n", desc)

		if tool, ok := skill.Attributes["tool"].(map[string]interface{}); ok {
			toolType, _ := tool["type"].(string)
			if toolType == "command" {
				binary, _ := tool["binary"].(string)
				args, _ := tool["args"].(string)
				if binary != "" {
					fmt.Fprintf(&sb, "    result = subprocess.run([%q", binary)
					if args != "" {
						for _, arg := range strings.Fields(args) {
							fmt.Fprintf(&sb, ", %q", arg)
						}
					}
					sb.WriteString("], capture_output=True, text=True)\n")
					sb.WriteString("    return result.stdout\n")
				} else {
					sb.WriteString("    return \"not implemented\"\n")
				}
			} else {
				sb.WriteString("    return \"not implemented\"\n")
			}
		} else {
			sb.WriteString("    return \"not implemented\"\n")
		}
		sb.WriteString("\n\n")
	}

	// Generate main async function
	sb.WriteString("async def main():\n")
	sb.WriteString("    client = LlamaStackClient(base_url=\"http://localhost:5000\")\n\n")

	// Use first agent as primary
	agent := agents[0]
	model := getStringAttr(agent, "model")
	if model == "" {
		model = "meta-llama/Llama-3.1-8B-Instruct"
	}

	promptName := getStringAttr(agent, "prompt")
	prompt := getPromptContent(doc, promptName)
	if prompt == "" {
		prompt = fmt.Sprintf("You are a %s agent.", agent.Name)
	}

	sb.WriteString("    agent_config = AgentConfig(\n")
	fmt.Fprintf(&sb, "        model=%q,\n", model)
	fmt.Fprintf(&sb, "        instructions=%q,\n", prompt)
	sb.WriteString("        enable_session_persistence=False,\n")

	maxTurns := getIntAttr(agent, "max_turns")
	if maxTurns > 0 {
		fmt.Fprintf(&sb, "        max_infer_iters=%d,\n", maxTurns)
	}

	sb.WriteString("    )\n\n")

	sb.WriteString("    agent_response = client.agents.create(agent_config=agent_config)\n")
	sb.WriteString("    agent_id = agent_response.agent_id\n\n")
	sb.WriteString("    session = client.agents.session.create(agent_id=agent_id, session_name=\"main\")\n")
	sb.WriteString("    session_id = session.session_id\n\n")

	sb.WriteString("    user_input = \" \".join(sys.argv[1:]) if len(sys.argv) > 1 else \"Hello\"\n\n")

	sb.WriteString("    response = client.agents.turn.create(\n")
	sb.WriteString("        agent_id=agent_id,\n")
	sb.WriteString("        session_id=session_id,\n")
	sb.WriteString("        messages=[{\"role\": \"user\", \"content\": user_input}],\n")
	sb.WriteString("        stream=False,\n")
	sb.WriteString("    )\n\n")
	sb.WriteString("    print(response.output_message.content)\n\n\n")

	sb.WriteString("if __name__ == \"__main__\":\n")
	sb.WriteString("    asyncio.run(main())\n")

	return plugins.GeneratedFile{Path: "agent.py", Content: sb.String(), Mode: "0755"}
}
