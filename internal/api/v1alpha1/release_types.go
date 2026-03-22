package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ReleaseSpec defines the desired state of a Release.
type ReleaseSpec struct {
	// AgentRef is the name of the Agent resource this release captures.
	// +kubebuilder:validation:Required
	AgentRef string `json:"agentRef"`

	// Version is the semantic version string for this release.
	// +kubebuilder:validation:Required
	Version string `json:"version"`

	// Snapshot stores the captured agent specification as raw JSON.
	// This preserves the exact agent configuration at the time of release.
	// +kubebuilder:validation:Required
	Snapshot runtime.RawExtension `json:"snapshot"`

	// Notes is an optional human-readable description of the release.
	// +optional
	Notes string `json:"notes,omitempty"`

	// PromoteTo is the target environment or stage to promote this release to.
	// +optional
	PromoteTo string `json:"promoteTo,omitempty"`
}

// ReleaseStatus defines the observed state of a Release.
type ReleaseStatus struct {
	// Phase is the current lifecycle phase of the release.
	// +kubebuilder:validation:Enum=Created;Promoted;RolledBack;Superseded
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the Release's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// PromotedAt is the timestamp when the release was promoted.
	// +optional
	PromotedAt *metav1.Time `json:"promotedAt,omitempty"`

	// PromotedTo is the environment or stage the release was promoted to.
	// +optional
	PromotedTo string `json:"promotedTo,omitempty"`

	// SupersededBy is the name of the Release that superseded this one.
	// +optional
	SupersededBy string `json:"supersededBy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGENT",type=string,JSONPath=`.spec.agentRef`
// +kubebuilder:printcolumn:name="VERSION",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Release is the Schema for the releases API.
// It represents an immutable snapshot of an agent configuration at a specific version.
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseSpec   `json:"spec,omitempty"`
	Status ReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReleaseList contains a list of Release resources.
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
