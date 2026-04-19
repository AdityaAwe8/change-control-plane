# ADR 0024: Secret Reference And Repository Mapping Strategy

## Status

Accepted

## Decision

Store integration configuration in metadata, but reference sensitive values through env-var names rather than persisting raw secrets. Persist repository mappings directly on repository records so discovered repos can later be linked to services and environments.

## Rationale

- the repository needed safer credential handling without a larger secret-manager integration in this session
- repository mapping is the minimum viable bridge between discovered SCM resources and the control-plane catalog

## Consequences

- integration setup now depends on env-var provisioning outside the database
- repository discovery can happen before service/environment mapping is known
- enterprise secret-manager and deeper discovery remain future work
