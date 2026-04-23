# Full Verification Matrix

Status values:

- `not yet reviewed`
- `reviewed but unverified`
- `partially verified`
- `verified with automated tests`
- `verified manually only`
- `verified and hardened`
- `blocked / cannot fully verify yet`

## Web UI

### Routes and Pages

Authenticated routes now read from page-owned page-state models only. The chattiest operational/admin pages use bundled backend page-state endpoints, while thinner routes intentionally keep small route-local multi-request loaders that are covered by browser request-isolation tests.

| Surface | Status | Evidence / Notes |
| --- | --- | --- |
| `#/dashboard` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, login landing, refresh, sign-out, and seeded dashboard posture visibility |
| `#/bootstrap` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, project creation, team create/update/archive with visible route refresh, and read-only org-member view |
| `#/service` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, service creation/update/archive with visible route refresh, and org-member visibility denial |
| `#/environment` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, environment creation/update/archive with visible route refresh, and org-member visibility denial |
| `#/rollout` | verified and hardened | Browser tests cover route-local loading/error states, bundled page-state request usage, execution creation, approve/start, dedicated pause/resume/rollback control routes, reconcile, signal ingest, rollback visibility, and advisory recommendation-only affordances |
| `#/deployments` | verified and hardened | Browser tests now cover route-local loading/error states, bundled page-state request usage, server-backed search submit, rollback-only filtering, service/environment/source/automation filters, next-page pagination, reset behavior, and paginated summary visibility |
| `#/settings` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, service-account create/deactivate, token issue/rotate/revoke, consecutive route-local mutation refreshes, and read-only org-member view |
| `#/catalog` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, and seeded catalog rendering |
| `#/change-review` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, and seeded latest-change rendering |
| `#/risk` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, and seeded risk summary rendering |
| `#/incidents` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, and the incident feed linking into the dedicated detail route |
| `#/incident-detail` | verified and hardened | Browser tests cover route-isolated request patterns, selected-incident load, no-selection state, not-found state, and API failure state with route-local loading visibility |
| `#/policies` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, persisted policy create/update/disable flows, recent policy-decision visibility, and read-only org-member behavior |
| `#/audit` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, and seeded audit rendering |
| `#/integrations` | verified and hardened | Browser tests cover route-local loading/error states, bundled page-state request usage, advisory configuration visibility, connection test and sync controls, schedule configuration, GitHub and GitLab repository mapping, discovered-resource mapping refresh, multi-instance creation, provider-aware SCM visibility, GitHub App onboarding affordances, and repository ownership/mapping-provenance visibility after mapping changes |
| `#/enterprise` | verified and hardened | Browser tests cover route-local loading/error states, bundled page-state request usage, enterprise identity-provider visibility, provider creation, public SSO entry points, browser-session diagnostics and admin revocation with route-local refresh, durable outbox diagnostics, retry/requeue controls with route-local refresh, failure feedback, read-only org-member behavior, and webhook diagnostics visibility |
| `#/graph` | verified and hardened | Browser tests cover route-local loading/error states, bundled page-state request usage, stale-route response suppression, seeded graph rendering, and provenance-bearing graph rows such as `team_repository_owner` plus `integration_graph_ingest` evidence |
| `#/costs` | verified and hardened | Browser tests cover route-local loading/error states, route-isolated request patterns, and seeded cost-card rendering |
| `#/simulation` | verified and hardened | Browser tests cover route-local loading/error states, bundled page-state request usage, and rollback-policy-backed simulation rendering |

### Buttons, Forms, and Controls

