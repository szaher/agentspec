package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	"github.com/szaher/agentspec/internal/operator/status"
)

const (
	stateStoreFinalizer      = "agentspec.io/statestore-cleanup"
	defaultOrphanGracePeriod = 24 * time.Hour

	// Condition type for drift detection (not in the shared status package).
	conditionDrifted = "Drifted"

	// Periodic reconciliation interval.
	stateStoreRequeueInterval = 5 * time.Minute
)

// StateStoreReconciler reconciles StateStore objects.
type StateStoreReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=statestores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=statestores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=statestores/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *StateStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// 1. Fetch the StateStore CR.
	var store v1alpha1.StateStore
	if err := r.Get(ctx, req.NamespacedName, &store); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Handle deletion with finalizer.
	if !store.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&store, stateStoreFinalizer) {
			log.Info("cleaning up StateStore resources")
			r.Recorder.Event(&store, corev1.EventTypeNormal, "CleanupStarted", "Cleaning up state store resources")

			// Clear all entries on deletion.
			store.Status.Entries = nil
			store.Status.Healthy = false
			if err := r.Status().Update(ctx, &store); err != nil {
				log.Error(err, "failed to clear status during cleanup")
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&store, stateStoreFinalizer)
			if err := r.Update(ctx, &store); err != nil {
				return ctrl.Result{}, err
			}
			r.Recorder.Event(&store, corev1.EventTypeNormal, "CleanupComplete", "State store cleanup finished")
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present.
	if !controllerutil.ContainsFinalizer(&store, stateStoreFinalizer) {
		controllerutil.AddFinalizer(&store, stateStoreFinalizer)
		if err := r.Update(ctx, &store); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 3. Read current entries from status.
	entries := store.Status.Entries

	// 4. Build a set of FQNs referenced by Agent CRs in this namespace/scope.
	referencedFQNs, err := r.collectReferencedFQNs(ctx, &store)
	if err != nil {
		log.Error(err, "failed to collect referenced FQNs from agents")
		r.Recorder.Event(&store, corev1.EventTypeWarning, "AgentListFailed", err.Error())
		status.SetFailed(&store.Status.Conditions, store.Generation, "AgentListFailed", err.Error())
		store.Status.Healthy = false
		if statusErr := r.Status().Update(ctx, &store); statusErr != nil {
			log.Error(statusErr, "failed to update status after agent list error")
		}
		return ctrl.Result{RequeueAfter: stateStoreRequeueInterval}, nil
	}

	// 5. Detect drift and mark orphans.
	driftDetected := false
	now := metav1.Now()
	var reconciledEntries []v1alpha1.StateEntryStatus

	for i := range entries {
		entry := entries[i]

		// Check if entry is still referenced by any AgentSpec.
		if _, referenced := referencedFQNs[entry.FQN]; !referenced {
			if entry.Status != "orphaned" {
				// First time marking as orphaned.
				log.Info("marking entry as orphaned", "fqn", entry.FQN)
				entry.Status = "orphaned"
				entry.OrphanedAt = now
				r.Recorder.Event(&store, corev1.EventTypeWarning, "EntryOrphaned",
					fmt.Sprintf("Entry %s is no longer referenced by any AgentSpec", entry.FQN))
			}

			// 6. Auto-delete orphaned entries after grace period.
			if !entry.OrphanedAt.IsZero() && time.Since(entry.OrphanedAt.Time) >= defaultOrphanGracePeriod {
				log.Info("deleting orphaned entry past grace period", "fqn", entry.FQN,
					"orphanedAt", entry.OrphanedAt.Time)
				r.Recorder.Event(&store, corev1.EventTypeNormal, "OrphanDeleted",
					fmt.Sprintf("Orphaned entry %s deleted after grace period", entry.FQN))
				// Skip adding to reconciled entries (effectively deleting it).
				continue
			}
		} else if entry.Status == "orphaned" {
			// Entry was orphaned but is now referenced again.
			log.Info("entry re-adopted", "fqn", entry.FQN)
			entry.Status = "applied"
			entry.OrphanedAt = metav1.Time{}
			r.Recorder.Event(&store, corev1.EventTypeNormal, "EntryReAdopted",
				fmt.Sprintf("Previously orphaned entry %s is referenced again", entry.FQN))
		}

		// Detect hash drift: compare stored hash against the referenced hash.
		if expectedHash, ok := referencedFQNs[entry.FQN]; ok && expectedHash != "" && expectedHash != entry.Hash {
			driftDetected = true
			log.Info("drift detected", "fqn", entry.FQN, "storedHash", entry.Hash, "expectedHash", expectedHash)
		}

		reconciledEntries = append(reconciledEntries, entry)
	}

	// Re-fetch to avoid stale resourceVersion conflicts.
	if err := r.Get(ctx, req.NamespacedName, &store); err != nil {
		return ctrl.Result{}, err
	}

	// 7. Update status conditions.
	store.Status.Entries = reconciledEntries

	if driftDetected {
		meta.SetStatusCondition(&store.Status.Conditions, metav1.Condition{
			Type:               conditionDrifted,
			Status:             metav1.ConditionTrue,
			ObservedGeneration: store.Generation,
			Reason:             "HashMismatch",
			Message:            "One or more entries have drifted from expected state",
		})
		r.Recorder.Event(&store, corev1.EventTypeWarning, "DriftDetected", "State drift detected in one or more entries")
	} else {
		meta.SetStatusCondition(&store.Status.Conditions, metav1.Condition{
			Type:               conditionDrifted,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: store.Generation,
			Reason:             "InSync",
			Message:            "All entries are in sync",
		})
	}

	// 8. Update Healthy field and Ready condition.
	hasFailedEntries := false
	for _, e := range reconciledEntries {
		if e.Status == "failed" {
			hasFailedEntries = true
			break
		}
	}

	healthy := !driftDetected && !hasFailedEntries
	store.Status.Healthy = healthy

	if healthy {
		status.SetReady(&store.Status.Conditions, store.Generation, "Healthy", "State store is operating normally")
	} else {
		reason := "Unhealthy"
		message := "State store has issues:"
		if driftDetected {
			message += " drift detected"
		}
		if hasFailedEntries {
			if driftDetected {
				message += ","
			}
			message += " failed entries present"
		}
		status.SetFailed(&store.Status.Conditions, store.Generation, reason, message)
	}

	if err := r.Status().Update(ctx, &store); err != nil {
		return ctrl.Result{}, err
	}

	// 9. Requeue after 5 minutes for periodic reconciliation.
	return ctrl.Result{RequeueAfter: stateStoreRequeueInterval}, nil
}

// collectReferencedFQNs lists all Agent CRs in the same namespace and collects
// FQNs that reference this state store scope. The returned map keys are FQNs
// and values are the expected content hashes (empty string if unknown).
func (r *StateStoreReconciler) collectReferencedFQNs(ctx context.Context, store *v1alpha1.StateStore) (map[string]string, error) {
	var agentList v1alpha1.AgentList
	if err := r.List(ctx, &agentList, client.InNamespace(store.Namespace)); err != nil {
		return nil, fmt.Errorf("listing agents: %w", err)
	}

	fqns := make(map[string]string)
	for _, agent := range agentList.Items {
		// Derive the agent scope from namespace/name and match against the store scope.
		agentScope := agent.Namespace + "/" + agent.Name
		if agentScope != store.Spec.Scope {
			continue
		}

		// Collect FQN from the agent as a reference into the state store.
		fqn := agentScope
		hash := ""
		if agent.Status.ObservedGeneration > 0 {
			hash = fmt.Sprintf("gen-%d", agent.Status.ObservedGeneration)
		}
		fqns[fqn] = hash
	}

	return fqns, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StateStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.StateStore{}).
		Complete(r)
}
