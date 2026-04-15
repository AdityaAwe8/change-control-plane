# Machine Actors

## Purpose

ChangeControlPlane now supports machine actors through persisted service accounts and hashed API tokens. This gives automation a first-class identity without collapsing human and non-human authentication into the same model.

## Model

- Human users continue to authenticate through signed bearer tokens in development mode.
- Service accounts are persisted, organization-scoped actors with an explicit role.
- API tokens are issued for service accounts, hashed before persistence, and only returned once at creation or rotation time.
- The request identity resolver distinguishes signed human tokens from opaque service-account tokens.

## Authorization

- Service accounts currently carry organization-scoped roles.
- Least privilege is the default. New service accounts default to `viewer` unless the creator selects a broader role.
- Service-account and token administration is restricted to human `org_admin` actors.
- Machine actors can execute read and operational flows according to their role, but they cannot mint or revoke their own credentials.

## Security Controls

- Raw API tokens are never stored after issuance.
- Token lookups use a public prefix plus a stored hash.
- Revoked and expired tokens are rejected during identity resolution.
- Token use updates `last_used_at` for observability without exposing the raw secret.

## Next Steps

- Project-scoped service-account memberships
- Token expiry policy packs
- Token usage anomaly detection
- External identity and workload identity federation
