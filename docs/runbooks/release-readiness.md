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

and now also refreshes:

```text
.tmp/live-proof/live-proof-preflight.json
.tmp/live-proof/live-proof-operator-checklist.md
```

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
5. Review `.tmp/release-readiness/release-readiness-report.md` and preserve it alongside the underlying proof artifacts.
