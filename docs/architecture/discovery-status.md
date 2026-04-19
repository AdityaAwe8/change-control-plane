# Discovery Status

This document tracks what the platform can really discover today versus what is still shallow or missing.

## Current Classification

| Discovery Area | State | Reality |
| --- | --- | --- |
| Repository discovery | partially scheduled | GitHub and GitLab repository discovery are real, persisted, mappable, and can be refreshed on demand or by schedule. |
| Repository-to-service/environment mapping | continuous and verified | Repository mappings persist cleanly, are visible in web and CLI, and are exercised by automated tests. |
| Workload discovery | partially scheduled | Kubernetes sync now persists first-class discovered workloads and surfaces them for mapping review. |
| Signal-target discovery | partially scheduled | Prometheus sync now persists discovered signal targets derived from configured queries and bindings. |
| Discovered-resource persistence | continuous and verified | Discovered resources are now first-class persisted records with provider identity, mapping fields, last-seen timestamps, and review status. |
| Discovered-resource mapping | continuous and verified | Operators can map discovered workloads and signal targets to services, environments, and repositories through API, web, and CLI. |
| Ownership import | partially implemented | GitHub and GitLab sync now do deterministic CODEOWNERS import, persist normalized ownership evidence on repositories, and expose that evidence through API, CLI, and web. There is still no directory-ownership parser beyond CODEOWNERS and no identity-backed team sync. |
| Dependency inference | shallow but real | The graph layer still depends primarily on explicit ingest and known mappings, but repository/discovered-resource mappings now infer owner-team edges deterministically from the mapped service’s team. |
| Unmapped-resource review flow | partially implemented | Unmapped resources are now queryable and visible, but the review flow is still a serious operator console rather than a polished wizard. |

## Honest Summary

Discovery is meaningfully deeper than it was before this milestone:

- GitHub and GitLab repositories are not the only discovered assets anymore.
- Kubernetes workloads and Prometheus signal targets now persist as first-class discovered resources.
- Mapping is no longer limited to repositories.
- Repositories now record deterministic CODEOWNERS ownership evidence when GitHub/GitLab access allows it.
- Mapped repositories and runtime resources now infer owner-team relationships from their linked service and carry explicit provenance markers.

Discovery is still shallow in the places businesses usually care about next:

- no identity-backed team ownership sync beyond CODEOWNERS and mapped-service inference
- no meaningful dependency inference from manifests or runtime edges
- no deep workload discovery across arbitrary clusters or namespaces
- no claim that the system can automatically understand a business environment without operator mapping help
