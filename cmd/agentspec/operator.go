package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	"github.com/szaher/agentspec/internal/operator/controller"
	_ "github.com/szaher/agentspec/internal/operator/metrics" // register Prometheus collectors
)

var operatorScheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(operatorScheme))
	utilruntime.Must(v1alpha1.AddToScheme(operatorScheme))
}

func newOperatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Kubernetes operator management commands",
	}
	cmd.AddCommand(newOperatorStartCmd())
	return cmd
}

func newOperatorStartCmd() *cobra.Command {
	var (
		metricsAddr          string
		healthProbeAddr      string
		enableLeaderElection bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the AgentSpec Kubernetes operator",
		Long:  "Start the controller manager that reconciles AgentSpec custom resources.",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := zap.Options{Development: verbose}
			ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
			log := ctrl.Log.WithName("operator")

			mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
				Scheme: operatorScheme,
				Metrics: metricsserver.Options{
					BindAddress: metricsAddr,
				},
				HealthProbeBindAddress: healthProbeAddr,
				LeaderElection:         enableLeaderElection,
				LeaderElectionID:       "agentspec-operator-lock",
			})
			if err != nil {
				return fmt.Errorf("creating manager: %w", err)
			}

			if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
				return fmt.Errorf("setting up health check: %w", err)
			}
			if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
				return fmt.Errorf("setting up ready check: %w", err)
			}

			// Register controllers.
			reconcilers := []interface {
				SetupWithManager(ctrl.Manager) error
			}{
				&controller.AgentReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "agent-controller"),
				},
				&controller.TaskReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "task-controller"),
				},
				&controller.WorkflowReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "workflow-controller"),
				},
				&controller.SessionReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "session-controller"),
				},
				&controller.MemoryClassReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "memoryclass-controller"),
				},
				&controller.ToolBindingReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "toolbinding-controller"),
				},
				&controller.PolicyReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "policy-controller"),
				},
				&controller.ClusterPolicyReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "clusterpolicy-controller"),
				},
				&controller.ScheduleReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "schedule-controller"),
				},
				&controller.ReleaseReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "release-controller"),
				},
				&controller.EvalRunReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "evalrun-controller"),
				},
				&controller.StateStoreReconciler{
					Client:   mgr.GetClient(),
					Scheme:   mgr.GetScheme(),
					Recorder: eventRecorder(mgr, "statestore-controller"),
				},
			}
			for _, rec := range reconcilers {
				if err := rec.SetupWithManager(mgr); err != nil {
					return fmt.Errorf("setting up controller: %w", err)
				}
			}

			log.Info("starting operator manager")
			if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
				return fmt.Errorf("running manager: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&metricsAddr, "metrics-bind-address", ":8080", "Address the metrics endpoint binds to")
	cmd.Flags().StringVar(&healthProbeAddr, "health-probe-bind-address", ":8081", "Address the health probe endpoint binds to")
	cmd.Flags().BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager")

	return cmd
}

//nolint:staticcheck // GetEventRecorderFor is deprecated but GetEventRecorder returns a different type.
func eventRecorder(mgr ctrl.Manager, name string) record.EventRecorder {
	return mgr.GetEventRecorderFor(name)
}
