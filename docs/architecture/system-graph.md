# System Graph

The system graph is the long-term digital twin of the customer's software estate.

## Scope

The graph must eventually model:

- organizations, teams, users, and ownership
- repositories, services, APIs, databases, queues, and infrastructure stacks
- environments, deployments, rollout strategies, incidents, and cost baselines
- policies, compliance zones, data classifications, and secret references
- operational signals such as SLOs, alerts, and business metrics

## Initial Approach

Phase 1 uses PostgreSQL with normalized relational tables and graph-friendly foreign-key relationships. This gives us:

- transactional integrity
- practical operational simplicity
- flexible metadata via JSONB
- enough structure to answer meaningful dependency and ownership questions

The current operational slice now adds persisted graph enrichment through:

- repositories
- typed graph relationships
- integration-source attribution
- idempotent relationship upserts

## Future Evolution

If graph query needs materially exceed relational ergonomics, we can add:

- a graph-read projection
- an event-sourced topology stream
- specialized graph storage for advanced pathing and impact analysis

The core contract remains the same: the control plane consumes a coherent model of the system, regardless of backing store.
