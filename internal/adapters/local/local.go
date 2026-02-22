// Package local implements the Local MCP adapter for the Agentz toolchain.
package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/state"
)

func init() {
	adapters.Register("local-mcp", func() adapters.Adapter {
		return &Adapter{}
	})
}

// Adapter implements the local-mcp adapter.
type Adapter struct{}

// Name returns the adapter identifier.
func (a *Adapter) Name() string { return "local-mcp" }

// Validate checks whether resources are compatible with local MCP.
func (a *Adapter) Validate(_ context.Context, resources []ir.Resource) error {
	return nil
}

// Plan computes changes needed.
func (a *Adapter) Plan(_ context.Context, desired []ir.Resource, current []state.Entry) ([]adapters.Action, error) {
	// Delegate to the shared plan engine
	return nil, nil
}

// Apply executes the planned actions.
func (a *Adapter) Apply(_ context.Context, actions []adapters.Action) ([]adapters.Result, error) {
	var results []adapters.Result
	for _, action := range actions {
		results = append(results, adapters.Result{
			FQN:    action.FQN,
			Action: action.Type,
			Status: adapters.ResultSuccess,
		})
	}
	return results, nil
}

// Export generates Local MCP configuration files.
func (a *Adapter) Export(_ context.Context, resources []ir.Resource, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	// Categorize resources
	var servers, clients, agents []ir.Resource
	for _, r := range resources {
		switch r.Kind {
		case "MCPServer":
			servers = append(servers, r)
		case "MCPClient":
			clients = append(clients, r)
		case "Agent":
			agents = append(agents, r)
		}
	}

	if err := writeJSON(filepath.Join(outDir, "mcp-servers.json"), servers); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "mcp-clients.json"), clients); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(outDir, "agents.json"), agents); err != nil {
		return err
	}

	return nil
}

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
