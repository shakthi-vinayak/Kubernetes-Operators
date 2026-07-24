# ADR-0007: v1 GA API Stability

## Status

Accepted

## Date

2025-07-23

## Context

The PlatformApplication API has evolved through v1alpha1 and v1beta1 versions.
v1beta1 added EnvFrom and PodAnnotations fields and became the storage hub.
The API has been stable for multiple releases with no breaking changes needed.
It's time to promote the API to GA (v1) to signal stability to users.

## Decision

1. Create a v1 API that is **identical** to v1beta1 (no new fields, no removed fields).
2. v1beta1 remains the **storage version** (hub) to avoid data migration.
3. v1 is a **spoke** version that converts to/from v1beta1 transparently.
4. v1alpha1 is **deprecated** and will be removed in v1.0.0.
5. v1beta1 will be **deprecated** in v1.1.0 and removed in a future release.

## Consequences

### Positive
- Users get a stable v1 API guarantee
- No data migration required (v1beta1 remains storage)
- Conversion is lossless (1:1 field mapping)
- Existing v1beta1 manifests continue to work

### Negative
- Three API versions to maintain temporarily
- Users on v1alpha1 must migrate before v1.0.0

### Mitigation
- Add deprecation warnings to v1alpha1 webhooks
- Provide migration guide in documentation
- Automated conversion testing in CI
