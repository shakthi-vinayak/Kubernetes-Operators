package subreconcilers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FieldManager is the field manager name used for Server-Side Apply.
const FieldManager = "platform-operator"

// ApplyResult represents the outcome of an apply operation.
type ApplyResult string

const (
	// ApplyResultCreated indicates the resource was created.
	ApplyResultCreated ApplyResult = "created"
	// ApplyResultUpdated indicates the resource was updated.
	ApplyResultUpdated ApplyResult = "updated"
	// ApplyResultUnchanged indicates the resource was unchanged.
	ApplyResultUnchanged ApplyResult = "unchanged"
)

// Apply creates or updates a Kubernetes resource using Server-Side Apply.
// It sets the owner reference so the resource is garbage-collected when the
// owner is deleted.
//
// This function is idempotent: calling it multiple times with the same desired
// state will not produce additional API writes. It uses Server-Side Apply to
// only manage the fields the operator owns, respecting managed fields for
// other controllers.
func Apply(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner metav1.Object, desired client.Object) (ApplyResult, error) {
	logger := log.FromContext(ctx)

	// Set owner reference for automatic garbage collection.
	if err := controllerutil.SetControllerReference(owner, desired, scheme); err != nil {
		return "", fmt.Errorf("setting controller reference: %w", err)
	}

	// Attempt Server-Side Apply.
	// Force=true means the operator takes ownership of its managed fields,
	// even if another manager previously owned them.
	existing := desired.DeepCopyObject().(client.Object)
	err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if apierrors.IsNotFound(err) {
		// Resource does not exist; create it.
		if err := c.Create(ctx, desired); err != nil {
			return "", fmt.Errorf("creating %s %s/%s: %w",
				desired.GetObjectKind().GroupVersionKind().Kind,
				desired.GetNamespace(), desired.GetName(), err)
		}
		logger.Info("resource created",
			"kind", desired.GetObjectKind().GroupVersionKind().Kind,
			"name", desired.GetName(),
			"namespace", desired.GetNamespace())
		return ApplyResultCreated, nil
	}
	if err != nil {
		return "", fmt.Errorf("getting %s %s/%s: %w",
			desired.GetObjectKind().GroupVersionKind().Kind,
			desired.GetNamespace(), desired.GetName(), err)
	}

	// Compare spec to determine if an update is needed.
	// We use semantic equality to avoid unnecessary writes from formatting differences.
	if equality.Semantic.DeepEqual(existing, desired) {
		return ApplyResultUnchanged, nil
	}

	// Update the resource using Server-Side Apply via Patch.
	if err := c.Patch(ctx, desired, client.Apply, client.FieldOwner(FieldManager), client.ForceOwnership); err != nil {
		return "", fmt.Errorf("applying %s %s/%s: %w",
			desired.GetObjectKind().GroupVersionKind().Kind,
			desired.GetNamespace(), desired.GetName(), err)
	}

	logger.Info("resource updated",
		"kind", desired.GetObjectKind().GroupVersionKind().Kind,
		"name", desired.GetName(),
		"namespace", desired.GetNamespace())
	return ApplyResultUpdated, nil
}

// DeleteIfExists deletes a resource if it exists. Returns nil if not found.
func DeleteIfExists(ctx context.Context, c client.Client, obj client.Object) error {
	if err := c.Delete(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting %s %s/%s: %w",
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetNamespace(), obj.GetName(), err)
	}
	return nil
}

// CommonLabels returns the standard labels applied to all managed resources.
func CommonLabels(appName string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       appName,
		"app.kubernetes.io/managed-by": "platform-operator",
		"app.kubernetes.io/part-of":    "platform",
	}
}

// MergeLabels merges multiple label maps. Later maps override earlier ones.
func MergeLabels(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
