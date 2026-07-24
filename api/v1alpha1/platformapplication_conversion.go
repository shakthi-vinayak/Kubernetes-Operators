package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	"github.com/example/platform-operator/api/v1beta1"
)

// ConvertTo converts this PlatformApplication (v1alpha1, spoke) to the Hub version (v1beta1).
// Fields that don't exist in v1alpha1 are left empty in v1beta1.
// This conversion is lossless for shared fields.
func (src *PlatformApplication) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.PlatformApplication)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec — shared fields
	dst.Spec.Image = v1beta1.ImageSpec{
		Repository: src.Spec.Image.Repository,
		Tag:        src.Spec.Image.Tag,
		PullPolicy: src.Spec.Image.PullPolicy,
	}
	dst.Spec.Replicas = v1beta1.ReplicasSpec{
		Min: src.Spec.Replicas.Min,
		Max: src.Spec.Replicas.Max,
	}
	dst.Spec.Service = v1beta1.ServiceSpec{
		Port: src.Spec.Service.Port,
		Type: src.Spec.Service.Type,
	}
	dst.Spec.Configuration = src.Spec.Configuration
	dst.Spec.Autoscaling = v1beta1.AutoscalingSpec{
		Enabled:              src.Spec.Autoscaling.Enabled,
		TargetCPUUtilization: src.Spec.Autoscaling.TargetCPUUtilization,
	}
	dst.Spec.Gateway = v1beta1.GatewaySpec{
		Enabled:    src.Spec.Gateway.Enabled,
		Host:       src.Spec.Gateway.Host,
		GatewayRef: src.Spec.Gateway.GatewayRef,
		PathPrefix: src.Spec.Gateway.PathPrefix,
	}
	dst.Spec.Observability = v1beta1.ObservabilitySpec{
		Metrics: src.Spec.Observability.Metrics,
	}
	dst.Spec.Security = v1beta1.SecuritySpec{
		NetworkPolicy: src.Spec.Security.NetworkPolicy,
	}
	dst.Spec.Resources = v1beta1.ResourcesSpec{
		Requests: v1beta1.ResourceSpec{
			CPU:    src.Spec.Resources.Requests.CPU,
			Memory: src.Spec.Resources.Requests.Memory,
		},
		Limits: v1beta1.ResourceSpec{
			CPU:    src.Spec.Resources.Limits.CPU,
			Memory: src.Spec.Resources.Limits.Memory,
		},
	}
	dst.Spec.HealthChecks = v1beta1.HealthChecksSpec{
		LivenessPath:  src.Spec.HealthChecks.LivenessPath,
		ReadinessPath: src.Spec.HealthChecks.ReadinessPath,
		Port:          src.Spec.HealthChecks.Port,
	}
	dst.Spec.Rollout = v1beta1.RolloutSpec{
		Strategy:       src.Spec.Rollout.Strategy,
		MaxUnavailable: src.Spec.Rollout.MaxUnavailable,
		MaxSurge:       src.Spec.Rollout.MaxSurge,
	}

	// v1beta1-only fields: left nil (no data in v1alpha1 to convert)
	dst.Spec.EnvFrom = nil
	dst.Spec.PodAnnotations = nil

	// Status
	dst.Status = v1beta1.PlatformApplicationStatus{
		ObservedGeneration: src.Status.ObservedGeneration,
		ReadyReplicas:      src.Status.ReadyReplicas,
		URL:                src.Status.URL,
		Conditions:         src.Status.Conditions,
		DeployedVersion:    src.Status.DeployedVersion,
	}

	return nil
}

// ConvertFrom converts the Hub version (v1beta1) to this PlatformApplication (v1alpha1, spoke).
// v1beta1-only fields (EnvFrom, PodAnnotations) are dropped in the conversion.
// This is a potentially lossy conversion for v1beta1 features not representable in v1alpha1.
func (dst *PlatformApplication) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.PlatformApplication)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec — shared fields
	dst.Spec.Image = ImageSpec{
		Repository: src.Spec.Image.Repository,
		Tag:        src.Spec.Image.Tag,
		PullPolicy: src.Spec.Image.PullPolicy,
	}
	dst.Spec.Replicas = ReplicasSpec{
		Min: src.Spec.Replicas.Min,
		Max: src.Spec.Replicas.Max,
	}
	dst.Spec.Service = ServiceSpec{
		Port: src.Spec.Service.Port,
		Type: src.Spec.Service.Type,
	}
	dst.Spec.Configuration = src.Spec.Configuration
	dst.Spec.Autoscaling = AutoscalingSpec{
		Enabled:              src.Spec.Autoscaling.Enabled,
		TargetCPUUtilization: src.Spec.Autoscaling.TargetCPUUtilization,
	}
	dst.Spec.Gateway = GatewaySpec{
		Enabled:    src.Spec.Gateway.Enabled,
		Host:       src.Spec.Gateway.Host,
		GatewayRef: src.Spec.Gateway.GatewayRef,
		PathPrefix: src.Spec.Gateway.PathPrefix,
	}
	dst.Spec.Observability = ObservabilitySpec{
		Metrics: src.Spec.Observability.Metrics,
	}
	dst.Spec.Security = SecuritySpec{
		NetworkPolicy: src.Spec.Security.NetworkPolicy,
	}
	dst.Spec.Resources = ResourcesSpec{
		Requests: ResourceSpec{
			CPU:    src.Spec.Resources.Requests.CPU,
			Memory: src.Spec.Resources.Requests.Memory,
		},
		Limits: ResourceSpec{
			CPU:    src.Spec.Resources.Limits.CPU,
			Memory: src.Spec.Resources.Limits.Memory,
		},
	}
	dst.Spec.HealthChecks = HealthChecksSpec{
		LivenessPath:  src.Spec.HealthChecks.LivenessPath,
		ReadinessPath: src.Spec.HealthChecks.ReadinessPath,
		Port:          src.Spec.HealthChecks.Port,
	}
	dst.Spec.Rollout = RolloutSpec{
		Strategy:       src.Spec.Rollout.Strategy,
		MaxUnavailable: src.Spec.Rollout.MaxUnavailable,
		MaxSurge:       src.Spec.Rollout.MaxSurge,
	}

	// Status
	dst.Status = PlatformApplicationStatus{
		ObservedGeneration: src.Status.ObservedGeneration,
		ReadyReplicas:      src.Status.ReadyReplicas,
		URL:                src.Status.URL,
		Conditions:         src.Status.Conditions,
		DeployedVersion:    src.Status.DeployedVersion,
	}

	return nil
}
