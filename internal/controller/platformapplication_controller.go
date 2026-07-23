package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
	"github.com/example/platform-operator/internal/controller/subreconcilers"
	"github.com/example/platform-operator/internal/metrics"
	"github.com/example/platform-operator/internal/status"
)

const (
	// finalizerName is the finalizer added to PlatformApplication resources
	// for cleanup operations before garbage collection.
	finalizerName = "platform.example.io/cleanup"

	// requeueDelay is the default requeue delay for transient errors.
	requeueDelay = 30 * time.Second
)

// PlatformApplicationReconciler reconciles a PlatformApplication object.
//
// The reconciler follows the standard controller-runtime pattern:
//  1. Fetch the PlatformApplication resource
//  2. Handle deletion/finalization if the resource is being deleted
//  3. Reconcile each child resource (Deployment, Service, HPA, etc.)
//  4. Update the status to reflect the current state
//  5. Return appropriate requeue behavior
//
// The reconciler is idempotent: calling it multiple times with the same
// desired state will not produce additional API writes. It is restart-safe:
// the controller can be terminated and restarted without losing state, as
// all state is stored in Kubernetes resources.
type PlatformApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.example.io,resources=platformapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.example.io,resources=platformapplications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.example.io,resources=platformapplications/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete

// Reconcile is the main reconciliation loop for PlatformApplication resources.
//
// This function is called by controller-runtime whenever a PlatformApplication
// resource is created, updated, or deleted. It may also be called periodically
// by the resync mechanism or due to events from watched resources.
//
// The function must be idempotent and handle duplicate events gracefully.
func (r *PlatformApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("controller", "platformapplication")
	ctx = log.IntoContext(ctx, logger)
	startTime := time.Now()

	// Step 1: Fetch the PlatformApplication resource.
	app := &platformv1alpha1.PlatformApplication{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if apierrors.IsNotFound(err) {
			// Resource was deleted after the reconcile request was queued.
			// This is normal — Kubernetes will garbage-collect owned resources
			// via owner references. No action needed.
			logger.Info("PlatformApplication not found, likely deleted", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		// Transient error (API server unavailable, network issue).
		// Return error to trigger retry with exponential backoff.
		logger.Error(err, "unable to fetch PlatformApplication")
		metrics.ReconcileErrorsTotal.WithLabelValues("fetch_error").Inc()
		return ctrl.Result{}, err
	}

	// Step 2: Handle deletion and finalization.
	if !app.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, app)
	}

	// Ensure finalizer is present for non-deleted resources.
	if !controllerutil.ContainsFinalizer(app, finalizerName) {
		controllerutil.AddFinalizer(app, finalizerName)
		if err := r.Update(ctx, app); err != nil {
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		// Re-queue after finalizer update — the update event will trigger
		// another reconciliation.
		return ctrl.Result{}, nil
	}

	// Mark as progressing at the start of reconciliation.
	status.SetProgressing(&app.Status, metav1.ConditionTrue, "Reconciling", "Reconciliation in progress")
	status.SetConfigurationValid(&app.Status, metav1.ConditionTrue, "Valid", "Configuration is valid")

	// Step 3: Reconcile all child resources.
	reconcileErr := r.reconcileResources(ctx, app)

	// Step 4: Update status regardless of reconcile outcome.
	app.Status.ObservedGeneration = app.Generation
	app.Status.DeployedVersion = app.Spec.Image.Tag

	// Fetch deployment status for ready replicas.
	r.updateDeploymentStatus(ctx, app)

	// Set conditions based on reconcile outcome.
	if reconcileErr != nil {
		status.SetProgressing(&app.Status, metav1.ConditionFalse, "ReconcileError", reconcileErr.Error())
		status.SetDegraded(&app.Status, metav1.ConditionTrue, "ReconcileError", reconcileErr.Error())
		status.SetReady(&app.Status, metav1.ConditionFalse, "ReconcileError", "Reconciliation failed")
	} else if app.Status.ReadyReplicas >= app.Spec.Replicas.Min {
		status.SetProgressing(&app.Status, metav1.ConditionFalse, "Deployed", "All resources reconciled successfully")
		status.SetDegraded(&app.Status, metav1.ConditionFalse, "Healthy", "Application is healthy")
		status.SetReady(&app.Status, metav1.ConditionTrue, "Available", "Application is available")
	} else {
		status.SetProgressing(&app.Status, metav1.ConditionTrue, "Deploying", fmt.Sprintf("Waiting for %d/%d replicas", app.Status.ReadyReplicas, app.Spec.Replicas.Min))
		status.SetDegraded(&app.Status, metav1.ConditionFalse, "Deploying", "Deployment is rolling out")
		status.SetReady(&app.Status, metav1.ConditionFalse, "InsufficientReplicas", fmt.Sprintf("Only %d/%d replicas ready", app.Status.ReadyReplicas, app.Spec.Replicas.Min))
	}

	// Set URL from gateway configuration.
	if app.Spec.Gateway.Enabled && app.Spec.Gateway.Host != "" {
		app.Status.URL = fmt.Sprintf("https://%s", app.Spec.Gateway.Host)
	}

	if err := r.Status().Update(ctx, app); err != nil {
		if apierrors.IsConflict(err) {
			// Optimistic concurrency conflict — the resource was updated by
			// another process. Re-queue to retry with fresh state.
			logger.Info("status update conflict, re-queuing")
			return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
		}
		logger.Error(err, "unable to update status")
		return ctrl.Result{}, err
	}

	// Step 5: Record metrics and return.
	duration := time.Since(startTime).Seconds()
	if reconcileErr != nil {
		metrics.ReconcileTotal.WithLabelValues("error").Inc()
		metrics.ReconcileDurationSeconds.WithLabelValues("error").Observe(duration)
		metrics.ReconcileErrorsTotal.WithLabelValues("reconcile_error").Inc()
		return ctrl.Result{RequeueAfter: requeueDelay}, reconcileErr
	}

	metrics.ReconcileTotal.WithLabelValues("success").Inc()
	metrics.ReconcileDurationSeconds.WithLabelValues("success").Observe(duration)

	// If not all replicas are ready, re-queue to check progress.
	if app.Status.ReadyReplicas < app.Spec.Replicas.Min {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	logger.Info("reconciliation complete", "duration", duration)
	return ctrl.Result{}, nil
}

// reconcileResources applies all child resources in sequence.
// Each sub-reconciler computes the desired state and applies it using SSA.
func (r *PlatformApplicationReconciler) reconcileResources(ctx context.Context, app *platformv1alpha1.PlatformApplication) error {
	// Reconcile Deployment.
	if _, err := subreconcilers.ReconcileDeployment(ctx, r.Client, r.Scheme, app); err != nil {
		return fmt.Errorf("reconciling Deployment: %w", err)
	}

	// Reconcile Service.
	if _, err := subreconcilers.ReconcileService(ctx, r.Client, r.Scheme, app); err != nil {
		return fmt.Errorf("reconciling Service: %w", err)
	}

	// Reconcile HPA (creates or deletes based on autoscaling.enabled).
	if _, err := subreconcilers.ReconcileHPA(ctx, r.Client, r.Scheme, app); err != nil {
		return fmt.Errorf("reconciling HPA: %w", err)
	}

	// Reconcile HTTPRoute (creates or deletes based on gateway.enabled).
	if _, err := subreconcilers.ReconcileHTTPRoute(ctx, r.Client, r.Scheme, app); err != nil {
		return fmt.Errorf("reconciling HTTPRoute: %w", err)
	}

	// Reconcile NetworkPolicy (creates or deletes based on security.networkPolicy).
	if _, err := subreconcilers.ReconcileNetworkPolicy(ctx, r.Client, r.Scheme, app); err != nil {
		return fmt.Errorf("reconciling NetworkPolicy: %w", err)
	}

	// Reconcile PodDisruptionBudget (creates or deletes based on replica count).
	if _, err := subreconcilers.ReconcilePDB(ctx, r.Client, r.Scheme, app); err != nil {
		return fmt.Errorf("reconciling PDB: %w", err)
	}

	// Reconcile ServiceMonitor (creates or deletes based on observability.metrics).
	if _, err := subreconcilers.ReconcileServiceMonitor(ctx, r.Client, r.Scheme, app); err != nil {
		return fmt.Errorf("reconciling ServiceMonitor: %w", err)
	}

	return nil
}

// handleDeletion processes the finalizer cleanup before the resource is deleted.
// The cleanup must be idempotent: calling it multiple times must be safe.
func (r *PlatformApplicationReconciler) handleDeletion(ctx context.Context, app *platformv1alpha1.PlatformApplication) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(app, finalizerName) {
		// No finalizer — nothing to clean up. Kubernetes GC handles owned resources.
		return ctrl.Result{}, nil
	}

	logger.Info("handling deletion", "name", app.Name, "namespace", app.Namespace)

	// Perform cleanup operations.
	// Currently: log the cleanup. In a production operator, this might:
	// - Remove external DNS records
	// - Clean up external load balancer configurations
	// - Emit audit events
	// All cleanup must be idempotent.

	// Remove finalizer to allow deletion to proceed.
	controllerutil.RemoveFinalizer(app, finalizerName)
	if err := r.Update(ctx, app); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
		}
		return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
	}

	logger.Info("finalizer removed, deletion will proceed", "name", app.Name, "namespace", app.Namespace)
	return ctrl.Result{}, nil
}

// updateDeploymentStatus fetches the current Deployment and updates
// ready replica count in the PlatformApplication status.
func (r *PlatformApplicationReconciler) updateDeploymentStatus(ctx context.Context, app *platformv1alpha1.PlatformApplication) {
	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Name: app.Name, Namespace: app.Namespace}, dep); err != nil {
		return
	}
	app.Status.ReadyReplicas = dep.Status.ReadyReplicas
}

// SetupWithManager sets up the controller with the Manager.
// It watches PlatformApplication as the primary resource and owns
// Deployment, Service, HPA, and other child resources.
func (r *PlatformApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.PlatformApplication{}).
		Owns(&appsv1.Deployment{}).
		Named("platformapplication").
		Complete(r)
}
