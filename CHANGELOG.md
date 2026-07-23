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
