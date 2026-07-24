// Package fakeclient provides a Kubernetes client interceptor for fault injection testing.
// It wraps any client.Client implementation and can inject configurable errors on
// specific operations (Get, Create, Patch, Delete, Update) to test error handling,
// retry behavior, and resilience of the reconciliation logic.
package fakeclient

import (
	"context"
	"fmt"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FaultType defines the kind of fault to inject.
type FaultType string

const (
	// FaultTransient simulates a transient error (e.g., API server timeout).
	FaultTransient FaultType = "transient"
	// FaultPermanent simulates a permanent error (e.g., RBAC denied).
	FaultPermanent FaultType = "permanent"
	// FaultConflict simulates an optimistic concurrency conflict.
	FaultConflict FaultType = "conflict"
)

// FaultConfig defines when and how to inject a fault.
type FaultConfig struct {
	// Operation targets a specific operation: "get", "create", "patch", "delete", "update", "status".
	Operation string
	// ResourceKind targets a specific resource kind (empty = all).
	ResourceKind string
	// Fault is the type of fault to inject.
	Fault FaultType
	// Count is the number of times to inject the fault before stopping (0 = always).
	Count int
}

// Interceptor wraps a client.Client and injects configurable faults.
type Interceptor struct {
	client.Client
	mu       sync.Mutex
	faults   []FaultConfig
	injected map[string]int // tracks injection counts per operation
}

// NewInterceptor creates a new fault-injecting interceptor around the given client.
func NewInterceptor(c client.Client, faults ...FaultConfig) *Interceptor {
	return &Interceptor{
		Client:   c,
		faults:   faults,
		injected: make(map[string]int),
	}
}

// InjectedCount returns the number of times a fault was injected for a given operation.
func (i *Interceptor) InjectedCount(operation string) int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.injected[operation]
}

// Reset clears all injection counters.
func (i *Interceptor) Reset() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.injected = make(map[string]int)
}

func (i *Interceptor) shouldFault(operation, kind string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	for _, f := range i.faults {
		if f.Operation != operation {
			continue
		}
		if f.ResourceKind != "" && f.ResourceKind != kind {
			continue
		}
		if f.Count > 0 && i.injected[operation] >= f.Count {
			continue
		}
		i.injected[operation]++
		return makeFaultError(f)
	}
	return nil
}

func makeFaultError(f FaultConfig) error {
	switch f.Fault {
	case FaultTransient:
		return &faultError{msg: "injected transient error: service unavailable", transient: true}
	case FaultPermanent:
		return &faultError{msg: "injected permanent error: forbidden", permanent: true}
	case FaultConflict:
		return &faultError{msg: "injected conflict error: resource version mismatch", conflict: true}
	default:
		return &faultError{msg: "injected unknown fault", transient: true}
	}
}

type faultError struct {
	msg       string
	transient bool
	permanent bool
	conflict  bool
}

func (e *faultError) Error() string { return e.msg }

// IsTransient returns true if this is a transient fault.
func (e *faultError) IsTransient() bool { return e.transient }

// IsPermanent returns true if this is a permanent fault.
func (e *faultError) IsPermanent() bool { return e.permanent }

// IsConflict returns true if this is a conflict fault.
func (e *faultError) IsConflict() bool { return e.conflict }

// Get intercepts Get calls.
func (i *Interceptor) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	kind := ""
	if obj != nil {
		kind = fmt.Sprintf("%T", obj)
	}
	if err := i.shouldFault("get", kind); err != nil {
		return err
	}
	return i.Client.Get(ctx, key, obj, opts...)
}

// Create intercepts Create calls.
func (i *Interceptor) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	kind := ""
	if obj != nil {
		kind = fmt.Sprintf("%T", obj)
	}
	if err := i.shouldFault("create", kind); err != nil {
		return err
	}
	return i.Client.Create(ctx, obj, opts...)
}

// Update intercepts Update calls.
func (i *Interceptor) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	kind := ""
	if obj != nil {
		kind = fmt.Sprintf("%T", obj)
	}
	if err := i.shouldFault("update", kind); err != nil {
		return err
	}
	return i.Client.Update(ctx, obj, opts...)
}

// Patch intercepts Patch calls.
func (i *Interceptor) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	kind := ""
	if obj != nil {
		kind = fmt.Sprintf("%T", obj)
	}
	if err := i.shouldFault("patch", kind); err != nil {
		return err
	}
	return i.Client.Patch(ctx, obj, patch, opts...)
}

// Delete intercepts Delete calls.
func (i *Interceptor) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	kind := ""
	if obj != nil {
		kind = fmt.Sprintf("%T", obj)
	}
	if err := i.shouldFault("delete", kind); err != nil {
		return err
	}
	return i.Client.Delete(ctx, obj, opts...)
}

// StatusWriter returns an intercepting status writer.
func (i *Interceptor) Status() client.SubResourceWriter {
	return &interceptingStatusWriter{
		SubResourceWriter: i.Client.Status(),
		interceptor:       i,
	}
}

type interceptingStatusWriter struct {
	client.SubResourceWriter
	interceptor *Interceptor
}

func (w *interceptingStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	kind := ""
	if obj != nil {
		kind = fmt.Sprintf("%T", obj)
	}
	if err := w.interceptor.shouldFault("status", kind); err != nil {
		return err
	}
	return w.SubResourceWriter.Update(ctx, obj, opts...)
}

// Ensure Interceptor implements client.Client.
var _ client.Client = (*Interceptor)(nil)
