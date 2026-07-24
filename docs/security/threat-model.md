# Security Threat Model

## Overview

The Platform Operator runs as a Kubernetes controller with elevated RBAC permissions to manage child resources. This document identifies threats, attack vectors, and mitigations.

## Architecture Boundary

```
┌─────────────────────────────────────────────────────┐
│ Kubernetes API Server                                │
│  ┌───────────────┐  ┌───────────────────────────┐   │
│  │ PlatformApp   │  │ Child Resources            │   │
│  │ CRs (input)   │  │ Deployment, Service, etc.  │   │
│  └───────┬───────┘  └────────────▲──────────────┘   │
│          │                       │                    │
│          ▼                       │                    │
│  ┌───────────────────────────────┘                    │
│  │  Platform Operator Pod                            │
│  │  (non-root, read-only rootfs, no caps)            │
│  │  ServiceAccount: platform-operator                │
│  └───────────────────────────────────┘               │
└─────────────────────────────────────────────────────┘
```

## Threat Categories

### 1. Container Escape / Compromise

| Threat | Likelihood | Impact | Mitigation |
|--------|-----------|--------|------------|
| Attacker gains code execution in operator pod | Low | Critical | Non-root (UID 65532), read-only rootfs, drop ALL capabilities, seccomp RuntimeDefault, distroless base image |
| Container breakout via kernel vulnerability | Very Low | Critical | Minimal syscall surface via seccomp, no privilege escalation, no host network/PID/IPC |

### 2. RBAC Escalation

| Threat | Likelihood | Impact | Mitigation |
|--------|-----------|--------|------------|
| Operator ServiceAccount compromised | Low | High | Least-privilege RBAC scoped to specific API groups. No cluster-admin. No Secret access |
| RBAC misconfiguration grants excessive permissions | Medium | High | RBAC rules generated from kubebuilder markers, reviewed in PRs. CI validates with `kubectl auth can-i` |
| Operator creates resources with elevated permissions | Low | Critical | Operator only creates resources within its RBAC scope. Child Deployments inherit pod security standards |

### 3. Supply Chain Attack

| Threat | Likelihood | Impact | Mitigation |
|--------|-----------|--------|------------|
| Malicious dependency in go.mod | Low | Critical | `go mod verify`, Dependabot for updates, govulncheck in CI, lock file committed |
| Compromised base image | Low | Critical | Distroless base (gcr.io/distroless/static:nonroot), minimal attack surface, Trivy scanning |
| Compromised CI/CD pipeline | Low | Critical | GitHub Actions with pinned action versions, minimal permissions, branch protection |

### 4. Data Exfiltration

| Threat | Likelihood | Impact | Mitigation |
|--------|-----------|--------|------------|
| Operator logs sensitive data | Medium | Medium | Structured logging via logr. No Secret values logged. Log fields: controller, resource, namespace, operation, result |
| Metrics endpoint exposes sensitive data | Low | Low | Metrics only contain counters/histograms. No resource contents. Bound to pod-internal network |
| Tracing exports sensitive data | Low | Medium | Trace spans contain only resource names/namespaces. No Secret/env values. OTLP endpoint configurable |

### 5. Denial of Service

| Threat | Likelihood | Impact | Mitigation |
|--------|-----------|--------|------------|
| Malicious CR causes infinite reconciliation | Medium | Medium | Idempotent reconcile loop. Error classification with permanent error detection (no infinite retry). Rate limiting via controller-runtime |
| Large number of CRs overwhelm API server | Low | High | `MaxConcurrentReconciles` limits parallelism. Controller-runtime workqueue with exponential backoff |
| Status update conflicts cause hot loop | Low | Medium | Conflict-aware retry with 1s delay. Maximum 3 retries before re-queue |

### 6. Drift / Unauthorized Modification

| Threat | Likelihood | Impact | Mitigation |
|--------|-----------|--------|------------|
| Manual modification of child resources | High | Low | Server-Side Apply with ForceOwnership automatically reverts drift. Alert on high drift rate |
| Another controller modifies same resources | Medium | Medium | Unique field manager (`platform-operator`). Owner references prevent accidental deletion |
| Admission webhook bypassed | Low | High | Webhook configured with `failurePolicy: Fail`. Webhook cert managed by cert-manager |

## RBAC Scope

The operator's ServiceAccount has permissions limited to:

| API Group | Resources | Verbs |
|-----------|-----------|-------|
| `platform.example.io` | `platformapplications`, `/status`, `/finalizers` | get, list, watch, create, update, patch, delete |
| `apps` | `deployments` | get, list, watch, create, update, patch, delete |
| `""` (core) | `services`, `events` | get, list, watch, create, update, patch, delete |
| `autoscaling` | `horizontalpodautoscalers` | get, list, watch, create, update, patch, delete |
| `gateway.networking.k8s.io` | `httproutes` | get, list, watch, create, update, patch, delete |
| `networking.k8s.io` | `networkpolicies` | get, list, watch, create, update, patch, delete |
| `policy` | `poddisruptionbudgets` | get, list, watch, create, update, patch, delete |
| `monitoring.coreos.com` | `servicemonitors` | get, list, watch, create, update, patch, delete |

**Not granted:** Secrets, ConfigMaps, Nodes, Namespaces, ClusterRoles, PersistentVolumes.

## Container Security Profile

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi
```

## Security Best Practices

1. **Always enable leader election** in production to prevent duplicate reconciliation
2. **Use NetworkPolicy** on the operator namespace to restrict ingress to the metrics/health ports only
3. **Rotate webhook certificates** using cert-manager with automatic renewal
4. **Review RBAC** after every API change — regenerate with `make manifests`
5. **Run govulncheck** in CI on every PR and weekly schedule
6. **Scan container images** with Trivy before deployment
7. **Monitor drift rate** — high drift indicates unauthorized modifications
8. **Enable audit logging** on the PlatformApplication API group for compliance
