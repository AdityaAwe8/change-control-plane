# Live Runtime Integration Status

This document focuses specifically on whether the control plane is truly capable of closed-loop rollout execution and runtime verification.

Status legend:

- `live_and_verified`
- `partially_live`
- `stubbed`
- `metadata_only`
- `simulated_only`
- `missing`

## Current Status

| Area | Status | Reality |
| --- | --- | --- |
| Rollout execution records | `live_and_verified` | Executions now persist desired state, backend state, progress, sync timestamps, and timeline evidence. |
| Rollout state machine | `live_and_verified` | Legal transitions are enforced and now converge with normalized provider state through the reconciler. |
| Worker control loop | `live_and_verified` | The worker claims executions, auto-starts eligible rollouts, reconciles provider state, evaluates runtime signals, and records automated decisions. |
| Orchestrator adapter model | `live_and_verified` | A real internal provider abstraction exists with submission, sync, pause, resume, and rollback methods. |
| Integration registry | `partially_live` | Descriptor-based integration metadata still exists, but rollout execution now also uses dedicated orchestrator and signal provider registries. |
| Kubernetes integration | `partially_live` | A near-real Kubernetes deployment provider seam and normalization layer now exist, but live cluster calls still require future client wiring. |
| GitHub integration | `metadata_only` | Present as a catalog descriptor only. No live workflow or deployment-state integration exists yet. |
| Runtime signal model | `live_and_verified` | Normalized signal snapshots are persisted and bound to rollout executions, services, environments, and plans. |
| Signal provider abstraction | `live_and_verified` | A signal-provider registry exists with a live simulated provider and a Prometheus-style normalization seam. |
| Verification persistence | `live_and_verified` | Verification results now persist automated/manual provenance, snapshot linkage, technical summaries, and decisions. |
| Automated verification engine | `live_and_verified` | A deterministic verification engine evaluates backend state, signal health, environment criticality, and risk posture. |
| Automated control decisions | `live_and_verified` | The control loop now generates verified, pause, rollback, and failed decisions from runtime evidence. |
| Audit for automated runtime actions | `live_and_verified` | Runtime sync, signal ingestion, automated verification, and reconcile failures all write audit-visible evidence. |
| Python intelligence in runtime loop | `metadata_only` | Python still augments risk and rollout planning only. The runtime control path remains deterministic by design. |
| Web execution visibility | `partially_live` | The web UI now shows backend status, signal snapshots, verification history, and timeline data, but it still lacks browser interaction tests. |
| CLI execution visibility | `live_and_verified` | The CLI can reconcile executions, ingest simulated signals, inspect detail, and watch execution state. |
| End-to-end closed loop | `live_and_verified` | The persisted smoke path now proves machine-authenticated start, backend reconcile, signal ingestion, and automated verification. |

## Top Blockers

1. The live backend path is simulated-first; Kubernetes and Prometheus remain near-real seams rather than production-connected clients.
2. Verification still depends on pushed or simulated signal snapshots, not on continuous polling from a live telemetry platform.
3. Worker claim semantics are safe for the current modular-monolith model, but distributed coordination will need stronger leasing later.
4. Web and CLI coverage improved materially, but browser interaction tests and richer operator ergonomics still lag the backend.
5. GitHub, Slack, and Jira remain metadata-oriented adapters rather than live runtime control participants.
