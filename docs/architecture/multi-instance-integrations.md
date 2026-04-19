# Multi-Instance Integrations

## Model

Each integration instance now has:

- `kind`
- `name`
- `instance_key`
- `scope_type`
- `scope_name`
- `auth_strategy`
- independent schedule, freshness, health, and sync history

## Practical Meaning

An organization can now keep separate instances such as:

- `github:corp-prod`
- `github:sandbox`
- `kubernetes:prod-cluster`
- `kubernetes:staging-cluster`
- `prometheus:customer-facing`
- `prometheus:internal-platform`

## Boundaries

This is a real multi-instance model, but not yet a full cross-instance topology engine. Shared resources discovered from overlapping instances still resolve to a primary source plus provenance hints rather than a perfect many-to-many ownership model.
