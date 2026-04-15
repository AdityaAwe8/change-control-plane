# Event Model

The platform is event-aware from the beginning because change governance naturally spans multiple asynchronous decisions and systems.

## Core Domain Events

- `organization.created`
- `project.created`
- `team.created`
- `service.registered`
- `environment.created`
- `change.ingested`
- `risk.assessed`
- `rollout.planned`
- `policy.evaluated`
- `deployment.started`
- `deployment.verified`
- `deployment.paused`
- `deployment.rolled_back`
- `incident.created`
- `approval.requested`
- `approval.decided`
- `simulation.started`
- `simulation.completed`

## Phase 1 Implementation

Phase 1 uses an in-memory event bus abstraction so domain services can publish events without coupling to a concrete broker.

## Future Projection

The same interface can later be backed by:

- NATS
- Kafka
- Temporal signals
- event projections for analytics, graph views, and audit pipelines
