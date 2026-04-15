# ADR 0008: App-Layer RBAC Enforcement

## Status

Accepted

## Decision

Perform RBAC decisions in the app layer using request identity and resource context.

## Rationale

- HTTP handlers stay thin
- the same authorization model can be reused by workers and future workflows
- audit and denial behavior stay consistent
