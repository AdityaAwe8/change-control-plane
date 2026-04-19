# Discovery And Mapping

Discovery now has two persisted planes:

1. repositories from GitHub-style source control discovery
2. discovered runtime resources from Kubernetes and Prometheus sync

## Resource Model

Discovered resources persist:

- integration id
- provider
- resource type
- external id
- namespace
- name
- status
- health
- summary
- last seen timestamp
- optional project, service, environment, and repository mapping
- provenance metadata for imported, manual, or inferred mapping decisions
- optional inferred owner-team evidence derived from mapped service ownership

Repositories now also persist:

- source integration id
- imported CODEOWNERS ownership evidence for GitHub and GitLab when available
- mapping provenance for project/service/environment links
- inferred owner-team evidence when a mapped service has a team

## Mapping Model

Mapping is intentionally operator-guided.

The system can make light deterministic inferences, but businesses should still assume that:

- repository mappings need review
- workload mappings need review
- signal-target bindings need review
- inferred team ownership from a mapped service is a helpful hint, not an identity-backed source of truth

The product shows unmapped items explicitly so pilots can see where coverage is incomplete rather than assuming automatic inference succeeded.

## Honest Limits

- ownership import is currently CODEOWNERS-based for GitHub and GitLab only
- dependency import is still shallow and remains graph-ingest plus mapping driven
- discovered-resource status mixes provider observation and review posture in a compact model; this is good enough for the current pilot, but not the final enterprise model
