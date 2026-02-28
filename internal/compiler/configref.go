// Package compiler implements the AgentSpec compilation pipeline that
// transforms .ias files into deployable artifacts.
package compiler

import (
	"fmt"
	"strings"
	"time"

	"github.com/szaher/designs/agentz/internal/runtime"
)

// AgentConfigRef holds config reference info for one agent.
type AgentConfigRef struct {
	AgentName string
	Params    []runtime.ConfigParamDef
}

// GenerateConfigRef produces a markdown config reference document
// listing all required and optional configuration parameters for
// all agents in the compiled artifact.
func GenerateConfigRef(agents []AgentConfigRef, artifactName string) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Configuration Reference: %s\n\n", artifactName)
	fmt.Fprintf(&b, "Generated: %s\n\n", time.Now().UTC().Format("2006-01-02 15:04:05 UTC"))

	for _, agent := range agents {
		fmt.Fprintf(&b, "## Agent: %s\n\n", agent.AgentName)

		if len(agent.Params) == 0 {
			b.WriteString("No configuration parameters declared.\n\n")
			continue
		}

		b.WriteString("| Parameter | Type | Required | Secret | Default | Env Variable | Description |\n")
		b.WriteString("|-----------|------|----------|--------|---------|-------------|-------------|\n")

		for _, p := range agent.Params {
			envVar := configEnvKey(agent.AgentName, p.Name)
			required := "No"
			if p.Required {
				required = "**Yes**"
			}
			secret := "No"
			if p.Secret {
				secret = "Yes"
			}
			def := "-"
			if p.HasDefault {
				def = fmt.Sprintf("`%s`", p.Default)
			}
			desc := p.Description
			if desc == "" {
				desc = "-"
			}

			fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | `%s` | %s |\n",
				p.Name, p.Type, required, secret, def, envVar, desc)
		}
		b.WriteString("\n")

		// Generate environment setup example
		b.WriteString("### Quick Setup\n\n")
		b.WriteString("```bash\n")
		for _, p := range agent.Params {
			envVar := configEnvKey(agent.AgentName, p.Name)
			if p.Secret {
				fmt.Fprintf(&b, "export %s=<secret>  # %s (required: %v)\n", envVar, p.Description, p.Required)
			} else if p.HasDefault {
				fmt.Fprintf(&b, "# export %s=%s  # %s (optional, default shown)\n", envVar, p.Default, p.Description)
			} else if p.Required {
				fmt.Fprintf(&b, "export %s=<value>  # %s\n", envVar, p.Description)
			} else {
				fmt.Fprintf(&b, "# export %s=<value>  # %s (optional)\n", envVar, p.Description)
			}
		}
		b.WriteString("```\n\n")
	}

	return b.String()
}

func configEnvKey(agentName, paramName string) string {
	agent := strings.ToUpper(strings.ReplaceAll(agentName, "-", "_"))
	param := strings.ToUpper(strings.ReplaceAll(paramName, "-", "_"))
	return "AGENTSPEC_" + agent + "_" + param
}
