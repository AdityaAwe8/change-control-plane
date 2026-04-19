# Verification Matrix

This matrix tracks the real proof level of each major subsystem.

The detailed route-by-route and control-by-control inventory now lives in [full-verification-matrix.md](/Users/aditya/Documents/ChangeControlPlane/docs/testing/full-verification-matrix.md).

| Subsystem | Implementation | Persistence | Auth | Audit | Tests | Web | CLI | Docs | Known Gaps |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Backend API | strong | strong | medium | medium | medium | medium | strong | medium | OpenAPI is materially closer to runtime truth now, but older CRUD routes still lag full response-envelope modeling. |
| Worker | strong | via API | medium | via API | strong | n/a | n/a | medium | Claiming is adequate for the current worker model, but not yet a distributed lease system. |
| Storage | strong | strong | n/a | n/a | medium | n/a | n/a | medium | Needs continued schema/contract discipline. |
| Auth / RBAC | medium | strong | medium | medium | medium | medium | medium | medium | No enterprise auth; edge-case tests can improve. |
| Service accounts / tokens | medium | strong | medium | medium | medium | medium | medium | medium | No policy-driven expiry or usage analytics. |
| System graph | medium | medium | medium | medium | medium | weak | weak | medium | Query depth and adapter realism are limited. |
| Change intelligence | strong | strong | medium | medium | strong | medium | medium | medium | Python intelligence is real, but still deterministic and not model-trained. |
| Rollout planning | medium | strong | medium | medium | medium | medium | medium | medium | Still advisory-heavy and not connected to external deploy systems. |
| Rollout execution | strong | strong | medium | strong | strong | medium | strong | medium | Automated rollback follow-through is real; Kubernetes is near-real but not cluster-proven. |
| Verification | strong | strong | medium | strong | strong | medium | strong | medium | Prometheus polling is now near-real, but continuous telemetry ingestion is still deferred. |
| Rollback policy | strong | strong | medium | strong | strong | medium | medium | strong | Scope resolution is specificity-based rather than a full inheritance merge engine. |
| Status history | strong | strong | medium | strong | strong | medium | medium | strong | Search is practical and indexed, but still not a full observability-grade analytics backend. |
| Integrations | strong | strong | medium | medium | strong | medium | medium | medium | GitHub now has App-style onboarding and multi-instance scoping in addition to the PAT path; Kubernetes and Prometheus remain harness-proven rather than live-environment-proven, and discovery is still shallow. |
| Eventing | weak | weak | n/a | n/a | weak | n/a | n/a | medium | In-memory bus only; NATS is unused. |
| Python intelligence | strong | via parent records | n/a | via parent flows | strong | medium | medium | medium | No batch analytics service or model registry yet. |
| Simulation | medium | via rollout plan metadata | n/a | via parent flows | medium | weak | weak | medium | Advisory simulation only; no traffic replay or stochastic models. |
| Web | medium | via API | medium | via API | medium | medium | n/a | medium | Primary operational browser flows are now covered, and integration onboarding/mapping now has browser proof, but several read-heavy routes remain thin. |
| CLI | medium | via API | medium | via API | weak | n/a | medium | medium | Integration onboarding commands exist now, but CLI test coverage remains shallow. |
| CI | medium | n/a | n/a | n/a | medium | medium | weak | medium | Browser coverage and Postgres-backed integration coverage now exist; long-running integration matrices still do not. |
