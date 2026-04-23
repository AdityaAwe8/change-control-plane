# Roadmap Status Matrix

This matrix is intentionally reality-based. It reflects what is implemented in this repository after the deployment evidence-pack work landed, not the full aspirational roadmap.

Status legend:

- `available_now`: implemented and usable in the product today
- `partial`: important building blocks exist, but the full workflow or product surface is not complete
- `missing`: not materially implemented beyond ideas, placeholders, or low-level types

## Requested Feature Status

| Feature | Status | What Exists Today | Biggest Remaining Gap |
| --- | --- | --- | --- |
| AI deployment readiness review | `partial` | Deterministic risk scoring, blast radius analysis, rollout recommendations, policy decisions, and plain-English explanations already exist in the change and rollout flows. | There is no LLM-backed question flow that inspects a deploy bundle and asks targeted readiness questions only when risk is present. |
| Release bundle selection | `missing` | `Release` types exist in the domain model. | There is no operator-facing release-bundle builder, dependency/conflict analysis, or release-composition workflow. |
| AI alerts and DevOps help | `missing` | Incidents, verification results, signal snapshots, and rollout timelines exist. | There is no AI incident assistant that correlates deploys, metrics, incidents, and changes into root-cause or rollback guidance. |
| Database integration and deployment queries | `missing` | Risk heuristics already detect schema and migration-looking changes. | There is no database connector, controlled validation-query workflow, migration evidence model, or DBA approval flow. |
| Configuration set management | `missing` | The local live-proof runner uses env-file based operator config outside the product runtime. | There is no first-class product model for per-environment config sets, overlays, drift, or config diff preview. |
| Secret reference management | `partial` | Integration setup uses env-var secret references such as `*_TOKEN_ENV`, `*_PRIVATE_KEY_ENV`, and `*_WEBHOOK_SECRET_ENV` instead of storing raw secrets inline. | This is not yet a full secret-manager-backed product workflow for release config, rotation, auditing, or runtime injection. |
| Secret encryption and decryption workflow | `missing` | The product avoids storing provider secrets inline when secret references are available. | There is no end-user secret vault, reveal flow, or encryption/decryption feature exposed as a product capability. |
| Change risk scoring | `available_now` | Risk assessments are persisted, exposed via API/CLI/web, and include score, level, explanation, blast radius, recommended approval level, rollout strategy, deployment window, and guardrails. | The model is deterministic and not yet learned from historical incidents or customer-specific telemetry. |
| Blast radius analysis | `available_now` | `BlastRadius` is part of the persisted risk assessment and already surfaces scope, services/resources impacted, production/customer-facing impact, and summary text. | It is not yet a richer visual multi-system graph with customer-surface overlays. |
| Policy-as-code deployment governance | `partial` | Persisted deterministic policies and policy decisions exist today for `risk_assessment` and `rollout_plan`, including advisory, block, and manual-review outcomes. | The engine is intentionally narrow and does not yet govern all deploy workflows, change windows, or external policy backends. |
| Deployment evidence pack | `available_now` | `GET /api/v1/rollout-executions/{id}/evidence-pack` now exports a single read model combining change, risk, plan, rollout detail, policy outcomes, incidents, repositories, discovered resources, graph relationships, integrations, and audit trail. The CLI now exposes `ccp rollout evidence --id <rollout_execution_id>`. | It is currently an API/CLI export surface rather than a dedicated browser workflow or downloadable signed artifact format. |
| Post-deploy verification | `partial` | Verification results, signal snapshots, runtime summaries, automated decisions, rollback policies, and signal-backed reconcile flows are already real. | The broader vision still lacks richer business-metric verification packs, synthetic-test integration, and more advanced explainability. |
| Rollback intelligence | `partial` | Rollback policies, verification decisions, advisory-only control suppression, and provider-backed rollout lifecycle evidence already exist. | There is no full cross-system rollback safety planner that reasons about migrations, feature flags, config reverts, and dependent-service ordering. |
| Drift detection and change provenance | `partial` | Repositories, discovered resources, graph relationships, mapping provenance, CODEOWNERS evidence, and runtime resource discovery already exist. | There is no full desired-vs-live drift engine or GitOps-grade reconciliation surface. |
| Release calendar and freeze windows | `missing` | Rollout plans can recommend deployment windows. | There is no persisted release calendar, maintenance-window engine, freeze enforcement, or collision detection workflow. |
| Dependency-aware release planning | `partial` | Service dependencies and graph relationships already exist as persisted entities. | There is no release planner that automatically sequences dependent service rollouts or enforces dependency order. |
| Team memory and knowledge graph | `partial` | The graph model, incidents, audit history, rollout history, and provenance edges provide a factual foundation. | There is no learned “team memory” layer that summarizes recurring patterns, risky tables, or habitual rollout practices over time. |
| Compliance and enterprise controls | `partial` | Audit events, RBAC, tenant scoping, OIDC identity providers, browser-session diagnostics/revocation, service accounts, and token lifecycle flows already exist. | Separation of duties, richer enterprise IAM breadth, exportable compliance packs, and exception workflows remain incomplete. |
| Communication automation | `missing` | The product has the underlying change, rollout, and incident data needed for release summaries. | There is no generated release-note, stakeholder-update, or handoff surface yet. |

## Implemented In This Change

- Added a first-class deployment evidence-pack API:
  - `GET /api/v1/rollout-executions/{id}/evidence-pack`
- Added a matching CLI command:
  - `ccp rollout evidence --id <rollout_execution_id>`
- The evidence pack currently exports:
  - rollout context: organization, project, service, environment
  - change context: change set, risk assessment, rollout plan
  - execution context: rollout detail, verification results, signal snapshots, rollback policy, runtime summary
  - governance context: policy decisions and approval posture summary
  - operational context: incidents, repositories, discovered resources, graph relationships, and audit trail

## What This Does Not Claim

- This change does not claim that the full AI, database, config-management, or enterprise roadmap is complete.
- It does claim that deployment evidence packs are now a real product surface backed by persisted data and verified routes.
