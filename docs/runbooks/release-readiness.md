# Release Readiness Runbook

This runbook describes the operator-facing ship gate for the strongest current ChangeControlPlane proof that can be aggregated locally.

## Command

Run:

```bash
make release-readiness
```

This writes:

```text
.tmp/release-readiness/release-readiness-report.md
```

Per-check command output is written to:

```text
.tmp/release-readiness/logs/*.log
```

and now also refreshes:

```text
.tmp/live-proof/live-proof-preflight.json
.tmp/live-proof/live-proof-operator-checklist.md
```

All of those paths live under `.tmp/`, which is intentionally gitignored. Preserve the generated artifacts separately when they are used as release evidence; do not stage them into the repository.

When `GOCACHE` or `GOTMPDIR` are unset, the gate now pins them to repo-local `.tmp/go-build` and `.tmp/go-tmp` paths so Go-based checks can still run in sandboxed or locked-down workstation environments without relying on `~/Library/Caches/go-build`.

## What The Gate Checks

The ship gate reruns or validates:

- `go test ./cmd/...`
- `go test ./internal/app/...`
- `go test ./internal/storage/...`
- `go test ./internal/integrations/...`
- `go test ./internal/events/...`
- `make web-typecheck`
- `make web-build`
- `make proof-contract`
- `make proof-harness`
- `make proof-live-preflight`
- `make reference-pilot-validate`
- `make proof-live-validate`
- a secret-safety scan across the generated release report, its supporting logs, and any preserved proof artifacts

## Reproducing The Full Local Matrix

For a clean engineer workstation or CI-like local runner, use:

```bash
git diff --check
go test ./...
python3 -m unittest discover -s python/tests -v
cd web && pnpm typecheck
cd web && pnpm build
cd web && pnpm test:e2e
make proof-contract
make proof-harness
make proof-postgres
make verify
make release-readiness
```

PostgreSQL-backed tests need a reachable PostgreSQL instance. The reproducible target is:

```bash
make proof-postgres
```

With the local Docker stack, the default DSN points at `localhost:15432`:

```bash
make compose-up
make proof-postgres
```

If Docker is unavailable, use a local PostgreSQL database instead:

```bash
createdb change_control_plane_test
CCP_TEST_DB_DSN='postgres:///change_control_plane_test?host=/tmp&sslmode=disable' make proof-postgres
```

These DSN examples are test-only environment values. The target does not persist DSNs; it only passes `CCP_TEST_DB_DSN` to the storage and app test processes. The app runtime database tests also create temporary databases for structured validation proof when the PostgreSQL role permits it, and otherwise skip with the underlying connection or permission error.

The default CI workflow runs the non-artifact local checks, browser e2e, provider harness proof, OpenAPI proof, and `make proof-postgres` with a GitHub Actions PostgreSQL service. It does not run `make release-readiness` by default because that gate is intentionally artifact-sensitive: it requires preserved reference-pilot and external live-proof reports, and it must block on missing or `hosted_like`-only external proof.

## Proof Classes

The report distinguishes:

- `local`: checks rerun directly from this repository
- `harness`: repo-managed contract or provider-harness proof
- `artifact`: saved proof bundles revalidated without rerunning the source environment
- `operator-proof`: whether a saved external proof bundle is strong enough to count as real hosted/customer evidence

When the external proof artifact is missing, the gate now points at the generated live-proof checklist instead of leaving only a vague missing-file warning.

## Default Blocking Rules

By default the ship gate fails when:

- a required local or harness command fails
- the saved reference-pilot proof artifact is missing or invalid
- the saved live-proof artifact is missing or invalid
- the saved live-proof artifact is only `hosted_like`

That last case is intentional: hosted-like proof is valuable, but it is not the same as preserved operator-run hosted/customer evidence.

The new preflight/checklist output is informative, not sufficient by itself. It narrows the remaining operator work; it does not replace the real external artifact.

## Dry-Run Override

To rehearse the gate before proof artifacts have been captured, run:

```bash
CCP_RELEASE_ALLOW_PROOF_GAPS=true make release-readiness
```

This downgrades missing proof artifacts or hosted-like-only external proof to warnings.

Use this only for local rehearsal. It does not turn missing or harness-only proof into real customer-environment evidence.

## What This Still Does Not Prove

The ship gate does not itself:

- execute a real hosted/customer `live-proof-verify` run
- replace browser interaction proof already covered by dedicated Playwright and CI flows
- act as a universal runtime metadata scrubber across every status event or arbitrary external log source
- replace operator judgment on intentionally limited subsystems such as narrow deterministic policy scope or incomplete enterprise IAM breadth

## Operator Checklist

If the live-proof artifact is missing, read:

```text
.tmp/live-proof/live-proof-operator-checklist.md
```

That checklist is regenerated on every `make release-readiness` run and is designed to answer:

- which env vars are still missing
- which provider secrets are referenced but not actually loaded
- whether the current selected SCM path is GitHub or GitLab
- what exact callback and webhook URL patterns still need to exist
- whether the current API base still needs public DNS, ingress, or a trusted tunnel before hosted SCM proof can succeed
- what Kubernetes cluster, namespace, and deployment access is still required
- what Prometheus endpoint, auth, and query inputs are still required

## Recommended Operator Flow

1. Capture or refresh the local reference-pilot proof.
2. Generate or refresh `.tmp/live-proof/live-proof-operator-checklist.md` with `make proof-live-preflight` and close its missing-input list.
3. Capture or refresh a real external `live-proof-verify` report for `customer_environment` or `hosted_saas`.
4. Run `make release-readiness`.
5. Review `.tmp/release-readiness/release-readiness-report.md` and `.tmp/release-readiness/logs/*.log`, then preserve them alongside the underlying proof artifacts.
