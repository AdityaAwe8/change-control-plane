# Graph Enrichment

## Purpose

The system graph remains PostgreSQL-first, but it now supports persisted enrichment from integration ingestion so the digital twin evolves beyond manually created catalog rows.

## Persistence Model

Graph enrichment currently uses:

- `repositories` for source-control system records
- `graph_relationships` for typed, source-tagged edges

Each graph relationship records:

- organization scope
- optional project scope
- source integration
- relationship type
- source resource type and id
- target resource type and id
- status
- last observed timestamp

## Initial Relationship Types

- `service_repository`
- `service_environment`
- `service_dependency`
- `change_repository`
- `service_integration_source`

## Ingestion Behavior

- Ingestion runs through the application layer, not direct SQL.
- Every referenced resource is validated against the active tenant scope.
- Repository and relationship ids are generated deterministically to keep ingestion idempotent.
- Re-ingesting the same payload updates timestamps and metadata rather than creating duplicates.
- Integration records capture `last_synced_at` to surface recency.

## Why Not A Graph Database Yet

PostgreSQL already holds the authoritative system metadata and gives strong transactional behavior for control-plane writes. The relationship table preserves future portability without forcing premature infrastructure complexity.
