# Security Architecture

Security is a first-class property of the platform, not a bolt-on.

## Foundational Controls

- tenant-aware models
- explicit ownership and organization boundaries
- policy hooks around critical actions
- audit recording for control-plane mutations and decisions
- hashed service-account token storage with one-time token display
- token revocation and expiry handling
- structured error handling that avoids leaking sensitive values
- configuration layering that keeps secrets out of source

## Future Security Roadmap

- SSO and SAML
- service accounts and API keys
- privileged action elevation workflows
- artifact provenance and attestations
- SBOM ingestion and gating
- stronger production access governance

## Current Notes

The platform still uses development-oriented human authentication, but it now has real persisted machine credentials, RBAC enforcement, and rollout control surfaces. This is intentionally practical rather than enterprise-complete.
