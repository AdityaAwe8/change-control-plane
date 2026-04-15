# RBAC Model

## Roles

Organization roles:

- `org_admin`
- `org_member`
- `viewer`

Project roles:

- `project_admin`
- `project_member`
- `service_owner`
- `viewer`

## Current Permission Shape

- `org_admin`: create projects, mutate core project resources, manage integrations, broad tenant control
- `org_member`: read tenant-scoped data and participate in governed delivery flows where allowed
- `project_admin`: mutate project-scoped resources and delivery plans
- `project_member`: read project data and participate in change/risk/rollout flows
- `service_owner`: project-scoped delivery actions and service-adjacent operations
- `viewer`: read-only access

## Enforcement Boundary

RBAC checks live in the app layer, not in handlers.

This preserves:

- consistent permission behavior across API and future worker flows
- auditable denial points
- room for later policy and ABAC expansion
