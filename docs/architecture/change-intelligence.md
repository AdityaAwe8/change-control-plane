# Change Intelligence

The change intelligence engine is responsible for turning raw change metadata into governed delivery guidance.

## Inputs

- change type and change surface
- environment
- service criticality and customer-facing posture
- infrastructure, IAM, secret, schema, or dependency impact
- historical incident signals
- rollback history
- observability and SLO coverage
- compliance zone and regulated-data posture

## Outputs

- risk score
- risk level
- blast radius summary
- approval recommendation
- rollout strategy recommendation
- required guardrails
- confidence and explanation trail

## Phase 1 Design

Phase 1 uses deterministic weighted rules with plain-language explanations. This gives teams:

- explainability
- fast iteration
- policy compatibility
- reliable baseline behavior

## Future Extensions

- incident similarity lookup
- service sensitivity weighting learned from history
- environment-specific risk profiles
- business-metric-aware rollout guidance
- model-assisted scoring layered on top of deterministic baselines
