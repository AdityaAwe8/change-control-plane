# ADR 0037: Durable Outbox Eventing

## Decision

Persist important internal platform events into an outbox table and dispatch them through the worker loop with retry metadata.

## Why

- In-memory-only eventing was too fragile for serious pilot operation.
- The repo needed restart-safe reliability without replacing the current architecture with a full distributed bus.

## Consequences

- Event publication is now materially more durable.
- Operators can inspect outbox state.
- Replay/dead-letter/multi-cluster concerns remain future work.
