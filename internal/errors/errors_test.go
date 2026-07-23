package errors

import (
	"errors"
	"fmt"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestReconcileError_Error(t *testing.T) {
	err := NewTransientError("my-deploy", "Deployment", fmt.Errorf("connection refused"))
	expected := "transient error on Deployment my-deploy: connection refused"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestReconcileError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("underlying error")
	err := NewPermanentError("my-svc", "Service", inner)
	if !errors.Is(err, inner) {
		t.Error("expected Unwrap to return inner error")
	}
}

func TestReconcileError_ClassChecks(t *testing.T) {
	tests := []struct {
		name      string
		err       *ReconcileError
		transient bool
		permanent bool
		conflict  bool
	}{
		{
			name:      "transient",
			err:       NewTransientError("r", "K", fmt.Errorf("timeout")),
			transient: true,
		},
		{
			name:      "permanent",
			err:       NewPermanentError("r", "K", fmt.Errorf("invalid")),
			permanent: true,
		},
		{
			name:     "conflict",
			err:      NewConflictError("r", "K", fmt.Errorf("version mismatch")),
			conflict: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.IsTransient() != tt.transient {
				t.Errorf("IsTransient: expected %v, got %v", tt.transient, tt.err.IsTransient())
			}
			if tt.err.IsPermanent() != tt.permanent {
				t.Errorf("IsPermanent: expected %v, got %v", tt.permanent, tt.err.IsPermanent())
			}
			if tt.err.IsConflict() != tt.conflict {
				t.Errorf("IsConflict: expected %v, got %v", tt.conflict, tt.err.IsConflict())
			}
		})
	}
}

func TestClassify_Nil(t *testing.T) {
	if got := Classify(nil); got != "" {
		t.Errorf("expected empty class for nil error, got %q", got)
	}
}

func TestClassify_AlreadyClassified(t *testing.T) {
	inner := NewPermanentError("x", "Y", fmt.Errorf("bad"))
	wrapped := fmt.Errorf("wrapping: %w", inner)
	if got := Classify(wrapped); got != ClassPermanent {
		t.Errorf("expected permanent from wrapped error, got %q", got)
	}
}

func TestClassify_KubernetesErrors(t *testing.T) {
	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}

	tests := []struct {
		name     string
		err      error
		expected ReconcileErrorClass
	}{
		{"conflict", apierrors.NewConflict(gr, "x", fmt.Errorf("version")), ClassConflict},
		{"server timeout", apierrors.NewServerTimeout(gr, "get", 5), ClassTransient},
		{"too many requests", apierrors.NewTooManyRequests("rate limited", 1), ClassTransient},
		{"service unavailable", apierrors.NewServiceUnavailable("unavailable"), ClassTransient},
		{"not found", apierrors.NewNotFound(gr, "x"), ClassPermanent},
		{"forbidden", apierrors.NewForbidden(gr, "x", fmt.Errorf("denied")), ClassPermanent},
		{"invalid", apierrors.NewInvalid(schema.GroupKind{Group: "apps", Kind: "Deployment"}, "x", nil), ClassPermanent},
		{"bad request", apierrors.NewBadRequest("bad"), ClassPermanent},
		{"unknown error", fmt.Errorf("something unexpected"), ClassUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.err); got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestClassifyOrWrap_Nil(t *testing.T) {
	if got := ClassifyOrWrap(nil, "r", "K"); got != nil {
		t.Error("expected nil for nil error")
	}
}

func TestClassifyOrWrap_AlreadyClassified(t *testing.T) {
	original := NewConflictError("deploy", "Deployment", fmt.Errorf("version"))
	result := ClassifyOrWrap(original, "deploy", "Deployment")
	if result != original {
		t.Error("expected same error when already classified")
	}
}

func TestClassifyOrWrap_ClassifiesUnknown(t *testing.T) {
	err := fmt.Errorf("something")
	result := ClassifyOrWrap(err, "my-svc", "Service")
	if result.Class != ClassTransient {
		t.Errorf("expected transient class for unknown error, got %q", result.Class)
	}
	if result.Resource != "my-svc" {
		t.Errorf("expected resource my-svc, got %q", result.Resource)
	}
}

func TestClassifyOrWrap_ClassifiesK8sError(t *testing.T) {
	k8sErr := apierrors.NewConflict(schema.GroupResource{Group: "apps", Resource: "deployments"}, "x", fmt.Errorf("version"))
	result := ClassifyOrWrap(k8sErr, "x", "Deployment")
	if result.Class != ClassConflict {
		t.Errorf("expected conflict, got %q", result.Class)
	}
}
