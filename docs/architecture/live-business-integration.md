# Live Business Integration

The live-business integration milestone keeps the existing control-plane model and adds a pilot-ready path around it.

## Current Model

The product now treats an integration as an organization-scoped instance with:

- `enabled` state
- advisory vs `active_control` mode
- `control_enabled` safety gate
- connection health
- last test and last sync timestamps
- latest error
- persisted sync/test/webhook history

This is intentionally narrower than a full marketplace, but the current implementation now supports multiple organization-scoped instances per integration kind with persisted instance keys and scope metadata.

## First Reference Path

The first serious live path is:

- GitHub for source control and change metadata
- Kubernetes for deployment/runtime observation
- Prometheus for runtime signal collection

## Onboarding Flow

1. Enable the integration in the org.
2. Set advisory mode first.
3. Provide env-var secret references such as `access_token_env`, `webhook_secret_env`, or `bearer_token_env`.
4. Run the connection test.
5. Run sync/discovery.
6. Map discovered repositories to services and environments.
7. Let webhook or rollout/runtime activity populate change, graph, and runtime evidence.

## Current Boundaries

- GitHub and GitLab discovery and webhook ingest are materially real.
- Kubernetes and Prometheus remain near-real HTTP-backed paths.
- Deterministic SCM ownership import is now real through CODEOWNERS-aware GitHub/GitLab sync, but deeper business discovery and live dependency inference are still future work.
