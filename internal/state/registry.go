package state

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// BackendFactory creates a Backend from configuration properties.
type BackendFactory func(props map[string]string) (Backend, error)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]BackendFactory)
)

// Register registers a backend factory for a given type name.
func Register(typeName string, factory BackendFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[typeName] = factory
}

// New creates a Backend by type name and properties.
func New(typeName string, props map[string]string) (Backend, error) {
	registryMu.RLock()
	factory, ok := registry[typeName]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown state backend type %q; available types: %v", typeName, Available())
	}
	return factory(props)
}

// Available returns the list of registered backend type names.
func Available() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func init() {
	Register("local", func(props map[string]string) (Backend, error) {
		path := props["path"]
		if path == "" {
			path = ".agentspec.state.json"
		}
		return NewLocalBackend(path), nil
	})

	Register("etcd", func(props map[string]string) (Backend, error) {
		endpoints := props["endpoints"]
		if endpoints == "" {
			return nil, fmt.Errorf("etcd backend requires 'endpoints' property")
		}
		prefix := props["prefix"]
		return NewEtcdBackend(endpoints, prefix)
	})

	Register("postgres", func(props map[string]string) (Backend, error) {
		dsn := props["dsn"]
		if dsn == "" {
			return nil, fmt.Errorf("postgres backend requires 'dsn' property")
		}
		return NewPostgresBackend(dsn, props["table"])
	})

	Register("s3", func(props map[string]string) (Backend, error) {
		bucket := props["bucket"]
		if bucket == "" {
			return nil, fmt.Errorf("s3 backend requires 'bucket' property")
		}
		return NewS3Backend(context.Background(), bucket, props["region"], props["prefix"], props["endpoint"])
	})

	Register("kubernetes", func(props map[string]string) (Backend, error) {
		namespace := props["namespace"]
		name := props["name"]
		return NewKubernetesBackend(namespace, name)
	})
}