| Control | Status | Evidence / Notes |
| --- | --- | --- |
| `#login-form` submit | verified and hardened | Browser-tested against real API; feedback added |
| `#refresh-button` | verified and hardened | Browser-tested; feedback added |
| `#logout-button` | verified and hardened | Browser-tested; feedback added |
| `#organization-switcher` | verified and hardened | Browser tests now prove multi-organization switching reloads route-local data for the selected tenant |
| `#create-project-form` | verified and hardened | Browser-tested |
| `#create-service-form` | verified and hardened | Browser-tested |
| `#archive-service-button` | verified and hardened | Browser-tested |
| `#create-environment-form` | verified and hardened | Browser-tested |
| `#archive-environment-button` | verified and hardened | Browser-tested |
| `#create-rollout-execution-form` | verified and hardened | Browser-tested |
| `#advance-rollout-form` | verified and hardened | Browser-tested for approve, start, pause, resume, and rollback, and now proved to hit the dedicated manual-control routes |
| `#reconcile-rollout-form` | verified and hardened | Browser-tested |
| `#create-signal-snapshot-form` | verified and hardened | Browser-tested |
| `#record-verification-form` | verified with automated tests | Browser tests now cover active verification submission, advisory recommendation submission, dedicated `/verification` route usage, and persisted execution/detail effects |
| `#status-search-input` | verified and hardened | Browser-tested against the server-backed search flow |
| `#status-rollback-only` | verified and hardened | Browser-tested against the server-backed search flow |
| `#status-search-reset` | verified with automated tests | Browser tests now cover filter reset and route-local dashboard refresh back to the default server-backed query |
| `[data-status-offset]` pagination buttons | verified with automated tests | Browser tests now cover next-page pagination against the bundled deployments page-state endpoint |
| `#create-identity-provider-form` | verified with automated tests | Browser admin surface is exercised and API/CLI create paths are directly tested |
| `.revoke-browser-session-button` | verified with automated tests | Browser tests cover admin-only visibility, successful browser-session revocation with route-local enterprise refresh, and read-only absence for org members |
| `.outbox-retry-button` | verified with automated tests | Browser tests cover admin-only visibility, successful retry with route-local enterprise refresh, and failure feedback |
| `.outbox-requeue-button` | verified with automated tests | Browser tests cover admin-only visibility and successful dead-letter requeue with route-local enterprise refresh |
| `#create-policy-form` | verified with automated tests | Browser tests cover persisted policy creation, route-local refresh, and policy-triggered recent-decision visibility |
| `.policy-config-form` submit | verified with automated tests | Browser tests cover inline policy update persistence on the Policy Center |
| `.policy-toggle-button` | verified with automated tests | Browser tests cover enable/disable persistence and read-only absence for org members |
| `.integration-config-form` submit | verified with automated tests | Browser tests cover persisted advisory/schedule configuration updates with route-local refresh |
| `.integration-test-button` | verified with automated tests | Browser tests cover integrations-route test invocation, success feedback, and recent-activity refresh |
| `.integration-sync-button` | verified with automated tests | Browser tests cover integrations-route sync invocation, success feedback, and recent-activity refresh |
| `.integration-webhook-sync-button` | verified with automated tests | Browser webhook-health surface is visible and API/CLI sync behavior is directly tested |
| `.repository-map-form` submit | verified with automated tests | Browser tests cover repository-to-service/environment mapping with visible provenance refresh |
| `.discovered-resource-map-form` submit | verified with automated tests | Browser tests cover discovered-resource mapping persistence and route-local refresh |
| `#create-team-form` | verified and hardened | Browser-tested |
| `#update-team-form` submit | verified and hardened | Browser-tested |
| `#archive-team-button` | verified and hardened | Browser-tested |
| `#update-service-form` submit | verified and hardened | Browser-tested |
| `#update-environment-form` submit | verified and hardened | Browser-tested |
| `.issue-token-form` submit | verified and hardened | Browser-tested |
| `.rotate-token-form` submit | verified and hardened | Browser-tested |
| `.revoke-token-button` | verified and hardened | Browser-tested |
| `.deactivate-service-account-button` | verified and hardened | Browser-tested |
| `#create-service-account-form` | verified and hardened | Browser-tested |

## API

### Auth, Catalog, and Metrics

| Endpoint | Status | Evidence / Notes |
| --- | --- | --- |
| `GET /healthz` | verified with automated tests | runtime contract test now validates the documented health envelope directly |
| `GET /readyz` | verified with automated tests | runtime contract test now validates the documented readiness envelope directly |
| `POST /api/v1/auth/dev/login` | verified and hardened | HTTP tests now cover cookie-backed dev bootstrap issuance, CLI test still covers bearer use, browser tests cover authenticated navigation, and smoke keeps the persisted path exercised |
| `POST /api/v1/auth/sign-up` | verified and hardened | HTTP tests cover password-account creation plus HttpOnly browser-session cookie issuance, and browser tests cover the sign-up-to-bootstrap path |
| `POST /api/v1/auth/sign-in` | verified and hardened | HTTP tests cover password sign-in, helpful invalid-credential feedback, and HttpOnly browser-session cookie issuance; browser tests cover protected-route landing |
| `GET /api/v1/auth/providers` | verified with automated tests | enterprise auth HTTP test and browser public SSO visibility |
| `POST /api/v1/auth/providers/{id}/start` | verified and hardened | enterprise auth HTTP tests cover authorize URL generation, signed state, and unsafe return-to normalization |
| `GET /api/v1/auth/providers/callback` | verified and hardened | enterprise auth HTTP tests cover callback, HttpOnly browser-session cookie issuance, session attribution, no-token-in-URL redirect behavior, and safe redirect fallback |
| `GET /api/v1/auth/session` | verified and hardened | HTTP tests cover cookie-backed session resolution plus expired/revoked-session rejection, CLI test still covers bearer lookup, browser tests cover reload persistence, and runtime contract validation covers the documented session envelope |
| `POST /api/v1/auth/logout` | verified and hardened | HTTP tests cover cookie-backed logout revocation, origin-guarded browser mutation behavior, and anonymous post-logout session truth; browser tests cover truthful sign-out |
| `GET /api/v1/catalog` | verified with automated tests | browser load path plus direct runtime contract validation now cover the documented catalog envelope |
| `GET /api/v1/metrics/basics` | verified and hardened | integration test, browser load path, and runtime contract validation now cover the documented metrics envelope |
| `GET /api/v1/incidents` | verified with automated tests | Direct HTTP route test now covers derived-incident listing, query filters, limit behavior, and cross-org denial; browser incident-feed coverage remains in place |
| `GET /api/v1/incidents/{id}` | verified with automated tests | Dedicated HTTP route test covers success, non-incident not-found, and cross-org forbidden behavior |

### Organization, Project, Team, Service, Environment

