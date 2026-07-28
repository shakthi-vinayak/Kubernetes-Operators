package subreconcilers

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

// ReconcileDeployment computes the desired Deployment and applies it.
func ReconcileDeployment(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	app *platformv1alpha1.PlatformApplication,
) (ApplyResult, error) {
	desired := buildDesiredDeployment(app)
	return Apply(ctx, c, scheme, app, desired)
}

// buildDesiredDeployment constructs the desired Deployment from PlatformApplication spec.
// This is a pure function, making it easy to unit test independently.
func buildDesiredDeployment(app *platformv1alpha1.PlatformApplication) *appsv1.Deployment {
	labels := MergeLabels(CommonLabels(app.Name), map[string]string{
		"app.kubernetes.io/instance": app.Name,
		"app.kubernetes.io/version":  app.Spec.Image.Tag,
	})

	replicas := app.Spec.Replicas.Min
	if replicas < 1 {
		replicas = 1
	}

	container := corev1.Container{
		Name:            app.Name,
		Image:           fmt.Sprintf("%s:%s", app.Spec.Image.Repository, app.Spec.Image.Tag),
		ImagePullPolicy: app.Spec.Image.PullPolicy,
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: app.Spec.Service.Port,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		// Security context: non-root, read-only rootfs, no privilege escalation.
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             boolPtr(true),
			ReadOnlyRootFilesystem:   boolPtr(true),
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
	}

	// Configure resource requests and limits.
	resources := corev1.ResourceRequirements{}
	if app.Spec.Resources.Requests.CPU != "" || app.Spec.Resources.Requests.Memory != "" {
		resources.Requests = corev1.ResourceList{}
		if app.Spec.Resources.Requests.CPU != "" {
			resources.Requests[corev1.ResourceCPU] = resource.MustParse(app.Spec.Resources.Requests.CPU)
		}
		if app.Spec.Resources.Requests.Memory != "" {
			resources.Requests[corev1.ResourceMemory] = resource.MustParse(app.Spec.Resources.Requests.Memory)
		}
	}
	if app.Spec.Resources.Limits.CPU != "" || app.Spec.Resources.Limits.Memory != "" {
		resources.Limits = corev1.ResourceList{}
		if app.Spec.Resources.Limits.CPU != "" {
			resources.Limits[corev1.ResourceCPU] = resource.MustParse(app.Spec.Resources.Limits.CPU)
		}
		if app.Spec.Resources.Limits.Memory != "" {
			resources.Limits[corev1.ResourceMemory] = resource.MustParse(app.Spec.Resources.Limits.Memory)
		}
	}
	container.Resources = resources

	// Configure health check probes.
	probePort := app.Spec.Service.Port
	if app.Spec.HealthChecks.Port != 0 {
		probePort = app.Spec.HealthChecks.Port
	}

	livenessPath := app.Spec.HealthChecks.LivenessPath
	if livenessPath == "" {
		livenessPath = "/healthz"
	}
	readinessPath := app.Spec.HealthChecks.ReadinessPath
	if readinessPath == "" {
		readinessPath = "/readyz"
	}

	container.LivenessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: livenessPath,
				Port: intstr.FromInt32(probePort),
			},
		},
		InitialDelaySeconds: 15,
		PeriodSeconds:       20,
		TimeoutSeconds:      5,
		FailureThreshold:    3,
	}

	container.ReadinessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: readinessPath,
				Port: intstr.FromInt32(probePort),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       10,
		TimeoutSeconds:      3,
		FailureThreshold:    3,
	}

	// Inject configuration as environment variables.
	// Keys are sorted to keep the pod template deterministic; Go map
	// iteration order is random and would churn ReplicaSets on every reconcile.
	if len(app.Spec.Configuration) > 0 {
		keys := make([]string, 0, len(app.Spec.Configuration))
		for k := range app.Spec.Configuration {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		envVars := make([]corev1.EnvVar, 0, len(keys))
		for _, k := range keys {
			envVars = append(envVars, corev1.EnvVar{Name: k, Value: app.Spec.Configuration[k]})
		}
		container.Env = envVars
	}

	// Configure rollout strategy.
	strategy := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
	}
	if app.Spec.Rollout.Strategy == "Recreate" {
		strategy.Type = appsv1.RecreateDeploymentStrategyType
	} else if app.Spec.Rollout.MaxUnavailable != "" || app.Spec.Rollout.MaxSurge != "" {
		ru := &appsv1.RollingUpdateDeployment{}
		if app.Spec.Rollout.MaxUnavailable != "" {
			v := intstr.FromString(app.Spec.Rollout.MaxUnavailable)
			if _, err := strconv.Atoi(app.Spec.Rollout.MaxUnavailable); err == nil {
				v = intstr.FromInt32(int32(mustAtoi(app.Spec.Rollout.MaxUnavailable)))
			}
			ru.MaxUnavailable = &v
		}
		if app.Spec.Rollout.MaxSurge != "" {
			v := intstr.FromString(app.Spec.Rollout.MaxSurge)
			if _, err := strconv.Atoi(app.Spec.Rollout.MaxSurge); err == nil {
				v = intstr.FromInt32(int32(mustAtoi(app.Spec.Rollout.MaxSurge)))
			}
			ru.MaxSurge = &v
		}
		strategy.RollingUpdate = ru
	}

	// Provide a writable /tmp for apps that need scratch space (e.g. nginx
	// temp dirs), since readOnlyRootFilesystem is enforced on the container.
	container.VolumeMounts = []corev1.VolumeMount{
		{Name: "tmp", MountPath: "/tmp"},
	}

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":     app.Name,
					"app.kubernetes.io/instance": app.Name,
				},
			},
			Strategy: strategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: boolPtr(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{container},
					Volumes: []corev1.Volume{
						{
							Name: "tmp",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	return dep
}

func boolPtr(b bool) *bool {
	return &b
}

func mustAtoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
