# Runbook: Status Conflicts

> **Alert:** PlatformOperatorStatusConflicts / PlatformOperatorHighConflictRate
> **Severity:** Warning | **For:** 5m / 10m

See the full runbook: [runbooks.md](./runbooks.md#platformoperatorstatusconflicts--highconflictrate)

## Quick Diagnosis

```bash
# Check for duplicate operator deployments
kubectl get deployments -A | grep platform-operator
kubectl get pods -A -l control-plane=controller-manager

# Check lease holder
kubectl get lease -n platform-operator-system platform-operator.example.io -o yaml

# Inspect managed fields on a resource
kubectl get deployment <name> -n <ns> -o jsonpath='{.metadata.managedFields}' | python3 -m json.tool
```

## Resolution

| Cause | Fix |
|-------|-----|
| Duplicate operator (no leader election) | Enable `--leader-elect`, delete duplicates |
| Another controller touching same fields | Coordinate field ownership via SSA |
| Watch event loop | Check for operator writing → watch → operator writing cycle |
