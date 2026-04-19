# Status Query Model

Operational history is now split into two API shapes:

1. `GET /api/v1/status-events`
2. `GET /api/v1/status-events/search`

## Why Two Shapes Exist

The list endpoint remains useful for simpler consumers and backward compatibility.

The search endpoint exists for serious operational surfaces that need:

- server-backed filtering
- pagination
- stable summaries
- explicit filter echoing

## Search Endpoint Result Model

The search endpoint returns:

- `events`
- `summary.total`
- `summary.returned`
- `summary.limit`
- `summary.offset`
- `summary.rollback_events`
- `summary.automated_events`
- `summary.latest_event_at`
- `summary.oldest_event_at`
- `filters`

## Web Usage

The deployment-history page now uses the search endpoint directly. The browser is no longer the primary search engine for operational history.

## Honest Limits

- this is still query-based, not streaming
- summary statistics are intentionally compact
- large-scale query performance is not yet benchmarked in-repo
