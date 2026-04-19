# Durable Eventing

## Purpose

The platform previously relied on in-memory event fanout only. That was too weak for serious pilot reliability because important events could disappear across failure or restart.

This milestone adds a durable outbox-backed layer for important internal platform events.

## Model

- publish important events into `outbox_events`
- worker claims due events
- dispatch to in-process subscribers
- mark processed on success
- record attempts, last error, and next-attempt time on failure

## Why This Design

- It materially improves reliability without replacing the current architecture.
- It keeps the event model simple and repo-proven.
- It leaves room for a future external bus if the product grows into that need.

## Honest Limits

- The current outbox is not a full streaming platform.
- Replay tooling is still minimal.
- Multi-process operational proof is still limited.
