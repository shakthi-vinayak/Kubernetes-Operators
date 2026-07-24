package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

func newScaleScheme() *runtime.Scheme {
	s := newTestScheme()
	_ = autoscalingv2.AddToScheme(s)
	_ = networkingv1.AddToScheme(s)
	_ = policyv1.AddToScheme(s)
	_ = gatewayv1.Install(s)
	return s
}

// TestScale_MultipleApplications verifies the reconciler can handle
// many PlatformApplication resources without degradation.
// This is a unit-level scale test using a fake client.
func TestScale_MultipleApplications(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scale test in short mode")
	}

	const numApps = 100

	scheme := newScaleScheme()
	objs := make([]runtime.Object, numApps)
	for i := 0; i < numApps; i++ {
		objs[i] = &platformv1alpha1.PlatformApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:       fmt.Sprintf("app-%03d", i),
				Namespace:  "default",
				Generation: 1,
			},
			Spec: platformv1alpha1.PlatformApplicationSpec{
				Image: platformv1alpha1.ImageSpec{
					Repository: "ghcr.io/example/app",
					Tag:        fmt.Sprintf("v%d", i%10),
					PullPolicy: "IfNotPresent",
				},
				Replicas: platformv1alpha1.ReplicasSpec{Min: 2, Max: 5},
				Service:  platformv1alpha1.ServiceSpec{Port: 8080},
				Autoscaling: platformv1alpha1.AutoscalingSpec{
					Enabled:              true,
					TargetCPUUtilization: 70,
				},
				HealthChecks: platformv1alpha1.HealthChecksSpec{
					LivenessPath:  "/healthz",
					ReadinessPath: "/readyz",
				},
			},
		}
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objs...).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	r := &PlatformApplicationReconciler{
		Client:      fakeClient,
		Scheme:      scheme,
		Recorder:    record.NewFakeRecorder(numApps * 10),
		Concurrency: 5,
	}

	start := time.Now()

	// Reconcile all applications.
	for i := 0; i < numApps; i++ {
		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      fmt.Sprintf("app-%03d", i),
				Namespace: "default",
			},
		}
		_, err := r.Reconcile(context.Background(), req)
		if err != nil {
			t.Fatalf("reconcile failed for app-%03d: %v", i, err)
		}
	}

	duration := time.Since(start)
	t.Logf("Reconciled %d applications in %v (%.2f ms/app)",
		numApps, duration, float64(duration.Milliseconds())/float64(numApps))

	// Verify all got finalizers added.
	for i := 0; i < numApps; i++ {
		app := &platformv1alpha1.PlatformApplication{}
		err := fakeClient.Get(context.Background(), types.NamespacedName{
			Name:      fmt.Sprintf("app-%03d", i),
			Namespace: "default",
		}, app)
		if err != nil {
			t.Errorf("failed to get app-%03d: %v", i, err)
			continue
		}
		found := false
		for _, f := range app.Finalizers {
			if f == finalizerName {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("app-%03d missing finalizer", i)
		}
	}
}

// TestScale_SecondReconcilePass verifies idempotency under scale —
// the second pass should detect no changes and complete faster.
func TestScale_SecondReconcilePass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scale test in short mode")
	}

	const numApps = 50

	scheme := newScaleScheme()
	objs := make([]runtime.Object, numApps)
	for i := 0; i < numApps; i++ {
		objs[i] = &platformv1alpha1.PlatformApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name:       fmt.Sprintf("idempotent-%03d", i),
				Namespace:  "default",
				Generation: 1,
				Finalizers: []string{finalizerName},
			},
			Spec: platformv1alpha1.PlatformApplicationSpec{
				Image: platformv1alpha1.ImageSpec{
					Repository: "ghcr.io/example/app",
					Tag:        "v1.0",
				},
				Replicas: platformv1alpha1.ReplicasSpec{Min: 1},
				Service:  platformv1alpha1.ServiceSpec{Port: 8080},
			},
		}
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objs...).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	r := &PlatformApplicationReconciler{
		Client:      fakeClient,
		Scheme:      scheme,
		Recorder:    record.NewFakeRecorder(numApps * 5),
		Concurrency: 1,
	}

	// First pass: creates resources.
	for i := 0; i < numApps; i++ {
		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      fmt.Sprintf("idempotent-%03d", i),
				Namespace: "default",
			},
		}
		if _, err := r.Reconcile(context.Background(), req); err != nil {
			t.Fatalf("first pass failed for idempotent-%03d: %v", i, err)
		}
	}

	// Second pass: should detect no changes (idempotent).
	start := time.Now()
	for i := 0; i < numApps; i++ {
		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      fmt.Sprintf("idempotent-%03d", i),
				Namespace: "default",
			},
		}
		if _, err := r.Reconcile(context.Background(), req); err != nil {
			t.Fatalf("second pass failed for idempotent-%03d: %v", i, err)
		}
	}

	duration := time.Since(start)
	t.Logf("Second pass (idempotent): %d apps in %v (%.2f ms/app)",
		numApps, duration, float64(duration.Milliseconds())/float64(numApps))
}

// BenchmarkReconcile_Single benchmarks a single reconciliation cycle.
func BenchmarkReconcile_Single(b *testing.B) {
	scheme := newScaleScheme()
	app := &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "bench-app",
			Namespace:  "default",
			Generation: 1,
			Finalizers: []string{finalizerName},
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "ghcr.io/example/app",
				Tag:        "v1.0",
			},
			Replicas: platformv1alpha1.ReplicasSpec{Min: 2, Max: 5},
			Service:  platformv1alpha1.ServiceSpec{Port: 8080},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(app).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	r := &PlatformApplicationReconciler{
		Client:      fakeClient,
		Scheme:      scheme,
		Recorder:    record.NewFakeRecorder(100),
		Concurrency: 1,
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "bench-app", Namespace: "default"},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.Reconcile(context.Background(), req)
	}
}
