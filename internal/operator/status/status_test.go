package status

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetReady(t *testing.T) {
	conditions := []metav1.Condition{}

	SetReady(&conditions, 1, "TestReason", "Test message")

	ready := meta.FindStatusCondition(conditions, ConditionReady)
	if ready == nil {
		t.Fatal("Ready condition not found")
	}
	if ready.Status != metav1.ConditionTrue {
		t.Errorf("Expected Ready status to be True, got %s", ready.Status)
	}
	if ready.ObservedGeneration != 1 {
		t.Errorf("Expected ObservedGeneration to be 1, got %d", ready.ObservedGeneration)
	}
	if ready.Reason != "TestReason" {
		t.Errorf("Expected Reason to be TestReason, got %s", ready.Reason)
	}
	if ready.Message != "Test message" {
		t.Errorf("Expected Message to be 'Test message', got %s", ready.Message)
	}

	// Verify Reconciling condition is removed
	reconciling := meta.FindStatusCondition(conditions, ConditionReconciling)
	if reconciling != nil {
		t.Error("Reconciling condition should be removed when Ready is set")
	}
}

func TestSetReconciling(t *testing.T) {
	conditions := []metav1.Condition{}

	SetReconciling(&conditions, 2, "ReconcilingReason", "Reconciling message")

	reconciling := meta.FindStatusCondition(conditions, ConditionReconciling)
	if reconciling == nil {
		t.Fatal("Reconciling condition not found")
	}
	if reconciling.Status != metav1.ConditionTrue {
		t.Errorf("Expected Reconciling status to be True, got %s", reconciling.Status)
	}
	if reconciling.Reason != "ReconcilingReason" {
		t.Errorf("Expected Reason to be ReconcilingReason, got %s", reconciling.Reason)
	}

	// Verify Ready is set to False when Reconciling
	ready := meta.FindStatusCondition(conditions, ConditionReady)
	if ready == nil {
		t.Fatal("Ready condition not found")
	}
	if ready.Status != metav1.ConditionFalse {
		t.Errorf("Expected Ready status to be False when Reconciling, got %s", ready.Status)
	}
	if ready.ObservedGeneration != 2 {
		t.Errorf("Expected ObservedGeneration to be 2, got %d", ready.ObservedGeneration)
	}
}

func TestSetDegraded(t *testing.T) {
	conditions := []metav1.Condition{}

	SetDegraded(&conditions, 3, "DegradedReason", "Degraded message")

	degraded := meta.FindStatusCondition(conditions, ConditionDegraded)
	if degraded == nil {
		t.Fatal("Degraded condition not found")
	}
	if degraded.Status != metav1.ConditionTrue {
		t.Errorf("Expected Degraded status to be True, got %s", degraded.Status)
	}
	if degraded.ObservedGeneration != 3 {
		t.Errorf("Expected ObservedGeneration to be 3, got %d", degraded.ObservedGeneration)
	}
	if degraded.Reason != "DegradedReason" {
		t.Errorf("Expected Reason to be DegradedReason, got %s", degraded.Reason)
	}
	if degraded.Message != "Degraded message" {
		t.Errorf("Expected Message to be 'Degraded message', got %s", degraded.Message)
	}
}

func TestSetFailed(t *testing.T) {
	conditions := []metav1.Condition{}

	SetFailed(&conditions, 4, "FailedReason", "Failed message")

	ready := meta.FindStatusCondition(conditions, ConditionReady)
	if ready == nil {
		t.Fatal("Ready condition not found")
	}
	if ready.Status != metav1.ConditionFalse {
		t.Errorf("Expected Ready status to be False, got %s", ready.Status)
	}
	if ready.ObservedGeneration != 4 {
		t.Errorf("Expected ObservedGeneration to be 4, got %d", ready.ObservedGeneration)
	}
	if ready.Reason != "FailedReason" {
		t.Errorf("Expected Reason to be FailedReason, got %s", ready.Reason)
	}
	if ready.Message != "Failed message" {
		t.Errorf("Expected Message to be 'Failed message', got %s", ready.Message)
	}

	// Verify Reconciling condition is removed
	reconciling := meta.FindStatusCondition(conditions, ConditionReconciling)
	if reconciling != nil {
		t.Error("Reconciling condition should be removed when Failed is set")
	}
}

