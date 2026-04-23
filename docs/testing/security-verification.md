# Security Verification

This document records the currently proven security posture and the remaining security risk areas.

## Verified In This Repository

### Authentication

- password sign-up and sign-in issue persisted HttpOnly browser sessions through automated tests
- dev bootstrap login issues persisted HttpOnly browser sessions through automated tests
- enterprise OIDC start/callback/session flows are exercised through automated tests where the harness path exists
- cookie-backed browser sessions are reloaded, expired, revoked, and logged out through automated tests
- org admins can list and revoke persisted browser sessions, including current-session cookie clearing, through automated tests
- service-account tokens are hashed at rest and lifecycle-tested through issue, use, and revoke paths
- revoked service-account tokens are denied

### Authorization and Tenancy

- unauthenticated access to protected routes returns `401`
- cross-tenant organization scope overrides are denied with `403`
- org members are denied project creation
- org members are denied service archive
- org members are denied browser-session administration
- org members are denied rollback-policy management
- cross-tenant signal-snapshot writes are denied
- cross-tenant status-event reads are denied

### Request Handling

- JSON decoding rejects unknown fields
- JSON decoding rejects trailing payloads
- browser cookie mutations are protected by explicit origin validation in addition to SameSite-Lax/HttpOnly cookies
- development browser use is protected by an explicit CORS policy rather than implicit failure
- disallowed origins are rejected when explicit origin allowlists are configured
- live-proof preflight and operator-checklist generation now expose required secret env names plus exact callback/webhook route expectations without echoing the secret values themselves

### Sensitive Surfaces With Audit Coverage

- service-account creation
- token issue, revoke, and rotate
- rollout lifecycle mutations
- verification recording
- rollback-policy mutations
- integration graph ingest

## Security Limits Still Present

- browser sessions are now persisted server-side and cookie-backed, but enterprise IAM breadth is still incomplete
- SCIM, SAML, deeper role/session administration, and broader role-mapping remain missing
- CLI and machine auth still rely on bearer tokens rather than a broader enterprise device/session model
- provider credential handling for Kubernetes and Prometheus is still minimal and metadata-driven
- the browser app still uses one-time token alerts for issued machine tokens rather than a purpose-built secure reveal component
- not every route has an explicit denial-path test yet
- release/proof artifacts and the generated live-proof operator checklist now have a central secret-safety scan, and the checklist can safely render exact callback/webhook URL patterns because it never includes secret values, but status-event metadata discipline still depends on callers; there is no universal runtime metadata scrubber yet

## Current Security Rating By Area

| Area | Status | Notes |
| --- | --- | --- |
| Human auth | partially verified | Password, dev bootstrap, OIDC callback/session attribution, cookie-backed browser sessions, and admin-scoped browser-session revocation are real; enterprise IAM breadth is still incomplete |
| Machine auth | verified | Hashed tokens plus issue/revoke/rotate/deactivate lifecycle are proven |
| RBAC | partially verified | Major high-risk mutations covered; edge coverage still incomplete |
| Multitenancy | partially verified | Core denial paths proven; not every read endpoint has dedicated coverage |
| Browser auth storage | verified | Persisted server-side sessions plus HttpOnly cookies are proven for browser use |
| Provider credential handling | partial | Near-real client seams exist, preflight/checklist output is now secret-safe and explicit about missing envs, but secret handling remains metadata-driven and intentionally narrow |
