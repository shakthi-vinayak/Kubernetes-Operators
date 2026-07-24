# Kubernetes Platform Operator

[![CI](https://github.com/shakthi-vinayak/Kubernetes-Operators/actions/workflows/ci.yaml/badge.svg)](https://github.com/shakthi-vinayak/Kubernetes-Operators/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/example/platform-operator)](https://goreportcard.com/report/github.com/example/platform-operator)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shakthi-vinayak/Kubernetes-Operators)](go.mod)

Kubernetes Platform Operator is a Kubernetes-native application platform controller that provides development teams with a high-level declarative API for deploying and operating production workloads on Kubernetes.

Instead of requiring developers to individually author and maintain Deployments, Services, HPAs, Ingresses, NetworkPolicies, PodDisruptionBudgets, and ServiceMonitors, the Platform Operator exposes a single unified custom resource: **`PlatformApplication`**.

```yaml
apiVersion: platform.example.io/v1alpha1
kind: PlatformApplication
metadata:
  name: payment-service
  namespace: payments
spec:
  image:
    repository: ghcr.io/example/payment-service
    tag: "2.4.1"
  replicas:
    min: 3
    max: 10
  service:
    port: 8080
  autoscaling:
    enabled: true
    targetCPUUtilization: 70
  gateway:
    enabled: true
    host: payments.example.com
  security:
    networkPolicy: true
  observability:
    metrics: true
  resources:
    requests:
      cpu: "100m"
      memory: "128Mi"
    limits:
      cpu: "500m"
      memory: "512Mi"
```

From this single resource, the operator generates and continuously reconciles a complete set of production-ready Kubernetes resources.

---

## Key Features

| Feature | Description |
|---------|-------------|
| **Declarative API** | Single `PlatformApplication` CR replaces 7+ individual Kubernetes resources |
| **Idempotent Reconciliation** | Desired state is always enforced; duplicate events and restarts are handled safely |
| **Drift Correction** | Manual changes to managed resources are automatically detected and reverted via Server-Side Apply |
| **Gateway API** | Native HTTPRoute support for modern Kubernetes ingress (no Ingress resource dependency) |
| **Autoscaling** | HPA generation with configurable CPU targets and min/max replica bounds |
| **Security by Default** | Non-root containers, read-only rootfs, dropped capabilities, seccomp, NetworkPolicy generation |
| **Observability** | Prometheus metrics, structured logging, optional ServiceMonitor generation for Prometheus Operator |
| **High Availability** | Leader election for multi-replica operator deployments with automatic failover |
| **Admission Control** | Defaulting and validating webhooks catch configuration errors before they reach the cluster |
| **Status Conditions** | Kubernetes-native status with Ready, Progressing, Degraded, and ConfigurationValid conditions |
| **Owner References** | Automatic garbage collection of all child resources when a PlatformApplication is deleted |
| **Finalizers** | Controlled cleanup lifecycle for operations that require explicit teardown |

---

## Architecture

```
Developer / GitOps
        |
        v
Kubernetes API Server
        |
        v
PlatformApplication CR
        |
        v
Platform Operator (controller-runtime)
        |
        +-- Reconcile Deployment      (container, replicas, probes, rollout)
        +-- Reconcile Service          (ports, selectors, type)
        +-- Reconcile HPA             (autoscaling, CPU targets)
        +-- Reconcile HTTPRoute       (Gateway API, hostnames, path matching)
        +-- Reconcile NetworkPolicy   (ingress/egress rules)
        +-- Reconcile PDB            (pod disruption budgets)
        +-- Reconcile ServiceMonitor  (Prometheus Operator integration)
        |
        v
Status Update (conditions, observedGeneration, readyReplicas, URL)
```

The operator follows the **sub-reconciler pattern**: a single `PlatformApplicationReconciler` orchestrates independent sub-reconcilers, each responsible for computing the desired state of one child resource. A shared Server-Side Apply mechanism handles all cluster interactions idempotently.

### Managed Resources

```
PlatformApplication
    ├── Deployment       (always created)
    ├── Service          (always created)
    ├── HPA              (when autoscaling.enabled = true)
    ├── HTTPRoute        (when gateway.enabled = true)
    ├── NetworkPolicy    (when security.networkPolicy = true)
    ├── PodDisruptionBudget  (when replicas.min > 1)
    └── ServiceMonitor   (when observability.metrics = true)
```

All child resources carry **owner references** pointing to the parent PlatformApplication, enabling Kubernetes' built-in garbage collector to clean up automatically on deletion.

---

## Quick Start

### Prerequisites

- Kubernetes cluster (v1.28+)
- `kubectl` configured with cluster access
- [Gateway API](https://gateway-api.sigs.k8s.io/) CRDs installed (for HTTPRoute support)

### Install the Operator

```bash
# Install CRDs
kubectl apply -f config/crd/bases/

# Deploy the operator
kubectl apply -k config/default/
```

### Create Your First Application

```bash
kubectl apply -f config/samples/platform_v1alpha1_platformapplication.yaml
```

### Verify

```bash
# Check the PlatformApplication status
kubectl get platformapplication

# View generated resources
kubectl get deployment,service,hpa,httproute,networkpolicy,pdb

# Watch reconciliation in action
kubectl logs -l control-plane=controller-manager -n platform-operator-system
```

---

## Reconciliation Model

The operator implements a **continuous reconciliation loop** based on the control-loop pattern:

1. **Fetch** the PlatformApplication resource
2. **Handle deletion** — if the resource is being deleted, run finalizer cleanup and allow GC
3. **Compute desired state** — each sub-reconciler generates the desired child resource as a pure function of the spec
4. **Apply** — use Server-Side Apply to create or update each child resource idempotently
5. **Detect drift** — SSA compares desired vs. actual state; only writes when changes are needed
6. **Update status** — set conditions, observedGeneration, readyReplicas, and URL

This design ensures:
- **Idempotency** — calling reconcile N times with the same input produces the same result
- **Restart safety** — the operator can crash and restart without losing state
- **Duplicate tolerance** — duplicate watch events do not cause duplicate API writes
- **Eventual consistency** — transient failures trigger automatic retries with exponential backoff

---

## API Reference

### PlatformApplication Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `image.repository` | string | Yes | Container image repository |
| `image.tag` | string | Yes | Container image tag |
| `image.pullPolicy` | string | No | Pull policy (default: `IfNotPresent`) |
| `replicas.min` | int32 | Yes | Minimum replica count (>= 1) |
| `replicas.max` | int32 | No | Maximum replica count (defaults to min) |
| `service.port` | int32 | Yes | Application port (1-65535) |
| `service.type` | string | No | Service type (default: `ClusterIP`) |
| `configuration` | map | No | Key-value pairs injected as environment variables |
| `autoscaling.enabled` | bool | No | Enable HPA generation |
| `autoscaling.targetCPUUtilization` | int32 | No | Target CPU utilization % (default: 80) |
| `gateway.enabled` | bool | No | Enable HTTPRoute generation |
| `gateway.host` | string | No | Hostname for the HTTPRoute |
| `gateway.gatewayRef` | string | No | Parent Gateway reference (`namespace/name`) |
| `gateway.pathPrefix` | string | No | Path prefix match (default: `/`) |
| `observability.metrics` | bool | No | Enable ServiceMonitor generation |
| `security.networkPolicy` | bool | No | Enable NetworkPolicy generation |
| `resources.requests.cpu` | string | No | CPU request (e.g., `100m`) |
| `resources.requests.memory` | string | No | Memory request (e.g., `128Mi`) |
| `resources.limits.cpu` | string | No | CPU limit |
| `resources.limits.memory` | string | No | Memory limit |
| `healthChecks.livenessPath` | string | No | Liveness probe path (default: `/healthz`) |
| `healthChecks.readinessPath` | string | No | Readiness probe path (default: `/readyz`) |
| `rollout.strategy` | string | No | `RollingUpdate` or `Recreate` (default: `RollingUpdate`) |

#### v1beta1 Additional Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `envFrom` | `[]EnvFromSource` | No | Environment variables sourced from ConfigMaps or Secrets |
| `podAnnotations` | `map[string]string` | No | Custom annotations applied to all generated pods |

### Status Conditions

| Condition | Meaning |
|-----------|---------|
| `Ready` | All child resources reconciled; Deployment has sufficient ready replicas |
| `Progressing` | A rollout or reconciliation is in progress |
| `Degraded` | Something is wrong but the application is partially functional |
| `ConfigurationValid` | The spec passed all validation checks |

---

## High Availability

The operator supports running multiple replicas with **Lease-based leader election**:

```
Operator Pod A (leader)  ─── active reconciliation
Operator Pod B (standby) ─── watching lease
Operator Pod C (standby) ─── watching lease
        |
        v
Kubernetes Lease (coordination.k8s.io/v1)
```

When the leader terminates, a standby pod acquires the lease and resumes reconciliation within seconds. No duplicate processing occurs because only the leader performs controller work.

Enable leader election in the manager deployment:

```yaml
args:
  - --leader-elect
```

---

## Security

The operator follows production Kubernetes security practices:

**Container hardening:**
- Runs as non-root (UID 65532, distroless image)
- Read-only root filesystem
- All Linux capabilities dropped
- Seccomp profile: `RuntimeDefault`
- No privilege escalation
- Resource requests and limits configured

**RBAC:**
- Least-privilege ClusterRole scoped to specific API groups
- No cluster-admin permissions
- Cannot access Secrets or unrelated resources

**Network:**
- Generated NetworkPolicies restrict ingress to the application port
- Egress allowed for DNS and external services

See [SECURITY.md](SECURITY.md) for the full threat model and vulnerability reporting process.

---

## Observability

### Metrics

The operator exposes Prometheus metrics on `:8080/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `platform_operator_reconcile_total` | Counter | Total reconciliation attempts (labeled by result) |
| `platform_operator_reconcile_errors_total` | Counter | Total reconciliation errors (labeled by error type) |
| `platform_operator_reconcile_duration_seconds` | Histogram | Reconciliation duration distribution |
| `platform_operator_managed_applications` | Gauge | Current number of managed PlatformApplications |
| `platform_operator_api_call_duration_seconds` | Histogram | Duration of Kubernetes API calls (get, create, patch, delete) |
| `platform_operator_noop_apply_total` | Counter | Apply operations where no changes were detected (semantic equality) |
| `platform_operator_active_reconcilers` | Gauge | Number of currently active reconcile workers |

### Structured Logging

All log entries use structured JSON via `logr` with standard fields:

```json
{"level":"info","controller":"platformapplication","resource":"payment-service",
 "namespace":"payments","operation":"reconcile","result":"success","duration":0.042}
```

No secrets are ever logged.

---

## Performance

The operator is designed for high performance and low overhead:

**Benchmark results** (Intel i5-2500, single-threaded):

| Operation | Time/op | Allocations |
|-----------|---------|-------------|
| Build all 6 resources | ~16µs | 90 allocs |
| Build Deployment | ~6.3µs | 37 allocs |
| Build Service | ~1.3µs | 6 allocs |
| Build HPA | ~1.1µs | 7 allocs |
| Build HTTPRoute | ~2.0µs | 18 allocs |
| Build NetworkPolicy | ~2.0µs | 13 allocs |
| Build PDB | ~1.6µs | 9 allocs |

**Scale testing** (100 concurrent reconciliations):

- 100 apps reconciled in ~73ms (0.73ms/app)
- Second pass (idempotent): detects no changes needed, minimal API writes

**Profiling** is available via pprof:

```bash
# Start operator with pprof enabled
./manager --pprof-bind-address=:6060

# Profile CPU
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Profile memory
go tool pprof http://localhost:6060/debug/pprof/heap

# View goroutine stacks
curl http://localhost:6060/debug/pprof/goroutine?debug=1
```

---

## Testing

The project implements a testing pyramid:

```bash
# Unit tests — pure functions (resource generation, validation, conditions)
go test ./internal/... ./api/...

# Race detection
go test -race ./...

# Benchmarks — performance testing for all sub-reconcilers
go test -bench=. -benchmem ./internal/...

# Scale tests — 100+ concurrent reconciliations
go test -run=TestScale ./internal/controller/ -v

# All tests
go test ./... -v
```

| Layer | Tool | Scope |
|-------|------|-------|
| Unit | `go test` | Resource generation, validation, label computation, condition helpers, conversion |
| Benchmarks | `go test -bench` | Sub-reconciler performance, label ops, error classification, full cycle |
| Scale | `go test -run=TestScale` | 100-app reconciliation, idempotency verification |
| Integration | envtest | Full reconciler against real API server, webhook behavior |
| E2E | Kind | End-to-end: install operator, create CR, verify resources, drift test, cleanup |
| Profiling | pprof | CPU/memory profiling via `--pprof-bind-address=:6060` |

---

## Development

### Prerequisites

- Go 1.22+
- Docker
- kubectl
- [Kind](https://kind.sigs.k8s.io/) (for local cluster)
- Make

### Local Development

```bash
# Clone the repository
git clone https://github.com/shakthi-vinayak/Kubernetes-Operators.git
cd Kubernetes-Operators

# Install development tools
make setup

# Build the binary
make build

# Run unit tests
make test-unit

# Create a local Kind cluster
make kind-create

# Install CRDs into the cluster
make install

# Run the operator locally (watches the Kind cluster)
make run

# In another terminal, create a PlatformApplication
kubectl apply -f config/samples/platform_v1alpha1_platformapplication.yaml
```

### Project Structure

```
├── api/
│   ├── v1alpha1/            # CRD types, deepcopy, webhooks, conversion (spoke)
│   └── v1beta1/             # Evolved CRD types, deepcopy, webhooks (hub, storage version)
├── cmd/                     # Entry point (main.go)
├── internal/
│   ├── controller/          # Reconciliation logic, scale tests
│   │   └── subreconcilers/  # Per-resource reconciliation + benchmarks
│   ├── errors/              # Distributed systems error classification
│   ├── metrics/             # Prometheus metrics registration
│   └── status/              # Status condition helpers
├── config/
│   ├── crd/                 # CRD Kustomize manifests
│   ├── manager/             # Operator Deployment manifest
│   ├── rbac/                # RBAC ClusterRole and bindings
│   ├── default/             # Kustomize overlay combining all configs
│   ├── overlays/            # Environment-specific overlays (dev/staging/production)
│   ├── webhook/             # Webhook manifests with cert-manager integration
│   └── samples/             # Example PlatformApplication resources (v1alpha1 + v1beta1)
├── charts/
│   └── platform-operator/   # Helm chart with templates, values, helpers
├── gitops/                  # Argo CD Application, ApplicationSet, Helm-based examples
├── test/                    # Integration and E2E test suites
├── hack/                    # Development scripts and Kind config
├── docs/                    # Architecture and operational documentation
├── Dockerfile               # Multi-stage build with distroless
├── Makefile                 # Build, test, deploy, and development targets
└── PROJECT                  # Kubebuilder project metadata
```

---

## API Versions

The operator supports two API versions with automatic conversion:

| Version | Status | Notes |
|---------|--------|-------|
| `platform.example.io/v1alpha1` | Spoke | Original API, fully supported |
| `platform.example.io/v1beta1` | Hub (Storage) | Adds `envFrom` and `podAnnotations` fields |

Conversion between versions is handled automatically by conversion webhooks. Objects are stored in the v1beta1 format (hub) and served in either version on request.

**v1beta1 new fields:**

| Field | Type | Description |
|-------|------|-------------|
| `envFrom` | `[]EnvFromSource` | Environment variables from ConfigMaps/Secrets |
| `podAnnotations` | `map[string]string` | Custom annotations applied to all pods |

---

## GitOps

The operator is designed for GitOps workflows with [Argo CD](https://argo-cd.readthedocs.io/):

```bash
# Deploy with Argo CD Application (single environment)
kubectl apply -f gitops/application.yaml

# Deploy with Argo CD ApplicationSet (multi-environment rollout)
kubectl apply -f gitops/applicationset.yaml

# Deploy with Helm-based Argo CD Application
kubectl apply -f gitops/application-helm.yaml
```

Environment-specific Kustomize overlays are provided:

| Overlay | Path | Use Case |
|---------|------|----------|
| `dev` | `config/overlays/dev/` | Single replica, no leader election, minimal resources |
| `staging` | `config/overlays/staging/` | 2 replicas, moderate resources |
| `production` | `config/overlays/production/` | 2 replicas, HA, topology spread, monitoring |

---

## Installation

### Kustomize Overlay

```bash
# Deploy a specific environment
kubectl apply -k config/overlays/dev/
kubectl apply -k config/overlays/staging/
kubectl apply -k config/overlays/production/
```

### Helm

```bash
helm install platform-operator charts/platform-operator \
  --namespace platform-operator-system \
  --create-namespace
```

---

## Roadmap

| Milestone | Status | Description |
|-----------|--------|-------------|
| M1: Project Foundation | Done | Go module, Kubebuilder scaffold, Makefile, Dockerfile, linting, CI skeleton, GitHub templates |
| M2: API Hardening | Done | Unit test coverage, ADRs, documentation |
| M3: Production Hardening | Done | Distributed systems error handling, HA testing, concurrency, failure engineering |
| M4: Observability | Done | Grafana dashboards, alerting rules, runbooks |
| M5: Tracing & Security | Done | OpenTelemetry spans, container scanning, SBOM, supply chain hardening |
| M6: Integration Testing | Done | envtest integration tests, E2E scaffolding, full CI pipeline |
| M7: CI/CD | Done | Full GitHub Actions pipeline, release automation, multi-arch builds, Helm chart |
| M8: Packaging & GitOps | Done | Kustomize overlays (dev/staging/prod), webhook manifests, Argo CD examples |
| M9: API Evolution | Done | v1beta1 API with EnvFrom/PodAnnotations, hub/spoke conversion webhooks |
| M10: Scale & Performance | Done | Benchmarks, scale testing (100-app), pprof profiling, performance metrics |
| M11: High Availability | Planned | Leader election tuning, multi-replica deployment, failover testing |
| M12: Concurrency | Planned | Configurable worker concurrency, race-safe shared state |
| M13: Failure Engineering | Planned | Fault injection tests, error categorization |
| M14: Advanced Observability | Planned | Enhanced Grafana dashboards, SLO-based alerting |
| M15: Advanced Tracing | Planned | Per-sub-reconciler OpenTelemetry spans with correlation IDs |
| M16: Advanced Security | Planned | Policy-as-code (OPA/Kyverno), signed artifacts, SLSA Level 3 |
| M17: Extended Testing | Planned | Chaos testing, mutation testing, coverage thresholds |
| M18: Extended CI/CD | Planned | Canary deployments, automated rollback, multi-cluster promotion |
| M19: Advanced Packaging | Planned | Helm operator bundles, OLM catalog, Kustomize components |
| M20: Advanced GitOps | Planned | Progressive delivery (Argo Rollouts), blue-green/canary strategies |
| M21: API Maturity | Planned | v1 GA API, field deprecation, conversion strategy validation |
| M22: Scale & Optimization | Planned | 1000+ CR scale testing, caching optimization, rate limiting |
| M23: Operations | Planned | Operations guide, troubleshooting guide, demo scenario |
| M24: Documentation | Planned | All docs, ADRs, README, CONTRIBUTING, SECURITY, CHANGELOG |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow, coding standards, and pull request guidelines.

## Security

See [SECURITY.md](SECURITY.md) for vulnerability reporting, threat model, and security best practices.

## License

[Apache License 2.0](LICENSE)
