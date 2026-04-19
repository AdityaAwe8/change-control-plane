# Live Business Integration Status

This document is intentionally reality-based. A route, descriptor, form, or metadata shape is not treated as live business integration on its own.

Status legend used here:

- `fully live`
- `near-live`
- `simulated only`
- `descriptor only`
- `partially implemented`
- `missing`

## Current Audit

| Area | Status | Reality |
| --- | --- | --- |
| Integration instance persistence | `near-live` | Organization-scoped integration instances are now persisted with enablement, advisory/control flags, instance identity, scope metadata, auth strategy, schedule/freshness state, last test/sync timestamps, and latest error state. Multiple named instances per kind are now supported for pilot use. |
| Integration sync history | `partially implemented` | Sync/test/webhook runs are now persisted and queryable. This gives the platform real connection history and webhook idempotency evidence, but not yet a full jobs/queue subsystem. |
| Secret handling | `partially implemented` | The platform now expects env-var secret references such as `access_token_env`, `webhook_secret_env`, and `bearer_token_env` instead of storing raw secrets in integration metadata. This is safer than inline secrets, but it is not a full secret-manager integration. |
| GitHub descriptor/catalog | `partially implemented` | The static registry entry still seeds a default org-scoped GitHub integration, but operators can now create additional named GitHub instances with different auth and scope metadata. |
| GitHub API connection test | `near-live` | A real token-backed GitHub API test path now exists through the integration test endpoint. |
| GitHub repository discovery | `near-live` | The platform can now call the GitHub API, discover repositories, persist them, and expose them in the onboarding surface for mapping. Pagination and very large org handling are still shallow. |
| GitHub App onboarding | `partially implemented` | The platform now has a real installation-style start/callback flow, signed state handling, installation metadata persistence, and dynamic installation-token minting from `app_id`, `private_key_env`, and `installation_id`. Marketplace polish, automatic webhook registration, and OAuth user-consent flow are still missing. |
| GitHub webhook ingest | `near-live` | The platform now accepts GitHub webhooks, validates `X-Hub-Signature-256`, records webhook runs, deduplicates by delivery id, and ingests mapped push or PR change metadata into persisted change sets. GitHub App installation onboarding now coexists with the legacy PAT path. |
| GitHub changed-file ingest | `near-live` | Push payloads use file lists from the webhook payload, and pull requests can fetch changed files through the GitHub API. Large PR pagination and richer review state are still limited. |
| Kubernetes provider runtime path | `near-live` | The Kubernetes provider still uses real HTTP calls and now has an onboarding/test/sync surface around it. It remains HTTP-backed and is not `client-go` or cluster-controller based. |
| Kubernetes onboarding and health visibility | `partially implemented` | Admins can now configure, test, and sync Kubernetes connectivity. Resource discovery is still shallow and target registration is metadata-driven. |
| Prometheus provider runtime path | `near-live` | The Prometheus provider still performs real query requests and now has connection test/sync visibility. Query templates and service bindings are still metadata-driven. |
| Prometheus onboarding and health visibility | `partially implemented` | Admins can now configure, test, and sync the Prometheus connection path. Continuous telemetry ingest is still not implemented. |
| Repository-to-service/environment mapping | `partially implemented` | Repositories are now persisted with mapping fields and can be linked from the web UI or CLI to services and environments. Mapping is still manual rather than fully inferred. |
| Topology discovery | `partially implemented` | Repository discovery and graph upserts now exist. Dependency, ownership, and workload discovery remain limited and mostly metadata-driven. |
| Deployment association | `partially implemented` | Rollout executions can already point at backend integrations, and advisory-safe live observation now exists. Automatic association of existing business deployments into rollout records still needs more depth. |
| Runtime signal sourcing | `near-live` | Prometheus-backed signal collection is materially real when configured. Scheduling, retention, and multi-tenant metrics hardening are still future work. |
| Onboarding UX | `partially implemented` | The web app now has a real integration onboarding surface with config, enablement, advisory mode, connection tests, sync, repository discovery, and repository mapping. It is still a serious first-run page rather than a polished multi-step wizard. |
| CLI onboarding support | `partially implemented` | The CLI can now create/list/show/update/test/sync integrations, start GitHub App onboarding, and list/map repositories. It is still JSON-first and operational rather than polished. |
| OpenAPI coverage | `partially implemented` | OpenAPI now covers the newest onboarding, sync, and multi-instance routes materially better than before, but older CRUD surfaces still need a fuller schema pass. |

## Reality Check

What is materially more real now:

- GitHub is no longer only a descriptor; there is now a real token-backed API path, a GitHub App installation-style onboarding flow, and webhook ingest.
- Integrations now have persisted health, sync history, and advisory/control state.
- The product can now model more than one GitHub, Kubernetes, or Prometheus instance per org.
- Repository discovery and mapping are now part of the product rather than only seed data or graph ingest utilities.
- Advisory mode now prevents live backend control actions during reconcile for non-simulated providers.

What is still not claimed:

- no marketplace-grade GitHub App product or OAuth setup flow
- no `client-go` Kubernetes controller or cluster inventory importer
- no continuous Prometheus scraping or long-running telemetry scheduler
- no fully automatic service/environment inference from manifests, CODEOWNERS, or deployment metadata
- no enterprise secret-manager or KMS integration
