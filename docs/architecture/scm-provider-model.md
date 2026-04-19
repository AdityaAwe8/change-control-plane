# SCM Provider Model

The control plane now treats source control as a shared integration surface instead of a GitHub-only feature with provider-specific exceptions.

## Shared Concepts

- `integration.kind`: identifies the SCM provider instance such as `github` or `gitlab`
- integration identity: `kind + instance_key`, plus human-readable `name`, `scope_type`, and `scope_name`
- repository catalog record: persisted `Repository` rows with `provider`, `source_integration_id`, normalized URL/default branch, and provider metadata
- normalized change ingest: webhook and sync paths normalize repository identity, branch or tag, commit SHA, changed files, issue keys, reviewers, approvers, labels, and provider metadata into the existing `ChangeSet` model
- webhook delivery evidence: each SCM webhook produces an `IntegrationSyncRun` with provider-specific operation names plus normalized summaries and details
- scheduled SCM refresh: GitHub and GitLab both use the same scheduled sync and freshness model for repository discovery

## Shared Layer

The shared SCM layer currently lives in the `internal/integrations` and `internal/app` packages through:

- a common `SCMClient` interface for connection test and repository discovery
- shared repository normalization into `SCMRepository`
- shared webhook-change normalization into `SCMWebhookChange`
- shared app-layer ingest that upserts repositories and creates mapped `ChangeSet` records

This means the product can treat GitHub and GitLab as the same operator-facing category for:

- integration management
- repository discovery and mapping
- webhook-driven change ingest
- coverage summaries
- sync health and freshness

## Provider-Specific Areas

The model is shared, but a few responsibilities remain intentionally provider-specific:

- auth and onboarding:
  - GitHub supports PAT and GitHub App installation-style onboarding
  - GitLab currently supports token-based onboarding only
- webhook validation:
  - GitHub validates `X-Hub-Signature-256`
  - GitLab validates `X-Gitlab-Token`
- event vocab:
  - GitHub uses push and pull-request semantics
  - GitLab uses push, merge-request, tag-push, and release-hook semantics
- changed-file enrichment:
  - GitHub can read changed files directly from webhook payloads for several events
  - GitLab merge requests use a follow-up API call to fetch file changes

## Attribution Model

Repository attribution is stronger than before, but still intentionally modest:

- each repository has a primary `source_integration_id`
- repository metadata keeps provider/source hints for UI and graph surfaces
- change ingest records `source_integration` and `scm_provider` in metadata

This is not yet a full many-to-many provenance model. When multiple SCM instances overlap on the same repository namespace, the system still prefers a primary source record plus provenance hints instead of maintaining a complete shared-ownership graph.

## What Is Real Now

- GitHub:
  - repository discovery
  - webhook ingest
  - shared change normalization
  - GitHub App installation-style onboarding
- GitLab:
  - token-based onboarding
  - project discovery
  - webhook ingest
  - merge-request changed-file enrichment
  - shared change normalization

## What Is Still Partial

- GitLab does not yet have OAuth or GitLab App-style onboarding
- GitHub still lacks automatic webhook registration
- cross-provider provenance is still primary-source-first
- older CRUD/OpenAPI surfaces still need a broader schema truth pass
