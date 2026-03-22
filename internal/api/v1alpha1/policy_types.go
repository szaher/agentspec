// Package v1alpha1 contains API Schema definitions for the agentspec.io v1alpha1 API group.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Policy{}, &PolicyList{})
}

// CostBudget defines cost budget constraints for an agent.
type CostBudget struct {
	// MaxDailyCost is the maximum allowed daily cost (e.g. "10.00").
	// +kubebuilder:validation:Required
	MaxDailyCost string `json:"maxDailyCost"`

	// Currency is the currency code for the cost budget.
	// +kubebuilder:default="USD"
	// +optional
	Currency string `json:"currency,omitempty"`
}

// RateLimits defines rate limiting constraints.
type RateLimits struct {
	// RequestsPerMinute is the maximum number of requests per minute.
	// +optional
	RequestsPerMinute int32 `json:"requestsPerMinute,omitempty"`

	// TokensPerMinute is the maximum number of tokens per minute.
	// +optional
	TokensPerMinute int64 `json:"tokensPerMinute,omitempty"`
}

// ContentFilter defines a content filtering rule.
type ContentFilter struct {
	// Type specifies whether the filter applies to input, output, or both.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=input;output;both
	Type string `json:"type"`

	// Pattern is the regex or matching pattern for the filter.
	// +kubebuilder:validation:Required
	Pattern string `json:"pattern"`
}

// ToolRestrictions defines which tools are allowed or denied.
type ToolRestrictions struct {
	// AllowedTools is a list of tool names that are explicitly allowed.
	// +optional
	AllowedTools []string `json:"allowedTools,omitempty"`

	// DeniedTools is a list of tool names that are explicitly denied.
	// +optional
	DeniedTools []string `json:"deniedTools,omitempty"`
}

// PolicySpecFields contains the shared spec fields used by both Policy and ClusterPolicy.
type PolicySpecFields struct {
	// CostBudget defines cost budget constraints.
	// +optional
	CostBudget *CostBudget `json:"costBudget,omitempty"`

	// AllowedModels is a list of model identifiers that are allowed.
	// +optional
	AllowedModels []string `json:"allowedModels,omitempty"`

	// DeniedModels is a list of model identifiers that are denied.
	// +optional
	DeniedModels []string `json:"deniedModels,omitempty"`

	// RateLimits defines rate limiting constraints.
	// +optional
	RateLimits *RateLimits `json:"rateLimits,omitempty"`

	// ContentFilters is a list of content filtering rules.
	// +optional
	ContentFilters []ContentFilter `json:"contentFilters,omitempty"`

	// ToolRestrictions defines which tools are allowed or denied.
	// +optional
	ToolRestrictions *ToolRestrictions `json:"toolRestrictions,omitempty"`

	// TargetSelector selects the agents this policy applies to.
	// +optional
	TargetSelector *metav1.LabelSelector `json:"targetSelector,omitempty"`
}

// PolicyStatusFields contains the shared status fields used by both Policy and ClusterPolicy.
type PolicyStatusFields struct {
	// Conditions represent the latest available observations of the policy's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// AffectedAgentCount is the number of agents affected by this policy.
	// +optional
	AffectedAgentCount int32 `json:"affectedAgentCount,omitempty"`

	// ViolationCount is the total number of policy violations observed.
	// +optional
	ViolationCount int64 `json:"violationCount,omitempty"`
}

// PolicySpec defines the desired state of a Policy.
type PolicySpec struct {
	PolicySpecFields `json:",inline"`
}

// PolicyStatus defines the observed state of a Policy.
type PolicyStatus struct {
	PolicyStatusFields `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="NAME",type=string,JSONPath=`.metadata.name`
// +kubebuilder:printcolumn:name="AFFECTED-AGENTS",type=integer,JSONPath=`.status.affectedAgentCount`
// +kubebuilder:printcolumn:name="VIOLATIONS",type=integer,JSONPath=`.status.violationCount`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Policy is the Schema for the policies API. It is namespace-scoped.
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicySpec   `json:"spec,omitempty"`
	Status PolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PolicyList contains a list of Policy resources.
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Policy `json:"items"`
}