| Endpoint | Status | Evidence / Notes |
| --- | --- | --- |
| `GET /api/v1/organizations` | verified with automated tests | integration flow lists orgs after login and runtime contract validation now covers the documented list envelope |
| `POST /api/v1/organizations` | verified with automated tests | dev-login bootstrap still exercises organization creation indirectly, and runtime contract validation now covers the documented create envelope directly |
| `GET /api/v1/organizations/{id}` | verified with automated tests | Direct HTTP CRUD test covers success and cross-org denial |
| `PATCH /api/v1/organizations/{id}` | verified with automated tests | Direct HTTP CRUD test covers update persistence and cross-org denial |
| `GET /api/v1/projects` | verified and hardened | auth, tenant denial, browser, CLI, integration |
| `POST /api/v1/projects` | verified and hardened | validation tests, RBAC denial, browser, integration |
| `GET /api/v1/projects/{id}` | verified with automated tests | Direct HTTP CRUD test covers success and cross-org denial |
| `PATCH /api/v1/projects/{id}` | verified with automated tests | Direct HTTP CRUD test covers update persistence and cross-org denial |
| `POST /api/v1/projects/{id}/archive` | verified with automated tests | Direct HTTP CRUD test covers archive status transition |
| `GET /api/v1/teams` | verified with automated tests | direct HTTP CRUD test now covers list plus browser load path and integration flows |
| `POST /api/v1/teams` | verified with automated tests | direct HTTP CRUD test plus integration flows |
| `GET /api/v1/teams/{id}` | verified with automated tests | dedicated HTTP CRUD test covers team detail retrieval |
| `PATCH /api/v1/teams/{id}` | verified with automated tests | dedicated HTTP CRUD test covers owner/status updates |
| `POST /api/v1/teams/{id}/archive` | verified with automated tests | dedicated HTTP CRUD test covers archive status transition |
| `GET /api/v1/services` | verified with automated tests | token auth, browser, integration |
| `POST /api/v1/services` | verified and hardened | browser and integration |
| `GET /api/v1/services/{id}` | verified with automated tests | Direct HTTP CRUD test covers success and cross-org denial |
| `PATCH /api/v1/services/{id}` | verified with automated tests | Direct HTTP CRUD test covers update persistence and cross-org denial |
| `POST /api/v1/services/{id}/archive` | verified and hardened | Direct HTTP CRUD test covers archive status transition; RBAC denial test and browser test remain in place |
| `GET /api/v1/environments` | verified with automated tests | browser and integration |
| `POST /api/v1/environments` | verified and hardened | browser and integration |
| `GET /api/v1/environments/{id}` | verified with automated tests | Direct HTTP CRUD test covers success and cross-org denial |
| `PATCH /api/v1/environments/{id}` | verified with automated tests | Direct HTTP CRUD test covers update persistence and cross-org denial |
| `POST /api/v1/environments/{id}/archive` | verified and hardened | Direct HTTP CRUD test covers archive status transition; browser test remains in place |

### Change, Risk, Rollout, Runtime

| Endpoint | Status | Evidence / Notes |
| --- | --- | --- |
| `GET /api/v1/changes` | verified with automated tests | Direct HTTP route test covers list membership and tenant scoping; browser load path and integration flow remain in place |
| `POST /api/v1/changes` | verified with automated tests | integration flow and smoke |
| `GET /api/v1/changes/{id}` | verified with automated tests | Direct HTTP route test covers success, cross-tenant denial, and not-found behavior |
| `GET /api/v1/risk-assessments` | verified with automated tests | Direct HTTP route test covers list membership and tenant scoping; browser load path remains in place |
| `POST /api/v1/risk-assessments` | verified and hardened | HTTP policy tests now prove persisted advisory policy-decision evaluation and response visibility alongside integration flow and Python augmentation coverage |
| `GET /api/v1/rollout-plans` | verified with automated tests | Direct HTTP route test covers list membership and tenant scoping; browser load path remains in place |
| `POST /api/v1/rollout-plans` | verified and hardened | HTTP policy tests now prove manual-review gating, blocked-plan rejection, persisted rollout-plan policy decisions, and audit/status evidence alongside integration flow and smoke |
| `GET /api/v1/page-state/rollout` | verified and hardened | Dedicated HTTP route test covers bundled rollout plans, execution detail, and integration context; browser request-isolation tests prove rollout now uses the bundled endpoint |
| `GET /api/v1/rollout-executions` | verified and hardened | browser load path, smoke, integration, and runtime-contract validation now cover the documented list envelope |
| `POST /api/v1/rollout-executions` | verified and hardened | browser, integration, smoke, and runtime-contract validation now cover the documented create envelope |
| `GET /api/v1/rollout-executions/{id}` | verified and hardened | direct API test, browser, and runtime-contract validation now cover the documented detail envelope |
| `GET /api/v1/rollout-executions/{id}/evidence-pack` | verified with automated tests | direct API test and runtime-contract validation now cover the exportable evidence-pack envelope, including rollout context, policy outcomes, incidents, mapped repositories/resources, graph relationships, and audit trail |
| `POST /api/v1/rollout-executions/{id}/advance` | verified and hardened | integration, browser, smoke, and runtime-contract validation now cover the documented execution envelope |
| `POST /api/v1/rollout-executions/{id}/pause` | verified with automated tests | Direct HTTP route test covers successful pause transition and rollout timeline evidence; browser form and CLI command now hit the dedicated route |
| `POST /api/v1/rollout-executions/{id}/resume` | verified with automated tests | Direct HTTP route test covers successful resume transition and rollout timeline evidence; browser form and CLI command now hit the dedicated route |
| `POST /api/v1/rollout-executions/{id}/rollback` | verified with automated tests | Direct HTTP route test covers successful rollback transition and rollout timeline evidence; browser form and CLI command now hit the dedicated route alongside the separate automated rollback path |
| `POST /api/v1/rollout-executions/{id}/reconcile` | verified and hardened | integration, browser, smoke, and documented detail-envelope coverage |
| `POST /api/v1/rollout-executions/{id}/signal-snapshots` | verified and hardened | tenant denial, browser, integration, smoke, and runtime-contract validation now cover the documented snapshot envelope |
| `POST /api/v1/rollout-executions/{id}/verification` | verified and hardened | Direct HTTP route test covers active decision application, advisory decision rewrite, persisted verification results, rollout-detail effects, rollout-timeline evidence, and cross-tenant denial; browser tests now cover both active and advisory submission paths |
| `GET /api/v1/rollout-executions/{id}/timeline` | verified and hardened | API, browser, and runtime-contract validation now cover the documented status-timeline envelope |

