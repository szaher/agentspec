package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	opmetrics "github.com/szaher/agentspec/internal/operator/metrics"
	"github.com/szaher/agentspec/internal/operator/status"
)

// ClusterPolicyReconciler reconciles cluster-scoped ClusterPolicy objects.
type ClusterPolicyReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=clusterpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=clusterpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=clusterpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile handles reconciliation of ClusterPolicy resources.
func (r *ClusterPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var cp v1alpha1.ClusterPolicy
	if err := r.Get(ctx, req.NamespacedName, &cp); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Validate the policy spec (same validation as namespace-scoped Policy).
	if err := validatePolicySpec(&cp.Spec.PolicySpecFields); err != nil {
		status.SetFailed(&cp.Status.Conditions, cp.Generation, "ValidationFailed", err.Error())
		_ = r.Status().Update(ctx, &cp)
		opmetrics.PolicyViolationsTotal.WithLabelValues("", cp.Name).Inc()
		return ctrl.Result{}, nil
	}

	// Count affected agents across all namespaces.
	affectedCount, err := r.countAffectedAgentsClusterWide(ctx, cp.Spec.TargetSelector)
	if err != nil {
		log.Error(err, "failed to count affected agents")
		status.SetFailed(&cp.Status.Conditions, cp.Generation, "AgentMatchError", err.Error())
		_ = r.Status().Update(ctx, &cp)
		return ctrl.Result{}, err
	}

	cp.Status.AffectedAgentCount = affectedCount

	// Set Ready condition.
	status.SetReady(&cp.Status.Conditions, cp.Generation, "Validated", "ClusterPolicy validated and applied")

	if err := r.Status().Update(ctx, &cp); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("clusterpolicy reconciled", "name", cp.Name, "affectedAgents", affectedCount)
	return ctrl.Result{}, nil
}

// countAffectedAgentsClusterWide lists agents across all namespaces that match the target selector.
func (r *ClusterPolicyReconciler) countAffectedAgentsClusterWide(ctx context.Context, selector *metav1.LabelSelector) (int32, error) {
	var agents v1alpha1.AgentList

	if selector != nil {
		sel, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return 0, fmt.Errorf("invalid targetSelector: %w", err)
		}
		if err := r.List(ctx, &agents, client.MatchingLabelsSelector{Selector: sel}); err != nil {
			return 0, err
		}
	} else {
		// No selector means match all agents.
		if err := r.List(ctx, &agents); err != nil {
			return 0, err
		}
	}

	return int32(len(agents.Items)), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ClusterPolicy{}).
		Complete(r)
}
