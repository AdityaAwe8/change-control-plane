# Automatic Webhook Registration

## Goal

SCM onboarding is not credible if every customer must manually wire webhooks before the platform can see change events. This milestone adds automatic webhook registration and repair for the supported SCM providers in this repo.

## Current Provider Coverage

- GitHub: organization-scoped webhook ensure/update
- GitLab: group-scoped webhook ensure/update

## Runtime Model

- integration metadata references a webhook secret env var
- integration update, connection test, sync, or explicit webhook-sync can attempt registration
- registration status is persisted per integration instance
- webhook deliveries update delivery-health state

## Operator Visibility

Integration pages now show:

- registration status
- delivery health
- callback URL
- scope identifier
- last registration
- last delivery
- latest error

## Honest Limits

- Automatic registration still depends on valid scope metadata and secret references.
- The platform does not yet generate provider webhooks or secrets end to end with zero operator input.
- Full webhook deletion and richer provider-native diagnostics are still future work.
