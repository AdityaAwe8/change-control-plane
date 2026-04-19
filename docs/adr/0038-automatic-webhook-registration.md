# ADR 0038: Automatic Webhook Registration

## Decision

Model webhook registration as a persisted integration-scoped resource and automatically ensure supported GitHub and GitLab webhooks when configuration permits it.

## Why

- Manual webhook setup made SCM onboarding feel like operator plumbing rather than product onboarding.
- Registration health needed first-class visibility alongside sync health.

## Consequences

- GitHub and GitLab integrations can now self-register or repair supported webhooks.
- Missing secret references or unsupported scopes surface as honest `manual_required` or `error` states.
- The current model is pragmatic and not yet a full provider-native lifecycle manager.
