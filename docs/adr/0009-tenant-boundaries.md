# ADR 0009: Organization-Scoped Tenant Boundaries

## Status

Accepted

## Decision

Use organizations as the primary tenant boundary and require project, service, environment, change, risk, rollout, audit, and integration records to stay within organization scope.

## Rationale

- consistent mental model for customers
- clear row ownership and API filtering behavior
- enterprise-friendly boundary for future SaaS and self-hosted packaging
