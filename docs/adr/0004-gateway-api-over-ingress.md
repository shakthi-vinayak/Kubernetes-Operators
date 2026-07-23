# ADR-0004: Gateway API over Ingress

## Status

Accepted

## Context

The operator needs to expose applications via external traffic. Two Kubernetes-native approaches exist:

1. **Ingress** (stable since v1.19) — the traditional approach, widely supported by ingress controllers
2. **Gateway API** (GA since v1.0) — the next-generation Kubernetes ingress standard with role-based resource model

## Decision

Use **Gateway API** (specifically `HTTPRoute` from `gateway.networking.k8s.io/v1`).

## Alternatives Considered

- **Ingress**: Stable, widely supported, simpler for basic use cases
- **Both**: Implement both with a toggle (increased complexity for marginal benefit)

## Consequences

- **Positive**: Forward-looking — Gateway API is the future of Kubernetes ingress
- **Positive**: Role separation — Gateway (cluster-admin owned) vs HTTPRoute (app team owned)
- **Positive**: More expressive routing: header matching, weight-based traffic, path types
- **Positive**: Standard across ingress controller implementations (Envoy, nginx, Traefik, etc.)
- **Negative**: Requires Gateway API CRDs installed on the cluster
- **Negative**: The Gateway itself is externally managed — the operator only creates HTTPRoutes
- **Negative**: Newer API surface — less operational experience in the community
- **Mitigation**: Operator checks for Gateway API CRD existence before creating HTTPRoutes; sets a condition if missing
