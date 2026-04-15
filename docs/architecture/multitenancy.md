# Multitenancy

Multi-tenancy is a core requirement for both SaaS and enterprise-hosted versions.

## Modeling Approach

- organizations are top-level tenant boundaries
- most domain entities carry `organization_id`
- project-scoped resources also carry `project_id`
- audit and policy records preserve tenant context

## Early Safeguards

- keep organization context explicit in APIs and repositories
- design repositories so per-tenant filtering is mandatory, not optional
- avoid hidden global state in domain services
- require authenticated actors to select or inherit an active organization scope
- deny explicit cross-org scope requests
- filter list endpoints by tenant membership

## Future Expansion

- tenant-scoped encryption boundaries
- regional data residency controls
- tenant policy packs
- enterprise multi-org administration
