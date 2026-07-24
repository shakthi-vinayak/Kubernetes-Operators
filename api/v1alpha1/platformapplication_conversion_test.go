package v1alpha1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/example/platform-operator/api/v1beta1"
)

// TestConvertTo verifies v1alpha1 -> v1beta1 conversion (spoke -> hub).
func TestConvertTo(t *testing.T) {
	src := &PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
		Spec: PlatformApplicationSpec{
			Image: ImageSpec{
				Repository: "nginx",
				Tag:        "alpine",
				PullPolicy: corev1.PullIfNotPresent,
			},
			Replicas: ReplicasSpec{
				Min: 2,
				Max: 5,
			},
			Service: ServiceSpec{
				Port: 80,
				Type: corev1.ServiceTypeClusterIP,
			},
			Configuration: map[string]string{
				"APP_ENV": "test",
			},
			Autoscaling: AutoscalingSpec{
				Enabled:              true,
				TargetCPUUtilization: 70,
			},
			Gateway: GatewaySpec{
				Enabled:    true,
				Host:       "test.example.io",
				GatewayRef: "infra/gw",
				PathPrefix: "/api",
			},
			Observability: ObservabilitySpec{
				Metrics: true,
			},
			Security: SecuritySpec{
				NetworkPolicy: true,
			},
			Resources: ResourcesSpec{
				Requests: ResourceSpec{CPU: "100m", Memory: "128Mi"},
				Limits:   ResourceSpec{CPU: "500m", Memory: "256Mi"},
			},
			HealthChecks: HealthChecksSpec{
				LivenessPath:  "/healthz",
				ReadinessPath: "/readyz",
				Port:          8080,
			},
			Rollout: RolloutSpec{
				Strategy:       "RollingUpdate",
				MaxUnavailable: "25%",
				MaxSurge:       "25%",
			},
		},
		Status: PlatformApplicationStatus{
			ObservedGeneration: 3,
			ReadyReplicas:      2,
			URL:                "https://test.example.io",
			DeployedVersion:    "alpine",
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Available"},
			},
		},
	}

	dst := &v1beta1.PlatformApplication{}
	if err := src.ConvertTo(dst); err != nil {
		t.Fatalf("ConvertTo() error: %v", err)
	}

	// Verify ObjectMeta
	if dst.Name != "test-app" || dst.Namespace != "default" {
		t.Errorf("ObjectMeta mismatch: got name=%s ns=%s", dst.Name, dst.Namespace)
	}

	// Verify shared spec fields
	if dst.Spec.Image.Repository != "nginx" || dst.Spec.Image.Tag != "alpine" {
		t.Errorf("Image mismatch: got %+v", dst.Spec.Image)
	}
	if dst.Spec.Replicas.Min != 2 || dst.Spec.Replicas.Max != 5 {
		t.Errorf("Replicas mismatch: got %+v", dst.Spec.Replicas)
	}
	if dst.Spec.Service.Port != 80 || dst.Spec.Service.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("Service mismatch: got %+v", dst.Spec.Service)
	}
	if dst.Spec.Configuration["APP_ENV"] != "test" {
		t.Errorf("Configuration mismatch: got %v", dst.Spec.Configuration)
	}
	if !dst.Spec.Autoscaling.Enabled || dst.Spec.Autoscaling.TargetCPUUtilization != 70 {
		t.Errorf("Autoscaling mismatch: got %+v", dst.Spec.Autoscaling)
	}
	if !dst.Spec.Gateway.Enabled || dst.Spec.Gateway.Host != "test.example.io" {
		t.Errorf("Gateway mismatch: got %+v", dst.Spec.Gateway)
	}
	if !dst.Spec.Observability.Metrics {
		t.Error("Observability.Metrics mismatch")
	}
	if !dst.Spec.Security.NetworkPolicy {
		t.Error("Security.NetworkPolicy mismatch")
	}
	if dst.Spec.Resources.Requests.CPU != "100m" || dst.Spec.Resources.Limits.Memory != "256Mi" {
		t.Errorf("Resources mismatch: got %+v", dst.Spec.Resources)
	}
	if dst.Spec.HealthChecks.Port != 8080 {
		t.Errorf("HealthChecks mismatch: got %+v", dst.Spec.HealthChecks)
	}
	if dst.Spec.Rollout.Strategy != "RollingUpdate" {
		t.Errorf("Rollout mismatch: got %+v", dst.Spec.Rollout)
	}

	// Verify v1beta1-only fields are nil (not in v1alpha1)
	if dst.Spec.EnvFrom != nil {
		t.Error("EnvFrom should be nil after v1alpha1 -> v1beta1 conversion")
	}
	if dst.Spec.PodAnnotations != nil {
		t.Error("PodAnnotations should be nil after v1alpha1 -> v1beta1 conversion")
	}

	// Verify status
	if dst.Status.ObservedGeneration != 3 || dst.Status.ReadyReplicas != 2 {
		t.Errorf("Status mismatch: got %+v", dst.Status)
	}
	if dst.Status.URL != "https://test.example.io" {
		t.Errorf("Status.URL mismatch: got %s", dst.Status.URL)
	}
	if len(dst.Status.Conditions) != 1 || dst.Status.Conditions[0].Type != "Ready" {
		t.Errorf("Status.Conditions mismatch: got %+v", dst.Status.Conditions)
	}
}

