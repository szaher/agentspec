package mcp

import (
	"context"
	"fmt"

	"github.com/szaher/designs/agentz/internal/llm"
)

// Discovery aggregates tool information from multiple MCP servers
// and converts them to LLM tool definitions.
type Discovery struct {
	pool *Pool
}

// NewDiscovery creates a new tool discovery service.
func NewDiscovery(pool *Pool) *Discovery {
	return &Discovery{pool: pool}
}

// DiscoverTools lists all tools from all connected MCP servers.
func (d *Discovery) DiscoverTools(ctx context.Context) ([]ToolInfo, error) {
	clients := d.pool.All()
	var allTools []ToolInfo

	for _, client := range clients {
		tools, err := client.ListTools(ctx)
		if err != nil {
			return nil, fmt.Errorf("discover tools from %s: %w", client.config.Name, err)
		}
		allTools = append(allTools, tools...)
	}

	return allTools, nil
}

// ToLLMTools converts MCP tool info to LLM tool definitions.
func ToLLMTools(tools []ToolInfo) []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, len(tools))
	for i, t := range tools {
		defs[i] = llm.ToolDefinition{
			Name:        t.ServerName + "/" + t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
	}
	return defs
}
