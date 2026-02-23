package mcp

import (
	"context"
	"fmt"
	"sync"
)

// Pool manages a set of MCP client connections keyed by server name.
type Pool struct {
	mu      sync.Mutex
	clients map[string]*Client
}

// NewPool creates an empty connection pool.
func NewPool() *Pool {
	return &Pool{
		clients: make(map[string]*Client),
	}
}

// Connect creates and connects an MCP client for the given server config.
// If a client already exists for this server name, it is returned as-is.
func (p *Pool) Connect(ctx context.Context, config ServerConfig) (*Client, error) {
	p.mu.Lock()
	if c, ok := p.clients[config.Name]; ok {
		p.mu.Unlock()
		return c, nil
	}
	p.mu.Unlock()

	client := NewClient(config)
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("pool connect %s: %w", config.Name, err)
	}

	p.mu.Lock()
	p.clients[config.Name] = client
	p.mu.Unlock()

	return client, nil
}

// Get returns an existing client by server name, or an error if not connected.
func (p *Pool) Get(name string) (*Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c, ok := p.clients[name]
	if !ok {
		return nil, fmt.Errorf("mcp server %q not connected", name)
	}
	return c, nil
}

// All returns all connected clients.
func (p *Pool) All() []*Client {
	p.mu.Lock()
	defer p.mu.Unlock()
	clients := make([]*Client, 0, len(p.clients))
	for _, c := range p.clients {
		clients = append(clients, c)
	}
	return clients
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var firstErr error
	for name, c := range p.clients {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close %s: %w", name, err)
		}
	}
	p.clients = make(map[string]*Client)
	return firstErr
}