### Audit, Status History, Rollback Policy

| Endpoint | Status | Evidence / Notes |
| --- | --- | --- |
| `GET /api/v1/audit-events` | verified and hardened | Direct HTTP route test now covers seeded audit evidence and tenant scoping alongside existing integration and smoke coverage |
| `GET /api/v1/policies` | verified and hardened | HTTP CRUD tests cover admin read plus tenant-scope denial, browser load path renders the route-local policy center, and runtime contract validation covers the documented list envelope |
| `POST /api/v1/policies` | verified and hardened | HTTP policy CRUD tests cover persisted creation, audit/status evidence, RBAC denial, and runtime contract validation covers the documented create envelope |
| `GET /api/v1/policies/{id}` | verified with automated tests | HTTP policy CRUD tests cover persisted show plus cross-tenant denial, and runtime contract validation covers the documented item envelope |
| `PATCH /api/v1/policies/{id}` | verified and hardened | HTTP policy CRUD tests cover persisted update/disable behavior, audit/status evidence, RBAC denial, and runtime contract validation covers the documented update envelope |
| `GET /api/v1/policy-decisions` | verified and hardened | HTTP policy tests cover persisted decision reads for risk and rollout workflows, rollout-block evidence, scoped filters, and runtime contract validation covers the documented list envelope |
| `GET /api/v1/status-events` | verified and hardened | API filter tests, browser, smoke, and runtime contract validation now cover the documented status-event list envelope |
| `GET /api/v1/status-events/search` | verified and hardened | API query-summary test, browser search flow, CLI command test, and runtime contract validation now cover the documented query-result envelope |
| `GET /api/v1/page-state/deployments` | verified and hardened | Dedicated HTTP route test covers bundled deployment dashboard payloads, filter/query propagation, rollback-policy inclusion, and browser request-isolation tests prove deployments now uses the bundled endpoint |
| `GET /api/v1/status-events/{id}` | verified with automated tests | direct API test, cross-tenant denial, and runtime contract validation now cover the documented detail envelope |
| `GET /api/v1/projects/{id}/status-events` | verified with automated tests | direct API test and runtime contract validation |
| `GET /api/v1/services/{id}/status-events` | verified with automated tests | direct API test and runtime contract validation |
| `GET /api/v1/environments/{id}/status-events` | verified with automated tests | direct API test and runtime contract validation |
| `GET /api/v1/rollback-policies` | verified and hardened | Direct HTTP route test now covers list membership, tenant scoping, and post-update persistence alongside browser load path |
| `POST /api/v1/rollback-policies` | verified and hardened | API creation, RBAC denial, integration path |
| `PATCH /api/v1/rollback-policies/{id}` | verified with automated tests | Direct HTTP route test covers update persistence, audit evidence, and not-found behavior |

### Integrations, Graph, Service Accounts, Tokens

