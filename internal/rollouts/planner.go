package rollouts

import (
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Planner struct{}

func NewPlanner() *Planner {
	return &Planner{}
}

func (p *Planner) Plan(change types.ChangeSet, service types.Service, environment types.Environment, assessment types.RiskAssessment, decisions []types.PolicyDecision) types.RolloutPlan {
	now := time.Now().UTC()

	verificationSignals := []string{"error-rate", "latency", "throughput"}
	if service.CustomerFacing {
		verificationSignals = append(verificationSignals, "customer-journey-health")
	}
	if service.RegulatedZone || environment.ComplianceZone != "" {
		verificationSignals = append(verificationSignals, "compliance-signal-check")
	}

	rollbackConditions := []string{
		"error-rate breach",
		"latency degradation beyond threshold",
	}
	if service.CustomerFacing {
		rollbackConditions = append(rollbackConditions, "customer journey degradation")
	}

	plan := types.RolloutPlan{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("roll"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID:           change.OrganizationID,
		ProjectID:                change.ProjectID,
		ChangeSetID:              change.ID,
		RiskAssessmentID:         assessment.ID,
		Strategy:                 assessment.RecommendedRolloutStrategy,
		ApprovalRequired:         requiresApproval(assessment, decisions),
		ApprovalLevel:            assessment.RecommendedApprovalLevel,
		DeploymentWindow:         assessment.RecommendedDeploymentWindow,
		AdditionalVerification:   assessment.Level == types.RiskLevelHigh || assessment.Level == types.RiskLevelCritical || service.RegulatedZone,
		RollbackPrecheckRequired: assessment.Level == types.RiskLevelHigh || assessment.Level == types.RiskLevelCritical || change.TouchesSchema,
		BusinessHoursRestriction: assessment.Level == types.RiskLevelLow && environment.Production,
		OffHoursPreferred:        assessment.RecommendedDeploymentWindow == "off-hours-preferred" || assessment.RecommendedDeploymentWindow == "off-hours-required",
		VerificationSignals:      verificationSignals,
		RollbackConditions:       rollbackConditions,
		Guardrails:               assessment.RecommendedGuardrails,
		Steps:                    rolloutSteps(assessment, environment),
		Explanation: []string{
			"rollout plan derived from deterministic risk score and policy decisions",
			"strategy optimized for safety while preserving delivery flow",
		},
	}

	if assessment.Level == types.RiskLevelCritical {
		plan.BusinessHoursRestriction = false
	}

	return plan
}

func requiresApproval(assessment types.RiskAssessment, decisions []types.PolicyDecision) bool {
	if assessment.RecommendedApprovalLevel != "self-serve" {
		return true
	}
	for _, decision := range decisions {
		if decision.Outcome == "require_approval" || decision.Outcome == "require_manual_review" {
			return true
		}
	}
	return false
}

func rolloutSteps(assessment types.RiskAssessment, environment types.Environment) []types.RolloutStep {
	switch assessment.RecommendedRolloutStrategy {
	case "phased-rollout":
		return []types.RolloutStep{
			{Name: "precheck", Description: "Validate rollback readiness, observability, and deployment freeze status.", Guards: []string{"rollback-ready", "verification-hooks-enabled"}},
			{Name: "phase-one", Description: "Roll out to the first low-risk cohort or region.", Guards: []string{"canary-metrics-green"}},
			{Name: "phase-two", Description: "Expand to additional production capacity with continuous verification.", Guards: []string{"customer-journey-steady"}},
			{Name: "complete", Description: "Complete rollout after guardrails remain green."},
		}
	case "canary":
		return []types.RolloutStep{
			{Name: "precheck", Description: "Validate release readiness and monitoring coverage.", Guards: []string{"health-gates"}},
			{Name: "canary", Description: "Deploy to a small traffic slice or service cohort.", Guards: []string{"error-rate-stable", "latency-stable"}},
			{Name: "promote", Description: "Promote to full traffic after validation."},
		}
	default:
		description := "Deploy directly with standard health checks."
		if environment.Production {
			description = "Deploy directly only after standard production health verification."
		}
		return []types.RolloutStep{
			{Name: "deploy", Description: description, Guards: []string{"baseline-health-checks"}},
		}
	}
}
