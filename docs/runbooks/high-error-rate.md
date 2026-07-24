# Runbook: High Error Rate

> **Alert:** PlatformOperatorHighErrorRate (>25%) / PlatformOperatorElevatedErrorRate (>10%)
> **Severity:** Critical / Warning | **For:** 10m / 15m

See the full runbook: [runbooks.md](./runbooks.md#platformoperatorhigherrorrate--elevatederrorrate)

## Quick Diagnosis

```bash
# 1. Check error classification breakdown in logs
kubectl logs -n platform-operator-system -l control-plane=controller-manager --tail=100 | grep "error_class"

# 2. Find degraded PlatformApplications
kubectl get platformapplications -A -o json | jq -r '.items[] | select(.status.conditions[] | select(.type=="Degraded" and .status=="True")) | "\(.metadata.namespace)/\(.metadata.name): \(.status.conditions[] | select(.type=="Degraded") | .message)"'

# 3. Check API server health
kubectl get --raw /healthz
```

## Error Class Actions

| Class | Meaning | Action |
|-------|---------|--------|
| `transient` | Temporary (API server down, timeout) | Wait — self-heals. If persistent >30min, escalate |
| `permanent` | Won't self-heal (RBAC, invalid spec) | Fix the root cause immediately |
| `conflict` | Resource version mismatch | Usually self-heals. If persistent, check for duplicate controllers |
| `unknown` | Unclassified | Check operator logs for stack traces |
