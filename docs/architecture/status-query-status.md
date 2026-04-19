# Status Query Status

This document describes the real state of operational history querying.

## Current Classification

| Area | State | Reality |
| --- | --- | --- |
| Backend status-event filtering | continuous and verified | The API supports scoped status-event filtering by project, service, environment, rollout, actor, source, event type, rollback-only, automation flag, search text, and time range. |
| Backend pagination | continuous and verified | The dedicated search endpoint now returns summary metadata with total, returned, limit, offset, rollback count, and automated count. |
| Browser dashboard querying | continuous and verified | The deployment-history page now uses the server-backed search endpoint instead of only client-side row hiding. |
| CLI status querying | continuous and verified | The CLI now queries the server-backed search endpoint and supports richer filters such as source, event type, automated, limit, and offset. |
| Search result summaries | partially implemented | Result summaries are now visible in API and web, but the UI could still do more to explain long time windows and incomplete coverage. |
| Query performance proof | partially verified | Additional indexes were added and repository tests cover correctness, but there is still no production-scale benchmark evidence in this repo. |

## Honest Summary

The status dashboard is no longer pretending that a preloaded feed is a real operational query model. The product now has a proper backend search route and uses it in the browser and CLI.

Remaining gaps:

- no dedicated performance benchmark against production-scale event volumes
- query summaries are useful but still fairly operator-centric
- older read surfaces still fetch simpler list endpoints and do not all use the richer query model yet
