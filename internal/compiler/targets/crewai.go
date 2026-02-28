package targets

import (
	"fmt"
	"strings"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/plugins"
)

func init() {
	Register(&CrewAITarget{})
}

// CrewAITarget compiles AgentSpec IR to CrewAI Python projects.
type CrewAITarget struct{}

func (t *CrewAITarget) Name() string { return "crewai" }

func (t *CrewAITarget) FeatureSupport() plugins.FeatureMap {
	return plugins.FeatureMap{
		"agent":                 plugins.FeatureFull,
		"skill":                 plugins.FeatureFull,
		"prompt":                plugins.FeaturePartial,
		"pipeline_sequential":   plugins.FeatureFull,
		"pipeline_hierarchical": plugins.FeatureFull,
		"pipeline_conditional":  plugins.FeatureNone,
		"loop_react":            plugins.FeatureFull,
		"loop_plan_execute":     plugins.FeaturePartial,
		"loop_reflexion":        plugins.FeatureNone,
		"loop_router":           plugins.FeatureNone,
		"loop_map_reduce":       plugins.FeatureNone,
		"validation_rules":      plugins.FeatureEmulated,
		"eval_cases":            plugins.FeatureEmulated,
		"config_params":         plugins.FeatureFull,
		"control_flow_if":       plugins.FeatureEmulated,
		"control_flow_foreach":  plugins.FeatureEmulated,
		"streaming":             plugins.FeatureFull,
		"sessions":              plugins.FeaturePartial,
		"type_definitions":      plugins.FeatureNone,
		"delegation":            plugins.FeatureFull,
		"inline_tools":          plugins.FeatureNone,
		"mcp_tools":             plugins.FeatureNone,
		"error_handling":        plugins.FeaturePartial,
	}
}

func (t *CrewAITarget) Compile(doc *ir.Document, name string) (*Result, error) {
	agents := extractAgents(doc)
	skills := extractSkills(doc)

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents found in IR document")
	}

	var files []plugins.GeneratedFile

	// pyproject.toml
	files = append(files, t.generatePyproject(name))

	// config/agents.yaml
	files = append(files, t.generateAgentsYAML(doc, agents))

	// config/tasks.yaml
	files = append(files, t.generateTasksYAML(agents))

	// tools/__init__.py
	files = append(files, t.generateTools(skills))

	// crew.py
	files = append(files, t.generateCrew(doc, agents, skills))

	// main.py
	files = append(files, t.generateMain(name))

	return &Result{
		Files: files,
		Metadata: plugins.CompileMetadata{
			Framework:        "crewai",
			FrameworkVersion: ">=0.86.0",
			PythonVersion:    ">=3.10",
			RunCommand:       "python main.py",
		},
	}, nil
}

func (t *CrewAITarget) generatePyproject(name string) plugins.GeneratedFile {
	content := fmt.Sprintf(`[project]
name = "%s"
version = "0.1.0"
description = "AgentSpec-compiled CrewAI project"
requires-python = ">=3.10"
dependencies = [
    "crewai>=0.86.0",
    "crewai-tools>=0.14.0",
]

[build-system]
requires = ["setuptools>=75.0"]
build-backend = "setuptools.backends._legacy:_Backend"
`, name)

	return plugins.GeneratedFile{Path: "pyproject.toml", Content: content, Mode: "0644"}
}

func (t *CrewAITarget) generateAgentsYAML(doc *ir.Document, agents []ir.Resource) plugins.GeneratedFile {
	var sb strings.Builder

	for _, agent := range agents {
		safeName := pythonSafe(agent.Name)
		sb.WriteString(fmt.Sprintf("%s:\n", safeName))

		// Role from prompt or agent name
		promptName := getStringAttr(agent, "prompt")
		prompt := getPromptContent(doc, promptName)
		if prompt != "" {
			// Use first sentence as role
			role := prompt
			if idx := strings.Index(prompt, "."); idx > 0 && idx < 100 {
				role = prompt[:idx]
			}
			sb.WriteString(fmt.Sprintf("  role: >-\n    %s\n", role))
			sb.WriteString(fmt.Sprintf("  backstory: >-\n    %s\n", prompt))
		} else {
			sb.WriteString(fmt.Sprintf("  role: >-\n    %s agent\n", agent.Name))
			sb.WriteString(fmt.Sprintf("  backstory: >-\n    You are a %s agent.\n", agent.Name))
		}

		sb.WriteString(fmt.Sprintf("  goal: >-\n    Accomplish tasks as the %s agent\n", agent.Name))

		model := getStringAttr(agent, "model")
		if model != "" {
			sb.WriteString(fmt.Sprintf("  llm: %s\n", model))
		}

		sb.WriteString("\n")
	}

	return plugins.GeneratedFile{Path: "config/agents.yaml", Content: sb.String(), Mode: "0644"}
}

