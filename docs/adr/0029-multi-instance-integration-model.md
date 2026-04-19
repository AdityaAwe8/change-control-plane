# ADR 0029: Multi-Instance Integration Model

## Status

Accepted

## Decision

Model integration identity with `kind + instance_key`, plus human-readable `name`, `scope_type`, and `scope_name`.

## Why

- the platform must support more than one GitHub, Kubernetes, or Prometheus integration per org
- schedule, freshness, and health already live naturally on the integration record
- this extends the current schema cleanly without replacing the architecture

## Consequences

- product surfaces can now distinguish instances coherently
- repository attribution gains a primary source integration
- deeper cross-instance ownership reconciliation is still future work