// TestConvertFrom verifies v1beta1 -> v1alpha1 conversion (hub -> spoke).
// v1beta1-only fields (EnvFrom, PodAnnotations) are dropped.
func TestConvertFrom(t *testing.T) {
	src := &v1beta1.PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "production",
		},
		Spec: v1beta1.PlatformApplicationSpec{
			Image: v1beta1.ImageSpec{
				Repository: "myapp",
				Tag:        "v2.0",
				PullPolicy: corev1.PullAlways,
			},
			Replicas: v1beta1.ReplicasSpec{
				Min: 3,
				Max: 10,
			},
			Service: v1beta1.ServiceSpec{
				Port: 8080,
				Type: corev1.ServiceTypeLoadBalancer,
			},
			Configuration: map[string]string{
				"DB_HOST": "postgres",
			},
			// v1beta1-only fields
			EnvFrom: []corev1.EnvFromSource{
				{
					ConfigMapRef: &corev1.ConfigMapEnvSource{
						LocalObjectReference: corev1.LocalObjectReference{Name: "my-config"},
					},
				},
			},
			PodAnnotations: map[string]string{
				"prometheus.io/scrape": "true",
			},
			Autoscaling: v1beta1.AutoscalingSpec{
				Enabled:              true,
				TargetCPUUtilization: 60,
			},
			Gateway: v1beta1.GatewaySpec{
				Enabled: true,
				Host:    "app.example.io",
			},
			Observability: v1beta1.ObservabilitySpec{Metrics: true},
			Security:      v1beta1.SecuritySpec{NetworkPolicy: true},
			Resources: v1beta1.ResourcesSpec{
				Requests: v1beta1.ResourceSpec{CPU: "200m", Memory: "256Mi"},
				Limits:   v1beta1.ResourceSpec{CPU: "1", Memory: "512Mi"},
			},
			HealthChecks: v1beta1.HealthChecksSpec{
				LivenessPath:  "/health",
				ReadinessPath: "/ready",
				Port:          9090,
			},
			Rollout: v1beta1.RolloutSpec{
				Strategy:       "Recreate",
				MaxUnavailable: "0",
				MaxSurge:       "1",
			},
		},
		Status: v1beta1.PlatformApplicationStatus{
			ObservedGeneration: 5,
			ReadyReplicas:      3,
			URL:                "https://app.example.io",
			DeployedVersion:    "v2.0",
		},
	}

	dst := &PlatformApplication{}
	if err := dst.ConvertFrom(src); err != nil {
		t.Fatalf("ConvertFrom() error: %v", err)
	}

	// Verify shared fields converted correctly
	if dst.Spec.Image.Repository != "myapp" || dst.Spec.Image.Tag != "v2.0" {
		t.Errorf("Image mismatch: got %+v", dst.Spec.Image)
	}
	if dst.Spec.Replicas.Min != 3 || dst.Spec.Replicas.Max != 10 {
		t.Errorf("Replicas mismatch: got %+v", dst.Spec.Replicas)
	}
	if dst.Spec.Service.Port != 8080 || dst.Spec.Service.Type != corev1.ServiceTypeLoadBalancer {
		t.Errorf("Service mismatch: got %+v", dst.Spec.Service)
	}
	if dst.Spec.Configuration["DB_HOST"] != "postgres" {
		t.Errorf("Configuration mismatch: got %v", dst.Spec.Configuration)
	}
	if dst.Spec.Rollout.Strategy != "Recreate" {
		t.Errorf("Rollout mismatch: got %+v", dst.Spec.Rollout)
	}

	// Verify status converted correctly
	if dst.Status.ObservedGeneration != 5 || dst.Status.ReadyReplicas != 3 {
		t.Errorf("Status mismatch: got %+v", dst.Status)
	}
	if dst.Status.DeployedVersion != "v2.0" {
		t.Errorf("DeployedVersion mismatch: got %s", dst.Status.DeployedVersion)
	}

	// Note: EnvFrom and PodAnnotations are v1beta1-only and dropped in v1alpha1
	// There's no field to check — they simply don't exist in v1alpha1
}

