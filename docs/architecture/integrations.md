# Integrations

The integration layer is intentionally adapter-driven.

## Initial Adapters

- GitHub
- Kubernetes
- Slack
- Jira

## Adapter Responsibilities

- expose capabilities and health state
- normalize external metadata into internal domain concepts
- enrich repositories and graph relationships in persisted storage
- preserve source-of-truth attribution
- support progressive adoption without forcing replacement of existing tools

## Design Principles

- integrate cleanly before attempting deep control
- isolate external APIs behind narrow internal contracts
- keep sync, ingestion, and action execution separate
- make adapters observable and auditable
- keep ingestion idempotent with stable identifiers and source markers
