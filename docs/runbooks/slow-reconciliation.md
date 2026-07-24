# Runbook: Slow Reconciliation

> **Alert:** PlatformOperatorSlowReconciliation | **Severity:** Warning | **For:** 10 minutes

See the full runbook: [runbooks.md](./runbooks.md#platformoperatorslowreconciliation)

## Quick Diagnosis

```bash
# Check sub-reconciler durations in logs
kubectl logs -n platform-operator-system -l control-plane=controller-manager --tail=50 | grep "duration"

# Check operator resource usage
kubectl top pod -n platform-operator-system -l control-plane=controller-manager

# Check API server request latency
kubectl get --raw /metrics | grep "apiserver_request_duration_seconds_sum" | head -5
```

## Resolution

| Cause | Fix |
|-------|-----|
| CPU throttling | Increase CPU limits |
| API server slow | Investigate etcd / API server |
| Many managed apps | Increase `--concurrent-reconciles` |
| Network latency | Check service mesh / network policies |
