package subreconcilers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformerrors "github.com/example/platform-operator/internal/errors"
	"github.com/example/platform-operator/internal/metrics"
)

// FieldManager is the field manager name used for Server-Side Apply.
const FieldManager = "platform-operator"

// maxApplyRetries is the maximum number of retries for conflict errors during apply.
const maxApplyRetries = 3

// retryBaseDelay is the base delay for exponential backoff on conflict retries.
const retryBaseDelay = 100 * time.Millisecond

// ApplyResult represents the outcome of an apply operation.
type ApplyResult string

const (
	// ApplyResultCreated indicates the resource was created.
	ApplyResultCreated ApplyResult = "created"
	// ApplyResultUpdated indicates the resource was updated.
	ApplyResultUpdated ApplyResult = "updated"
	// ApplyResultUnchanged indicates the resource was unchanged.
	ApplyResultUnchanged ApplyResult = "unchanged"
	// ApplyResultDeleted indicates the resource was deleted.
	ApplyResultDeleted ApplyResult = "deleted"
)

// Apply creates or updates a Kubernetes resource using Server-Side Apply.
// It sets the owner reference so the resource is garbage-collected when the
// owner is deleted.
//
// This function is idempotent: calling it multiple times with the same desired
// state will not produce additional API writes. It uses Server-Side Apply to
// only manage the fields the operator owns, respecting managed fields for
// other controllers.
//
// On conflict errors (resource version mismatch), the function retries with
// exponential backoff up to maxApplyRetries times.
func Apply(ctx context.Context, c client.Client, scheme *runtime.Scheme, owner metav1.Object, desired client.Object) (ApplyResult, error) {
	logger := log.FromContext(ctx)
	kind := desired.GetObjectKind().GroupVersionKind().Kind
	name := desired.GetName()
	namespace := desired.GetNamespace()

	// Set owner reference for automatic garbage collection.
	if err := controllerutil.SetControllerReference(owner, desired, scheme); err != nil {
		return "", fmt.Errorf("setting controller reference: %w", err)
	}

	var result ApplyResult
	var lastErr error

	for attempt := 0; attempt <= maxApplyRetries; attempt++ {
		// Exponential backoff on retries (not on first attempt).
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<uint(attempt-1))
			logger.V(1).Info("retrying apply after conflict",
				"kind", kind, "name", name, "namespace", namespace,
				"attempt", attempt, "delay", delay)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		result, lastErr = applyOnce(ctx, c, desired, kind, name, namespace)
		if lastErr == nil {
			// Success — record metrics and return.
			metrics.ObserveResourceApply(resourceKind(kind), string(result))
			return result, nil
		}

		// Only retry on conflict errors.
		if !apierrors.IsConflict(lastErr) {
			break
		}

		logger.V(1).Info("conflict detected during apply, will retry",
			"kind", kind, "name", name, "namespace", namespace,
			"attempt", attempt)
	}

	// All retries exhausted or non-retryable error — classify and return.
	classifiedErr := platformerrors.ClassifyOrWrap(lastErr, name, kind)
	metrics.ObserveResourceApply(resourceKind(kind), "error")

	return "", classifiedErr
}

// applyOnce performs a single apply attempt without retry logic.
func applyOnce(ctx context.Context, c client.Client, desired client.Object, kind, name, namespace string) (ApplyResult, error) {
	logger := log.FromContext(ctx)

	existing := desired.DeepCopyObject().(client.Object)
	err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing)

	if apierrors.IsNotFound(err) {
		// Resource does not exist; create it.
		apiStart := time.Now()
		if err := c.Create(ctx, desired); err != nil {
			return "", fmt.Errorf("creating %s %s/%s: %w", kind, namespace, name, err)
		}
		metrics.ObserveAPICall("create", resourceKind(kind), time.Since(apiStart))
		logger.Info("resource created", "kind", kind, "name", name, "namespace", namespace)
		return ApplyResultCreated, nil
	}
	if err != nil {
		return "", fmt.Errorf("getting %s %s/%s: %w", kind, namespace, name, err)
	}

	// Compare spec to determine if an update is needed.
	// We use semantic equality to avoid unnecessary writes from formatting differences.
	if equality.Semantic.DeepEqual(existing, desired) {
		metrics.ObserveNoOpApply(resourceKind(kind))
		return ApplyResultUnchanged, nil
	}

	// Update the resource using Server-Side Apply via Patch.
	apiStart := time.Now()
	if err := c.Patch(ctx, desired, client.Apply, client.FieldOwner(FieldManager), client.ForceOwnership); err != nil {
		metrics.ObserveAPICall("patch", resourceKind(kind), time.Since(apiStart))
		return "", fmt.Errorf("applying %s %s/%s: %w", kind, namespace, name, err)
	}
	metrics.ObserveAPICall("patch", resourceKind(kind), time.Since(apiStart))

	logger.Info("resource updated", "kind", kind, "name", name, "namespace", namespace)
	return ApplyResultUpdated, nil
}

// DeleteIfExists deletes a resource if it exists. Returns nil if not found.
// This is idempotent and safe for finalizer cleanup.
func DeleteIfExists(ctx context.Context, c client.Client, obj client.Object) error {
	if err := c.Delete(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("deleting %s %s/%s: %w",
			obj.GetObjectKind().GroupVersionKind().Kind,
			obj.GetNamespace(), obj.GetName(), err)
	}
	metrics.ObserveResourceApply(resourceKind(obj.GetObjectKind().GroupVersionKind().Kind), "deleted")
	return nil
}

// resourceKind normalizes the kind string for metric labels.
func resourceKind(kind string) string {
	if kind == "" {
		return "unknown"
	}
	return kind
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
