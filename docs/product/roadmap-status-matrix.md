# Roadmap Status Matrix

This document is intentionally reality-based. A feature is not counted as implemented because a type, page, route, or doc exists. The status below reflects codepaths inspected in this repository and the verification rerun in this pass.

Status legend:

- `available_now`: materially implemented end-to-end and usable today
- `partial`: meaningful product surface exists, but important parts of the full vision are still missing
- `missing`: not materially implemented beyond helpers, heuristics, or ideas

## Requested Feature Status

| Feature | Status | What Exists Today | Biggest Remaining Gap |
| --- | --- | --- | --- |
| AI deployment readiness review | `partial` | Release analysis now generates deterministic readiness-review items with severity, category, question, plain-English reason, evidence, and acknowledgment-required signals. These now include DB-aware checks for compatibility, irreversibility, required validation checks, and explicit acknowledgment on risky database changes. They are exposed through API, CLI, web, incident detail, and evidence-pack linked release context. | There is still no true LLM-backed conversational review flow or persisted human acknowledgment workflow. |
| Release bundle selection | `available_now` | Operators can now create, list, inspect, and update persisted release bundles, attach config sets, analyze combined risk and blast radius, see dependency/conflict findings, and link bundles to rollout executions and evidence packs through API, CLI, and web UI. | There is still no richer release-queue UX, branch/PR-native dependency graph, or bundle approval workflow beyond the current deterministic release surface. |
| AI alerts and DevOps help | `partial` | Deterministic assisted reasoning now exists through `OpsAssistantSummary` on release analysis and incident detail, including likely cause, suspicious changes, and rollback vs fix-forward guidance. | There is still no true LLM-backed incident copilot or live observability-tool correlation beyond the current deterministic reasoning layer. |
| Database integration and deployment queries | `partial` | First-class persisted database governance records now exist for database changes, connection references, connection-health tests, controlled validation checks, and runtime validation executions across API, CLI, web, Postgres, evidence packs, rollout page-state, release analysis, and rollout execution gating. The product now supports environment-scoped Postgres connection references with explicit source posture (`env_dsn` plus executable env-bound `secret_ref_dsn` via `secret_ref_env`), can run persisted connection tests, and can execute a narrow safe read-only runtime validator for structured checks such as table existence, migration markers, and row-count assertions with persisted execution evidence. Unbound secret refs are surfaced truthfully as unresolved rather than being faked. | There is still no broad database connector catalog, migration execution runner, secret-manager-backed credential resolution, generic DBA workflow, or customer-environment DB proof. |
| Configuration set management | `partial` | First-class config sets now exist with persistence, per-environment scope, optional service scope, validation, diff summary, missing/deprecated key detection, invalid secret-ref detection, release linkage, API, CLI, web UI, and rollout-route visibility. | Overlays, broader provenance/source-of-truth tracking, and drift-aware config governance are still missing. |
| Secret reference management | `partial` | Config sets validate and preserve secret references explicitly via `value_type=secret_ref`, and DB connection references now support secret-safe source indirection rather than plaintext DSNs. Runtime execution now supports `env_dsn` and env-bound `secret_ref_dsn` references via `secret_ref_env`, while unsupported or unbound secret refs are surfaced truthfully as unresolved instead of being faked. Release analysis, DB execution, connection-health tests, and artifact scans preserve redacted secret-safe outputs while still surfacing connection-reference posture truthfully. | There is still no secret-manager integration, runtime injection engine, or access/audit workflow for secret-reference lifecycle. |
| Secret encryption and decryption workflow | `missing` | The product now leans into secret references instead of plaintext config, which is the correct direction. | There is still no encrypted secret vault, safe reveal workflow, rotation UX, or runtime-managed secret storage surface. |
| Change risk scoring | `available_now` | Deterministic risk assessments remain real and persisted with score, level, explanation, blast radius, recommended approvals, rollout strategy, deployment window, and guardrails across API, CLI, web, release analysis, and evidence packs. | It is still deterministic rather than history-learned or customer-telemetry-trained. |
| Blast radius analysis | `available_now` | Blast radius remains part of persisted risk assessment and is now also rolled up into release analysis and evidence-pack summary. | The current surface is still summary-oriented rather than a richer visual multi-system blast-radius explorer. |
| Policy-as-code deployment governance | `partial` | Persisted deterministic policies and decisions exist for `risk_assessment` and `rollout_plan`, and release analysis now highlights policy posture in plain language. DB-aware governance now also contributes deterministic blockers/highlights for irreversible or insufficiently validated database changes before a rollout execution can be created from a linked release. | Policies still do not govern release bundles, config sets, change windows, or external policy backends end-to-end through a unified policy engine. |
| Deployment evidence pack | `available_now` | Evidence packs are real and now include release context plus release analysis alongside change, risk, plan, execution detail, policy outcomes, incidents, repositories, discovered resources, graph relationships, integrations, audit trail, and DB governance context (`database_changes`, `database_checks`, `database_connections`, `database_executions`, and `database_posture`). API and CLI surfaces are live and tested. | It is still an API/CLI export surface rather than a signed downloadable artifact workflow or dedicated browser export experience. |
| Post-deploy verification | `partial` | Verification results, signal snapshots, reconcile flows, automated decisions, incident linkage, and runtime summaries are real and persisted. | Broader business-metric verification, synthetic-test integration, and richer explainability remain incomplete. |
| Rollback intelligence | `partial` | Release analysis now emits structured rollback guidance with safety posture, strategy, steps, and blockers; existing rollback policies and runtime decisions remain real. Database posture now feeds rollback safety directly, including compatibility/irreversibility warnings and fix-forward-biased posture when rollback is DB-constrained. | There is still no full cross-system rollback planner for migrations, feature flags, and coordinated dependent-service rollback. |
| Drift detection and change provenance | `partial` | Repositories, discovered resources, graph relationships, provenance metadata, and mapped runtime/resource evidence remain real and are included in evidence packs. | Desired-vs-live drift, GitOps-grade sync posture, and config/secret drift detection are still incomplete. |
| Release calendar and freeze windows | `partial` | Risk assessments still recommend deployment windows, and release analysis now surfaces window/collision findings in deterministic form. | There is still no persisted release calendar, freeze-period model, or operator-managed maintenance window workflow. |
| Dependency-aware release planning | `partial` | Release analysis now emits dependency-plan rows derived from persisted service/graph relationships and links them to release composition and rollback posture. | Automatic ordering enforcement and deeper app/db/frontend/consumer sequencing rules remain incomplete. |
| Team memory and knowledge graph | `partial` | The graph/provenance foundation remains real, and release analysis now emits deterministic team-memory insights derived from historical risk and incident-shaped evidence. | There is still no learned long-term memory layer or richer historical pattern-mining pipeline. |
| Compliance and enterprise controls | `partial` | Audit events, RBAC, browser-session diagnostics/revocation, service accounts, tenant scoping, OIDC foundations, and export-friendly evidence packs are real. | Separation of duties, stronger exception workflows, broader enterprise IAM breadth, and richer compliance reporting remain incomplete. |
| Communication automation | `partial` | Release analysis now generates deterministic communication drafts for release notes, approver summary, stakeholder update, maintenance notice, incident handoff, and postmortem starter content. | There is still no dedicated outbound workflow, approvals around generated comms, or richer AI-authored variation layer. |

