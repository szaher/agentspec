package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StateStoreSpec defines the desired state of a StateStore.
type StateStoreSpec struct {
	// Scope identifies the package or namespace scope for this state store.
	// +kubebuilder:validation:Required
	Scope string `json:"scope"`
}

// StateEntryStatus represents a single state entry stored in the CRD status.
type StateEntryStatus struct {
	// FQN is the fully qualified name of the resource.
	FQN string `json:"fqn"`

	// Hash is the content hash of the resource.
	Hash string `json:"hash"`

	// Status is the lifecycle status: applied, failed, or orphaned.
	// +kubebuilder:validation:Enum=applied;failed;orphaned
	Status string `json:"status"`

	// LastApplied is the timestamp of the last successful apply.
	// +optional
	LastApplied metav1.Time `json:"lastApplied,omitempty"`

	// Adapter is the adapter name used for deployment.
	// +optional
	Adapter string `json:"adapter,omitempty"`

	// Error is the error message if status is failed.
	// +optional
	Error string `json:"error,omitempty"`

	// OrphanedAt is the timestamp when the entry was first marked orphaned.
	// +optional
	OrphanedAt metav1.Time `json:"orphanedAt,omitempty"`
}

// StateStoreStatus defines the observed state of a StateStore.
type StateStoreStatus struct {
	// Entries contains all state entries for this scope.
	// +optional
	Entries []StateEntryStatus `json:"entries,omitempty"`

	// LastWrite is the timestamp of the last write operation.
	// +optional
	LastWrite metav1.Time `json:"lastWrite,omitempty"`

	// Healthy indicates whether the state store is operating normally.
	// +optional
	Healthy bool `json:"healthy,omitempty"`

	// Conditions represent the latest available observations of the StateStore's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Scope",type=string,JSONPath=`.spec.scope`
// +kubebuilder:printcolumn:name="Entries",type=integer,JSONPath=`.status.entries`
// +kubebuilder:printcolumn:name="Healthy",type=boolean,JSONPath=`.status.healthy`

// StateStore is the Schema for the statestores API.
type StateStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StateStoreSpec   `json:"spec,omitempty"`
	Status StateStoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StateStoreList contains a list of StateStore.
type StateStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StateStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StateStore{}, &StateStoreList{})
}
