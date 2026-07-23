package subreconcilers

import (
	"context"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

// ReconcileHPA computes the desired HorizontalPodAutoscaler and applies it,
// or deletes it if autoscaling is disabled.
func ReconcileHPA(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	app *platformv1alpha1.PlatformApplication,
) (ApplyResult, error) {
	if !app.Spec.Autoscaling.Enabled {
		return deleteHPAIfExists(ctx, c, app)
	}

	desired := buildDesiredHPA(app)
	return Apply(ctx, c, scheme, app, desired)
}

func buildDesiredHPA(app *platformv1alpha1.PlatformApplication) *autoscalingv2.HorizontalPodAutoscaler {
	labels := CommonLabels(app.Name)

	minReplicas := app.Spec.Replicas.Min
	maxReplicas := app.Spec.Replicas.Max
	if maxReplicas < minReplicas {
		maxReplicas = minReplicas
	}

	targetCPU := app.Spec.Autoscaling.TargetCPUUtilization
	if targetCPU == 0 {
		targetCPU = 80
	}

	return &autoscalingv2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "autoscaling/v2",
			Kind:       "HorizontalPodAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       app.Name,
			},
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "cpu",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &targetCPU,
						},
					},
				},
			},
		},
	}
}

func deleteHPAIfExists(ctx context.Context, c client.Client, app *platformv1alpha1.PlatformApplication) (ApplyResult, error) {
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}
	if err := DeleteIfExists(ctx, c, hpa); err != nil {
		return "", err
	}
	return ApplyResultUnchanged, nil
}
