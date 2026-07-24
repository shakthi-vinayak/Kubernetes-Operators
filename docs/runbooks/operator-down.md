# Runbook: Platform Operator Down

> **Alert:** PlatformOperatorDown | **Severity:** Critical | **For:** 5 minutes

See the full runbook: [runbooks.md](./runbooks.md#platformoperatordown)

## Quick Recovery

```bash
# 1. Check pod status
kubectl get pods -n platform-operator-system -l control-plane=controller-manager

# 2. Check logs for crash reason
kubectl logs -n platform-operator-system -l control-plane=controller-manager --previous --tail=50

# 3. If CrashLoopBackOff, check events
kubectl describe pod -n platform-operator-system -l control-plane=controller-manager

# 4. Redeploy if necessary
kubectl rollout restart deployment -n platform-operator-system platform-operator-controller-manager
```

## Common Causes

| Cause | Fix |
|-------|-----|
| OOMKilled | Increase memory limits in Deployment |
| RBAC denied | `kubectl apply -k config/rbac/` |
| Leader election timeout | Check network between replicas |
| Invalid flag | Verify container args in Deployment spec |
