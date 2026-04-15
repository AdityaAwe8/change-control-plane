# ADR 0011: Rollout Plans Produce Durable Execution Records

## Status

Accepted

## Context

Static rollout plans were useful for governance, but not enough to make ChangeControlPlane feel like an operational control plane.

## Decision

- Introduce a dedicated `rollout_executions` record.
- Keep rollout plans immutable and execution records mutable.
- Enforce state transitions in deterministic application logic.
- Persist verification results separately and link them to executions.
- Use audit events for transition history while keeping the current execution state on the execution record.

## Consequences

- The platform can track live rollout posture without requiring a workflow engine on day one.
- Later Temporal or controller integrations can drive the same execution model instead of replacing it.
