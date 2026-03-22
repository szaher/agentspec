// Package v1alpha1 contains API Schema definitions for the agentspec.io v1alpha1 API group.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowStep defines a single step in a workflow.
type WorkflowStep struct {
	// Name is the unique identifier for this step.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// AgentRef is the name of the AgentSpec resource this step invokes.
	// +kubebuilder:validation:Required
	AgentRef string `json:"agentRef"`

	// Input is the input data to pass to the agent.
	// +optional
	Input string `json:"input,omitempty"`

	// DependsOn lists step names that must complete before this step runs.
	// +optional
	DependsOn []string `json:"dependsOn,omitempty"`

	// Timeout is the maximum duration for this step.
	// +kubebuilder:default="5m"
	// +optional
	Timeout string `json:"timeout,omitempty"`
}

// WorkflowSpec defines the desired state of Workflow.
type WorkflowSpec struct {
	// Steps is the ordered list of workflow steps to execute.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Steps []WorkflowStep `json:"steps"`

	// FailFast controls whether the workflow stops on the first step failure.
	// +kubebuilder:default=true
	// +optional
	FailFast *bool `json:"failFast,omitempty"`

	// Finally lists steps that always run after the main steps complete, regardless of failure.
	// +optional
	Finally []WorkflowStep `json:"finally,omitempty"`
}

// StepStatus describes the observed state of a single workflow step.
type StepStatus struct {
	// Name is the name of the step.
	Name string `json:"name,omitempty"`

	// Phase is the current phase of the step (Pending, Running, Succeeded, Failed).
	Phase string `json:"phase,omitempty"`

	// TaskRef is a reference to the underlying task resource created for this step.
	TaskRef string `json:"taskRef,omitempty"`

	// StartTime is the time the step started executing.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the step finished executing.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Output is the output produced by the step.
	// +optional
	Output string `json:"output,omitempty"`
}

// WorkflowStatus defines the observed state of Workflow.
type WorkflowStatus struct {
	// Phase is the current phase of the workflow (Pending, Running, Succeeded, Failed).
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the workflow's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// StepStatuses contains the status of each step in the workflow.
	// +optional
	StepStatuses []StepStatus `json:"stepStatuses,omitempty"`

	// StartTime is the time the workflow started executing.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time the workflow finished executing.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="NAME",type=string,JSONPath=`.metadata.name`
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="STEPS",type=integer,JSONPath=`.spec.steps[*]`,description="Number of steps in the workflow"
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Workflow is the Schema for the workflows API.
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkflowList contains a list of Workflow.
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}
