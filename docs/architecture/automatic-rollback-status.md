# Automatic Rollback Status

This document tracks the real implementation state of automated rollback in Change Control Plane.

Status legend:

- `live_and_verified`
- `simulated_and_verified`
- `partially_implemented`
- `missing`

## Current Status

| Area | Status | Reality |
| --- | --- | --- |
| Rollback decision path | `live_and_verified` | The deterministic verification engine now evaluates provider failure, signal health, and resolved rollback policy before deciding `pause`, `rollback`, `manual_review_required`, `failed`, or `verified`. |
| Rollback policy persistence | `live_and_verified` | Rollback policies are persisted in PostgreSQL, queryable through the API, and resolved by scope specificity at runtime. |
| Built-in fallback policy | `live_and_verified` | When no persisted override matches, the control plane uses a deterministic built-in policy derived from service criticality, environment production posture, and risk level. |
| Automated rollback execution | `simulated_and_verified` | For the simulated backend, the worker now reaches a real rollback state end to end and persists the resulting runtime and verification evidence. |
| Provider-issued rollback action | `partially_implemented` | The provider abstraction now supports rollback against both simulated backends and near-real HTTP-backed adapters. Kubernetes rollback requires either a configured rollback endpoint or a configured container/image patch target. |
| Rollback idempotency | `partially_implemented` | Duplicate rollback requests are suppressed for already-rolled-back and rollback-requested executions, but richer distributed lease semantics are still future work. |
| Manual rollback override safety | `live_and_verified` | Manual pause/resume/rollback and manual verification outcomes that force rollback now require elevated rollout override permissions. |
| Rollback visibility | `live_and_verified` | Rollback decisions, reasons, status transitions, and related signal evidence appear in rollout detail, status history, and audit records. |

## Reality Check

What is real now:

- rollback policies are persisted and effective at execution time
- automatic rollback decisions are explainable and auditable
- the worker can carry a rollback decision through to a provider action
- rollout detail now exposes the effective rollback policy and canonical status timeline
- smoke verification proves automated rollback and status history against the simulated runtime path

What is still limited:

- Kubernetes rollback is near-real, not cluster-proven in this repository
- rollback does not yet integrate with a live artifact history or revision ledger
- rollback retry/backoff handling is still basic rather than orchestrator-grade
