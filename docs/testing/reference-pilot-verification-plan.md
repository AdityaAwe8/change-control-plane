# Reference Pilot Verification Plan

This plan captures the minimum proof set for the local reference pilot environment.

## Goal

Demonstrate one end-to-end advisory rollout flow against:

- a real local cluster
- a real local Prometheus deployment
- a real control-plane API
- a real local GitLab-style SCM fixture

## Core Commands

Bring up the pilot environment:

```bash
make reference-pilot-up
source .tmp/reference-pilot/reference-pilot.env
```

Run the proof:

```bash
make reference-pilot-verify
```

Optional broader repo verification:

```bash
go test ./...
cd web && npm run typecheck && npm run test:e2e
```

Tear down:

```bash
make reference-pilot-down
```

## Proof Checks

The reference pilot is considered successful when:

- integrations report healthy connection and fresh sync state
- a GitLab repository is discovered and mapped
- a Kubernetes workload is discovered and mapped
- Prometheus signal targets are collected and mapped
- a GitLab merge-request webhook produces a `ChangeSet`
- a rollout execution reaches advisory verification
- runtime verification records `advisory_rollback`
- the provider action disposition is `suppressed`
- audit events, status events, and signal snapshots are present in the proof report

## Evidence Sources

- `.tmp/reference-pilot/reference-pilot-report.json`
- `.tmp/reference-pilot/*.log`
- API responses from the pilot control-plane instance
- optional browser inspection of the pilot state

## Current Classification

- `Kubernetes`: local-cluster proven
- `Prometheus`: local-metrics proven
- `SCM`: local GitLab fixture proven
- `Advisory mode`: local-reference proven
- `Hosted production environments`: not proven
