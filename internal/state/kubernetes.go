package state

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubernetesBackend implements Backend, HealthChecker, and Closer using Kubernetes ConfigMaps.
// Note: Uses ConfigMaps for simplicity; the full StateStore CRD is implemented in Phase 5.
type KubernetesBackend struct {
	client    *kubernetes.Clientset
	namespace string
	name      string
}

// NewKubernetesBackend creates a new KubernetesBackend with in-cluster configuration.
// Default namespace is "default", default name is "agentspec-state".
func NewKubernetesBackend(namespace, name string) (*KubernetesBackend, error) {
	if namespace == "" {
		namespace = "default"
	}
	if name == "" {
		name = "agentspec-state"
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesBackend{
		client:    clientset,
		namespace: namespace,
		name:      name,
	}, nil
}

// Load reads all state entries from the ConfigMap.
func (k *KubernetesBackend) Load() ([]Entry, error) {
	ctx := context.Background()
	cm, err := k.client.CoreV1().ConfigMaps(k.namespace).Get(ctx, k.name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("failed to get configmap: %w", err)
	}

	entriesJSON, ok := cm.Data["entries"]
	if !ok || entriesJSON == "" {
		return []Entry{}, nil
	}

	var entries []Entry
	if err := json.Unmarshal([]byte(entriesJSON), &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal entries: %w", err)
	}

	return entries, nil
}

// Save writes all state entries to the ConfigMap.
func (k *KubernetesBackend) Save(entries []Entry) error {
	ctx := context.Background()

	entriesJSON, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}

	// Try to get existing ConfigMap first
	cm, err := k.client.CoreV1().ConfigMaps(k.namespace).Get(ctx, k.name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new ConfigMap
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      k.name,
					Namespace: k.namespace,
				},
				Data: map[string]string{
					"entries": string(entriesJSON),
				},
			}
			_, err = k.client.CoreV1().ConfigMaps(k.namespace).Create(ctx, cm, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create configmap: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to get configmap: %w", err)
	}

	// Update existing ConfigMap with resourceVersion for optimistic concurrency
	cm.Data = map[string]string{
		"entries": string(entriesJSON),
	}
	_, err = k.client.CoreV1().ConfigMaps(k.namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update configmap: %w", err)
	}

	return nil
}

// Get retrieves a single entry by FQN.
func (k *KubernetesBackend) Get(fqn string) (*Entry, error) {
	entries, err := k.Load()
	if err != nil {
		return nil, err
	}

	for i := range entries {
		if entries[i].FQN == fqn {
			return &entries[i], nil
		}
	}

	return nil, fmt.Errorf("entry not found: %s", fqn)
}

// List returns all entries, optionally filtered by status.
func (k *KubernetesBackend) List(status *Status) ([]Entry, error) {
	entries, err := k.Load()
	if err != nil {
		return nil, err
	}

	if status == nil {
		return entries, nil
	}

	var filtered []Entry
	for _, entry := range entries {
		if entry.Status == *status {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

// Ping attempts to list ConfigMaps with limit=1 to verify connectivity.
func (k *KubernetesBackend) Ping(ctx context.Context) error {
	limit := int64(1)
	_, err := k.client.CoreV1().ConfigMaps(k.namespace).List(ctx, metav1.ListOptions{
		Limit: limit,
	})
	if err != nil {
		return fmt.Errorf("kubernetes health check failed: %w", err)
	}
	return nil
}

// Close is a no-op for KubernetesBackend.
func (k *KubernetesBackend) Close() error {
	return nil
}
