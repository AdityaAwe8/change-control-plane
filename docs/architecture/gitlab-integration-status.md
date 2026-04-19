# GitLab Integration Status

| Area | Status | Reality |
| --- | --- | --- |
| Token-based onboarding | real but limited | A GitLab instance can now be configured through `api_base_url`, `group`, `access_token_env`, and `webhook_secret_env`, and onboarding state becomes `configured` when valid setup is saved. |
| OAuth or app-style onboarding | missing | There is no GitLab OAuth or GitLab App installation flow yet. |
| Connection test | real and scalable enough | The platform performs real user and optional group lookups against GitLab and records sync-run evidence. |
| Project discovery | real but limited | Group-scoped and membership-based project listing now work, but discovery breadth and metadata depth are still modest. |
| Webhook validation | real and scalable enough | `X-Gitlab-Token` validation and delivery-id dedupe are live. |
| Push webhook ingest | real and scalable enough | Push events normalize repository, branch/tag, commit SHA, changed files, and issue keys into the shared change model. |
| Merge request webhook ingest | real and scalable enough | Merge requests normalize MR metadata and fetch changed files from the GitLab API. |
| Tag and release metadata ingest | partially implemented | Tag-push and release hooks are accepted and recorded, but they currently add metadata evidence more than deep release lifecycle behavior. |
| Repository mapping and attribution | real but limited | GitLab-discovered projects map through the existing repository surface and retain primary source-integration attribution. |
| Automatic webhook registration | real but limited | Group-scoped GitLab integrations can now auto-register and repair webhook delivery through the existing integration webhook-registration surface, but this is still token-based rather than OAuth/App-style onboarding. |

## Bottom Line

GitLab is now a real SCM provider in the product. It is not yet fair to call it fully enterprise-grade onboarding or a finished GitLab app ecosystem integration.
