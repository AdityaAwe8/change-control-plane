# ADR 0015: Live Runtime Control Loop

## Status

Accepted

## Context

The platform already persisted rollout plans and executions, but it did not yet reconcile desired execution state against a backend or runtime verification evidence. We needed a production-leaning control loop without prematurely introducing a distributed workflow system.

## Decision

We added a deterministic live runtime control loop with these properties:

- rollout executions store both desired state and backend runtime state
- the worker claims and reconciles executions through the app layer
- orchestrator interactions go through a provider registry
- runtime signal collection goes through a signal-provider registry
- verification decisions remain deterministic and auditable
- automated actions run as explicit actors and preserve RBAC boundaries

## Consequences

- the simulated backend path is now the live verified execution path for local development and CI
- manual actions remain desired-state mutations; reconciliation converges backend state safely afterward
- external provider clients can be added without rewriting the execution model
- the current claim strategy is adequate for the modular-monolith worker, but future distributed deployment will need stronger leases
