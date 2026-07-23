package subreconcilers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

// ReconcileService computes the desired Service and applies it.
func ReconcileService(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	app *platformv1alpha1.PlatformApplication,
) (ApplyResult, error) {
	desired := buildDesiredService(app)
	return Apply(ctx, c, scheme, app, desired)
}

func buildDesiredService(app *platformv1alpha1.PlatformApplication) *corev1.Service {
	labels := CommonLabels(app.Name)

	svcType := corev1.ServiceTypeClusterIP
	if app.Spec.Service.Type != "" {
		svcType = app.Spec.Service.Type
	}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type: svcType,
			Selector: map[string]string{
				"app.kubernetes.io/name":     app.Name,
				"app.kubernetes.io/instance": app.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       app.Spec.Service.Port,
					TargetPort: intstr.FromString("http"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
