# Security Architecture

Security is a first-class property of the platform, not a bolt-on.

## Foundational Controls

- tenant-aware models
- explicit ownership and organization boundaries
- password auth plus server-backed HttpOnly browser sessions
- OIDC provider foundation with signed callback state and session attribution
- policy hooks around critical actions
- audit recording for control-plane mutations and decisions
- hashed service-account token storage with one-time token display
- token revocation and expiry handling
- cookie-authenticated mutation protection through SameSite=Lax plus `Origin` / `Referer` validation
- structured error handling that avoids leaking sensitive values
- configuration layering that keeps secrets out of source
- secret-safe integration, config-set, database-governance, evidence-pack, and proof-artifact conventions that store env/ref names rather than resolved values

## Future Security Roadmap

- broader enterprise IAM beyond the current OIDC foundation, including SCIM and SAML
- privileged action elevation workflows
- artifact provenance and attestations
- SBOM ingestion and gating
- stronger production access governance
- real secret-vault integration and rotation UX

## Current Notes

The platform now has real password auth, a browser-session model, OIDC foundations, persisted machine credentials, RBAC enforcement, and rollout control surfaces. It is intentionally practical rather than enterprise-complete: SCIM, SAML, global logout propagation, device posture, and real secret-vault workflows remain outside the current implementation.
