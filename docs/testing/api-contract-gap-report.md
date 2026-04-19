# API Contract Gap Report

This report captures the truth pass between the current HTTP handlers and `docs/api/openapi.yaml`.

## Routes Audited

- auth sign-up/sign-in/session
- enterprise auth provider list/start/callback and admin identity-provider routes
- integrations create/list/update/test/sync/sync-runs/webhook
- integration webhook-registration diagnostics
- GitHub onboarding start/callback
- repositories list/update
- outbox diagnostics
- rollout execution pause/resume/rollback/reconcile/verification/timeline
- status-event query surface

## Gaps Resolved In This Milestone

| Gap | Previous State | Current State |
| --- | --- | --- |
| Auth onboarding routes missing from OpenAPI | `POST /api/v1/auth/sign-up` and `POST /api/v1/auth/sign-in` existed in code but not in `openapi.yaml` | added to OpenAPI |
| Live integration routes missing from OpenAPI | test/sync/sync-runs/github-webhook/repository routes existed in code but not in `openapi.yaml` | added to OpenAPI |
| Integration create and GitHub App onboarding routes missing from OpenAPI | create/start/callback handlers existed without accurate contract coverage | added to OpenAPI with request/response schemas and auth notes |
| Integration list filters stale | handler supported multi-instance filters such as `instance_key`, `scope_type`, `auth_strategy`, `enabled`, and `search` that were not documented | query params added to OpenAPI |
| Advisory-mode semantics absent from contract text | rollout action and verification routes did not explain advisory suppression/rewrite behavior | route descriptions now call this out explicitly |
| Status-event query params incomplete | handler accepted many filters beyond the documented subset | OpenAPI now documents the wider filter surface used by the server |
| Integration update schema stale | `enabled`, `control_enabled`, and `metadata` were live but undocumented | schema updated |
| Repository update schema missing | route existed without request schema | schema added |
| Runtime/advisory summary fields missing | rollout detail now exposes advisory and provider-action disposition fields | schema added |
| Enterprise auth routes missing from OpenAPI | public provider list/start/callback and authenticated identity-provider admin handlers existed without contract coverage | added to OpenAPI with request/response schemas |
| Webhook-registration diagnostics undocumented | registration show/sync handlers and their persisted status model were live but absent from OpenAPI | added to OpenAPI with result schemas |
| Durable outbox diagnostics undocumented | authenticated outbox list handler existed without route or schema coverage | added to OpenAPI |

## Remaining Contract Weaknesses

| Area | Current Truth | Why It Still Matters |
| --- | --- | --- |
| Response envelopes | OpenAPI now covers the newest enterprise and integration routes more accurately, but many older endpoints still document success responses descriptively rather than with fully referenced envelope schemas | client generation and strict contract tooling would still need another pass |
| Some older CRUD routes | many stable CRUD endpoints still describe success responses without complete item/list schema references | acceptable for now, but not ideal for long-term SDK generation |
| Error-model specificity | shared `invalid_request`/`forbidden` behavior exists in code, but not every route enumerates concrete error responses | pilot operators still rely partly on runtime error text |

## Verification Added

- A contract spot-check test now fails if the OpenAPI file drops the enterprise auth, webhook-registration, outbox, multi-instance, or GitHub App onboarding routes and schemas that are now live.
- HTTP integration tests continue to prove the key advisory and integration endpoint behavior directly against handlers.
