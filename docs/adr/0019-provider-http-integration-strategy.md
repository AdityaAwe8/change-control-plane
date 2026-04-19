# ADR 0019: Provider HTTP Integration Strategy

## Status

Accepted

## Decision

Deepen the Kubernetes and Prometheus seams through HTTP-backed near-real provider clients while preserving the simulated providers as the fully verified local path.

## Rationale

- simulated providers remain essential for deterministic CI and smoke coverage
- the platform still needs a meaningful path toward real external integration
- HTTP-backed provider clients can be tested thoroughly without introducing cluster-specific bootstrapping in this phase

## Consequences

- Kubernetes and Prometheus are now near-real rather than normalization-only
- production-grade cluster auth and full provider SDKs remain future work
- the provider abstraction remains stable for later extraction or richer clients
