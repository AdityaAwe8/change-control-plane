# ADR 0013: Python Intelligence Boundary

## Status

Accepted

## Context

ChangeControlPlane needs a real Python subsystem for supplemental analytics, explainable risk augmentation, and rollout simulation. At the same time, the platform must keep core online decisions, auth, RBAC, audit, and persistence deterministic and reliable.

## Decision

Use Python as a subprocess-based analytics boundary invoked by the Go application through JSON over `stdin` and `stdout`.

Current commands:

- `risk-augment`
- `rollout-simulate`

## Consequences

Positive:

- keeps Go authoritative for online governance
- makes Python real and callable immediately
- avoids introducing a second long-running service prematurely
- keeps future ML/statistical evolution possible

Tradeoffs:

- subprocess invocation adds some latency
- failure handling must be explicit
- Python remains supplemental until deeper analytics pipelines exist

## Notes

If Python is unavailable, the deterministic Go baseline remains the source of truth and persists that supplemental intelligence was unavailable.