## Implemented In This Pass

- Added first-class persisted DB connection-reference workflows:
  - `GET /api/v1/database-connection-references`
  - `POST /api/v1/database-connection-references`
  - `GET /api/v1/database-connection-references/{id}`
  - `PATCH /api/v1/database-connection-references/{id}`
- Added first-class persisted DB connection-health workflows:
  - `POST /api/v1/database-connection-references/{id}/test`
  - `GET /api/v1/database-connection-tests`
  - `GET /api/v1/database-connection-tests/{id}`
- Added persisted runtime DB validation execution workflows:
  - `POST /api/v1/database-validation-checks/{id}/execute`
  - `GET /api/v1/database-validation-executions`
  - `GET /api/v1/database-validation-executions/{id}`
- Added a safe first runtime DB execution slice:
  - explicit connection source posture (`env_dsn`, env-bound executable `secret_ref_dsn`, and truthfully unresolved unbound secret refs) instead of plaintext DSNs
  - persisted connection-health tests with redacted summaries and failure classes
  - runtime-only secret-safe connection resolution rather than persisting resolved DSNs
  - Postgres-only read-only execution
  - structured runtime checks for existence, migration markers, and row-count assertions
  - persisted execution evidence and redacted connection metadata
- Extended release analysis, rollout gating, and evidence packs so runtime DB execution results now drive:
  - required-check posture
  - connection-health posture
  - blocking/non-blocking rollout readiness
  - DB-aware rollback guidance
  - DB execution evidence export
- Added first-class persisted release-bundle workflows:
  - `GET /api/v1/releases`
  - `POST /api/v1/releases`
  - `GET /api/v1/releases/{id}`
  - `PATCH /api/v1/releases/{id}`
- Added first-class persisted config-set workflows:
  - `GET /api/v1/config-sets`
  - `POST /api/v1/config-sets`
  - `GET /api/v1/config-sets/{id}`
  - `PATCH /api/v1/config-sets/{id}`
- Extended rollout executions to optionally link a release bundle and carry that link into evidence packs and incident reasoning.
- Added deterministic release analysis that now powers:
  - readiness review
  - combined risk and blast radius
  - dependency-aware release planning
  - database findings
  - change-window warnings
  - rollback guidance
  - ops-assistant summaries
  - team-memory insights
  - communication drafts
- Added web UI and CLI support for config sets, release bundles, release analysis, and linked rollout execution creation.
- Fixed the rollout route so it now bundles catalog context directly instead of depending on another route having loaded it first.

## What This Does Not Claim

- This repository still does not have a real secret vault, general-purpose database connector catalog, migration runner, or learned AI copilot.
- `secret_ref_dsn` connection refs are executable only through the current env-bound `secret_ref_env` path; there is still no secret-manager-backed resolver, vault UX, or broader secret-runtime system.
- Many roadmap items are now materially further along than before, but several remain partial rather than complete.
- Strict release readiness is still blocked by external proof classification until a `customer_environment` or `hosted_saas` live-proof artifact is captured.
