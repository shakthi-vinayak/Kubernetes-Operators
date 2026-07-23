package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ReconcileTotal counts total reconciliation attempts.
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "platform_operator_reconcile_total",
			Help:      "Total number of reconciliation attempts",
			Namespace: "platform_operator",
		},
		[]string{"result"},
	)

	// ReconcileErrorsTotal counts reconciliation errors.
	ReconcileErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "platform_operator_reconcile_errors_total",
			Help:      "Total number of reconciliation errors",
			Namespace: "platform_operator",
		},
		[]string{"error_type"},
	)

	// ReconcileDurationSeconds records reconciliation duration.
	ReconcileDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:      "platform_operator_reconcile_duration_seconds",
			Help:      "Duration of reconciliation in seconds",
			Namespace: "platform_operator",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"result"},
	)

	// ManagedApplications tracks the current number of managed applications.
	ManagedApplications = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:      "platform_operator_managed_applications",
			Help:      "Current number of managed PlatformApplication resources",
			Namespace: "platform_operator",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(
		ReconcileTotal,
		ReconcileErrorsTotal,
		ReconcileDurationSeconds,
		ManagedApplications,
	)
}
