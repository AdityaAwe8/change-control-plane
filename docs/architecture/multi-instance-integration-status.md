# Multi-Instance Integration Status

This document describes the actual state of multi-instance integration support.

| Concern | Status | Reality |
| --- | --- | --- |
| Multiple instances per kind | implemented | Integrations are now keyed by `kind` plus `instance_key` rather than an implicit one-row-per-kind mental model. |
| Instance identity | implemented | Each integration now carries `name`, `instance_key`, `scope_type`, and `scope_name`. |
| Per-instance schedule/freshness | implemented | Scheduling, retry, stale detection, sync history, and health remain per integration instance. |
| Per-instance repository attribution | partially implemented | Repositories now persist `source_integration_id` and provenance hints, but overlapping discovery from multiple GitHub instances still resolves to a primary source rather than a perfect many-to-many model. |
| Per-instance discovered runtime resources | implemented | Discovered workloads and signal targets are already integration-scoped via `integration_id`. |
| API filters | implemented | Integration list routes now accept `kind`, `instance_key`, `scope_type`, `auth_strategy`, `enabled`, and `search`. |
| Web instance clarity | implemented | The integrations page now shows instance, scope, auth strategy, onboarding state, and supports creating new instances. |
| CLI instance clarity | implemented | CLI can now create integration instances and filter/show them more precisely. |
| Cross-instance ownership/dependency reasoning | limited | The product does not yet reconcile overlapping integration scopes into a deeper shared topology model. |

## Honest Remaining Gap

The model is now good enough for pilots with multiple GitHub orgs, clusters, or Prometheus backends. It is not yet a full enterprise integration inventory system with deep cross-instance conflict resolution.
