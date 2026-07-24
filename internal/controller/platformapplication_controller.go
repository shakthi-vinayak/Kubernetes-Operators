package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
	"github.com/example/platform-operator/internal/controller/subreconcilers"
	platformerrors "github.com/example/platform-operator/internal/errors"
	"github.com/example/platform-operator/internal/metrics"
	"github.com/example/platform-operator/internal/status"
	"github.com/example/platform-operator/internal/tracing"
)

const (
	// finalizerName is the finalizer added to PlatformApplication resources
	// for cleanup operations before garbage collection.
	finalizerName = "platform.example.io/cleanup"

	// requeueDelay is the default requeue delay for transient errors.
	requeueDelay = 30 * time.Second

	// conflictRequeueDelay is a short delay for optimistic concurrency conflicts.
	conflictRequeueDelay = 1 * time.Second

	// progressCheckDelay is the requeue delay when waiting for replicas to become ready.
	progressCheckDelay = 10 * time.Second
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
// The reconciler is idempotent, restart-safe, and handles errors with
// classification-based retry strategies:
//   - Transient errors: exponential backoff via controller-runtime
//   - Conflict errors: short requeue delay (1s) for fresh state
//   - Permanent errors: no retry, condition set to Degraded
type PlatformApplicationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	// Concurrency is the maximum number of concurrent reconcile workers.
	// Defaults to 1 if not set.
	Concurrency int
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
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is the main reconciliation loop for PlatformApplication resources.
//
// This function is called by controller-runtime whenever a PlatformApplication
// resource is created, updated, or deleted. It may also be called periodically
// by the resync mechanism or due to events from watched resources.
//
// The function is idempotent and handles duplicate events gracefully.
func (r *PlatformApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues(
		"controller", "platformapplication",
		"resource", req.NamespacedName.String(),
	)
	ctx = log.IntoContext(ctx, logger)
	startTime := time.Now()

	// Track active reconcile workers.
	metrics.ActiveReconcilers.Inc()
	defer metrics.ActiveReconcilers.Dec()

	// Start tracing span for this reconciliation.
	ctx, span := tracing.StartSpan(ctx, "reconcile.PlatformApplication",
		attribute.String("reconcile.name", req.Name),
		attribute.String("reconcile.namespace", req.Namespace),
	)
	defer span.End()

	// Step 1: Fetch the PlatformApplication resource.
	app := &platformv1alpha1.PlatformApplication{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if apierrors.IsNotFound(err) {
			// Resource was deleted after the reconcile request was queued.
			// Kubernetes GC handles owned resources via owner references.
			logger.Info("resource not found, likely deleted")
			return ctrl.Result{}, nil
		}
		// Transient error — return error to trigger retry with exponential backoff.
		logger.Error(err, "unable to fetch resource")
		metrics.ObserveError("fetch_error", string(platformerrors.Classify(err)))
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
			logger.Error(err, "unable to add finalizer")
			return ctrl.Result{}, fmt.Errorf("adding finalizer: %w", err)
		}
		// The update event will trigger another reconciliation.
		return ctrl.Result{}, nil
	}

	// Mark as progressing at the start of reconciliation.
	status.SetProgressing(&app.Status, metav1.ConditionTrue, "Reconciling", "Reconciliation in progress")
	status.SetConfigurationValid(&app.Status, metav1.ConditionTrue, "Valid", "Configuration is valid")

	// Step 3: Reconcile all child resources with sub-reconciler metrics.
	reconcileErr := r.reconcileResources(ctx, app)

	// Step 4: Update status regardless of reconcile outcome.
	app.Status.ObservedGeneration = app.Generation
	app.Status.DeployedVersion = app.Spec.Image.Tag

	// Fetch deployment status for ready replicas.
	r.updateDeploymentStatus(ctx, app)

	// Set conditions based on reconcile outcome.
	result := r.setConditions(app, reconcileErr)

	// Set URL from gateway configuration.
	if app.Spec.Gateway.Enabled && app.Spec.Gateway.Host != "" {
		app.Status.URL = fmt.Sprintf("https://%s", app.Spec.Gateway.Host)
	}

	// Update status on the API server.
	if err := r.Status().Update(ctx, app); err != nil {
		metrics.StatusUpdateTotal.WithLabelValues("error").Inc()
		if apierrors.IsConflict(err) {
			metrics.StatusUpdateTotal.WithLabelValues("conflict").Inc()
			metrics.ReconcileRequeueTotal.WithLabelValues("status_conflict").Inc()
			logger.Info("status update conflict, re-queuing")
			return ctrl.Result{RequeueAfter: conflictRequeueDelay}, nil
		}
		logger.Error(err, "unable to update status")
		return ctrl.Result{}, err
	}
	metrics.StatusUpdateTotal.WithLabelValues("success").Inc()

	// Step 5: Record metrics and return.
	duration := time.Since(startTime)

	if reconcileErr != nil {
		return r.handleErrorResult(ctx, app, reconcileErr, duration)
	}

	metrics.ObserveReconcile("success", duration)

	// If not all replicas are ready, re-queue to check progress.
	if app.Status.ReadyReplicas < app.Spec.Replicas.Min {
		metrics.ReconcileRequeueTotal.WithLabelValues("replicas_not_ready").Inc()
		logger.Info("waiting for replicas",
			"ready", app.Status.ReadyReplicas,
			"desired", app.Spec.Replicas.Min)
		return ctrl.Result{RequeueAfter: progressCheckDelay}, nil
	}

	logger.Info("reconciliation complete", "duration_seconds", duration.Seconds(),
		"readyReplicas", app.Status.ReadyReplicas)
	return result, nil
}

