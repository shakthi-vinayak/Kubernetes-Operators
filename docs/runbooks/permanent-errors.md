# Runbook: Permanent Errors

> **Alert:** PlatformOperatorPermanentErrors | **Severity:** Critical | **For:** 5 minutes

See the full runbook: [runbooks.md](./runbooks.md#platformoperatorpermanenterrors)

## Quick Diagnosis

```bash
# Check which resources have permanent errors
kubectl logs -n platform-operator-system -l control-plane=controller-manager --tail=50 | grep "permanent"

# Verify RBAC
kubectl auth can-i --list --as=system:serviceaccount:platform-operator-system:platform-operator | head -20

# Verify CRDs
kubectl get crd platformapplications.platform.example.io
kubectl get crd httproutes.gateway.networking.k8s.io
```

## Resolution

| Error | Fix |
|-------|-----|
| `Forbidden` | `kubectl apply -k config/rbac/` |
| `NotFound` (CRD) | `kubectl apply -k config/crd/` then install Gateway API CRDs |
| `Invalid` (spec) | Fix the PlatformApplication spec — check `ConfigurationValid` condition |
| `BadRequest` | Check operator container args |
