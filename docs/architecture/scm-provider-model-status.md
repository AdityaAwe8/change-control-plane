# SCM Provider Model Status

This document captures the actual state of the shared SCM abstraction after GitLab support was added.

| Area | Status | Reality |
| --- | --- | --- |
| SCM provider kind and integration instance model | shared and reusable | `github` and `gitlab` now fit the same integration-instance model with multi-instance scope, auth strategy, onboarding status, sync health, and freshness. |
| Repository persistence model | shared and reusable | Both providers upsert the same `Repository` records with `provider`, `source_integration_id`, normalized URL/default branch, and metadata. |
| Change normalization | shared and reusable | GitHub push/PR and GitLab push/MR events now flow into the same change-set creation path with shared normalized fields plus provider metadata. |
| Webhook delivery evidence | shared and reusable | Both providers record webhook results as integration sync runs with dedupe/idempotency and normalized summaries. |
| Provider-specific webhook validation | provider-specific and brittle | Validation is still header-specific per provider, which is correct for now but not abstracted into a single strategy object. |
| Onboarding state model | shared and reusable | `auth_strategy` and `onboarding_status` now work for GitHub and GitLab through the same integration record. |
| GitHub App onboarding | GitHub-specific but adaptable | Real and isolated cleanly, but still GitHub-only by nature. |
| GitLab onboarding | shared and reusable | The product path is token-based, but it fits the shared integration model without hacks. |
| Repository/source attribution | GitHub-specific but adaptable | The system is now provider-aware, but provenance is still a primary-source model rather than a full many-to-many SCM-source graph. |
| API, web, and CLI SCM awareness | shared and reusable | Provider-aware integration management now exists across the main product surfaces. |
| Docs/OpenAPI SCM truth | shared and reusable | The SCM routes and coverage fields are materially more accurate now, but older CRUD schemas still lag a full truth pass. |

## Bottom Line

The product now has a real shared SCM layer, not just GitHub plus special cases. It is still fair to call provenance and full provider abstraction partial rather than finished.
