package policies

import (
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Evaluator interface {
	Policies() []types.Policy
	Evaluate(change types.ChangeSet, service types.Service, environment types.Environment, assessment types.RiskAssessment) []types.PolicyDecision
}

type DefaultEvaluator struct {
	policies []types.Policy
}

func NewDefaultEvaluator() *DefaultEvaluator {
	now := time.Now().UTC()
	return &DefaultEvaluator{
		policies: []types.Policy{
			{
				BaseRecord:  types.BaseRecord{ID: "policy_prod_high_risk", CreatedAt: now, UpdatedAt: now},
				Name:        "Production High Risk Approval",
				Code:        "prod-high-risk-approval",
				Scope:       "deployment",
				Mode:        "enforce",
				Enabled:     true,
				Description: "Require elevated approval for high-risk or critical production changes.",
				Triggers:    []string{"environment=production", "risk>=high"},
			},
			{
				BaseRecord:  types.BaseRecord{ID: "policy_regulated_guardrails", CreatedAt: now, UpdatedAt: now},
				Name:        "Regulated Zone Guardrails",
				Code:        "regulated-zone-guardrails",
				Scope:       "change",
				Mode:        "enforce",
				Enabled:     true,
				Description: "Require additional verification and explicit controls for regulated workloads.",
				Triggers:    []string{"regulated_zone=true"},
			},
			{
				BaseRecord:  types.BaseRecord{ID: "policy_observability_coverage", CreatedAt: now, UpdatedAt: now},
				Name:        "Observability Coverage Advisory",
				Code:        "observability-coverage",
				Scope:       "service",
				Mode:        "advise",
				Enabled:     true,
				Description: "Advise additional caution when observability or SLO coverage is weak.",
				Triggers:    []string{"has_slo=false", "has_observability=false"},
			},
		},
	}
}

func (e *DefaultEvaluator) Policies() []types.Policy {
	return append([]types.Policy(nil), e.policies...)
}

func (e *DefaultEvaluator) Evaluate(change types.ChangeSet, service types.Service, environment types.Environment, assessment types.RiskAssessment) []types.PolicyDecision {
	now := time.Now().UTC()
	decisions := make([]types.PolicyDecision, 0, 3)

	if environment.Production && (assessment.Level == types.RiskLevelHigh || assessment.Level == types.RiskLevelCritical) {
		decisions = append(decisions, types.PolicyDecision{
			BaseRecord:     types.BaseRecord{ID: common.NewID("pdec"), CreatedAt: now, UpdatedAt: now},
			OrganizationID: change.OrganizationID,
			ProjectID:      change.ProjectID,
			PolicyID:       "policy_prod_high_risk",
			ChangeSetID:    change.ID,
			Outcome:        "require_approval",
			Summary:        "Production high-risk changes require elevated approval.",
			Reasons: []string{
				"environment is production",
				"risk level is " + string(assessment.Level),
			},
		})
	}

	if service.RegulatedZone || environment.ComplianceZone != "" {
		decisions = append(decisions, types.PolicyDecision{
			BaseRecord:     types.BaseRecord{ID: common.NewID("pdec"), CreatedAt: now, UpdatedAt: now},
			OrganizationID: change.OrganizationID,
			ProjectID:      change.ProjectID,
			PolicyID:       "policy_regulated_guardrails",
			ChangeSetID:    change.ID,
			Outcome:        "require_guardrails",
			Summary:        "Regulated workloads require stronger verification and rollback readiness.",
			Reasons: []string{
				"service or environment is marked as regulated",
				"additional evidence and verification are recommended",
			},
		})
	}

	if !service.HasObservability || !service.HasSLO {
		reasons := []string{}
		if !service.HasObservability {
			reasons = append(reasons, "service lacks observability coverage")
		}
		if !service.HasSLO {
			reasons = append(reasons, "service lacks SLO coverage")
		}
		decisions = append(decisions, types.PolicyDecision{
			BaseRecord:     types.BaseRecord{ID: common.NewID("pdec"), CreatedAt: now, UpdatedAt: now},
			OrganizationID: change.OrganizationID,
			ProjectID:      change.ProjectID,
			PolicyID:       "policy_observability_coverage",
			ChangeSetID:    change.ID,
			Outcome:        "advise_caution",
			Summary:        "Weak observability coverage reduces safe rollout confidence.",
			Reasons:        reasons,
		})
	}

	return decisions
}
