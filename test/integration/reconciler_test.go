package integration

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

// TestIntegration_CreateApplication verifies that creating a PlatformApplication
// causes the controller to create the expected child resources (Deployment, Service).
func TestIntegration_CreateApplication(t *testing.T) {
	ctx := context.Background()
	app := newTestApp("create-test")

	// Create the PlatformApplication.
	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create PlatformApplication: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, app)
	}()

	// Wait for the Deployment to be created by the controller.
	dep := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "create-test", Namespace: testNamespace}
	if err := waitForResource(ctx, k8sClient, key, dep, 15*time.Second); err != nil {
		t.Fatalf("timeout waiting for Deployment to be created: %v", err)
	}

	// Verify Deployment spec.
	if *dep.Spec.Replicas != 2 {
		t.Errorf("expected 2 replicas, got %d", *dep.Spec.Replicas)
	}
	expectedImage := "ghcr.io/example/test-app:1.0.0"
	if dep.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, dep.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify labels.
	if dep.Labels["app.kubernetes.io/name"] != "create-test" {
		t.Errorf("expected label app.kubernetes.io/name=create-test, got %q", dep.Labels["app.kubernetes.io/name"])
	}
	if dep.Labels["app.kubernetes.io/managed-by"] != "platform-operator" {
		t.Errorf("expected managed-by label, got %q", dep.Labels["app.kubernetes.io/managed-by"])
	}

	// Wait for the Service to be created.
	svc := &corev1.Service{}
	if err := waitForResource(ctx, k8sClient, key, svc, 15*time.Second); err != nil {
		t.Fatalf("timeout waiting for Service to be created: %v", err)
	}

	if svc.Spec.Ports[0].Port != 8080 {
		t.Errorf("expected service port 8080, got %d", svc.Spec.Ports[0].Port)
	}
	if string(svc.Spec.Type) != "ClusterIP" {
		t.Errorf("expected ClusterIP, got %s", svc.Spec.Type)
	}

	// Verify owner references on the Deployment.
	if len(dep.OwnerReferences) == 0 {
		t.Error("expected Deployment to have owner references")
	} else if dep.OwnerReferences[0].Kind != "PlatformApplication" {
		t.Errorf("expected owner kind PlatformApplication, got %s", dep.OwnerReferences[0].Kind)
	}
}

// TestIntegration_FinalizerAdded verifies that the controller adds a finalizer
// to newly created PlatformApplication resources.
func TestIntegration_FinalizerAdded(t *testing.T) {
	ctx := context.Background()
	app := newTestApp("finalizer-test")

	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create PlatformApplication: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, app)
	}()

	// Wait for the finalizer to be added.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		current := &platformv1alpha1.PlatformApplication{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), current); err != nil {
			continue
		}
		for _, f := range current.Finalizers {
			if f == "platform.example.io/cleanup" {
				return // success
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Error("timeout waiting for finalizer to be added")
}

// TestIntegration_StatusConditions verifies that the controller sets status
// conditions on the PlatformApplication after reconciliation.
func TestIntegration_StatusConditions(t *testing.T) {
	ctx := context.Background()
	app := newTestApp("status-test")

	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create PlatformApplication: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, app)
	}()

	// Wait for status conditions to appear.
	deadline := time.Now().Add(15 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		current := &platformv1alpha1.PlatformApplication{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), current); err != nil {
			continue
		}
		if len(current.Status.Conditions) > 0 {
			found = true
			// Should have Progressing condition.
			for _, c := range current.Status.Conditions {
				if c.Type == "Progressing" {
					return // success
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !found {
		t.Error("timeout waiting for status conditions to be set")
	} else {
		t.Error("expected Progressing condition but not found")
	}
}

// TestIntegration_ObservedGeneration verifies that the controller sets
// observedGeneration after reconciliation.
func TestIntegration_ObservedGeneration(t *testing.T) {
	ctx := context.Background()
	app := newTestApp("observed-gen-test")

	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create PlatformApplication: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, app)
	}()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		current := &platformv1alpha1.PlatformApplication{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), current); err != nil {
			continue
		}
		if current.Status.ObservedGeneration > 0 {
			if current.Status.ObservedGeneration >= current.Generation {
				return // success
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Error("timeout waiting for observedGeneration to be updated")
}

// TestIntegration_UpdateApplication verifies that updating the PlatformApplication
// spec causes the controller to update child resources accordingly.
func TestIntegration_UpdateApplication(t *testing.T) {
	ctx := context.Background()
	app := newTestApp("update-test")

	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create PlatformApplication: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, app)
	}()

	// Wait for the Deployment to be created.
	dep := &appsv1.Deployment{}
	key := types.NamespacedName{Name: "update-test", Namespace: testNamespace}
	if err := waitForResource(ctx, k8sClient, key, dep, 15*time.Second); err != nil {
		t.Fatalf("timeout waiting for Deployment: %v", err)
	}

	// Update the spec: change replicas from 2 to 4.
	current := &platformv1alpha1.PlatformApplication{}
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), current); err != nil {
		t.Fatalf("failed to get app for update: %v", err)
	}
	current.Spec.Replicas.Min = 4
	if err := k8sClient.Update(ctx, current); err != nil {
		t.Fatalf("failed to update app: %v", err)
	}

	// Wait for the Deployment replicas to change.
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		updatedDep := &appsv1.Deployment{}
		if err := k8sClient.Get(ctx, key, updatedDep); err != nil {
			continue
		}
		if *updatedDep.Spec.Replicas == 4 {
			return // success
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Error("timeout waiting for Deployment replicas to be updated to 4")
}

// TestIntegration_DeleteApplication verifies that deleting a PlatformApplication
// triggers the finalizer and the resource is eventually removed.
func TestIntegration_DeleteApplication(t *testing.T) {
	ctx := context.Background()
	app := newTestApp("delete-test")

	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create PlatformApplication: %v", err)
	}

	// Wait for finalizer to be added.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		current := &platformv1alpha1.PlatformApplication{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), current); err != nil {
			continue
		}
		for _, f := range current.Finalizers {
			if f == "platform.example.io/cleanup" {
				goto deleteNow
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("finalizer was never added, cannot test deletion")

deleteNow:
	// Delete the application.
	if err := k8sClient.Delete(ctx, app); err != nil {
		t.Fatalf("failed to delete PlatformApplication: %v", err)
	}

	// Wait for the resource to be fully deleted (finalizer removed + GC).
	deadline = time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		current := &platformv1alpha1.PlatformApplication{}
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), current)
		if err != nil {
			return // success — resource is gone
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Error("timeout waiting for PlatformApplication to be deleted")
}

// TestIntegration_DeployedVersion verifies that the deployed version in status
// matches the image tag from the spec.
func TestIntegration_DeployedVersion(t *testing.T) {
	ctx := context.Background()
	app := newTestApp("version-test")

	if err := k8sClient.Create(ctx, app); err != nil {
		t.Fatalf("failed to create PlatformApplication: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, app)
	}()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		current := &platformv1alpha1.PlatformApplication{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(app), current); err != nil {
			continue
		}
		if current.Status.DeployedVersion == "1.0.0" {
			return // success
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Error("timeout waiting for deployedVersion to be set to 1.0.0")
}
