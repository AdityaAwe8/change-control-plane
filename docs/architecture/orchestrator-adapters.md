# Orchestrator Adapters

Rollout execution now depends on a dedicated orchestrator-provider abstraction rather than ad hoc state updates.

## Provider Contract

Each provider supports:

- `Submit`
- `Sync`
- `Pause`
- `Resume`
- `Rollback`

Each call returns a normalized sync result with:

- backend type
- backend execution id
- backend status
- progress percent
- current step
- summary
- explanation
- metadata
- update time

## Current Providers

### Simulated

This is the live verified provider path today.

It models:

- submission and stable backend execution ids
- queued and progressing states
- verification checkpoints
- pause and rollback
- terminal success and failure

It is deterministic on purpose so CI and local development can prove control-loop behavior reliably.

### Kubernetes Deployment

This is a near-real provider seam.

It currently offers:

- real HTTP calls against Kubernetes-style deployment endpoints
- deployment-status normalization from native JSON or normalized adapter payloads
- success/progress/failure mapping
- pause and resume via explicit action endpoints or deployment patching
- rollback via explicit rollback endpoints or container image patching when configured
- typed transient versus terminal provider errors

It does not yet include:

- a `client-go`-based controller
- cluster discovery or kubeconfig bootstrapping
- watched resource streams
- cluster-backed release creation beyond the configured HTTP surface

This keeps the current seam genuinely usable for near-real integration and provider contract verification while preserving the fully deterministic simulated backend for CI and local smoke tests.
