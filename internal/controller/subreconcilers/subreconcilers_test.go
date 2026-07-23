package subreconcilers

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

func TestBuildDesiredService(t *testing.T) {
	tests := []struct {
		name         string
		app          *platformv1alpha1.PlatformApplication
		expectedPort int32
		expectedType string
	}{
		{
			name: "basic service",
			app: &platformv1alpha1.PlatformApplication{
				ObjectMeta: metav1.ObjectMeta{Name: "web-app", Namespace: "default"},
				Spec: platformv1alpha1.PlatformApplicationSpec{
					Image:    platformv1alpha1.ImageSpec{Repository: "ghcr.io/example/web", Tag: "1.0"},
					Replicas: platformv1alpha1.ReplicasSpec{Min: 2},
					Service:  platformv1alpha1.ServiceSpec{Port: 8080},
				},
			},
			expectedPort: 8080,
			expectedType: "ClusterIP",
		},
		{
			name: "load balancer service",
			app: &platformv1alpha1.PlatformApplication{
				ObjectMeta: metav1.ObjectMeta{Name: "api-gw", Namespace: "ingress"},
				Spec: platformv1alpha1.PlatformApplicationSpec{
					Image:    platformv1alpha1.ImageSpec{Repository: "ghcr.io/example/gw", Tag: "2.0"},
					Replicas: platformv1alpha1.ReplicasSpec{Min: 3},
					Service:  platformv1alpha1.ServiceSpec{Port: 443, Type: "LoadBalancer"},
				},
			},
			expectedPort: 443,
			expectedType: "LoadBalancer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := buildDesiredService(tt.app)

			if svc.Name != tt.app.Name {
				t.Errorf("expected name %q, got %q", tt.app.Name, svc.Name)
			}
			if svc.Namespace != tt.app.Namespace {
				t.Errorf("expected namespace %q, got %q", tt.app.Namespace, svc.Namespace)
			}
			if svc.Spec.Ports[0].Port != tt.expectedPort {
				t.Errorf("expected port %d, got %d", tt.expectedPort, svc.Spec.Ports[0].Port)
			}
			if string(svc.Spec.Type) != tt.expectedType {
				t.Errorf("expected type %s, got %s", tt.expectedType, svc.Spec.Type)
			}
			if svc.Spec.Selector["app.kubernetes.io/name"] != tt.app.Name {
				t.Errorf("expected selector name %q", tt.app.Name)
			}
			if svc.Labels["app.kubernetes.io/managed-by"] != "platform-operator" {
				t.Error("expected managed-by label")
			}
		})
	}
}

func TestBuildDesiredHPA(t *testing.T) {
	tests := []struct {
		name        string
		app         *platformv1alpha1.PlatformApplication
		expectedMin int32
		expectedMax int32
		expectedCPU int32
	}{
		{
			name: "standard HPA",
			app: &platformv1alpha1.PlatformApplication{
				ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: "default"},
				Spec: platformv1alpha1.PlatformApplicationSpec{
					Image:       platformv1alpha1.ImageSpec{Repository: "ghcr.io/example/worker", Tag: "1.0"},
					Replicas:    platformv1alpha1.ReplicasSpec{Min: 3, Max: 15},
					Service:     platformv1alpha1.ServiceSpec{Port: 8080},
					Autoscaling: platformv1alpha1.AutoscalingSpec{Enabled: true, TargetCPUUtilization: 60},
				},
			},
			expectedMin: 3,
			expectedMax: 15,
			expectedCPU: 60,
		},
		{
			name: "max less than min defaults to min",
			app: &platformv1alpha1.PlatformApplication{
				ObjectMeta: metav1.ObjectMeta{Name: "svc", Namespace: "default"},
				Spec: platformv1alpha1.PlatformApplicationSpec{
					Image:       platformv1alpha1.ImageSpec{Repository: "ghcr.io/example/svc", Tag: "1.0"},
					Replicas:    platformv1alpha1.ReplicasSpec{Min: 5, Max: 2},
					Service:     platformv1alpha1.ServiceSpec{Port: 8080},
					Autoscaling: platformv1alpha1.AutoscalingSpec{Enabled: true},
				},
			},
			expectedMin: 5,
			expectedMax: 5,
			expectedCPU: 80,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hpa := buildDesiredHPA(tt.app)

			if hpa.Name != tt.app.Name {
				t.Errorf("expected name %q, got %q", tt.app.Name, hpa.Name)
			}
			if *hpa.Spec.MinReplicas != tt.expectedMin {
				t.Errorf("expected min %d, got %d", tt.expectedMin, *hpa.Spec.MinReplicas)
			}
			if hpa.Spec.MaxReplicas != tt.expectedMax {
				t.Errorf("expected max %d, got %d", tt.expectedMax, hpa.Spec.MaxReplicas)
			}
			cpu := hpa.Spec.Metrics[0].Resource.Target.AverageUtilization
			if *cpu != tt.expectedCPU {
				t.Errorf("expected CPU %d, got %d", tt.expectedCPU, *cpu)
			}
			if hpa.Spec.ScaleTargetRef.Name != tt.app.Name {
				t.Errorf("expected scale target %q, got %q", tt.app.Name, hpa.Spec.ScaleTargetRef.Name)
			}
		})
	}
}

