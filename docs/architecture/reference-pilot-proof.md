# Reference Pilot Proof

This milestone adds a reproducible reference pilot environment that proves one end-to-end advisory flow against a real local cluster and a real local Prometheus instance, without overclaiming production readiness.

The reference pilot is intentionally distinct from the external `live-proof-verify` path:

- `reference-pilot`: local-reference proof against repo-owned infrastructure
- `live-proof-verify --environment-class hosted_like`: hosted-like harness proof against realistic fake or proxied environments
- `live-proof-verify --environment-class customer_environment`: operator-declared proof against a real customer-owned environment
- `live-proof-verify --environment-class hosted_saas`: operator-declared proof against real hosted SaaS endpoints such as GitHub Cloud or GitLab SaaS

## What The Reference Pilot Includes

- a dedicated local PostgreSQL, Redis, and NATS stack from `deploy/reference-pilot/docker-compose.yml`
- a local `k3s` cluster started by `scripts/reference-pilot-up.sh`
- a real Kubernetes deployment and service for the sample `checkout` workload from `deploy/reference-pilot/k8s/reference-pilot.yaml`
- a real Prometheus deployment in the same cluster scraping the sample workload
- a local GitLab fixture server from `cmd/reference-pilot-gitlab`
- the control-plane API running against the pilot dependency stack
- a verification command, `cmd/reference-pilot-verify`, that configures integrations, maps discovered resources, ingests a real merge-request style webhook, creates a rollout execution, and checks advisory-only behavior

## What Is Proven

The reference pilot now proves a real local flow for:

- GitLab repository discovery and webhook-backed change ingest through a local fixture
- automatic GitLab webhook registration and health reconciliation for the pilot scope
- Kubernetes workload discovery and rollout observation against a real local cluster
- Prometheus query-range collection against a real local metrics server
- discovered repository, workload, and signal-target mapping into the control-plane model
- advisory-mode rollout behavior where runtime verification recommends rollback but does not mutate the live backend
- preserved audit events, status events, signal snapshots, verification results, and rollout timeline evidence

The canonical proof artifact is written to:

- `.tmp/reference-pilot/reference-pilot-report.json`

## What The Proof Flow Does

The proof command currently exercises this path:

1. signs in as the seeded admin user
2. ensures the pilot organization, project, team, service, and environment exist
3. configures GitLab, Kubernetes, and Prometheus integrations in advisory mode
4. tests and syncs those integrations
5. maps the discovered GitLab repository, Kubernetes workload, and Prometheus signal target
6. updates the sample workload into a degraded state through its admin endpoint
7. posts a GitLab merge-request webhook to the control-plane API
8. waits for the resulting `ChangeSet`
9. creates a risk assessment, rollout plan, and rollout execution
10. starts the rollout and reconciles until runtime verification records an advisory-only rollback recommendation

## What This Does Not Yet Prove

This milestone is intentionally not a production-readiness claim.

Still not proven:

- hosted GitHub or GitLab enterprise environments in the reference pilot flow
- live-cluster Kubernetes auth patterns such as kubeconfig distribution, cluster certificates, or controller-runtime execution
- live hosted Prometheus routing or enterprise auth models
- browser-only pilot operation as the canonical proof path
- active-control rollout execution against a customer environment
- long-running soak or failure-injection proof over many hours or days

For those cases, use the hardened `live-proof-verify` path and preserve its saved report as the operator evidence bundle.

## Current Confidence Level

- `GitLab`: local-reference proven through a real control-plane webhook/configuration path and a local fixture server
- `Kubernetes`: local-cluster proven for discovery, observation, and advisory-only rollout suppression
- `Prometheus`: local-metrics proven for query windows, changing values, and persisted signal snapshots
- `Advisory mode`: local-reference proven for recommendation-only rollback behavior with explicit suppression evidence

Use `docs/architecture/reference-pilot-proof-status.md` for the honest classification table and `docs/runbooks/reference-pilot-environment.md` for setup steps.
