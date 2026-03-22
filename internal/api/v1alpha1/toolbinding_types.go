// Package v1alpha1 contains API Schema definitions for the agentspec.io v1alpha1 API group.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CommandToolSpec defines a tool backed by a local command execution.
type CommandToolSpec struct {
	// Binary is the path or name of the binary to execute.
	// +kubebuilder:validation:Required
	Binary string `json:"binary"`

	// Args is the list of arguments to pass to the binary.
	// +optional
	Args []string `json:"args,omitempty"`
}

// MCPToolSpec defines a tool backed by a Model Context Protocol server.
type MCPToolSpec struct {
	// ServerRef is the name of the MCP server resource to connect to.
	// +kubebuilder:validation:Required
	ServerRef string `json:"serverRef"`
}

// HTTPToolSpec defines a tool backed by an HTTP endpoint.
type HTTPToolSpec struct {
	// URL is the endpoint URL for the HTTP tool.
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Method is the HTTP method to use (GET, POST, PUT, DELETE, etc.).
	// +kubebuilder:validation:Required
	Method string `json:"method"`
}

// AccessPolicy defines which namespaces may use this tool binding.
type AccessPolicy struct {
	// AllowedNamespaces lists the namespaces permitted to reference this ToolBinding.
	// +optional
	AllowedNamespaces []string `json:"allowedNamespaces,omitempty"`
}

// ToolBindingSpec defines the desired state of ToolBinding.
type ToolBindingSpec struct {
	// ToolType is the type of tool backing this binding.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=command;mcp;http
	ToolType string `json:"toolType"`

	// Name is the human-readable name of the tool.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Description provides a brief explanation of what the tool does.
	// +optional
	Description string `json:"description,omitempty"`

	// Command holds the configuration when ToolType is "command".
	// +optional
	Command *CommandToolSpec `json:"command,omitempty"`

	// MCP holds the configuration when ToolType is "mcp".
	// +optional
	MCP *MCPToolSpec `json:"mcp,omitempty"`

	// HTTP holds the configuration when ToolType is "http".
	// +optional
	HTTP *HTTPToolSpec `json:"http,omitempty"`

	// AccessPolicy defines namespace-level access controls for this tool.
	// +optional
	AccessPolicy *AccessPolicy `json:"accessPolicy,omitempty"`
}

// ToolBindingStatus defines the observed state of ToolBinding.
type ToolBindingStatus struct {
	// Phase is the current phase of the tool binding (Pending, Ready, Error).
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the ToolBinding's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastProbeTime is the last time the tool endpoint was probed for health.
	// +optional
	LastProbeTime *metav1.Time `json:"lastProbeTime,omitempty"`

	// BoundAgentCount is the number of agents currently using this tool binding.
	// +optional
	BoundAgentCount int32 `json:"boundAgentCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="NAME",type=string,JSONPath=`.metadata.name`
// +kubebuilder:printcolumn:name="TYPE",type=string,JSONPath=`.spec.toolType`
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="BOUND-AGENTS",type=integer,JSONPath=`.status.boundAgentCount`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// ToolBinding is the Schema for the toolbindings API.
type ToolBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ToolBindingSpec   `json:"spec,omitempty"`
	Status ToolBindingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ToolBindingList contains a list of ToolBinding.
type ToolBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ToolBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ToolBinding{}, &ToolBindingList{})
}
