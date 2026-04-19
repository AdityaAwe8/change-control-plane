# ADR 0032: Shared SCM Provider Model

## Status

Accepted

## Decision

Introduce a shared SCM abstraction for repository discovery, webhook-change normalization, and change-set ingest, with provider-specific onboarding and validation layered on top.

## Why

- GitLab should fit the same product model as GitHub instead of duplicating product logic
- repository mapping, change ingest, sync evidence, and coverage summaries are operator concepts, not provider-specific concepts
- this keeps future SCM providers possible without replacing the architecture

## Consequences

- GitHub and GitLab now share repository and change normalization paths
- provider-specific auth and webhook validation remain isolated where they truly differ
- provenance is improved but still not a full many-to-many source model
