package operator_test

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

func waitForAgentPhase(t *testing.T, key types.NamespacedName, phase string, timeout time.Duration) *v1alpha1.Agent {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var agent v1alpha1.Agent
		if err := k8sClient.Get(ctx, key, &agent); err == nil {
			if agent.Status.Phase == phase {
				return &agent
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for agent %s to reach phase %s", key.Name, phase)
	return nil
}

func TestAgentCreateReadyLifecycle(t *testing.T) {
	ns := createTestNamespace(t)

	agent := &v1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: ns,
		},
		Spec: v1alpha1.AgentSpec{
			Model: "gpt-4",
		},
	}

	if err := k8sClient.Create(ctx, agent); err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result := waitForAgentPhase(t, types.NamespacedName{Name: "test-agent", Namespace: ns}, "Ready", 10*time.Second)

	// Verify Ready condition.
	readyCond := meta.FindStatusCondition(result.Status.Conditions, "Ready")
	if readyCond == nil {
		t.Fatal("expected Ready condition to be set")
	}
	if readyCond.Status != metav1.ConditionTrue {
		t.Errorf("expected Ready condition True, got %s", readyCond.Status)
	}

	// Verify ObservedGeneration.
	if result.Status.ObservedGeneration != result.Generation {
		t.Errorf("expected ObservedGeneration %d, got %d", result.Generation, result.Status.ObservedGeneration)
	}

	// Verify LastReconcileTime is set.
	if result.Status.LastReconcileTime == nil {
		t.Error("expected LastReconcileTime to be set")
	}
}

func TestAgentUpdateReReconcile(t *testing.T) {
	ns := createTestNamespace(t)

	agent := &v1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "update-agent",
			Namespace: ns,
		},
		Spec: v1alpha1.AgentSpec{
			Model: "gpt-4",
		},
	}

	if err := k8sClient.Create(ctx, agent); err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	waitForAgentPhase(t, types.NamespacedName{Name: "update-agent", Namespace: ns}, "Ready", 10*time.Second)

	// Update model.
	var latest v1alpha1.Agent
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: "update-agent", Namespace: ns}, &latest); err != nil {
		t.Fatalf("failed to get agent: %v", err)
	}
	latest.Spec.Model = "claude-opus-4-20250514"
	if err := k8sClient.Update(ctx, &latest); err != nil {
		t.Fatalf("failed to update agent: %v", err)
	}

	// Wait for re-reconcile.
	result := waitForAgentPhase(t, types.NamespacedName{Name: "update-agent", Namespace: ns}, "Ready", 10*time.Second)
	if result.Status.ObservedGeneration < 2 {
		t.Errorf("expected ObservedGeneration >= 2 after update, got %d", result.Status.ObservedGeneration)
	}
}

func TestAgentDeleteCleanup(t *testing.T) {
	ns := createTestNamespace(t)

	agent := &v1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "delete-agent",
			Namespace: ns,
		},
		Spec: v1alpha1.AgentSpec{
			Model: "gpt-4",
		},
	}

	if err := k8sClient.Create(ctx, agent); err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	waitForAgentPhase(t, types.NamespacedName{Name: "delete-agent", Namespace: ns}, "Ready", 10*time.Second)

	// Delete.
	if err := k8sClient.Delete(ctx, agent); err != nil {
		t.Fatalf("failed to delete agent: %v", err)
	}

	// Wait for deletion.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var check v1alpha1.Agent
		err := k8sClient.Get(ctx, types.NamespacedName{Name: "delete-agent", Namespace: ns}, &check)
		if err != nil {
			break // deleted
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func TestAgentBrokenReference(t *testing.T) {
	ns := createTestNamespace(t)

	agent := &v1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "broken-ref-agent",
			Namespace: ns,
		},
		Spec: v1alpha1.AgentSpec{
			Model:     "gpt-4",
			PromptRef: "nonexistent-configmap",
		},
	}

	if err := k8sClient.Create(ctx, agent); err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	result := waitForAgentPhase(t, types.NamespacedName{Name: "broken-ref-agent", Namespace: ns}, "Failed", 10*time.Second)

	readyCond := meta.FindStatusCondition(result.Status.Conditions, "Ready")
	if readyCond == nil {
		t.Fatal("expected Ready condition to be set")
	}
	if readyCond.Reason != "BrokenReference" {
		t.Errorf("expected reason BrokenReference, got %s", readyCond.Reason)
	}
}

// createTestNamespace creates a unique namespace for test isolation.
func createTestNamespace(t *testing.T) string {
	t.Helper()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}
	if err := k8sClient.Create(ctx, ns); err != nil {
		t.Fatalf("failed to create test namespace: %v", err)
	}
	t.Cleanup(func() {
		_ = k8sClient.Delete(ctx, ns, client.PropagationPolicy(metav1.DeletePropagationForeground))
	})
	return ns.Name
}
