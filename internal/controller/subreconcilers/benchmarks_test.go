package subreconcilers

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

func newBenchApp(name string) *platformv1alpha1.PlatformApplication {
	return &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "bench-ns",
			Labels:    map[string]string{"env": "bench"},
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "ghcr.io/example/bench-app",
				Tag:        "1.0.0",
				PullPolicy: "IfNotPresent",
			},
			Replicas: platformv1alpha1.ReplicasSpec{Min: 3, Max: 10},
			Service:  platformv1alpha1.ServiceSpec{Port: 8080, Type: "ClusterIP"},
			Configuration: map[string]string{
				"APP_ENV":   "bench",
				"LOG_LEVEL": "info",
				"DB_HOST":   "postgres",
			},
			Autoscaling: platformv1alpha1.AutoscalingSpec{
				Enabled:              true,
				TargetCPUUtilization: 70,
			},
			Gateway: platformv1alpha1.GatewaySpec{
				Enabled:    true,
				Host:       "bench.example.io",
				GatewayRef: "infra/gw",
				PathPrefix: "/api",
			},
			Observability: platformv1alpha1.ObservabilitySpec{Metrics: true},
			Security:      platformv1alpha1.SecuritySpec{NetworkPolicy: true},
			Resources: platformv1alpha1.ResourcesSpec{
				Requests: platformv1alpha1.ResourceSpec{CPU: "100m", Memory: "128Mi"},
				Limits:   platformv1alpha1.ResourceSpec{CPU: "500m", Memory: "256Mi"},
			},
			HealthChecks: platformv1alpha1.HealthChecksSpec{
				LivenessPath:  "/healthz",
				ReadinessPath: "/readyz",
				Port:          8080,
			},
			Rollout: platformv1alpha1.RolloutSpec{
				Strategy:       "RollingUpdate",
				MaxUnavailable: "25%",
				MaxSurge:       "25%",
			},
		},
	}
}

// BenchmarkBuildDesiredDeployment benchmarks the most complex sub-reconciler.
func BenchmarkBuildDesiredDeployment(b *testing.B) {
	app := newBenchApp("bench-deploy")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dep := buildDesiredDeployment(app)
		if dep == nil {
			b.Fatal("expected non-nil deployment")
		}
	}
}

// BenchmarkBuildDesiredService benchmarks the Service generator.
func BenchmarkBuildDesiredService(b *testing.B) {
	app := newBenchApp("bench-svc")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		svc := buildDesiredService(app)
		if svc == nil {
			b.Fatal("expected non-nil service")
		}
	}
}

// BenchmarkBuildDesiredHPA benchmarks the HPA generator.
func BenchmarkBuildDesiredHPA(b *testing.B) {
	app := newBenchApp("bench-hpa")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		hpa := buildDesiredHPA(app)
		if hpa == nil {
			b.Fatal("expected non-nil HPA")
		}
	}
}

// BenchmarkBuildDesiredHTTPRoute benchmarks the HTTPRoute generator.
func BenchmarkBuildDesiredHTTPRoute(b *testing.B) {
	app := newBenchApp("bench-route")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		route := buildDesiredHTTPRoute(app)
		if route == nil {
			b.Fatal("expected non-nil HTTPRoute")
		}
	}
}

// BenchmarkBuildDesiredNetworkPolicy benchmarks the NetworkPolicy generator.
func BenchmarkBuildDesiredNetworkPolicy(b *testing.B) {
	app := newBenchApp("bench-np")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		np := buildDesiredNetworkPolicy(app)
		if np == nil {
			b.Fatal("expected non-nil NetworkPolicy")
		}
	}
}

// BenchmarkBuildDesiredPDB benchmarks the PDB generator.
func BenchmarkBuildDesiredPDB(b *testing.B) {
	app := newBenchApp("bench-pdb")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		pdb := buildDesiredPDB(app)
		if pdb == nil {
			b.Fatal("expected non-nil PDB")
		}
	}
}

// BenchmarkCommonLabels benchmarks label generation.
func BenchmarkCommonLabels(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		labels := CommonLabels("bench-app")
		if len(labels) != 3 {
			b.Fatalf("expected 3 labels, got %d", len(labels))
		}
	}
}

// BenchmarkMergeLabels benchmarks merging 3 label maps.
func BenchmarkMergeLabels(b *testing.B) {
	l1 := CommonLabels("app")
	l2 := map[string]string{"version": "1.0", "instance": "app"}
	l3 := map[string]string{"env": "prod", "team": "platform"}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		merged := MergeLabels(l1, l2, l3)
		if len(merged) < 5 {
			b.Fatal("expected at least 5 merged labels")
		}
	}
}

// BenchmarkBuildAllResources benchmarks computing all 6 child resources
// in sequence (the core reconcileResources work minus the API calls).
func BenchmarkBuildAllResources(b *testing.B) {
	app := newBenchApp("bench-all")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = buildDesiredDeployment(app)
		_ = buildDesiredService(app)
		_ = buildDesiredHPA(app)
		_ = buildDesiredHTTPRoute(app)
		_ = buildDesiredNetworkPolicy(app)
		_ = buildDesiredPDB(app)
	}
}
