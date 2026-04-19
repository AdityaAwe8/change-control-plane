# ADR 0036: Enterprise Auth Foundation

## Decision

Add an organization-scoped OIDC-style identity-provider model with start and callback flows on top of the existing auth/session architecture.

## Why

- The repo needed a materially more credible enterprise sign-in story than dev bootstrap and password-only auth.
- OIDC provides a practical foundation for serious pilots without requiring a full IAM rebuild.

## Consequences

- Enterprise sessions now carry provider attribution.
- Identity links are persisted for future provisioning work.
- The design is future-ready for SCIM, but SCIM is not implemented in this ADR.
