# Runtime Signals

Runtime verification now operates on persisted normalized `signal_snapshots`.

## Signal Snapshot Model

Each snapshot is scoped to:

- organization
- project
- rollout execution
- rollout plan
- change set
- service
- environment
- provider type
- optional source integration
- verification window

Each snapshot carries:

- normalized `health`
- human-readable `summary`
- structured `signals`
- provider/source metadata

## Normalized Signal Values

Signals are stored with:

- `name`
- `category`
- `value`
- `unit`
- `status`
- `threshold`
- `comparator`

This keeps verification deterministic while preserving enough context for later provider-specific expansion.

## Current Providers

- `simulated`
  - live and verified
  - intended for local development, CI, and demo control loops
  - snapshots are ingested through the API/CLI/web
- `prometheus`
  - near-real and verified
  - issues real HTTP `query_range` requests against a Prometheus-style endpoint
  - normalizes response values into persisted signal snapshots during reconcile
  - supports configured windows, step size, thresholds, units, and severity hints

## Verification Inputs

The verification engine currently considers:

- backend execution status
- latest normalized signal health
- explicit rollback-policy threshold breaches derived from normalized signal values
- rollout plan verification requirements
- environment production posture
- service criticality and customer-facing posture
- assessed risk level

Python intelligence is not part of the safety-critical decision path here by design.
