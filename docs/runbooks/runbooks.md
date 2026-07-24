# Platform Operator Runbooks

Operational runbooks for all Platform Operator alerts. Each runbook follows the format: **Impact → Diagnosis → Resolution → Prevention**.

---

## PlatformOperatorDown

**Severity:** Critical | **For:** 5m

### Impact
The operator is not running. No PlatformApplication resources are being reconciled. Existing applications continue running but new changes, deployments, and drift corrections are suspended.

### Diagnosis

```bash
# Check if the operator pod exists
kubectl get pods -n platform-operator-system -l control-plane=controller-manager

# Check pod status and events
kubectl describe pod -n platform-operator-system -l control-plane=controller-manager

# Check operator logs
kubectl logs -n platform-operator-system -l control-plane=controller-manager --tail=100

# Check if metrics endpoint is reachable
kubectl port-forward -n platform-operator-system svc/platform-operator-metrics 8080:8080
curl http://localhost:8080/metrics | head -5
```

### Resolution

| Symptom | Action |
|---------|--------|
| Pod in CrashLoopBackOff | Check logs for panic/fatal errors. Common causes: RBAC misconfiguration, invalid flags, leader election timeout |
| Pod Pending | Check node resources: `kubectl describe nodes`. Look for scheduling constraints |
| Pod Running but no metrics | Check Service port name matches ServiceMonitor (`metrics`). Verify network policies allow scraping |
| No pod exists | Redeploy: `kubectl apply -k config/default/` |

### Prevention
- Ensure resource requests/limits are set on the operator Deployment
- Configure pod anti-affinity for multi-replica deployments
- Set up a `PodMonitor` fallback in addition to `ServiceMonitor`

---

## PlatformOperatorHighErrorRate / ElevatedErrorRate

**Severity:** Critical (>25%) / Warning (>10%) | **For:** 10m / 15m

### Impact
A significant portion of reconciliation attempts are failing. Managed applications may be in a degraded state — Deployments may not be updating, Services may be stale.

### Diagnosis

```bash
# Check error classification breakdown
kubectl logs -n platform-operator-system -l control-plane=controller-manager | grep "error" | tail -50

# Check which resources are failing
kubectl get platformapplications -A -o json | jq '.items[] | select(.status.conditions[] | select(.type=="Degraded" and .status=="True")) | .metadata.name'

# Check API server health
kubectl get --raw /healthz
kubectl get componentstatuses
```

### Resolution

| Error Class | Likely Cause | Action |
|-------------|-------------|--------|
| `transient` | API server overload, network issues | Wait — these self-heal. If persistent, check API server health |
| `permanent` | Invalid CR spec, missing CRD, RBAC denied | Fix the PlatformApplication spec or cluster RBAC |
| `conflict` | Another controller modifying same resources | Identify the conflicting controller via managed fields |
| `unknown` | Unexpected error type | Check operator logs for stack traces |

```bash
# Check for permanent errors on specific resources
kubectl get platformapplication <name> -n <ns> -o jsonpath='{.status.conditions}'

# Check managed fields to identify conflicting controllers
kubectl get deployment <name> -n <ns> -o jsonpath='{.metadata.managedFields}'
```

### Prevention
- Validate PlatformApplication specs via admission webhooks (enabled by default)
- Monitor API server health separately
- Ensure RBAC is correctly configured for the operator's ServiceAccount

---

## PlatformOperatorPermanentErrors

**Severity:** Critical | **For:** 5m

### Impact
The operator has encountered errors that will not self-heal. Affected PlatformApplications are stuck in their current state. Manual intervention is required.

### Diagnosis

```bash
# Identify which resources have permanent errors
kubectl logs -n platform-operator-system -l control-plane=controller-manager | grep "permanent" | tail -20

# Check for RBAC issues
kubectl auth can-i --list --as=system:serviceaccount:platform-operator-system:platform-operator

# Check if CRDs are installed
kubectl get crd | grep platform.example.io
kubectl get crd | grep gateway.networking.k8s.io
```

### Resolution

| Cause | Action |
|-------|--------|
| `Forbidden` / RBAC denied | Apply RBAC: `kubectl apply -k config/rbac/` |
| `NotFound` (CRD missing) | Install CRDs: `kubectl apply -k config/crd/` |
| `Invalid` (spec error) | Fix the PlatformApplication spec — check webhook validation messages |
| `BadRequest` | Check operator flags and configuration |

### Prevention
- Run `make install` before deploying the operator
- Use the validating webhook to catch spec errors before they reach the reconciler
- Test RBAC with `kubectl auth can-i` in CI

---

## PlatformOperatorSlowReconciliation

**Severity:** Warning | **For:** 10m

### Impact
Reconciliation is slow. Changes to PlatformApplication resources take longer to propagate. Rolling updates may be delayed.

### Diagnosis

