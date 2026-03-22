package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	AgentsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentspec_agents_total",
			Help: "Total number of agents by phase",
		},
		[]string{"namespace", "phase"},
	)

	TasksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentspec_tasks_total",
			Help: "Total number of tasks by result",
		},
		[]string{"namespace", "result"},
	)

	WorkflowDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "agentspec_workflow_duration_seconds",
			Help:    "Workflow execution duration in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
		[]string{"namespace", "workflow"},
	)

	PolicyViolationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentspec_policy_violations_total",
			Help: "Total number of policy violations",
		},
		[]string{"namespace", "policy"},
	)

	ScheduleTriggersTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentspec_schedule_triggers_total",
			Help: "Total number of schedule triggers",
		},
		[]string{"namespace", "schedule"},
	)

	ScheduleMissesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "agentspec_schedule_misses_total",
			Help: "Total number of missed schedules",
		},
		[]string{"namespace", "schedule"},
	)

	EvalRunScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "agentspec_evalrun_score",
			Help: "Latest evaluation run score",
		},
		[]string{"namespace", "agent"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		AgentsTotal,
		TasksTotal,
		WorkflowDurationSeconds,
		PolicyViolationsTotal,
		ScheduleTriggersTotal,
		ScheduleMissesTotal,
		EvalRunScore,
	)
}
