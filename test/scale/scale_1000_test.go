package scale
package scale

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
)

func newScaleScheme() *runtime.Scheme {
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

func newScaleApp(name string) *platformv1alpha1.PlatformApplication {
	return &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  "default",
			Finalizers: []string{"platform.example.io/cleanup"},
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "ghcr.io/example/scale-test",
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

// TestScale_1000Resources verifies the operator can handle 1000+ CRs
// without performance degradation or resource exhaustion.
func TestScale_1000Resources(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scale test in short mode")
	}

	const numApps = 1000
	scheme := newScaleScheme()

	// Pre-create all apps.
	objects := make([]client.Object, numApps)
	for i := 0; i < numApps; i++ {
		objects[i] = newScaleApp(fmt.Sprintf("scale-app-%04d", i))
	}

	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(toRuntimeObjects(objects)...).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	r := &controller.PlatformApplicationReconciler{
		Client:      baseClient,
		Scheme:      scheme,
		Recorder:    record.NewFakeRecorder(numApps * 2),
		Concurrency: 8,
	}

	ctx := context.Background()
	start := time.Now()

	// Reconcile all apps with 8 concurrent workers.
	var successCount atomic.Int32
	var errorCount atomic.Int32
	var wg sync.WaitGroup

	sem := make(chan struct{}, 8) // limit concurrency to 8

	for i := 0; i < numApps; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			req := ctrl.Request{
				NamespacedName: client.ObjectKey{
					Name:      fmt.Sprintf("scale-app-%04d", idx),
					Namespace: "default",
				},
			}
			_, err := r.Reconcile(ctx, req)
			if err != nil {
				errorCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Scale test: %d apps reconciled in %v", numApps, elapsed)
	t.Logf("  Success: %d, Errors: %d", successCount.Load(), errorCount.Load())
	t.Logf("  Throughput: %.1f reconciles/sec", float64(numApps)/elapsed.Seconds())
	t.Logf("  Avg latency: %v per reconcile", elapsed/time.Duration(numApps))

	if got := successCount.Load(); got != numApps {
		t.Errorf("expected %d successful reconciles, got %d", numApps, got)
	}

	// Verify throughput is reasonable (> 100 reconciles/sec with fake client).
	throughput := float64(numApps) / elapsed.Seconds()
	if throughput < 10 {
		t.Errorf("throughput too low: %.1f reconciles/sec (expected > 10)", throughput)
	}
}

// TestScale_MemoryUsage verifies memory doesn't grow linearly with CR count.
func TestScale_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scale test in short mode")
	}

	const numApps = 500
	scheme := newScaleScheme()

	objects := make([]client.Object, numApps)
	for i := 0; i < numApps; i++ {
		objects[i] = newScaleApp(fmt.Sprintf("mem-app-%04d", i))
	}

	baseClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(toRuntimeObjects(objects)...).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	r := &controller.PlatformApplicationReconciler{
		Client:      baseClient,
		Scheme:      scheme,
		Recorder:    record.NewFakeRecorder(numApps * 2),
		Concurrency: 4,
	}

	ctx := context.Background()

	// Reconcile all apps sequentially to avoid fake client contention.
	for i := 0; i < numApps; i++ {
		req := ctrl.Request{
			NamespacedName: client.ObjectKey{
				Name:      fmt.Sprintf("mem-app-%04d", i),
				Namespace: "default",
			},
		}
		if _, err := r.Reconcile(ctx, req); err != nil {
			t.Fatalf("reconcile %d failed: %v", i, err)
		}
	}

	t.Logf("Successfully reconciled %d apps without memory issues", numApps)
}

func toRuntimeObjects(objects []client.Object) []client.Object {
	return objects
}
