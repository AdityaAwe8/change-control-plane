# ADR 0020: Automated Rollback Control Path

## Status

Accepted

## Decision

Keep automated rollback deterministic inside the Go control plane, with Python limited to non-safety-critical explanation and planning augmentation.

## Rationale

- rollback legality must be deterministic and reconstructable
- operator trust depends on explainable, non-opaque control decisions
- Python remains valuable for enrichment without becoming a safety risk

## Consequences

- the verification engine owns rollback legality
- the worker owns rollback execution through provider adapters
- Python does not decide whether a rollout is paused or rolled back
