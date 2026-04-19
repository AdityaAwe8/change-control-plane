# Live Environment Proof Status

This document is intentionally conservative. “Live-like” means the repository now proves behavior against richer upstream harnesses and changing state. It does not mean the subsystem has been proven against a real customer cluster or production metrics backend.

## Current Classification

| Area | Status | Notes |
| --- | --- | --- |
| Kubernetes provider sync and action shaping | live-like and credible | Provider tests cover sync, pause, rollback patch shaping, upstream failures, and changing rollout state against realistic HTTP responses. |
| Kubernetes workload discovery | local-cluster/local-metrics proven | The reference pilot now syncs a real local `k3s` deployment through `kubectl proxy` and persists the mapped workload resource. |
| Prometheus query collection | local-cluster/local-metrics proven | The reference pilot now queries a real in-cluster Prometheus instance and persists real signal snapshots from the sample workload. |
| Prometheus coverage persistence | local-cluster/local-metrics proven | Signal-target resources are persisted and mapped in the reference pilot, though they are still derived from configured queries rather than a broader discovery protocol. |
| End-to-end advisory recommendation flow | local-cluster/local-metrics proven | The reference pilot report now proves `advisory_rollback`, `advisory_only=true`, and suppressed provider actions with preserved audit and status evidence. |
| External proof runner | implemented and verified in-repo | `cmd/live-proof-verify`, `scripts/live-proof-verify.sh`, `scripts/live-proof-validate.sh`, and command tests now prove the repository has a reusable path for hosted GitHub/GitLab plus customer-like Kubernetes/Prometheus verification and saved-report validation. |
| Kubernetes live cluster proof | partial | The client remains HTTP-backed and repository-proven, not `client-go` or live-cluster proven. |
| Prometheus live metrics proof | partial | The client remains HTTP-backed and harness-proven, not validated against a real hosted or production Prometheus deployment. |

## What Improved In This Milestone

- Repeated Kubernetes sync now proves disappearing inventory and resource disappearance handling instead of only steady-state success.
- Repeated Prometheus collection now proves configured collection windows and honest warning behavior when no samples are returned.
- Coverage summaries no longer overclaim runtime coverage when discovered workloads disappear from the latest sync.
- The repository now includes a reproducible reference pilot environment that proves one advisory-only end-to-end flow against a real local cluster and real local Prometheus deployment.
- The repository now also includes a reusable external proof runner for hosted SCM plus customer-like Kubernetes/Prometheus environments, instead of only local pilot and harness-oriented tracks.

## What Is Still Not Proven

- Real Kubernetes auth models such as kubeconfig, cluster certificates, or controller-runtime reconciliation.
- Real Prometheus auth/routing patterns beyond bearer-token HTTP access.
- Hosted GitLab or GitHub enterprise providers as the SCM source in the reference pilot flow.
- Long-running live-environment soak behavior in a real business environment.