// reconcileResources applies all child resources in sequence.
// Each sub-reconciler is timed and its metrics are recorded.
func (r *PlatformApplicationReconciler) reconcileResources(ctx context.Context, app *platformv1alpha1.PlatformApplication) error {
	type subReconciler struct {
		name string
		fn   func() error
	}

	reconcilers := []subReconciler{
		{"Deployment", func() error {
			_, err := subreconcilers.ReconcileDeployment(ctx, r.Client, r.Scheme, app)
			return err
		}},
		{"Service", func() error {
			_, err := subreconcilers.ReconcileService(ctx, r.Client, r.Scheme, app)
			return err
		}},
		{"HPA", func() error {
			_, err := subreconcilers.ReconcileHPA(ctx, r.Client, r.Scheme, app)
			return err
		}},
		{"HTTPRoute", func() error {
			_, err := subreconcilers.ReconcileHTTPRoute(ctx, r.Client, r.Scheme, app)
			return err
		}},
		{"NetworkPolicy", func() error {
			_, err := subreconcilers.ReconcileNetworkPolicy(ctx, r.Client, r.Scheme, app)
			return err
		}},
		{"PodDisruptionBudget", func() error {
			_, err := subreconcilers.ReconcilePDB(ctx, r.Client, r.Scheme, app)
			return err
		}},
		{"ServiceMonitor", func() error {
			_, err := subreconcilers.ReconcileServiceMonitor(ctx, r.Client, r.Scheme, app)
			return err
		}},
	}

	for _, sr := range reconcilers {
		srStart := time.Now()

		// Create a tracing span for this sub-reconciler.
		srCtx, srSpan := tracing.StartSpan(ctx, "subreconcile."+sr.name,
			tracing.SubReconcileAttributes(sr.name, app.Name, app.Namespace)...)
		_ = srCtx

		if err := sr.fn(); err != nil {
			srSpan.RecordError(err)
			srSpan.SetStatus(codes.Error, err.Error())
			srSpan.End()
			srDuration := time.Since(srStart)
			classifiedErr := platformerrors.ClassifyOrWrap(err, app.Name, sr.name)
			metrics.ObserveSubReconcile(sr.name, "error", srDuration)
			metrics.ObserveError(sr.name, string(classifiedErr.Class))

			// Record a Kubernetes event for the failure.
			r.Recorder.Eventf(app, "Warning", "ReconcileFailed",
				"Failed to reconcile %s: %v", sr.name, classifiedErr)

			return fmt.Errorf("reconciling %s: %w", sr.name, classifiedErr)
		}

		srDuration := time.Since(srStart)
		srSpan.SetStatus(codes.Ok, "")
		srSpan.End()
		metrics.ObserveSubReconcile(sr.name, "success", srDuration)
	}

	return nil
}

// setConditions sets status conditions based on the reconciliation outcome.
func (r *PlatformApplicationReconciler) setConditions(app *platformv1alpha1.PlatformApplication, reconcileErr error) ctrl.Result {
	if reconcileErr != nil {
		status.SetProgressing(&app.Status, metav1.ConditionFalse, "ReconcileError", reconcileErr.Error())
		status.SetDegraded(&app.Status, metav1.ConditionTrue, "ReconcileError", reconcileErr.Error())
		status.SetReady(&app.Status, metav1.ConditionFalse, "ReconcileError", "Reconciliation failed")
		return ctrl.Result{}
	}

	if app.Status.ReadyReplicas >= app.Spec.Replicas.Min {
		status.SetProgressing(&app.Status, metav1.ConditionFalse, "Deployed", "All resources reconciled successfully")
		status.SetDegraded(&app.Status, metav1.ConditionFalse, "Healthy", "Application is healthy")
		status.SetReady(&app.Status, metav1.ConditionTrue, "Available", "Application is available")
		return ctrl.Result{}
	}

	status.SetProgressing(&app.Status, metav1.ConditionTrue, "Deploying",
		fmt.Sprintf("Waiting for %d/%d replicas", app.Status.ReadyReplicas, app.Spec.Replicas.Min))
	status.SetDegraded(&app.Status, metav1.ConditionFalse, "Deploying", "Deployment is rolling out")
	status.SetReady(&app.Status, metav1.ConditionFalse, "InsufficientReplicas",
		fmt.Sprintf("Only %d/%d replicas ready", app.Status.ReadyReplicas, app.Spec.Replicas.Min))
	return ctrl.Result{RequeueAfter: progressCheckDelay}
}

