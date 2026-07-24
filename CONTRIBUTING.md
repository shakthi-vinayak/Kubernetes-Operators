# Contributing to Platform Operator

Thank you for your interest in contributing! This guide covers how to get started,
development workflow, and coding standards.

## Getting Started

### Prerequisites

- Go 1.24+
- Docker
- kubectl
- kind (for local development)
- make

### Development Setup

```bash
# Clone the repository
git clone https://github.com/shakthi-vinayak/Kubernetes-Operators.git
cd Kubernetes-Operators

# Install dependencies
make setup

# Build the operator
make build

# Run unit tests
make test-unit

# Run all tests (requires envtest)
make test
```

### Local Development with Kind

```bash
# Create a Kind cluster
make kind-create

# Build and load the operator image
make kind-load

# Deploy the operator
kubectl apply -k config/overlays/dev

# Create a test application
kubectl apply -f config/samples/platform_v1alpha1_platformapplication.yaml
```

## Development Workflow

### Making Changes

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-change`
3. Make your changes with tests
4. Run `make fmt vet lint test`
5. Commit with a descriptive message
6. Push and open a Pull Request

### Code Standards

- **Formatting**: `go fmt ./...` (enforced by CI)
- **Linting**: `golangci-lint run ./...` (enforced by CI)
- **Testing**: All new code must have unit tests
- **Documentation**: Update relevant docs for API changes

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add support for custom pod annotations
fix: handle nil pointer in HPA reconciliation
docs: update operations guide with scaling instructions
test: add chaos test for rapid create/delete cycles
refactor: extract sub-reconciler metrics into helper
```

### Pull Request Process

1. All PRs require at least 1 approval
2. CI must pass (lint, test, build)
3. Squash merge is preferred
4. Update CHANGELOG.md for user-facing changes

## Architecture

### Project Structure

```
├── api/                    # CRD type definitions (v1alpha1, v1beta1, v1)
├── charts/                 # Helm chart
├── cmd/                    # Main entry point
├── config/                 # Kustomize manifests
│   ├── crd/               # Generated CRDs
│   ├── rbac/              # RBAC roles
│   ├── overlays/          # Environment overlays
│   └── components/        # Reusable components
├── internal/              # Core logic
│   ├── controller/        # Reconciler
│   ├── errors/            # Error classification
│   ├── metrics/           # Prometheus metrics
│   ├── status/            # Status condition helpers
│   └── tracing/           # OpenTelemetry tracing
├── test/                  # Test suites
│   ├── chaos/             # Chaos tests
│   ├── failure/           # Failure injection tests
│   ├── ha/                # HA tests
│   └── scale/             # Scale tests
├── docs/                  # Documentation
├── gitops/                # Argo CD configurations
└── hack/                  # Development utilities
```

### Key Patterns

- **Sub-reconciler pattern**: Each child resource has its own reconciliation function
- **Server-Side Apply**: Used for drift correction and conflict resolution
- **Error classification**: Errors are classified as transient, conflict, or permanent
- **Hub/Spoke conversion**: v1beta1 is the storage hub; v1alpha1 and v1 are spokes

## Testing

### Running Tests

```bash
# Unit tests
make test-unit

# Integration tests (requires envtest)
make test-integration

# Chaos tests
make test-chaos

# Scale tests
go test ./test/scale/... -v

# Benchmark tests
make test-bench

# Coverage with threshold
make test-coverage
```

### Writing Tests

- Unit tests: alongside source code in `_test.go` files
- Integration tests: `test/integration/` (uses envtest)
- E2E tests: `test/e2e/` (requires real cluster)
- Chaos tests: `test/chaos/` (fault injection)
- Scale tests: `test/scale/` (1000+ CRs)

## Release Process

1. Create a release branch: `git checkout -b release/v0.x.0`
2. Update CHANGELOG.md
3. Update Chart.yaml version
4. Open PR for review
5. After merge, tag: `git tag v0.x.0`
6. CI builds and pushes the Docker image and Helm chart
# Contributing to Platform Operator

Thank you for your interest in contributing! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.22+
- Docker
- kubectl
- Kind (for local development)
- Make (or compatible build tool)

### Development Setup

```bash
# Clone the repository
git clone https://github.com/example/platform-operator.git
cd platform-operator

# Install development dependencies
make setup

# Run tests
make test

# Build the binary
make build

# Create a local Kind cluster
make kind-create

# Install CRDs
make install

# Run the operator locally (against the Kind cluster)
make run
```

## Development Workflow

### Code Standards

- All Go code must pass `gofmt`, `go vet`, and `golangci-lint`.
- Follow idiomatic Go practices and [Effective Go](https://go.dev/doc/effective_go) guidelines.
- Write tests for all new functionality.
- Use structured logging via `logr` (do not use `fmt.Print`).
- Document exported types and functions with Go doc comments.

### Making Changes

1. Fork the repository and create a feature branch.
2. Make your changes with appropriate tests.
3. Ensure all tests pass: `make test`
4. Ensure linting passes: `make lint`
5. Submit a pull request with a clear description.

### Testing

```bash
# Unit tests
make test-unit

# Integration tests (requires envtest)
make test-integration

# All tests with race detector
make test-race

# E2E tests (requires a running cluster)
make e2e
```

### Pull Request Process

1. All PRs require at least one review.
2. CI must pass (lint, test, build, docker build).
3. Ensure your branch is up to date with main.
4. Squash commits before merging.

## Architecture

The operator follows the standard controller-runtime pattern:

- `api/` — CRD type definitions and webhook logic
- `cmd/` — Entry point (main.go)
- `internal/controller/` — Reconciliation logic
- `internal/controller/subreconcilers/` — Per-resource reconciliation
- `internal/status/` — Status condition helpers
- `internal/metrics/` — Prometheus metrics
- `config/` — Kustomize manifests and RBAC

## Reporting Issues

- Use GitHub Issues for bug reports and feature requests.
- Include Kubernetes version, operator version, and relevant logs.
- For security issues, see [SECURITY.md](SECURITY.md).

## Code of Conduct

This project follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/main/code-of-conduct.md).
