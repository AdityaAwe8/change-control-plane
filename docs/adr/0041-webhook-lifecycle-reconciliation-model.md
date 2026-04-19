# ADR 0041: Webhook Lifecycle Reconciliation Model

## Decision

Treat webhook registration as a persisted integration-scoped record whose status is reconciled from:

- integration enablement
- secret/config readiness
- registration sync outcome
- recent delivery health
- recent validation timestamps

## Why

A simple `registered` boolean was too optimistic for serious pilot operations.

## Tradeoff

This remains a practical repair model, not a full webhook fleet-management subsystem.
