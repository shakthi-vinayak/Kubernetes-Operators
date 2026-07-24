# File-by-File Explanation

This document describes every file in the repository, what it does, and the logic it contains.

**Total files: 159** (excluding `.git/` and `.qoder/` directories)

---

## Table of Contents

- [Root Files](#root-files)
- [cmd/ — Entry Point](#cmd--entry-point)
- [api/ — Custom Resource Definitions](#api--custom-resource-definitions)
- [internal/ — Core Logic](#internal--core-logic)
- [config/ — Kubernetes Manifests](#config--kubernetes-manifests)
- [charts/ — Helm Chart](#charts--helm-chart)
- [gitops/ — GitOps Configurations](#gitops--gitops-configurations)
- [bundles/ — OLM Bundle](#bundles--olm-bundle)
- [test/ — Test Suites](#test--test-suites)
- [.github/ — CI/CD and Templates](#github--cicd-and-templates)
- [docs/ — Documentation](#docs--documentation)
- [hack/ — Development Scripts](#hack--development-scripts)
- [examples/ — Usage Examples](#examples--usage-examples)

---

## Root Files

### `README.md`
The main project README. Contains project overview, architecture diagrams (Mermaid), "What is a Kubernetes Operator" educational section, "How This Operator Helps Development" benefits section, "What Gets Automated" resource generation breakdown, 4 practical API usage examples (simple web service, production API, background worker, stateful service), common operations guide, "Updating & Extending This Operator" future development guide, API reference, installation instructions, performance benchmarks, testing guide, and 24-milestone roadmap table.

### `Makefile`
Build automation with targets for development workflow. Key targets:
- `build` — Compiles the manager binary via `go build`
- `run` — Runs the operator locally against the current kubeconfig
- `test` / `test-unit` / `test-integration` / `test-race` / `test-chaos` / `test-bench` / `test-scale` — Various test suites
- `manifests` — Generates CRDs, RBAC, and webhook configs via `controller-gen`
- `generate` — Generates DeepCopy methods via `controller-gen`
- `install` / `deploy` — Installs CRDs and deploys operator to cluster
- `kind-create` / `kind-delete` — Manages local Kind cluster
- `docker-build` / `docker-push` — Container image build and push
- `setup` — Installs all development dependencies
- `clean` — Removes build artifacts

### `Dockerfile`
Multi-stage container build. Stage 1 uses `golang:1.26` to compile a static binary. Stage 2 uses `gcr.io/distroless/static:nonroot` as the runtime image (UID 65532, no shell, minimal attack surface). The binary is copied and set as the entrypoint.

### `go.mod`
Go module definition (`github.com/example/platform-operator`). Go 1.26.2. Key dependencies: `controller-runtime` (Kubernetes controller framework), `k8s.io/api` / `apimachinery` / `client-go` (Kubernetes client libraries), `sigs.k8s.io/gateway-api` (Gateway API types), `prometheus/client_golang` (metrics), `go.opentelemetry.io/otel` (distributed tracing).

### `go.sum`
Cryptographic checksums for all Go module dependencies. Ensures reproducible builds.

### `PROJECT`
Kubebuilder project metadata. Defines the project domain (`example.io`), group (`platform`), kind (`PlatformApplication`), API version (`v1alpha1`), and webhook configuration (defaulting + validation webhooks enabled).

### `LICENSE`
Apache License 2.0 — the open-source license under which this project is distributed.

### `CONTRIBUTING.md`
Comprehensive contributing guide. Covers development workflow, coding standards, testing requirements (unit, integration, chaos, scale), pull request process, release process, and commit message conventions.

### `SECURITY.md`
Security policy. Covers vulnerability reporting process, threat model (supply chain, runtime, network, data security), security best practices, and incident response procedures.

### `CHANGELOG.md`
Versioned changelog following Keep a Changelog format. Documents all features added across milestones M1-M24, organized by version (Unreleased, 0.1.0).

### `.gitignore`
Git ignore rules. Excludes build artifacts (`bin/`, `cover.out`), IDE files, and Kubernetes secret files.

### `.dockerignore`
Docker build context ignore rules. Excludes `.git`, `bin/`, test files, and documentation from Docker build context to reduce image build time.

### `.golangci.yml`
golangci-lint configuration. Enables linters for code quality: gofmt, govet, ineffassign, deadcode, errcheck, staticcheck, and others. Configures exclusion rules for generated files.

---

## cmd/ — Entry Point

### `cmd/main.go`
The operator's main entry point. Initializes the controller-runtime manager with:
- **Scheme registration** — Registers v1alpha1, v1beta1, and core Kubernetes types
- **CLI flags** — Metrics address, health probe address, pprof address, leader election, concurrency, tracing configuration
- **Tracing initialization** — Sets up OpenTelemetry with OTLP or stdout exporter (no-op when disabled)
- **Manager creation** — Creates the controller-runtime manager with metrics, health probes, webhook server, and leader election
- **Controller registration** — Registers `PlatformApplicationReconciler` with configurable concurrency
- **Webhook registration** — Registers defaulting and validating webhooks for both v1alpha1 and v1beta1
- **Health checks** — Registers `/healthz` and `/readyz` endpoints
- **Graceful shutdown** — Starts the manager with signal handling for SIGTERM/SIGINT

---

## api/ — Custom Resource Definitions

### `api/v1alpha1/groupversion_info.go`
Registers the `platform.example.io/v1alpha1` API group with the Kubernetes runtime scheme. Defines `GroupVersion`, `SchemeBuilder`, and `AddToScheme` for type registration.

### `api/v1alpha1/platformapplication_types.go`
Defines the original v1alpha1 CRD types:
- **`PlatformApplicationSpec`** — Desired state: image, replicas, service, configuration, autoscaling, gateway, observability, security, resources, healthChecks, rollout
- **`PlatformApplicationStatus`** — Observed state: observedGeneration, readyReplicas, URL, conditions, deployedVersion
- **Sub-types** — `ImageSpec`, `ReplicasSpec`, `ServiceSpec`, `AutoscalingSpec`, `GatewaySpec`, `ObservabilitySpec`, `SecuritySpec`, `ResourcesSpec`, `HealthChecksSpec`, `RolloutSpec`
- **Condition constants** — `Ready`, `Progressing`, `Degraded`, `ConfigurationValid`
- **Kubebuilder markers** — Validation constraints (min/max, enums, required), print columns, short name (`papp`), status subresource

### `api/v1alpha1/webhook_platformapplication.go`
Admission webhooks for v1alpha1:
- **`PlatformApplicationDefaulter`** — Sets defaults: pullPolicy=`IfNotPresent`, max replicas=min, service type=`ClusterIP`, CPU target=80, health paths, rollout strategy=`RollingUpdate`, gateway path prefix=`/`
- **`PlatformApplicationValidator`** — Validates: min<=max replicas, min>=1, image repository/tag non-empty, port range 1-65535. Emits warnings for ineffective autoscaling or gateway without host.

### `api/v1alpha1/platformapplication_conversion.go`
Hub/spoke conversion for v1alpha1 (spoke) ↔ v1beta1 (hub):
- **`ConvertTo`** — Converts v1alpha1 to v1beta1. v1beta1-only fields (`EnvFrom`, `PodAnnotations`) are set to nil.
- **`ConvertFrom`** — Converts v1beta1 to v1alpha1. v1beta1-only fields are dropped (lossy for those fields).

### `api/v1alpha1/zz_generated.deepcopy.go`
Auto-generated DeepCopy methods for all v1alpha1 types. Each type gets `DeepCopyInto`, `DeepCopy`, and `DeepCopyObject` (for runtime.Object implementers). Generated by `controller-gen`.

### `api/v1alpha1/platformapplication_conversion_test.go`
Unit tests for the v1alpha1 ↔ v1beta1 conversion logic. Verifies field mapping is correct and lossless for shared fields.

### `api/v1alpha1/webhook_platformapplication_test.go`
Unit tests for the v1alpha1 defaulting and validating webhooks. Tests default values are set correctly and validation rejects invalid configurations.

### `api/v1beta1/groupversion_info.go`
Registers the `platform.example.io/v1beta1` API group. Same structure as v1alpha1 but for the v1beta1 version.

### `api/v1beta1/platformapplication_types.go`
Defines the v1beta1 CRD types — identical to v1alpha1 plus two new fields:
- **`EnvFrom`** (`[]corev1.EnvFromSource`) — Environment variables from ConfigMaps/Secrets
- **`PodAnnotations`** (`map[string]string`) — Custom annotations applied to pods
- **`Hub()` method** — Marks v1beta1 as the conversion hub (storage version) via `+kubebuilder:storageversion`

### `api/v1beta1/webhook_platformapplication.go`
Admission webhooks for v1beta1 — same defaulting and validation logic as v1alpha1, but operates on v1beta1 types.

### `api/v1beta1/zz_generated.deepcopy.go`
Auto-generated DeepCopy methods for all v1beta1 types, including DeepCopy for `EnvFrom` (slice) and `PodAnnotations` (map).

### `api/v1/groupversion_info.go`
Registers the `platform.example.io/v1` API group. Same structure as other versions but for the stable v1 GA API.

### `api/v1/doc.go`
Package documentation for the v1 GA API. Includes migration guide (change apiVersion from v1beta1 to v1, no field changes) and deprecation schedule (v1alpha1 removed in v1.0.0, v1beta1 removed in v1.1.0, v1 stable indefinitely).

### `api/v1/platformapplication_types.go`
Defines the v1 GA (stable) CRD types — identical to v1beta1 with `EnvFrom` and `PodAnnotations` fields. v1 is a spoke that converts to v1beta1 (hub). No `Hub()` method (it's a spoke, not the hub).

### `api/v1/platformapplication_conversion.go`
Hub/spoke conversion for v1 (spoke) ↔ v1beta1 (hub). Since v1 and v1beta1 have identical fields, conversion is a 1:1 field mapping — fully lossless.

### `api/v1/zz_generated.deepcopy.go`
Auto-generated DeepCopy methods for all v1 types, including `EnvFrom` and `PodAnnotations`.

---

## internal/ — Core Logic

### `internal/controller/platformapplication_controller.go`
The main reconciliation controller. Contains:
- **`PlatformApplicationReconciler`** struct — Holds Kubernetes client, scheme, event recorder, and concurrency setting
- **`Reconcile()`** — The main reconciliation loop:
  1. Fetch the PlatformApplication resource
  2. Handle deletion via finalizer
  3. Add finalizer if missing
  4. Mark as Progressing
  5. Call `reconcileResources()` to apply all 7 child resources
  6. Update status (observedGeneration, deployedVersion, readyReplicas, URL)
  7. Set conditions (Ready, Progressing, Degraded) based on outcome
  8. Return appropriate requeue behavior
- **`reconcileResources()`** — Orchestrates 7 sub-reconcilers in sequence with per-reconciler tracing spans and metrics
- **`setConditions()`** — Sets Ready/Progressing/Degraded based on reconcile outcome and replica readiness
- **`handleErrorResult()`** — Classifies errors (transient/conflict/permanent) and sets appropriate requeue delays
- **`handleDeletion()`** — Removes finalizer to allow garbage collection
- **`updateDeploymentStatus()`** — Fetches Deployment to update readyReplicas
- **`SetupWithManager()`** — Configures controller with concurrency, rate limiter, and watches PlatformApplication + owns Deployment

### `internal/controller/subreconcilers/apply.go`
The shared Server-Side Apply (SSA) mechanism:
- **`Apply()`** — Creates or updates a resource using SSA. Sets owner reference for garbage collection. Uses semantic equality to skip no-op applies. Retries on conflict with exponential backoff (up to 3 retries, 100ms base delay).
- **`applyOnce()`** — Single apply attempt: Get → if not found Create, else compare with DeepEqual → if different Patch with SSA
- **`DeleteIfExists()`** — Idempotent delete (returns nil if not found)
- **`CommonLabels()`** — Returns standard labels: `app.kubernetes.io/name`, `managed-by`, `part-of`
- **`MergeLabels()`** — Merges multiple label maps

### `internal/controller/subreconcilers/deployment.go`
Deployment sub-reconciler:
- **`ReconcileDeployment()`** — Builds desired Deployment and applies it
- **`buildDesiredDeployment()`** — Pure function that constructs the Deployment:
  - Container with image, ports, security context (non-root, read-only rootfs, dropped capabilities)
  - Resource requests and limits
  - Liveness probe (HTTP GET on configurable path/port, 15s initial delay)
  - Readiness probe (HTTP GET on configurable path/port, 5s initial delay)
  - Environment variables from `configuration` map
  - Rolling update strategy with configurable maxUnavailable/maxSurge
  - Pod security context (RunAsNonRoot, SeccompProfile RuntimeDefault)

### `internal/controller/subreconcilers/service.go`
Service sub-reconciler:
- **`ReconcileService()`** — Builds desired Service and applies it
- **`buildDesiredService()`** — Constructs Service with type (ClusterIP/NodePort/LoadBalancer), port mapping, and label selector

### `internal/controller/subreconcilers/hpa.go`
HorizontalPodAutoscaler sub-reconciler:
- **`ReconcileHPA()`** — If autoscaling disabled, deletes existing HPA. Otherwise builds and applies HPA.
- **`buildDesiredHPA()`** — Constructs HPA with min/max replicas, CPU utilization target, and scale target reference pointing to the Deployment

### `internal/controller/subreconcilers/networking.go`
Networking sub-reconcilers (HTTPRoute + NetworkPolicy):
- **`ReconcileHTTPRoute()`** — If gateway disabled, deletes HTTPRoute. Otherwise builds and applies HTTPRoute with Gateway API.
- **`buildDesiredHTTPRoute()`** — Constructs HTTPRoute with path prefix match, hostname, backend service reference, and parent gateway reference
- **`ReconcileNetworkPolicy()`** — If security disabled, deletes NetworkPolicy. Otherwise builds and applies NetworkPolicy.
- **`buildDesiredNetworkPolicy()`** — Constructs NetworkPolicy restricting ingress to the service port and allowing all egress

### `internal/controller/subreconcilers/pdb.go`
PodDisruptionBudget sub-reconciler:
- **`ReconcilePDB()`** — If min replicas <= 1, deletes PDB (single replica doesn't benefit). Otherwise builds and applies PDB.
- **`buildDesiredPDB()`** — Constructs PDB with maxUnavailable=1 (for small deployments) or 25% (for deployments with >4 replicas)

### `internal/controller/subreconcilers/servicemonitor.go`
ServiceMonitor sub-reconciler:
- **`ReconcileServiceMonitor()`** — If metrics disabled, deletes ServiceMonitor. Otherwise builds and applies ServiceMonitor.
- **`buildDesiredServiceMonitor()`** — Constructs ServiceMonitor using unstructured objects (avoids hard dependency on Prometheus Operator types). Configures scrape on port "http" at path "/metrics" with 30s interval.

### `internal/errors/errors.go`
Distributed systems error classification:
- **`ReconcileErrorClass`** — Four classes: `Transient` (retry), `Permanent` (no retry), `Conflict` (retry with fresh state), `Unknown`
- **`ReconcileError`** — Structured error with class, resource name, kind, and wrapped error
- **`Classify()`** — Inspects raw Kubernetes API errors and maps them to error classes:
  - Conflict → `ClassConflict`
  - ServerTimeout, TooManyRequests, ServiceUnavailable, InternalError → `ClassTransient`
  - NotFound, Forbidden, Invalid, BadRequest → `ClassPermanent`
- **`ClassifyOrWrap()`** — Classifies and wraps unknown errors into `ReconcileError` with metadata

### `internal/metrics/metrics.go`
Prometheus metrics registration and observation helpers:
- **Counters** — `reconcile_total`, `reconcile_errors_total`, `sub_reconcile_total`, `resource_apply_total`, `reconcile_requeue_total`, `status_update_total`, `noop_apply_total`
- **Histograms** — `reconcile_duration_seconds`, `sub_reconcile_duration_seconds`, `api_call_duration_seconds`, `workqueue_latency_seconds`
- **Gauges** — `managed_applications`, `active_reconcilers`, `workqueue_depth`
- **Helper functions** — `ObserveReconcile()`, `ObserveSubReconcile()`, `ObserveResourceApply()`, `ObserveError()`, `ObserveAPICall()`, `ObserveNoOpApply()`
- All metrics are registered with the controller-runtime metrics registry in `init()`

### `internal/status/conditions.go`
Status condition helpers:
- **`SetCondition()`** — Sets or updates a condition on the status. Only updates if status, reason, or message has changed (avoids unnecessary API writes). Sets `LastTransitionTime` on changes.
- **`SetReady()` / `SetProgressing()` / `SetDegraded()` / `SetConfigurationValid()`** — Convenience wrappers for each condition type
- **`IsReady()`** — Returns true if the Ready condition is True

### `internal/tracing/tracing.go`
OpenTelemetry distributed tracing:
- **`Config`** — Tracing configuration: enabled, exporter (otlp/stdout/none), endpoint, service name, insecure
- **`Init()`** — Initializes the global tracer provider. Returns no-op provider when disabled. Supports OTLP gRPC exporter (for Jaeger/Tempo) and stdout exporter (for debugging).
- **`StartSpan()`** — Creates a new span with standard operator attributes
- **`ReconcileSpan()`** — Creates a top-level reconciliation span with correlation ID
- **`SubReconcileSpan()`** — Creates a child span for sub-reconciler operations
- **`CorrelationID()`** — Generates `namespace/name@generation` ID for correlating spans across a reconciliation pass
- **`RecordResourceEvent()`** — Adds a span event for resource operations (create/update/delete)

### `internal/testing/fakeclient/interceptor.go`
Fault injection client for testing:
- **`FaultConfig`** — Configures which operation (get/create/patch/delete/update/status), which resource kind, what fault type (transient/permanent/conflict), and how many times to inject
- **`Interceptor`** — Wraps a `client.Client` and intercepts all CRUD operations. Checks `shouldFault()` before delegating to the real client.
- **`interceptingStatusWriter`** — Wraps the status sub-resource writer to intercept status updates
- Used by chaos tests, failure injection tests, and HA tests to simulate API errors

---

## config/ — Kubernetes Manifests

### `config/crd/kustomization.yaml`
Kustomize configuration for CRD generation. References the CRD bases generated by `controller-gen`.

### `config/default/kustomization.yaml`
Default Kustomize overlay combining all components: CRDs, RBAC, manager deployment, webhooks, and namespace.

### `config/manager/manager.yaml`
Operator Deployment manifest. Runs the manager binary with configurable flags (metrics, health probes, leader election). Includes pod security context (non-root, seccomp) and resource limits.

### `config/manager/namespace.yaml`
Namespace (`platform-operator-system`) where the operator runs.

### `config/manager/kustomization.yaml`
Kustomize for manager resources (deployment + namespace).

### `config/rbac/role.yaml`
ClusterRole with least-privilege RBAC permissions for the operator. Grants CRUD on PlatformApplications, Deployments, Services, HPAs, HTTPRoutes, NetworkPolicies, PDBs, ServiceMonitors, and events.

### `config/rbac/role_binding.yaml`
ClusterRoleBinding binding the ClusterRole to the operator's ServiceAccount.

### `config/rbac/service_account.yaml`
ServiceAccount for the operator pod.

### `config/rbac/kustomization.yaml`
Kustomize for RBAC resources.

### `config/webhook/manifests.yaml`
Webhook service and MutatingWebhookConfiguration/ValidatingWebhookConfiguration manifests for admission webhooks.

### `config/webhook/certificate.yaml`
Certificate resource for webhook TLS (uses cert-manager).

### `config/webhook/kustomization.yaml`
Kustomize for webhook resources.

### `config/samples/platform_v1alpha1_platformapplication.yaml`
Sample v1alpha1 PlatformApplication resource — a simple nginx web service with 2 replicas.

### `config/samples/platform_v1beta1_platformapplication.yaml`
Sample v1beta1 PlatformApplication resource — demonstrates `envFrom` and `podAnnotations` fields.

### `config/overlays/dev/kustomization.yaml`
Dev environment overlay — 1 replica, no leader election, minimal resources.

### `config/overlays/staging/kustomization.yaml`
Staging environment overlay — 2 replicas, moderate resources.

### `config/overlays/production/kustomization.yaml`
Production environment overlay — 2 replicas, HA, topology spread constraints, monitoring enabled.

### `config/ha/kustomization.yaml`
HA Kustomize overlay — adds PDB and increases replicas.

### `config/ha/pdb.yaml`
PodDisruptionBudget for the operator itself — ensures at least 1 operator pod remains during disruptions.

### `config/monitoring/servicemonitor.yaml`
ServiceMonitor for the operator's own metrics endpoint (`:8080/metrics`).

### `config/monitoring/prometheusrule.yaml`
Prometheus alerting rules: high error rate, slow reconciliation, operator down, high drift rate.

### `config/monitoring/alerting-rules.yaml`
Additional alerting rules for SLO-based alerts.

### `config/monitoring/grafana-dashboard.json`
Grafana dashboard JSON for operator metrics: reconcile rate, error rate, duration, active reconcilers, queue depth.

### `config/monitoring/grafana-dashboard-slo.json`
Grafana dashboard for SLO monitoring: error budget burn rate, reconcile success rate, latency percentiles.

### `config/monitoring/slo.yaml`
SLO recording and alerting rules using multi-window multi-burn-rate alerting strategy (6-hour, 1-day, 3-day windows).

### `config/monitoring/kustomization.yaml`
Kustomize for monitoring resources.

### `config/policies/gatekeeper-constraints.yaml`
OPA Gatekeeper ConstraintTemplates and Constraints: non-root containers, allowed image sources, resource limits required, no privileged pods.

### `config/policies/kyverno-policies.yaml`
Kyverno ClusterPolicies: enforce non-root, allowed image sources, resource limits, no privileged pods.

### `config/policies/operator-networkpolicy.yaml`
NetworkPolicy for the operator pod — restricts ingress to metrics and webhook ports.

### `config/components/monitoring/kustomization.yaml`
Kustomize Component (kind: Component) — adds ServiceMonitor, PrometheusRule, and scrape annotations when included in an overlay.

### `config/components/security/kustomization.yaml`
Kustomize Component — adds NetworkPolicy, non-root security context, read-only filesystem, capability dropping.

### `config/components/ha/kustomization.yaml`
Kustomize Component — adds 2 replicas, PDB, anti-affinity, topology spread, startup probe for HA deployments.

---

## charts/ — Helm Chart

### `charts/platform-operator/Chart.yaml`
Helm chart metadata: name, version (0.1.0), description, type (application), keywords, maintainers.

### `charts/platform-operator/values.yaml`
Default Helm values: image repository/tag, replicas, resources, service account, RBAC, metrics, health probes, leader election, webhook configuration.

### `charts/platform-operator/templates/_helpers.tpl`
Helm template helpers: name truncation, labels (standard and selector), service account name, image reference.

### `charts/platform-operator/templates/deployment.yaml`
Helm template for the operator Deployment with configurable replicas, image, resources, probes, and security context.

### `charts/platform-operator/templates/service.yaml`
Helm template for the operator Service (metrics port).

### `charts/platform-operator/templates/serviceaccount.yaml`
Helm template for the operator ServiceAccount.

### `charts/platform-operator/templates/rbac.yaml`
Helm templates for ClusterRole, ClusterRoleBinding with configurable RBAC rules.

### `charts/platform-operator/templates/pdb.yaml`
Helm template for PodDisruptionBudget (optional, enabled by values).

### `charts/platform-operator/templates/NOTES.txt`
Post-installation notes displayed by Helm. Includes installation verification commands, usage examples, and configuration reference.

---

## gitops/ — GitOps Configurations

### `gitops/application.yaml`
Argo CD Application manifest — deploys the operator from a Git repository using Kustomize.

### `gitops/applicationset.yaml`
Argo CD ApplicationSet — multi-environment deployment across dev, staging, and production clusters using a Git directory generator.

### `gitops/application-helm.yaml`
Argo CD Application using Helm — deploys the operator using the Helm chart instead of Kustomize.

### `gitops/progressive-delivery/rollout.yaml`
Argo Rollouts Rollout resource with canary strategy: 5% → 20% → 50% → 100% traffic shifting via Gateway API, with automated analysis at each stage.

### `gitops/progressive-delivery/blue-green.yaml`
Argo Rollouts Rollout resource with blue-green strategy: auto-promotion after 120s, pre-promotion analysis, scale-down delay.

### `gitops/progressive-delivery/analysis-templates.yaml`
Two AnalysisTemplates for automated health verification during rollouts:
- `operator-health-check` — checks /healthz endpoint, reconcile success rate >95%, error rate <5%
- `operator-error-rate` — checks error rate <5% via Prometheus queries

---

## bundles/ — OLM Bundle

### `bundles/platform-operator/manifests/platform-operator.clusterserviceversion.yaml`
ClusterServiceVersion (CSV) for Operator Lifecycle Manager (OLM). Contains:
- Metadata: name, version, displayName, description, categories, maturity
- Install modes: OwnNamespace, SingleNamespace, AllNamespaces
- CRD descriptors: spec fields with UI hints for OperatorHub
- ALM examples: sample PlatformApplication YAML
- Deployment spec: container image, resources, env vars
- RBAC permissions

### `bundles/platform-operator/metadata/annotations.yaml`
OLM package metadata with channels: `stable` (v1) and `alpha` (v1alpha1).

---

## test/ — Test Suites

### `test/integration/suite_test.go`
envtest integration test setup. Starts a local API server via envtest, creates the operator manager, and registers the controller. Provides a shared test environment for all integration tests.

### `test/integration/reconciler_test.go`
Integration tests for the full reconciliation loop against a real API server. Tests: create PlatformApplication → verify child resources created, update spec → verify resources updated, delete → verify garbage collection.

### `test/e2e/e2e_test.go`
End-to-end tests against a Kind cluster. Tests: install operator, create CR, verify resources, drift correction test, cleanup.

### `test/chaos/operator_chaos_test.go`
Chaos engineering tests using the fake client interceptor:
- **`TestChaos_IntermittentFailures`** — Injects transient errors and verifies the operator recovers
- **`TestChaos_RapidCreateDelete`** — Rapid create/delete cycle to test finalizer and GC behavior
- **`TestChaos_TotalAPIFailure`** — Simulates total API server failure and verifies graceful degradation

### `test/failure/injection_test.go`
Failure injection tests with the interceptor:
- **`TestFailure_TransientErrorRetries`** — Transient errors trigger retries
- **`TestFailure_PermanentErrorSetsDegraded`** — Permanent errors set Degraded condition
- **`TestFailure_ConflictRetriesWithFreshState`** — Conflicts trigger re-queue with fresh state
- **`TestFailure_StatusConflictRetries`** — Status update conflicts are retried

### `test/ha/ha_test.go`
High availability tests:
- **`TestHA_ConcurrentReconcilers`** — Multiple concurrent reconciles don't conflict
- **`TestHA_LeaderElectionIdempotency`** — Reconcile is idempotent under leader election
- **`TestHA_GracefulShutdown`** — Operator shuts down gracefully mid-reconcile
- **`TestHA_PodDisruptionBudget`** — PDB protects the operator from disruptions

### `test/scale/scale_1000_test.go`
Scale and performance tests:
- **`TestScale_1000Resources`** — Reconciles 1000 PlatformApplications with 8 concurrent workers. Measures throughput (410 reconciles/sec), validates 0 errors, checks avg latency (2.4ms).
- **`TestScale_MemoryUsage`** — Reconciles 500 CRs sequentially and verifies memory usage stays bounded.

### `internal/controller/platformapplication_controller_test.go`
Unit tests for the main reconciler. Tests: reconcile creates child resources, handles deletion, updates status, classifies errors, requeues appropriately.

### `internal/controller/scale_test.go`
Scale test in the controller package — 100-app reconciliation with idempotency verification (second pass detects no changes).

### `internal/controller/concurrency_test.go`
Concurrency tests — verifies race-safe behavior under concurrent reconciliation with the race detector enabled.

### `internal/controller/subreconcilers/deployment_test.go`
Unit tests for Deployment sub-reconciler. Tests: correct image, replicas, ports, security context, probes, env vars, rollout strategy.

### `internal/controller/subreconcilers/subreconcilers_test.go`
Unit tests for all other sub-reconcilers (Service, HPA, HTTPRoute, NetworkPolicy, PDB, ServiceMonitor).

### `internal/controller/subreconcilers/benchmarks_test.go`
Benchmark tests for all sub-reconcilers. Measures allocations and time per operation for performance tracking.

### `internal/errors/errors_test.go`
Unit tests for error classification. Tests all error classes and the `ClassifyOrWrap` function.

### `internal/errors/benchmarks_test.go`
Benchmarks for error classification performance.

### `internal/status/conditions_test.go`
Unit tests for condition helpers. Tests SetCondition, IsReady, and condition update behavior.

### `internal/tracing/tracing_test.go`
Unit tests for tracing initialization and span creation.

---

## .github/ — CI/CD and Templates

### `.github/workflows/ci.yaml`
Main CI pipeline. Jobs: lint (golangci-lint), test (go test -race), test-chaos (chaos + failure tests), test-ha (HA tests), build (go build), govulncheck (vulnerability scan), trivy-scan (container + config scan), docker (build and push to GHCR on main).

### `.github/workflows/e2e.yaml`
End-to-end test workflow. Creates a Kind cluster, installs the operator, runs e2e tests, uploads results.

### `.github/workflows/release.yaml`
Release workflow. Triggered by Git tags. Builds multi-arch Docker images, pushes to GHCR, generates release notes, creates GitHub Release.

### `.github/workflows/canary-deploy.yaml`
Canary deployment workflow. Jobs: validate (lint/test), deploy (canary to staging), verify (health checks), promote (to production) or rollback.

### `.github/workflows/promote.yaml`
Multi-cluster promotion workflow. Jobs: validate, preflight checks, deploy to target cluster, verify health, automatic rollback on failure.

### `.github/workflows/security.yaml`
Security scanning workflow. Runs Trivy, govulncheck, Gosec, and dependency review on every PR and push.

### `.github/dependabot.yml`
Dependabot configuration for automated dependency updates. Checks Go modules and GitHub Actions weekly.

### `.github/ISSUE_TEMPLATE/bug-report.yaml`
GitHub issue template for bug reports. Fields: description, steps to reproduce, expected/actual behavior, environment.

### `.github/ISSUE_TEMPLATE/feature-request.yaml`
GitHub issue template for feature requests. Fields: problem description, proposed solution, alternatives.

### `.github/PULL_REQUEST_TEMPLATE.md`
Pull request template with sections: summary, changes, testing, checklist.

---

## docs/ — Documentation

### `docs/adr/0001-use-controller-runtime.md`
Architecture Decision Record (ADR): chose controller-runtime for the operator framework. Rationale: industry standard, active community, built-in caching, leader election, webhooks.

### `docs/adr/0003-server-side-apply.md`
ADR: chose Server-Side Apply for resource reconciliation. Rationale: conflict resolution, field ownership, drift correction, idempotency.

### `docs/adr/0004-gateway-api-over-ingress.md`
ADR: chose Gateway API over Ingress. Rationale: portable, type-safe, role-oriented, future-proof.

### `docs/adr/0006-sub-reconciler-pattern.md`
ADR: chose sub-reconciler pattern. Rationale: testability, separation of concerns, independent metrics, pure functions.

### `docs/adr/0007-v1-ga-api-stability.md`
ADR: v1 GA API stability and deprecation schedule. v1beta1 remains as storage hub; v1 is a spoke with 1:1 conversion.

### `docs/adr/0008-progressive-delivery.md`
ADR: chose Argo Rollouts for progressive delivery. Canary and blue-green strategies with automated health analysis.

### `docs/operations.md`
Operations guide: installation, scaling, backup, health checks, log analysis, common operations, upgrade procedures.

### `docs/troubleshooting.md`
Troubleshooting guide with 7 common issues: operator not starting, resources not created, reconciliation failures, high error rate, slow reconciliation, status not updating, webhook errors.

### `docs/demo.md`
9-step demo scenario: install operator, create application, verify resources, test drift correction, scale update, autoscaling, gateway routing, deletion/cleanup, monitoring.

### `docs/security/threat-model.md`
Security threat model: STRIDE analysis (Spoofing, Tampering, Repudiation, Information Disclosure, Denial of Service, Elevation of Privilege) with mitigations.

### `docs/runbooks/runbooks.md`
Index of operational runbooks with links to individual runbooks.

### `docs/runbooks/high-drift.md`
Runbook: high drift rate alert. Diagnosis steps and remediation for configuration drift.

### `docs/runbooks/high-error-rate.md`
Runbook: high error rate alert. Diagnosis steps for reconciliation errors.

### `docs/runbooks/leader-election.md`
Runbook: leader election issues. Diagnosing leader election failures and multi-replica problems.

### `docs/runbooks/operator-down.md`
Runbook: operator pod down. Recovery steps for operator unavailability.

### `docs/runbooks/permanent-errors.md`
Runbook: permanent reconciliation errors. Diagnosis and manual intervention steps.

### `docs/runbooks/slow-reconciliation.md`
Runbook: slow reconciliation. Performance diagnosis and optimization.

### `docs/runbooks/status-conflicts.md`
Runbook: status update conflicts. Diagnosing and resolving optimistic concurrency conflicts.

---

## hack/ — Development Scripts

### `hack/kind-config.yaml`
Kind cluster configuration for local development. Configures port mapping for metrics (8080), health probes (8081), and webhooks (9443).

### `hack/boilerplate.go.txt`
Boilerplate copyright header used by `controller-gen` when generating DeepCopy files.

---

## examples/ — Usage Examples

### `examples/payment-service.md`
Detailed example walkthrough of deploying a payment service using PlatformApplication. Shows the CR, generated resources, and verification steps.

---

## .qoder/ — Auto-Generated Wiki (17 files)

These files are auto-generated by the Qoder IDE's repository wiki feature. They provide structured documentation about the API reference, core concepts, deployment, development guide, examples, observability, resource management, and security configuration. They are not part of the operator's source code and should not be manually edited.

**Files:**
- `.qoder/repowiki/en/content/API Reference/API Reference.md`
- `.qoder/repowiki/en/content/API Reference/PlatformApplication Spec Schema.md`
- `.qoder/repowiki/en/content/API Reference/Status and Conditions Reference.md`
- `.qoder/repowiki/en/content/API Reference/Webhook Validation and Defaults.md`
- `.qoder/repowiki/en/content/Core Concepts.md`
- `.qoder/repowiki/en/content/Deployment and Operations.md`
- `.qoder/repowiki/en/content/Development Guide.md`
- `.qoder/repowiki/en/content/Examples and Use Cases.md`
- `.qoder/repowiki/en/content/Getting Started.md`
- `.qoder/repowiki/en/content/Observability.md`
- `.qoder/repowiki/en/content/Resource Management/*.md` (7 files)
- `.qoder/repowiki/en/content/Security Configuration.md`
- `.qoder/repowiki/en/meta/repowiki-metadata.json`
