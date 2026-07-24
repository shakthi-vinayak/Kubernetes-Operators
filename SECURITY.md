# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes       |
| < 0.1.0 | No        |

## Reporting a Vulnerability

We take security vulnerabilities seriously. Please report them responsibly.

### Private Reporting (Preferred)

Use [GitHub Security Advisories](https://github.com/shakthi-vinayak/Kubernetes-Operators/security/advisories/new)
to report vulnerabilities privately.

### Email

Send details to: security@example.io (replace with actual security contact)

**Do NOT open public issues for security vulnerabilities.**

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Suggested fix (if any)

### Response Timeline

| Step | Timeline |
|------|----------|
| Acknowledgment | Within 48 hours |
| Assessment | Within 7 days |
| Fix Development | Within 30 days (critical) |
| Public Disclosure | After fix is released |

## Security Measures

### Supply Chain Security

- **Container scanning**: Trivy scans all Docker images for vulnerabilities
- **Dependency scanning**: govulncheck runs on every PR
- **Signed images**: Release images are signed with cosign
- **SLSA Level 3**: Build provenance is generated for all releases

### Runtime Security

- **Non-root containers**: All containers run as non-root
- **Read-only filesystem**: Root filesystem is read-only
- **No privilege escalation**: `allowPrivilegeEscalation: false`
- **Capability dropping**: All Linux capabilities are dropped
- **Seccomp profile**: RuntimeDefault seccomp profile is used
- **Network policies**: Default-deny ingress/egress policies
- **Least-privilege RBAC**: Minimal permissions required

### Network Security

- **TLS everywhere**: Webhooks use cert-manager for TLS
- **Network policies**: Operator namespace has restricted egress
- **Service mesh compatible**: Works with Istio/Linkerd mTLS

### Data Security

- **No secrets in logs**: Sensitive data is never logged
- **RBAC scoped**: ClusterRole limited to required resources
- **Audit logging**: Kubernetes audit logs capture all API access

## Security Updates

Security patches are released as soon as possible after a vulnerability
is confirmed. Users are notified via:

- GitHub Security Advisories
- Release notes
- Slack channel (if configured)

## Dependency Management

- Dependencies are reviewed monthly
- `go mod tidy` is run regularly
- Vulnerable dependencies are updated within 7 days of disclosure
- `govulncheck` runs in CI on every PR

## Compliance

This project follows:
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [OWASP Kubernetes Security Cheat Sheet](https://owasp.org/www-project-kubernetes-top-ten/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)
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
