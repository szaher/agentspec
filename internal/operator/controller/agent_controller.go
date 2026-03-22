package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	opmetrics "github.com/szaher/agentspec/internal/operator/metrics"
	"github.com/szaher/agentspec/internal/operator/status"
)

const (
	agentFinalizer      = "agentspec.io/agent-cleanup"
	agentRuntimeImage   = "agentspec-runtime:latest"
	agentRuntimePort    = 8080
	agentManagedByLabel = "app.kubernetes.io/managed-by"
	agentNameLabel      = "app.kubernetes.io/name"
	agentInstanceLabel  = "app.kubernetes.io/instance"
)

// AgentReconciler reconciles Agent objects.
type AgentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// RuntimeImage overrides the default agent runtime container image.
	RuntimeImage string
}

// +kubebuilder:rbac:groups=agentspec.io,resources=agents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=agents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=agents/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods;services;configmaps;secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=toolbindings,verbs=get;list;watch
// +kubebuilder:rbac:groups=agentspec.io,resources=memoryclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=agentspec.io,resources=sessions,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=tasks,verbs=get;list;watch;delete

func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var agent v1alpha1.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion with finalizer.
	if !agent.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&agent, agentFinalizer) {
			if err := r.cleanupOwnedResources(ctx, &agent); err != nil {
				log.Error(err, "failed to cleanup owned resources")
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&agent, agentFinalizer)
			if err := r.Update(ctx, &agent); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present.
	if !controllerutil.ContainsFinalizer(&agent, agentFinalizer) {
		controllerutil.AddFinalizer(&agent, agentFinalizer)
		if err := r.Update(ctx, &agent); err != nil {
			return ctrl.Result{}, err
		}
		// Return and let the re-queue pick up with the fresh resourceVersion.
		return ctrl.Result{}, nil
	}

	// Validate cross-resource references.
	if err := r.validateReferences(ctx, &agent); err != nil {
		r.Recorder.Event(&agent, corev1.EventTypeWarning, "BrokenReference", err.Error())
		return r.setFailedStatus(ctx, req, agent.Generation, "BrokenReference", err.Error(), 30*time.Second)
	}

	// Resolve bound tools.
	boundTools, err := r.resolveBoundTools(ctx, &agent)
	if err != nil {
		log.Error(err, "failed to resolve tool bindings")
	}

	// Resolve effective policy.
	effectivePolicy := ""
	if agent.Spec.PolicyRef != "" {
		effectivePolicy = agent.Spec.PolicyRef
	}

	// Ensure the agent spec ConfigMap exists.
	if err := r.ensureSpecConfigMap(ctx, &agent); err != nil {
		r.Recorder.Event(&agent, corev1.EventTypeWarning, "ConfigMapFailed", err.Error())
		return r.setFailedStatus(ctx, req, agent.Generation, "ConfigMapFailed", err.Error(), 10*time.Second)
	}

	// Ensure the agent runtime Deployment exists.
	if err := r.ensureDeployment(ctx, &agent); err != nil {
		r.Recorder.Event(&agent, corev1.EventTypeWarning, "DeploymentFailed", err.Error())
		return r.setFailedStatus(ctx, req, agent.Generation, "DeploymentFailed", err.Error(), 10*time.Second)
	}

	// Ensure the agent Service exists.
	if err := r.ensureService(ctx, &agent); err != nil {
		r.Recorder.Event(&agent, corev1.EventTypeWarning, "ServiceFailed", err.Error())
		return r.setFailedStatus(ctx, req, agent.Generation, "ServiceFailed", err.Error(), 10*time.Second)
	}

	// Re-fetch the agent to get the latest resourceVersion before the status update.
	// Creating/updating the Deployment and Service may trigger re-queues that modify
	// the object, causing stale-version conflicts if we use the old copy.
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		return ctrl.Result{}, err
	}

	// Set Ready status in a single update.
	now := metav1.Now()
	agent.Status.BoundTools = boundTools
	agent.Status.EffectivePolicy = effectivePolicy
	agent.Status.LastReconcileTime = &now
	agent.Status.ObservedGeneration = agent.Generation
	status.SetReady(&agent.Status.Conditions, agent.Generation, "Provisioned", "Agent resources provisioned successfully")
	agent.Status.Phase = status.DerivePhase(agent.Status.Conditions, false)

	if err := r.Status().Update(ctx, &agent); err != nil {
		return ctrl.Result{}, err
	}

	// Update metrics.
	opmetrics.AgentsTotal.WithLabelValues(agent.Namespace, agent.Status.Phase).Set(1)

	r.Recorder.Event(&agent, corev1.EventTypeNormal, "Ready", "Agent is ready")
	log.Info("agent reconciled", "name", agent.Name, "phase", agent.Status.Phase)

	return ctrl.Result{}, nil
}

