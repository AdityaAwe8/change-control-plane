# Status History Status

This document tracks whether the control plane has a real operational history model rather than scattered audit rows.

Status legend:

- `live_and_verified`
- `partially_implemented`
- `missing`

## Current Status

| Area | Status | Reality |
| --- | --- | --- |
| Canonical status event model | `live_and_verified` | A dedicated `status_events` table now exists with operational scope, actor, source, state transition, severity, and explanation fields. |
| Rollout timeline support | `live_and_verified` | Rollout execution detail now includes a canonical status timeline in addition to the underlying audit trail. |
| Search and filters | `live_and_verified` | Status history is queryable by project, service, environment, rollout execution, resource, actor, source, rollback-only mode, time range, and text search. |
| Tenant isolation | `live_and_verified` | Status queries are organization-scoped and flow through the same authenticated API boundary as the rest of the control plane. |
| Dashboard visibility | `partially_implemented` | The web console now exposes a real operational status dashboard and searchable timeline feed, but it is still a lightweight client-side operator console rather than a streaming incident workspace. |
| CLI visibility | `live_and_verified` | The CLI can list status history and rollout timelines, including rollback-focused filtering. |
| Coverage breadth | `partially_implemented` | The status stream covers runtime, verification, rollout, token, and mutation events, but not every future subsystem emits first-class status events yet. |

## Reality Check

The status history model is now materially different from the audit trail:

- audit events remain the immutable compliance-oriented record
- status events now provide the operator-facing operational feed
- rollout detail and dashboard pages use status events for timeline visibility

The main remaining gap is breadth, not architecture. More subsystems can now adopt the same status-event pattern without reworking the data model.
