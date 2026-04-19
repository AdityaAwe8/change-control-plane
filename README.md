# ChangeControlPlane

ChangeControlPlane is an autonomous change control plane for software delivery, infrastructure, reliability, security, compliance, and cost-aware DevOps.

It is designed to treat every software change as a governed business event instead of a raw pipeline execution. The platform sits above existing delivery, infrastructure, observability, security, and collaboration tooling to decide how change should move, how it should be verified, and when it should be paused, rolled back, approved, or escalated.

## Vision

ChangeControlPlane is built to become a strategic DevOps operating system for:

- startups that need a premium path from zero to production
- scaling teams that need safer rollouts and clearer ownership
- enterprises that need auditability, policy controls, and progressive adoption across existing tooling

The product is intentionally broader than CI/CD, a service catalog, or an AI wrapper. Its core job is to understand change, assess risk, orchestrate rollout, observe runtime impact, enforce governance, and provide a control surface that teams can trust.

## Phase 1 Scope

This repository establishes the first serious baseline:

- monorepo scaffold with strong internal boundaries
- Go API, worker, and CLI entrypoints
- initial domain model for organizations, services, environments, changes, risk, rollout, policy, audit, integrations, incidents, and simulation
- deterministic risk scoring engine v1
- rollout planning engine v1
- PostgreSQL-backed application core with in-memory fallback for tests and local experiments
- versioned REST API under `/api/v1`
- premium frontend scaffold in TypeScript
- Docker Compose local dependencies
- OpenAPI contract, ADRs, and architecture docs
- Python intelligence subsystem for supplemental risk augmentation and rollout simulation
- worker control loop for rollout reconciliation
- test foundation for core flows plus authenticated smoke verification
- browser-level verification for the primary operational web flows

## Product Pillars

1. System Graph
   A live model of ownership, environments, services, dependencies, controls, and change history.
2. Change Intelligence
   Deterministic and explainable risk scoring, blast radius analysis, and rollout recommendations.
3. Delivery Governance
   Policy-aware control above pipelines, deploy systems, and environment promotion workflows.
4. Runtime Verification
   Post-deploy verification using technical and business-aware signals.
5. Enterprise Governance
   Auditability, approval flows, policy evaluation, compliance evidence, and secure operations.

## Architecture

The repository uses a modular-monolith approach for the first stage:

- one Go module for the application core and executable entrypoints
- domain-oriented internal packages with explicit seams for later extraction
- PostgreSQL-first data model and migrations
- in-memory repositories for fast local execution and tests
- event bus abstraction for domain events
- pluggable policy evaluation and integration adapters
- TypeScript web application scaffold with premium information architecture
- Python analytics and simulation subsystem invoked through a structured subprocess boundary

See the architecture docs in [docs/architecture/overview.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/overview.md) and the ADRs in [docs/adr](/Users/aditya/Documents/ChangeControlPlane/docs/adr/0001-monorepo.md).

## Repository Layout

```text
cmd/                  Executable entrypoints for api, worker, and cli
internal/             Domain modules, application services, repositories, and adapters
pkg/                  Exportable SDK/client/types groundwork
python/               Risk models, analytics, simulation, and tests
web/                  TypeScript frontend application
db/migrations/        PostgreSQL schema evolution
deploy/               Local Docker, Kubernetes, Helm, and Terraform examples
docs/                 Product, architecture, API, runbooks, and ADRs
test/                 Integration, end-to-end, and fixture assets
```

## What Is Implemented

The current baseline includes:

- CRUD-style APIs for organizations, projects, teams, services, and environments
- update and archive semantics for projects, teams, services, and environments
- change ingestion endpoint
- risk assessment endpoint backed by deterministic weighted rules
- rollout plan endpoint backed by risk-aware heuristics and Python simulation enrichment
- PostgreSQL-backed store behind the app storage seam
- signed dev auth bootstrap with persisted users and memberships
- organization and project RBAC enforcement with active tenant scope
- service-account and API-token lifecycle foundations with hashed token persistence
- rollout execution records, state transitions, and persisted verification outcomes
- worker-driven rollout auto-start and verified-to-complete reconciliation
- Python-backed supplemental risk augmentation with persisted metadata and explanations
- persisted graph enrichment for repositories and integration-sourced relationships
- policy evaluation abstraction with default production and regulated-zone policies
- audit event recording for critical actions
- starter integration registry with GitHub, Kubernetes, Slack, and Jira adapters
- health endpoints and structured JSON responses
- CLI commands for common control-plane actions
- TypeScript frontend scaffold with core control-plane pages

## What Is Intentionally Staged

