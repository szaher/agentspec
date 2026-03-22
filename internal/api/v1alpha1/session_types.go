package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SessionSpec defines the desired state of a Session.
type SessionSpec struct {
	// AgentRef is the name of the Agent resource this session is associated with.
	// +kubebuilder:validation:Required
	AgentRef string `json:"agentRef"`

	// MemoryClassRef is an optional reference to a MemoryClass resource for this session.
	// +optional
	MemoryClassRef string `json:"memoryClassRef,omitempty"`

	// Metadata is an optional set of key-value pairs for session metadata.
	// +optional
	Metadata map[string]string `json:"metadata,omitempty"`
}

// SessionStatus defines the observed state of a Session.
type SessionStatus struct {
	// Phase represents the current lifecycle phase of the Session.
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the Session's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// MessageCount is the number of messages in this session.
	// +optional
	MessageCount int32 `json:"messageCount,omitempty"`

	// CreatedAt is the timestamp when the session was created.
	// +optional
	CreatedAt *metav1.Time `json:"createdAt,omitempty"`

	// LastActivityTime is the timestamp of the last activity in this session.
	// +optional
	LastActivityTime *metav1.Time `json:"lastActivityTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGENT",type=string,JSONPath=`.spec.agentRef`
// +kubebuilder:printcolumn:name="PHASE",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="MESSAGES",type=integer,JSONPath=`.status.messageCount`
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Session is the Schema for the sessions API.
type Session struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SessionSpec   `json:"spec,omitempty"`
	Status SessionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SessionList contains a list of Session resources.
type SessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Session `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Session{}, &SessionList{})
}
