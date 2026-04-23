# Session Hardening Status

## Current Classification

| Area | Status | Notes |
| --- | --- | --- |
| OIDC signed state | real and credible | Signed callback state, expiry checking, and org/provider attribution are implemented and tested. |
| Return-to validation | partial | Browser return targets are now normalized to relative URLs or allowed local/configured origins. Unsafe external origins are dropped. |
| Browser token exposure | hardened | Enterprise auth callbacks establish server-side browser sessions and no longer need bearer tokens in the redirect URL. CLI and machine clients still use bearer/API-token flows. |
| Browser session persistence | real and server-backed | Browser sessions are persisted server-side and represented in the browser by an HttpOnly SameSite=Lax cookie. Legacy browser-token storage is no longer the primary browser session model. |
| Session expiry visibility | real and server-enforced | Session responses expose issue/expiry timestamps, expired or revoked browser-session cookies are rejected server-side, logout revokes the active browser session, and authenticated `401` responses force the browser back to sign-in with explicit feedback. |
| Cookie/session-server model | real and hardened | Password sign-in, dev login, and OIDC callback paths issue HttpOnly browser-session cookies backed by persisted session records. Cookie-authenticated mutations are protected by same-origin/SameSite behavior plus `Origin`/`Referer` validation. |
| Enterprise IAM breadth | partial | OIDC exists, but SCIM, SAML, stronger logout propagation, device posture, and broader enterprise session controls are still future work. |

## Main Residual Concerns

- Bearer/API tokens are still used intentionally for CLI and machine-auth flows.
- Browser-session revocation is implemented for active sessions, but there is still no enterprise-wide global logout propagation across upstream identity providers.
- Return-to validation is intentionally practical, not a full enterprise redirect-allowlist management system.
- The model is a credible cookie-backed foundation, not a complete enterprise IAM platform with SAML, SCIM, device posture, or richer conditional-access controls.
