package v1beta1

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

	// EnvFrom is a list of sources to populate environment variables from
	// (ConfigMaps and Secrets). Applied after Configuration.
	// +optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`

	// PodAnnotations are additional annotations added to the managed pods.
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
	// Repository is the container image repository (e.g., ghcr.io/example/app).
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Repository string `json:"repository"`

	// Tag is the container image tag.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Tag string `json:"tag"`

	// PullPolicy defines the image pull policy.
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +kubebuilder:default=IfNotPresent
	// +optional
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

// ReplicasSpec defines the replica count range.
type ReplicasSpec struct {
	// Min is the minimum number of replicas.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	Min int32 `json:"min"`

	// Max is the maximum number of replicas. Defaults to Min if not set.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	// +optional
	Max int32 `json:"max,omitempty"`
}

// ServiceSpec defines the Kubernetes Service configuration.
type ServiceSpec struct {
	// Port is the port the application listens on.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// Type is the Kubernetes Service type.
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	// +kubebuilder:default=ClusterIP
	// +optional
	Type corev1.ServiceType `json:"type,omitempty"`
}

// AutoscalingSpec defines horizontal pod autoscaling configuration.
type AutoscalingSpec struct {
	// Enabled controls whether HPA is created.
	Enabled bool `json:"enabled"`

	// TargetCPUUtilization is the target average CPU utilization percentage.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=80
	// +optional
	TargetCPUUtilization int32 `json:"targetCPUUtilization,omitempty"`
}

// GatewaySpec defines Gateway API HTTPRoute configuration.
type GatewaySpec struct {
	// Enabled controls whether an HTTPRoute is created.
	Enabled bool `json:"enabled"`

	// Host is the hostname for the HTTPRoute.
	// +optional
	Host string `json:"host,omitempty"`

	// GatewayRef is the namespace/name of the parent Gateway resource.
	// Format: "namespace/gateway-name"
	// +optional
	GatewayRef string `json:"gatewayRef,omitempty"`

	// PathPrefix is the path prefix to match. Defaults to "/".
	// +kubebuilder:default="/"
	// +optional
	PathPrefix string `json:"pathPrefix,omitempty"`
}

// ObservabilitySpec defines monitoring configuration.
type ObservabilitySpec struct {
	// Metrics enables ServiceMonitor generation for Prometheus Operator.
	Metrics bool `json:"metrics,omitempty"`
}

// SecuritySpec defines security resource configuration.
type SecuritySpec struct {
	// NetworkPolicy enables NetworkPolicy generation.
	NetworkPolicy bool `json:"networkPolicy,omitempty"`
}

// ResourcesSpec defines compute resource requirements.
type ResourcesSpec struct {
	// Requests defines minimum resource requirements.
	// +optional
	Requests ResourceSpec `json:"requests,omitempty"`

	// Limits defines maximum resource limits.
	// +optional
	Limits ResourceSpec `json:"limits,omitempty"`
}

// ResourceSpec defines CPU and memory values.
type ResourceSpec struct {
	// CPU resource value (e.g., "100m", "1").
	// +optional
	CPU string `json:"cpu,omitempty"`

	// Memory resource value (e.g., "128Mi", "1Gi").
	// +optional
	Memory string `json:"memory,omitempty"`
}

// HealthChecksSpec defines probe configuration.
type HealthChecksSpec struct {
	// LivenessPath is the HTTP path for liveness probes.
	// +kubebuilder:default="/healthz"
	// +optional
	LivenessPath string `json:"livenessPath,omitempty"`

	// ReadinessPath is the HTTP path for readiness probes.
	// +kubebuilder:default="/readyz"
	// +optional
	ReadinessPath string `json:"readinessPath,omitempty"`

	// Port is the port for health check probes. Defaults to service port.
	// +optional
	Port int32 `json:"port,omitempty"`
}

// RolloutSpec defines the deployment rollout strategy.
type RolloutSpec struct {
	// Strategy is the deployment update strategy.
	// +kubebuilder:validation:Enum=RollingUpdate;Recreate
	// +kubebuilder:default=RollingUpdate
	// +optional
	Strategy string `json:"strategy,omitempty"`

	// MaxUnavailable is the maximum number of pods that can be unavailable during update.
	// +optional
	MaxUnavailable string `json:"maxUnavailable,omitempty"`

	// MaxSurge is the maximum number of pods that can be created above desired during update.
	// +optional
	MaxSurge string `json:"maxSurge,omitempty"`
}

// PlatformApplicationStatus defines the observed state of PlatformApplication.
type PlatformApplicationStatus struct {
	// ObservedGeneration is the most recent generation observed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ReadyReplicas is the number of ready replicas in the managed Deployment.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// URL is the external URL for the application (derived from Gateway).
	// +optional
	URL string `json:"url,omitempty"`

	// Conditions represent the latest available observations of the resource's state.
	// +optional
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// DeployedVersion is the image tag currently running.
	// +optional
	DeployedVersion string `json:"deployedVersion,omitempty"`
}

// Condition types for PlatformApplication.
const (
	// ConditionReady indicates the application is fully reconciled and available.
	ConditionReady = "Ready"

	// ConditionProgressing indicates a rollout or reconciliation is in progress.
	ConditionProgressing = "Progressing"

	// ConditionDegraded indicates the application is partially functional.
	ConditionDegraded = "Degraded"

	// ConditionConfigurationValid indicates the spec passed validation.
	ConditionConfigurationValid = "ConfigurationValid"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=papp
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.deployedVersion"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PlatformApplication is the Schema for the platformapplications API (v1beta1).
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

// Hub marks this type as a conversion hub.
func (*PlatformApplication) Hub() {}

// Hub marks this type as a conversion hub.
func (*PlatformApplicationList) Hub() {}

func init() {
	SchemeBuilder.Register(&PlatformApplication{}, &PlatformApplicationList{})
}
