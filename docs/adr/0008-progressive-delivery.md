# ADR-0008: Progressive Delivery with Argo Rollouts

## Status

Accepted

## Date

2025-07-23

## Context

The operator deployment needs zero-downtime update capabilities. Standard
Kubernetes Deployment rolling updates have limitations:
- No traffic weight control during rollout
- No automated health analysis between steps
- No automatic rollback on degraded health

Argo Rollouts provides advanced deployment strategies with built-in
analysis and traffic management capabilities.

## Decision

1. Use **Argo Rollouts** for operator deployment updates
2. Support two strategies:
   - **Canary**: Gradual traffic shift (5% → 20% → 50% → 100%) with analysis
   - **Blue-Green**: Instant traffic switch with pre-promotion analysis
3. Analysis templates verify:
   - Health endpoint (`/healthz`)
   - Reconcile success rate (> 95%)
   - Error rate (< 5%)
   - P95 latency (< 2s)
   - Pod restart count (= 0)
4. Automatic rollback on analysis failure

## Consequences

### Positive
- Zero-downtime deployments with automated health verification
- Automatic rollback prevents bad deployments from affecting users
- Traffic weight control enables safe incremental rollouts
- Analysis templates are reusable across strategies

### Negative
- Requires Argo Rollouts controller installation
- Adds complexity to the deployment pipeline
- Analysis depends on Prometheus availability

### Mitigation
- Provide both Rollout and standard Deployment manifests
- Analysis templates gracefully handle missing Prometheus
- Standard Deployment remains the default installation method
