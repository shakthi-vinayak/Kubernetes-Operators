package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ReconcileTotal counts total reconciliation attempts.
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "platform_operator_reconcile_total",
			Help: "Total number of reconciliation attempts",
		},
		[]string{"result"},
	)

	// ReconcileErrorsTotal counts reconciliation errors by type.
	ReconcileErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "platform_operator_reconcile_errors_total",
			Help: "Total number of reconciliation errors",
		},
		[]string{"error_type", "error_class"},
	)

	// ReconcileDurationSeconds records reconciliation duration.
	ReconcileDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "platform_operator_reconcile_duration_seconds",
			Help:    "Duration of reconciliation in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"result"},
	)

	// ManagedApplications tracks the current number of managed applications.
	ManagedApplications = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "platform_operator_managed_applications",
			Help: "Current number of managed PlatformApplication resources",
		},
	)

	// SubReconcileTotal counts sub-reconciler operations by resource kind and result.
	SubReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "platform_operator_sub_reconcile_total",
			Help: "Total number of sub-reconciler operations",
		},
		[]string{"resource", "result"},
	)

	// SubReconcileDurationSeconds records sub-reconciler operation duration.
	SubReconcileDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "platform_operator_sub_reconcile_duration_seconds",
			Help:    "Duration of sub-reconciler operations in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"resource"},
	)

	// ResourceApplyTotal counts resource apply operations by kind and outcome.
	ResourceApplyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "platform_operator_resource_apply_total",
			Help: "Total number of resource apply operations (SSA)",
		},
		[]string{"resource", "outcome"},
	)

	// ReconcileRequeueTotal counts requeues by reason.
	ReconcileRequeueTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "platform_operator_reconcile_requeue_total",
			Help: "Total number of reconciliation requeues",
		},
		[]string{"reason"},
	)

	// StatusUpdateTotal counts status update attempts and conflicts.
	StatusUpdateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "platform_operator_status_update_total",
			Help: "Total number of status update attempts",
		},
		[]string{"result"},
	)

	// APICallDurationSeconds records the latency of Kubernetes API calls.
	APICallDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "platform_operator_api_call_duration_seconds",
			Help:    "Duration of Kubernetes API calls (get, create, patch, delete)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
		},
		[]string{"operation", "resource"},
	)

	// NoOpApplyTotal counts apply operations that detected no changes.
	NoOpApplyTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "platform_operator_noop_apply_total",
			Help: "Total number of apply operations where no changes were detected (semantic equality)",
		},
		[]string{"resource"},
	)

	// ActiveReconcilers tracks the number of currently active reconcile workers.
	ActiveReconcilers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "platform_operator_active_reconcilers",
			Help: "Number of currently active reconcile workers",
		},
	)

	// WorkQueueDepth tracks the depth of the reconcile work queue.
	WorkQueueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "platform_operator_workqueue_depth",
			Help: "Current depth of the reconcile work queue",
		},
		[]string{"name"},
	)

	// WorkQueueLatency tracks how long items wait in the queue.
	WorkQueueLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "platform_operator_workqueue_latency_seconds",
			Help:    "Time items spend in the work queue before processing",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"name"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		ReconcileTotal,
		ReconcileErrorsTotal,
		ReconcileDurationSeconds,
		ManagedApplications,
		SubReconcileTotal,
		SubReconcileDurationSeconds,
		ResourceApplyTotal,
		ReconcileRequeueTotal,
		StatusUpdateTotal,
		APICallDurationSeconds,
		NoOpApplyTotal,
		ActiveReconcilers,
		WorkQueueDepth,
		WorkQueueLatency,
	)
}

// ObserveReconcile records reconciliation metrics (total, duration, requeue).
func ObserveReconcile(result string, duration time.Duration) {
	ReconcileTotal.WithLabelValues(result).Inc()
	ReconcileDurationSeconds.WithLabelValues(result).Observe(duration.Seconds())
}

// ObserveSubReconcile records sub-reconciler metrics.
func ObserveSubReconcile(resource, result string, duration time.Duration) {
	SubReconcileTotal.WithLabelValues(resource, result).Inc()
	SubReconcileDurationSeconds.WithLabelValues(resource).Observe(duration.Seconds())
}

// ObserveResourceApply records a resource apply operation.
func ObserveResourceApply(resource, outcome string) {
	ResourceApplyTotal.WithLabelValues(resource, outcome).Inc()
}

// ObserveError records a reconciliation error with classification.
func ObserveError(errorType, errorClass string) {
	ReconcileErrorsTotal.WithLabelValues(errorType, errorClass).Inc()
}

// ObserveAPICall records API call latency metrics.
func ObserveAPICall(operation, resource string, duration time.Duration) {
	APICallDurationSeconds.WithLabelValues(operation, resource).Observe(duration.Seconds())
}

// ObserveNoOpApply records a no-op apply (resource unchanged).
func ObserveNoOpApply(resource string) {
	NoOpApplyTotal.WithLabelValues(resource).Inc()
}
