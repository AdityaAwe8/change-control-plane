# ADR 0010: Machine Actors Use Persisted Service Accounts And Hashed API Tokens

## Status

Accepted

## Context

The platform needed non-human actors for rollout execution, integration ingestion, and future workflow automation. Reusing human dev-session tokens would blur audit boundaries and create poor long-term security posture.

## Decision

- Add a persisted `service_accounts` model scoped to organizations.
- Issue opaque API tokens for service accounts.
- Store only token prefix and token hash.
- Resolve service-account identities separately from signed human session tokens.
- Restrict service-account and token management to human organization administrators.

## Consequences

- The platform gets clean actor separation and better audit fidelity.
- Credential rotation and revocation become explicit control-plane operations.
- Project-scoped machine permissions remain a follow-on capability.
