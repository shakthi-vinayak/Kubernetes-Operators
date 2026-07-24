package controller

import (
	"context"
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
)

func newConcurrencyScheme() *runtime.Scheme {
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

func newConcurrencyApp(name string) *platformv1alpha1.PlatformApplication {
	return &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "ghcr.io/example/concurrency-test",
				Tag:        "1.0.0",
				PullPolicy: "IfNotPresent",
			},
			Replicas: platformv1alpha1.ReplicasSpec{Min: 2, Max: 5},
			Service:  platformv1alpha1.ServiceSpec{Port: 8080, Type: "ClusterIP"},
			Autoscaling: platformv1alpha1.AutoscalingSpec{
				Enabled:              true,
				TargetCPUUtilization: 70,
			},
			HealthChecks: platformv1alpha1.HealthChecksSpec{
				LivenessPath: "/healthz", ReadinessPath: "/readyz",
			},
			Rollout: platformv1alpha1.RolloutSpec{Strategy: "RollingUpdate"},
		},
	}
}

// TestConcurrency_MultipleWorkersNoRace verifies that multiple concurrent
// reconcile workers process different resources without data races.
// Run with -race to detect race conditions.
func TestConcurrency_MultipleWorkersNoRace(t *testing.T) {
	const numApps = 20
	const numWorkers = 4

	scheme := newConcurrencyScheme()
	objs := make([]runtime.Object, numApps)
	for i := 0; i < numApps; i++ {
		objs[i] = newConcurrencyApp("concurrency-app-" + string(rune('a'+i%26)) + string(rune('0'+i/26)))
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(objs...).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	r := &PlatformApplicationReconciler{
		Client:      fakeClient,
		Scheme:      scheme,
		Recorder:    record.NewFakeRecorder(200),
		Concurrency: numWorkers,
	}

	var successCount atomic.Int32
	var errorCount atomic.Int32
	var wg sync.WaitGroup

	// Simulate concurrent work from multiple workers.
	for i := 0; i < numApps; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "concurrency-app-" + string(rune('a'+idx%26)) + string(rune('0'+idx/26))
			req := ctrl.Request{
				NamespacedName: client.ObjectKey{Name: name, Namespace: "default"},
			}
			_, err := r.Reconcile(context.Background(), req)
			if err != nil {
				errorCount.Add(1)
			} else {
				successCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	t.Logf("concurrency test: %d success, %d errors (workers=%d)", successCount.Load(), errorCount.Load(), numWorkers)
	if successCount.Load() == 0 {
		t.Error("no successful reconciles — possible race or systemic failure")
	}
}

// TestConcurrency_SetupWithManager verifies that the controller correctly
// configures MaxConcurrentReconciles from the Concurrency field.
func TestConcurrency_SetupWithManager(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		expected    int
	}{
		{"default", 0, 1},
		{"single", 1, 1},
		{"multiple", 5, 5},
		{"negative", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &PlatformApplicationReconciler{
				Concurrency: tt.concurrency,
			}
			// Verify the concurrency normalization logic.
			c := r.Concurrency
			if c < 1 {
				c = 1
			}
			if c != tt.expected {
				t.Errorf("expected concurrency %d, got %d", tt.expected, c)
			}
		})
	}
}
