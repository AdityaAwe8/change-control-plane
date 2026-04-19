# Reference Integration Maturity

The platform's reference integration story now has five real pillars:

1. A shared SCM provider layer for GitHub and GitLab repository discovery and webhook-backed change ingest
2. GitHub discovery, webhook ingest, and GitHub App installation-style onboarding
3. Multiple integration instances per kind with independent scope, schedule, and health
4. Recurring advisory-safe sync across those instances
5. Harness-backed proof for Kubernetes and Prometheus runtime paths

## What Changed

- GitHub now supports a real install-style onboarding flow for `github_app` instances.
- GitLab now supports a real token-based onboarding, project discovery, and webhook ingest path on the same SCM product surface.
- Integration rows are no longer treated as effectively singleton by product surfaces.
- Repository attribution now carries a primary source integration and provenance hints across both GitHub and GitLab.
- Proof depth for Kubernetes and Prometheus now includes changing upstream state and failure/recovery behavior.

## What This Means For Pilots

A serious pilot can now:

- onboard more than one GitHub, GitLab, Kubernetes, or Prometheus instance
- distinguish those instances clearly in API, web, and CLI
- use GitHub App credentials for GitHub or token-based onboarding for GitLab
- map repositories discovered from either GitHub or GitLab into the same control-plane change model
- rely on stronger in-repo proof that recurring sync behaves sensibly across changing upstream conditions

## What It Still Does Not Mean

It still does not mean:

- GitHub App marketplace polish or OAuth enterprise-ready onboarding is complete
- GitLab OAuth or GitLab App-style onboarding is complete
- live-cluster or live-metrics proof exists
- overlapping integration scopes are perfectly reconciled
- the scheduler is hardened for large multi-worker scale
