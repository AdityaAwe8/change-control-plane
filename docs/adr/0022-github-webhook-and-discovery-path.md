# ADR 0022: GitHub Webhook And Discovery Path

## Status

Accepted

## Decision

Use a token-backed GitHub API client for repository discovery and PR-file enrichment, plus a webhook endpoint with signature validation and delivery-id deduplication for change ingest.

## Rationale

- GitHub was the highest-value missing source-of-truth path for real business onboarding
- discovery and webhook ingest together are enough to make repository-backed change ingest materially real without building a full SCM product

## Consequences

- mapped repositories can now produce persisted change sets from GitHub events
- unmapped repositories are retained and visible for later mapping
- GitHub App installation and richer review state remain future work
