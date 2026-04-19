# ADR 0035: SCM Attribution and Provenance Strategy

## Status

Accepted

## Decision

Keep repository attribution as a primary `source_integration_id` plus provider/source metadata hints, rather than introducing a full many-to-many provenance graph in this pass.

## Why

- GitLab support needed a stronger attribution story immediately, but not a complete source-ownership redesign
- the current repository and graph models already support a pragmatic primary-source approach
- this preserves architecture stability while improving mixed-provider visibility

## Consequences

- repositories and changes now show clearer SCM source attribution across GitHub and GitLab instances
- overlapping SCM scopes are more understandable than before
- deeper many-to-many provenance and conflict resolution remain future work
