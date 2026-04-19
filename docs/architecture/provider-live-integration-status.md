# Provider Live Integration Status

This document tracks how real the orchestrator and signal providers actually are.

Status legend:

- `live_and_verified`
- `near_real_and_verified`
- `simulated_and_verified`
- `normalization_only`
- `missing`

## Current Status

| Provider Surface | Status | Reality |
| --- | --- | --- |
| Simulated orchestrator | `live_and_verified` | Fully exercised in tests and smoke flows. Supports submission, sync, pause, resume, rollback, and verification checkpoints. |
| Simulated signal provider | `live_and_verified` | Fully exercised in tests and smoke flows. Produces deterministic normalized snapshots for CI and local control-loop verification. |
| Kubernetes provider | `near_real_and_verified` | Now performs real HTTP calls against Kubernetes-style deployment status endpoints, normalizes deployment JSON, supports pause/resume through deployment patching, is harness-proven for bearer-auth headers plus custom status-path handling, and is now part of the reusable `live-proof-verify` external proof track. Rollback supports configured action endpoints or configured image patch targets. |
| Prometheus provider | `near_real_and_verified` | Now performs real HTTP query-range requests, normalizes results into signal snapshots, classifies overall signal health deterministically, is harness-proven for bearer-auth headers plus custom query-path handling, and is now part of the reusable `live-proof-verify` external proof track. |
| Provider error classification | `near_real_and_verified` | HTTP-backed providers now distinguish transient from terminal failures through typed provider errors. |
| GitHub runtime integration | `normalization_only` | Still catalog and metadata oriented only. No live deployment-state integration. |
| Slack runtime integration | `normalization_only` | Still descriptor-level only. |
| Jira runtime integration | `normalization_only` | Still descriptor-level only. |

## Reality Check

What became more real in this milestone:

- providers no longer rely only on metadata normalization helpers
- Kubernetes and Prometheus can now interrogate external HTTP surfaces directly
- the simulated providers remain the deterministic, fully verified local path
- the repository now has an explicit operator-facing live proof runner for hosted SCM plus customer-like Kubernetes/Prometheus attachment

What is still not claimed:

- no live cluster credential bootstrap
- no `client-go`-based Kubernetes controller loop
- no production-grade Prometheus auth/tenant routing layer
- no live GitHub, Slack, or Jira operational control integration
