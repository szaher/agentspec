package controller

import (
	"context"
	"time"

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

// TaskReconciler reconciles Task objects.
type TaskReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=tasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=tasks/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var task v1alpha1.Task
	if err := r.Get(ctx, req.NamespacedName, &task); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Skip if already terminal.
	if task.Status.Phase == "Completed" || task.Status.Phase == "Failed" || task.Status.Phase == "TimedOut" {
		return ctrl.Result{}, nil
	}

	// Parse timeout.
	timeout := 5 * time.Minute
	if task.Spec.Timeout != "" {
		parsed, err := time.ParseDuration(task.Spec.Timeout)
		if err != nil {
			log.Error(err, "invalid timeout, using default", "timeout", task.Spec.Timeout)
		} else {
			timeout = parsed
		}
	}

	now := metav1.Now()

	switch task.Status.Phase {
	case "", "Pending":
		// Transition to Running.
		task.Status.Phase = "Running"
		task.Status.StartTime = &now
		status.SetReconciling(&task.Status.Conditions, task.Generation, "Running", "Task execution started")
		if err := r.Status().Update(ctx, &task); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("task started", "task", task.Name)
		// Requeue to simulate completion after a brief delay.
		return ctrl.Result{RequeueAfter: time.Second}, nil

	case "Running":
		// Check timeout.
		if task.Status.StartTime != nil && time.Since(task.Status.StartTime.Time) > timeout {
			task.Status.Phase = "TimedOut"
			task.Status.Error = "task exceeded timeout"
			task.Status.CompletionTime = &now
			status.SetFailed(&task.Status.Conditions, task.Generation, "TimedOut", "Task exceeded timeout")
			if err := r.Status().Update(ctx, &task); err != nil {
				return ctrl.Result{}, err
			}
			opmetrics.TasksTotal.WithLabelValues(task.Namespace, "timeout").Inc()
			log.Info("task timed out", "task", task.Name)
			return ctrl.Result{}, nil
		}

		// Simulate agent invocation: mark completed.
		task.Status.Phase = "Completed"
		task.Status.Output = "task completed"
		task.Status.CompletionTime = &now
		status.SetReady(&task.Status.Conditions, task.Generation, "Completed", "Task completed successfully")
		if err := r.Status().Update(ctx, &task); err != nil {
			return ctrl.Result{}, err
		}
		opmetrics.TasksTotal.WithLabelValues(task.Namespace, "success").Inc()
		log.Info("task completed", "task", task.Name)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Task{}).
		Complete(r)
}
