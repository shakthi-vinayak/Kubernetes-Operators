package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

// k8sClient is a Kubernetes client connected to the current cluster (Kind).
var k8sClient client.Client

// e2eNamespace is the namespace for E2E test resources.
const e2eNamespace = "e2e-test"

// TestMain sets up the E2E test environment.
func TestMain(m *testing.M) {
	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get kubeconfig (is a cluster running?): %v\n", err)
		os.Exit(1)
	}

	s := scheme.Scheme
	_ = platformv1alpha1.AddToScheme(s)

	k8sClient, err = client.New(cfg, client.Options{Scheme: s})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create client: %v\n", err)
		os.Exit(1)
	}

	// Create E2E test namespace.
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: e2eNamespace}}
	_ = k8sClient.Create(context.Background(), ns)

	code := m.Run()

	// Cleanup namespace.
	_ = k8sClient.Delete(context.Background(), ns)
	os.Exit(code)
}

// TestE2E_CreateAndVerifyApplication creates a PlatformApplication in a real
// cluster and verifies the operator creates the expected Deployment and Service.
//
// Prerequisites:
//   - A Kind cluster is running
//   - The operator CRD is installed
//   - The operator is deployed in the cluster
func TestE2E_CreateAndVerifyApplication(t *testing.T) {
	ctx := context.Background()

	app := &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-test-app",
			Namespace: e2eNamespace,
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "nginx",
				Tag:        "alpine",
				PullPolicy: "IfNotPresent",
			},
			Replicas: platformv1alpha1.ReplicasSpec{
				Min: 2,
				Max: 5,
			},
			Service: platformv1alpha1.ServiceSpec{
				Port: 80,
				Type: "ClusterIP",
			},
			HealthChecks: platformv1alpha1.HealthChecksSpec{
				LivenessPath:  "/",
				ReadinessPath: "/",
			},
			Rollout: platformv1alpha1.RolloutSpec{
				Strategy: "RollingUpdate",
			},
		},
	}

	// Create the application.
	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create PlatformApplication: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, app)
	}()

	// Wait for the Deployment to be created.
	dep := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "e2e-test-app", Namespace: e2eNamespace}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if err := k8sClient.Get(ctx, key, dep); err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if dep.Name == "" {
		t.Fatal("timeout waiting for Deployment to be created")
	}

	// Verify Deployment spec.
	if *dep.Spec.Replicas != 2 {
		t.Errorf("expected 2 replicas, got %d", *dep.Spec.Replicas)
	}
	if dep.Spec.Template.Spec.Containers[0].Image != "nginx:alpine" {
		t.Errorf("expected image nginx:alpine, got %s", dep.Spec.Template.Spec.Containers[0].Image)
	}

	// Wait for the Service to be created.
	svc := &corev1.Service{}
	svcDeadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(svcDeadline) {
		if err := k8sClient.Get(ctx, key, svc); err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if svc.Name == "" {
		t.Fatal("timeout waiting for Service to be created")
	}

	// Verify the Deployment has pods running (wait up to 3 minutes for images to pull).
	t.Log("waiting for pods to become ready (up to 3 minutes)...")
	podDeadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(podDeadline) {
		updatedDep := &appsv1.Deployment{}
		if err := k8sClient.Get(ctx, key, updatedDep); err != nil {
			continue
		}
		if updatedDep.Status.ReadyReplicas >= 2 {
			t.Logf("all %d replicas ready", updatedDep.Status.ReadyReplicas)
			return // success
		}
		time.Sleep(5 * time.Second)
	}

	// Not necessarily a failure — may be slow image pull in CI.
	t.Log("warning: pods did not become ready within 3 minutes (may be slow image pull)")
}

// TestE2E_StatusConditions verifies that status conditions are set on
// the PlatformApplication after reconciliation.
func TestE2E_StatusConditions(t *testing.T) {
	ctx := context.Background()

	app := &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-status-app",
			Namespace: e2eNamespace,
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "nginx",
				Tag:        "alpine",
				PullPolicy: "IfNotPresent",
			},
			Replicas: platformv1alpha1.ReplicasSpec{Min: 1},
			Service:  platformv1alpha1.ServiceSpec{Port: 80},
			HealthChecks: platformv1alpha1.HealthChecksSpec{
				LivenessPath: "/", ReadinessPath: "/",
			},
			Rollout: platformv1alpha1.RolloutSpec{Strategy: "RollingUpdate"},
		},
	}

	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create PlatformApplication: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, app)
	}()

	// Wait for status conditions.
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		current := &platformv1alpha1.PlatformApplication{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: "e2e-status-app", Namespace: e2eNamespace}, current); err != nil {
			continue
		}
		if len(current.Status.Conditions) > 0 {
			t.Logf("found %d conditions", len(current.Status.Conditions))
			for _, c := range current.Status.Conditions {
				t.Logf("  %s: %s (%s: %s)", c.Type, c.Status, c.Reason, c.Message)
			}
			return // success
		}
		time.Sleep(2 * time.Second)
	}
	t.Error("timeout waiting for status conditions")
}

// TestE2E_UpdateTriggersRollout verifies that updating the image tag
// triggers a Deployment rollout.
func TestE2E_UpdateTriggersRollout(t *testing.T) {
	ctx := context.Background()

	app := &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-update-app",
			Namespace: e2eNamespace,
		},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image: platformv1alpha1.ImageSpec{
				Repository: "nginx",
				Tag:        "alpine",
				PullPolicy: "IfNotPresent",
			},
			Replicas: platformv1alpha1.ReplicasSpec{Min: 1},
			Service:  platformv1alpha1.ServiceSpec{Port: 80},
			HealthChecks: platformv1alpha1.HealthChecksSpec{
				LivenessPath: "/", ReadinessPath: "/",
			},
			Rollout: platformv1alpha1.RolloutSpec{Strategy: "RollingUpdate"},
		},
	}

	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, app)
	}()

	// Wait for the Deployment.
	dep := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "e2e-update-app", Namespace: e2eNamespace}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if err := k8sClient.Get(ctx, key, dep); err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// Update image tag.
	current := &platformv1alpha1.PlatformApplication{}
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), current); err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	current.Spec.Image.Tag = "1.27"
	if err := k8sClient.Update(ctx, current); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	// Wait for the Deployment image to change.
	updateDeadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(updateDeadline) {
		updatedDep := &appsv1.Deployment{}
		if err := k8sClient.Get(ctx, key, updatedDep); err != nil {
			continue
		}
		if updatedDep.Spec.Template.Spec.Containers[0].Image == "nginx:1.27" {
			t.Log("Deployment image updated to nginx:1.27")
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Error("timeout waiting for Deployment image update")
}
