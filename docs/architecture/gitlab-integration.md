# GitLab Integration

GitLab is now a real SCM integration in the control plane, built on top of the shared SCM provider model.

## Current Onboarding Model

GitLab currently uses a product-shaped token-based setup:

- `kind = gitlab`
- `auth_strategy = personal_access_token`
- metadata includes:
  - `api_base_url`
  - `group` or namespace scope
  - `access_token_env`
  - `webhook_secret_env`

This is intentionally secure within the current product stage because the control plane stores env references instead of raw tokens.

When valid token-based configuration is saved, the integration onboarding state moves to `configured`.

## Connection Test

GitLab connection tests now do real provider work:

- `GET /user` resolves the authenticated principal
- `GET /groups/{group}` resolves the configured group or namespace when scope is set

The resulting sync-run evidence is visible in the product as connection details rather than a descriptor-only success badge.

## Repository Discovery

GitLab repository discovery supports:

- group-scoped project listing through `/groups/{group}/projects`
- fallback membership-based listing through `/projects?membership=true`
- normalization into the shared repository model with:
  - project id
  - path-with-namespace
  - default branch
  - visibility/private state
  - archive state
  - source integration attribution

Discovered projects are then mapped through the existing repository-to-service/environment flow.

## Webhook Ingest

GitLab webhook deliveries now land at:

- `POST /api/v1/integrations/{id}/webhooks/gitlab`

Current validation and normalization:

- validates `X-Gitlab-Token` against the configured secret env reference
- dedupes by `X-Gitlab-Event-UUID` or fallback request id
- records sync-run evidence for each accepted delivery
- normalizes supported events into the shared SCM change model

Supported event families today:

- `Push Hook`
- `Merge Request Hook`
- `Tag Push Hook`
- `Release Hook`

Merge requests additionally enrich changed files through:

- `GET /projects/{id}/merge_requests/{iid}/changes`

## Change Creation

GitLab changes now normalize into the existing control-plane `ChangeSet` model with:

- repository identity
- branch or tag
- commit SHA
- changed files
- reviewer and approver metadata where available
- labels
- issue-key extraction
- provider-specific metadata preserved in the change metadata blob

The product does not create a separate “GitLab change” concept. GitHub and GitLab now feed the same operator-facing change model.

## What Is Real

- token-based GitLab onboarding
- connection test
- project discovery
- repository attribution
- push and merge-request webhook ingest
- tag/release metadata ingest
- mapped change-set creation

## What Is Still Partial

- no GitLab OAuth flow
- no GitLab App-style onboarding
- no automatic webhook registration
- no full many-to-many provenance model for overlapping SCM sources