func TestBuildDesiredPDB(t *testing.T) {
	tests := []struct {
		name          string
		app           *platformv1alpha1.PlatformApplication
		expectPercent bool
	}{
		{
			name: "small deployment uses int",
			app: &platformv1alpha1.PlatformApplication{
				ObjectMeta: metav1.ObjectMeta{Name: "small-app", Namespace: "default"},
				Spec: platformv1alpha1.PlatformApplicationSpec{
					Image:    platformv1alpha1.ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
					Replicas: platformv1alpha1.ReplicasSpec{Min: 3},
					Service:  platformv1alpha1.ServiceSpec{Port: 8080},
				},
			},
			expectPercent: false,
		},
		{
			name: "large deployment uses percentage",
			app: &platformv1alpha1.PlatformApplication{
				ObjectMeta: metav1.ObjectMeta{Name: "large-app", Namespace: "default"},
				Spec: platformv1alpha1.PlatformApplicationSpec{
					Image:    platformv1alpha1.ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
					Replicas: platformv1alpha1.ReplicasSpec{Min: 10},
					Service:  platformv1alpha1.ServiceSpec{Port: 8080},
				},
			},
			expectPercent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdb := buildDesiredPDB(tt.app)

			if pdb.Spec.MaxUnavailable == nil {
				t.Fatal("expected maxUnavailable to be set")
			}

			if tt.expectPercent {
				if pdb.Spec.MaxUnavailable.Type != 1 { // intstr.String = 1
					t.Errorf("expected string type (percentage), got %v", pdb.Spec.MaxUnavailable.Type)
				}
				if pdb.Spec.MaxUnavailable.StrVal != "25%" {
					t.Errorf("expected 25%%, got %s", pdb.Spec.MaxUnavailable.StrVal)
				}
			} else {
				if pdb.Spec.MaxUnavailable.IntVal != 1 {
					t.Errorf("expected int 1, got %d", pdb.Spec.MaxUnavailable.IntVal)
				}
			}

			if pdb.Spec.Selector.MatchLabels["app.kubernetes.io/name"] != tt.app.Name {
				t.Errorf("expected selector name %q", tt.app.Name)
			}
		})
	}
}

func TestBuildDesiredNetworkPolicy(t *testing.T) {
	app := &platformv1alpha1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{Name: "secure-app", Namespace: "production"},
		Spec: platformv1alpha1.PlatformApplicationSpec{
			Image:    platformv1alpha1.ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
			Replicas: platformv1alpha1.ReplicasSpec{Min: 2},
			Service:  platformv1alpha1.ServiceSpec{Port: 8080},
			Security: platformv1alpha1.SecuritySpec{NetworkPolicy: true},
		},
	}

	np := buildDesiredNetworkPolicy(app)

	if np.Name != "secure-app-netpol" {
		t.Errorf("expected name secure-app-netpol, got %s", np.Name)
	}
	if np.Namespace != "production" {
		t.Errorf("expected namespace production, got %s", np.Namespace)
	}
	if np.Spec.PodSelector.MatchLabels["app.kubernetes.io/name"] != "secure-app" {
		t.Error("expected pod selector to match app name")
	}
	if len(np.Spec.Ingress) != 1 {
		t.Errorf("expected 1 ingress rule, got %d", len(np.Spec.Ingress))
	}
	if len(np.Spec.Egress) != 1 {
		t.Errorf("expected 1 egress rule, got %d", len(np.Spec.Egress))
	}
}