These areas are designed now and expanded later:

- Temporal-backed long-running orchestration
- graph query engine beyond initial data model and blast-radius heuristics
- richer deployment verification and incident correlation
- policy packs, compliance packs, and premium gating
- advanced simulation and business-aware rollout verification
- AI-assisted explanations on top of deterministic outputs

## Local Development

### Prerequisites

- Go 1.26+
- Python 3.9+
- Node.js 22+
- pnpm 10+
- Docker Desktop or compatible container runtime

### Quick Start

```bash
cp .env.sample .env
make compose-up
make migrate
make build
make verify
make run-api
```

In another terminal:

```bash
make web-install
make web-dev
```

The default API address is `http://localhost:8080`.

`make compose-up` starts only the local dependency services on ports chosen to avoid common host collisions:

- PostgreSQL on `localhost:15432`
- Redis on `localhost:16379`
- NATS on `localhost:14222`

`make compose-up` now waits for the dependency containers to become ready before returning. The `make run-api`, `make migrate`, and `make run-worker` targets automatically use those dependency ports unless you override the `CCP_*` environment variables yourself.

If the web console runs on a different origin, set `CCP_ALLOWED_ORIGINS` accordingly. The sample environment file includes the default local Vite origins used by browser verification.

To run the worker as an authenticated machine actor, first issue a service-account token:

```bash
go run ./cmd/cli auth login --email owner@acme.local --name "Acme Owner" --organization-name Acme --organization-slug acme
go run ./cmd/cli service-account create --organization <org_id> --name worker-bot --role org_member
go run ./cmd/cli token issue --service-account <service_account_id> --name worker
export CCP_WORKER_TOKEN=<issued_token>
export CCP_WORKER_ORGANIZATION_ID=<org_id>
make run-worker
```

### Docker Dependencies

```bash
make compose-up
make compose-down
```

This starts:

- PostgreSQL
- Redis
- NATS

To run the API and worker fully inside Docker instead of on the host:

```bash
make compose-up-full
```

This rebuilds the Dockerized API and worker from the current repository state and exposes the containerized API on `http://localhost:28080` by default.

If that host port is already in use on your machine, override it:

```bash
CCP_DOCKER_API_HOST_PORT=38080 make compose-up-full
```

## API

The API contract lives in [docs/api/openapi.yaml](/Users/aditya/Documents/ChangeControlPlane/docs/api/openapi.yaml).

Core endpoints:

- `GET /healthz`
- `GET /readyz`
- `POST /api/v1/auth/dev/login`
- `GET /api/v1/auth/session`
- `GET|POST /api/v1/organizations`
- `GET|POST /api/v1/projects`
- `GET|POST /api/v1/teams`
- `GET|POST /api/v1/services`
- `GET|POST /api/v1/environments`
- `GET|POST /api/v1/changes`
- `GET|POST /api/v1/risk-assessments`
- `GET|POST /api/v1/rollout-plans`
- `GET|POST /api/v1/rollout-executions`
- `POST /api/v1/rollout-executions/{id}/verification`
- `GET /api/v1/policies`
- `GET /api/v1/audit-events`
- `GET /api/v1/integrations`
- `GET|POST /api/v1/service-accounts`
- `POST /api/v1/service-accounts/{id}/tokens`

## CLI

The `ccp` CLI now covers the main operator/admin surface for the product. Today it supports:

- `ccp auth login`
- `ccp auth session`
- `ccp org list`
- `ccp org create`
- `ccp project list`
- `ccp project create`
- `ccp team list`
- `ccp team create`
- `ccp team show`
- `ccp team update`
- `ccp team archive`
- `ccp service list`
- `ccp service register`
- `ccp service update`
- `ccp service archive`
- `ccp env list`
- `ccp env create`
- `ccp env update`
- `ccp env archive`
- `ccp service-account create`
- `ccp service-account list`
- `ccp service-account deactivate`
- `ccp token issue`
- `ccp token list`
- `ccp token revoke`
- `ccp token rotate`
- `ccp change list`
- `ccp change show`
- `ccp identity-provider list`
- `ccp identity-provider create`
- `ccp identity-provider update`
- `ccp identity-provider test`
- `ccp repository list`
- `ccp repository map`
- `ccp discovery list`
- `ccp discovery map`
- `ccp graph list`
- `ccp policy list`
- `ccp policy show`
- `ccp policy create`
- `ccp policy update`
- `ccp policy enable`
- `ccp policy disable`
- `ccp risk list`
- `ccp change analyze`
- `ccp rollout-plan list`
- `ccp rollout plan`
- `ccp rollout execute`
- `ccp rollout list`
- `ccp rollout show`
- `ccp rollout status`
- `ccp rollout advance`
- `ccp rollout pause`
- `ccp rollout resume`
- `ccp rollout rollback`
- `ccp rollout timeline`
- `ccp rollout reconcile`
- `ccp signal ingest`
- `ccp verification record`
- `ccp status list`
- `ccp rollback-policy list`
- `ccp rollback-policy create`
- `ccp rollback-policy update`
- `ccp audit list`
- `ccp incident list`
- `ccp incident show`
- `ccp outbox list`
- `ccp outbox retry`
- `ccp outbox requeue`
- `ccp integrations list`
- `ccp integrations create`
- `ccp integrations show`
- `ccp integrations update`
- `ccp integrations coverage`
- `ccp integrations test`
- `ccp integrations sync`
- `ccp integrations runs`
- `ccp integrations github-start`
- `ccp integrations webhook-show`
- `ccp integrations webhook-sync`

