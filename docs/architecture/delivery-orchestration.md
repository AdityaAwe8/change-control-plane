# Delivery Orchestration

ChangeControlPlane is not trying to replace every pipeline on day one. It provides a governance layer above them.

## Control Responsibilities

- decide recommended rollout strategy
- determine approval needs
- restrict change windows when necessary
- enforce guardrails for critical or regulated changes
- coordinate rollback and verification requirements

## Adoption Modes

- read-only ingestion
- advisory mode
- approval and policy mode
- rollout governance mode
- full orchestration mode

## Phase 1

Phase 1 includes rollout planning primitives and policy evaluation hooks. External deployment execution remains integrated through adapters and future workflow layers.

## Phase 2+

- Temporal-backed workflow coordination
- staged regional, cohort, and tenant rollouts
- freeze windows and release trains
- deployment verification feedback loops
