// Package local implements the Local MCP adapter for the AgentSpec toolchain.
package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/szaher/designs/agentz/internal/adapters"
	"github.com/szaher/designs/agentz/internal/ir"
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

// Status returns an empty status list for the local-mcp adapter.
func (a *Adapter) Status(_ context.Context) ([]adapters.ResourceStatus, error) {
	return nil, nil
}

// Logs is not supported for the local-mcp adapter.
func (a *Adapter) Logs(_ context.Context, w io.Writer, _ adapters.LogOptions) error {
	_, err := fmt.Fprintln(w, "Log streaming is not supported for local-mcp adapter.")
	return err
}

// Destroy is a no-op for the local-mcp adapter.
func (a *Adapter) Destroy(_ context.Context) ([]adapters.Result, error) {
	return nil, nil
}

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
