# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- v1 GA API with full backward compatibility (M21)
- Argo Rollouts progressive delivery (canary + blue-green) (M20)
- OLM operator bundle for OperatorHub distribution (M19)
- Kustomize components for monitoring, security, and HA (M19)
- 1000-CR scale tests with throughput validation (M22)
- Operations guide and troubleshooting documentation (M23)
- CONTRIBUTING, SECURITY, and CHANGELOG files (M24)
- Multi-cluster promotion GitHub Actions workflow (M18)
- Canary deployment workflow with traffic weight control (M18)
- SLO-based alerting with multi-window burn rate (M14)
- Advanced tracing with correlation IDs (M15)
- OPA Gatekeeper and Kyverno policy-as-code (M16)
- Chaos testing and failure injection framework (M17)
- PodDisruptionBudget for high availability (M11)
- Custom rate limiter with configurable concurrency (M12)
- Client interceptor for fault injection testing (M13)

### Changed
- Helm chart improved with NOTES.txt and better defaults
- CI workflow consolidated with chaos and HA test jobs

### Deprecated
- v1alpha1 API (use v1beta1 or v1 instead)
- v1beta1 API will be deprecated in v1.1.0 (use v1 instead)

## [0.1.0] - 2025-01-01

### Added
- Initial PlatformApplication CRD with v1alpha1 API
- Server-Side Apply reconciliation for Deployment, Service, HPA, HTTPRoute, NetworkPolicy, PDB, ServiceMonitor
- Sub-reconciler pattern for modular child resource management
- Error classification (transient, conflict, permanent) with appropriate retry strategies
- Prometheus metrics (reconcile total, duration, errors, status updates)
- Grafana dashboards for operator observability
- OpenTelemetry tracing for reconciliation spans
- Container security scanning (Trivy) and govulncheck
- SBOM generation for supply chain security
- envtest integration tests
- E2E test scaffolding
- Full GitHub Actions CI/CD pipeline
- Helm chart for operator deployment
- Kustomize overlays for dev/staging/production
- Argo CD GitOps ApplicationSet for multi-environment deployment
- v1beta1 API with EnvFrom and PodAnnotations support
- Hub/spoke API conversion between v1alpha1 and v1beta1
- Admission webhooks (defaulting and validating)
- Performance benchmarks and pprof profiling
- Scale testing with 100-app reconciliation
- Leader election for high availability
- Architecture Decision Records (ADRs)
- Operational runbooks for common issues
- Security threat model
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial project scaffolding with Kubebuilder conventions
- `PlatformApplication` CRD (v1alpha1) with full spec/status design
- Core reconciliation engine with idempotent sub-reconcilers
- Deployment sub-reconciler (container image, replicas, resources, probes, security context, rollout strategy)
- Service sub-reconciler (port, type, selectors)
- HorizontalPodAutoscaler sub-reconciler (CPU target, min/max replicas)
- Gateway API HTTPRoute sub-reconciler (hostname, path prefix, parent gateway reference)
- NetworkPolicy sub-reconciler (ingress/egress rules)
- PodDisruptionBudget sub-reconciler (max unavailable, dynamic thresholds)
- ServiceMonitor sub-reconciler (Prometheus Operator integration via unstructured)
- Server-Side Apply (SSA) for drift correction and idempotent reconciliation
- Owner references for automatic garbage collection of child resources
- Finalizer lifecycle for controlled cleanup on deletion
- Status conditions: Ready, Progressing, Degraded, ConfigurationValid
- Prometheus metrics: reconcile_total, reconcile_errors_total, reconcile_duration_seconds, managed_applications
- Structured logging via logr with reconcile ID, resource, namespace fields
- Leader election support for high availability
- Container hardening: non-root, read-only rootfs, drop ALL capabilities, seccomp
- Least-privilege RBAC (no cluster-admin)
- GitHub Actions CI pipeline (lint, test, build, docker build)
- Kustomize configuration for CRDs, RBAC, and manager deployment
- Helm chart scaffolding directory
- Kind cluster configuration for local development
- Comprehensive Makefile with setup, test, build, deploy, and E2E targets
- Multi-stage Dockerfile with distroless base image
- Apache 2.0 license, security policy, contributing guidelines
