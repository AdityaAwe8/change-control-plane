# ADR 0021: Integration Instance And Sync-Run Model

## Status

Accepted

## Decision

Treat the existing org-scoped integration rows as the current integration-instance model and add persisted sync-run records for tests, syncs, and webhook deliveries.

## Rationale

- the repository already had organization-scoped integration persistence
- the highest-value gap was operational state, not a full marketplace redesign
- sync-run records give the product real health, recency, and webhook idempotency evidence

## Consequences

- the current product supports one seeded instance per integration kind per org
- connection history now exists as first-class persisted evidence
- many-named-instances-per-kind remains future work
