# ADR 0006: PostgreSQL Store Behind The Existing App Seam

## Status

Accepted

## Decision

Keep the existing application architecture and replace the concrete in-memory runtime with a storage contract that supports both in-memory and PostgreSQL-backed implementations.

## Rationale

- preserves the current architecture instead of forcing a rewrite
- keeps tests fast with an in-memory option
- allows the API runtime to become durable and tenant-aware
- keeps SQL isolated from the app layer
