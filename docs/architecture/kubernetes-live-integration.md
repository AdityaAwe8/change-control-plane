# Kubernetes Live Integration

The Kubernetes integration remains a near-real HTTP-backed provider path.

## What Is Real Now

- persisted org-scoped Kubernetes integration instance
- enablement and advisory/control flags
- connection test route
- sync route for operator visibility
- rollout runtime provider that can observe deployment status
- pause/resume/rollback provider methods available through the provider abstraction
- advisory-mode guardrails preventing live control during reconcile unless active control is enabled

## Current Configuration

The current path uses metadata such as:

- `api_base_url`
- `status_path`
- `namespace`
- `deployment_name`
- `bearer_token_env`

## Important Limits

- no `client-go`
- no kubeconfig parsing flow
- no cluster inventory discovery
- no controller-runtime reconciliation loop
- target registration is still metadata-driven
