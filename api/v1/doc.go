// Package v1 contains the GA (stable) API for PlatformApplication.
//
// This is the stable v1 API for the Platform Operator. It is backward-compatible
// with v1beta1 and adds no new fields — it simply promotes the existing API to
// GA stability. v1alpha1 and v1beta1 remain as spoke versions that convert to
// v1 for storage.
//
// # Migration Guide
//
// To migrate from v1beta1 to v1:
//   - Change apiVersion from platform.example.io/v1beta1 to platform.example.io/v1
//   - No field changes required (fully backward compatible)
//
// # Deprecation Schedule
//   - v1alpha1: Deprecated since v0.2.0, removed in v1.0.0
//   - v1beta1:  Deprecated since v0.3.0, removed in v1.1.0
//   - v1:       Stable, supported indefinitely
package v1
