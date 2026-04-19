# Reference Integration Maturity Status

This document tracks the real maturity of the platform's reference integration story after the shared SCM and GitLab expansion work.

| Area | Status | Reality |
| --- | --- | --- |
| GitHub PAT path | real but limited | Still supported for legacy setup and webhook ingest, but it remains operator-plumbing-heavy compared with the GitHub App path. |
| GitHub App onboarding | partially implemented | The platform now has a real install-style start/callback flow, installation scope persistence, and installation-token minting from `app_id` + `private_key_env` + `installation_id`. Marketplace polish, OAuth user-consent flow, and automatic webhook registration are still missing. |
| GitLab onboarding | real but limited | GitLab now has a product-shaped token-based onboarding path with scope, health, sync, and webhook support. OAuth and app-style onboarding are still missing. |
| Shared SCM model | real but limited | GitHub and GitLab now fit the same repository discovery, webhook normalization, change ingest, and coverage model. Some provider-specific seams and provenance limits remain. |
| Multi-instance integration persistence | real and scalable enough for pilots | Multiple instances per org and kind now exist through `instance_key`, `scope_type`, `scope_name`, `auth_strategy`, and per-instance schedule/freshness state. |
| Multi-instance operator visibility | real but limited | API, web, and CLI now distinguish named instances and scope, but some summaries still remain aggregate rather than fully drill-down-first. |
| Kubernetes proof level | near-real only | HTTP-backed status/action normalization is now proven through richer repeated-state and failure harness tests, not through a live cluster. |
| Prometheus proof level | near-real only | Query-range collection, repeated windows, empty results, and failure handling are harness-proven, not live-metrics proven. |
| Scheduler contention confidence | real but limited | DB-backed claims already existed, and the worker now has explicit contention-focused proof for duplicate claim prevention. It is still not a distributed scheduler with crash-recovery proof. |
| Repository/discovery scope handling | partially implemented | Repositories now retain a primary `source_integration_id` and multi-instance provenance hints, but shared-repo attribution across overlapping GitHub or GitLab scopes is still imperfect. |
| Coverage summaries across instances | real but limited | Coverage now counts enabled/stale/healthy integrations across all instances and tracks kind-level totals, but summary cards still compress nuance. |

## Bottom Line

The platform is now materially closer to a credible reference integration product:

- GitHub is no longer only a PAT-and-env story.
- GitLab is now a real SCM integration instead of a future placeholder.
- The product can now model more than one GitHub, GitLab, Kubernetes, or Prometheus instance per org.
- Kubernetes and Prometheus are better proven through live-like harnesses.

It is still not accurate to call the reference paths fully production-proven live integrations.