// setFailedStatus re-fetches the agent and sets a failed condition. This avoids
// stale-resourceVersion conflicts by always reading the latest version first.
func (r *AgentReconciler) setFailedStatus(ctx context.Context, req ctrl.Request, generation int64, reason, message string, requeueAfter time.Duration) (ctrl.Result, error) {
	var agent v1alpha1.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		return ctrl.Result{}, err
	}
	status.SetFailed(&agent.Status.Conditions, generation, reason, message)
	agent.Status.Phase = status.DerivePhase(agent.Status.Conditions, false)
	_ = r.Status().Update(ctx, &agent)
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}

// ensureDeployment creates or updates the Deployment for the agent runtime.
func (r *AgentReconciler) ensureDeployment(ctx context.Context, agent *v1alpha1.Agent) error {
	deployName := agentDeploymentName(agent.Name)
	labels := agentLabels(agent.Name)
	image := r.runtimeImage()

	var replicas int32 = 1

	desired := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deployName,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					agentInstanceLabel: agent.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "agentspec-runtime",
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/agentspec", "run",
								"/etc/agentspec/agent.ias",
								"--no-auth",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: agentRuntimePort,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							Env: r.agentEnvVars(agent),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "agent-spec",
									MountPath: "/etc/agentspec",
									ReadOnly:  true,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromInt32(agentRuntimePort),
									},
								},
								InitialDelaySeconds: 10,
								PeriodSeconds:       15,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/healthz",
										Port: intstr.FromInt32(agentRuntimePort),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "agent-spec",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: agentSpecConfigMapName(agent.Name),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Set owner reference so the Deployment is cleaned up when the Agent is deleted.
	if err := controllerutil.SetControllerReference(agent, desired, r.Scheme); err != nil {
		return fmt.Errorf("setting owner reference on deployment: %w", err)
	}

	// Create or update.
	var existing appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Name: deployName, Namespace: agent.Namespace}, &existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update the existing deployment's spec.
	existing.Spec = desired.Spec
	existing.Labels = desired.Labels
	return r.Update(ctx, &existing)
}

// ensureService creates or updates the Service for the agent runtime.
func (r *AgentReconciler) ensureService(ctx context.Context, agent *v1alpha1.Agent) error {
	svcName := agentServiceName(agent.Name)
	labels := agentLabels(agent.Name)

	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				agentInstanceLabel: agent.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       agentRuntimePort,
					TargetPort: intstr.FromInt32(agentRuntimePort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if err := controllerutil.SetControllerReference(agent, desired, r.Scheme); err != nil {
		return fmt.Errorf("setting owner reference on service: %w", err)
	}

	var existing corev1.Service
	err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: agent.Namespace}, &existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update the existing service's spec (preserve ClusterIP).
	existing.Spec.Selector = desired.Spec.Selector
	existing.Spec.Ports = desired.Spec.Ports
	existing.Labels = desired.Labels
	return r.Update(ctx, &existing)
}