## Verification

The current verification policy and evidence matrix live in:

- [docs/testing/full-verification-plan.md](/Users/aditya/Documents/ChangeControlPlane/docs/testing/full-verification-plan.md)
- [docs/testing/full-verification-matrix.md](/Users/aditya/Documents/ChangeControlPlane/docs/testing/full-verification-matrix.md)
- [docs/testing/validation-criteria.md](/Users/aditya/Documents/ChangeControlPlane/docs/testing/validation-criteria.md)
- [docs/testing/security-verification.md](/Users/aditya/Documents/ChangeControlPlane/docs/testing/security-verification.md)

For an operator-facing ship gate across the strongest current local and preserved-proof checks, run:

```bash
make release-readiness
```

This command:

- reruns the highest-value local Go, web build, contract, and provider-harness checks
- revalidates `.tmp/reference-pilot/reference-pilot-report.json` and `.tmp/live-proof/live-proof-report.json` when present
- scans the generated release report, supporting logs, and preserved proof artifacts for configured secret-backed env leakage
- writes `.tmp/release-readiness/release-readiness-report.md`
- blocks by default on missing proof artifacts or hosted-like-only external proof

For a dry run before preserved proof bundles exist, set:

```bash
CCP_RELEASE_ALLOW_PROOF_GAPS=true make release-readiness
```

This override is only for local rehearsal. It does not turn hosted-like or missing external proof into real customer-environment evidence.
- [docs/testing/residual-risk-register.md](/Users/aditya/Documents/ChangeControlPlane/docs/testing/residual-risk-register.md)

## Documentation Map

- Product vision: [docs/product/vision.md](/Users/aditya/Documents/ChangeControlPlane/docs/product/vision.md)
- Personas: [docs/product/personas.md](/Users/aditya/Documents/ChangeControlPlane/docs/product/personas.md)
- Use cases: [docs/product/use-cases.md](/Users/aditya/Documents/ChangeControlPlane/docs/product/use-cases.md)
- Architecture overview: [docs/architecture/overview.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/overview.md)
- System graph: [docs/architecture/system-graph.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/system-graph.md)
- Change intelligence: [docs/architecture/change-intelligence.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/change-intelligence.md)
- Python intelligence: [docs/architecture/python-intelligence.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/python-intelligence.md)
- Delivery orchestration: [docs/architecture/delivery-orchestration.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/delivery-orchestration.md)
- Persistence: [docs/architecture/persistence.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/persistence.md)
- Auth model: [docs/architecture/auth-model.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/auth-model.md)
- Machine actors: [docs/architecture/machine-actors.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/machine-actors.md)
- RBAC: [docs/architecture/rbac.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/rbac.md)
- Security: [docs/architecture/security.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/security.md)
- Integrations: [docs/architecture/integrations.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/integrations.md)
- Graph enrichment: [docs/architecture/graph-enrichment.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/graph-enrichment.md)
- Rollout execution: [docs/architecture/rollout-execution.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/rollout-execution.md)
- Update/delete semantics: [docs/architecture/update-delete-semantics.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/update-delete-semantics.md)
- Event model: [docs/architecture/event-model.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/event-model.md)
- Multitenancy: [docs/architecture/multitenancy.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/multitenancy.md)
- Roadmap: [roadmap.md](/Users/aditya/Documents/ChangeControlPlane/roadmap.md)

## Commercial Direction

The codebase is structured so future packaging can cleanly separate:

- community or startup bootstrap mode
- growth and business tiers
- enterprise governance, compliance, identity, and self-hosting capabilities

No licensing is hard-coded in the application core. The design leaves room for future entitlement and feature-gating layers without polluting domain logic.
