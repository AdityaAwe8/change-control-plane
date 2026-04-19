# Enterprise Auth

## What Exists Now

The platform now supports a first-pass enterprise browser sign-in foundation with a hardened browser session path:

- organization-scoped identity-provider records
- OIDC-style issuer discovery or explicit endpoint overrides
- browser initiation route
- callback route
- signed state handling
- external subject to internal user linking
- allowed-domain filtering
- default-role mapping
- session attribution with `auth_method`, `auth_provider_id`, and `auth_provider`
- persisted browser-session records with expiry and revocation
- server-issued opaque browser session cookies
- HttpOnly cookie-backed session resolution on `/api/v1/auth/session`
- logout that revokes the active browser session and clears the cookie
- same-origin/SameSite-Lax browser mutation protection with explicit `Origin` / `Referer` validation
- audit-compatible auth events and timestamps on the provider

## Product Model

1. An org admin configures an identity provider.
2. The public auth page lists enabled providers.
3. The browser starts sign-in through `/api/v1/auth/providers/{id}/start`.
4. The callback reconciles the external identity into a user plus org membership.
5. The callback issues a persisted browser session and sets an HttpOnly cookie instead of redirecting a long-lived bearer token through the browser URL.
6. Subsequent browser requests resolve identity from that cookie, while CLI and machine clients continue to use bearer/API tokens.

## Session Model

- Browser sessions are stored server-side and referenced by an opaque cookie.
- Password sign-in, dev bootstrap login, and enterprise OIDC callback all land in the same browser-session model.
- Session expiry and explicit revocation are enforced server-side.
- Logout invalidates the current browser session immediately rather than only clearing client-side state.
- Active organization selection still rides through the authenticated session plus `X-CCP-Organization-ID`; this pass did not redesign organization switching.

## Security Notes

- Browser sessions are now intended to be the primary web-auth path.
- CLI bearer tokens, service-account tokens, and API-token flows remain bearer-based on purpose.
- Cookie-authenticated browser mutations rely on same-site cookies plus `Origin` / `Referer` validation rather than a separate anti-CSRF token framework.
- This is a materially stronger browser session model than the earlier token-in-browser-storage path, but it is not the same thing as a fully expanded enterprise IAM platform.

## Honest Limits

- Only an OIDC-style provider is supported in this milestone.
- This is not full enterprise IAM.
- There is no SCIM provisioning or deprovisioning yet.
- There is no SAML support yet.
- There is no deep role-mapping UX yet.
- Session fleet administration remains shallow: there is no device/session inventory UI, admin-driven global session revocation console, or advanced session rotation policy surface yet.
