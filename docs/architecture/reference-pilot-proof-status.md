# Reference Pilot Proof Status

This document is intentionally conservative. “Local-reference proven” means the repository now demonstrates the flow against a real local cluster and real local Prometheus instance with reproducible setup scripts. It does not mean a customer production environment has been proven.

## Current Classification

| Area | Status | Notes |
| --- | --- | --- |
| Reference pilot environment bootstrap | local-cluster/local-metrics proven | `scripts/reference-pilot-up.sh` now brings up dedicated pilot dependencies, a local `k3s` cluster, the sample workload, Prometheus, the local GitLab fixture, and the API. |
| SCM change path in the pilot | local-cluster/local-metrics proven | The proof flow uses a real control-plane GitLab integration, automatic webhook registration, repository discovery, and merge-request webhook ingest against the local GitLab fixture. |
| Repository, workload, and signal mapping | local-cluster/local-metrics proven | `cmd/reference-pilot-verify` maps the discovered GitLab repository, Kubernetes workload, and Prometheus signal target into the same project/service/environment scope. |
| Kubernetes workload discovery and observation | local-cluster/local-metrics proven | The product syncs against a real local `k3s` deployment through `kubectl proxy` and records the observed workload as a discovered resource. |
| Prometheus query-window collection | local-cluster/local-metrics proven | The product collects real query windows from the in-cluster Prometheus deployment and persists the resulting signal snapshot. |
| Advisory-only runtime recommendation | local-cluster/local-metrics proven | The reference flow now records `advisory_rollback`, `advisory_only=true`, and `last_action_disposition=suppressed` after collecting critical runtime signals. |
| Audit and status evidence | local-cluster/local-metrics proven | The proof report includes audit events, status events, signal snapshots, verification results, and rollout timeline evidence from the live local flow. |
| Outbox and recovery behavior in the pilot | partial | Durable eventing remains real, but the reference pilot flow is not yet a crash/restart soak test of the outbox worker. |
| Browser-only operator flow in the pilot environment | partial | The web app can inspect the same pilot state, but the canonical proof today is script/API driven rather than a full browser-only walkthrough. |
| Hosted-provider and production-environment proof | missing | The reference pilot uses a local GitLab fixture, a local `k3s` cluster, and a local Prometheus deployment. |

## Main Remaining Gaps

- The SCM portion of the reference pilot is still local-fixture proof, not hosted GitLab or GitHub proof.
- Kubernetes is now local-cluster proven, but still not honestly production-cluster proven.
- Prometheus is now local-metrics proven, but still not honestly hosted or production telemetry proven.
- The reference pilot proves advisory-mode observation and recommendation, not active control of a live customer deployment.
