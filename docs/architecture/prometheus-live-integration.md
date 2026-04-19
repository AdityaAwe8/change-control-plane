# Prometheus Live Integration

The Prometheus path is still near-real rather than fully production-hardened.

## What Is Real Now

- persisted org-scoped Prometheus integration instance
- enablement and advisory/control flags
- connection test route
- sync/validation route
- query-backed signal collection during rollout reconcile
- normalized signal snapshots stored in the existing runtime model

## Current Configuration

The current path uses metadata such as:

- `api_base_url`
- `query_path`
- `bearer_token_env`
- `queries`

## Important Limits

- no continuous scheduler for background collection
- no richer tenant routing or auth broker
- no long-term metrics retention model inside this repository
- query templates are still metadata-driven rather than fully modeled resources
