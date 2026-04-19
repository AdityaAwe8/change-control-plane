# Live Environment Verification Plan

This plan captures the minimum external proof set for the hosted/customer-like verification track.

## Goal

Demonstrate that ChangeControlPlane can attach to real external-style SCM and runtime endpoints without depending on the local reference-pilot fixture stack.

## Core Command

Run:

```bash
make proof-live-verify
```

Then validate the saved artifact:

```bash
make proof-live-validate
```

## Success Criteria

The live-environment proof track is considered successful when:

- `.tmp/live-proof/live-proof-report.json` exists
- `make proof-live-validate` succeeds against the saved report
- the report `profile` is `live`
- the report `scm_kind` matches the requested provider
- the SCM integration test result is healthy
- the SCM sync result includes at least one discovered repository
- the SCM webhook registration result is present
- the mapped repository status is `mapped`
- the Kubernetes integration test and sync both succeed
- the mapped Kubernetes resource is `kubernetes_workload`
- the Prometheus integration test and sync both succeed
- the mapped Prometheus resource is `prometheus_signal_target`
- the coverage summary is present

GitHub-specific additional criteria:

- `github_onboarding_start` is present
- `github_onboarding_completion` is present

## Honest Classification

This plan proves:

- external-facing operator workflow readiness
- hosted-like SCM onboarding and webhook registration behavior
- customer-like Kubernetes/Prometheus attachment and mapping behavior

This plan does not by itself prove:

- production approval for customer environments
- long-running operational resilience
- full live rollout mutation in a real customer environment
