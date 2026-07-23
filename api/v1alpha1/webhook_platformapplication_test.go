package v1alpha1

import (
	"context"
	"testing"
)

func TestPlatformApplicationDefaulter(t *testing.T) {
	tests := []struct {
		name     string
		input    *PlatformApplication
		validate func(t *testing.T, app *PlatformApplication)
	}{
		{
			name: "defaults image pull policy",
			input: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
					Replicas: ReplicasSpec{Min: 3},
					Service:  ServiceSpec{Port: 8080},
				},
			},
			validate: func(t *testing.T, app *PlatformApplication) {
				if string(app.Spec.Image.PullPolicy) != "IfNotPresent" {
					t.Errorf("expected pullPolicy IfNotPresent, got %s", app.Spec.Image.PullPolicy)
				}
			},
		},
		{
			name: "defaults max replicas to min",
			input: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0", PullPolicy: "Always"},
					Replicas: ReplicasSpec{Min: 5},
					Service:  ServiceSpec{Port: 8080},
				},
			},
			validate: func(t *testing.T, app *PlatformApplication) {
				if app.Spec.Replicas.Max != 5 {
					t.Errorf("expected max replicas 5, got %d", app.Spec.Replicas.Max)
				}
			},
		},
		{
			name: "defaults service type to ClusterIP",
			input: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0", PullPolicy: "Always"},
					Replicas: ReplicasSpec{Min: 1, Max: 1},
					Service:  ServiceSpec{Port: 8080},
				},
			},
			validate: func(t *testing.T, app *PlatformApplication) {
				if string(app.Spec.Service.Type) != "ClusterIP" {
					t.Errorf("expected service type ClusterIP, got %s", app.Spec.Service.Type)
				}
			},
		},
		{
			name: "defaults autoscaling target CPU",
			input: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:       ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0", PullPolicy: "Always"},
					Replicas:    ReplicasSpec{Min: 2, Max: 10},
					Service:     ServiceSpec{Port: 8080, Type: "ClusterIP"},
					Autoscaling: AutoscalingSpec{Enabled: true},
				},
			},
			validate: func(t *testing.T, app *PlatformApplication) {
				if app.Spec.Autoscaling.TargetCPUUtilization != 80 {
					t.Errorf("expected target CPU 80, got %d", app.Spec.Autoscaling.TargetCPUUtilization)
				}
			},
		},
		{
			name: "defaults health check paths",
			input: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0", PullPolicy: "Always"},
					Replicas: ReplicasSpec{Min: 1, Max: 1},
					Service:  ServiceSpec{Port: 8080, Type: "ClusterIP"},
				},
			},
			validate: func(t *testing.T, app *PlatformApplication) {
				if app.Spec.HealthChecks.LivenessPath != "/healthz" {
					t.Errorf("expected liveness path /healthz, got %s", app.Spec.HealthChecks.LivenessPath)
				}
				if app.Spec.HealthChecks.ReadinessPath != "/readyz" {
					t.Errorf("expected readiness path /readyz, got %s", app.Spec.HealthChecks.ReadinessPath)
				}
			},
		},
		{
			name: "defaults rollout strategy",
			input: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0", PullPolicy: "Always"},
					Replicas: ReplicasSpec{Min: 1, Max: 1},
					Service:  ServiceSpec{Port: 8080, Type: "ClusterIP"},
				},
			},
			validate: func(t *testing.T, app *PlatformApplication) {
				if app.Spec.Rollout.Strategy != "RollingUpdate" {
					t.Errorf("expected strategy RollingUpdate, got %s", app.Spec.Rollout.Strategy)
				}
			},
		},
		{
			name: "defaults gateway path prefix",
			input: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0", PullPolicy: "Always"},
					Replicas: ReplicasSpec{Min: 1, Max: 1},
					Service:  ServiceSpec{Port: 8080, Type: "ClusterIP"},
					Gateway:  GatewaySpec{Enabled: true, Host: "app.example.com"},
				},
			},
			validate: func(t *testing.T, app *PlatformApplication) {
				if app.Spec.Gateway.PathPrefix != "/" {
					t.Errorf("expected path prefix /, got %s", app.Spec.Gateway.PathPrefix)
				}
			},
		},
		{
			name: "does not override existing values",
			input: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:       ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0", PullPolicy: "Never"},
					Replicas:    ReplicasSpec{Min: 3, Max: 20},
					Service:     ServiceSpec{Port: 9090, Type: "LoadBalancer"},
					Autoscaling: AutoscalingSpec{Enabled: true, TargetCPUUtilization: 50},
					HealthChecks: HealthChecksSpec{
						LivenessPath:  "/health",
						ReadinessPath: "/ready",
					},
					Rollout: RolloutSpec{Strategy: "Recreate"},
					Gateway: GatewaySpec{Enabled: true, PathPrefix: "/api"},
				},
			},
			validate: func(t *testing.T, app *PlatformApplication) {
				if string(app.Spec.Image.PullPolicy) != "Never" {
					t.Errorf("expected pullPolicy Never (preserved), got %s", app.Spec.Image.PullPolicy)
				}
				if app.Spec.Replicas.Max != 20 {
					t.Errorf("expected max 20 (preserved), got %d", app.Spec.Replicas.Max)
				}
				if string(app.Spec.Service.Type) != "LoadBalancer" {
					t.Errorf("expected LoadBalancer (preserved), got %s", app.Spec.Service.Type)
				}
				if app.Spec.Autoscaling.TargetCPUUtilization != 50 {
					t.Errorf("expected CPU 50 (preserved), got %d", app.Spec.Autoscaling.TargetCPUUtilization)
				}
				if app.Spec.HealthChecks.LivenessPath != "/health" {
					t.Errorf("expected /health (preserved), got %s", app.Spec.HealthChecks.LivenessPath)
				}
				if app.Spec.Rollout.Strategy != "Recreate" {
					t.Errorf("expected Recreate (preserved), got %s", app.Spec.Rollout.Strategy)
				}
				if app.Spec.Gateway.PathPrefix != "/api" {
					t.Errorf("expected /api (preserved), got %s", app.Spec.Gateway.PathPrefix)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaulter := &PlatformApplicationDefaulter{}
			if err := defaulter.Default(context.Background(), tt.input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.validate(t, tt.input)
		})
	}
}

