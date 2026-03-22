package status

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Standard condition types for all AgentSpec resources.
const (
	ConditionReady       = "Ready"
	ConditionReconciling = "Reconciling"
	ConditionDegraded    = "Degraded"
)

// SetReady sets the Ready condition to True.
func SetReady(conditions *[]metav1.Condition, generation int64, reason, message string) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               ConditionReady,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
	meta.RemoveStatusCondition(conditions, ConditionReconciling)
}

// SetReconciling sets the Reconciling condition to True and Ready to False.
func SetReconciling(conditions *[]metav1.Condition, generation int64, reason, message string) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               ConditionReconciling,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               ConditionReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
}

// SetDegraded sets the Degraded condition to True.
func SetDegraded(conditions *[]metav1.Condition, generation int64, reason, message string) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               ConditionDegraded,
		Status:             metav1.ConditionTrue,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
}

// SetFailed sets the Ready condition to False with a failure reason.
func SetFailed(conditions *[]metav1.Condition, generation int64, reason, message string) {
	meta.SetStatusCondition(conditions, metav1.Condition{
		Type:               ConditionReady,
		Status:             metav1.ConditionFalse,
		ObservedGeneration: generation,
		Reason:             reason,
		Message:            message,
	})
	meta.RemoveStatusCondition(conditions, ConditionReconciling)
}

// DerivePhase computes a convenience phase string from conditions.
func DerivePhase(conditions []metav1.Condition, completable bool) string {
	ready := meta.FindStatusCondition(conditions, ConditionReady)
	reconciling := meta.FindStatusCondition(conditions, ConditionReconciling)

	if ready != nil && ready.Status == metav1.ConditionTrue {
		if completable {
			return "Completed"
		}
		return "Ready"
	}

	if reconciling != nil && reconciling.Status == metav1.ConditionTrue {
		return "Reconciling"
	}

	if ready != nil && ready.Status == metav1.ConditionFalse {
		return "Failed"
	}

	return "Pending"
}
