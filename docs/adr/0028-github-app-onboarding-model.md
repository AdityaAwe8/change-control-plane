# ADR 0028: GitHub App Onboarding Model

## Status

Accepted

## Decision

Add a GitHub App installation-style onboarding flow with signed state, callback persistence, and dynamic installation-token minting from `app_id`, `installation_id`, and `private_key_env`.

## Why

- moves the product beyond a PAT-only onboarding story
- keeps secrets as env references instead of persisting raw credentials
- fits the current architecture without introducing a full external identity broker

## Consequences

- GitHub App onboarding is now materially real
- PAT remains supported as a legacy path
- OAuth and marketplace-grade install UX remain future work
