# ADR 0005: Event-Driven Core Interfaces

## Status

Accepted

## Decision

Introduce domain event abstractions from the start, even while initial delivery remains in-process.

## Rationale

- change governance crosses services, workflows, and time
- event publication is useful even before an external broker is required
- future projections, automation, and analytics depend on stable domain event contracts

## Consequences

- domain services publish meaningful events
- transport stays swappable
- consumers can remain loosely coupled
