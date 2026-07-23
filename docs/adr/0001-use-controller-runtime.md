# ADR-0001: Use controller-runtime over Operator SDK

## Status

Accepted

## Context

When building a Kubernetes operator, two primary framework options exist:

1. **Operator SDK** — Red Hat's higher-level framework built on top of controller-runtime
2. **controller-runtime** — The underlying Kubernetes controller framework used by Operator SDK

Operator SDK adds convenience features (Ansible/Helm operators, scorecard, OLM integration) but introduces additional dependencies and abstraction layers.

## Decision

Use **controller-runtime** directly.

## Alternatives Considered

- **Operator SDK**: More opinionated, includes OLM/OLM integration, supports Ansible/Helm operators
- **Raw client-go**: Maximum control but requires reimplementing reconciliation loops, caching, leader election

## Consequences

- **Positive**: Lighter dependency tree, direct access to controller-runtime APIs, no unnecessary abstraction, easier to understand internals, better for demonstrating Kubernetes software engineering knowledge
- **Positive**: Same underlying framework as Operator SDK — skills transfer directly
- **Negative**: Must implement some conveniences manually (e.g., metrics registration, webhook setup)
- **Neutral**: No OLM integration by default (can be added later if needed)
