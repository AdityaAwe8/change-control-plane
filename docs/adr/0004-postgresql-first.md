# ADR 0004: PostgreSQL as the First Primary Store

## Status

Accepted

## Decision

Use PostgreSQL as the initial system of record.

## Rationale

- strong transactional integrity
- flexible JSONB metadata
- mature indexing and operational tooling
- practical support for graph-friendly relationship modeling

## Consequences

- graph-specific optimizations may need projection layers later
- repository interfaces should isolate storage details from domain logic
