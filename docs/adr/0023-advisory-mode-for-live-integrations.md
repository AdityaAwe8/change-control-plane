# ADR 0023: Advisory Mode For Live Integrations

## Status

Accepted

## Decision

When a live backend integration is not explicitly in `active_control`, the runtime should observe and record advisory recommendations instead of executing provider control actions.

## Rationale

- brownfield business adoption is safer in read-only or advisory mode first
- the platform needs honest evidence before it becomes an active deployment controller

## Consequences

- reconcile for live integrations can now be safe-by-default
- advisory verification decisions are recorded without mutating rollout state into executed control outcomes
- richer UX for advisory recommendations is still needed
