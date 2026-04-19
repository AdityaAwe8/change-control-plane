# ADR 0039: Session Hardening Approach

## Decision

Keep the existing bearer-token SPA model for this milestone, but harden it materially by:

- preferring fragment-based token delivery on enterprise-auth redirects
- preferring `sessionStorage` over `localStorage`
- exposing session expiry information
- validating return-to targets more strictly

## Why

This improves enterprise review posture without forcing a disruptive switch to a new cookie/session architecture mid-stream.

## Tradeoff

The product is still not a server-managed browser-session system.
