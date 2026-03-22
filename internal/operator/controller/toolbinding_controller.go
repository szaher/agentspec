package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	"github.com/szaher/agentspec/internal/operator/status"
)

// Valid tool types.
var validToolTypes = map[string]bool{
	"command": true,
	"mcp":     true,
	"http":    true,
}

// ToolBindingReconciler reconciles ToolBinding objects.
type ToolBindingReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=toolbindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=toolbindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=toolbindings/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *ToolBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var tb v1alpha1.ToolBinding
	if err := r.Get(ctx, req.NamespacedName, &tb); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	previousPhase := tb.Status.Phase

	// Validate tool type.
	if !validToolTypes[tb.Spec.ToolType] {
		msg := fmt.Sprintf("invalid tool type %q: must be one of command, mcp, http", tb.Spec.ToolType)
		r.setUnavailable(ctx, &tb, "InvalidToolType", msg, previousPhase)
		return ctrl.Result{}, nil
	}

	// Validate corresponding spec exists for the declared tool type.
	if err := r.validateToolSpec(&tb); err != nil {
		r.setUnavailable(ctx, &tb, "InvalidToolSpec", err.Error(), previousPhase)
		return ctrl.Result{}, nil
	}

	// Count bound agents.
	boundCount, err := r.countBoundAgents(ctx, &tb)
	if err != nil {
		log.Error(err, "failed to count bound agents")
		// Mark degraded but continue.
		now := metav1.Now()
		tb.Status.Phase = "Degraded"
		tb.Status.LastProbeTime = &now
		tb.Status.BoundAgentCount = 0
		status.SetDegraded(&tb.Status.Conditions, tb.Generation, "AgentCountError", err.Error())
		status.SetFailed(&tb.Status.Conditions, tb.Generation, "Degraded", "Unable to count bound agents")
		if updateErr := r.Status().Update(ctx, &tb); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		r.emitPhaseTransitionEvent(&tb, previousPhase)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Set Available phase.
	now := metav1.Now()
	tb.Status.Phase = "Available"
	tb.Status.LastProbeTime = &now
	tb.Status.BoundAgentCount = boundCount

	// Clear any degraded condition and set ready.
	status.SetReady(&tb.Status.Conditions, tb.Generation, "Available", "Tool binding configuration is valid")

	if err := r.Status().Update(ctx, &tb); err != nil {
		return ctrl.Result{}, err
	}

	r.emitPhaseTransitionEvent(&tb, previousPhase)
	log.Info("toolbinding reconciled", "name", tb.Name, "phase", tb.Status.Phase, "boundAgents", boundCount)

	return ctrl.Result{}, nil
}

// validateToolSpec checks that the corresponding spec block exists for the declared tool type.
func (r *ToolBindingReconciler) validateToolSpec(tb *v1alpha1.ToolBinding) error {
	switch tb.Spec.ToolType {
	case "command":
		if tb.Spec.Command == nil {
			return fmt.Errorf("toolType is %q but command spec is missing", tb.Spec.ToolType)
		}
		if tb.Spec.Command.Binary == "" {
			return fmt.Errorf("command spec requires a non-empty binary field")
		}
	case "mcp":
		if tb.Spec.MCP == nil {
			return fmt.Errorf("toolType is %q but mcp spec is missing", tb.Spec.ToolType)
		}
		if tb.Spec.MCP.ServerRef == "" {
			return fmt.Errorf("mcp spec requires a non-empty serverRef field")
		}
	case "http":
		if tb.Spec.HTTP == nil {
			return fmt.Errorf("toolType is %q but http spec is missing", tb.Spec.ToolType)
		}
		if tb.Spec.HTTP.URL == "" {
			return fmt.Errorf("http spec requires a non-empty url field")
		}
		if tb.Spec.HTTP.Method == "" {
			return fmt.Errorf("http spec requires a non-empty method field")
		}
	default:
		return fmt.Errorf("unsupported tool type %q", tb.Spec.ToolType)
	}
	return nil
}

// countBoundAgents lists all Agents in the same namespace and counts those referencing this ToolBinding.
func (r *ToolBindingReconciler) countBoundAgents(ctx context.Context, tb *v1alpha1.ToolBinding) (int32, error) {
	var agents v1alpha1.AgentList
	if err := r.List(ctx, &agents, client.InNamespace(tb.Namespace)); err != nil {
		return 0, fmt.Errorf("failed to list agents: %w", err)
	}

	var count int32
	for i := range agents.Items {
		agent := &agents.Items[i]
		for _, ref := range agent.Spec.ToolBindingRefs {
			if ref == tb.Name {
				count++
				break
			}
		}
		// Also check skillRefs since agent controller treats them as tool bindings.
		for _, ref := range agent.Spec.SkillRefs {
			if ref == tb.Name {
				count++
				break
			}
		}
	}

	return count, nil
}

// setUnavailable sets the ToolBinding to Unavailable phase and updates status.
func (r *ToolBindingReconciler) setUnavailable(ctx context.Context, tb *v1alpha1.ToolBinding, reason, message, previousPhase string) {
	now := metav1.Now()
	tb.Status.Phase = "Unavailable"
	tb.Status.LastProbeTime = &now
	status.SetFailed(&tb.Status.Conditions, tb.Generation, reason, message)
	_ = r.Status().Update(ctx, tb)
	r.emitPhaseTransitionEvent(tb, previousPhase)
}

// emitPhaseTransitionEvent emits a Kubernetes event when the phase changes.
func (r *ToolBindingReconciler) emitPhaseTransitionEvent(tb *v1alpha1.ToolBinding, previousPhase string) {
	if tb.Status.Phase == previousPhase {
		return
	}
	switch tb.Status.Phase {
	case "Available":
		r.Recorder.Event(tb, corev1.EventTypeNormal, "Available", "ToolBinding is available")
	case "Unavailable":
		r.Recorder.Event(tb, corev1.EventTypeWarning, "Unavailable", "ToolBinding is unavailable: invalid configuration")
	case "Degraded":
		r.Recorder.Event(tb, corev1.EventTypeWarning, "Degraded", "ToolBinding is degraded")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ToolBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ToolBinding{}).
		Complete(r)
}