func (t *CrewAITarget) generateTasksYAML(agents []ir.Resource) plugins.GeneratedFile {
	var sb strings.Builder

	for _, agent := range agents {
		safeName := pythonSafe(agent.Name)
		sb.WriteString(fmt.Sprintf("%s_task:\n", safeName))
		sb.WriteString(fmt.Sprintf("  description: >-\n    Execute the %s agent task with the given input: {input}\n", agent.Name))
		sb.WriteString(fmt.Sprintf("  expected_output: >-\n    The result of the %s agent processing\n", agent.Name))
		sb.WriteString(fmt.Sprintf("  agent: %s\n", safeName))
		sb.WriteString("\n")
	}

	return plugins.GeneratedFile{Path: "config/tasks.yaml", Content: sb.String(), Mode: "0644"}
}

func (t *CrewAITarget) generateTools(skills []ir.Resource) plugins.GeneratedFile {
	var sb strings.Builder

	sb.WriteString("from crewai.tools import tool\n")
	sb.WriteString("import subprocess\n")
	sb.WriteString("import urllib.request\n")
	sb.WriteString("import json\n\n")

	for _, skill := range skills {
		safeName := pythonSafe(skill.Name)
		desc := getStringAttr(skill, "description")
		if desc == "" {
			desc = fmt.Sprintf("%s tool", skill.Name)
		}

		sb.WriteString(fmt.Sprintf("@tool\n"))
		sb.WriteString(fmt.Sprintf("def %s(", safeName))

		// Generate input parameters
		if inputs, ok := skill.Attributes["input"].([]interface{}); ok {
			var params []string
			for _, inp := range inputs {
				if m, ok := inp.(map[string]interface{}); ok {
					paramName := pythonSafe(fmt.Sprintf("%v", m["name"]))
					paramType := "str"
					if t, ok := m["type"].(string); ok {
						switch t {
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

		sb.WriteString(fmt.Sprintf(") -> str:\n"))
		sb.WriteString(fmt.Sprintf("    \"\"\"%s\"\"\"\n", desc))

		// Generate tool body based on tool type
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
					sb.WriteString("    # TODO: implement tool logic\n")
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
					sb.WriteString("    # TODO: implement HTTP tool\n")
					sb.WriteString("    return \"not implemented\"\n")
				}
			default:
				sb.WriteString("    # TODO: implement tool logic\n")
				sb.WriteString("    return \"not implemented\"\n")
			}
		} else {
			sb.WriteString("    # TODO: implement tool logic\n")
			sb.WriteString("    return \"not implemented\"\n")
		}

		sb.WriteString("\n\n")
	}

	return plugins.GeneratedFile{Path: "tools/__init__.py", Content: sb.String(), Mode: "0644"}
}

func (t *CrewAITarget) generateCrew(doc *ir.Document, agents []ir.Resource, skills []ir.Resource) plugins.GeneratedFile {
	var sb strings.Builder

	sb.WriteString("import os\n")
	sb.WriteString("import yaml\n")
	sb.WriteString("from crewai import Agent, Crew, Process, Task\n")

	// Import tools
	if len(skills) > 0 {
		sb.WriteString("from tools import ")
		var toolNames []string
		for _, s := range skills {
			toolNames = append(toolNames, pythonSafe(s.Name))
		}
		sb.WriteString(strings.Join(toolNames, ", "))
		sb.WriteString("\n")
	}

	sb.WriteString("\n\n")
	sb.WriteString("class AgentCrew:\n")
	sb.WriteString("    \"\"\"AgentSpec-compiled CrewAI crew.\"\"\"\n\n")

	// Load config files
	sb.WriteString("    def __init__(self):\n")
	sb.WriteString("        config_dir = os.path.join(os.path.dirname(__file__), \"config\")\n")
	sb.WriteString("        with open(os.path.join(config_dir, \"agents.yaml\")) as f:\n")
	sb.WriteString("            self.agents_config = yaml.safe_load(f)\n")
	sb.WriteString("        with open(os.path.join(config_dir, \"tasks.yaml\")) as f:\n")
	sb.WriteString("            self.tasks_config = yaml.safe_load(f)\n\n")

	// Create agent methods
	for _, agent := range agents {
		safeName := pythonSafe(agent.Name)
		sb.WriteString(fmt.Sprintf("    def %s(self):\n", safeName))
		sb.WriteString(fmt.Sprintf("        config = self.agents_config[\"%s\"]\n", safeName))

		// Determine which tools this agent uses
		agentSkills := getStringSliceAttr(agent, "skills")

		sb.WriteString("        return Agent(\n")
		sb.WriteString("            role=config[\"role\"],\n")
		sb.WriteString("            goal=config[\"goal\"],\n")
		sb.WriteString("            backstory=config[\"backstory\"],\n")
		if len(agentSkills) > 0 {
			sb.WriteString("            tools=[")
			var toolRefs []string
			for _, s := range agentSkills {
				toolRefs = append(toolRefs, pythonSafe(s))
			}
			sb.WriteString(strings.Join(toolRefs, ", "))
			sb.WriteString("],\n")
		}
		sb.WriteString("            verbose=True,\n")
		if model := getStringAttr(agent, "model"); model != "" {
			sb.WriteString(fmt.Sprintf("            llm=%q,\n", model))
		}
		sb.WriteString("        )\n\n")
	}

	// Create task methods
	for _, agent := range agents {
		safeName := pythonSafe(agent.Name)
		sb.WriteString(fmt.Sprintf("    def %s_task(self):\n", safeName))
		sb.WriteString(fmt.Sprintf("        config = self.tasks_config[\"%s_task\"]\n", safeName))
		sb.WriteString("        return Task(\n")
		sb.WriteString("            description=config[\"description\"],\n")
		sb.WriteString("            expected_output=config[\"expected_output\"],\n")
		sb.WriteString(fmt.Sprintf("            agent=self.%s(),\n", safeName))
		sb.WriteString("        )\n\n")
	}

	// Create crew method
	sb.WriteString("    def crew(self):\n")
	sb.WriteString("        return Crew(\n")
	sb.WriteString("            agents=[")
	var agentRefs []string
	for _, a := range agents {
		agentRefs = append(agentRefs, fmt.Sprintf("self.%s()", pythonSafe(a.Name)))
	}
	sb.WriteString(strings.Join(agentRefs, ", "))
	sb.WriteString("],\n")
	sb.WriteString("            tasks=[")
	var taskRefs []string
	for _, a := range agents {
		taskRefs = append(taskRefs, fmt.Sprintf("self.%s_task()", pythonSafe(a.Name)))
	}
	sb.WriteString(strings.Join(taskRefs, ", "))
	sb.WriteString("],\n")
	sb.WriteString("            process=Process.sequential,\n")
	sb.WriteString("            verbose=True,\n")
	sb.WriteString("        )\n")

	return plugins.GeneratedFile{Path: "crew.py", Content: sb.String(), Mode: "0644"}
}

func (t *CrewAITarget) generateMain(name string) plugins.GeneratedFile {
	content := fmt.Sprintf(`#!/usr/bin/env python3
"""Entry point for the %s CrewAI project."""

import sys
from crew import AgentCrew


def main():
    inputs = {"input": " ".join(sys.argv[1:]) if len(sys.argv) > 1 else ""}
    crew = AgentCrew()
    result = crew.crew().kickoff(inputs=inputs)
    print(result)


if __name__ == "__main__":
    main()
`, name)

	return plugins.GeneratedFile{Path: "main.py", Content: content, Mode: "0755"}
}
