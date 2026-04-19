# GitHub App Or OAuth Onboarding

## Current Supported Live Path

The product now materially supports a GitHub App installation-style flow:

1. create or select a GitHub integration instance
2. configure `auth_strategy=github_app`
3. save `app_id`, `app_slug`, `private_key_env`, optional `owner`, and `webhook_secret_env`
4. call the onboarding start route
5. redirect the operator to the generated GitHub App install URL
6. accept the callback with `state` and `installation_id`
7. persist the installation metadata and mint installation tokens dynamically for later sync/test calls

## Legacy Path

The PAT path still exists:

- `auth_strategy=personal_access_token`
- `access_token_env`
- `webhook_secret_env`

This remains supported for compatibility, not as the preferred reference onboarding experience.

## Honest Gaps

- No full OAuth user-consent flow yet
- No automatic webhook registration handshake yet
- No marketplace-grade install UX
- No encrypted secret vault inside the product; env references are still used

## Why This Still Matters

Even with those gaps, this is materially closer to real product onboarding than asking operators to start with a PAT and a webhook secret alone.
