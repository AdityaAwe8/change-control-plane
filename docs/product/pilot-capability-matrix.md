# Pilot Capability Matrix

This matrix is intentionally honest about the proof level of each major pilot capability.

| Capability | Current State | Current Proof | Important Limits |
| --- | --- | --- | --- |
| GitLab repository discovery | available | local-reference proven | uses a local GitLab fixture, not hosted GitLab |
| SCM CODEOWNERS ownership evidence | available | repo-proven | deterministic GitHub/GitLab sync path only; not yet part of the canonical reference-pilot walkthrough and not an identity-backed ownership sync |
| GitLab merge-request webhook ingest | available | local-reference proven | automatic webhook registration is proven only against the local fixture in this pilot |
| GitHub onboarding and webhook paths | available | harness-proven and runner-hardened | not part of the reference pilot flow; real hosted GitHub proof still requires an operator-run external report |
| Kubernetes workload discovery | available | local-cluster proven | proven against local `k3s`, not a customer cluster |
| Kubernetes rollout observation | available | local-cluster proven | advisory-mode suppression is proven; customer live control is not |
| Kubernetes pause/resume/rollback request shaping | available | live-like and repo-proven | not part of the default reference pilot validation flow |
| Prometheus signal collection | available | local-metrics proven | query templates are configured explicitly; no broader signal discovery |
| Prometheus freshness and coverage | available | local-metrics proven | based on the configured pilot queries, not a full production telemetry estate |
| Advisory-only runtime recommendation | available | local-reference proven | the pilot proves recommendation-only rollback behavior, not executed rollback on a live backend |
| Audit and status history | available | local-reference proven | proven for the scripted reference flow; broader long-running operational history is still limited |
| Deterministic governance policies | available | repo-proven and browser-proven | deterministic in-app policy model only; persisted custom policies currently govern `risk_assessment` and `rollout_plan`, not every workflow |
| Durable outbox eventing | available | repo-proven and partially live | not yet a replayable distributed bus or long-running soak-proven ops plane |
| Enterprise OIDC sign-in foundation | available | repo-proven | not exercised as the primary path in the reference pilot |
| Browser operator experience | available | browser-proven for primary admin and operational flows; reference pilot remains script/API first | sign-in, org switching, project/team/service/environment administration, rollout controls, policy authoring, integrations configuration plus connection-test and sync actions, repository/runtime mapping, and service-account/token lifecycle are browser-tested, but the canonical reference pilot is still script/API first |
| Graph provenance and owner edges | available | repo-proven and browser-proven | owner and provenance edges are deterministic from mappings/imports, but dependency inference is still limited |
| Release-readiness ship gate | available | locally validated and runner-hardened | validates local/harness checks plus saved proof artifacts, now scans generated release evidence for secret-backed env leakage, but still depends on preserved external live-proof evidence for real hosted/customer claims |
| Hosted-provider production readiness | partial | runner-hardened but not yet operator-captured | the reference pilot is intentionally local and customer-like, and real hosted/customer proof still depends on preserved `live-proof-verify` evidence from actual environments |
