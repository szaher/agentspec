package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&ClusterPolicy{}, &ClusterPolicyList{})
}

// ClusterPolicySpec defines the desired state of a ClusterPolicy.
type ClusterPolicySpec struct {
	PolicySpecFields `json:",inline"`
}

// ClusterPolicyStatus defines the observed state of a ClusterPolicy.
type ClusterPolicyStatus struct {
	PolicyStatusFields `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="NAME",type=string,JSONPath=`.metadata.name`
// +kubebuilder:printcolumn:name="AFFECTED-AGENTS",type=integer,JSONPath=`.status.affectedAgentCount`
// +kubebuilder:printcolumn:name="VIOLATIONS",type=integer,JSONPath=`.status.violationCount`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// ClusterPolicy is the Schema for the clusterpolicies API. It is cluster-scoped.
type ClusterPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterPolicySpec   `json:"spec,omitempty"`
	Status ClusterPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterPolicyList contains a list of ClusterPolicy resources.
type ClusterPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterPolicy `json:"items"`
}