| Endpoint | Status | Evidence / Notes |
| --- | --- | --- |
| `GET /api/v1/integrations` | verified with automated tests | CLI tests cover instance-scoped filters and browser load path exercises the multi-instance surface |
| `GET /api/v1/page-state/integrations` | verified and hardened | Dedicated HTTP route test covers bundled integrations read-model payloads, including team label context for ownership summaries, and browser request-isolation tests prove the page uses the bundled endpoint instead of frontend sync-run/webhook fan-out |
| `POST /api/v1/integrations` | verified with automated tests | HTTP integration test covers multi-instance create and browser flow exercises the create form |
| `GET /api/v1/integrations/coverage` | verified with automated tests | HTTP integration test and browser load path |
| `PATCH /api/v1/integrations/{id}` | verified with automated tests | integration onboarding browser flow and HTTP integration tests |
| `POST /api/v1/integrations/{id}/github/onboarding/start` | verified with automated tests | HTTP integration test covers signed state generation and GitHub App install URL creation |
| `GET /api/v1/integrations/github/callback` | verified with automated tests | HTTP integration test covers callback persistence, onboarding status update, installation-id capture, and the post-onboarding webhook-registration state used by the hosted-harness sync path |
| `POST /api/v1/integrations/{id}/test` | verified and hardened | HTTP integration tests cover GitHub, GitLab, Kubernetes, and Prometheus test runs, including Kubernetes and Prometheus bearer-auth/custom-path handling against realistic fake upstreams |
| `POST /api/v1/integrations/{id}/sync` | verified and hardened | HTTP integration tests cover GitHub and GitLab SCM sync plus deterministic CODEOWNERS ownership import and graceful no-CODEOWNERS behavior, provider-backed Kubernetes and Prometheus sync evidence, disappearing Kubernetes inventory handling, and no-sample Prometheus warning behavior |
| `GET /api/v1/integrations/{id}/sync-runs` | verified with automated tests | HTTP integration tests and browser load path |
| `GET /api/v1/integrations/{id}/webhook-registration` | verified with automated tests | HTTP enterprise test, CLI command test, browser webhook-health visibility, and hosted-harness tests now cover both stored repair state and provider-scoped callback persistence |
| `POST /api/v1/integrations/{id}/webhook-registration/sync` | verified and hardened | HTTP enterprise test and CLI command test cover automatic registration/repair, and hosted-harness tests now prove GitLab group repair plus GitHub App post-install registration and subsequent repair against realistic fake upstreams |
| `POST /api/v1/integrations/{id}/graph-ingest` | verified and hardened | idempotency API test and runtime contract validation cover the documented ingest envelope, and direct HTTP tests now prove repository service mapping, inferred owner-team persistence, and deterministic owner-edge creation from graph ingest inputs |
| `POST /api/v1/integrations/{id}/webhooks/github` | verified and hardened | signed webhook ingest, dedupe, mapped change creation, and sync-run persistence tests |
| `POST /api/v1/integrations/{id}/webhooks/gitlab` | verified and hardened | token-validated webhook ingest, merge-request file enrichment, mapped change creation, and sync-run persistence tests |
| `GET /api/v1/graph/relationships` | verified and hardened | Direct HTTP tests now cover filtered ownership-edge queries plus provenance metadata, and runtime contract validation covers the documented list envelope |
| `GET /api/v1/page-state/graph` | verified and hardened | Dedicated HTTP route test covers bundled topology payloads, including team/repository label context, and browser tests prove bundled route usage, stale-response suppression during rapid navigation, and visible provenance summaries on graph rows |
| `GET /api/v1/repositories` | verified and hardened | browser load path plus HTTP integration tests now cover provider/source-integration filters, CODEOWNERS ownership metadata, and inferred-owner metadata after mapping or graph ingest |
| `PATCH /api/v1/repositories/{id}` | verified with automated tests | repository mapping browser flow and HTTP integration tests |
| `GET /api/v1/discovered-resources` | verified with automated tests | HTTP integration tests cover unmapped listing and browser load path exercises the surface |
| `PATCH /api/v1/discovered-resources/{id}` | verified with automated tests | HTTP integration test covers mapping persistence and coverage summary impact |
| `GET /api/v1/identity-providers` | verified with automated tests | browser load path and CLI/admin diagnostics |
| `GET /api/v1/page-state/enterprise` | verified and hardened | Dedicated HTTP route test covers bundled enterprise/admin diagnostics and browser request-isolation tests prove enterprise uses the bundled endpoint |
| `POST /api/v1/identity-providers` | verified with automated tests | enterprise auth HTTP test, CLI create test, and browser admin surface |
| `PATCH /api/v1/identity-providers/{id}` | verified with automated tests | browser/admin surface exists and runtime contract validation now covers the documented update envelope directly |
| `POST /api/v1/identity-providers/{id}/test` | verified with automated tests | enterprise auth HTTP test covers provider discovery validation |
| `GET /api/v1/browser-sessions` | verified and hardened | HTTP tests cover admin success, member denial, cross-tenant exclusion, enterprise page-state bundling, CLI filters, and runtime-contract validation now covers the documented list envelope |
| `POST /api/v1/browser-sessions/{id}/revoke` | verified and hardened | HTTP tests cover org-admin revocation, cross-tenant denial, current-session cookie clearing, persisted revocation truth, CLI command coverage, browser admin refresh behavior, and runtime-contract validation now covers the documented item envelope |
| `GET /api/v1/outbox-events` | verified and hardened | CLI diagnostics test, authenticated web load path, and runtime-contract validation cover the documented envelope |
| `POST /api/v1/outbox-events/{id}/retry` | verified and hardened | HTTP enterprise tests cover admin success, audit evidence, RBAC denial, tenant-scope denial, dispatch compatibility, repeated-attempt rejection once the row is already `pending`, and recovery-race rejection when a worker claim lands before the guarded update; runtime-contract validation covers the documented response envelope |
| `POST /api/v1/outbox-events/{id}/requeue` | verified and hardened | HTTP enterprise tests cover admin success, audit evidence, preserved failure history, dispatch compatibility, and repeated-attempt rejection once the row is already `pending`; runtime-contract validation covers the documented response envelope |
| `GET /api/v1/page-state/simulation` | verified and hardened | Dedicated HTTP route test covers bundled scenario-planning payloads and browser request-isolation tests prove simulation uses the bundled endpoint |
| `GET /api/v1/service-accounts` | verified with automated tests | browser and integration flow plus runtime contract validation |
| `POST /api/v1/service-accounts` | verified and hardened | browser, integration, smoke |
| `POST /api/v1/service-accounts/{id}/deactivate` | verified with automated tests | dedicated HTTP route test covers deactivation plus machine-token invalidation; CLI command coverage now exists |
| `GET /api/v1/service-accounts/{id}/tokens` | verified and hardened | browser and API lifecycle tests plus runtime-contract validation cover the documented token-list envelope |
| `POST /api/v1/service-accounts/{id}/tokens` | verified and hardened | browser, API lifecycle tests, smoke, and runtime-contract validation cover the documented issued-token envelope |
| `POST /api/v1/service-accounts/{id}/tokens/{token_id}/revoke` | verified and hardened | browser and API lifecycle tests, plus runtime contract validation now cover the documented revoke envelope |
| `POST /api/v1/service-accounts/{id}/tokens/{token_id}/rotate` | verified and hardened | dedicated HTTP route test covers rotation, old-token revocation, rotated-token auth, audit evidence, CLI command coverage, and runtime contract validation now cover the documented rotate envelope |

