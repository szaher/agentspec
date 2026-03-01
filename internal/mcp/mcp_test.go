package mcp

import (
	"context"
	"testing"
)

// --- Pool Tests ---

func TestNewPool(t *testing.T) {
	pool := NewPool()
	if pool == nil {
		t.Fatal("expected non-nil Pool")
	}
}

func TestPoolAllEmpty(t *testing.T) {
	pool := NewPool()
	clients := pool.All()
	if len(clients) != 0 {
		t.Errorf("expected 0 clients in empty pool, got %d", len(clients))
	}
}

func TestPoolGetNonExistent(t *testing.T) {
	pool := NewPool()

	_, err := pool.Get("nonexistent-server")
	if err == nil {
		t.Fatal("expected error when getting non-existent server, got nil")
	}

	expectedMsg := `mcp server "nonexistent-server" not connected`
	if err.Error() != expectedMsg {
		t.Errorf("expected error %q, got %q", expectedMsg, err.Error())
	}
}

func TestPoolCloseEmpty(t *testing.T) {
	pool := NewPool()

	err := pool.Close()
	if err != nil {
		t.Errorf("expected no error closing empty pool, got: %v", err)
	}
}

func TestPoolCloseMultipleTimes(t *testing.T) {
	pool := NewPool()

	// Closing an empty pool multiple times should be safe
	if err := pool.Close(); err != nil {
		t.Errorf("first Close error: %v", err)
	}
	if err := pool.Close(); err != nil {
		t.Errorf("second Close error: %v", err)
	}
}

// --- ServerConfig Tests ---

func TestServerConfigConstruction(t *testing.T) {
	config := ServerConfig{
		Name:      "my-server",
		Transport: "stdio",
		Command:   "/usr/bin/my-mcp-server",
		Args:      []string{"--flag", "value"},
	}

	if config.Name != "my-server" {
		t.Errorf("expected Name='my-server', got %q", config.Name)
	}
	if config.Transport != "stdio" {
		t.Errorf("expected Transport='stdio', got %q", config.Transport)
	}
	if config.Command != "/usr/bin/my-mcp-server" {
		t.Errorf("expected Command='/usr/bin/my-mcp-server', got %q", config.Command)
	}
	if len(config.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(config.Args))
	}
	if config.Args[0] != "--flag" || config.Args[1] != "value" {
		t.Errorf("unexpected args: %v", config.Args)
	}
}

func TestServerConfigSSE(t *testing.T) {
	config := ServerConfig{
		Name:      "remote-server",
		Transport: "sse",
		URL:       "https://mcp.example.com/sse",
	}

	if config.Transport != "sse" {
		t.Errorf("expected Transport='sse', got %q", config.Transport)
	}
	if config.URL != "https://mcp.example.com/sse" {
		t.Errorf("expected URL='https://mcp.example.com/sse', got %q", config.URL)
	}
}

// --- Discovery Tests ---

func TestNewDiscovery(t *testing.T) {
	pool := NewPool()
	discovery := NewDiscovery(pool)

	if discovery == nil {
		t.Fatal("expected non-nil Discovery")
	}
	if discovery.pool != pool {
		t.Error("expected Discovery to reference the provided pool")
	}
}

// --- ToLLMTools Tests ---

func TestToLLMToolsEmpty(t *testing.T) {
	defs := ToLLMTools(nil)
	if len(defs) != 0 {
		t.Errorf("expected 0 tool definitions for nil input, got %d", len(defs))
	}
}

func TestToLLMToolsConversion(t *testing.T) {
	tools := []ToolInfo{
		{
			ServerName:  "fs-server",
			Name:        "read_file",
			Description: "Read a file from disk",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "file path",
					},
				},
			},
		},
		{
			ServerName:  "git-server",
			Name:        "git_status",
			Description: "Get git status",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
	}

	defs := ToLLMTools(tools)

	if len(defs) != 2 {
		t.Fatalf("expected 2 tool definitions, got %d", len(defs))
	}

	// First tool: name should be "server/tool"
	if defs[0].Name != "fs-server/read_file" {
		t.Errorf("expected name='fs-server/read_file', got %q", defs[0].Name)
	}
	if defs[0].Description != "Read a file from disk" {
		t.Errorf("expected description='Read a file from disk', got %q", defs[0].Description)
	}
	if defs[0].InputSchema == nil {
		t.Error("expected non-nil InputSchema")
	}

	// Second tool
	if defs[1].Name != "git-server/git_status" {
		t.Errorf("expected name='git-server/git_status', got %q", defs[1].Name)
	}
	if defs[1].Description != "Get git status" {
		t.Errorf("expected description='Get git status', got %q", defs[1].Description)
	}
}

// --- Client Tests (construction only, no subprocess) ---

func TestNewClient(t *testing.T) {
	config := ServerConfig{
		Name:      "test-server",
		Transport: "stdio",
		Command:   "echo",
		Args:      []string{"hello"},
	}

	client := NewClient(config)
	if client == nil {
		t.Fatal("expected non-nil Client")
	}
	if client.config.Name != "test-server" {
		t.Errorf("expected config.Name='test-server', got %q", client.config.Name)
	}
	if client.session != nil {
		t.Error("expected nil session before Connect()")
	}
}

func TestClientCloseWithoutConnect(t *testing.T) {
	client := NewClient(ServerConfig{Name: "test"})

	// Close without Connect should not error
	err := client.Close()
	if err != nil {
		t.Errorf("expected no error closing unconnected client, got: %v", err)
	}
}

