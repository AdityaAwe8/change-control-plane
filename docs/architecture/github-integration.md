# GitHub Integration

## What Is Real Now

- GitHub App installation-style onboarding with signed state and callback persistence
- installation-token minting from GitHub App credentials stored as env references
- token-backed connection test against the GitHub API
- repository discovery through the GitHub API
- persisted repository records
- repository-to-service/environment mapping
- webhook endpoint with `X-Hub-Signature-256` validation
- webhook delivery id deduplication through persisted sync runs
- mapped push and PR webhook ingest into persisted change sets

## Current Configuration

The GitHub integration currently expects metadata such as:

- `api_base_url`
- `owner`
- `webhook_secret_env`
- `auth_strategy`

Depending on the auth strategy, it also expects:

- `access_token_env` for `personal_access_token`
- `app_id`, `app_slug`, `private_key_env`, and `installation_id` for `github_app`

Secrets are referenced by env var name instead of being stored inline in the database.

## Current Webhook Behavior

Supported webhook families:

- `push`
- `pull_request`
- `release`
- `workflow_run`

Push and PR events can create change sets when the repository has been mapped to a service and environment.

## Important Limitations

- no OAuth install flow yet
- no marketplace-grade GitHub App management surface yet
- no automatic webhook registration handshake yet
- no long-tail pagination for very large orgs
- no full review-state or merge-queue model
- unmapped repositories do not create change sets; they remain discoverable and map-able
