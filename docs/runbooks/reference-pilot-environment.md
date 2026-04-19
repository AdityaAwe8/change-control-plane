# Reference Pilot Environment Runbook

This runbook stands up the local reference pilot used to prove the control plane against a real local cluster, a real local Prometheus instance, and a real advisory-only rollout flow.

## Prerequisites

- Docker with enough memory to run a small `k3s` cluster and Prometheus
- `kubectl`
- Go 1.26 or newer
- free local ports for:
  - `38080` control-plane API
  - `39480` local GitLab fixture
  - `18091` `kubectl proxy`
  - `18092` workload admin / port-forward
  - `19090` Prometheus port-forward
  - `25432` pilot PostgreSQL
  - `26379` pilot Redis
  - `24222` pilot NATS

## What Gets Started

`make reference-pilot-up` now provisions:

- PostgreSQL, Redis, and NATS from `deploy/reference-pilot/docker-compose.yml`
- a local `k3s` cluster named `ccp-reference-pilot-k3s`
- the sample `checkout` workload in namespace `ccp-pilot`
- an in-cluster Prometheus scraping that workload
- a local GitLab fixture server
- the control-plane API bound to the pilot dependency stack
- `kubectl proxy` and two `kubectl port-forward` processes for the workload and Prometheus

## Start The Environment

```bash
make reference-pilot-up
source .tmp/reference-pilot/reference-pilot.env
```

The generated environment file exports the base URLs and secret references used by the proof flow.

Important values:

- `CCP_REFERENCE_PILOT_API_BASE_URL`
- `CCP_REFERENCE_PILOT_GITLAB_BASE_URL`
- `CCP_REFERENCE_PILOT_KUBE_API_BASE_URL`
- `CCP_REFERENCE_PILOT_WORKLOAD_ADMIN_URL`
- `CCP_REFERENCE_PILOT_PROMETHEUS_BASE_URL`

## Run The Proof Flow

```bash
make reference-pilot-verify
```

This runs `go run ./cmd/reference-pilot-verify --report .tmp/reference-pilot/reference-pilot-report.json`.

The verification command:

- signs in as `admin@changecontrolplane.local`
- configures the pilot GitLab, Kubernetes, and Prometheus integrations
- tests and syncs those integrations
- maps the discovered repository, workload, and signal target
- degrades the sample workload through its admin endpoint
- posts a GitLab merge-request style webhook into the control plane
- creates and advances a rollout execution in advisory mode
- confirms that runtime verification records an advisory-only rollback recommendation instead of mutating the live backend

## Review The Proof Artifact

The proof report is written to:

```bash
.tmp/reference-pilot/reference-pilot-report.json
```

Key evidence to inspect:

- `gitlab_integration.connection_health`
- `kubernetes_integration.connection_health`
- `prometheus_integration.connection_health`
- `repository.status`
- `kubernetes_resource.status`
- `prometheus_resource.health`
- `execution_detail.runtime_summary.advisory_only`
- `execution_detail.runtime_summary.latest_decision`
- `status_event_count`
- `timeline_event_count`

## Optional Web Review

You can also inspect the pilot state in the web app:

```bash
make web-install
make web-dev
```

Then sign in with:

- email: `admin@changecontrolplane.local`
- password: `ChangeMe123!`

Recommended checks:

- `#/integrations`: GitLab, Kubernetes, and Prometheus should show healthy connection and fresh sync state
- `#/deployments`: the pilot execution should show advisory-only recommendation language, not executed rollback language
- `#/service` and `#/environment`: the mapped service/environment should match the pilot report

## Teardown

```bash
make reference-pilot-down
```

This removes the pilot `k3s` container and the dedicated pilot dependency stack.

## Troubleshooting

If bootstrap fails:

- make sure Docker can run privileged containers
- check whether the required ports are already in use
- inspect `.tmp/reference-pilot/*.log`

If verification fails:

- rerun `source .tmp/reference-pilot/reference-pilot.env`
- confirm the API is reachable at `http://127.0.0.1:38080/readyz`
- confirm the GitLab fixture is reachable at `http://127.0.0.1:39480/readyz`
- confirm Prometheus is reachable at `http://127.0.0.1:19090/-/ready`
- confirm the workload admin endpoint is reachable at `http://127.0.0.1:18092/healthz`

## Honest Limits

This environment is strong enough for a careful reference pilot, but it is still not a production environment:

- SCM proof uses a local GitLab fixture, not hosted GitLab or GitHub
- Kubernetes proof uses a local `k3s` cluster and `kubectl proxy`
- Prometheus proof uses a local in-cluster deployment and fixed query templates
- advisory mode remains the safe default; the pilot does not prove customer-facing active control
