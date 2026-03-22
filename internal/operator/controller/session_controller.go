package controller

import (
	"context"
	"fmt"
	"time"

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

// SessionReconciler reconciles Session objects.
type SessionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agentspec.io,resources=sessions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agentspec.io,resources=sessions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agentspec.io,resources=sessions/finalizers,verbs=update
// +kubebuilder:rbac:groups=agentspec.io,resources=agents,verbs=get;list;watch
// +kubebuilder:rbac:groups=agentspec.io,resources=memoryclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *SessionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var session v1alpha1.Session
	if err := r.Get(ctx, req.NamespacedName, &session); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Skip terminal phases.
	if session.Status.Phase == "Expired" || session.Status.Phase == "Terminated" {
		return ctrl.Result{}, nil
	}

	// Validate agentRef exists in the same namespace.
	var agent v1alpha1.Agent
	if err := r.Get(ctx, types.NamespacedName{Name: session.Spec.AgentRef, Namespace: session.Namespace}, &agent); err != nil {
		if errors.IsNotFound(err) {
			msg := fmt.Sprintf("agentRef %q not found in namespace %q", session.Spec.AgentRef, session.Namespace)
			status.SetFailed(&session.Status.Conditions, session.Generation, "AgentNotFound", msg)
			session.Status.Phase = "Failed"
			_ = r.Status().Update(ctx, &session)
			r.Recorder.Event(&session, corev1.EventTypeWarning, "AgentNotFound", msg)
			log.Info("session agentRef not found", "session", session.Name, "agentRef", session.Spec.AgentRef)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// Validate memoryClassRef exists (cluster-scoped, no namespace).
	if session.Spec.MemoryClassRef != "" {
		var mc v1alpha1.MemoryClass
		if err := r.Get(ctx, types.NamespacedName{Name: session.Spec.MemoryClassRef}, &mc); err != nil {
			if errors.IsNotFound(err) {
				msg := fmt.Sprintf("memoryClassRef %q not found", session.Spec.MemoryClassRef)
				status.SetFailed(&session.Status.Conditions, session.Generation, "MemoryClassNotFound", msg)
				session.Status.Phase = "Failed"
				_ = r.Status().Update(ctx, &session)
				r.Recorder.Event(&session, corev1.EventTypeWarning, "MemoryClassNotFound", msg)
				log.Info("session memoryClassRef not found", "session", session.Name, "memoryClassRef", session.Spec.MemoryClassRef)
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			return ctrl.Result{}, err
		}
	}

	// Set owner reference to the Agent so sessions are garbage-collected with the agent.
	if !hasOwnerRef(session.OwnerReferences, agent.UID) {
		if err := controllerutil.SetOwnerReference(&agent, &session, r.Scheme); err != nil {
			log.Error(err, "failed to set owner reference")
			return ctrl.Result{}, err
		}
		if err := r.Update(ctx, &session); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Initialize createdAt and lastActivityTime on first reconcile.
	now := metav1.Now()
	if session.Status.CreatedAt == nil {
		session.Status.CreatedAt = &now
	}
	if session.Status.LastActivityTime == nil {
		session.Status.LastActivityTime = &now
	}

	// Check TTL expiration if memoryClassRef is set.
	if session.Spec.MemoryClassRef != "" {
		var mc v1alpha1.MemoryClass
		if err := r.Get(ctx, types.NamespacedName{Name: session.Spec.MemoryClassRef}, &mc); err == nil {
			if mc.Spec.TTL != "" {
				ttl, err := time.ParseDuration(mc.Spec.TTL)
				if err == nil && session.Status.LastActivityTime != nil {
					if time.Since(session.Status.LastActivityTime.Time) > ttl {
						session.Status.Phase = "Expired"
						status.SetFailed(&session.Status.Conditions, session.Generation, "Expired", "Session exceeded TTL")
						if err := r.Status().Update(ctx, &session); err != nil {
							return ctrl.Result{}, err
						}
						r.Recorder.Event(&session, corev1.EventTypeNormal, "Expired", "Session expired due to TTL")
						log.Info("session expired", "session", session.Name)
						return ctrl.Result{}, nil
					}
				}
			}
		}
	}

	// Set Active phase and Ready condition.
	session.Status.Phase = "Active"
	status.SetReady(&session.Status.Conditions, session.Generation, "Active", "Session is active")

	if err := r.Status().Update(ctx, &session); err != nil {
		return ctrl.Result{}, err
	}

	r.Recorder.Event(&session, corev1.EventTypeNormal, "Active", "Session is active")
	log.Info("session reconciled", "session", session.Name, "phase", session.Status.Phase,
		"messageCount", session.Status.MessageCount)

	return ctrl.Result{}, nil
}

// hasOwnerRef checks if the given UID is already in the owner references.
func hasOwnerRef(refs []metav1.OwnerReference, uid types.UID) bool {
	for _, ref := range refs {
		if ref.UID == uid {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *SessionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Session{}).
		Complete(r)
}
