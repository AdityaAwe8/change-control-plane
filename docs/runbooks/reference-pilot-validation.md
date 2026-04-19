# Reference Pilot Validation Checklist

Use this checklist after bringing up the reference pilot environment.

## 1. Bootstrap

Run:

```bash
make reference-pilot-up
source .tmp/reference-pilot/reference-pilot.env
```

Expected outcome:

- the API is reachable on `http://127.0.0.1:38080`
- the local GitLab fixture is reachable on `http://127.0.0.1:39480`
- the workload is reachable on `http://127.0.0.1:18092`
- Prometheus is reachable on `http://127.0.0.1:19090`

## 2. End-to-End Advisory Proof

Run:

```bash
make reference-pilot-verify
make reference-pilot-validate
```

Expected outcome:

- the command exits successfully
- `.tmp/reference-pilot/reference-pilot-report.json` exists
- the saved report revalidates successfully without rerunning the pilot flow

## 3. Report Assertions

Inspect the report and confirm:

- `gitlab_integration.status` is `connected`
- `kubernetes_integration.status` is `connected`
- `prometheus_integration.status` is `connected`
- `repository.provider` is `gitlab`
- `repository.status` is `mapped`
- `kubernetes_resource.resource_type` is `kubernetes_workload`
- `kubernetes_resource.status` is `mapped`
- `prometheus_resource.resource_type` is `prometheus_signal_target`
- `prometheus_resource.health` is `critical`
- `change_set.metadata.scm_provider` is `gitlab`
- `execution_detail.runtime_summary.advisory_only` is `true`
- `execution_detail.runtime_summary.latest_decision` is `advisory_rollback`
- `execution_detail.runtime_summary.last_action_disposition` is `suppressed`
- `status_event_count` is greater than `0`
- `timeline_event_count` is greater than `0`

## 4. Operator Review

Optional browser review:

1. start the web app with `make web-dev`
2. sign in as `admin@changecontrolplane.local` with `ChangeMe123!`
3. confirm integrations show healthy and fresh pilot state
4. confirm the pilot rollout renders recommendation-only wording, not executed rollback wording
5. confirm stale-session handling returns the UI to sign-in instead of failing silently

## 5. Cleanup

Run:

```bash
make reference-pilot-down
```

Expected outcome:

- the pilot `k3s` container is gone
- the pilot compose services are stopped

## Validation Scope

This checklist proves:

- local-cluster Kubernetes observation
- local-metrics Prometheus collection
- local GitLab fixture webhook and repository discovery
- advisory-only runtime recommendation and evidence persistence

This checklist does not prove:

- production hosted SCM behavior
- production Kubernetes auth and networking
- production Prometheus auth and routing
- active-control rollout mutation in a customer environment
