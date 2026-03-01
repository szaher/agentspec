package mcp

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
)

// Pool manages a set of MCP client connections keyed by server name.
// Uses sync.Map for concurrent-safe reads and singleflight for deduplication.
type Pool struct {
	clients sync.Map // map[string]*Client
	group   singleflight.Group
	mu      sync.Mutex // for Close()
}

// NewPool creates an empty connection pool.
func NewPool() *Pool {
	return &Pool{}
}

// Connect creates and connects an MCP client for the given server config.
// If a client already exists for this server name, it is returned as-is.
// Uses singleflight to ensure only one connection attempt per key.
func (p *Pool) Connect(ctx context.Context, config ServerConfig) (*Client, error) {
	// Fast path: check if already connected
	if c, ok := p.clients.Load(config.Name); ok {
		return c.(*Client), nil
	}

	// Use singleflight to deduplicate concurrent connection attempts
	result, err, _ := p.group.Do(config.Name, func() (interface{}, error) {
		// Double-check after acquiring singleflight
		if c, ok := p.clients.Load(config.Name); ok {
			return c.(*Client), nil
		}

		client := NewClient(config)
		if err := client.Connect(ctx); err != nil {
			return nil, fmt.Errorf("pool connect %s: %w", config.Name, err)
		}

		p.clients.Store(config.Name, client)
		return client, nil
	})

	if err != nil {
		return nil, err
	}
	return result.(*Client), nil
}

// Get returns an existing client by server name, or an error if not connected.
func (p *Pool) Get(name string) (*Client, error) {
	c, ok := p.clients.Load(name)
	if !ok {
		return nil, fmt.Errorf("mcp server %q not connected", name)
	}
	return c.(*Client), nil
}

// All returns all connected clients.
func (p *Pool) All() []*Client {
	var clients []*Client
	p.clients.Range(func(_, value interface{}) bool {
		clients = append(clients, value.(*Client))
		return true
	})
	return clients
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var firstErr error
	p.clients.Range(func(key, value interface{}) bool {
		name := key.(string)
		c := value.(*Client)
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close %s: %w", name, err)
		}
		p.clients.Delete(key)
		return true
	})
	return firstErr
}
