# ADR 0040: Outbox Recovery and Retry Model

## Decision

Extend the durable outbox with:

- reclaimable stale processing claims
- bounded retry with backoff
- dead-letter terminal state
- failure-class metadata and recovery hints

## Why

This gives the worker a more operationally credible recovery model without introducing a full distributed event bus.

## Tradeoff

Replay and dead-letter requeue remain future work.
