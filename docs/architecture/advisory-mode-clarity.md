# Advisory-Mode Clarity Audit

This document records the pilot-readiness audit for recommendation-versus-execution clarity.

## Audit Outcome

| Surface | Prior State | Current State | Classification |
| --- | --- | --- | --- |
| Reconcile backend behavior | advisory safety existed in code, but suppression was mostly implicit in provider sync metadata | live non-simulated advisory reconciles now persist explicit `last_action_disposition=suppressed`, `recommended_action`, `control_mode`, and `control_rationale` fields | materially clarified |
| Manual rollout controls | pause, continue, and rollback could still look like valid control actions on advisory live backends | manual pause, continue/resume, and rollback now fail fast with a validation error that explicitly says advisory mode blocks the action | fixed bug |
| Verification recording | advisory recommendations were recorded, but the distinction from control-capable decisions was not obvious in every response | verification results now expose `action_state` and `control_mode`, and advisory decisions remain prefixed as `advisory_*` | materially clarified |
| Rollout detail API | detail payload showed backend state and latest decision, but not whether a provider action was executed or suppressed | runtime summary now exposes `advisory_only`, `control_mode`, `control_enabled`, `recommended_action`, `last_provider_action`, `last_action_disposition`, and `control_rationale` | fixed bug |
| Status timeline | status summaries could require inference to understand whether an action was observed, suppressed, or executed | explicit `rollout.execution.action_suppressed` status events now record advisory-only suppression without creating a misleading audit action | fixed bug |
| Audit trail | recommendation-only provider suppression could have been confused with an executed machine action | suppressed provider actions now record status events without audit mutation records; executed provider actions still produce audit plus status evidence | materially clarified |
| Web rollout UI | latest decision and provider state were visible, but advisory execution could still be misread as active control | advisory banner, control-mode cards, provider-action card, and disabled manual control affordances now make recommendation-only mode explicit | fixed bug |
| Web integration UI | onboarding exposed mode toggles, but did not explain clearly what the current mode actually meant | integration panels now distinguish “Advisory only” from “Active control” and label read-only test/sync actions accordingly | materially clarified |
| CLI | CLI remains JSON-first and did not get a separate prose mode | clearer run summaries/details now flow through the API, but the CLI still depends on operators reading structured JSON fields | partially improved |
| Docs | advisory docs admitted ambiguity, but there was no dedicated clarity audit | this document plus the updated advisory/openapi docs now describe suppression and recommendation behavior directly | fixed gap |

## Ambiguities Classified As Bugs

The following issues were treated as product bugs during this milestone:

1. A pilot operator could trigger manual pause or rollback against an advisory live backend and reasonably think the external system had been mutated.
2. Rollout detail did not expose an explicit “suppressed vs executed” provider-action disposition.
3. Status history did not emit a dedicated advisory-suppression event for provider actions.
4. Integration and rollout copy did not consistently say “observe and recommend only”.

## Current Semantics

- `advisory_*` verification decisions mean the control plane recorded a recommendation only.
- `rollout.execution.action_suppressed` means an external provider action was recommended but not executed because active control was disabled.
- `rollout.execution.action_executed` means the control plane actually issued a provider mutation.
- Manual pause, continue/resume, and rollback are blocked for live advisory backends.
- Manual verification can still be recorded in advisory mode, but pause/rollback/failure outcomes are rewritten into advisory recommendations.

## What Is Still Not Perfect

- The CLI is still structured-data heavy rather than explicitly narrative.
- Older status and audit event types outside the rollout/advisory path still rely on summary text more than a fully normalized effect taxonomy.
- Browser rendering does not yet offer a dedicated diff view between “recommended”, “suppressed”, and “executed” events beyond table labels and banners.
