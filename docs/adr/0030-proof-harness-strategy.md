# ADR 0030: Proof Harness Strategy

## Status

Accepted

## Decision

Use realistic fake upstream HTTP servers and repeated-state integration tests as the primary next step between repository-only proof and live-environment proof.

## Why

- live external systems are not reliably available in repository CI
- protocol-level harnesses catch far more integration risk than descriptor-only tests
- this improves pilot credibility without overclaiming live proof

## Consequences

- GitHub App token exchange, Kubernetes status changes, and Prometheus collection now have stronger proof
- docs must continue to distinguish harness proof from live proof
