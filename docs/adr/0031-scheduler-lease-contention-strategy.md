# ADR 0031: Scheduler Lease Contention Strategy

## Status

Accepted

## Decision

Keep the pragmatic persisted lease model for scheduled integration syncs, but add explicit contention-focused proof for duplicate-claim prevention.

## Why

- the current worker model is intentionally lightweight
- multi-instance support increases pressure on claim correctness
- stronger contention proof is more valuable right now than a full scheduler redesign

## Consequences

- the scheduler is better proven for advisory pilot use
- it is still not a fully hardened distributed scheduling system
