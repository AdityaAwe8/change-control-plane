# Live-Proof Harnesses

The platform now distinguishes between three proof levels:

- `repo-proven`: repository tests cover local logic and route flow
- `harness-proven`: realistic fake upstreams prove protocol-level behavior
- `live-proven`: real external systems are exercised

The harness track can now be run explicitly with `make proof-harness`.

## Current State

### GitHub

- GitHub App onboarding and installation-token exchange are harness-proven.
- GitHub App post-install webhook registration and repair are harness-proven.
- Webhook ingest is repo-proven and harness-proven.
- Real GitHub org installation is not yet verified in this repository.

### GitLab

- Token-based onboarding, group-scoped webhook registration, and webhook repair are harness-proven.
- Real hosted GitLab onboarding is not yet verified in this repository.

### Kubernetes

- Status normalization, repeated upstream transitions, and failure classification are harness-proven.
- Bearer-auth request shaping and custom status-path handling are harness-proven.
- Live cluster behavior is not yet proven.

### Prometheus

- Query execution, empty-result handling, changing samples, and server failures are harness-proven.
- Bearer-auth request shaping and custom query-path handling are harness-proven.
- Live metrics backend behavior is not yet proven.

## Why This Matters

The harness layer makes the product more credible for evaluation without pretending that repository tests alone are enough.
