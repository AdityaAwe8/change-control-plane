# Update And Delete Semantics

## Principle

The control plane now exposes broader mutation APIs, but destructive behavior is explicit and conservative.

## Current Semantics

- Organizations: update only
- Projects: update and archive
- Teams: update and archive
- Services: update and archive
- Environments: update and archive
- Integrations: update
- Service accounts: create, list, deactivate
- API tokens: issue, list, revoke, rotate
- Change sets: immutable after ingest
- Rollout plans: immutable after creation

## Why Archive Instead Of Delete

Projects, teams, services, and environments are referenced by change, risk, rollout, audit, and graph records. Hard deletion would weaken explainability and auditability. Archiving preserves lineage while removing the resource from active operation workflows.

## Audit Requirement

Every mutating action writes an audit event with:

- actor id
- actor type
- tenant scope
- action
- resource type
- resource id
- outcome
- details

## Follow-On Work

- list filtering that hides archived records by default on more surfaces
- explicit restore semantics
- retention and purge policy controls for self-hosted deployments
