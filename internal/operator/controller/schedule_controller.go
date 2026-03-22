package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	opmetrics "github.com/szaher/agentspec/internal/operator/metrics"
	opstatus "github.com/szaher/agentspec/internal/operator/status"
)

// ScheduleReconciler reconciles Schedule objects.
type ScheduleReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=schedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=schedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=schedules/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=tasks,verbs=create;get;list;watch
// +kubebuilder:rbac:groups=agentspec.io,resources=workflows,verbs=create;get;list;watch
// +kubebuilder:rbac:groups=agentspec.io,resources=evalruns,verbs=create;get;list;watch

func (r *ScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var schedule v1alpha1.Schedule
	if err := r.Get(ctx, req.NamespacedName, &schedule); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if suspended.
	if schedule.Spec.Suspend != nil && *schedule.Spec.Suspend {
		log.Info("schedule is suspended", "name", schedule.Name)
		return ctrl.Result{}, nil
	}

	// Parse cron expression.
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(schedule.Spec.Schedule)
	if err != nil {
		opstatus.SetFailed(&schedule.Status.Conditions, schedule.Generation, "InvalidSchedule", fmt.Sprintf("invalid cron expression: %v", err))
		_ = r.Status().Update(ctx, &schedule)
		return ctrl.Result{}, nil
	}

	now := time.Now()

	// Compute next run time.
	var lastScheduled time.Time
	if schedule.Status.LastScheduleTime != nil {
		lastScheduled = schedule.Status.LastScheduleTime.Time
	} else {
		lastScheduled = schedule.CreationTimestamp.Time
	}
	nextRun := sched.Next(lastScheduled)
	nextRunMeta := metav1.NewTime(nextRun)
	schedule.Status.NextScheduleTime = &nextRunMeta

	// Check if it's time to trigger.
	if now.Before(nextRun) {
		opstatus.SetReady(&schedule.Status.Conditions, schedule.Generation, "Scheduled", fmt.Sprintf("Next run at %s", nextRun.Format(time.RFC3339)))
		_ = r.Status().Update(ctx, &schedule)
		requeueAfter := nextRun.Sub(now)
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	// Check starting deadline.
	if schedule.Spec.StartingDeadlineSeconds != nil {
		deadline := nextRun.Add(time.Duration(*schedule.Spec.StartingDeadlineSeconds) * time.Second)
		if now.After(deadline) {
			schedule.Status.MissedScheduleCount++
			opmetrics.ScheduleMissesTotal.WithLabelValues(schedule.Namespace, schedule.Name).Inc()
			log.Info("missed schedule deadline", "name", schedule.Name, "nextRun", nextRun)
			nowMeta := metav1.NewTime(now)
			schedule.Status.LastScheduleTime = &nowMeta
			_ = r.Status().Update(ctx, &schedule)
			return ctrl.Result{RequeueAfter: time.Until(sched.Next(now))}, nil
		}
	}

	// Check concurrency policy.
	if schedule.Spec.ConcurrencyPolicy == "Forbid" && len(schedule.Status.ActiveTaskRefs) > 0 {
		log.Info("concurrency policy Forbid: skipping because active tasks exist", "name", schedule.Name)
		return ctrl.Result{RequeueAfter: time.Until(sched.Next(now))}, nil
	}

	if schedule.Spec.ConcurrencyPolicy == "Replace" && len(schedule.Status.ActiveTaskRefs) > 0 {
		for _, ref := range schedule.Status.ActiveTaskRefs {
			var task v1alpha1.Task
			if err := r.Get(ctx, client.ObjectKey{Name: ref, Namespace: schedule.Namespace}, &task); err == nil {
				_ = r.Delete(ctx, &task)
			}
		}
		schedule.Status.ActiveTaskRefs = nil
	}

	// Create the triggered resource.
	taskName, err := r.createTriggeredResource(ctx, &schedule)
	if err != nil {
		log.Error(err, "failed to create triggered resource")
		opstatus.SetFailed(&schedule.Status.Conditions, schedule.Generation, "TriggerFailed", err.Error())
		_ = r.Status().Update(ctx, &schedule)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status.
	nowMeta := metav1.NewTime(now)
	schedule.Status.LastScheduleTime = &nowMeta
	schedule.Status.ActiveTaskRefs = append(schedule.Status.ActiveTaskRefs, taskName)

	// Enforce history limit.
	if schedule.Spec.SuccessfulTasksHistoryLimit != nil {
		limit := int(*schedule.Spec.SuccessfulTasksHistoryLimit)
		if len(schedule.Status.ActiveTaskRefs) > limit {
			schedule.Status.ActiveTaskRefs = schedule.Status.ActiveTaskRefs[len(schedule.Status.ActiveTaskRefs)-limit:]
		}
	}

	opmetrics.ScheduleTriggersTotal.WithLabelValues(schedule.Namespace, schedule.Name).Inc()
	r.Recorder.Eventf(&schedule, "Normal", "Triggered", "Created task %s", taskName)

	opstatus.SetReady(&schedule.Status.Conditions, schedule.Generation, "Triggered", fmt.Sprintf("Triggered task %s", taskName))
	nextAfterNow := sched.Next(now)
	nextAfterNowMeta := metav1.NewTime(nextAfterNow)
	schedule.Status.NextScheduleTime = &nextAfterNowMeta
	_ = r.Status().Update(ctx, &schedule)

	return ctrl.Result{RequeueAfter: time.Until(nextAfterNow)}, nil
}

func (r *ScheduleReconciler) createTriggeredResource(ctx context.Context, schedule *v1alpha1.Schedule) (string, error) {
	ref := schedule.Spec.TargetRef
	taskName := fmt.Sprintf("%s-%d", schedule.Name, time.Now().Unix())

	switch ref.Kind {
	case "Agent":
		task := &v1alpha1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskName,
				Namespace: schedule.Namespace,
			},
			Spec: v1alpha1.TaskSpec{
				AgentRef: ref.Name,
			},
		}
		if schedule.Spec.TaskTemplate != nil {
			task.Spec.Input = schedule.Spec.TaskTemplate.Input
			task.Spec.Parameters = schedule.Spec.TaskTemplate.Parameters
		}
		if err := r.Create(ctx, task); err != nil {
			return "", fmt.Errorf("creating task: %w", err)
		}
		return taskName, nil

	case "Workflow":
		// For workflow triggers, create a Task that references the workflow's first agent.
		// Full workflow triggering would create the Workflow CR directly.
		task := &v1alpha1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskName,
				Namespace: schedule.Namespace,
				Labels: map[string]string{
					"agentspec.io/triggered-by": schedule.Name,
					"agentspec.io/target-kind":  "Workflow",
					"agentspec.io/target-name":  ref.Name,
				},
			},
			Spec: v1alpha1.TaskSpec{
				AgentRef: ref.Name,
			},
		}
		if schedule.Spec.TaskTemplate != nil {
			task.Spec.Input = schedule.Spec.TaskTemplate.Input
		}
		if err := r.Create(ctx, task); err != nil {
			return "", fmt.Errorf("creating workflow trigger task: %w", err)
		}
		return taskName, nil

	case "EvalRun":
		task := &v1alpha1.Task{
			ObjectMeta: metav1.ObjectMeta{
				Name:      taskName,
				Namespace: schedule.Namespace,
				Labels: map[string]string{
					"agentspec.io/triggered-by": schedule.Name,
					"agentspec.io/target-kind":  "EvalRun",
					"agentspec.io/target-name":  ref.Name,
				},
			},
			Spec: v1alpha1.TaskSpec{
				AgentRef: ref.Name,
			},
		}
		if err := r.Create(ctx, task); err != nil {
			return "", fmt.Errorf("creating evalrun trigger task: %w", err)
		}
		return taskName, nil

	default:
		return "", fmt.Errorf("unsupported target kind: %s", ref.Kind)
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Schedule{}).
		Complete(r)
}
