package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	opmetrics "github.com/szaher/agentspec/internal/operator/metrics"
	"github.com/szaher/agentspec/internal/operator/status"
)

// WorkflowReconciler reconciles Workflow objects.
type WorkflowReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=workflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=workflows/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var wf v1alpha1.Workflow
	if err := r.Get(ctx, req.NamespacedName, &wf); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Skip terminal states.
	if wf.Status.Phase == "Completed" || wf.Status.Phase == "Failed" {
		return ctrl.Result{}, nil
	}

	// Validate DAG on first reconcile.
	if wf.Status.Phase == "" || wf.Status.Phase == "Pending" {
		steps := stepsToDAG(wf.Spec.Steps)
		if _, err := topoSort(steps); err != nil {
			wf.Status.Phase = "Failed"
			status.SetFailed(&wf.Status.Conditions, wf.Generation, "CycleDetected", err.Error())
			_ = r.Status().Update(ctx, &wf)
			r.Recorder.Event(&wf, "Warning", "CycleDetected", err.Error())
			return ctrl.Result{}, nil
		}

		// Initialize step statuses and transition to Running.
		now := metav1.Now()
		wf.Status.Phase = "Running"
		wf.Status.StartTime = &now
		wf.Status.StepStatuses = make([]v1alpha1.StepStatus, len(wf.Spec.Steps))
		for i, step := range wf.Spec.Steps {
			wf.Status.StepStatuses[i] = v1alpha1.StepStatus{
				Name:  step.Name,
				Phase: "Pending",
			}
		}
		status.SetReconciling(&wf.Status.Conditions, wf.Generation, "Running", "Workflow execution started")
		if err := r.Status().Update(ctx, &wf); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("workflow started", "workflow", wf.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	// Running: create tasks for ready steps, check completion.
	stepPhase := make(map[string]string, len(wf.Status.StepStatuses))
	for _, ss := range wf.Status.StepStatuses {
		stepPhase[ss.Name] = ss.Phase
	}

	failFast := wf.Spec.FailFast == nil || *wf.Spec.FailFast
	anyFailed := false
	allDone := true
	requeue := false

	for i, step := range wf.Spec.Steps {
		ss := &wf.Status.StepStatuses[i]

		if ss.Phase == "Succeeded" || ss.Phase == "Failed" {
			if ss.Phase == "Failed" {
				anyFailed = true
			}
			continue
		}

		// If fail-fast and a step already failed, skip remaining.
		if failFast && anyFailed {
			ss.Phase = "Failed"
			continue
		}

		// Check if all dependencies are met.
		depsReady := true
		for _, dep := range step.DependsOn {
			p := stepPhase[dep]
			if p != "Succeeded" {
				depsReady = false
				if p == "Failed" {
					ss.Phase = "Failed"
				}
				break
			}
		}
		if !depsReady {
			allDone = false
			continue
		}

		// Create or check task.
		taskName := fmt.Sprintf("%s-%s", wf.Name, step.Name)
		if ss.TaskRef == "" {
			// Create the Task CR.
			task := &v1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      taskName,
					Namespace: wf.Namespace,
				},
				Spec: v1alpha1.TaskSpec{
					AgentRef: step.AgentRef,
					Input:    step.Input,
					Timeout:  step.Timeout,
				},
			}
			if err := controllerutil.SetControllerReference(&wf, task, r.Scheme); err != nil {
				log.Error(err, "failed to set owner reference", "task", taskName)
			}
			if err := r.Create(ctx, task); err != nil {
				if !errors.IsAlreadyExists(err) {
					return ctrl.Result{}, err
				}
			}
			now := metav1.Now()
			ss.TaskRef = taskName
			ss.Phase = "Running"
			ss.StartTime = &now
			allDone = false
			requeue = true
		} else {
			// Check existing task status.
			var task v1alpha1.Task
			if err := r.Get(ctx, client.ObjectKey{Name: ss.TaskRef, Namespace: wf.Namespace}, &task); err != nil {
				allDone = false
				requeue = true
				continue
			}
			switch task.Status.Phase {
			case "Completed":
				now := metav1.Now()
				ss.Phase = "Succeeded"
				ss.CompletionTime = &now
				ss.Output = task.Status.Output
			case "Failed", "TimedOut":
				now := metav1.Now()
				ss.Phase = "Failed"
				ss.CompletionTime = &now
				anyFailed = true
			default:
				allDone = false
				requeue = true
			}
		}
	}

	// Handle finally steps.
	if allDone || (failFast && anyFailed) {
		r.runFinallySteps(ctx, &wf)
		// Check finally step task completion.
		for i := range wf.Status.StepStatuses {
			ss := &wf.Status.StepStatuses[i]
			if ss.Phase == "Running" && ss.TaskRef != "" {
				var task v1alpha1.Task
				if err := r.Get(ctx, client.ObjectKey{Name: ss.TaskRef, Namespace: wf.Namespace}, &task); err == nil {
					switch task.Status.Phase {
					case "Completed":
						now := metav1.Now()
						ss.Phase = "Succeeded"
						ss.CompletionTime = &now
						ss.Output = task.Status.Output
					case "Failed", "TimedOut":
						now := metav1.Now()
						ss.Phase = "Failed"
						ss.CompletionTime = &now
					default:
						requeue = true
					}
				} else {
					requeue = true
				}
			}
		}
	}

	// Determine workflow outcome.
	allDoneNow := r.allStepsDone(&wf)
	if allDoneNow {
		now := metav1.Now()
		wf.Status.CompletionTime = &now
		if anyFailed {
			wf.Status.Phase = "Failed"
			status.SetFailed(&wf.Status.Conditions, wf.Generation, "StepFailed", "One or more steps failed")
		} else {
			wf.Status.Phase = "Completed"
			status.SetReady(&wf.Status.Conditions, wf.Generation, "Completed", "All steps completed successfully")
		}
		// Record duration metric.
		if wf.Status.StartTime != nil {
			duration := now.Time.Sub(wf.Status.StartTime.Time).Seconds()
			opmetrics.WorkflowDurationSeconds.WithLabelValues(wf.Namespace, wf.Name).Observe(duration)
		}
		log.Info("workflow finished", "workflow", wf.Name, "phase", wf.Status.Phase)
		requeue = false
	}

	// Capture the computed status before re-fetching.
	newStepStatuses := wf.Status.StepStatuses
	newPhase := wf.Status.Phase
	newConditions := wf.Status.Conditions
	newCompletionTime := wf.Status.CompletionTime
	savedGeneration := wf.Generation
	savedStartTime := wf.Status.StartTime

	// Re-fetch to get the latest resourceVersion before updating status.
	// Creating/checking Tasks via .Owns() may have caused re-queues that
	// modified the object.
	if err := r.Get(ctx, req.NamespacedName, &wf); err != nil {
		return ctrl.Result{}, err
	}

	// Apply computed status to the fresh copy.
	wf.Status.StepStatuses = newStepStatuses
	wf.Status.Phase = newPhase
	wf.Status.Conditions = newConditions
	wf.Status.CompletionTime = newCompletionTime
	if wf.Status.StartTime == nil {
		wf.Status.StartTime = savedStartTime
	}
	_ = savedGeneration // used in conditions already

	if err := r.Status().Update(ctx, &wf); err != nil {
		return ctrl.Result{}, err
	}

	if requeue {
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

// runFinallySteps ensures finally steps are created and tracked.
func (r *WorkflowReconciler) runFinallySteps(ctx context.Context, wf *v1alpha1.Workflow) {
	log := log.FromContext(ctx)

	for _, step := range wf.Spec.Finally {
		// Check if already tracked.
		found := false
		for _, ss := range wf.Status.StepStatuses {
			if ss.Name == step.Name {
				found = true
				break
			}
		}
		if found {
			continue
		}

		taskName := fmt.Sprintf("%s-finally-%s", wf.Name, step.Name)
		task := &v1alpha1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskName,
				Namespace: wf.Namespace,
			},
			Spec: v1alpha1.TaskSpec{
				AgentRef: step.AgentRef,
				Input:    step.Input,
				Timeout:  step.Timeout,
			},
		}
		if err := controllerutil.SetControllerReference(wf, task, r.Scheme); err != nil {
			log.Error(err, "failed to set owner reference for finally task", "task", taskName)
		}
		if err := r.Create(ctx, task); err != nil && !errors.IsAlreadyExists(err) {
			log.Error(err, "failed to create finally task", "task", taskName)
			continue
		}

		now := metav1.Now()
		wf.Status.StepStatuses = append(wf.Status.StepStatuses, v1alpha1.StepStatus{
			Name:      step.Name,
			Phase:     "Running",
			TaskRef:   taskName,
			StartTime: &now,
		})
	}
}

// allStepsDone returns true if every tracked step is in a terminal state.
func (r *WorkflowReconciler) allStepsDone(wf *v1alpha1.Workflow) bool {
	for _, ss := range wf.Status.StepStatuses {
		if ss.Phase != "Succeeded" && ss.Phase != "Failed" {
			return false
		}
	}
	return len(wf.Status.StepStatuses) > 0
}

// stepsToDAG converts WorkflowSteps to DAGSteps for validation.
func stepsToDAG(steps []v1alpha1.WorkflowStep) []DAGStep {
	dag := make([]DAGStep, len(steps))
	for i, s := range steps {
		dag[i] = DAGStep{
			Name:      s.Name,
			AgentRef:  s.AgentRef,
			DependsOn: s.DependsOn,
			Input:     s.Input,
		}
	}
	return dag
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Workflow{}).
		Owns(&v1alpha1.Task{}).
		Complete(r)
}
