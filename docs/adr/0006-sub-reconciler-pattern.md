# ADR-0006: Sub-Reconciler Pattern

## Status

Accepted

## Context

The operator manages 7 different child resource types (Deployment, Service, HPA, HTTPRoute, NetworkPolicy, PDB, ServiceMonitor). Two architectural approaches exist:

1. **Monolithic reconciler** — a single large `Reconcile` function handles all resource types inline
2. **Sub-reconciler pattern** — each resource type has its own dedicated reconciliation function, orchestrated by a parent reconciler

## Decision

Use the **sub-reconciler pattern**: each child resource type has an independent sub-reconciler function in `internal/controller/subreconcilers/`.

Each sub-reconciler:
- Computes the desired resource as a **pure function** of the PlatformApplication spec
- Calls a shared `Apply()` function to create/update via SSA
- Returns an `ApplyResult` (created/updated/unchanged) for observability

## Alternatives Considered

- **Monolithic reconciler**: Simpler call structure but harder to test and maintain as resource count grows
- **Separate controllers**: One controller per resource type — overkill for this scope, increases complexity

## Consequences

- **Positive**: Each sub-reconciler is independently testable with table-driven unit tests
- **Positive**: Adding a new resource type requires only a new file + one line in the parent orchestrator
- **Positive**: Pure `buildDesired*` functions have zero side effects — trivially testable
- **Positive**: Clear separation of "compute desired state" from "apply to cluster"
- **Negative**: Small overhead from the abstraction layer for simple operators
- **Neutral**: The shared `Apply()` function enforces consistent owner reference and SSA behavior across all sub-reconcilers
