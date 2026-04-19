# Topology Discovery

Topology discovery is now split into two layers.

## Layer 1: Discovery

What is currently discoverable:

- GitHub and GitLab repositories through the shared SCM discovery path
- repository metadata such as owner, URL, default branch, archive/private flags
- deterministic CODEOWNERS ownership evidence for GitHub/GitLab repositories when the file is present and readable
- Kubernetes and Prometheus discovered resources through provider-backed sync

## Layer 2: Mapping

What is currently map-able:

- repository to service
- repository to environment
- repository to project
- discovered resource to service
- discovered resource to environment
- discovered resource to repository
- repository or discovered resource to inferred owner team through mapped service ownership

These mappings drive:

- graph relationships
- change-to-repository linkage
- GitHub webhook change ingestion into mapped services
- provenance-bearing owner and runtime-resource edges such as `team_repository_owner`, `team_discovered_resource_owner`, and `discovered_resource_repository`

## Current Limits

- no automatic service inference from manifests
- no identity-backed ownership graph importer beyond CODEOWNERS and mapped-service inference
- dependency import remains graph-ingest based rather than fully discovered from live systems
