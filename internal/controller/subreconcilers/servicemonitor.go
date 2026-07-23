package subreconcilers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

// ReconcileServiceMonitor computes the desired ServiceMonitor and applies it,
// or deletes it if observability.metrics is disabled.
//
// ServiceMonitor is from the Prometheus Operator CRD (monitoring.coreos.com/v1).
// We use unstructured objects to avoid a hard dependency on the Prometheus Operator
// types. If the CRD is not installed, the apply will fail gracefully.
func ReconcileServiceMonitor(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	app *platformv1alpha1.PlatformApplication,
) (ApplyResult, error) {
	if !app.Spec.Observability.Metrics {
		return deleteServiceMonitorIfExists(ctx, c, app)
	}

	desired := buildDesiredServiceMonitor(app)
	return Apply(ctx, c, scheme, app, desired)
}

func buildDesiredServiceMonitor(app *platformv1alpha1.PlatformApplication) *unstructured.Unstructured {
	labels := CommonLabels(app.Name)

	sm := &unstructured.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitor",
	})
	sm.SetName(fmt.Sprintf("%s-monitor", app.Name))
	sm.SetNamespace(app.Namespace)
	sm.SetLabels(labels)

	sm.Object["spec"] = map[string]interface{}{
		"selector": map[string]interface{}{
			"matchLabels": map[string]interface{}{
				"app.kubernetes.io/name":     app.Name,
				"app.kubernetes.io/instance": app.Name,
			},
		},
		"endpoints": []interface{}{
			map[string]interface{}{
				"port":     "http",
				"path":     "/metrics",
				"interval": "30s",
			},
		},
	}

	return sm
}

func deleteServiceMonitorIfExists(ctx context.Context, c client.Client, app *platformv1alpha1.PlatformApplication) (ApplyResult, error) {
	sm := &unstructured.Unstructured{}
	sm.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitor",
	})
	sm.SetName(fmt.Sprintf("%s-monitor", app.Name))
	sm.SetNamespace(app.Namespace)

	if err := DeleteIfExists(ctx, c, sm); err != nil {
		return "", err
	}
	return ApplyResultUnchanged, nil
}
