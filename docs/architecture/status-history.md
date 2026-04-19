# Status History

The platform now distinguishes between:

- audit events: immutable compliance-oriented action records
- status events: operator-facing operational history

## Why A Dedicated Status Event Model Exists

Audit rows alone were not sufficient for:

- rollout timelines
- rollback visibility
- provider sync visibility
- project-space operational dashboards
- search by service, environment, rollout execution, or rollback activity

The `status_events` model adds:

- operational scope fields
- previous and new state
- source/provider labeling
- automated vs manual attribution
- summary and explanation
- search and filtering support

## Query Model

Status history can now be filtered by:

- organization
- project
- service
- environment
- rollout execution
- resource type and resource id
- actor type and actor id
- source
- rollback-only mode
- time range
- text search

## Surfaces

Status history is now visible through:

- `GET /api/v1/status-events`
- `GET /api/v1/rollout-executions/{id}/timeline`
- project, service, and environment scoped status-event endpoints
- the web operational status dashboard
- CLI status-history commands
