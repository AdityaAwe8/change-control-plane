# Live Environment Proof

This milestone now has a stronger local-reference proof story than the earlier harness-only state. The product is still not claiming production proof, but it now has one reproducible reference pilot environment that exercises real cluster-backed observation and real Prometheus collection.

## Reference Pilot Scope

The repository now includes a reproducible reference pilot environment with:

- a local `k3s` cluster
- a real Prometheus deployment scraping a sample workload
- a local GitLab fixture for repository discovery and webhook ingest
- the control-plane API running against dedicated pilot dependencies
- an end-to-end verification command that proves advisory-only rollout behavior

See:

- `docs/architecture/reference-pilot-proof.md`
- `docs/runbooks/reference-pilot-environment.md`
- `docs/runbooks/reference-pilot-validation.md`
- `docs/runbooks/live-environment-proof.md`
- `docs/testing/live-environment-verification-plan.md`

## Kubernetes

- Provider behavior is now repo-proven against changing upstream workload state.
- The product is also now local-cluster proven against the reference `k3s` environment for workload discovery, rollout observation, sync evidence, and mapped workload persistence.
- Inventory-driven discovery marks disappeared workloads as `missing`.
- Coverage summaries stop counting missing workloads as active runtime coverage.

Still missing:

- production cluster auth and controller-style reconciliation
- checked-in proof from a real hosted or customer Kubernetes environment, even though the repository now includes a reusable external proof runner

## Prometheus

- Query collection is repo-proven with configured window and step semantics.
- The product is also now local-metrics proven against the in-cluster reference Prometheus deployment.
- Empty query results surface as warnings instead of silently reading as healthy zeros.
- Repeated collection proof covers changing upstream signal behavior and structured failures.

Still missing:

- checked-in proof from a real hosted or customer Prometheus environment, even though the repository now includes a reusable external proof runner
- broader signal discovery beyond configured query templates

## Advisory Operation

The reference pilot now proves a real advisory-only flow:

- SCM webhook change exists
- repository is mapped
- workload is observed
- metrics are collected
- runtime verification records an advisory-only rollback recommendation
- external backend mutation is suppressed and recorded explicitly

That is a meaningful pilot-proof milestone, but it is still not equivalent to production approval for live customer control.

## External Proof Track

The repository now also includes a reusable external-facing proof runner:

- `make proof-live-verify`
- `make proof-live-validate`
- `./scripts/live-proof-verify.sh`
- `./scripts/live-proof-validate.sh`
- `cmd/live-proof-verify`

This track is intended for hosted GitHub or GitLab plus customer-like Kubernetes and Prometheus environments. It captures onboarding, webhook registration, repository discovery, runtime resource mapping, and coverage evidence into `.tmp/live-proof/live-proof-report.json`.

The repository can now also revalidate a saved proof artifact locally via `make proof-live-validate`, which checks that the report still satisfies the expected evidence shape without contacting external systems.

It narrows the “no external proof path exists” gap, but it does not itself mean the repository now contains checked-in proof from a real hosted customer environment.