// agentEnvVars returns environment variables for the agent runtime container.
func (r *AgentReconciler) agentEnvVars(agent *v1alpha1.Agent) []corev1.EnvVar {
	vars := []corev1.EnvVar{
		{Name: "AGENTSPEC_AGENT_NAME", Value: agent.Name},
		{Name: "AGENTSPEC_MODEL", Value: agent.Spec.Model},
		{Name: "AGENTSPEC_STRATEGY", Value: agent.Spec.Strategy},
	}
	if agent.Spec.PromptRef != "" {
		vars = append(vars, corev1.EnvVar{
			Name:  "AGENTSPEC_PROMPT_REF",
			Value: agent.Spec.PromptRef,
		})
	}
	// Inject secrets from secretRefs as env vars sourced from K8s Secrets.
	for _, ref := range agent.Spec.SecretRefs {
		vars = append(vars, corev1.EnvVar{
			Name: ref.Key,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: ref.Name},
					Key:                  ref.Key,
				},
			},
		})
	}
	return vars
}

func (r *AgentReconciler) runtimeImage() string {
	if r.RuntimeImage != "" {
		return r.RuntimeImage
	}
	return agentRuntimeImage
}

// ensureSpecConfigMap creates or updates a ConfigMap containing a generated .ias
// file so the agentspec runtime container can load the agent definition.
func (r *AgentReconciler) ensureSpecConfigMap(ctx context.Context, agent *v1alpha1.Agent) error {
	cmName := agentSpecConfigMapName(agent.Name)
	labels := agentLabels(agent.Name)
	iasContent := r.generateIAS(agent)

	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: agent.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"agent.ias": iasContent,
		},
	}

	if err := controllerutil.SetControllerReference(agent, desired, r.Scheme); err != nil {
		return fmt.Errorf("setting owner reference on configmap: %w", err)
	}

	var existing corev1.ConfigMap
	err := r.Get(ctx, types.NamespacedName{Name: cmName, Namespace: agent.Namespace}, &existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	existing.Data = desired.Data
	existing.Labels = desired.Labels
	return r.Update(ctx, &existing)
}

// generateIAS produces a minimal IntentLang spec from the Agent CR fields.
func (r *AgentReconciler) generateIAS(agent *v1alpha1.Agent) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("package %q version \"1.0.0\" lang \"3.0\"\n\n", agent.Name))

	// Emit prompt if promptRef is set (assumes the ConfigMap has a "system-prompt" key).
	if agent.Spec.PromptRef != "" {
		b.WriteString(fmt.Sprintf("prompt %q {\n", agent.Spec.PromptRef))
		b.WriteString("  content \"You are a helpful assistant.\"\n")
		b.WriteString("}\n\n")
	}

	b.WriteString(fmt.Sprintf("agent %q {\n", agent.Name))
	if agent.Spec.PromptRef != "" {
		b.WriteString(fmt.Sprintf("  uses prompt %q\n", agent.Spec.PromptRef))
	}
	b.WriteString(fmt.Sprintf("  model %q\n", agent.Spec.Model))
	if agent.Spec.Strategy != "" {
		b.WriteString(fmt.Sprintf("  strategy %q\n", agent.Spec.Strategy))
	}
	if agent.Spec.MaxTurns > 0 {
		b.WriteString(fmt.Sprintf("  max_turns %d\n", agent.Spec.MaxTurns))
	}
	b.WriteString("}\n\n")

	b.WriteString("deploy \"k8s\" target \"process\" {\n")
	b.WriteString("  port 8080\n")
	b.WriteString("  health {\n")
	b.WriteString("    path \"/healthz\"\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")

	return b.String()
}

func agentSpecConfigMapName(agentName string) string {
	return fmt.Sprintf("agent-%s-spec", agentName)
}

func agentDeploymentName(agentName string) string {
	return fmt.Sprintf("agent-%s", agentName)
}

func agentServiceName(agentName string) string {
	return fmt.Sprintf("agent-%s", agentName)
}

func agentLabels(agentName string) map[string]string {
	return map[string]string{
		agentManagedByLabel: "agentspec-operator",
		agentNameLabel:      "agentspec-runtime",
		agentInstanceLabel:  agentName,
	}
}

