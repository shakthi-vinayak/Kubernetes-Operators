# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it responsibly:

1. **Do not** open a public GitHub issue.
2. Email the details to security@example.io (replace with your actual contact).
3. Include a description of the vulnerability, steps to reproduce, and potential impact.
4. We will acknowledge receipt within 48 hours and provide a timeline for resolution.

## Security Best Practices

### Operator Deployment

- The operator runs as non-root (UID 65532) with a read-only root filesystem.
- All Linux capabilities are dropped; no privilege escalation is allowed.
- The seccomp profile is set to RuntimeDefault.
- Resource requests and limits are configured to prevent resource exhaustion.

### RBAC

- The operator uses least-privilege RBAC scoped to specific API groups and resources.
- No cluster-admin permissions are required.
- The operator cannot access secrets, configmaps outside its managed resources, or other namespaces' resources.

### Network Policy

- When enabled, the operator generates NetworkPolicy resources that restrict ingress traffic to the application port.
- Egress is allowed for DNS resolution and external service communication.

### Supply Chain

- Container images are built using multi-stage builds with distroless base images.
- Dependencies are tracked via `go.sum` and scanned for vulnerabilities.

## Threat Model

### What the operator CAN access:
- PlatformApplication CRs (read/write)
- Managed child resources: Deployments, Services, HPAs, HTTPRoutes, NetworkPolicies, PDBs, ServiceMonitors (read/write)
- Leases for leader election (read/write)

### What the operator CANNOT access:
- Secrets (no RBAC permissions)
- ConfigMaps not managed by the operator
- Resources in other namespaces (unless explicitly scoped)
- Cluster-level resources (nodes, persistent volumes, etc.)
