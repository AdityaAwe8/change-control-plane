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
- `make reference-pilot-validate`
- `make proof-live-validate`
- a secret-safety scan across the generated release report, its supporting logs, and any preserved proof artifacts

## Proof Classes

The report distinguishes:

- `local`: checks rerun directly from this repository
- `harness`: repo-managed contract or provider-harness proof
- `artifact`: saved proof bundles revalidated without rerunning the source environment
- `operator-proof`: whether a saved external proof bundle is strong enough to count as real hosted/customer evidence

## Default Blocking Rules

By default the ship gate fails when:

- a required local or harness command fails
- the saved reference-pilot proof artifact is missing or invalid
- the saved live-proof artifact is missing or invalid
- the saved live-proof artifact is only `hosted_like`

That last case is intentional: hosted-like proof is valuable, but it is not the same as preserved operator-run hosted/customer evidence.

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

## Recommended Operator Flow

1. Capture or refresh the local reference-pilot proof.
2. Capture or refresh a real external `live-proof-verify` report for `customer_environment` or `hosted_saas`.
3. Run `make release-readiness`.
4. Review `.tmp/release-readiness/release-readiness-report.md` and preserve it alongside the underlying proof artifacts.