## Storage / Database

| Path | Status | Evidence / Notes |
| --- | --- | --- |
| Core entity round-trips | verified with automated tests | PostgreSQL round-trip test covers org/project/team/service/env/change/risk/plan/audit/integration/repository/graph/service-account/token/execution/snapshot/verification/policy/policy-decision/status-event, including repository/discovered-resource/graph metadata persistence for ownership and provenance fields |
| Browser session persistence | verified with automated tests | In-memory auth tests cover issue, expiry, revocation, logout invalidation, and admin-scoped session listing/revocation; PostgreSQL round-trip test covers create, lookup-by-id, org/user/status-scoped listing, and revoked-session persistence |
| Status-event search/filter | verified with automated tests | PostgreSQL test covers rollback-only filter, text search, service scope, and not-found behavior |
| Outbox compare-and-update status guard | verified with automated tests | Dedicated PostgreSQL test covers status-guarded outbox updates succeeding for the expected source status and rejecting stale expected statuses without overwriting a fresher `processing` claim; local runs skip when PostgreSQL is unavailable |
| Fresh bootstrap / migration apply | verified with automated tests | dedicated PostgreSQL test now provisions a fresh temporary database when the local cluster allows it, proves `AutoMigrate` applies the full migration chain, replays migrations idempotently, and confirms the bootstrapped schema accepts real writes; local runs still skip when compatible PostgreSQL access is unavailable |
| Duplicate handling | verified with automated tests | dedicated PostgreSQL test now covers duplicate organization slug, duplicate API-token prefix, and duplicate repository URL rejection semantics |
| Transaction safety | verified with automated tests | dedicated PostgreSQL test now proves `WithinTransaction` rolls back persisted writes on callback error and commits them on success |
| Pagination and ordering | verified with automated tests | dedicated PostgreSQL test now proves list ordering and limit/offset behavior on catalog entities, in addition to the existing status-event query path |

## Control Loop / Worker / Rollback / Status History

| Surface | Status | Evidence / Notes |
| --- | --- | --- |
| Control-loop claim and reconcile | verified with automated tests | workflow tests and smoke |
| Scheduled integration sync claim and run | verified and hardened | workflow tests cover due integration claim, scheduled sync execution, freshness advancement, duplicate-claim prevention under concurrent contention, durable-event dispatch during worker passes, and outbox runtime behavior through the event bus tests, including fresh-claim skip, stale-claim reclaim, dead-letter handling, and recovery-history preservation across additional dispatch attempts |
| Healthy verification path | verified and hardened | workflow tests, smoke, browser |
| Automatic rollback path | verified and hardened | verification tests, workflow tests, smoke, browser |
| Duplicate rollback prevention | verified and hardened | workflow tests now prove repeated reconcile does not create a second automated rollback verification record or duplicate `rollout.execution.verified_automatically` status event once the rollout already reflects the rollback decision; broader multi-worker lease proof remains a separate gap |
| Pause/resume semantics | verified with automated tests | Control-plane execution pause/resume/rollback transitions are now directly proven through API, CLI, and browser tests. External provider mutation proof for customer-like live backends remains a separate gap. |
| Audit event creation | verified with automated tests | integration and smoke |
| Status-event creation | verified and hardened | API tests, DB tests, browser, smoke |
| Rollback-only filtering | verified with automated tests | API, DB, browser, smoke |
| Coverage summary calculation | verified with automated tests | HTTP integration and workflow tests cover discovered-resource and workload-coverage outcomes |

## Security

