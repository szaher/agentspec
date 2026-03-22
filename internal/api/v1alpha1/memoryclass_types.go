// Package v1alpha1 contains API Schema definitions for the agentspec.io v1alpha1 API group.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemoryClassSpec defines the desired state of MemoryClass.
type MemoryClassSpec struct {
	// Strategy is the memory management strategy to use.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=sliding_window;summary;full
	Strategy string `json:"strategy"`

	// MaxMessages is the maximum number of messages to retain.
	// +kubebuilder:default=100
	// +optional
	MaxMessages int32 `json:"maxMessages,omitempty"`

	// TTL is the time-to-live for memory entries (e.g., "24h", "7d").
	// +optional
	TTL string `json:"ttl,omitempty"`

	// Backend is the storage backend for memory data.
	// +kubebuilder:default="in-memory"
	// +kubebuilder:validation:Enum=in-memory;redis
	// +optional
	Backend string `json:"backend,omitempty"`

	// BackendConfig holds backend-specific configuration key-value pairs.
	// +optional
	BackendConfig map[string]string `json:"backendConfig,omitempty"`
}

// MemoryClassStatus defines the observed state of MemoryClass.
type MemoryClassStatus struct {
	// Conditions represent the latest available observations of the MemoryClass's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// SessionCount is the number of active sessions using this MemoryClass.
	// +optional
	SessionCount int32 `json:"sessionCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="NAME",type=string,JSONPath=`.metadata.name`
// +kubebuilder:printcolumn:name="STRATEGY",type=string,JSONPath=`.spec.strategy`
// +kubebuilder:printcolumn:name="BACKEND",type=string,JSONPath=`.spec.backend`
// +kubebuilder:printcolumn:name="SESSIONS",type=integer,JSONPath=`.status.sessionCount`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// MemoryClass is the Schema for the memoryclasses API.
// It is a cluster-scoped resource that defines memory management policies for agents.
type MemoryClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemoryClassSpec   `json:"spec,omitempty"`
	Status MemoryClassStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MemoryClassList contains a list of MemoryClass.
type MemoryClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MemoryClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MemoryClass{}, &MemoryClassList{})
}
