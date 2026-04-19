# ADR 0027: Server-Backed Status Query Model

## Status

Accepted

## Decision

Introduce a dedicated `status-events/search` endpoint with server-backed filters, pagination, and summary metadata, and move the browser dashboard onto it.

## Why

- client-side filtering of a preloaded window is not a credible operational query model
- pilots need honest pagination, totals, and scoped search
- the backend already had richer filtering semantics than the browser was using

## Consequences

- the deployment-history page now reflects backend truth
- CLI status queries can use the same richer endpoint
- some older read surfaces still use simpler list endpoints and can be upgraded later
