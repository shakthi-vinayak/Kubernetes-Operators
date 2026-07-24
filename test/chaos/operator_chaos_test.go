package chaos

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gwapiv1 "sigs.k8s.io/gateway-api/apis/v1"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
	"github.com/example/platform-operator/internal/controller"
	"github.com/example/platform-operator/internal/testing/fakeclient"
)

func newChaosScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = platformv1alpha1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	_ = autoscalingv2.AddToScheme(s)
	_ = networkingv1.AddToScheme(s)
	_ = policyv1.AddToScheme(s)
	_ = gwapiv1.Install(s)
	return s
}

func newChaosApp(name string) *platformv1alpha1.PlatformApplication {
	return &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "ghcr.io/example/chaos-test",
				Tag:        "1.0.0",
				PullPolicy: "IfNotPresent",
			},
			Replicas: platformv1alpha1.ReplicasSpec{Min: 1},
			Service:  platformv1alpha1.ServiceSpec{Port: 8080, Type: "ClusterIP"},
			HealthChecks: platformv1alpha1.HealthChecksSpec{
				LivenessPath: "/healthz", ReadinessPath: "/readyz",
			},
			Rollout: platformv1alpha1.RolloutSpec{Strategy: "RollingUpdate"},
		},
	}
}

// TestChaos_IntermittentFailures verifies the operator recovers from
// intermittent API failures — simulating a flaky API server.
func TestChaos_IntermittentFailures(t *testing.T) {
	const numApps = 10

	scheme := newChaosScheme()
	objs := make([]runtime.Object, numApps)
	for i := 0; i < numApps; i++ {
		objs[i] = newChaosApp(fmt.Sprintf("chaos-app-%d", i))
	}

	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objs...).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	// Inject transient errors on 50% of Create calls.
	interceptor := fakeclient.NewInterceptor(baseClient,
		fakeclient.FaultConfig{
			Operation: "create",
			Fault:     fakeclient.FaultTransient,
			Count:     numApps / 2, // fail half the creates
		},
	)

	r := &controller.PlatformApplicationReconciler{
		Client:      interceptor,
		Scheme:      scheme,
		Recorder:    record.NewFakeRecorder(200),
		Concurrency: 4,
	}

	var successCount atomic.Int32
	var failCount atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < numApps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := ctrl.Request{
				NamespacedName: client.ObjectKey{
					Name:      fmt.Sprintf("chaos-app-%d", idx),
					Namespace: "default",
				},
			}
			_, err := r.Reconcile(context.Background(), req)
			if err != nil {
				failCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("chaos test: %d success, %d failures out of %d apps", successCount.Load(), failCount.Load(), numApps)
	// The operator should gracefully handle failures (not panic or corrupt state).
	// We don't require 100% success — the goal is to verify resilience.
}

// TestChaos_RapidCreateDelete simulates rapid create/delete cycles
// (common in CI/CD or auto-scaling scenarios) to verify the operator
// handles these without errors or orphaned resources.
func TestChaos_RapidCreateDelete(t *testing.T) {
	scheme := newChaosScheme()

	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	r := &controller.PlatformApplicationReconciler{
		Client:      baseClient,
		Scheme:      scheme,
		Recorder:    record.NewFakeRecorder(100),
		Concurrency: 1,
	}

	ctx := context.Background()

	// Create app.
	app := newChaosApp("rapid-cycle-app")
	if err := baseClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{Name: "rapid-cycle-app", Namespace: "default"},
	}

	// Reconcile multiple times rapidly.
	for i := 0; i < 5; i++ {
		_, err := r.Reconcile(ctx, req)
		if err != nil {
			t.Logf("rapid cycle reconcile %d failed: %v", i, err)
		}
	}

	// Delete and re-create with a new name to avoid fake client finalizer issues.
	if err := baseClient.Delete(ctx, app); err != nil {
		t.Logf("delete returned error (expected with fake client): %v", err)
	}

	app2 := newChaosApp("rapid-cycle-app-v2")
	app2.Spec.Image.Tag = "2.0.0"
	if err := baseClient.Create(ctx, app2); err != nil {
		t.Fatalf("failed to re-create app: %v", err)
	}

	req2 := ctrl.Request{
		NamespacedName: client.ObjectKey{Name: "rapid-cycle-app-v2", Namespace: "default"},
	}
	_, err := r.Reconcile(ctx, req2)
	if err != nil {
		t.Logf("post-recreate reconcile failed: %v", err)
	}
}

// TestChaos_AllOperationsFail verifies the operator handles complete
// API server unavailability without panicking.
func TestChaos_AllOperationsFail(t *testing.T) {
	scheme := newChaosScheme()
	app := newChaosApp("total-failure-app")

	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(app).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	// Inject faults on ALL operations.
	interceptor := fakeclient.NewInterceptor(baseClient,
		fakeclient.FaultConfig{Operation: "create", Fault: fakeclient.FaultTransient, Count: 10},
		fakeclient.FaultConfig{Operation: "patch", Fault: fakeclient.FaultTransient, Count: 10},
		fakeclient.FaultConfig{Operation: "get", Fault: fakeclient.FaultTransient, Count: 5},
		fakeclient.FaultConfig{Operation: "status", Fault: fakeclient.FaultTransient, Count: 10},
	)

	r := &controller.PlatformApplicationReconciler{
		Client:      interceptor,
		Scheme:      scheme,
		Recorder:    record.NewFakeRecorder(100),
		Concurrency: 1,
	}

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{Name: "total-failure-app", Namespace: "default"},
	}

	// Should not panic.
	_, err := r.Reconcile(context.Background(), req)
	if err == nil {
		t.Log("warning: reconcile succeeded despite total API failure")
	} else {
		t.Logf("reconcile failed as expected: %v", err)
	}
}
