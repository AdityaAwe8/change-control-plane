# ADR 0016: Runtime Signal Snapshot Model

## Status

Accepted

## Context

Verification decisions need reconstructable evidence. Live telemetry systems vary widely, and we do not want rollout safety to depend on provider-specific payload shapes or opaque model outputs.

## Decision

We normalize runtime verification input into persisted `signal_snapshots`:

- snapshots are bound to organization, project, rollout execution, plan, change set, service, and environment
- each snapshot stores normalized health, summary, signal values, provider type, and time window
- automated verification decisions reference the snapshot ids they used
- signal providers can be push-based or pull-based as long as they emit the normalized shape

## Consequences

- verification inputs are durable and auditable
- provider adapters can evolve independently from the control path
- simulated and future Prometheus-style providers can share one deterministic verification engine
- the schema is ready for richer business metrics without coupling the runtime path to a single telemetry vendor
