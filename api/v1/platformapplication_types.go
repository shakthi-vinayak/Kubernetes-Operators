// Package v1 contains the stable GA API for PlatformApplication.
//
// This is the v1 GA (stable) API. It is fully backward-compatible with v1beta1.
// v1beta1 is the hub/storage version; v1 is a spoke that converts to v1beta1.
//
// +kubebuilder:object:generate=true
// +groupName=platform.example.io
package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PlatformApplicationSpec defines the desired state of PlatformApplication.
type PlatformApplicationSpec struct {
	// Image specifies the container image to run.
	// +kubebuilder:validation:Required
	Image ImageSpec `json:"image"`

	// Replicas defines the desired replica count range.
	// +kubebuilder:validation:Required
	Replicas ReplicasSpec `json:"replicas"`

	// Service defines the service exposure configuration.
	// +kubebuilder:validation:Required
	Service ServiceSpec `json:"service"`

	// Configuration is a map of key-value pairs injected as environment variables.
	// +optional
	Configuration map[string]string `json:"configuration,omitempty"`

	// EnvFrom is a list of sources to populate environment variables from.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// PodAnnotations are additional annotations added to managed pods.
	// +optional
	PodAnnotations map[string]string `json:"podAnnotations,omitempty"`

	// Autoscaling configures horizontal pod autoscaling.
	// +optional
	Autoscaling AutoscalingSpec `json:"autoscaling,omitempty"`

	// Gateway configures Gateway API HTTPRoute creation.
	// +optional
	Gateway GatewaySpec `json:"gateway,omitempty"`

	// Observability configures monitoring resource generation.
	// +optional
	Observability ObservabilitySpec `json:"observability,omitempty"`

	// Security configures security-related resource generation.
	// +optional
	Security SecuritySpec `json:"security,omitempty"`

	// Resources defines compute resource requirements.
	// +optional
	Resources ResourcesSpec `json:"resources,omitempty"`

	// HealthChecks configures liveness and readiness probes.
	// +optional
	HealthChecks HealthChecksSpec `json:"healthChecks,omitempty"`

	// Rollout configures the deployment update strategy.
	// +optional
	Rollout RolloutSpec `json:"rollout,omitempty"`
}

// ImageSpec defines the container image configuration.
type ImageSpec struct {
	Repository string            `json:"repository"`
	Tag        string            `json:"tag"`
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

// ReplicasSpec defines the replica count range.
type ReplicasSpec struct {
	Min int32 `json:"min"`
	Max int32 `json:"max,omitempty"`
}

// ServiceSpec defines the Kubernetes Service configuration.
type ServiceSpec struct {
	Port int32              `json:"port"`
	Type corev1.ServiceType `json:"type,omitempty"`
}

// AutoscalingSpec defines horizontal pod autoscaling configuration.
type AutoscalingSpec struct {
	Enabled              bool  `json:"enabled"`
	TargetCPUUtilization int32 `json:"targetCPUUtilization,omitempty"`
}

// GatewaySpec defines Gateway API HTTPRoute configuration.
type GatewaySpec struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host,omitempty"`
	GatewayRef string `json:"gatewayRef,omitempty"`
	PathPrefix string `json:"pathPrefix,omitempty"`
}

// ObservabilitySpec defines monitoring configuration.
type ObservabilitySpec struct {
	Metrics bool `json:"metrics,omitempty"`
}

// SecuritySpec defines security resource configuration.
type SecuritySpec struct {
	NetworkPolicy bool `json:"networkPolicy,omitempty"`
}

// ResourcesSpec defines compute resource requirements.
type ResourcesSpec struct {
	Requests ResourceSpec `json:"requests,omitempty"`
	Limits   ResourceSpec `json:"limits,omitempty"`
}

// ResourceSpec defines CPU and memory values.
type ResourceSpec struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// HealthChecksSpec defines probe configuration.
type HealthChecksSpec struct {
	LivenessPath  string `json:"livenessPath,omitempty"`
	ReadinessPath string `json:"readinessPath,omitempty"`
	Port          int32  `json:"port,omitempty"`
}

// RolloutSpec defines the deployment rollout strategy.
type RolloutSpec struct {
	Strategy       string `json:"strategy,omitempty"`
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
	MaxSurge       string `json:"maxSurge,omitempty"`
}

// PlatformApplicationStatus defines the observed state of PlatformApplication.
type PlatformApplicationStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	ReadyReplicas      int32              `json:"readyReplicas,omitempty"`
	URL                string             `json:"url,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	DeployedVersion    string             `json:"deployedVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=papp
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.deployedVersion"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PlatformApplication is the Schema for the platformapplications API (v1 GA).
type PlatformApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlatformApplicationSpec   `json:"spec,omitempty"`
	Status PlatformApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PlatformApplicationList contains a list of PlatformApplication.
type PlatformApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlatformApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlatformApplication{}, &PlatformApplicationList{})
}
