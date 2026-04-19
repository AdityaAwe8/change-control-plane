# Enterprise Readiness Status

This status page is intentionally strict. A feature is not counted as enterprise-ready just because a route or page exists.

## Current Audit

| Area | Status | Reality |
| --- | --- | --- |
| Password auth and dev bootstrap | real and credible | Password sign-up/sign-in, dev bootstrap, RBAC, and multitenant session scope are real and verified. |
| Enterprise browser sign-in foundation | partial | Organization-scoped OIDC provider configuration, browser start/callback, domain filtering, identity linking, and session attribution are real. This is a first enterprise-auth layer, not a full IAM product. |
| Enterprise user lifecycle | partial | OIDC callback can reconcile a user and organization membership inside the active tenant, but there is no SCIM, no deprovisioning workflow, and no serious directory-sync model yet. |
| Role mapping | partial | Default-role assignment and simple claim-driven role hooks exist, but there is no enterprise-grade role-mapping UI or deeper policy matrix yet. |
| Service-account boundary | real and credible | Human browser auth and machine API tokens remain distinct, and service accounts do not inherit human session state. |
| Durable runtime reliability | partial | Important events now persist through an outbox and the worker dispatches them durably with retry metadata. This is materially stronger than in-memory only, but still not a distributed event bus. |
| SCM webhook registration | partial | GitHub and GitLab webhook registration is now automatic for supported org/group-scoped integrations when required secret references are configured. Some scopes and missing config still fall back to honest manual/error states. |
| Webhook health visibility | real and credible | Integration pages expose webhook registration status, delivery health, last delivery, and latest error. |
| Restart/recovery reliability | partial | Outbox-backed events and scheduled sync runs survive process restart more credibly than before, but there is still no replay console, dead-letter queue, or multi-process dispatcher proof. |
| Enterprise docs and contract | partial | OpenAPI and docs now reflect the new enterprise-auth, outbox, and webhook-registration routes, but older CRUD surfaces still need a broader schema truth pass. |

## Honest Summary

- The product is no longer dev-auth only.
- The product is no longer in-memory-event only for important internal events.
- SCM onboarding no longer assumes manual webhook plumbing in every successful path.

It is still not honest to call the platform a full enterprise identity or reliability platform yet. The strongest remaining gaps are SCIM/provisioning, richer role mapping, durable replay/operations tooling, and broader live-environment proof.
