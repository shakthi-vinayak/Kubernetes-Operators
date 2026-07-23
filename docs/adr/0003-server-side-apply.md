# ADR-0003: Server-Side Apply for Resource Management

## Status

Accepted

## Context

The operator needs to create and update child resources (Deployments, Services, etc.) in a way that:

1. Is idempotent (calling apply N times produces the same result)
2. Detects and corrects drift (manual changes are reverted)
3. Respects fields managed by other controllers
4. Avoids unnecessary API writes when state is already correct

Three approaches exist for managing child resources:

1. **Annotation-based tracking** — store the last-applied state in an annotation, compute a three-way merge patch
2. **Full replace** — always overwrite the entire resource spec (simple but destructive)
3. **Server-Side Apply (SSA)** — use Kubernetes' built-in managed fields tracking via `PATCH` with `application/apply-patch+yaml`

## Decision

Use **Server-Side Apply** via `client.Patch` with `client.Apply` and `client.FieldOwner`.

## Alternatives Considered

- **Annotation-based tracking**: Requires storing last-applied JSON in annotations, manual three-way merge logic, annotation size limits
- **Full replace**: Overwrites fields managed by other controllers (e.g., HPA-managed replica count), causes unnecessary API writes

## Consequences

- **Positive**: Kubernetes-native field ownership tracking — no custom annotation logic
- **Positive**: Built-in conflict detection when multiple managers touch the same fields
- **Positive**: Respects managed fields — the operator only owns the fields it explicitly sets
- **Positive**: Avoids unnecessary writes — SSA only patches fields that differ
- **Negative**: Requires understanding of field manager semantics
- **Negative**: `ForceOwnership` needed when the operator must reclaim fields from another manager
- **Neutral**: The field manager name (`platform-operator`) is a contract — changing it would cause field ownership conflicts
