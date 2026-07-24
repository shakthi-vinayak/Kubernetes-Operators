# Demo Scenario: Deploy and Operate a Web Application

This demo walks through deploying the Platform Operator and using it to manage
a sample web application lifecycle.

## Prerequisites

- Kind cluster (or any Kubernetes 1.28+ cluster)
- kubectl configured
- Helm 3 installed

## Step 1: Set Up the Environment

```bash
# Create a local Kind cluster
make kind-create

# Install cert-manager (for webhook TLS)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.0/cert-manager.yaml
kubectl wait --for=condition=available deploy/cert-manager-webhook -n cert-manager --timeout=120s
```

## Step 2: Install the Platform Operator

```bash
# Build and load the operator image
make docker-build kind-load

# Deploy using Helm
helm install platform-operator charts/platform-operator \
  --namespace platform-operator-system --create-namespace \
  --set replicaCount=1 \
  --set operator.leaderElect=true
```

Wait for the operator to be ready:

```bash
kubectl wait --for=condition=available deploy/platform-operator \
  -n platform-operator-system --timeout=120s
```

## Step 3: Deploy a Sample Application

Create a `demo-app.yaml`:

```yaml
apiVersion: platform.example.io/v1beta1
kind: PlatformApplication
metadata:
  name: demo-webapp
  namespace: default
spec:
  image:
    repository: nginx
    tag: "1.25"
    pullPolicy: IfNotPresent
  replicas:
    min: 2
    max: 5
  service:
    port: 80
    type: ClusterIP
  autoscaling:
    enabled: true
    targetCPUUtilization: 70
  healthChecks:
    livenessPath: /
    readinessPath: /
  rollout:
    strategy: RollingUpdate
```

```bash
kubectl apply -f demo-app.yaml
```

## Step 4: Verify Generated Resources

```bash
# Check PlatformApplication status
kubectl get papp demo-webapp -o wide

# Check all generated resources
kubectl get deployment,service,hpa,networkpolicy,pdb demo-webapp

# Verify Deployment is running
kubectl rollout status deployment/demo-webapp --timeout=120s

# Check pod status
kubectl get pods -l app.kubernetes.io/name=demo-webapp
```

## Step 5: Test Drift Correction

Manually modify the Deployment:

```bash
# Scale down the deployment
kubectl scale deployment demo-webapp --replicas=1

# Watch the operator restore the desired state
kubectl get deployment demo-webapp --watch
```

Within a few seconds, the operator restores the replica count.

## Step 6: Update the Application

```bash
# Update the image tag
kubectl patch papp demo-webapp --type='merge' \
  -p '{"spec":{"image":{"tag":"1.26"}}}'

# Watch the rollout
kubectl rollout status deployment/demo-webapp
```

## Step 7: Scale Up

```bash
# Increase minimum replicas
kubectl patch papp demo-webapp --type='merge' \
  -p '{"spec":{"replicas":{"min":3}}}'

# Verify new replica count
kubectl get deployment demo-webapp
```

## Step 8: Monitor the Operator

```bash
# Check operator logs
kubectl logs -n platform-operator-system \
  deploy/platform-operator -f

# Port-forward metrics
kubectl port-forward -n platform-operator-system \
  svc/platform-operator-metrics 8080:8080 &

# Query metrics
curl -s http://localhost:8080/metrics | grep platform_operator
```

## Step 9: Clean Up

```bash
# Delete the application (operator handles cleanup via owner references)
kubectl delete papp demo-webapp

# Verify all child resources are cleaned up
kubectl get deployment,service,hpa,networkpolicy,pdb -l app.kubernetes.io/name=demo-webapp

# Uninstall the operator
helm uninstall platform-operator -n platform-operator-system

# Delete the cluster
make kind-delete
```

## What You Learned

1. **Declarative management**: Define your app once, operator manages 7 child resources
2. **Drift correction**: Manual changes are automatically reverted
3. **Rolling updates**: Image changes trigger zero-downtime rollouts
4. **Scaling**: Replica changes propagate to Deployments and HPAs
5. **Observability**: Metrics show reconciliation performance
6. **Cleanup**: Owner references ensure garbage collection
