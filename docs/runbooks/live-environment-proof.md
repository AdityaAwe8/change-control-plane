# Live Environment Proof Runbook

This runbook covers the new external-facing proof track for ChangeControlPlane.

It is intentionally narrower than the local reference pilot:

- it does not assume a local fixture stack
- it does not assume a local `k3s` cluster
- it does not inject a synthetic SCM webhook or run a full advisory rollout flow

Instead, it captures the strongest honest external proof this repository can automate today:

- hosted or customer-like SCM onboarding and repository discovery
- automatic SCM webhook registration
- customer-like Kubernetes workload observation
- customer-like Prometheus signal collection
- repository/resource mapping and coverage evidence

## Command

Run:

```bash
make proof-live-verify
```

This wraps:

```bash
./scripts/live-proof-verify.sh
```

and writes:

```text
.tmp/live-proof/live-proof-report.json
```

Validate an existing saved report:

```bash
make proof-live-validate
```

This wraps:

```bash
./scripts/live-proof-validate.sh
```

## Required Configuration

Common:

- `CCP_LIVE_PROOF_API_BASE_URL`
- `CCP_LIVE_PROOF_ADMIN_EMAIL`
- `CCP_LIVE_PROOF_ADMIN_PASSWORD`

Kubernetes:

- `CCP_LIVE_PROOF_KUBE_API_BASE_URL`
- `CCP_LIVE_PROOF_KUBE_NAMESPACE`
- `CCP_LIVE_PROOF_KUBE_DEPLOYMENT`
- optional `CCP_LIVE_PROOF_KUBE_STATUS_PATH`
- optional `CCP_LIVE_PROOF_KUBE_TOKEN_ENV`

Prometheus:

- `CCP_LIVE_PROOF_PROMETHEUS_BASE_URL`
- `CCP_LIVE_PROOF_PROMETHEUS_QUERY`
- optional `CCP_LIVE_PROOF_PROMETHEUS_TOKEN_ENV`

GitLab mode:

- `CCP_LIVE_PROOF_SCM_KIND=gitlab`
- `CCP_LIVE_PROOF_GITLAB_BASE_URL`
- `CCP_LIVE_PROOF_GITLAB_GROUP`
- `CCP_LIVE_PROOF_GITLAB_TOKEN_ENV`
- `CCP_LIVE_PROOF_GITLAB_WEBHOOK_SECRET_ENV`

GitHub mode:

- `CCP_LIVE_PROOF_SCM_KIND=github`
- `CCP_LIVE_PROOF_GITHUB_API_BASE_URL`
- `CCP_LIVE_PROOF_GITHUB_WEB_BASE_URL`
- `CCP_LIVE_PROOF_GITHUB_OWNER`
- `CCP_LIVE_PROOF_GITHUB_APP_ID`
- `CCP_LIVE_PROOF_GITHUB_APP_SLUG`
- `CCP_LIVE_PROOF_GITHUB_PRIVATE_KEY_ENV`
- `CCP_LIVE_PROOF_GITHUB_WEBHOOK_SECRET_ENV`
- `CCP_LIVE_PROOF_GITHUB_INSTALLATION_ID`

## What The Runner Proves

When successful, the report proves:

- the control-plane API accepted authenticated operator access
- org/project/team/service/environment scope was created or reused successfully
- the SCM integration was created or reused and tested successfully
- automatic webhook registration succeeded for the selected SCM provider
- repository discovery succeeded and at least one repository was mapped
- the Kubernetes integration tested and synced successfully
- at least one Kubernetes workload resource was discovered and mapped
- the Prometheus integration tested and synced successfully
- at least one Prometheus signal target was discovered and mapped
- integration coverage summary is available from the API

In GitHub mode, the report also proves:

- GitHub App onboarding start returned a signed authorize URL
- the callback path accepted the provided installation id and persisted it

## What This Does Not Yet Prove

- a full external end-to-end rollout execution with live SCM webhook ingest and live customer rollback recommendation
- production network, routing, RBAC, and auth behavior for a real customer environment
- long-running soak or failure-recovery behavior in a real business environment
- GitLab OAuth or GitLab App-style onboarding

## Evidence

Primary artifact:

- `.tmp/live-proof/live-proof-report.json`

Useful secondary evidence:

- control-plane audit and status-event queries
- browser inspection of the integrations, discovery, and coverage pages
- provider-side webhook configuration screenshots or exported settings

Validation note:

- `live-proof-verify` now validates the generated report structure before writing it
- `live-proof-validate` rechecks a saved report without contacting external systems

## Release Gate Integration

`make release-readiness` now consumes the saved `.tmp/live-proof/live-proof-report.json` artifact through `make proof-live-validate`.

Important truth boundary:

- `environment_class=hosted_like` remains useful harness evidence, but it does not close the operator-run hosted/customer proof gap
- only saved `customer_environment` or `hosted_saas` live-proof artifacts satisfy that part of the ship gate without the dry-run override

So the ship gate validates the report and classifies its proof level, but it still does not execute a real hosted/customer run for you.
