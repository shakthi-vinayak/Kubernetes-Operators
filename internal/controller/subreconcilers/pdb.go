package subreconcilers

import (
	"context"
	"fmt"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1alpha1 "github.com/example/platform-operator/api/v1alpha1"
)

// ReconcilePDB computes the desired PodDisruptionBudget and applies it.
// PDB is only created when min replicas > 1 (single replica apps don't benefit).
func ReconcilePDB(
	ctx context.Context,
	c client.Client,
	scheme *runtime.Scheme,
	app *platformv1alpha1.PlatformApplication,
) (ApplyResult, error) {
	// Skip PDB for single-replica applications.
	if app.Spec.Replicas.Min <= 1 {
		return deletePDBIfExists(ctx, c, app)
	}

	desired := buildDesiredPDB(app)
	return Apply(ctx, c, scheme, app, desired)
}

func buildDesiredPDB(app *platformv1alpha1.PlatformApplication) *policyv1.PodDisruptionBudget {
	labels := CommonLabels(app.Name)

	// Allow at most 1 pod unavailable, or 25% for larger deployments.
	maxUnavailable := intstr.FromInt32(1)
	if app.Spec.Replicas.Min > 4 {
		maxUnavailable = intstr.FromString("25%")
	}

	return &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pdb", app.Name),
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name":     app.Name,
					"app.kubernetes.io/instance": app.Name,
				},
			},
		},
	}
}

func deletePDBIfExists(ctx context.Context, c client.Client, app *platformv1alpha1.PlatformApplication) (ApplyResult, error) {
	pdb := &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-pdb", app.Name),
			Namespace: app.Namespace,
		},
	}
	if err := DeleteIfExists(ctx, c, pdb); err != nil {
		return "", err
	}
	return ApplyResultUnchanged, nil
}
