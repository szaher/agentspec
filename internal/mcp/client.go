// Package mcp provides MCP client wrappers for the AgentSpec runtime.
package mcp

import (
	"context"
	"fmt"
	"os/exec"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerConfig holds the configuration for connecting to an MCP server.
type ServerConfig struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"` // "stdio", "sse", "streamable-http"
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	URL       string   `json:"url,omitempty"` // For SSE/HTTP transports
}

// ToolInfo describes a tool available on an MCP server.
type ToolInfo struct {
	ServerName  string                 `json:"server_name"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Client wraps the MCP SDK client for a single server connection.
type Client struct {
	config  ServerConfig
	client  *mcpsdk.Client
	session *mcpsdk.ClientSession
}

// NewClient creates a new MCP client for the given server config.
func NewClient(config ServerConfig) *Client {
	return &Client{config: config}
}

// Connect establishes a connection to the MCP server.
func (c *Client) Connect(ctx context.Context) error {
	impl := &mcpsdk.Implementation{
		Name:    "agentspec",
		Version: "0.3.0",
	}
	c.client = mcpsdk.NewClient(impl, nil)

	switch c.config.Transport {
	case "stdio":
		cmd := exec.CommandContext(ctx, c.config.Command, c.config.Args...)
		transport := &mcpsdk.CommandTransport{
			Command: cmd,
		}
		session, err := c.client.Connect(ctx, transport, nil)
		if err != nil {
			return fmt.Errorf("mcp connect to %s: %w", c.config.Name, err)
		}
		c.session = session
	default:
		return fmt.Errorf("unsupported MCP transport: %s", c.config.Transport)
	}

	return nil
}

// ListTools returns all tools available on this server.
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	if c.session == nil {
		return nil, fmt.Errorf("mcp client not connected")
	}

	var tools []ToolInfo
	for tool, err := range c.session.Tools(ctx, nil) {
		if err != nil {
			return nil, fmt.Errorf("mcp list tools: %w", err)
		}
		schema := make(map[string]interface{})
		if tool.InputSchema != nil {
			schema["type"] = "object"
		}
		tools = append(tools, ToolInfo{
			ServerName:  c.config.Name,
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: schema,
		})
	}

	return tools, nil
}

// CallTool invokes a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	if c.session == nil {
		return "", fmt.Errorf("mcp client not connected")
	}

	result, err := c.session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("mcp call tool %s: %w", name, err)
	}

	if result.IsError {
		return "", fmt.Errorf("mcp tool %s returned error", name)
	}

	// Extract text content from result
	var text string
	for _, content := range result.Content {
		if tc, ok := content.(*mcpsdk.TextContent); ok {
			if text != "" {
				text += "\n"
			}
			text += tc.Text
		}
	}

	return text, nil
}

// Close gracefully closes the MCP connection.
func (c *Client) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}
