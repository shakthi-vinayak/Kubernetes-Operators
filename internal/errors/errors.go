package errors

import (
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// ReconcileErrorClass categorizes reconciliation errors for operational
// observability and retry behavior decisions.
type ReconcileErrorClass string

const (
	// ClassTransient indicates a temporary error that should be retried
	// (e.g., API server unavailable, network timeout, rate limiting).
	ClassTransient ReconcileErrorClass = "transient"

	// ClassPermanent indicates an error that will not resolve on its own
	// (e.g., invalid configuration, missing CRD, RBAC denied).
	ClassPermanent ReconcileErrorClass = "permanent"

	// ClassConflict indicates an optimistic concurrency conflict that
	// should be retried with fresh state (e.g., resource version mismatch).
	ClassConflict ReconcileErrorClass = "conflict"

	// ClassUnknown indicates an unclassified error.
	ClassUnknown ReconcileErrorClass = "unknown"
)

// ReconcileError is a structured error that carries its classification
// for metrics labeling, logging, and retry decisions.
type ReconcileError struct {
	Class    ReconcileErrorClass
	Resource string
	Kind     string
	Err      error
}

func (e *ReconcileError) Error() string {
	return fmt.Sprintf("%s error on %s %s: %v", e.Class, e.Kind, e.Resource, e.Err)
}

func (e *ReconcileError) Unwrap() error {
	return e.Err
}

// IsTransient returns true if the error is transient and should be retried.
func (e *ReconcileError) IsTransient() bool {
	return e.Class == ClassTransient
}

// IsPermanent returns true if the error is permanent and should not be retried.
func (e *ReconcileError) IsPermanent() bool {
	return e.Class == ClassPermanent
}

// IsConflict returns true if the error is a conflict and should be retried with fresh state.
func (e *ReconcileError) IsConflict() bool {
	return e.Class == ClassConflict
}

// NewTransientError creates a transient reconciliation error.
func NewTransientError(resource, kind string, err error) *ReconcileError {
	return &ReconcileError{
		Class:    ClassTransient,
		Resource: resource,
		Kind:     kind,
		Err:      err,
	}
}

// NewPermanentError creates a permanent reconciliation error.
func NewPermanentError(resource, kind string, err error) *ReconcileError {
	return &ReconcileError{
		Class:    ClassPermanent,
		Resource: resource,
		Kind:     kind,
		Err:      err,
	}
}

// NewConflictError creates a conflict reconciliation error.
func NewConflictError(resource, kind string, err error) *ReconcileError {
	return &ReconcileError{
		Class:    ClassConflict,
		Resource: resource,
		Kind:     kind,
		Err:      err,
	}
}

// Classify inspects an error and returns its ReconcileErrorClass.
// This is used to classify raw Kubernetes API errors that are not
// already wrapped as ReconcileError.
func Classify(err error) ReconcileErrorClass {
	if err == nil {
		return ""
	}

	// Check if already classified.
	var recErr *ReconcileError
	if errors.As(err, &recErr) {
		return recErr.Class
	}

	// Classify based on Kubernetes API error types.
	if apierrors.IsConflict(err) {
		return ClassConflict
	}
	if apierrors.IsServerTimeout(err) || apierrors.IsTimeout(err) {
		return ClassTransient
	}
	if apierrors.IsTooManyRequests(err) {
		return ClassTransient
	}
	if apierrors.IsServiceUnavailable(err) {
		return ClassTransient
	}
	if apierrors.IsInternalError(err) {
		return ClassTransient
	}
	if apierrors.IsNotFound(err) {
		return ClassPermanent
	}
	if apierrors.IsForbidden(err) {
		return ClassPermanent
	}
	if apierrors.IsInvalid(err) {
		return ClassPermanent
	}
	if apierrors.IsBadRequest(err) {
		return ClassPermanent
	}

	return ClassUnknown
}

// ClassifyOrWrap classifies an error and, if it is not already a
// ReconcileError, wraps it with the appropriate class and metadata.
func ClassifyOrWrap(err error, resource, kind string) *ReconcileError {
	if err == nil {
		return nil
	}

	var recErr *ReconcileError
	if errors.As(err, &recErr) {
		return recErr
	}

	class := Classify(err)
	switch class {
	case ClassConflict:
		return NewConflictError(resource, kind, err)
	case ClassPermanent:
		return NewPermanentError(resource, kind, err)
	case ClassTransient:
		return NewTransientError(resource, kind, err)
	default:
		return NewTransientError(resource, kind, err)
	}
}
