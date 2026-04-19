# ADR 0018: Status Event Model

## Status

Accepted

## Decision

Introduce a dedicated `status_events` model for operator-facing operational history instead of overloading audit events.

## Rationale

- audit and operational history serve different purposes
- rollout timelines need scope, source, state transition, and search fields not present in audit rows
- dashboards and CLI history queries need a stable operational feed

## Consequences

- audit remains the compliance-oriented record
- status events become the searchable control-plane timeline
- new subsystems can emit status events without changing audit semantics
