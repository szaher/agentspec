// Package runtime implements the AgentSpec runtime lifecycle and HTTP server.
package runtime

import (
	"fmt"

	"github.com/szaher/designs/agentz/internal/ir"
)

// AgentConfig holds the runtime configuration for a single agent.
type AgentConfig struct {
	Name        string           `json:"name"`
	FQN         string           `json:"fqn"`
	Model       string           `json:"model"`
	System      string           `json:"system"`
	Skills      []string         `json:"skills"`
	Strategy    string           `json:"strategy"`
	MaxTurns    int              `json:"max_turns"`
	Timeout     string           `json:"timeout"`
	TokenBudget int              `json:"token_budget"`
	Temperature *float64         `json:"temperature,omitempty"`
	Stream      bool             `json:"stream"`
	OnError     string           `json:"on_error"`
	MaxRetries  int              `json:"max_retries"`
	Fallback    string           `json:"fallback,omitempty"`
	Memory      *MemoryConfig    `json:"memory,omitempty"`
	Delegates   []DelegateConfig `json:"delegates,omitempty"`
}

// MemoryConfig holds memory strategy configuration.
type MemoryConfig struct {
	Strategy    string `json:"strategy"`
	MaxMessages int    `json:"max_messages"`
}

// DelegateConfig holds agent delegation configuration.
type DelegateConfig struct {
	Agent     string `json:"agent"`
	Condition string `json:"condition"`
}

// SkillConfig holds the runtime configuration for a skill.
type SkillConfig struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tool        map[string]interface{} `json:"tool"`
}

// ServerConfig holds the runtime configuration for an MCP server.
type ServerConfig struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
}

// PipelineConfig holds the runtime configuration for a pipeline.
type PipelineConfig struct {
	Name  string               `json:"name"`
	Steps []PipelineStepConfig `json:"steps"`
}

// PipelineStepConfig holds the runtime configuration for a pipeline step.
type PipelineStepConfig struct {
	Name      string   `json:"name"`
	AgentRef  string   `json:"agent_ref"`
	Input     string   `json:"input,omitempty"`
	Output    string   `json:"output,omitempty"`
	DependsOn []string `json:"depends_on,omitempty"`
}

// RuntimeConfig is the complete runtime configuration parsed from IR.
type RuntimeConfig struct {
	PackageName string            `json:"package_name"`
	Agents      []AgentConfig     `json:"agents"`
	Skills      []SkillConfig     `json:"skills"`
	MCPServers  []ServerConfig    `json:"mcp_servers"`
	Prompts     map[string]string `json:"prompts"`
	Pipelines   []PipelineConfig  `json:"pipelines,omitempty"`
}

// FromIR converts an IR Document into a RuntimeConfig.
func FromIR(doc *ir.Document) (*RuntimeConfig, error) {
	config := &RuntimeConfig{
		PackageName: doc.Package.Name,
		Prompts:     make(map[string]string),
	}

	for _, r := range doc.Resources {
		switch r.Kind {
		case "Agent":
			agent := AgentConfig{
				Name:     r.Name,
				FQN:      r.FQN,
				Strategy: "react",
				MaxTurns: 10,
				Stream:   true,
				OnError:  "retry",
			}
			if m, ok := r.Attributes["model"].(string); ok {
				agent.Model = m
			}
			if s, ok := r.Attributes["strategy"].(string); ok {
				agent.Strategy = s
			}
			if mt, ok := r.Attributes["max_turns"]; ok {
				agent.MaxTurns = toInt(mt)
			}
			if t, ok := r.Attributes["timeout"].(string); ok {
				agent.Timeout = t
			}
			if tb, ok := r.Attributes["token_budget"]; ok {
				agent.TokenBudget = toInt(tb)
			}
			if temp, ok := r.Attributes["temperature"]; ok {
				f := toFloat(temp)
				agent.Temperature = &f
			}
			if s, ok := r.Attributes["stream"].(bool); ok {
				agent.Stream = s
			}
			if oe, ok := r.Attributes["on_error"].(string); ok {
				agent.OnError = oe
			}
			if mr, ok := r.Attributes["max_retries"]; ok {
				agent.MaxRetries = toInt(mr)
			}
			if fb, ok := r.Attributes["fallback"].(string); ok {
				agent.Fallback = fb
			}
			if refs, ok := r.Attributes["skill_refs"].([]interface{}); ok {
				for _, ref := range refs {
					if s, ok := ref.(string); ok {
						agent.Skills = append(agent.Skills, s)
					}
				}
			}
			config.Agents = append(config.Agents, agent)

		case "Prompt":
			if c, ok := r.Attributes["content"].(string); ok {
				config.Prompts[r.Name] = c
			}

		case "Skill":
			skill := SkillConfig{
				Name: r.Name,
			}
			if d, ok := r.Attributes["description"].(string); ok {
				skill.Description = d
			}
			if t, ok := r.Attributes["tool"].(map[string]interface{}); ok {
				skill.Tool = t
			}
			config.Skills = append(config.Skills, skill)

		case "MCPServer":
			server := ServerConfig{
				Name: r.Name,
			}
			if t, ok := r.Attributes["transport"].(string); ok {
				server.Transport = t
			}
			if c, ok := r.Attributes["command"].(string); ok {
				server.Command = c
			}
			if a, ok := r.Attributes["args"].([]interface{}); ok {
				for _, arg := range a {
					if s, ok := arg.(string); ok {
						server.Args = append(server.Args, s)
					}
				}
			}
			config.MCPServers = append(config.MCPServers, server)

		case "Pipeline":
			p := PipelineConfig{Name: r.Name}
			if steps, ok := r.Attributes["steps"].([]interface{}); ok {
				for _, step := range steps {
					if stepMap, ok := step.(map[string]interface{}); ok {
						sc := PipelineStepConfig{}
						if n, ok := stepMap["name"].(string); ok {
							sc.Name = n
						}
						if a, ok := stepMap["agent"].(string); ok {
							sc.AgentRef = a
						}
						if in, ok := stepMap["input"].(string); ok {
							sc.Input = in
						}
						if out, ok := stepMap["output"].(string); ok {
							sc.Output = out
						}
						if deps, ok := stepMap["depends_on"].([]interface{}); ok {
							for _, d := range deps {
								if ds, ok := d.(string); ok {
									sc.DependsOn = append(sc.DependsOn, ds)
								}
							}
						}
						p.Steps = append(p.Steps, sc)
					}
				}
			}
			config.Pipelines = append(config.Pipelines, p)
		}
	}

	// Resolve prompt references for agents
	for i, agent := range config.Agents {
		for _, r := range doc.Resources {
			if r.Kind == "Agent" && r.Name == agent.Name {
				if refs, ok := r.Attributes["prompt_refs"].([]interface{}); ok {
					for _, ref := range refs {
						if name, ok := ref.(string); ok {
							if content, exists := config.Prompts[name]; exists {
								if config.Agents[i].System != "" {
									config.Agents[i].System += "\n\n"
								}
								config.Agents[i].System += content
							}
						}
					}
				}
			}
		}
	}

	if len(config.Agents) == 0 {
		return nil, fmt.Errorf("no agents found in IR document")
	}

	return config, nil
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
