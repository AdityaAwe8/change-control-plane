# ADR 0026: Discovered Resource Model

## Status

Accepted

## Decision

Persist runtime-discovered assets as first-class `discovered_resources` records instead of burying them inside sync-run details or graph edges only.

## Why

- operators need a reviewable mapping surface for workloads and signal targets
- coverage summaries need a stable source of truth
- runtime discovery should survive beyond a single sync-run payload

## Consequences

- runtime discovery can now be listed, filtered, and mapped explicitly
- repository discovery and runtime discovery remain separate but compatible surfaces
- the status/review model is intentionally compact and may need future refinement
