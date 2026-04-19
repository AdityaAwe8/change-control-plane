# Advisory Mode Status

Status legend used here:

- `fully live`
- `near-live`
- `simulated only`
- `descriptor only`
- `partially implemented`
- `missing`

## Current Status

| Advisory Surface | Status | Reality |
| --- | --- | --- |
| Integration advisory flag | `fully live` | Integrations now persist advisory vs `active_control` mode and an explicit `control_enabled` flag. |
| Backend reconcile safety | `fully live` | For non-simulated live backends, advisory mode now downgrades provider actions into observation-only sync behavior instead of executing submit/pause/resume/rollback calls. |
| Verification evidence in advisory mode | `partially implemented` | Advisory reconcile can now record advisory verification decisions without changing the desired rollout state. The UI still shows the raw decision strings rather than a richer recommendation component. |
| Web visibility | `partially implemented` | The integrations page exposes mode, control enablement, health, and sync history. Broader product-wide advisory banners are still limited. |
| CLI visibility | `partially implemented` | The CLI can show and update advisory settings through `integrations show` and `integrations update`. There is not yet a dedicated `advisory-mode show/set` top-level command. |
| Audit/status evidence | `fully live` | Integration tests, syncs, and GitHub webhook ingest now create persisted sync-run evidence and audit/status events. |
| Manual operator protections | `partially implemented` | Automated runtime control is blocked in advisory mode, but manual rollout state transitions can still be recorded through existing operator routes. More explicit manual-control UX guardrails are still needed. |

## Honest Summary

Advisory mode is now materially real for the live-style runtime path:

- the platform can observe backend state
- ingest runtime signals
- record verification evidence
- store advisory recommendations
- avoid mutating the external backend during reconcile

What still needs hardening:

- product-wide wording that distinguishes “recommended rollback” from “rollback executed”
- clearer manual-action restrictions for live integrations left in advisory mode
- more explicit end-user explanation on rollout detail pages when advisory decisions were recorded instead of control actions
