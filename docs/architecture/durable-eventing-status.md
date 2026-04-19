# Durable Eventing Status

## Classification

| Area | Status | Reality |
| --- | --- | --- |
| Event persistence | real and credible | Important platform events now persist in `outbox_events` before dispatch. |
| Dispatch loop | real and credible | The worker dispatches pending outbox items on each pass before rollout and integration work. |
| Retry behavior | partial | Failed dispatches record status, attempts, last error, and next-attempt backoff. |
| Recovery after restart | partial | Pending and errored items survive restart because they live in storage. |
| Replay / dead-letter tooling | missing | There is no operator replay UI/CLI and no dead-letter queue yet. |
| Multi-worker proof | partial | Claiming is persisted and stale claims are recoverable, but there is no serious multi-process chaos proof yet. |

## Durable Today

- Webhook receipts can be recorded durably.
- Sync requested / completed / failed flows can be published durably.
- Status-event and rollout-related publication now has a restart-safe substrate instead of only an in-memory fanout.

## Not Durable Yet

- This is not a general distributed event bus.
- There is no external subscriber ecosystem.
- There is no guaranteed once-only global delivery story.

The current design is intentionally pragmatic: durable enough for serious pilot operations, not overbuilt beyond what this repo can honestly prove.