func TestDerivePhase_Ready(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:   ConditionReady,
			Status: metav1.ConditionTrue,
		},
	}

	phase := DerivePhase(conditions, false)
	if phase != "Ready" {
		t.Errorf("Expected phase to be 'Ready', got %s", phase)
	}
}

func TestDerivePhase_Completed(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:   ConditionReady,
			Status: metav1.ConditionTrue,
		},
	}

	phase := DerivePhase(conditions, true)
	if phase != "Completed" {
		t.Errorf("Expected phase to be 'Completed', got %s", phase)
	}
}

func TestDerivePhase_Reconciling(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:   ConditionReady,
			Status: metav1.ConditionFalse,
		},
		{
			Type:   ConditionReconciling,
			Status: metav1.ConditionTrue,
		},
	}

	phase := DerivePhase(conditions, false)
	if phase != "Reconciling" {
		t.Errorf("Expected phase to be 'Reconciling', got %s", phase)
	}
}

func TestDerivePhase_Failed(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:   ConditionReady,
			Status: metav1.ConditionFalse,
		},
	}

	phase := DerivePhase(conditions, false)
	if phase != "Failed" {
		t.Errorf("Expected phase to be 'Failed', got %s", phase)
	}
}

func TestDerivePhase_Pending(t *testing.T) {
	conditions := []metav1.Condition{}

	phase := DerivePhase(conditions, false)
	if phase != "Pending" {
		t.Errorf("Expected phase to be 'Pending', got %s", phase)
	}
}

func TestConditionTransitions(t *testing.T) {
	conditions := []metav1.Condition{}

	// Start with Reconciling
	SetReconciling(&conditions, 1, "StartReconciling", "Starting reconciliation")
	phase := DerivePhase(conditions, false)
	if phase != "Reconciling" {
		t.Errorf("Expected phase to be 'Reconciling', got %s", phase)
	}

	// Transition to Ready
	SetReady(&conditions, 2, "ReconciliationComplete", "Successfully reconciled")
	phase = DerivePhase(conditions, false)
	if phase != "Ready" {
		t.Errorf("Expected phase to be 'Ready', got %s", phase)
	}

	// Verify Reconciling is removed
	reconciling := meta.FindStatusCondition(conditions, ConditionReconciling)
	if reconciling != nil {
		t.Error("Reconciling condition should be removed after SetReady")
	}

	// Transition to Failed
	SetFailed(&conditions, 3, "ReconciliationFailed", "Failed to reconcile")
	phase = DerivePhase(conditions, false)
	if phase != "Failed" {
		t.Errorf("Expected phase to be 'Failed', got %s", phase)
	}
}

func TestSetDegraded_DoesNotAffectPhase(t *testing.T) {
	conditions := []metav1.Condition{}

	// Set Ready
	SetReady(&conditions, 1, "Ready", "Ready")

	// Add Degraded
	SetDegraded(&conditions, 1, "PartiallyDegraded", "Some features degraded")

	// Phase should still be Ready
	phase := DerivePhase(conditions, false)
	if phase != "Ready" {
		t.Errorf("Expected phase to be 'Ready' even with Degraded condition, got %s", phase)
	}

	// Verify both conditions exist
	ready := meta.FindStatusCondition(conditions, ConditionReady)
	degraded := meta.FindStatusCondition(conditions, ConditionDegraded)
	if ready == nil || degraded == nil {
		t.Error("Both Ready and Degraded conditions should exist")
	}
}
