package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Schedule{}, &ScheduleList{})
}

// TargetRef identifies the target resource for a scheduled task.
type TargetRef struct {
	// Kind is the kind of the target resource.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Agent;Workflow;EvalRun
	Kind string `json:"kind"`

	// Name is the name of the target resource.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// TaskTemplate defines the template for a scheduled task execution.
type TaskTemplate struct {
	// Input is the input message to pass to the target.
	// +optional
	Input string `json:"input,omitempty"`

	// Parameters is a map of additional parameters for the task.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// ScheduleSpec defines the desired state of a Schedule.
type ScheduleSpec struct {
	// Schedule is a cron expression defining when to run the task.
	// +kubebuilder:validation:Required
	Schedule string `json:"schedule"`

	// Timezone is the timezone for interpreting the cron schedule.
	// +kubebuilder:default="UTC"
	// +optional
	Timezone string `json:"timezone,omitempty"`

	// TargetRef identifies the target resource to invoke on schedule.
	// +kubebuilder:validation:Required
	TargetRef TargetRef `json:"targetRef"`

	// TaskTemplate defines the template for task execution.
	// +optional
	TaskTemplate *TaskTemplate `json:"taskTemplate,omitempty"`

	// ConcurrencyPolicy specifies how to treat concurrent executions.
	// +kubebuilder:default="Forbid"
	// +kubebuilder:validation:Enum=Allow;Forbid;Replace
	// +optional
	ConcurrencyPolicy string `json:"concurrencyPolicy,omitempty"`

	// StartingDeadlineSeconds is the deadline in seconds for starting the task
	// if it misses its scheduled time.
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// Suspend indicates whether the schedule is suspended.
	// +kubebuilder:default=false
	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	// SuccessfulTasksHistoryLimit is the number of successful tasks to retain.
	// +kubebuilder:default=3
	// +optional
	SuccessfulTasksHistoryLimit *int32 `json:"successfulTasksHistoryLimit,omitempty"`
}

// ScheduleStatus defines the observed state of a Schedule.
type ScheduleStatus struct {
	// Conditions represent the latest available observations of the schedule's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastScheduleTime is the last time a task was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// NextScheduleTime is the next time a task is scheduled to run.
	// +optional
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`

	// ActiveTaskRefs is a list of references to currently active tasks.
	// +optional
	ActiveTaskRefs []string `json:"activeTaskRefs,omitempty"`

	// MissedScheduleCount is the number of times the schedule was missed.
	// +optional
	MissedScheduleCount int64 `json:"missedScheduleCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="NAME",type=string,JSONPath=`.metadata.name`
// +kubebuilder:printcolumn:name="SCHEDULE",type=string,JSONPath=`.spec.schedule`
// +kubebuilder:printcolumn:name="NEXT-RUN",type=date,JSONPath=`.status.nextScheduleTime`
// +kubebuilder:printcolumn:name="SUSPEND",type=boolean,JSONPath=`.spec.suspend`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Schedule is the Schema for the schedules API. It is namespace-scoped.
type Schedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduleSpec   `json:"spec,omitempty"`
	Status ScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScheduleList contains a list of Schedule resources.
type ScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Schedule `json:"items"`
}
