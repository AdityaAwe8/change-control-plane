# Control Loop

The worker is now a real reconciler, not a heartbeat placeholder.

## Responsibilities

- list rollout executions within the worker actor's tenant scope
- claim each execution for the current reconciliation pass
- auto-start `planned` and `approved` executions when auto-advance is enabled
- auto-complete `verified` executions when auto-advance is enabled
- reconcile desired state against the selected orchestrator provider
- collect runtime signal context from the selected signal provider
- resolve the effective rollback policy for each execution
- run deterministic verification evaluation
- trigger automated pause or rollback when policy and signal conditions require it
- persist runtime state, decisions, and failures
- persist canonical status events alongside audit-visible evidence
- emit audit-visible evidence for every automated action

## Idempotency Strategy

- execution claims update `last_reconciled_at`
- provider submission is stable because backend execution identifiers are persistent
- verification decisions are deduplicated against the latest automated result and snapshot linkage
- rollback calls are suppressed when the execution is already in a rollback terminal or in-flight rollback state
- reconcile writes update stored backend status instead of appending uncontrolled duplicate state

## Current Limits

- claims are sufficient for the current single-process modular-monolith worker, but not yet a full distributed lease system
- retries are safe for the simulated provider path, and the HTTP-backed Kubernetes and Prometheus adapters now classify transient versus terminal failures, but cluster-grade controller behavior is still future work
- the worker logs progress, counts, and failures, but metrics export is still minimal
- continuous telemetry ingestion is still not implemented; runtime signals are collected during reconcile or ingested through explicit APIs
