package controller

import (
	"context"
	"fmt"
	"strconv"

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

// PolicyReconciler reconciles namespace-scoped Policy objects.
type PolicyReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=policies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=policies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=policies/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile handles reconciliation of Policy resources.
func (r *PolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var policy v1alpha1.Policy
	if err := r.Get(ctx, req.NamespacedName, &policy); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Validate the policy spec.
	if err := validatePolicySpec(&policy.Spec.PolicySpecFields); err != nil {
		status.SetFailed(&policy.Status.Conditions, policy.Generation, "ValidationFailed", err.Error())
		_ = r.Status().Update(ctx, &policy)
		opmetrics.PolicyViolationsTotal.WithLabelValues(policy.Namespace, policy.Name).Inc()
		return ctrl.Result{}, nil
	}

	// Count affected agents by matching the target selector in the same namespace.
	affectedCount, err := r.countAffectedAgents(ctx, policy.Namespace, policy.Spec.TargetSelector)
	if err != nil {
		log.Error(err, "failed to count affected agents")
		status.SetFailed(&policy.Status.Conditions, policy.Generation, "AgentMatchError", err.Error())
		_ = r.Status().Update(ctx, &policy)
		return ctrl.Result{}, err
	}

	policy.Status.AffectedAgentCount = affectedCount

	// Set Ready condition.
	status.SetReady(&policy.Status.Conditions, policy.Generation, "Validated", "Policy validated and applied")

	if err := r.Status().Update(ctx, &policy); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("policy reconciled", "name", policy.Name, "affectedAgents", affectedCount)
	return ctrl.Result{}, nil
}

// countAffectedAgents lists agents in the given namespace that match the target selector.
func (r *PolicyReconciler) countAffectedAgents(ctx context.Context, namespace string, selector *metav1.LabelSelector) (int32, error) {
	var agents v1alpha1.AgentList
	opts := []client.ListOption{client.InNamespace(namespace)}

	if selector != nil {
		sel, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return 0, fmt.Errorf("invalid targetSelector: %w", err)
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: sel})
	}

	if err := r.List(ctx, &agents, opts...); err != nil {
		return 0, err
	}
	return int32(len(agents.Items)), nil
}

// validatePolicySpec validates the fields of a PolicySpecFields.
func validatePolicySpec(spec *v1alpha1.PolicySpecFields) error {
	if spec.CostBudget != nil {
		if spec.CostBudget.MaxDailyCost == "" {
			return fmt.Errorf("costBudget.maxDailyCost is required when costBudget is set")
		}
		val, err := strconv.ParseFloat(spec.CostBudget.MaxDailyCost, 64)
		if err != nil {
			return fmt.Errorf("costBudget.maxDailyCost must be a valid number: %w", err)
		}
		if val <= 0 {
			return fmt.Errorf("costBudget.maxDailyCost must be positive")
		}
	}

	if spec.RateLimits != nil {
		if spec.RateLimits.RequestsPerMinute < 0 {
			return fmt.Errorf("rateLimits.requestsPerMinute must be non-negative")
		}
		if spec.RateLimits.TokensPerMinute < 0 {
			return fmt.Errorf("rateLimits.tokensPerMinute must be non-negative")
		}
	}

	for i, f := range spec.ContentFilters {
		if f.Type == "" {
			return fmt.Errorf("contentFilters[%d].type is required", i)
		}
		if f.Pattern == "" {
			return fmt.Errorf("contentFilters[%d].pattern is required", i)
		}
	}

	// Validate that allowedModels and deniedModels don't overlap.
	if len(spec.AllowedModels) > 0 && len(spec.DeniedModels) > 0 {
		allowed := make(map[string]bool, len(spec.AllowedModels))
		for _, m := range spec.AllowedModels {
			allowed[m] = true
		}
		for _, m := range spec.DeniedModels {
			if allowed[m] {
				return fmt.Errorf("model %q appears in both allowedModels and deniedModels", m)
			}
		}
	}

	// Validate that tool restrictions don't have overlapping allowed/denied.
	if spec.ToolRestrictions != nil {
		if len(spec.ToolRestrictions.AllowedTools) > 0 && len(spec.ToolRestrictions.DeniedTools) > 0 {
			allowed := make(map[string]bool, len(spec.ToolRestrictions.AllowedTools))
			for _, t := range spec.ToolRestrictions.AllowedTools {
				allowed[t] = true
			}
			for _, t := range spec.ToolRestrictions.DeniedTools {
				if allowed[t] {
					return fmt.Errorf("tool %q appears in both allowedTools and deniedTools", t)
				}
			}
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Policy{}).
		Complete(r)
}
