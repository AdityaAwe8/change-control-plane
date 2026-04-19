# Definition Of Done

This repository treats "done" as an operational standard, not a scaffolding standard.

## Global Definition Of Done

A subsystem is only considered done when all applicable criteria are true:

- domain model exists and is coherent
- persistence exists where the subsystem is stateful
- service or application logic exists
- API or callable interface exists where relevant
- auth and permission boundaries are enforced
- audit coverage exists for sensitive operations
- tests prove key success and denial paths
- docs explain actual current behavior
- CLI or UI surfaces exist where operator access is expected
- end-to-end flow is executable
- error handling is sane and avoids secret leakage
- the implementation is observable and explainable
- adjacent subsystems are actually integrated, not just referenced

## Subsystem-Specific Done Criteria

### Foundation And Core Platform

- tenant-scoped persistence exists for orgs, projects, teams, users, memberships, service accounts, tokens, audit, and integrations
- RBAC checks are enforced on read and write operations
- audit records exist for sensitive writes
- CLI or API flows can exercise the main lifecycle

### Service Catalog And System Graph

- services, environments, repositories, and relationships are persisted
- graph ingestion is idempotent
- relationship queries are available
- ownership and tenant consistency rules are enforced

### Change Intelligence

- change ingestion persists usable inputs
- deterministic scoring produces explainable outputs
- any supplemental intelligence is structured, explainable, and testable
- results are persisted and visible through API surfaces

### Rollout Planning And Execution

- rollout plans are persisted
- rollout executions are persisted separately from plans
- legal transitions are validated
- transitions are audited
- execution status can be listed and inspected

### Verification And Runtime Control

- verification results are persisted
- decision outcomes update rollout state deterministically
- explanations and summaries are retained
- tests cover the main decision paths

### Security And Governance

- human and machine auth are separate
- raw API tokens are never persisted
- revoked or expired credentials are rejected
- tenant isolation is enforced on list and read flows
- denial paths are tested

### Integrations

- adapter boundaries are explicit
- configuration and ingestion behavior are validated
- source-of-truth attribution is preserved
- docs clearly state whether an adapter is metadata-only, advisory, or execution-capable

### Web

- routes are backed by real API calls
- authenticated and unauthorized states are distinct
- create and action flows have usable validation and error handling
- pages are not dead-end shells for features claimed as implemented

### CLI

- session reuse works
- create, update, archive, and operational actions are executable
- errors are surfaced clearly
- commands align with current API behavior

### Python Intelligence

- real Python package or runtime exists
- dependency management exists
- tests exist
- interface contract is documented
- Go or worker runtime can call it through a clean boundary
- outputs are structured and explainable

### Documentation And Verification

- docs distinguish implemented behavior from future direction
- local run instructions are accurate
- CI verifies the languages and layers that actually exist
