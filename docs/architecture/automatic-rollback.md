# Automatic Rollback

Automatic rollback is now a first-class deterministic control-plane capability.

## Model

The control plane resolves an effective rollback policy for each rollout execution:

1. persisted scope-specific rollback policy if one matches
2. otherwise a deterministic built-in fallback policy

Effective policy resolution currently prefers the most specific enabled policy by:

1. service + environment
2. service
3. environment
4. project
5. organization

Within the same specificity level, higher `priority` wins, then newer records win.

## Decision Inputs

The verification engine evaluates:

- orchestrator backend status
- latest normalized signal snapshot
- persisted rollback policy thresholds
- prior verification failure count
- rollout execution context

The control path remains deterministic. Python is not used to decide transition legality.

## Decision Outputs

The verification engine can emit:

- `verified`
- `manual_review_required`
- `pause`
- `rollback`
- `failed`

## Execution

When the control loop records an automated rollback decision:

1. the verification result is persisted
2. the rollout execution status is updated to `rolled_back`
3. the provider rollback action is issued if needed
4. runtime state, status events, and audit records are persisted

## Current Boundary

Fully verified today:

- simulated backend rollback
- persisted rollback policies
- API, CLI, and web visibility
- status history and audit coverage

Near-real but not cluster-proven:

- Kubernetes HTTP-backed rollback via configured endpoints or image patch target
