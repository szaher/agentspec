// Package v1alpha1 contains API Schema definitions for the agentspec.io v1alpha1 API group.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretRef references a key within a Kubernetes Secret.
type SecretRef struct {
	// Name is the name of the Secret.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Key is the key within the Secret data.
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// AgentSpec defines the desired state of an Agent.
type AgentSpec struct {
	// Model is the LLM model identifier to use for this agent.
	// +kubebuilder:validation:Required
	Model string `json:"model"`

	// PromptRef is an optional reference to a ConfigMap or resource containing the system prompt.
	// +optional
	PromptRef string `json:"promptRef,omitempty"`

	// Strategy is the agent execution strategy.
	// +kubebuilder:default="react"
	// +optional
	Strategy string `json:"strategy,omitempty"`

	// MaxTurns is the maximum number of LLM interaction turns.
	// +kubebuilder:default=10
	// +optional
	MaxTurns int32 `json:"maxTurns,omitempty"`

	// Stream enables streaming responses from the LLM.
	// +kubebuilder:default=false
	// +optional
	Stream bool `json:"stream,omitempty"`

	// SkillRefs is a list of Skill resource names to attach to this agent.
	// +optional
	SkillRefs []string `json:"skillRefs,omitempty"`

	// ToolBindingRefs is a list of ToolBinding resource names to attach to this agent.
	// +optional
	ToolBindingRefs []string `json:"toolBindingRefs,omitempty"`

	// MemoryClassRef is an optional reference to a MemoryClass resource.
	// +optional
	MemoryClassRef string `json:"memoryClassRef,omitempty"`

	// PolicyRef is an optional reference to a Policy resource.
	// +optional
	PolicyRef string `json:"policyRef,omitempty"`

	// SecretRefs is a list of secret references for sensitive configuration.
	// +optional
	SecretRefs []SecretRef `json:"secretRefs,omitempty"`
}

// AgentStatus defines the observed state of an Agent.
type AgentStatus struct {
	// Phase represents the current lifecycle phase of the Agent.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the Agent's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// BoundTools lists the tools currently bound to this agent.
	// +optional
	BoundTools []string `json:"boundTools,omitempty"`

	// EffectivePolicy is the resolved policy name applied to this agent.
	// +optional
	EffectivePolicy string `json:"effectivePolicy,omitempty"`

	// LastReconcileTime is the timestamp of the last successful reconciliation.
	// +optional
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="MODEL",type=string,JSONPath=`.spec.model`
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Agent is the Schema for the agents API.
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AgentList contains a list of Agent resources.
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Agent{}, &AgentList{})
}
