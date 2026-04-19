# Prometheus Pilot Readiness

This document describes the actual pilot-readiness state of the Prometheus live signal path.

## Current Classification

| Area | State | Reality |
| --- | --- | --- |
| Integration descriptor and persistence | fully live | org-scoped integration instances persist mode, enablement, health, schedule state, freshness, test/sync timestamps, errors, and sync-run history |
| Config validation | partially implemented | `api_base_url` and at least one query template are now required for enabled integrations; auth and query governance remain lightweight |
| Connection test | near-live | connection tests now use the real Prometheus collection path and record normalized signal evidence rather than a shallow HTTP probe |
| Query execution | near-live | range-query collection is real against HTTP-backed Prometheus endpoints, including threshold normalization |
| Signal normalization | materially live | snapshots normalize health, summary, signal values, thresholds, comparator semantics, and collection windows |
| Error handling | partially implemented | invalid responses, missing sample values, empty results, and HTTP failures are handled, but richer retry/backoff and query linting are still limited |
| Operator evidence surface | materially live | sync runs now expose snapshot health, signal count, window bounds, summary, individual signal details, freshness, and discovered signal targets |
| Live metrics proof | missing | no live Prometheus environment is proven in repository CI |

## What Improved In This Milestone

- Connection test and sync now use the real Prometheus provider collection path.
- Scheduled collection can now keep Prometheus-backed signal evidence fresh without a manual sync button press.
- Sync-run details now include:
  - snapshot count
  - source
  - health
  - window start/end
  - signal count
  - snapshot summary
  - individual signal value and status details
- Query-backed discovery can now persist first-class signal targets for mapping and coverage summaries.
- Empty Prometheus results are now explicitly tested and normalize to zero-valued signals.
- Missing sample values are explicitly tested as parse failures instead of being silently accepted.

## What This Supports For A Pilot

- Real query-range collection against a configured endpoint
- Safe, explainable normalization into the control-plane signal model
- Recurring collection with freshness and stale-state visibility
- Runtime evidence attached to rollout executions
- Advisory-mode verification and recommendation flows backed by Prometheus data

## What It Does Not Yet Support

- Query registry lifecycle management beyond metadata configuration
- Query linting against live cardinality, cost, or tenancy rules
- Multi-source aggregation across several Prometheus environments

## Honest Pilot Position

The Prometheus path is now strong enough for a careful advisory pilot where a business can provide stable query templates and an accessible endpoint. It is still not strong enough to claim fully proven production telemetry integration without live-environment verification.
