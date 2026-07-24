package main

import (
	"context"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g., Azure, GCP, OIDC, etc.)
	// to ensure that gcp, azure, oidc, etc. auth plugins are properly registered.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
	"github.com/example/platform-operator/internal/controller"
	_ "github.com/example/platform-operator/internal/metrics" // Register custom metrics
	"github.com/example/platform-operator/internal/tracing"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(platformv1alpha1.AddToScheme(scheme))
}

// main is the entry point for the Platform Operator.
//
// It initializes the controller-runtime manager with:
//   - Leader election for high availability
//   - Health and readiness probes
//   - Prometheus metrics endpoint
//   - Configurable concurrency and rate limiting
//
// The operator watches PlatformApplication custom resources and reconciles
// them into a set of Kubernetes child resources (Deployment, Service, HPA,
// HTTPRoute, NetworkPolicy, PDB, ServiceMonitor).
func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool
	var leaderElectionID string
	var concurrentReconciles int
	var enableTracing bool
	var tracingExporter string
	var tracingEndpoint string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure "+
			"there is only one active controller manager.")
	flag.StringVar(&leaderElectionID, "leader-election-id", "platform-operator.example.io",
		"The name of the leader election resource.")
	flag.IntVar(&concurrentReconciles, "concurrent-reconciles", 1,
		"The number of concurrent reconcile workers.")
	flag.BoolVar(&enableTracing, "enable-tracing", false,
		"Enable OpenTelemetry tracing. Disabled by default.")
	flag.StringVar(&tracingExporter, "tracing-exporter", "none",
		"Trace exporter: 'otlp', 'stdout', or 'none'.")
	flag.StringVar(&tracingEndpoint, "tracing-endpoint", "localhost:4317",
		"OTLP collector endpoint (only used when exporter is 'otlp').")

	opts := zap.Options{
		Development: false,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Initialize tracing (no-op when disabled).
	tp, err := tracing.Init(context.Background(), tracing.Config{
		Enabled:     enableTracing,
		Exporter:    tracingExporter,
		Endpoint:    tracingEndpoint,
		ServiceName: "platform-operator",
		Insecure:    true,
	})
	if err != nil {
		setupLog.Error(err, "unable to initialize tracing")
		os.Exit(1)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			setupLog.Error(err, "error shutting down tracer provider")
		}
	}()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       leaderElectionID,
		// LeaderElectionNamespace defaults to the pod namespace when running
		// in-cluster. For local development, it uses the kubeconfig namespace.
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.PlatformApplicationReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Recorder:    mgr.GetEventRecorderFor("platform-operator"),
		Concurrency: concurrentReconciles,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PlatformApplication")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager",
		"leader-election", enableLeaderElection,
		"concurrent-reconciles", concurrentReconciles,
		"tracing", enableTracing,
		"tracing-exporter", tracingExporter,
	)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
