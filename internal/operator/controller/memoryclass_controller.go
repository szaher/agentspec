package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	"github.com/szaher/agentspec/internal/operator/status"
)

// validStrategies defines the allowed memory strategies.
var validStrategies = map[string]bool{
	"sliding_window": true,
	"summarization":  true,
	"summary":        true,
	"full":           true,
}

// validBackends defines the allowed memory backends.
var validBackends = map[string]bool{
	"in-memory": true,
	"redis":     true,
}

// MemoryClassReconciler reconciles MemoryClass objects.
type MemoryClassReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=memoryclasses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=memoryclasses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=memoryclasses/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=sessions,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *MemoryClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// MemoryClass is cluster-scoped, so req.NamespacedName has no namespace.
	var mc v1alpha1.MemoryClass
	if err := r.Get(ctx, req.NamespacedName, &mc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Validate spec fields.
	if err := r.validateSpec(&mc); err != nil {
		status.SetFailed(&mc.Status.Conditions, mc.Generation, "ValidationFailed", err.Error())
		_ = r.Status().Update(ctx, &mc)
		r.Recorder.Event(&mc, "Warning", "ValidationFailed", err.Error())
		log.Info("memoryclass validation failed", "name", mc.Name, "error", err.Error())
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Count sessions referencing this MemoryClass across all namespaces.
	var sessions v1alpha1.SessionList
	if err := r.List(ctx, &sessions); err != nil {
		log.Error(err, "failed to list sessions for session count")
		return ctrl.Result{}, err
	}

	var sessionCount int32
	for i := range sessions.Items {
		if sessions.Items[i].Spec.MemoryClassRef == mc.Name {
			sessionCount++
		}
	}
	mc.Status.SessionCount = sessionCount

	// Set Ready condition.
	status.SetReady(&mc.Status.Conditions, mc.Generation, "Validated", "MemoryClass is valid and ready")

	if err := r.Status().Update(ctx, &mc); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&mc, "Normal", "Ready", "MemoryClass is ready")
	log.Info("memoryclass reconciled", "name", mc.Name, "sessionCount", sessionCount)

	return ctrl.Result{}, nil
}

func (r *MemoryClassReconciler) validateSpec(mc *v1alpha1.MemoryClass) error {
	if mc.Spec.Strategy == "" {
		return fmt.Errorf("strategy is required")
	}
	if !validStrategies[mc.Spec.Strategy] {
		return fmt.Errorf("invalid strategy %q: must be one of sliding_window, summarization, summary, full", mc.Spec.Strategy)
	}

	if mc.Spec.Backend != "" && !validBackends[mc.Spec.Backend] {
		return fmt.Errorf("invalid backend %q: must be one of in-memory, redis", mc.Spec.Backend)
	}

	if mc.Spec.MaxMessages < 0 {
		return fmt.Errorf("maxMessages must be non-negative, got %d", mc.Spec.MaxMessages)
	}

	if mc.Spec.TTL != "" {
		if _, err := time.ParseDuration(mc.Spec.TTL); err != nil {
			return fmt.Errorf("invalid ttl %q: %w", mc.Spec.TTL, err)
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MemoryClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.MemoryClass{}).
		Complete(r)
}
