# Pilot Runtime Depth

This milestone pushes the product from a manually exercised pilot toward a continuously operating advisory control plane.

## What Is New

1. Integrations now carry schedule and freshness state.
2. The worker can run scheduled syncs and retries in addition to rollout reconcile.
3. Kubernetes and Prometheus integrations now persist first-class discovered runtime resources.
4. Coverage summaries now expose what is connected, stale, mapped, and still missing.
5. The status dashboard now uses server-backed search and pagination.

## Operating Model

The runtime-depth path is intentionally modest:

- one worker loop
- org-scoped integration instances
- claim-and-run scheduled sync
- persisted sync-run evidence
- advisory-safe recurring observation

This is not intended to be a distributed scheduler or a controller mesh. It is intended to be enough for a careful business pilot to leave the system running and inspect whether connected systems continue to refresh.

## What The Product Can Do Now

- keep Kubernetes and Prometheus integrations fresh on a configured cadence
- classify freshness as fresh, scheduled, manual-only, error, stale, or stale-after-error
- persist scheduled, retry, manual, and webhook-triggered sync runs
- show last attempted, last successful, last failed, and next due times
- surface discovered workloads and signal targets for mapping review
- show coverage gaps and stale integrations in operator-facing pages

## What It Still Does Not Do

- provide production-grade distributed scheduling
- prove live-cluster or live-metrics correctness
- infer broad ownership and dependency topology automatically
- guarantee full environment freshness without a healthy configured integration
