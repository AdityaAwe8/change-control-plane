# ADR 0017: Rollback Policy Model

## Status

Accepted

## Decision

Add a persisted `rollback_policies` model scoped by organization with optional project, service, and environment narrowing.

The runtime resolves the most specific enabled policy and falls back to a deterministic built-in policy when no persisted override matches.

## Rationale

- automated rollback needs explicit persisted control-plane guardrails
- enterprise operators need visible override points
- the decision path must remain deterministic and explainable

## Consequences

- rollback behavior is now configuration-backed rather than hardcoded only
- policy breadth can expand without changing the execution model
- policy inheritance is specificity-based in v1, not a full hierarchical merge system
