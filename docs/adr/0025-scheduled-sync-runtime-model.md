# ADR 0025: Scheduled Sync Runtime Model

## Status

Accepted

## Decision

Use a lightweight persisted schedule model on each integration instance and execute due syncs from the existing worker loop.

## Why

- preserves the current architecture
- avoids introducing a separate distributed scheduler prematurely
- is enough for an advisory pilot that needs continuous refresh rather than button-driven syncs

## Consequences

- sync cadence, freshness, and retries are visible on the integration itself
- due sync claiming remains simple and database-backed
- the model is suitable for a careful pilot, not yet for large multi-worker control-plane scale
