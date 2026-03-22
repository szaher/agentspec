package state

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

const (
	defaultDialTimeout = 5 * time.Second
	defaultPrefix      = "/agentspec/state/"
	lockKey            = "lock"
)

// EtcdBackend implements Backend, HealthChecker, Locker, and Closer using etcd v3.
type EtcdBackend struct {
	cli     *clientv3.Client
	prefix  string
	session *concurrency.Session
	mutex   *concurrency.Mutex
}

// NewEtcdBackend creates an etcd-backed state backend.
// endpoints is a comma-separated list of etcd server addresses.
// prefix defaults to "/agentspec/state/" if empty.
func NewEtcdBackend(endpoints string, prefix string) (*EtcdBackend, error) {
	if endpoints == "" {
		return nil, fmt.Errorf("etcd endpoints cannot be empty")
	}

	if prefix == "" {
		prefix = defaultPrefix
	}

	// Ensure prefix ends with /
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	endpointList := strings.Split(endpoints, ",")
	for i := range endpointList {
		endpointList[i] = strings.TrimSpace(endpointList[i])
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpointList,
		DialTimeout: defaultDialTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &EtcdBackend{
		cli:    cli,
		prefix: prefix,
	}, nil
}

// Load reads all state entries from etcd.
func (e *EtcdBackend) Load() ([]Entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
	defer cancel()

	key := e.prefix + "entries/"
	resp, err := e.cli.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to load entries from etcd: %w", err)
	}

	entries := make([]Entry, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var entry Entry
		if err := json.Unmarshal(kv.Value, &entry); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entry %s: %w", kv.Key, err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Save writes all state entries to etcd, removing entries that no longer exist.
func (e *EtcdBackend) Save(entries []Entry) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
	defer cancel()

	// Load existing keys to determine what to delete
	key := e.prefix + "entries/"
	resp, err := e.cli.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return fmt.Errorf("failed to list existing entries: %w", err)
	}

	existingKeys := make(map[string]bool)
	for _, kv := range resp.Kvs {
		existingKeys[string(kv.Key)] = true
	}

	// Save new/updated entries
	newKeys := make(map[string]bool)
	for _, entry := range entries {
		entryKey := e.prefix + "entries/" + entry.FQN
		newKeys[entryKey] = true

		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal entry %s: %w", entry.FQN, err)
		}

		if _, err := e.cli.Put(ctx, entryKey, string(data)); err != nil {
			return fmt.Errorf("failed to save entry %s: %w", entry.FQN, err)
		}
	}

	// Delete entries that no longer exist
	for existingKey := range existingKeys {
		if !newKeys[existingKey] {
			if _, err := e.cli.Delete(ctx, existingKey); err != nil {
				return fmt.Errorf("failed to delete entry %s: %w", existingKey, err)
			}
		}
	}

	return nil
}

// Get retrieves a single entry by FQN.
func (e *EtcdBackend) Get(fqn string) (*Entry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
	defer cancel()

	key := e.prefix + "entries/" + fqn
	resp, err := e.cli.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry %s: %w", fqn, err)
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	var entry Entry
	if err := json.Unmarshal(resp.Kvs[0].Value, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entry %s: %w", fqn, err)
	}

	return &entry, nil
}

// List returns all entries, optionally filtered by status.
func (e *EtcdBackend) List(status *Status) ([]Entry, error) {
	entries, err := e.Load()
	if err != nil {
		return nil, err
	}

	if status == nil {
		return entries, nil
	}

	filtered := make([]Entry, 0)
	for _, entry := range entries {
		if entry.Status == *status {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

// Ping checks the health of the etcd cluster.
func (e *EtcdBackend) Ping(ctx context.Context) error {
	// Get the first endpoint
	endpoints := e.cli.Endpoints()
	if len(endpoints) == 0 {
		return fmt.Errorf("no etcd endpoints configured")
	}

	_, err := e.cli.Status(ctx, endpoints[0])
	if err != nil {
		return fmt.Errorf("etcd health check failed: %w", err)
	}

	return nil
}

// Lock acquires a distributed lock using etcd.
func (e *EtcdBackend) Lock(ctx context.Context) error {
	// Create a session if we don't have one
	if e.session == nil {
		session, err := concurrency.NewSession(e.cli)
		if err != nil {
			return fmt.Errorf("failed to create etcd session: %w", err)
		}
		e.session = session
	}

	// Create a mutex if we don't have one
	if e.mutex == nil {
		e.mutex = concurrency.NewMutex(e.session, e.prefix+lockKey)
	}

	if err := e.mutex.Lock(ctx); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	return nil
}

// Unlock releases the distributed lock.
func (e *EtcdBackend) Unlock() error {
	if e.mutex == nil {
		return fmt.Errorf("no lock to release")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDialTimeout)
	defer cancel()

	if err := e.mutex.Unlock(ctx); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}

// Close closes the etcd client connection.
func (e *EtcdBackend) Close() error {
	// Close session if it exists
	if e.session != nil {
		if err := e.session.Close(); err != nil {
			return fmt.Errorf("failed to close etcd session: %w", err)
		}
	}

	// Close client
	if err := e.cli.Close(); err != nil {
		return fmt.Errorf("failed to close etcd client: %w", err)
	}

	return nil
}
