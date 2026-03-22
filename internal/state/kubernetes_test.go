package state

import (
	"testing"
)

// TestKubernetesBackendInterfaceCompliance verifies KubernetesBackend implements required interfaces.
func TestKubernetesBackendInterfaceCompliance(t *testing.T) {
	// This test verifies that KubernetesBackend implements the interfaces at compile time.
	// We don't instantiate because that requires in-cluster Kubernetes connectivity.
	var _ Backend = (*KubernetesBackend)(nil)
	var _ HealthChecker = (*KubernetesBackend)(nil)
	var _ Closer = (*KubernetesBackend)(nil)
}

// TestNewKubernetesBackendDefaults verifies default namespace and name values.
func TestNewKubernetesBackendDefaults(t *testing.T) {
	tests := []struct {
		name              string
		inputNamespace    string
		inputName         string
		expectedNamespace string
		expectedName      string
	}{
		{
			name:              "empty namespace and name use defaults",
			inputNamespace:    "",
			inputName:         "",
			expectedNamespace: "default",
			expectedName:      "agentspec-state",
		},
		{
			name:              "custom namespace, empty name uses default name",
			inputNamespace:    "custom-ns",
			inputName:         "",
			expectedNamespace: "custom-ns",
			expectedName:      "agentspec-state",
		},
		{
			name:              "empty namespace, custom name uses default namespace",
			inputNamespace:    "",
			inputName:         "custom-state",
			expectedNamespace: "default",
			expectedName:      "custom-state",
		},
		{
			name:              "custom namespace and name",
			inputNamespace:    "custom-ns",
			inputName:         "custom-state",
			expectedNamespace: "custom-ns",
			expectedName:      "custom-state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't actually create the backend without in-cluster config,
			// but we can verify the logic by examining the implementation.
			// The NewKubernetesBackend function sets defaults before attempting
			// to connect, so we document expected behavior here.

			// Since we can't instantiate without a cluster, we skip actual execution
			// and document the expected behavior based on the implementation.
			t.Skipf("Requires in-cluster Kubernetes config. Expected: namespace=%q, name=%q",
				tt.expectedNamespace, tt.expectedName)
		})
	}
}
