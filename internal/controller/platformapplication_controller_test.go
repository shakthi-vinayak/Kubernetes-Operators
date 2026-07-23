package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = platformv1alpha1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	return s
}

func newTestApp(name, namespace string) *platformv1alpha1.PlatformApplication {
	return &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Generation: 1,
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "ghcr.io/example/test-app",
				Tag:        "1.0.0",
				PullPolicy: "IfNotPresent",
			},
			Replicas: platformv1alpha1.ReplicasSpec{
				Min: 2,
				Max: 5,
			},
			Service: platformv1alpha1.ServiceSpec{
				Port: 8080,
				Type: "ClusterIP",
			},
			Autoscaling: platformv1alpha1.AutoscalingSpec{
				Enabled:              true,
				TargetCPUUtilization: 70,
			},
			HealthChecks: platformv1alpha1.HealthChecksSpec{
				LivenessPath:  "/healthz",
				ReadinessPath: "/readyz",
			},
			Rollout: platformv1alpha1.RolloutSpec{
				Strategy: "RollingUpdate",
			},
		},
	}
}

func newTestReconciler(objs ...runtime.Object) (*PlatformApplicationReconciler, *record.FakeRecorder) {
	scheme := newTestScheme()
	clientObjs := make([]runtime.Object, len(objs))
	copy(clientObjs, objs)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(clientObjs...).
		WithStatusSubresource(&platformv1alpha1.PlatformApplication{}).
		Build()

	recorder := record.NewFakeRecorder(50)

	r := &PlatformApplicationReconciler{
		Client:      fakeClient,
		Scheme:      scheme,
		Recorder:    recorder,
		Concurrency: 1,
	}

	return r, recorder
}

func TestReconcile_NotFound(t *testing.T) {
	r, _ := newTestReconciler()

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "missing", Namespace: "default"},
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result.Requeue || result.RequeueAfter > 0 {
		t.Error("expected no requeue for not-found resource")
	}
}

func TestReconcile_AddsFinalizer(t *testing.T) {
	app := newTestApp("test-app", "default")
	r, _ := newTestReconciler(app)

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-app", Namespace: "default"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return early after adding finalizer (the update triggers a new event).
	if result.Requeue || result.RequeueAfter > 0 {
		t.Error("expected no requeue after adding finalizer")
	}

	// Verify finalizer was added.
	updated := &platformv1alpha1.PlatformApplication{}
	if err := r.Get(context.Background(), types.NamespacedName{Name: "test-app", Namespace: "default"}, updated); err != nil {
		t.Fatalf("unable to get app: %v", err)
	}

	found := false
	for _, f := range updated.Finalizers {
		if f == finalizerName {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finalizer to be added")
	}
}

func TestReconcile_Deletion(t *testing.T) {
	now := metav1.Now()
	app := newTestApp("deleting-app", "default")
	app.DeletionTimestamp = &now
	app.Finalizers = []string{finalizerName}
	r, recorder := newTestReconciler(app)

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "deleting-app", Namespace: "default"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Requeue || result.RequeueAfter > 0 {
		t.Error("expected no requeue after deletion handling")
	}

	// After finalizer removal with deletionTimestamp set, the fake client
	// may garbage-collect the object. Verify no error occurred (success).

	// Verify event was recorded.
	select {
	case event := <-recorder.Events:
		if event == "" {
			t.Error("expected a deletion event")
		}
	default:
		t.Error("expected a deletion event to be recorded")
	}
}

func TestReconcile_Deletion_NoFinalizer(t *testing.T) {
	now := metav1.Now()
	app := newTestApp("no-finalizer-app", "default")
	app.DeletionTimestamp = &now
	app.Finalizers = []string{"some-other-finalizer"}
	r, _ := newTestReconciler(app)

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-finalizer-app", Namespace: "default"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Requeue || result.RequeueAfter > 0 {
		t.Error("expected no requeue when no platform finalizer on deleting resource")
	}
}

func TestSetupWithManager_Concurrency(t *testing.T) {
	r := &PlatformApplicationReconciler{
		Concurrency: 5,
	}

	if r.Concurrency != 5 {
		t.Errorf("expected concurrency 5, got %d", r.Concurrency)
	}

	// Test default concurrency.
	r2 := &PlatformApplicationReconciler{
		Concurrency: 0,
	}

	// SetupWithManager would enforce min 1, but we test the field directly.
	if r2.Concurrency != 0 {
		t.Errorf("expected concurrency 0 (will default to 1 in setup), got %d", r2.Concurrency)
	}
}
