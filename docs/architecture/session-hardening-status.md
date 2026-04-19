# Session Hardening Status

## Current Classification

| Area | Status | Notes |
| --- | --- | --- |
| OIDC signed state | real and credible | Signed callback state, expiry checking, and org/provider attribution are implemented and tested. |
| Return-to validation | partial | Browser return targets are now normalized to relative URLs or allowed local/configured origins. Unsafe external origins are dropped. |
| Browser token exposure | partial | Enterprise auth redirects now prefer fragment-based token transfer for hash-routed SPA flows, which reduces token exposure in server logs and referrer paths. |
| Browser session persistence | partial | Browser sessions now prefer `sessionStorage` over `localStorage`, which reduces persistence across browser restarts. Legacy local-storage sessions are migrated/cleared. |
| Session expiry visibility | real but limited | Session responses now expose issue/expiry timestamps, the SPA clears expired browser tokens on read, and authenticated `401` responses now force the browser back to sign-in with explicit feedback instead of failing silently. |
| Cookie/session-server model | missing | The product is still a bearer-token SPA flow, not an HttpOnly cookie-backed browser session architecture. |
| Enterprise IAM breadth | partial | OIDC exists, but SCIM, SAML, stronger logout propagation, device posture, and broader enterprise session controls are still future work. |

## Main Residual Concerns

- Bearer tokens are still used by the browser, even though they are now handled in a safer way.
- There is still no centralized server-side session invalidation story for browser sessions.
- Return-to validation is intentionally practical, not a full enterprise redirect-allowlist management system.
- Mid-session expiry handling is clearer in the browser now, but it is still a client-managed bearer-session recovery model rather than a server-managed browser session.
