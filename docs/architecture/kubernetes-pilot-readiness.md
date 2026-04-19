# Kubernetes Pilot Readiness

This document describes the actual pilot-readiness state of the Kubernetes live integration path.

## Current Classification

| Area | State | Reality |
| --- | --- | --- |
| Integration descriptor and persistence | fully live | org-scoped integration instances persist mode, enablement, control flags, health, last test/sync, and sync-run history |
| Config validation | partially implemented | `api_base_url` plus `status_path` or `namespace + deployment_name` are now required for enabled integrations; richer auth and TLS validation are still future work |
| Connection test | near-live | connection tests now use the real Kubernetes provider sync path and record normalized workload evidence instead of a shallow ping |
| Sync behavior | near-live | sync runs now normalize backend status, progress, step, paused state, and replica counts through the provider, and can now run on a recurring schedule |
| Pause/resume/rollback provider methods | near-live | methods exist and are covered against realistic HTTP-backed behavior, but are not live-cluster proven in this repository |
| Advisory-mode suppression | fully live | live advisory integrations suppress external submit/pause/resume/rollback during reconcile and emit explicit suppressed-action evidence |
| Operator evidence surface | materially live | web/API surfaces now show control mode, freshness, stale state, last provider action, disposition, discovered workloads, and sync-run details |
| Live cluster proof | missing | no `client-go`, no kubeconfig support, and no live cluster verification in repository CI |

## What Improved In This Milestone

- Connection test and sync now exercise the actual Kubernetes provider normalization path.
- Scheduled sync can now keep Kubernetes workload state fresh without operator-triggered manual sync.
- Sync-run details now include:
  - backend status
  - progress percent
  - current step
  - namespace
  - deployment name
  - replica counts
  - paused state
- Sync can now persist discovered workload inventory as first-class runtime resources for later mapping.
- Advisory-mode manual pause/continue/rollback is blocked at the API layer for live advisory backends.
- Advisory reconcile now records explicit suppression evidence rather than leaving operators to infer it from generic runtime updates.

## What This Supports For A Pilot

- Read-only attachment to a known workload target
- Normalized deployment-state observation
- Recurring workload refresh with freshness and stale-state visibility
- Provider-backed health evidence in sync history
- Advisory-only reconcile with recommendation recording
- Safer operator understanding of what was observed versus executed

## What It Does Not Yet Support

- Native Kubernetes auth models such as kubeconfig, service-account discovery, or mTLS profile management
- Live cluster integration proof in automated verification
- Watch-based or controller-style reconciliation
- Workload discovery across namespaces
- Revision history lookup for true rollback provenance

## Honest Pilot Position

The Kubernetes path is now credible for a careful advisory pilot when the business can provide a stable HTTP-accessible Kubernetes-style endpoint and explicit target mapping. It is still not deep enough to overclaim “production-grade Kubernetes control-plane integration” without live cluster proof.
