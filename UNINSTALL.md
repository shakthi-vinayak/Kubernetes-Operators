# Uninstall Guide

This guide covers how to completely remove the Platform Operator from your cluster, regardless of the installation method used.

---

## Table of Contents

- [Step 0: Delete Application CRs First](#step-0-delete-application-crs-first)
- [Method 1: Makefile / Kustomize Uninstall](#method-1-makefile--kustomize-uninstall)
- [Method 2: Helm Uninstall](#method-2-helm-uninstall)
- [Method 3: Argo CD (GitOps) Uninstall](#method-3-argo-cd-gitops-uninstall)
- [Troubleshooting](#troubleshooting)
- [Optional: Remove Shared Prerequisites](#optional-remove-shared-prerequisites)

---

## Step 0: Delete Application CRs First

> **IMPORTANT:** Always delete all `PlatformApplication` custom resources **before** removing the operator. The CRs carry finalizers — if the operator is removed first, CR deletion will hang indefinitely.

```bash
# Delete all PlatformApplications across all namespaces
kubectl delete platformapplications --all --all-namespaces

# Verify none remain
kubectl get platformapplications -A
```

Deleting a `PlatformApplication` automatically garbage-collects all its child resources (Deployment, Service, HPA, HTTPRoute, NetworkPolicy, PDB, ServiceMonitor) via owner references — no manual cleanup needed.

---

## Method 1: Makefile / Kustomize Uninstall

Use this if you installed with `make install` / `make deploy` or `kubectl apply -k`.

```bash
# 1. Remove the operator (controller Deployment, RBAC, webhooks, namespace)
make undeploy

# 2. Remove the CRDs (this is always the LAST step)
make uninstall
```

If you deployed via a Kustomize overlay directly:

```bash
# Remove the environment overlay you applied
kubectl delete -k config/overlays/dev/          # or staging/ production/

# Remove the CRDs
kubectl delete -k config/crd/
```

---

## Method 2: Helm Uninstall

Use this if you installed with `helm install`.

```bash
# 1. Uninstall the Helm release (removes controller, RBAC, webhooks)
helm uninstall platform-operator -n platform-operator-system

# 2. Helm does NOT remove CRDs placed in the chart's crds/ directory.
#    Delete them manually:
kubectl delete crd platformapplications.platform.example.io

# 3. Remove the namespace if it is now empty
kubectl delete namespace platform-operator-system
```

---

## Method 3: Argo CD (GitOps) Uninstall

Use this if you installed via the manifests in `gitops/`.

```bash
# Delete the Argo CD Application with cascading prune
argocd app delete platform-operator --cascade

# Or via kubectl:
kubectl delete application platform-operator -n argocd
```

If you used an `ApplicationSet` for multi-environment rollout, delete the ApplicationSet instead — it will cascade to all generated Applications:

```bash
kubectl delete applicationset platform-operator -n argocd
```

Then remove the CRDs manually if they were managed outside the Application:

```bash
kubectl delete crd platformapplications.platform.example.io
```

---

## Troubleshooting

### CR stuck in `Terminating`

This happens when the operator was removed **before** the CRs were deleted — the finalizer can no longer be processed. Force-remove the finalizer:

```bash
kubectl patch platformapplication <name> -n <namespace> \
  --type json -p '[{"op":"remove","path":"/metadata/finalizers"}]'
```

### Namespace stuck in `Terminating`

Usually caused by leftover resources with finalizers inside the namespace. Find them:

```bash
kubectl api-resources --verbs=list --namespaced -o name | \
  xargs -n 1 kubectl get -n platform-operator-system --ignore-not-found --show-kind
```

Then patch away finalizers on whatever remains.

### Webhook blocks resource operations after uninstall

If the operator was removed but its webhook configurations remain, API operations may fail with `connection refused` errors. Clean them up:

```bash
kubectl delete validatingwebhookconfiguration -l app.kubernetes.io/name=platform-operator
kubectl delete mutatingwebhookconfiguration -l app.kubernetes.io/name=platform-operator
```

---

## Optional: Remove Shared Prerequisites

Only remove these if **nothing else** in your cluster uses them:

```bash
# Gateway API CRDs (shared cluster infrastructure)
kubectl delete -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml

# Prometheus Operator / kube-prometheus-stack (if installed just for this operator)
helm uninstall kube-prometheus-stack -n monitoring
```

---

## Verification Checklist

After uninstalling, confirm everything is gone:

```bash
# No CRs or CRDs
kubectl get crds | grep platform.example.io          # expect: no output

# No operator workloads
kubectl get all -n platform-operator-system          # expect: No resources / namespace not found

# No leftover webhooks
kubectl get validatingwebhookconfigurations | grep platform   # expect: no output
kubectl get mutatingwebhookconfigurations | grep platform     # expect: no output

# No leftover cluster-scoped RBAC
kubectl get clusterroles,clusterrolebindings | grep platform-operator   # expect: no output
```
