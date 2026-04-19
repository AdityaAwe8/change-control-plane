# Session Hardening

This milestone hardens the existing browser-session model without replacing the product’s auth architecture.

## Changes

- Enterprise auth redirects now prefer fragment-based token delivery for hash-routed SPA returns.
- Browser session tokens now prefer `sessionStorage` instead of `localStorage`.
- Expired browser tokens are cleared on read.
- Session payloads now expose issue and expiry timestamps.
- Return URLs are normalized to safer relative or allowed local/configured origins.

## What This Does Not Claim

- This is not yet an HttpOnly cookie session architecture.
- This is not SAML, SCIM, or a complete enterprise IAM platform.
- Global logout and stronger browser-session invalidation still need future work.
