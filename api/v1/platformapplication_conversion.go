package v1

import (
	platformv1beta1 "github.com/example/platform-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this v1 PlatformApplication to the Hub version (v1beta1).
func (src *PlatformApplication) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*platformv1beta1.PlatformApplication)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec conversion (v1 fields map 1:1 to v1beta1)
	dst.Spec.Image.Repository = src.Spec.Image.Repository
	dst.Spec.Image.Tag = src.Spec.Image.Tag
	dst.Spec.Image.PullPolicy = src.Spec.Image.PullPolicy
	dst.Spec.Replicas.Min = src.Spec.Replicas.Min
	dst.Spec.Replicas.Max = src.Spec.Replicas.Max
	dst.Spec.Service.Port = src.Spec.Service.Port
	dst.Spec.Service.Type = src.Spec.Service.Type
	dst.Spec.Configuration = src.Spec.Configuration
	dst.Spec.EnvFrom = src.Spec.EnvFrom
	dst.Spec.PodAnnotations = src.Spec.PodAnnotations
	dst.Spec.Autoscaling.Enabled = src.Spec.Autoscaling.Enabled
	dst.Spec.Autoscaling.TargetCPUUtilization = src.Spec.Autoscaling.TargetCPUUtilization
	dst.Spec.Gateway.Enabled = src.Spec.Gateway.Enabled
	dst.Spec.Gateway.Host = src.Spec.Gateway.Host
	dst.Spec.Gateway.GatewayRef = src.Spec.Gateway.GatewayRef
	dst.Spec.Gateway.PathPrefix = src.Spec.Gateway.PathPrefix
	dst.Spec.Observability.Metrics = src.Spec.Observability.Metrics
	dst.Spec.Security.NetworkPolicy = src.Spec.Security.NetworkPolicy
	dst.Spec.Resources.Requests.CPU = src.Spec.Resources.Requests.CPU
	dst.Spec.Resources.Requests.Memory = src.Spec.Resources.Requests.Memory
	dst.Spec.Resources.Limits.CPU = src.Spec.Resources.Limits.CPU
	dst.Spec.Resources.Limits.Memory = src.Spec.Resources.Limits.Memory
	dst.Spec.HealthChecks.LivenessPath = src.Spec.HealthChecks.LivenessPath
	dst.Spec.HealthChecks.ReadinessPath = src.Spec.HealthChecks.ReadinessPath
	dst.Spec.HealthChecks.Port = src.Spec.HealthChecks.Port
	dst.Spec.Rollout.Strategy = src.Spec.Rollout.Strategy
	dst.Spec.Rollout.MaxUnavailable = src.Spec.Rollout.MaxUnavailable
	dst.Spec.Rollout.MaxSurge = src.Spec.Rollout.MaxSurge

	// Status conversion
	dst.Status.ObservedGeneration = src.Status.ObservedGeneration
	dst.Status.ReadyReplicas = src.Status.ReadyReplicas
	dst.Status.URL = src.Status.URL
	dst.Status.Conditions = src.Status.Conditions
	dst.Status.DeployedVersion = src.Status.DeployedVersion

	return nil
}

// ConvertFrom converts from the Hub version (v1beta1) to this v1 PlatformApplication.
func (dst *PlatformApplication) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*platformv1beta1.PlatformApplication)

	// ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Spec conversion (v1beta1 fields map 1:1 to v1)
	dst.Spec.Image.Repository = src.Spec.Image.Repository
	dst.Spec.Image.Tag = src.Spec.Image.Tag
	dst.Spec.Image.PullPolicy = src.Spec.Image.PullPolicy
	dst.Spec.Replicas.Min = src.Spec.Replicas.Min
	dst.Spec.Replicas.Max = src.Spec.Replicas.Max
	dst.Spec.Service.Port = src.Spec.Service.Port
	dst.Spec.Service.Type = src.Spec.Service.Type
	dst.Spec.Configuration = src.Spec.Configuration
	dst.Spec.EnvFrom = src.Spec.EnvFrom
	dst.Spec.PodAnnotations = src.Spec.PodAnnotations
	dst.Spec.Autoscaling.Enabled = src.Spec.Autoscaling.Enabled
	dst.Spec.Autoscaling.TargetCPUUtilization = src.Spec.Autoscaling.TargetCPUUtilization
	dst.Spec.Gateway.Enabled = src.Spec.Gateway.Enabled
	dst.Spec.Gateway.Host = src.Spec.Gateway.Host
	dst.Spec.Gateway.GatewayRef = src.Spec.Gateway.GatewayRef
	dst.Spec.Gateway.PathPrefix = src.Spec.Gateway.PathPrefix
	dst.Spec.Observability.Metrics = src.Spec.Observability.Metrics
	dst.Spec.Security.NetworkPolicy = src.Spec.Security.NetworkPolicy
	dst.Spec.Resources.Requests.CPU = src.Spec.Resources.Requests.CPU
	dst.Spec.Resources.Requests.Memory = src.Spec.Resources.Requests.Memory
	dst.Spec.Resources.Limits.CPU = src.Spec.Resources.Limits.CPU
	dst.Spec.Resources.Limits.Memory = src.Spec.Resources.Limits.Memory
	dst.Spec.HealthChecks.LivenessPath = src.Spec.HealthChecks.LivenessPath
	dst.Spec.HealthChecks.ReadinessPath = src.Spec.HealthChecks.ReadinessPath
	dst.Spec.HealthChecks.Port = src.Spec.HealthChecks.Port
	dst.Spec.Rollout.Strategy = src.Spec.Rollout.Strategy
	dst.Spec.Rollout.MaxUnavailable = src.Spec.Rollout.MaxUnavailable
	dst.Spec.Rollout.MaxSurge = src.Spec.Rollout.MaxSurge

	// Status conversion
	dst.Status.ObservedGeneration = src.Status.ObservedGeneration
	dst.Status.ReadyReplicas = src.Status.ReadyReplicas
	dst.Status.URL = src.Status.URL
	dst.Status.Conditions = src.Status.Conditions
	dst.Status.DeployedVersion = src.Status.DeployedVersion

	return nil
}
