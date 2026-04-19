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

## Current Implementation

The current runtime uses a hybrid model:

- Go owns the primary deterministic risk score and recommendation path.
- Python provides supplemental explainable analytics through a subprocess boundary.
- Python outputs are persisted in `risk_assessments.metadata` and `rollout_plans.metadata`.
- If Python is unavailable, the deterministic Go baseline still succeeds and records that supplemental intelligence was unavailable.

## Phase 1 Design

Phase 1 uses deterministic weighted rules with plain-language explanations. This gives teams:

- explainability
- fast iteration
- policy compatibility
- reliable baseline behavior

The Python layer currently adds:

- normalized risk-factor scoring
- change clustering
- historical-pattern summaries
- confidence adjustments
- rollout simulation notes
- additional verification focus signals

## Future Extensions

- incident similarity lookup
- service sensitivity weighting learned from history
- environment-specific risk profiles
- business-metric-aware rollout guidance
- model-assisted scoring layered on top of deterministic baselines
