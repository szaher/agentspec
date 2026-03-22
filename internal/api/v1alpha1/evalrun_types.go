package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EvalTokenUsage tracks LLM token consumption for a single evaluation.
type EvalTokenUsage struct {
	// PromptTokens is the number of tokens in the prompt.
	PromptTokens int64 `json:"promptTokens,omitempty"`
	// CompletionTokens is the number of tokens in the completion.
	CompletionTokens int64 `json:"completionTokens,omitempty"`
	// TotalTokens is the sum of prompt and completion tokens.
	TotalTokens int64 `json:"totalTokens,omitempty"`
}

// EvalTestCase defines a single test case for agent evaluation.
type EvalTestCase struct {
	// Name is the unique name of this test case.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Input is the prompt or message sent to the agent.
	// +kubebuilder:validation:Required
	Input string `json:"input"`

	// ExpectedOutput is the expected agent response to compare against.
	// +optional
	ExpectedOutput string `json:"expectedOutput,omitempty"`

	// MatchType determines how ActualOutput is compared to ExpectedOutput.
	// +kubebuilder:default="contains"
	// +kubebuilder:validation:Enum=exact;contains;regex
	// +optional
	MatchType string `json:"matchType,omitempty"`

	// Timeout is the maximum duration to wait for the agent response.
	// +kubebuilder:default="30s"
	// +optional
	Timeout string `json:"timeout,omitempty"`
}

// EvalResult captures the outcome of a single test case execution.
type EvalResult struct {
	// Name is the name of the test case this result corresponds to.
	Name string `json:"name,omitempty"`

	// Passed indicates whether the test case passed.
	Passed bool `json:"passed,omitempty"`

	// ActualOutput is the agent's actual response.
	ActualOutput string `json:"actualOutput,omitempty"`

	// LatencyMs is the response latency in milliseconds.
	LatencyMs int64 `json:"latencyMs,omitempty"`

	// TokenUsage tracks the token consumption for this test case.
	// +optional
	TokenUsage *EvalTokenUsage `json:"tokenUsage,omitempty"`
}

// EvalSummary provides aggregate statistics for an evaluation run.
type EvalSummary struct {
	// Total is the total number of test cases executed.
	Total int32 `json:"total"`

	// Passed is the number of test cases that passed.
	Passed int32 `json:"passed"`

	// Failed is the number of test cases that failed.
	Failed int32 `json:"failed"`

	// Score is the pass rate as a human-readable string (e.g. "80%").
	Score string `json:"score,omitempty"`

	// TotalTokens is the aggregate token usage across all test cases.
	TotalTokens int64 `json:"totalTokens,omitempty"`
}

// EvalRunSpec defines the desired state of an EvalRun.
type EvalRunSpec struct {
	// AgentRef is the name of the Agent resource to evaluate.
	// +kubebuilder:validation:Required
	AgentRef string `json:"agentRef"`

	// TestCases is the list of test cases to execute.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	TestCases []EvalTestCase `json:"testCases"`

	// Parallelism is the number of test cases to run concurrently.
	// +kubebuilder:default=1
	// +optional
	Parallelism int32 `json:"parallelism,omitempty"`
}

// EvalRunStatus defines the observed state of an EvalRun.
type EvalRunStatus struct {
	// Phase is the current phase of the evaluation run.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the EvalRun's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Results contains the outcome of each test case.
	// +optional
	Results []EvalResult `json:"results,omitempty"`

	// Summary provides aggregate statistics for the evaluation run.
	// +optional
	Summary *EvalSummary `json:"summary,omitempty"`

	// StartTime is the timestamp when the evaluation run started.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the timestamp when the evaluation run completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGENT",type=string,JSONPath=`.spec.agentRef`
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="SCORE",type=string,JSONPath=`.status.summary.score`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// EvalRun is the Schema for the evalruns API.
// It represents a single evaluation run of an agent against a set of test cases.
type EvalRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EvalRunSpec   `json:"spec,omitempty"`
	Status EvalRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EvalRunList contains a list of EvalRun resources.
type EvalRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EvalRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EvalRun{}, &EvalRunList{})
}
