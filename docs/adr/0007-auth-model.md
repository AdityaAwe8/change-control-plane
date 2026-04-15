# ADR 0007: Signed Dev Tokens With Persisted Users And Memberships

## Status

Accepted

## Decision

Use signed bearer tokens with persisted users, organization memberships, and project memberships as the first authentication and authorization foundation.

## Rationale

- practical for local and early product evaluation
- no need to block the platform on enterprise identity work
- keeps authentication separate from authorization
- can evolve into richer identity providers later
