# Runbook: High Drift Rate

> **Alert:** PlatformOperatorHighDriftRate | **Severity:** Warning | **For:** 15 minutes

See the full runbook: [runbooks.md](./runbooks.md#platformoperatorhighdriftrate)

## Quick Diagnosis

```bash
# Check which resources are being frequently corrected
kubectl logs -n platform-operator-system -l control-plane=controller-manager --tail=100 | grep "resource updated"

# Check recent events on managed resources
kubectl get events -A --field-selector reason=Updated --sort-by='.lastTimestamp' | tail -20
```

## Resolution

| Cause | Fix |
|-------|-----|
| Manual `kubectl edit` on child resources | Educate teams — modify the PlatformApplication CR instead |
| Another automation modifying resources | Coordinate with the automation owner |
| HPA adjusting replicas | Ensure operator doesn't fight HPA on the `replicas` field |

## Note

Drift correction is **expected behavior** — the operator's job is to enforce desired state. This alert fires when the correction rate is unusually high, indicating a process issue.
