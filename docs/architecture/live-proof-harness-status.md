# Live-Proof Harness Status

This document records what is proven through realistic harnesses instead of true live-environment tests.

The harness suite can now be invoked explicitly with `make proof-harness`, and CI runs the same proof track as a dedicated job.

| Path | Status | Reality |
| --- | --- | --- |
| GitHub App installation token flow | harness-proven | A fake GitHub API now proves app-JWT signing, installation-token exchange, onboarding callback persistence, placeholder webhook-registration state, and instance-scoped sync. |
| GitHub App webhook registration / repair | harness-proven | Realistic fake GitHub org-hook endpoints now prove post-install automatic webhook registration and subsequent repair/update behavior through the application HTTP routes. |
| GitHub webhook ingest | repo-proven and harness-proven | Signed webhook validation, dedupe, mapped change ingestion, and sync-run persistence are proven in repository tests. |
| GitLab token onboarding and webhook repair | harness-proven | Realistic fake GitLab group-hook endpoints now prove token-based onboarding, automatic webhook registration on integration update, and subsequent repair/update behavior through the application HTTP routes. |
| Kubernetes status sync | harness-proven | Repeated sync against changing upstream deployment state and provider-failure paths are now covered. |
| Kubernetes auth/path request shaping | harness-proven | Application integration tests now prove bearer-token headers and custom status paths against realistic fake upstreams. |
| Kubernetes action shaping | harness-proven | Pause and rollback request shaping are covered through fake HTTP providers. Advisory suppression remains separately proven in app tests. |
| Prometheus repeated collection | harness-proven | Repeated query windows, degraded transitions, empty results, and upstream failures are all proven against fake Prometheus responses. |
| Prometheus auth/path request shaping | harness-proven | Application integration tests now prove bearer-token headers and custom query paths against realistic fake upstreams. |
| Scheduler contention | repo-proven | Duplicate-claim prevention is proven in concurrent worker-style tests, but not under real multi-process crash/restart scenarios. |

## Honest Boundary

Harness proof is stronger than simple unit or descriptor proof. It is still weaker than:

- a real GitHub App installed on a real org
- a real Kubernetes cluster with real auth and object semantics
- a real Prometheus backend with production routing and auth
