# Authentication And Authorization Model

## Authentication

The current runtime uses a practical, product-shaped authentication model:

- password sign-up and sign-in for persisted users
- signed bearer tokens for CLI and API clients
- server-backed browser sessions represented by HttpOnly SameSite=Lax cookies
- opaque service-account API tokens
- persisted users, browser sessions, service accounts, and token records in PostgreSQL
- dev bootstrap login endpoint at `POST /api/v1/auth/dev/login`
- organization-scoped OIDC foundation with provider records, start/callback routes, signed state, and session attribution
- request identity loaded on each authenticated API call

The CLI bearer-token format is HMAC-signed and intentionally simple. Browser clients now use persisted opaque session cookies instead of long-lived bearer tokens in browser storage.

Browser sessions:

- are stored server-side by hashed opaque session token
- include issue/expiry, revocation, auth-method, provider, user-agent, and IP-attribution metadata
- are revoked by logout and by org-admin browser-session administration
- reject expired or revoked cookies during identity resolution
- keep bearer/API-token flows working for CLI and machine clients

Service-account tokens are different:

- raw token only returned on issue or rotation
- token prefix plus token hash persisted
- revocation and expiry enforced during identity resolution
- human and machine authentication paths remain separate in code

OIDC is a foundation, not a complete enterprise IAM suite:

- no SCIM provisioning or deprovisioning
- no SAML support
- no upstream global logout propagation
- no device posture or conditional-access model
- no broad role-mapping administration UI

## Authorization

Authorization is handled separately from authentication.

- authentication proves who the actor is
- authorization decides what that actor can do within tenant scope

The request context contains:

- actor id
- actor type
- active organization scope
- organization memberships
- project memberships
- organization role map for machine actors

The app layer uses `internal/auth/authorizer.go` to enforce write and read boundaries.
