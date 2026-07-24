package v1beta1

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Ensure PlatformApplication implements the webhook interfaces.
var _ admission.Defaulter[*PlatformApplication] = &PlatformApplicationDefaulter{}
var _ admission.Validator[*PlatformApplication] = &PlatformApplicationValidator{}

// PlatformApplicationDefaulter implements admission.Defaulter for PlatformApplication v1beta1.
type PlatformApplicationDefaulter struct{}

// Default sets sensible defaults for fields that are not explicitly configured.
func (d *PlatformApplicationDefaulter) Default(_ context.Context, r *PlatformApplication) error {
	// Default image pull policy.
	if r.Spec.Image.PullPolicy == "" {
		r.Spec.Image.PullPolicy = "IfNotPresent"
	}

	// Default max replicas to min replicas.
	if r.Spec.Replicas.Max == 0 {
		r.Spec.Replicas.Max = r.Spec.Replicas.Min
	}

	// Default service type.
	if r.Spec.Service.Type == "" {
		r.Spec.Service.Type = "ClusterIP"
	}

	// Default autoscaling target CPU.
	if r.Spec.Autoscaling.Enabled && r.Spec.Autoscaling.TargetCPUUtilization == 0 {
		r.Spec.Autoscaling.TargetCPUUtilization = 80
	}

	// Default health check paths.
	if r.Spec.HealthChecks.LivenessPath == "" {
		r.Spec.HealthChecks.LivenessPath = "/healthz"
	}
	if r.Spec.HealthChecks.ReadinessPath == "" {
		r.Spec.HealthChecks.ReadinessPath = "/readyz"
	}

	// Default rollout strategy.
	if r.Spec.Rollout.Strategy == "" {
		r.Spec.Rollout.Strategy = "RollingUpdate"
	}

	// Default gateway path prefix.
	if r.Spec.Gateway.Enabled && r.Spec.Gateway.PathPrefix == "" {
		r.Spec.Gateway.PathPrefix = "/"
	}

	return nil
}

// PlatformApplicationValidator implements admission.Validator for PlatformApplication v1beta1.
type PlatformApplicationValidator struct{}

// ValidateCreate validates the object on creation.
func (v *PlatformApplicationValidator) ValidateCreate(_ context.Context, r *PlatformApplication) (admission.Warnings, error) {
	return validatePlatformApplication(r)
}

// ValidateUpdate validates the object on update.
func (v *PlatformApplicationValidator) ValidateUpdate(_ context.Context, _, r *PlatformApplication) (admission.Warnings, error) {
	return validatePlatformApplication(r)
}

// ValidateDelete validates the object on deletion.
func (v *PlatformApplicationValidator) ValidateDelete(_ context.Context, _ *PlatformApplication) (admission.Warnings, error) {
	return nil, nil
}

// validatePlatformApplication performs cross-field validation.
func validatePlatformApplication(r *PlatformApplication) (admission.Warnings, error) {
	var warnings admission.Warnings

	// Validate min <= max replicas.
	if r.Spec.Replicas.Max > 0 && r.Spec.Replicas.Min > r.Spec.Replicas.Max {
		return warnings, fmt.Errorf("spec.replicas.min (%d) must be <= spec.replicas.max (%d)",
			r.Spec.Replicas.Min, r.Spec.Replicas.Max)
	}

	// Validate min replicas >= 1.
	if r.Spec.Replicas.Min < 1 {
		return warnings, fmt.Errorf("spec.replicas.min must be >= 1, got %d", r.Spec.Replicas.Min)
	}

	// Validate image repository is not empty.
	if r.Spec.Image.Repository == "" {
		return warnings, fmt.Errorf("spec.image.repository must not be empty")
	}

	// Validate image tag is not empty.
	if r.Spec.Image.Tag == "" {
		return warnings, fmt.Errorf("spec.image.tag must not be empty")
	}

	// Validate service port range.
	if r.Spec.Service.Port < 1 || r.Spec.Service.Port > 65535 {
		return warnings, fmt.Errorf("spec.service.port must be between 1 and 65535, got %d", r.Spec.Service.Port)
	}

	// Warn if autoscaling is enabled but max equals min.
	if r.Spec.Autoscaling.Enabled && r.Spec.Replicas.Min == r.Spec.Replicas.Max {
		warnings = append(warnings, "autoscaling is enabled but min and max replicas are equal; HPA will have no effect")
	}

	// Warn if gateway is enabled but no host is configured.
	if r.Spec.Gateway.Enabled && r.Spec.Gateway.Host == "" {
		warnings = append(warnings, "gateway is enabled but no host is configured; HTTPRoute will match all hosts")
	}

	return warnings, nil
}
