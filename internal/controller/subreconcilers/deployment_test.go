package subreconcilers

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

func TestBuildDesiredDeployment(t *testing.T) {
	tests := []struct {
		name               string
		app                *platformv1alpha1.PlatformApplication
		expectedReplicas   int32
		expectedImage      string
		expectedPort       int32
		expectedPullPolicy string
	}{
		{
			name: "basic application",
			app: &platformv1alpha1.PlatformApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: platformv1alpha1.PlatformApplicationSpec{
					Image: platformv1alpha1.ImageSpec{
						Repository: "ghcr.io/example/app",
						Tag:        "1.0.0",
						PullPolicy: "IfNotPresent",
					},
					Replicas: platformv1alpha1.ReplicasSpec{
						Min: 3,
						Max: 10,
					},
					Service: platformv1alpha1.ServiceSpec{
						Port: 8080,
					},
				},
			},
			expectedReplicas:   3,
			expectedImage:      "ghcr.io/example/app:1.0.0",
			expectedPort:       8080,
			expectedPullPolicy: "IfNotPresent",
		},
		{
			name: "single replica",
			app: &platformv1alpha1.PlatformApplication{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "single-app",
					Namespace: "default",
				},
				Spec: platformv1alpha1.PlatformApplicationSpec{
					Image: platformv1alpha1.ImageSpec{
						Repository: "ghcr.io/example/single",
						Tag:        "2.0.0",
					},
					Replicas: platformv1alpha1.ReplicasSpec{
						Min: 1,
					},
					Service: platformv1alpha1.ServiceSpec{
						Port: 3000,
					},
				},
			},
			expectedReplicas:   1,
			expectedImage:      "ghcr.io/example/single:2.0.0",
			expectedPort:       3000,
			expectedPullPolicy: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := buildDesiredDeployment(tt.app)

			// Verify name and namespace.
			if dep.Name != tt.app.Name {
				t.Errorf("expected name %q, got %q", tt.app.Name, dep.Name)
			}
			if dep.Namespace != tt.app.Namespace {
				t.Errorf("expected namespace %q, got %q", tt.app.Namespace, dep.Namespace)
			}

			// Verify replicas.
			if *dep.Spec.Replicas != tt.expectedReplicas {
				t.Errorf("expected replicas %d, got %d", tt.expectedReplicas, *dep.Spec.Replicas)
			}

			// Verify container image.
			container := dep.Spec.Template.Spec.Containers[0]
			if container.Image != tt.expectedImage {
				t.Errorf("expected image %q, got %q", tt.expectedImage, container.Image)
			}

			// Verify container port.
			if container.Ports[0].ContainerPort != tt.expectedPort {
				t.Errorf("expected port %d, got %d", tt.expectedPort, container.Ports[0].ContainerPort)
			}

			// Verify pull policy.
			if tt.expectedPullPolicy != "" && string(container.ImagePullPolicy) != tt.expectedPullPolicy {
				t.Errorf("expected pull policy %q, got %q", tt.expectedPullPolicy, container.ImagePullPolicy)
			}

			// Verify labels.
			if dep.Labels["app.kubernetes.io/name"] != tt.app.Name {
				t.Errorf("expected label app.kubernetes.io/name=%q, got %q", tt.app.Name, dep.Labels["app.kubernetes.io/name"])
			}
			if dep.Labels["app.kubernetes.io/managed-by"] != "platform-operator" {
				t.Errorf("expected label app.kubernetes.io/managed-by=platform-operator, got %q", dep.Labels["app.kubernetes.io/managed-by"])
			}

			// Verify security context.
			if dep.Spec.Template.Spec.SecurityContext == nil {
				t.Error("expected pod security context to be set")
			} else if !*dep.Spec.Template.Spec.SecurityContext.RunAsNonRoot {
				t.Error("expected RunAsNonRoot to be true")
			}

			// Verify container security context.
			if container.SecurityContext == nil {
				t.Error("expected container security context to be set")
			} else {
				if !*container.SecurityContext.ReadOnlyRootFilesystem {
					t.Error("expected ReadOnlyRootFilesystem to be true")
				}
				if *container.SecurityContext.AllowPrivilegeEscalation {
					t.Error("expected AllowPrivilegeEscalation to be false")
				}
			}

			// Verify probes.
			if container.LivenessProbe == nil {
				t.Error("expected liveness probe to be set")
			}
			if container.ReadinessProbe == nil {
				t.Error("expected readiness probe to be set")
			}
		})
	}
}

func TestCommonLabels(t *testing.T) {
	labels := CommonLabels("my-app")

	if labels["app.kubernetes.io/name"] != "my-app" {
		t.Errorf("expected name label 'my-app', got %q", labels["app.kubernetes.io/name"])
	}
	if labels["app.kubernetes.io/managed-by"] != "platform-operator" {
		t.Errorf("expected managed-by label, got %q", labels["app.kubernetes.io/managed-by"])
	}
	if labels["app.kubernetes.io/part-of"] != "platform" {
		t.Errorf("expected part-of label, got %q", labels["app.kubernetes.io/part-of"])
	}
}

func TestMergeLabels(t *testing.T) {
	base := map[string]string{"a": "1", "b": "2"}
	override := map[string]string{"b": "3", "c": "4"}

	result := MergeLabels(base, override)

	if result["a"] != "1" {
		t.Errorf("expected a=1, got %q", result["a"])
	}
	if result["b"] != "3" {
		t.Errorf("expected b=3 (overridden), got %q", result["b"])
	}
	if result["c"] != "4" {
		t.Errorf("expected c=4, got %q", result["c"])
	}
}