// --- ToolInfo Tests ---

func TestToolInfoConstruction(t *testing.T) {
	ti := ToolInfo{
		ServerName:  "myserver",
		Name:        "search",
		Description: "Search for things",
		InputSchema: map[string]interface{}{"type": "object"},
	}

	if ti.ServerName != "myserver" {
		t.Errorf("expected ServerName='myserver', got %q", ti.ServerName)
	}
	if ti.Name != "search" {
		t.Errorf("expected Name='search', got %q", ti.Name)
	}
	if ti.Description != "Search for things" {
		t.Errorf("expected Description='Search for things', got %q", ti.Description)
	}
}

// --- Pool with manually stored clients ---

func TestPoolGetExistingClient(t *testing.T) {
	pool := NewPool()

	// Manually store a client (simulating a successful connection)
	client := NewClient(ServerConfig{Name: "server-a", Transport: "stdio", Command: "echo"})
	pool.clients.Store("server-a", client)

	got, err := pool.Get("server-a")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != client {
		t.Error("expected same client instance from Get")
	}
}

func TestPoolAllWithClients(t *testing.T) {
	pool := NewPool()

	c1 := NewClient(ServerConfig{Name: "s1"})
	c2 := NewClient(ServerConfig{Name: "s2"})
	c3 := NewClient(ServerConfig{Name: "s3"})
	pool.clients.Store("s1", c1)
	pool.clients.Store("s2", c2)
	pool.clients.Store("s3", c3)

	all := pool.All()
	if len(all) != 3 {
		t.Errorf("expected 3 clients, got %d", len(all))
	}

	// Verify all clients are present (order may vary with sync.Map)
	found := map[string]bool{}
	for _, c := range all {
		found[c.config.Name] = true
	}
	for _, name := range []string{"s1", "s2", "s3"} {
		if !found[name] {
			t.Errorf("expected client %q in All() results", name)
		}
	}
}

func TestPoolCloseWithClients(t *testing.T) {
	pool := NewPool()

	// Store clients that have no session (Close should succeed)
	c1 := NewClient(ServerConfig{Name: "s1"})
	c2 := NewClient(ServerConfig{Name: "s2"})
	pool.clients.Store("s1", c1)
	pool.clients.Store("s2", c2)

	err := pool.Close()
	if err != nil {
		t.Errorf("expected no error closing pool with unconnected clients, got: %v", err)
	}

	// After close, pool should be empty
	all := pool.All()
	if len(all) != 0 {
		t.Errorf("expected 0 clients after Close, got %d", len(all))
	}
}

func TestPoolGetAfterClose(t *testing.T) {
	pool := NewPool()

	c := NewClient(ServerConfig{Name: "s1"})
	pool.clients.Store("s1", c)

	pool.Close()

	_, err := pool.Get("s1")
	if err == nil {
		t.Fatal("expected error getting client after Close, got nil")
	}
}

// --- DiscoverTools with empty pool ---

func TestDiscoverToolsEmptyPool(t *testing.T) {
	pool := NewPool()
	discovery := NewDiscovery(pool)

	tools, err := discovery.DiscoverTools(context.Background())
	if err != nil {
		t.Fatalf("DiscoverTools error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected 0 tools from empty pool, got %d", len(tools))
	}
}

// --- Client ListTools/CallTool without connection ---

func TestClientListToolsNotConnected(t *testing.T) {
	client := NewClient(ServerConfig{Name: "test"})

	_, err := client.ListTools(context.Background())
	if err == nil {
		t.Fatal("expected error for ListTools without connection, got nil")
	}
	if err.Error() != "mcp client not connected" {
		t.Errorf("expected 'mcp client not connected', got %q", err.Error())
	}
}

func TestClientCallToolNotConnected(t *testing.T) {
	client := NewClient(ServerConfig{Name: "test"})

	_, err := client.CallTool(context.Background(), "some_tool", nil)
	if err == nil {
		t.Fatal("expected error for CallTool without connection, got nil")
	}
	if err.Error() != "mcp client not connected" {
		t.Errorf("expected 'mcp client not connected', got %q", err.Error())
	}
}

// --- ToLLMTools with single tool ---

func TestToLLMToolsSingle(t *testing.T) {
	tools := []ToolInfo{
		{
			ServerName:  "my-server",
			Name:        "do_something",
			Description: "Does something useful",
			InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
		},
	}

	defs := ToLLMTools(tools)
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Name != "my-server/do_something" {
		t.Errorf("expected name 'my-server/do_something', got %q", defs[0].Name)
	}
}

// --- Connect with unsupported transport ---

func TestPoolConnectUnsupportedTransport(t *testing.T) {
	pool := NewPool()

	_, err := pool.Connect(context.Background(), ServerConfig{
		Name:      "bad-server",
		Transport: "unsupported",
		Command:   "echo",
	})
	if err == nil {
		t.Fatal("expected error for unsupported transport, got nil")
	}
}

// --- Connect with invalid command (stdio) ---

func TestPoolConnectInvalidCommand(t *testing.T) {
	pool := NewPool()

	_, err := pool.Connect(context.Background(), ServerConfig{
		Name:      "bad-cmd",
		Transport: "stdio",
		Command:   "/nonexistent/binary/path",
	})
	if err == nil {
		t.Fatal("expected error for invalid command, got nil")
	}
}
