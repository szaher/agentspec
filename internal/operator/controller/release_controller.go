package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	"github.com/szaher/agentspec/internal/operator/status"
)

var semverRegexp = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[a-zA-Z0-9.]+)?(\+[a-zA-Z0-9.]+)?$`)

// ReleaseReconciler reconciles Release objects.
type ReleaseReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=releases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=releases/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=agents,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *ReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var release v1alpha1.Release
	if err := r.Get(ctx, req.NamespacedName, &release); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Skip if already in a terminal phase (Superseded or RolledBack).
	if release.Status.Phase == "Superseded" || release.Status.Phase == "RolledBack" {
		return ctrl.Result{}, nil
	}

	// Validate semver version format.
	if !semverRegexp.MatchString(release.Spec.Version) {
		status.SetFailed(&release.Status.Conditions, release.Generation, "InvalidVersion",
			fmt.Sprintf("version %q is not valid semver", release.Spec.Version))
		release.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, &release)
		r.Recorder.Event(&release, corev1.EventTypeWarning, "InvalidVersion",
			fmt.Sprintf("version %q is not valid semver", release.Spec.Version))
		return ctrl.Result{}, nil
	}

	// Fetch the referenced Agent.
	var agent v1alpha1.Agent
	if err := r.Get(ctx, types.NamespacedName{Name: release.Spec.AgentRef, Namespace: release.Namespace}, &agent); err != nil {
		status.SetFailed(&release.Status.Conditions, release.Generation, "AgentNotFound",
			fmt.Sprintf("agent %q not found: %v", release.Spec.AgentRef, err))
		release.Status.Phase = "Failed"
		_ = r.Status().Update(ctx, &release)
		r.Recorder.Event(&release, corev1.EventTypeWarning, "AgentNotFound",
			fmt.Sprintf("agent %q not found", release.Spec.AgentRef))
		return ctrl.Result{}, nil
	}

	// Set owner reference to the Agent if not already set.
	if !r.hasOwnerRef(&release, &agent) {
		if err := controllerutil.SetOwnerReference(&agent, &release, r.Scheme); err != nil {
			log.Error(err, "failed to set owner reference")
			return ctrl.Result{}, err
		}
		if err := r.Update(ctx, &release); err != nil {
			return ctrl.Result{}, err
		}
		// Re-fetch after update.
		if err := r.Get(ctx, req.NamespacedName, &release); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle initial creation: capture snapshot and set phase to Created.
	if release.Status.Phase == "" {
		// Capture agent spec as snapshot if not already set.
		if release.Spec.Snapshot.Raw == nil {
			specJSON, err := json.Marshal(agent.Spec)
			if err != nil {
				log.Error(err, "failed to marshal agent spec")
				return ctrl.Result{}, err
			}
			release.Spec.Snapshot = runtime.RawExtension{Raw: specJSON}
			if err := r.Update(ctx, &release); err != nil {
				return ctrl.Result{}, err
			}
			// Re-fetch after update.
			if err := r.Get(ctx, req.NamespacedName, &release); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Mark older releases for the same agent as Superseded.
		if err := r.supersedePreviousReleases(ctx, &release); err != nil {
			log.Error(err, "failed to supersede previous releases")
			return ctrl.Result{}, err
		}

		release.Status.Phase = "Created"
		status.SetReady(&release.Status.Conditions, release.Generation, "Created", "Release created successfully")
		if err := r.Status().Update(ctx, &release); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&release, corev1.EventTypeNormal, "Created",
			fmt.Sprintf("Release %s created for agent %s", release.Spec.Version, release.Spec.AgentRef))
		log.Info("release created", "name", release.Name, "version", release.Spec.Version)
		return ctrl.Result{}, nil
	}

	// Handle promotion: when promoteTo is set and phase is Created.
	if release.Spec.PromoteTo != "" && release.Status.Phase == "Created" {
		// Apply snapshot back to the agent spec.
		if err := r.applySnapshotToAgent(ctx, &release, &agent); err != nil {
			status.SetFailed(&release.Status.Conditions, release.Generation, "PromotionFailed", err.Error())
			release.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, &release)
			r.Recorder.Event(&release, corev1.EventTypeWarning, "PromotionFailed", err.Error())
			return ctrl.Result{}, nil
		}

		now := metav1.Now()
		release.Status.Phase = "Promoted"
		release.Status.PromotedAt = &now
		release.Status.PromotedTo = release.Spec.PromoteTo
		status.SetReady(&release.Status.Conditions, release.Generation, "Promoted",
			fmt.Sprintf("Release promoted to %s", release.Spec.PromoteTo))
		if err := r.Status().Update(ctx, &release); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Event(&release, corev1.EventTypeNormal, "Promoted",
			fmt.Sprintf("Release %s promoted to %s", release.Spec.Version, release.Spec.PromoteTo))
		log.Info("release promoted", "name", release.Name, "promoteTo", release.Spec.PromoteTo)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// hasOwnerRef checks if the release already has an owner reference to the agent.
func (r *ReleaseReconciler) hasOwnerRef(release *v1alpha1.Release, agent *v1alpha1.Agent) bool {
	for _, ref := range release.OwnerReferences {
		if ref.UID == agent.UID {
			return true
		}
	}
	return false
}

// supersedePreviousReleases marks older releases for the same agent as Superseded.
func (r *ReleaseReconciler) supersedePreviousReleases(ctx context.Context, current *v1alpha1.Release) error {
	var releases v1alpha1.ReleaseList
	if err := r.List(ctx, &releases, client.InNamespace(current.Namespace)); err != nil {
		return err
	}

	for i := range releases.Items {
		rel := &releases.Items[i]
		if rel.Name == current.Name {
			continue
		}
		if rel.Spec.AgentRef != current.Spec.AgentRef {
			continue
		}
		if rel.Status.Phase == "Superseded" || rel.Status.Phase == "RolledBack" {
			continue
		}
		rel.Status.Phase = "Superseded"
		rel.Status.SupersededBy = current.Name
		status.SetReady(&rel.Status.Conditions, rel.Generation, "Superseded",
			fmt.Sprintf("Superseded by release %s", current.Name))
		if err := r.Status().Update(ctx, rel); err != nil {
			return err
		}
	}
	return nil
}

// applySnapshotToAgent restores the agent spec from the release snapshot.
func (r *ReleaseReconciler) applySnapshotToAgent(ctx context.Context, release *v1alpha1.Release, agent *v1alpha1.Agent) error {
	if release.Spec.Snapshot.Raw == nil {
		return fmt.Errorf("release %s has no snapshot", release.Name)
	}

	var spec v1alpha1.AgentSpec
	if err := json.Unmarshal(release.Spec.Snapshot.Raw, &spec); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot: %w", err)
	}

	agent.Spec = spec
	if err := r.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent from snapshot: %w", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Release{}).
		Complete(r)
}