// handleErrorResult processes reconciliation errors and returns appropriate requeue behavior.
func (r *PlatformApplicationReconciler) handleErrorResult(ctx context.Context, app *platformv1alpha1.PlatformApplication, reconcileErr error, duration time.Duration) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Classify the error.
	var recErr *platformerrors.ReconcileError
	if errors.As(reconcileErr, &recErr) {
		metrics.ObserveReconcile("error", duration)
		metrics.ObserveError(recErr.Kind, string(recErr.Class))

		switch recErr.Class {
		case platformerrors.ClassConflict:
			metrics.ReconcileRequeueTotal.WithLabelValues("conflict").Inc()
			logger.Info("conflict error, re-queuing with short delay")
			return ctrl.Result{RequeueAfter: conflictRequeueDelay}, nil

		case platformerrors.ClassPermanent:
			// Permanent errors should not be retried aggressively.
			// Record event and requeue with long delay for eventual manual intervention.
			r.Recorder.Eventf(app, "Warning", "PermanentError",
				"Permanent error (manual intervention required): %v", recErr)
			logger.Error(recErr, "permanent reconciliation error — will retry slowly")
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil

		case platformerrors.ClassTransient:
			metrics.ReconcileRequeueTotal.WithLabelValues("transient").Inc()
			logger.Info("transient error, will retry with backoff", "error", recErr.Err)
			return ctrl.Result{RequeueAfter: requeueDelay}, reconcileErr
		}
	}

	// Unknown error — use default backoff.
	metrics.ObserveReconcile("error", duration)
	metrics.ObserveError("unknown", string(platformerrors.ClassUnknown))
	metrics.ReconcileRequeueTotal.WithLabelValues("unknown").Inc()
	return ctrl.Result{RequeueAfter: requeueDelay}, reconcileErr
}

// handleDeletion processes the finalizer cleanup before the resource is deleted.
// The cleanup is idempotent: calling it multiple times is safe.
func (r *PlatformApplicationReconciler) handleDeletion(ctx context.Context, app *platformv1alpha1.PlatformApplication) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(app, finalizerName) {
		return ctrl.Result{}, nil
	}

	logger.Info("handling deletion", "name", app.Name, "namespace", app.Namespace)

	// Perform cleanup operations (currently a no-op — owner references handle GC).
	// In a production operator, this might:
	// - Remove external DNS records
	// - Clean up external load balancer configurations
	// - Emit audit events
	// All cleanup must be idempotent.

	// Remove finalizer to allow deletion to proceed.
	controllerutil.RemoveFinalizer(app, finalizerName)
	if err := r.Update(ctx, app); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{RequeueAfter: conflictRequeueDelay}, nil
		}
		return ctrl.Result{}, fmt.Errorf("removing finalizer: %w", err)
	}

	r.Recorder.Event(app, "Normal", "Deleting", "Finalizer removed, deletion will proceed")
	logger.Info("finalizer removed, deletion will proceed")
	return ctrl.Result{}, nil
}

// updateDeploymentStatus fetches the current Deployment and updates
// the ready replica count in the PlatformApplication status.
func (r *PlatformApplicationReconciler) updateDeploymentStatus(ctx context.Context, app *platformv1alpha1.PlatformApplication) {
	dep := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Name: app.Name, Namespace: app.Namespace}, dep); err != nil {
		return
	}
	app.Status.ReadyReplicas = dep.Status.ReadyReplicas
}

// SetupWithManager sets up the controller with the Manager.
// It configures concurrency, rate limiting, and resource watches.
func (r *PlatformApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Wire the event recorder if not already set.
	if r.Recorder == nil {
		r.Recorder = mgr.GetEventRecorderFor("platform-operator")
	}

	concurrency := r.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.PlatformApplication{}).
		Owns(&appsv1.Deployment{}).
		Named("platformapplication").
		WithOptions(controller.Options{
			MaxConcurrentReconciles: concurrency,
			// controller-runtime's default rate limiter provides
			// exponential backoff (5ms to 1000s) for error retries.
		}).
		Complete(r)
}
