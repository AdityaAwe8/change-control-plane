# ADR 0034: SCM Webhook Normalization Strategy

## Status

Accepted

## Decision

Normalize GitHub and GitLab webhook deliveries into shared SCM change records before creating mapped `ChangeSet` entities.

## Why

- the operator should not need separate GitHub and GitLab change mental models
- the control plane already has a strong shared `ChangeSet` abstraction
- provider-specific facts can still be preserved in metadata without fragmenting product logic

## Consequences

- push, pull-request, and merge-request style events now land in the same ingest model
- provider-specific headers, webhook security, and file-enrichment steps remain provider-specific
- tag/release events can be recorded even when they do not yet produce deeper release lifecycle behavior
