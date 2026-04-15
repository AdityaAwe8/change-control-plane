# Authentication And Authorization Model

## Authentication

Phase 2 uses a practical development-ready authentication model:

- signed bearer tokens
- opaque service-account API tokens
- persisted users in PostgreSQL
- persisted service accounts and token records in PostgreSQL
- dev bootstrap login endpoint at `POST /api/v1/auth/dev/login`
- request identity loaded on each authenticated API call

The token format is HMAC-signed and intentionally simple. This keeps the early platform dependency-light while preserving a clean seam for future SSO, OIDC, or session-backed models.

Service-account tokens are different:

- raw token only returned on issue or rotation
- token prefix plus token hash persisted
- revocation and expiry enforced during identity resolution
- human and machine authentication paths remain separate in code

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