| Surface | Status | Evidence / Notes |
| --- | --- | --- |
| Human browser and bearer auth | verified and hardened | auth tests cover password/dev/OIDC login landing in HttpOnly cookie-backed browser sessions, logout revocation, expired/revoked-session rejection, cookie-backed reload persistence, and preserved bearer-token auth for CLI/API clients |
| Service-account token lifecycle | verified and hardened | API tests, browser tests, smoke |
| Cross-tenant scope denial | verified with automated tests | project, signal, status-event, identity-provider, service-account, rollout-control, and integration-mutation tests now cover representative cross-tenant denial paths |
| RBAC on admin mutations | verified with automated tests | targeted HTTP and browser tests now cover org-admin success plus org-member denial across identity-provider create/update/test, outbox retry/requeue, service-account issue/revoke/rotate, service/environment archive, integration mutation/mapping, rollout pause/rollback, and project creation |
| Cookie-authenticated mutation origin guard | verified with automated tests | HTTP auth tests cover configured-origin allow, disallowed-origin rejection, and bearer-auth mutation behavior remaining outside the cookie-origin guard |
| CORS for browser use | verified and hardened | handler tests and browser path now cover echoed origin plus credentialed browser support |
| Secret leakage prevention | verified with automated tests | `live-proof-verify` command tests now cover both full proof runs and preflight-only checklist generation without embedding configured GitHub/GitLab/Kubernetes/Prometheus secret values, and the new `artifact-safety-check` command tests plus `make release-readiness` now scan generated release reports, supporting logs, preserved proof artifacts, and the operator-facing live-proof checklist path for configured secret-backed env values without printing those secret values back out. This is still not a universal runtime metadata scrubber. |

## Reference Pilot Proof

| Surface | Status | Evidence / Notes |
| --- | --- | --- |
| `make reference-pilot-up` | verified manually only | starts dedicated pilot PostgreSQL, Redis, NATS, local `k3s`, the sample workload, in-cluster Prometheus, local GitLab fixture, API, `kubectl proxy`, and required port-forwards |
| `make reference-pilot-verify` | verified manually only | writes `.tmp/reference-pilot/reference-pilot-report.json` after configuring integrations, mapping discovered resources, ingesting a GitLab merge-request webhook, collecting real metrics, and recording advisory-only rollback behavior |
| `make reference-pilot-validate` | verified with automated tests | wraps `cmd/reference-pilot-verify --validate-report` and revalidates a saved `.tmp/reference-pilot/reference-pilot-report.json` artifact, including advisory-only runtime assertions |
| `make proof-live-preflight` | verified with automated tests | wraps `cmd/live-proof-verify --preflight-only` and writes `.tmp/live-proof/live-proof-preflight.json` plus `.tmp/live-proof/live-proof-operator-checklist.md`; command tests now cover both missing-input and ready=true preflight paths, including rendered callback/webhook URL patterns plus non-public API-base reachability warnings, without leaking configured secrets |
| `make proof-live-verify` | verified and hardened | wraps `cmd/live-proof-verify` and writes `.tmp/live-proof/live-proof-report.json`; command tests now cover hosted-like GitLab and GitHub App onboarding plus customer-like Kubernetes/Prometheus sync, stricter provider-specific config validation, explicit environment-class classification, secret-safe report output, richer proof checks/evidence summaries, and report validation before write |
| `make proof-live-validate` | verified with automated tests | wraps `cmd/live-proof-verify --validate-report` and revalidates a saved `.tmp/live-proof/live-proof-report.json` artifact |
| `make release-readiness` | verified manually with dry-run and blocker-path checks | aggregates the strongest current local Go/web build checks, contract and provider harness proof, live-proof preflight/checklist generation, saved reference-pilot and live-proof validation, scans the generated release report/logs/proof artifacts for secret-backed env leakage, writes `.tmp/release-readiness/release-readiness-report.md`, blocks by default on missing/hosted-like-only external proof, and now points missing-artifact failures at `.tmp/live-proof/live-proof-operator-checklist.md` for exact remaining operator inputs |
| Local GitLab fixture onboarding and webhook registration | verified manually only | reference pilot proves token-based GitLab discovery, automatic webhook registration/repair, and merge-request webhook ingest against the fixture |
| Local-cluster Kubernetes observation | verified manually only | reference pilot report proves discovered workload mapping, real deployment observation, and rollout runtime evidence from `k3s` |
| Local-metrics Prometheus collection | verified manually only | reference pilot report proves real query-window collection and persisted signal snapshots from the in-cluster Prometheus deployment |
| Advisory-only runtime suppression | verified manually only | reference pilot report proves `advisory_only=true`, `last_action_disposition=suppressed`, and `advisory_rollback` without live backend mutation |

## CLI

