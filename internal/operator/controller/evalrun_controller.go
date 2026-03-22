package controller

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	opmetrics "github.com/szaher/agentspec/internal/operator/metrics"
	"github.com/szaher/agentspec/internal/operator/status"
)

// EvalRunReconciler reconciles EvalRun objects.
type EvalRunReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=evalruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=evalruns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=evalruns/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *EvalRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var evalRun v1alpha1.EvalRun
	if err := r.Get(ctx, req.NamespacedName, &evalRun); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Skip if already in a terminal phase.
	if evalRun.Status.Phase == "Completed" || evalRun.Status.Phase == "Failed" {
		return ctrl.Result{}, nil
	}

	// Validate the referenced Agent exists.
	var agent v1alpha1.Agent
	if err := r.Get(ctx, types.NamespacedName{Name: evalRun.Spec.AgentRef, Namespace: evalRun.Namespace}, &agent); err != nil {
		status.SetFailed(&evalRun.Status.Conditions, evalRun.Generation, "AgentNotFound",
			fmt.Sprintf("agent %q not found: %v", evalRun.Spec.AgentRef, err))
		evalRun.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, &evalRun)
		r.Recorder.Event(&evalRun, corev1.EventTypeWarning, "AgentNotFound",
			fmt.Sprintf("agent %q not found", evalRun.Spec.AgentRef))
		return ctrl.Result{}, nil
	}

	// Transition from Pending to Running.
	if evalRun.Status.Phase == "" || evalRun.Status.Phase == "Pending" {
		now := metav1.Now()
		evalRun.Status.Phase = "Running"
		evalRun.Status.StartTime = &now
		status.SetReconciling(&evalRun.Status.Conditions, evalRun.Generation, "Running", "Evaluation run started")
		if err := r.Status().Update(ctx, &evalRun); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&evalRun, corev1.EventTypeNormal, "Running", "Evaluation run started")
		log.Info("eval run started", "name", evalRun.Name, "agent", evalRun.Spec.AgentRef)
		// Requeue to execute test cases.
		return ctrl.Result{Requeue: true}, nil
	}

	// Execute test cases.
	if evalRun.Status.Phase == "Running" {
		results := r.executeTestCases(ctx, &evalRun)

		// Compute summary.
		summary := r.computeSummary(results)

		now := metav1.Now()
		evalRun.Status.Results = results
		evalRun.Status.Summary = summary
		evalRun.Status.CompletionTime = &now
		evalRun.Status.Phase = "Completed"
		status.SetReady(&evalRun.Status.Conditions, evalRun.Generation, "Completed", "Evaluation run completed")

		if err := r.Status().Update(ctx, &evalRun); err != nil {
			return ctrl.Result{}, err
		}

		// Update metric.
		if summary.Total > 0 {
			scoreFloat := float64(summary.Passed) / float64(summary.Total) * 100
			opmetrics.EvalRunScore.WithLabelValues(evalRun.Namespace, evalRun.Spec.AgentRef).Set(scoreFloat)
		}

		r.Recorder.Event(&evalRun, corev1.EventTypeNormal, "Completed",
			fmt.Sprintf("Evaluation completed: %s", summary.Score))
		log.Info("eval run completed", "name", evalRun.Name, "score", summary.Score)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// executeTestCases runs all test cases with parallelism control.
func (r *EvalRunReconciler) executeTestCases(_ context.Context, evalRun *v1alpha1.EvalRun) []v1alpha1.EvalResult {
	testCases := evalRun.Spec.TestCases
	parallelism := evalRun.Spec.Parallelism
	if parallelism <= 0 {
		parallelism = 1
	}

	results := make([]v1alpha1.EvalResult, len(testCases))
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup

	for i, tc := range testCases {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore.
		go func(idx int, testCase v1alpha1.EvalTestCase) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore.

			results[idx] = r.runTestCase(testCase)
		}(i, tc)
	}
	wg.Wait()

	return results
}

// runTestCase simulates a single test case execution.
func (r *EvalRunReconciler) runTestCase(tc v1alpha1.EvalTestCase) v1alpha1.EvalResult {
	start := time.Now()

	// Simulate agent invocation: set actualOutput to input.
	actualOutput := tc.Input

	latencyMs := time.Since(start).Milliseconds()

	// Match against expected output.
	passed := matchOutput(actualOutput, tc.ExpectedOutput, tc.MatchType)

	return v1alpha1.EvalResult{
		Name:         tc.Name,
		Passed:       passed,
		ActualOutput: actualOutput,
		LatencyMs:    latencyMs,
		TokenUsage: &v1alpha1.EvalTokenUsage{
			PromptTokens:     int64(len(tc.Input)),
			CompletionTokens: int64(len(actualOutput)),
			TotalTokens:      int64(len(tc.Input) + len(actualOutput)),
		},
	}
}

// matchOutput compares actual output against expected output using the specified match type.
func matchOutput(actual, expected, matchType string) bool {
	if expected == "" {
		return true
	}

	switch matchType {
	case "exact":
		return actual == expected
	case "contains":
		return strings.Contains(actual, expected)
	case "regex":
		matched, err := regexp.MatchString(expected, actual)
		if err != nil {
			return false
		}
		return matched
	default:
		// Default to contains.
		return strings.Contains(actual, expected)
	}
}

// computeSummary aggregates results into a summary.
func (r *EvalRunReconciler) computeSummary(results []v1alpha1.EvalResult) *v1alpha1.EvalSummary {
	var passed, failed int32
	var totalTokens int64

	for _, result := range results {
		if result.Passed {
			passed++
		} else {
			failed++
		}
		if result.TokenUsage != nil {
			totalTokens += result.TokenUsage.TotalTokens
		}
	}

	total := int32(len(results))
	var score string
	if total > 0 {
		pct := float64(passed) / float64(total) * 100
		score = fmt.Sprintf("%.0f%%", pct)
	} else {
		score = "0%"
	}

	return &v1alpha1.EvalSummary{
		Total:       total,
		Passed:      passed,
		Failed:      failed,
		Score:       score,
		TotalTokens: totalTokens,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *EvalRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.EvalRun{}).
		Complete(r)
}
