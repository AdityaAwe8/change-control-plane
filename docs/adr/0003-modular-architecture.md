# ADR 0003: Modular Monolith with Extraction Seams

## Status

Accepted

## Decision

Start with a modular monolith instead of a large microservice fleet.

## Rationale

- the product needs strong domain cohesion more than early network boundaries
- many workflows span risk, rollout, audit, policies, and integrations
- extraction becomes easier when modules have already proven their contracts

## Consequences

- faster iteration
- lower operational overhead
- explicit responsibility to preserve package boundaries and repository interfaces
