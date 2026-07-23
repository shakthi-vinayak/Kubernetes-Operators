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
