# Pilot Runtime Depth Status

This audit is intentionally skeptical. A runtime-depth area is not considered complete because a button, route, or provider seam exists.

## Current Classification

| Area | State | Reality |
| --- | --- | --- |
| Integration test/sync model | partially scheduled | Manual test and sync still exist, but integrations now also persist schedule state, freshness, next due time, and scheduled retry behavior. |
| Worker scheduling capability | continuous and verified | The worker now scans due integrations, claims them, runs scheduled syncs, and records success or failure in automated tests. |
| Sync-run persistence | continuous and verified | Sync runs now persist trigger, scheduled time, error class, summary details, and completion state for manual, scheduled, retry, and webhook-triggered flows. |
| Freshness and staleness | continuous and verified | Integrations now expose last attempted, last successful, last failed, freshness state, sync lag, and stale indicators. |
| Kubernetes recurring sync | partially scheduled | Scheduled sync is real and provider-backed, but it is still repository-proven rather than live-cluster-proven. |
| Prometheus recurring collection | partially scheduled | Scheduled collection is real and persists signal-backed discovery evidence, but it is still repository-proven rather than live-metrics-proven. |
| GitHub refresh path | partially scheduled | Webhook ingest remains primary; scheduled reconciliation exists through the common sync runner and now supports both PAT and GitHub App installation-token auth. It is still not a full enterprise OAuth or marketplace-grade install model. |
| Dashboard/runtime freshness visibility | partially implemented | The web app now shows schedule, freshness, coverage, and stale warnings, but the product still needs more polished summary storytelling for broader pilots. |
| Coverage summaries | partially implemented | Coverage summaries now exist for integrations, repositories, workload discovery, and signal targets, but ownership/dependency coverage remains shallow. |
| Documentation accuracy | partially implemented | Runtime-depth docs now describe scheduled sync and server-backed search honestly, but older architecture docs still reference more manual behavior than the product now has. |

## Honest Summary

The product no longer depends entirely on operators manually pressing sync just to keep live-integration data current. It now has a real recurring sync layer, persisted freshness state, scheduled retry behavior, and worker-backed execution.

That said, runtime depth is still not equivalent to live-environment proof:

- Kubernetes remains near-real and HTTP-backed rather than live-cluster-proven.
- Prometheus remains near-real and query-backed rather than production-metrics-proven.
- GitHub now supports both PAT and GitHub App installation-token flows, but it is still not a full OAuth or marketplace-grade install experience.
- Coverage is still strong enough for a careful pilot, not a claim of broad autonomous environment comprehension.