| Command Surface | Status | Evidence / Notes |
| --- | --- | --- |
| `auth login` | verified with automated tests | command test |
| `auth session` | verified with automated tests | dedicated command test covers session lookup output and bearer-token header use |
| `change list/show/analyze` | verified with automated tests | command tests cover persisted change creation, list/show reads, and analyze output including risk recommendation and policy decisions |
| `risk list` | verified with automated tests | dedicated CLI test now covers the persisted risk-assessment read surface exposed by the API |
| `rollout-plan list` | verified with automated tests | dedicated CLI test now covers the persisted rollout-plan read surface exposed by the API |
| `integrations create/list` | verified with automated tests | command tests cover creation, provider-aware instance-scoped listing behavior, and mixed-provider visibility |
| `integrations show/test/sync/runs/coverage/github-start/webhook-show/webhook-sync` | verified with automated tests | command tests now cover show, schedule update, coverage, provider test and sync calls, GitHub App onboarding start, and webhook registration diagnostics |
| `identity-provider list/create/update/test` | verified with automated tests | dedicated CLI test now covers the full command group, including list output, create/update request bodies, and test-route invocation |
| `browser-session list/revoke` | verified with automated tests | dedicated CLI test covers filter encoding, organization-scope headers, and persisted browser-session revocation routing |
| `policy list/show/create/update/enable/disable` | verified with automated tests | dedicated CLI test covers list, show, create, update, and enable/disable routing plus expected request bodies |
| `policy-decision list` | verified with automated tests | dedicated CLI test now covers persisted policy-decision reads with project/policy/change/risk/rollout filters and organization-scope headers |
| `outbox list` | verified with automated tests | command test covers filters and output |
| `outbox retry/requeue` | verified with automated tests | dedicated CLI test covers both recovery commands and their routed API calls |
| `repository list/map` | verified with automated tests | dedicated CLI test now covers provider/source filter encoding, mapped repository output, and repository mapping request bodies |
| `discovery list/map` | verified with automated tests | command tests cover discovered-resource listing and mapping |
| `graph list` | verified with automated tests | dedicated CLI test covers relationship-type, source-integration, from/to, limit, and offset filter encoding |
| `status list/show/project/service/env` | verified and hardened | command tests now cover the server-backed search endpoint, status-event detail reads, scoped project/service/environment history routes, richer filters, and output |
| `incident list/show` | verified with automated tests | command tests cover incident list filter encoding and dedicated incident detail lookup |
| `rollout plan/execute/list/show/status/advance/timeline/reconcile` | verified with automated tests | dedicated CLI test now covers the core rollout recommendation, execution, detail, transition, timeline, and reconcile command set |
| `verification record` and `signal ingest` | verified with automated tests | dedicated CLI test now covers persisted verification submission and signal-snapshot ingestion against the runtime routes |
| `rollback-policy list/create/update` | verified with automated tests | dedicated CLI test now covers persisted rollback-policy reads and mutation payloads |
| `audit list` | verified with automated tests | dedicated CLI test now covers the audit-event read surface |
| `live-proof-verify` | verified and hardened | command tests cover hosted-like GitLab and GitHub App proof runs, repository/runtime mapping, explicit proof environment-class handling, secret-safe preflight/checklist generation, fail-fast missing-secret validation, hosted-SaaS/local-endpoint rejection, secret-safe report generation, saved-report validation mode, and richer proof check/evidence output |
| `team create/list/show/update/archive` | verified with automated tests | dedicated CLI test covers team lifecycle commands, including owner updates and archive |
| `rollout pause/resume/rollback` | verified with automated tests | command tests now prove the dedicated control endpoints and encoded operator reasons |
| `org/project/service/env/token` commands | verified with automated tests | dedicated CLI tests now cover org/project list/create, service list/register/update/archive, environment list/create/update/archive, and token issue/list/revoke request routing plus expected payloads and organization-scope header use |

## Python Intelligence

| Surface | Status | Evidence / Notes |
| --- | --- | --- |
| Python package and unit tests | verified with automated tests | `python/tests/test_intelligence.py` |
| Go subprocess boundary | verified with automated tests | `internal/intelligence/python_test.go` |
| Persistence of augmentation output | verified with automated tests | HTTP intelligence test |

## Providers

| Provider | Status | Evidence / Notes |
| --- | --- | --- |
| GitHub SCM provider | verified and hardened | shared SCM model, webhook ingest, repository discovery, GitHub App onboarding, installation-token exchange, hosted-like organization webhook registration/repair, and CODEOWNERS file lookup/parse behavior are integration-tested directly, including graceful no-CODEOWNERS behavior; not marketplace/live GitHub proven |
| GitLab SCM provider | verified and hardened | token-based onboarding, project discovery, webhook ingest, merge-request file enrichment, automatic webhook registration/repair, and CODEOWNERS file lookup/parse behavior are integration-tested directly, including graceful no-CODEOWNERS behavior; not OAuth/live GitLab proven |
| Simulated orchestrator | verified and hardened | unit tests, workflow tests, smoke, browser |
| Simulated signal provider | verified and hardened | unit tests, workflow tests, smoke, browser |
| Kubernetes provider | verified with automated tests | near-real HTTP client tested against realistic responses, normalized sync evidence, advisory suppression, disappearing-resource handling, and bearer-auth/custom-path request shaping; no live cluster proof |
| Prometheus provider | verified with automated tests | near-real HTTP client tested against realistic responses, empty-result handling, signal normalization, and bearer-auth/custom-path query routing; no live backend proof |

## Documentation / Claims

| Surface | Status | Evidence / Notes |
| --- | --- | --- |
| OpenAPI | verified and hardened | contract tests now assert every registered HTTP route has a documented OpenAPI path/method entry, compare key schema properties to runtime Go types, and validate serialized runtime responses for rollout, incident, integration, outbox, token, and older catalog/governance surfaces. Verification is still not a full schema/runtime diff. |
| Architecture docs | partially verified | current runtime milestones are represented, but some breadth docs still exceed proof |
| README / local dev docs | reviewed and aligned this pass | release-readiness, live-proof preflight/checklist generation, preserved-proof validation, artifact secret-safety scanning, and proof-class truth are now documented directly in README and runbooks |