func (r *AgentReconciler) validateReferences(ctx context.Context, agent *v1alpha1.Agent) error {
	ns := agent.Namespace

	// Validate promptRef (ConfigMap).
	if agent.Spec.PromptRef != "" {
		var cm corev1.ConfigMap
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.PromptRef, Namespace: ns}, &cm); err != nil {
			return fmt.Errorf("promptRef %q not found: %w", agent.Spec.PromptRef, err)
		}
	}

	// Validate skillRefs (ToolBindings).
	for _, ref := range agent.Spec.SkillRefs {
		var tb v1alpha1.ToolBinding
		if err := r.Get(ctx, types.NamespacedName{Name: ref, Namespace: ns}, &tb); err != nil {
			return fmt.Errorf("skillRef %q not found: %w", ref, err)
		}
	}

	// Validate toolBindingRefs.
	for _, ref := range agent.Spec.ToolBindingRefs {
		var tb v1alpha1.ToolBinding
		if err := r.Get(ctx, types.NamespacedName{Name: ref, Namespace: ns}, &tb); err != nil {
			return fmt.Errorf("toolBindingRef %q not found: %w", ref, err)
		}
	}

	// Validate memoryClassRef (cluster-scoped).
	if agent.Spec.MemoryClassRef != "" {
		var mc v1alpha1.MemoryClass
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.MemoryClassRef}, &mc); err != nil {
			return fmt.Errorf("memoryClassRef %q not found: %w", agent.Spec.MemoryClassRef, err)
		}
	}

	// Validate policyRef.
	if agent.Spec.PolicyRef != "" {
		var policy v1alpha1.Policy
		if err := r.Get(ctx, types.NamespacedName{Name: agent.Spec.PolicyRef, Namespace: ns}, &policy); err != nil {
			return fmt.Errorf("policyRef %q not found: %w", agent.Spec.PolicyRef, err)
		}
	}

	// Validate secretRefs.
	for _, ref := range agent.Spec.SecretRefs {
		var secret corev1.Secret
		if err := r.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ns}, &secret); err != nil {
			return fmt.Errorf("secretRef %q not found: %w", ref.Name, err)
		}
	}

	return nil
}

func (r *AgentReconciler) resolveBoundTools(ctx context.Context, agent *v1alpha1.Agent) ([]string, error) {
	var tools []string
	for _, ref := range agent.Spec.ToolBindingRefs {
		var tb v1alpha1.ToolBinding
		if err := r.Get(ctx, types.NamespacedName{Name: ref, Namespace: agent.Namespace}, &tb); err != nil {
			continue
		}
		tools = append(tools, tb.Name)
	}
	for _, ref := range agent.Spec.SkillRefs {
		var tb v1alpha1.ToolBinding
		if err := r.Get(ctx, types.NamespacedName{Name: ref, Namespace: agent.Namespace}, &tb); err != nil {
			continue
		}
		tools = append(tools, tb.Name)
	}
	return tools, nil
}

func (r *AgentReconciler) cleanupOwnedResources(ctx context.Context, agent *v1alpha1.Agent) error {
	log := log.FromContext(ctx)
	ns := agent.Namespace

	// Delete owned Sessions.
	var sessions v1alpha1.SessionList
	if err := r.List(ctx, &sessions, client.InNamespace(ns)); err == nil {
		for i := range sessions.Items {
			s := &sessions.Items[i]
			if s.Spec.AgentRef == agent.Name {
				log.Info("deleting owned session", "session", s.Name)
				if err := r.Delete(ctx, s); err != nil && !errors.IsNotFound(err) {
					return err
				}
			}
		}
	}

	// Delete owned Tasks.
	var tasks v1alpha1.TaskList
	if err := r.List(ctx, &tasks, client.InNamespace(ns)); err == nil {
		for i := range tasks.Items {
			t := &tasks.Items[i]
			if t.Spec.AgentRef == agent.Name {
				log.Info("deleting owned task", "task", t.Name)
				if err := r.Delete(ctx, t); err != nil && !errors.IsNotFound(err) {
					return err
				}
			}
		}
	}

	// Deployment and Service are cleaned up automatically via owner references.
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Agent{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
