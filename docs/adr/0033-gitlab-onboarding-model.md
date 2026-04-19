# ADR 0033: GitLab Onboarding Model

## Status

Accepted

## Decision

Adopt a product-shaped token-based GitLab onboarding path using `access_token_env` and `webhook_secret_env` on the existing integration-instance model.

## Why

- it provides a real and secure first GitLab path without waiting for OAuth or GitLab App work
- it fits the current integration-instance and secret-reference architecture
- it keeps the product honest about what is supported now versus what is future work

## Consequences

- GitLab onboarding is now materially real for pilots
- onboarding state can move to `configured` from saved token-based setup
- OAuth, automatic webhook registration, and GitLab App-style onboarding remain future work