func TestPlatformApplicationValidator(t *testing.T) {
	validator := &PlatformApplicationValidator{}

	tests := []struct {
		name        string
		app         *PlatformApplication
		expectErr   bool
		expectWarn  bool
		errContains string
	}{
		{
			name: "valid application",
			app: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
					Replicas: ReplicasSpec{Min: 3, Max: 10},
					Service:  ServiceSpec{Port: 8080},
				},
			},
			expectErr:  false,
			expectWarn: false,
		},
		{
			name: "min greater than max",
			app: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
					Replicas: ReplicasSpec{Min: 10, Max: 3},
					Service:  ServiceSpec{Port: 8080},
				},
			},
			expectErr:   true,
			errContains: "min",
		},
		{
			name: "min replicas zero",
			app: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
					Replicas: ReplicasSpec{Min: 0},
					Service:  ServiceSpec{Port: 8080},
				},
			},
			expectErr:   true,
			errContains: "min",
		},
		{
			name: "empty repository",
			app: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "", Tag: "1.0"},
					Replicas: ReplicasSpec{Min: 1},
					Service:  ServiceSpec{Port: 8080},
				},
			},
			expectErr:   true,
			errContains: "repository",
		},
		{
			name: "empty tag",
			app: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: ""},
					Replicas: ReplicasSpec{Min: 1},
					Service:  ServiceSpec{Port: 8080},
				},
			},
			expectErr:   true,
			errContains: "tag",
		},
		{
			name: "invalid port",
			app: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
					Replicas: ReplicasSpec{Min: 1},
					Service:  ServiceSpec{Port: 0},
				},
			},
			expectErr:   true,
			errContains: "port",
		},
		{
			name: "autoscaling with equal min/max warns",
			app: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:       ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
					Replicas:    ReplicasSpec{Min: 3, Max: 3},
					Service:     ServiceSpec{Port: 8080},
					Autoscaling: AutoscalingSpec{Enabled: true, TargetCPUUtilization: 70},
				},
			},
			expectErr:  false,
			expectWarn: true,
		},
		{
			name: "gateway enabled without host warns",
			app: &PlatformApplication{
				Spec: PlatformApplicationSpec{
					Image:    ImageSpec{Repository: "ghcr.io/example/app", Tag: "1.0"},
					Replicas: ReplicasSpec{Min: 1},
					Service:  ServiceSpec{Port: 8080},
					Gateway:  GatewaySpec{Enabled: true},
				},
			},
			expectErr:  false,
			expectWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := validator.ValidateCreate(context.Background(), tt.app)

			if tt.expectErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectWarn && len(warnings) == 0 {
				t.Error("expected warnings, got none")
			}
			if !tt.expectWarn && len(warnings) > 0 {
				t.Errorf("unexpected warnings: %v", warnings)
			}
		})
	}
}

func TestValidateDelete(t *testing.T) {
	validator := &PlatformApplicationValidator{}
	warnings, err := validator.ValidateDelete(context.Background(), &PlatformApplication{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(warnings) > 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}
