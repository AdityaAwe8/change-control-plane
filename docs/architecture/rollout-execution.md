# Rollout Execution Lifecycle

## Purpose

Rollout plans are now paired with durable rollout execution records so the control plane can move from static governance into operational decision-making.

## Execution Entity

Each rollout execution binds together:

- organization
- project
- rollout plan
- change set
- service
- environment
- current status
- current step
- latest decision
- latest verification reference

## State Machine

Supported states:

- `planned`
- `awaiting_approval`
- `approved`
- `in_progress`
- `paused`
- `verified`
- `rolled_back`
- `failed`
- `completed`

Supported transition actions:

- `approve`
- `start`
- `pause`
- `continue`
- `complete`
- `rollback`
- `fail`

Invalid transitions are rejected in the application layer before persistence.

## Verification Coupling

Verification results are persisted separately and linked to rollout executions. A verification result records:

- outcome
- control decision
- technical signal summary
- business signal summary
- human-readable summary and explanation

Decision handling currently maps to rollout status updates:

- `continue` -> `verified`
- `pause` -> `paused`
- `manual_review_required` -> `paused`
- `rollback` -> `rolled_back`

## Audit Model

Every execution creation, transition, and verification write is recorded in audit events with actor identity and tenant scope.

## Future Extensions

- approval records linked directly to executions
- Temporal-backed long-running orchestration
- phased step completion tracking
- automated rollback hooks