// TestRoundTripConversion verifies v1alpha1 -> v1beta1 -> v1alpha1 preserves data.
func TestRoundTripConversion(t *testing.T) {
	original := &PlatformApplication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "roundtrip-app",
			Namespace: "test",
		},
		Spec: PlatformApplicationSpec{
			Image: ImageSpec{
				Repository: "myrepo",
				Tag:        "v1.0",
				PullPolicy: corev1.PullIfNotPresent,
			},
			Replicas: ReplicasSpec{Min: 2, Max: 4},
			Service:  ServiceSpec{Port: 8080, Type: corev1.ServiceTypeClusterIP},
			Configuration: map[string]string{
				"KEY": "value",
			},
			Autoscaling: AutoscalingSpec{
				Enabled:              true,
				TargetCPUUtilization: 75,
			},
			Gateway: GatewaySpec{
				Enabled:    true,
				Host:       "rt.example.io",
				GatewayRef: "ns/gw",
				PathPrefix: "/app",
			},
			Observability: ObservabilitySpec{Metrics: true},
			Security:      SecuritySpec{NetworkPolicy: true},
			Resources: ResourcesSpec{
				Requests: ResourceSpec{CPU: "50m", Memory: "64Mi"},
				Limits:   ResourceSpec{CPU: "200m", Memory: "128Mi"},
			},
			HealthChecks: HealthChecksSpec{
				LivenessPath:  "/live",
				ReadinessPath: "/ready",
				Port:          8080,
			},
			Rollout: RolloutSpec{
				Strategy:       "RollingUpdate",
				MaxUnavailable: "1",
				MaxSurge:       "1",
			},
		},
		Status: PlatformApplicationStatus{
			ObservedGeneration: 1,
			ReadyReplicas:      2,
			URL:                "https://rt.example.io",
			DeployedVersion:    "v1.0",
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Available", Message: "All good"},
			},
		},
	}

	// v1alpha1 -> v1beta1
	hub := &v1beta1.PlatformApplication{}
	if err := original.ConvertTo(hub); err != nil {
		t.Fatalf("ConvertTo() error: %v", err)
	}

	// v1beta1 -> v1alpha1
	roundTripped := &PlatformApplication{}
	if err := roundTripped.ConvertFrom(hub); err != nil {
		t.Fatalf("ConvertFrom() error: %v", err)
	}

	// Verify round-trip preserves all shared fields
	if roundTripped.Spec.Image.Repository != original.Spec.Image.Repository {
		t.Errorf("Image.Repository mismatch")
	}
	if roundTripped.Spec.Image.Tag != original.Spec.Image.Tag {
		t.Errorf("Image.Tag mismatch")
	}
	if roundTripped.Spec.Replicas.Min != original.Spec.Replicas.Min {
		t.Errorf("Replicas.Min mismatch")
	}
	if roundTripped.Spec.Replicas.Max != original.Spec.Replicas.Max {
		t.Errorf("Replicas.Max mismatch")
	}
	if roundTripped.Spec.Service.Port != original.Spec.Service.Port {
		t.Errorf("Service.Port mismatch")
	}
	if roundTripped.Spec.Service.Type != original.Spec.Service.Type {
		t.Errorf("Service.Type mismatch")
	}
	if roundTripped.Spec.Configuration["KEY"] != original.Spec.Configuration["KEY"] {
		t.Errorf("Configuration mismatch")
	}
	if roundTripped.Spec.Gateway.Host != original.Spec.Gateway.Host {
		t.Errorf("Gateway.Host mismatch")
	}
	if roundTripped.Spec.Gateway.GatewayRef != original.Spec.Gateway.GatewayRef {
		t.Errorf("Gateway.GatewayRef mismatch")
	}
	if roundTripped.Spec.Rollout.Strategy != original.Spec.Rollout.Strategy {
		t.Errorf("Rollout.Strategy mismatch")
	}
	if roundTripped.Status.ObservedGeneration != original.Status.ObservedGeneration {
		t.Errorf("Status.ObservedGeneration mismatch")
	}
	if roundTripped.Status.DeployedVersion != original.Status.DeployedVersion {
		t.Errorf("Status.DeployedVersion mismatch")
	}
	if len(roundTripped.Status.Conditions) != len(original.Status.Conditions) {
		t.Errorf("Status.Conditions length mismatch: %d vs %d",
			len(roundTripped.Status.Conditions), len(original.Status.Conditions))
	}
}
