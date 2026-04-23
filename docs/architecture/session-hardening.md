# Session Hardening

This milestone replaces the older browser-token storage path with a server-backed browser-session foundation while preserving bearer/API tokens for CLI and machine-auth clients.

## Changes

- Password sign-in, dev login, and enterprise OIDC callback paths now issue HttpOnly SameSite=Lax browser-session cookies.
- Browser sessions are persisted server-side and include issue/expiry, revocation, auth-method, provider, user-agent, and IP-attribution metadata.
- Logout revokes the active browser session and clears the cookie.
- Expired or revoked browser-session cookies are rejected server-side.
- Organization administrators can list and revoke persisted browser sessions for the active organization.
- Cookie-authenticated mutations preserve origin protections through same-origin/SameSite behavior plus `Origin`/`Referer` validation.
- Session payloads now expose issue and expiry timestamps.
- Return URLs are normalized to safer relative or allowed local/configured origins.

## What This Does Not Claim

- CLI and machine clients still use bearer/API tokens by design.
- This is not SAML, SCIM, or a complete enterprise IAM platform.
- Global logout propagation to upstream identity providers, device posture, and richer conditional-access controls still need future work.
