package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TokenUsage tracks LLM token consumption for a task.
type TokenUsage struct {
	// InputTokens is the number of input tokens consumed.
	// +optional
	InputTokens int64 `json:"inputTokens,omitempty"`

	// OutputTokens is the number of output tokens produced.
	// +optional
	OutputTokens int64 `json:"outputTokens,omitempty"`
}

// TaskSpec defines the desired state of a Task.
type TaskSpec struct {
	// AgentRef is the name of the Agent resource to execute this task.
	// +kubebuilder:validation:Required
	AgentRef string `json:"agentRef"`

	// Input is the user input or prompt to send to the agent.
	// +optional
	Input string `json:"input,omitempty"`

	// Parameters is an optional set of key-value parameters for the task.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`

	// Timeout is the maximum duration for the task execution.
	// +kubebuilder:default="5m"
	// +optional
	Timeout string `json:"timeout,omitempty"`
}

// TaskStatus defines the observed state of a Task.
type TaskStatus struct {
	// Phase represents the current lifecycle phase of the Task.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the Task's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Output is the result produced by the agent after task completion.
	// +optional
	Output string `json:"output,omitempty"`

	// StartTime is the timestamp when the task execution started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the timestamp when the task execution completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// TokenUsage tracks the LLM token consumption for this task.
	// +optional
	TokenUsage TokenUsage `json:"tokenUsage,omitempty"`

	// Error contains the error message if the task failed.
	// +optional
	Error string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGENT",type=string,JSONPath=`.spec.agentRef`
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Task is the Schema for the tasks API.
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TaskList contains a list of Task resources.
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
