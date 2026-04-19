# Local Development Runbook

## Start Dependencies

```bash
make compose-up
```

This starts PostgreSQL on `localhost:15432`, Redis on `localhost:16379`, and NATS on `localhost:14222`.
The command waits for PostgreSQL and Redis to report healthy before it returns, so `make migrate` and `make run-api` can start immediately afterward.

## Start the API

```bash
make migrate
make run-api
```

These targets automatically use the local dependency ports from `make compose-up` unless you override `CCP_DB_DSN`, `CCP_REDIS_ADDR`, or `CCP_NATS_URL`.

## Start the Worker

Issue a worker token first:

```bash
go run ./cmd/cli auth login --email owner@acme.local --name "Acme Owner" --organization-name Acme --organization-slug acme
go run ./cmd/cli service-account create --organization <org_id> --name worker-bot --role org_member
go run ./cmd/cli token issue --service-account <service_account_id> --name worker
export CCP_WORKER_TOKEN=<issued_token>
export CCP_WORKER_ORGANIZATION_ID=<org_id>
```

Then start the worker:

```bash
make run-worker
```

## Start the Web App

```bash
make web-install
make web-dev
```

If the web console is served from a different origin than the API, keep `CCP_ALLOWED_ORIGINS` aligned with the browser origin. Development defaults now cover the common Vite ports.

## Optional Full Docker Stack

```bash
make compose-up-full
```

This rebuilds the API and worker images from the current repository state, then starts the API in Docker on `http://localhost:28080` by default and the worker in Docker alongside the dependency services.

If that host port is already occupied on your machine, override it:

```bash
CCP_DOCKER_API_HOST_PORT=38080 make compose-up-full
```

## Verify the Baseline

```bash
make verify
make web-e2e
make smoke
```

## Reference Pilot Environment

For the local-cluster/local-metrics reference pilot flow, use:

```bash
make reference-pilot-up
source .tmp/reference-pilot/reference-pilot.env
make reference-pilot-verify
make reference-pilot-validate
```

This stands up a dedicated pilot PostgreSQL/Redis/NATS stack, a local `k3s` cluster, the sample workload, an in-cluster Prometheus deployment, the local GitLab fixture, and the control-plane API on pilot ports.

When finished:

```bash
make reference-pilot-down
```

See `docs/runbooks/reference-pilot-environment.md` and `docs/runbooks/reference-pilot-validation.md` for the full pilot runbook and checklist.

## External Live Proof Runner

Use the hardened external proof runner when you want evidence beyond the local reference pilot:

```bash
make proof-live-verify \
  CCP_LIVE_PROOF_ENVIRONMENT_CLASS=hosted_like \
  CCP_LIVE_PROOF_SCM_KIND=gitlab
```

The runner now supports these proof classes:

- `hosted_like`: realistic harness, proxy, or staged environment validation
- `customer_environment`: operator-run proof against customer-owned infrastructure
- `hosted_saas`: operator-run proof against real hosted SaaS endpoints such as GitHub Cloud or GitLab SaaS

The saved report is written to `.tmp/live-proof/live-proof-report.json` and now includes:

- a declared environment class
- secret-safe config summary
- proof checks with warnings and hints
- an evidence summary for SCM discovery, webhook state, Kubernetes discovery, Prometheus signal capture, and mapping coverage

Validate a saved proof bundle with:

```bash
make proof-live-validate
```

This runner is now much stricter about misconfiguration. Missing secret envs, invalid URLs, unsupported proof classes, and misleading hosted-SaaS/local-endpoint combinations fail before deeper execution.

## Release Readiness Ship Gate

When you want one truthful operator-facing gate across the strongest current local, harness, and preserved-proof checks, run:

```bash
make release-readiness
```

The ship gate writes:

```text
.tmp/release-readiness/release-readiness-report.md
```

It currently reruns:

- Go command, app, storage, integration, and event-bus tests
- web typecheck and build
- `make proof-contract`
- `make proof-harness`
- saved-report validation for the reference pilot and external live proof bundles

By default it blocks when:

- `.tmp/reference-pilot/reference-pilot-report.json` is missing or invalid
- `.tmp/live-proof/live-proof-report.json` is missing or invalid
- the saved live proof report is only `hosted_like` instead of `customer_environment` or `hosted_saas`

For a local dry run before those artifacts have been captured, use:

```bash
CCP_RELEASE_ALLOW_PROOF_GAPS=true make release-readiness
```

This override downgrades missing proof artifacts to warnings so you can rehearse the gate locally, but it does not create real hosted/customer proof.
