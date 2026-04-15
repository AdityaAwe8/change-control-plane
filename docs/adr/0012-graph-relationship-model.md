# ADR 0012: Persist Integration Enrichment In A Relational Graph-Relationship Table

## Status

Accepted

## Context

The digital twin required richer dependency and ownership edges, but moving to a dedicated graph database during Phase 1.x would increase operational complexity without clear product validation.

## Decision

- Keep PostgreSQL as the source of truth.
- Persist repositories explicitly.
- Persist typed relationships in a generic `graph_relationships` table.
- Use deterministic ids and integration-source markers for idempotent ingestion.

## Consequences

- The system graph becomes materially more useful immediately.
- Query shapes stay portable.
- A future graph database can be introduced as a read model or accelerator rather than a breaking rewrite.
