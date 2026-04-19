# ADR 0014: Authenticated Worker Control Loop

## Status

Accepted

## Context

The platform already persists rollout plans, rollout executions, verification results, machine actors, and audit records. A worker that only emits heartbeats does not satisfy the control-plane goal of governing execution readiness.

## Decision

Implement the worker as an authenticated machine actor that uses the same application, authorization, and audit boundaries as API callers.

Current worker behavior:

- authenticate with a service-account token
- list rollout executions within the active organization scope
- auto-start `planned` and `approved` executions when auto-advance is enabled
- auto-complete `verified` executions when auto-advance is enabled

## Consequences

Positive:

- no privileged bypass path around auth or audit
- same RBAC model applies to automated actors
- establishes a durable seam for richer workflow integration later

Tradeoffs:

- worker bootstrap now requires token provisioning
- current control loop is intentionally narrow
- external deploy-system integration still needs to be built
