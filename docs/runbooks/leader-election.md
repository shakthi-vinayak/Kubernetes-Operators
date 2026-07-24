# Runbook: Leader Election Flapping

> **Alert:** PlatformOperatorLeaderElectionFlapping | **Severity:** Warning | **For:** 2 minutes

See the full runbook: [runbooks.md](./runbooks.md#platformoperatorleaderelectionflapping)

## Quick Diagnosis

```bash
# Check lease transitions
kubectl get lease -n platform-operator-system platform-operator.example.io -o jsonpath='{.spec.holderIdentity}{"\n"}{.spec.acquireTime}{"\n"}{.spec.renewTime}'

# Check pod health across replicas
kubectl get pods -n platform-operator-system -l control-plane=controller-manager -o wide

# Check node health
kubectl get nodes -o wide | grep -v " Ready"
kubectl describe node <node-with-operator-pod> | grep -A5 "Conditions"
```

## Resolution

| Cause | Fix |
|-------|-----|
| Node instability (eviction, OOM) | Move operator pods to stable nodes |
| Network partition between nodes | Check inter-node connectivity |
| Lease duration too short | Increase `--leader-election-lease-duration` (default: 15s) |
| Clock skew between nodes | Sync NTP across cluster nodes |

## Prevention

- Use `topologySpreadConstraints` to spread replicas across nodes/zones
- Set `--leader-election-lease-duration=30s` and `--leader-election-renew-deadline=20s` for stability
- Monitor node health separately with node-exporter alerts
