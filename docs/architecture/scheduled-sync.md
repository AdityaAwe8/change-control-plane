# Scheduled Sync

Scheduled sync is now a real persisted runtime behavior.

## Persisted Fields

Each integration can now persist:

- `schedule_enabled`
- `schedule_interval_seconds`
- `sync_stale_after_seconds`
- `next_scheduled_sync_at`
- `last_sync_attempted_at`
- `last_sync_succeeded_at`
- `last_sync_failed_at`
- `sync_consecutive_failures`

Each sync run can now persist:

- `trigger`
- `scheduled_for`
- `error_class`

## Trigger Semantics

- `manual`: operator-triggered test or sync
- `scheduled`: a due scheduled sync
- `retry`: a scheduled sync after recent failures
- `webhook`: an inbound GitHub delivery

## Worker Behavior

The worker now:

1. scans integrations after rollout reconcile work
2. ignores disabled or unscheduled integrations
3. checks whether `next_scheduled_sync_at` is due
4. claims the sync with a lease-like timestamp update
5. runs the provider-backed sync
6. persists the run outcome and next due time

Retry timing is intentionally simple exponential backoff capped below the normal schedule interval.

## Safety

Scheduled sync is advisory-safe:

- it collects state
- it persists evidence
- it does not authorize external control actions for live advisory integrations