```bash
# Check which sub-reconcilers are slow
kubectl logs -n platform-operator-system -l control-plane=controller-manager | grep "duration" | tail -20

# Check API server latency
kubectl get --raw /metrics | grep apiserver_request_duration_seconds

# Check operator resource usage
kubectl top pod -n platform-operator-system -l control-plane=controller-manager
```

### Resolution

| Cause | Action |
|-------|--------|
| API server latency | Investigate API server performance. Check etcd health |
| High resource count | Increase `--concurrent-reconciles` flag |
| Operator CPU throttling | Increase CPU limits on the operator Deployment |
| Network latency | Check network policies and service mesh configuration |

### Prevention
- Set appropriate resource limits on the operator
- Monitor sub-reconciler duration separately (p95 per resource type)
- Scale horizontally with `--concurrent-reconciles` for large clusters

---

## PlatformOperatorStatusConflicts / HighConflictRate

**Severity:** Warning | **For:** 5m / 10m

### Impact
Status updates are being rejected due to resource version conflicts. The operator's view of the resource state is stale. This usually resolves automatically but may delay status updates.

### Diagnosis

```bash
# Check if multiple controllers are managing PlatformApplications
kubectl get pods -A -l control-plane=controller-manager

# Check for duplicate operator deployments
kubectl get deployments -A | grep platform-operator

# Inspect managed fields on a specific resource
kubectl get deployment <name> -n <ns> -o jsonpath='{.metadata.managedFields}' | jq
```

### Resolution

| Cause | Action |
|-------|--------|
| Duplicate operator replicas without leader election | Enable `--leader-elect` flag |
| Another controller modifying the same fields | Identify via managed fields, coordinate field ownership |
| High-frequency watch events | Check for resource update loops (operator writing → watch event → operator writing) |

### Prevention
- Always enable leader election in HA deployments
- Ensure only one instance of the operator manages PlatformApplication resources
- Use SSA field ownership to avoid conflicts with other controllers

---

## PlatformOperatorHighDriftRate

**Severity:** Warning | **For:** 15m

### Impact
The operator is frequently correcting drift — meaning managed resources are being manually modified and then reverted. This indicates process issues, not operator issues.

### Diagnosis

```bash
# Check which resources are being frequently updated
kubectl logs -n platform-operator-system -l control-plane=controller-manager | grep "resource updated" | tail -30

# Check recent events on a specific deployment
kubectl get events -n <ns> --field-selector involvedObject.name=<name> --sort-by='.lastTimestamp'

# Check who modified a resource via audit logs (if available)
# Look for non-operator user agents in audit log entries
```

### Resolution

| Cause | Action |
|-------|--------|
| Manual `kubectl edit` on managed resources | Educate teams: modify the PlatformApplication CR, not child resources |
| Another automation modifying resources | Coordinate with the other automation's owner |
| HPA adjusting replicas (expected drift) | Ensure the operator doesn't manage `replicas` when HPA is enabled |

### Prevention
- Document that managed resources should not be modified directly
- Use RBAC to restrict write access to operator-managed resources
- Enable audit logging to track who/what modifies resources

---

## PlatformOperatorLeaderElectionFlapping

**Severity:** Warning | **For:** 2m

### Impact
Leader election is changing frequently. During leader transitions, there is a brief period (seconds) where no reconciliation occurs. Frequent flapping increases reconcile latency.

### Diagnosis

```bash
# Check lease holder
kubectl get lease -n platform-operator-system platform-operator.example.io -o yaml

# Check pod health across replicas
kubectl get pods -n platform-operator-system -l control-plane=controller-manager -o wide

# Check for node issues
kubectl get nodes -o wide | grep -v Ready
kubectl describe node <node-with-operator-pod>
```

### Resolution

| Cause | Action |
|-------|--------|
| Node instability (eviction, OOM) | Check node resources. Move operator pods to more stable nodes |
| Network partition | Check network connectivity between nodes running operator replicas |
| Leader lease too short | Increase `--leader-election-lease-duration` and `--leader-election-renew-deadline` |

### Prevention
- Use pod anti-affinity to spread operator replicas across nodes
- Set appropriate leader election timeouts (default: 15s lease, 10s renew, 2s retry)
- Monitor node health separately

---

## General Troubleshooting

### Quick Health Check

```bash
# 1. Operator pod status
kubectl get pods -n platform-operator-system -l control-plane=controller-manager

# 2. Recent events
kubectl get events -n platform-operator-system --sort-by='.lastTimestamp' | tail -10

# 3. Operator logs (errors only)
kubectl logs -n platform-operator-system -l control-plane=controller-manager | grep -i error | tail -20

# 4. PlatformApplication status overview
kubectl get platformapplications -A -o custom-columns='NAME:.metadata.name,NAMESPACE:.metadata.namespace,READY:.status.conditions[?(@.type=="Ready")].status,REASON:.status.conditions[?(@.type=="Ready")].reason'
```

### Escalation Path

1. **On-call SRE** — for PlatformOperatorDown, HighErrorRate (critical)
2. **Platform team** — for PermanentErrors, drift issues
3. **Infrastructure team** — for API server issues, node problems, leader election
