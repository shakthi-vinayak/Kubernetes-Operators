# Troubleshooting Guide

## Common Issues

### Operator Pod CrashLoopBackOff

**Symptoms:** Operator pod repeatedly crashes.

**Diagnosis:**
```bash
kubectl logs -n platform-operator-system \
  deploy/platform-operator-controller-manager --previous
```

**Common Causes:**
1. **Missing CRD**: Install CRDs first: `make install`
2. **RBAC errors**: Check ClusterRole bindings
3. **OOMKilled**: Increase memory limits in values.yaml
4. **Liveness probe failure**: Check `/healthz` endpoint

**Resolution:**
```bash
# Check events
kubectl describe pod -n platform-operator-system \
  -l control-plane=controller-manager

# Check resource usage
kubectl top pod -n platform-operator-system
```

---

### Application Stuck in "Progressing" State

**Symptoms:** PlatformApplication shows `Progressing=True` indefinitely.

**Diagnosis:**
```bash
kubectl get papp my-app -o jsonpath='{.status.conditions}' | jq

# Check if Deployment exists
kubectl get deployment my-app -o yaml

# Check Deployment events
kubectl describe deployment my-app
```

**Common Causes:**
1. **ImagePullBackOff**: Image doesn't exist or pull secret missing
2. **CrashLoopBackOff**: Application container crashing
3. **Resource limits**: Insufficient cluster resources

**Resolution:**
```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=my-app

# Check events
kubectl get events --field-selector involvedObject.name=my-app-xxx
```

---

### High Reconcile Error Rate

**Symptoms:** Alert fires for `PlatformOperatorReconcileErrors` > 5%.

**Diagnosis:**
```bash
# Check error metrics
kubectl port-forward -n platform-operator-system \
  svc/platform-operator-controller-manager-metrics 8080:8080

curl -s http://localhost:8080/metrics | \
  grep 'platform_operator_reconcile_total{result="error"}'
```

**Common Causes:**
1. **API server pressure**: Rate limiting or throttling
2. **Webhook timeout**: Admission webhook not responding
3. **Conflict errors**: Multiple controllers fighting

**Resolution:**
```bash
# Check operator logs for error details
kubectl logs -n platform-operator-system \
  deploy/platform-operator-controller-manager | grep -i error | tail -20

# Check API server health
kubectl get --raw /healthz
```

---

### Leader Election Issues

**Symptoms:** Multiple operators running simultaneously or no operator active.

**Diagnosis:**
```bash
# Check leader lease
kubectl get lease -n platform-operator-system \
  platform-operator.example.io -o yaml

# Check all operator pods
kubectl get pods -n platform-operator-system \
  -l control-plane=controller-manager -o wide
```

**Resolution:**
```bash
# Force lease release (emergency only)
kubectl delete lease -n platform-operator-system \
  platform-operator.example.io

# Restart operator
kubectl rollout restart deploy/platform-operator-controller-manager \
  -n platform-operator-system
```

---

### Webhook Certificate Errors

**Symptoms:** `x509: certificate signed by unknown authority`

**Diagnosis:**
```bash
# Check cert-manager status
kubectl get certificates -n platform-operator-system
kubectl get certificaterequests -n platform-operator-system

# Check webhook configuration
kubectl get validatingwebhookconfiguration \
  platform-operator-validating-webhook -o yaml
```

**Resolution:**
```bash
# Restart cert-manager
kubectl rollout restart deploy/cert-manager -n cert-manager

# Re-trigger certificate issuance
kubectl delete secret -n platform-operator-system \
  platform-operator-webhook-cert
```

---

### Slow Reconciliation

**Symptoms:** Changes take minutes to apply.

**Diagnosis:**
```bash
# Check reconcile duration metrics
curl -s http://localhost:8080/metrics | \
  grep platform_operator_reconcile_duration_seconds

# Check work queue depth
curl -s http://localhost:8080/metrics | \
  grep platform_operator_work_queue_depth
```

**Resolution:**
1. Increase `--concurrent-reconciles`
2. Check for API server throttling
3. Reduce watched resources if possible

---

### Drift Not Being Corrected

**Symptoms:** Manual changes to child resources persist.

**Diagnosis:**
```bash
# Check if SSA is working
kubectl get deployment my-app -o jsonpath='{.metadata.managedFields}'

# Verify owner references
kubectl get deployment my-app -o jsonpath='{.metadata.ownerReferences}'
```

**Resolution:**
1. Verify the PlatformApplication still exists
2. Check operator logs for SSA conflicts
3. Ensure no other controllers are managing the same resources

---

## Debug Commands

```bash
# Full operator status
kubectl get deploy,pdb,svc,serviceaccount -n platform-operator-system

# All PlatformApplications with status
kubectl get papp -A -o custom-columns=\
NAME:.metadata.name,\
NAMESPACE:.metadata.namespace,\
READY:.status.conditions[?(@.type=='Ready')].status,\
REPLICAS:.status.readyReplicas,\
VERSION:.status.deployedVersion

# Operator resource usage
kubectl top pod -n platform-operator-system

# Recent operator events
kubectl get events -n platform-operator-system --sort-by=.lastTimestamp | tail -20
```

## Escalation Path

1. Check this troubleshooting guide
2. Review operator logs: `kubectl logs -n platform-operator-system deploy/...`
3. Check metrics in Grafana dashboard
4. Review runbooks in `docs/runbooks/`
5. Open an issue with logs and metrics
