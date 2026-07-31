# Outbox Recovery Status

## Current Classification

| Area | Status | Notes |
| --- | --- | --- |
| Durable event persistence | real and credible | Important events are persisted in the outbox before dispatch. |
| Worker dispatch and retry | real and credible | The worker claims pending items and retries failed dispatch with backoff. |
| Stale-claim recovery | real and credible | Tests now prove stale `processing` items are reclaimed after the claim window. |
| Dead-letter / quarantine behavior | partial | Events now stop retrying forever and move to `dead_letter` when failures are permanent or retry budget is exhausted. Recovery hints are recorded in metadata. |
| Recovery tooling | partial | Org admins can retry `error` events and requeue `dead_letter` events through API, CLI, and web. There is still no processed-event replay console or broad event-replay workflow. |
| Multi-process crash proof | partial | The model is more restart-safe than before, but it is still not proven as a broader distributed event bus. |

## Notes

- The outbox remains intentionally lean. This milestone improves reliability and operator diagnostics without turning the product into a full message-bus platform.
- The retry/requeue path and dead-letter state are real operational improvements, but they are not yet equivalent to a full dead-letter queue or replay platform.
