# Integrations

The integration layer is intentionally adapter-driven.

## Initial Adapters

- GitHub
- GitLab
- Kubernetes
- Prometheus

## Adapter Responsibilities

- expose capabilities and health state
- normalize external metadata into internal domain concepts
- enrich repositories, ownership evidence, discovered resources, and graph relationships in persisted storage
- preserve source-of-truth attribution
- support progressive adoption without forcing replacement of existing tools

## Design Principles

- integrate cleanly before attempting deep control
- isolate external APIs behind narrow internal contracts
- keep sync, ingestion, and action execution separate
- make adapters observable and auditable
- keep ingestion idempotent with stable identifiers and source markers

## Current Reality

The current implementation details and honesty docs now live in:

- [live-business-integration-status.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/live-business-integration-status.md)
- [github-integration.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/github-integration.md)
- [kubernetes-live-integration.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/kubernetes-live-integration.md)
- [prometheus-live-integration.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/prometheus-live-integration.md)
- [advisory-mode.md](/Users/aditya/Documents/ChangeControlPlane/docs/architecture/advisory-mode.md)
