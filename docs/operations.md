# Operations Guide

## Prerequisites

- Kubernetes 1.28+ cluster
- kubectl configured with cluster access
- Prometheus + Grafana (recommended)
- cert-manager (for webhook TLS)

## Installation

### Via Helm (Recommended)

```bash
helm install platform-operator charts/platform-operator \
  --namespace platform-operator-system --create-namespace \
  --set replicaCount=2 \
  --set operator.leaderElect=true \
  --set podDisruptionBudget.enabled=true
```

### Via Kustomize

```bash
# Development
kubectl apply -k config/overlays/dev

# Staging
kubectl apply -k config/overlays/staging

# Production
kubectl apply -k config/overlays/production
```

## Day-to-Day Operations

### Creating an Application

```yaml
apiVersion: platform.example.io/v1beta1
kind: PlatformApplication
metadata:
  name: my-service
  namespace: default
spec:
  image:
    repository: my-registry.io/my-service
    tag: "1.2.3"
  replicas:
    min: 2
    max: 10
  service:
    port: 8080
    type: ClusterIP
  autoscaling:
    enabled: true
    targetCPUUtilization: 70
  healthChecks:
    livenessPath: /healthz
    readinessPath: /readyz
```

```bash
kubectl apply -f my-service.yaml
kubectl get papp my-service
```

### Updating an Application

```bash
# Update image tag
kubectl patch papp my-service --type='merge' \
  -p '{"spec":{"image":{"tag":"1.3.0"}}}'

# Scale up
kubectl patch papp my-service --type='merge' \
  -p '{"spec":{"replicas":{"min":3}}}'
```

### Monitoring Status

```bash
# Check all platform applications
kubectl get papp -A

# Check specific application status
kubectl get papp my-service -o jsonpath='{.status.conditions}'

# Watch for changes
kubectl get papp -w
```

## Scaling the Operator

### Horizontal Scaling

The operator supports multiple replicas with leader election:

```yaml
# values.yaml
replicaCount: 3
operator:
  leaderElect: true
  concurrentReconciles: 4
```

### Worker Concurrency

Increase concurrent reconciles for high CR counts:

```bash
# Helm
--set operator.concurrentReconciles=8

# Kustomize overlay
containers:
  - name: manager
    args:
      - --concurrent-reconciles=8
```

**Recommended concurrency by scale:**
- 0-100 CRs: 1-2 workers
- 100-500 CRs: 3-4 workers
- 500-1000 CRs: 4-8 workers
- 1000+ CRs: 8-16 workers

## Backup and Recovery

### Backup CRDs and Custom Resources

```bash
# Backup CRDs
kubectl get crd platformapplications.platform.example.io -o yaml > crd-backup.yaml

# Backup all custom resources
kubectl get papp -A -o yaml > papp-backup.yaml
```

### Restore

```bash
# Restore CRD first
kubectl apply -f crd-backup.yaml

# Restore resources (operator will reconcile)
kubectl apply -f papp-backup.yaml
```

## Health Checks

### Operator Health

```bash
# Check operator pods
kubectl get pods -n platform-operator-system -l control-plane=controller-manager

# Check liveness
kubectl exec -n platform-operator-system deploy/platform-operator-controller-manager \
  -- wget -qO- http://localhost:8081/healthz

# Check readiness
kubectl exec -n platform-operator-system deploy/platform-operator-controller-manager \
  -- wget -qO- http://localhost:8081/readyz
```

### Metrics

```bash
# Port-forward metrics
kubectl port-forward -n platform-operator-system \
  svc/platform-operator-controller-manager-metrics 8080:8080

# Query metrics
curl http://localhost:8080/metrics | grep platform_operator
```

Key metrics:
- `platform_operator_reconcile_total` - Total reconcile count by result
- `platform_operator_reconcile_duration_seconds` - Reconcile latency histogram
- `platform_operator_active_reconcilers` - Currently active workers
- `platform_operator_sub_reconcile_total` - Per-resource reconcile counts
