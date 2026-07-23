# Example: payment-service

This example demonstrates deploying a `PlatformApplication` for a payment service.

## Apply the PlatformApplication

```bash
kubectl apply -f ../config/samples/platform_v1alpha1_platformapplication.yaml
```

## Verify Generated Resources

```bash
# Check PlatformApplication status
kubectl get platformapplication payment-service -o yaml

# Check generated Deployment
kubectl get deployment payment-service -o yaml

# Check generated Service
kubectl get service payment-service -o yaml

# Check generated HPA
kubectl get hpa payment-service -o yaml

# Check generated NetworkPolicy
kubectl get networkpolicy payment-service-netpol -o yaml

# Check generated PDB
kubectl get pdb payment-service-pdb -o yaml
```

## Test Drift Correction

Manually change the replica count:

```bash
kubectl scale deployment payment-service --replicas=1
```

Wait a few seconds and verify the operator restores the desired state:

```bash
kubectl get deployment payment-service -o jsonpath='{.spec.replicas}'
```

## Update the Application

Change the image tag:

```bash
kubectl patch platformapplication payment-service --type='merge' \
  -p '{"spec":{"image":{"tag":"2.5.0"}}}'
```

Watch the rollout:

```bash
kubectl rollout status deployment/payment-service
```

## Delete and Verify Cleanup

```bash
kubectl delete platformapplication payment-service

# Verify all child resources are garbage-collected
kubectl get deployment,service,hpa,networkpolicy,pdb -l app.kubernetes.io/name=payment-service
```
