# ADR 0001: Monorepo First

## Status

Accepted

## Decision

Use a monorepo for the API, worker, CLI, frontend, shared types, docs, deployment assets, and future analytics packages.

## Rationale

- cross-cutting change governance requires shared models and contracts
- docs, OpenAPI, CLI, and UI evolve together
- one repo keeps architectural intent visible while the product surface is still forming
- internal boundaries can still be made explicit without operational microservice overhead

## Consequences

- stronger shared context
- simpler early-stage coordination
- need disciplined boundaries to avoid a blob
