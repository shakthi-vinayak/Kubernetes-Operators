package errors

import (
	"fmt"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BenchmarkClassify_K8sConflict benchmarks classifying a Kubernetes conflict error.
func BenchmarkClassify_K8sConflict(b *testing.B) {
	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}
	err := apierrors.NewConflict(gr, "test", fmt.Errorf("version mismatch"))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Classify(err)
	}
}

// BenchmarkClassify_Transient benchmarks classifying a transient error.
func BenchmarkClassify_Transient(b *testing.B) {
	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}
	err := apierrors.NewServerTimeout(gr, "get", 5)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Classify(err)
	}
}

// BenchmarkClassify_Permanent benchmarks classifying a permanent error.
func BenchmarkClassify_Permanent(b *testing.B) {
	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}
	err := apierrors.NewForbidden(gr, "test", fmt.Errorf("denied"))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Classify(err)
	}
}

// BenchmarkClassify_AlreadyClassified benchmarks detecting a pre-classified error.
func BenchmarkClassify_AlreadyClassified(b *testing.B) {
	err := NewPermanentError("r", "K", fmt.Errorf("bad"))
	wrapped := fmt.Errorf("wrapping: %w", err)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Classify(wrapped)
	}
}

// BenchmarkClassifyOrWrap benchmarks the full classify-and-wrap path.
func BenchmarkClassifyOrWrap(b *testing.B) {
	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}
	err := apierrors.NewConflict(gr, "test", fmt.Errorf("version"))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ClassifyOrWrap(err, "test", "Deployment")
	}
}

// BenchmarkNewReconcileError benchmarks creating a structured error.
func BenchmarkNewReconcileError(b *testing.B) {
	inner := fmt.Errorf("something")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewTransientError("deploy", "Deployment", inner)
	}
}
