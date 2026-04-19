# Outbox Recovery

The outbox remains the durable backbone for important operational events.

## Hardening Added

- retries now back off more conservatively
- stale `processing` claims are explicitly proven to be reclaimable
- permanent failures or exhausted retry budgets now move to `dead_letter`
- outbox metadata now records failure class and recovery hints

## Current Operating Model

- `pending` or `error` items can be claimed when due
- stale `processing` items can be reclaimed
- `processed` items are terminal
- `dead_letter` items are terminal until future replay tooling exists

## Remaining Gaps

- no replay endpoint yet
- no dead-letter requeue workflow yet
- still not a distributed bus or full operations platform
