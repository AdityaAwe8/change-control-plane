package rollouts

import (
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestPlanHighRiskProductionRollout(t *testing.T) {
	planner := NewPlanner()
	now := time.Now().UTC()

	change := types.ChangeSet{
		BaseRecord:     types.BaseRecord{ID: "chg_123", CreatedAt: now, UpdatedAt: now},
		OrganizationID: "org_123",
		ProjectID:      "proj_123",
	}
	service := types.Service{
		CustomerFacing: true,
		RegulatedZone:  true,
	}
	environment := types.Environment{
		Production: true,
	}
	assessment := types.RiskAssessment{
		BaseRecord:                  types.BaseRecord{ID: "risk_123", CreatedAt: now, UpdatedAt: now},
		RecommendedRolloutStrategy:  "canary",
		RecommendedApprovalLevel:    "platform-owner",
		RecommendedDeploymentWindow: "off-hours-preferred",
		RecommendedGuardrails:       []string{"manual-rollback-ready", "canary-metric-checks"},
		Level:                       types.RiskLevelHigh,
	}
	decisions := []types.PolicyDecision{
		{
			Outcome: "require_approval",
		},
	}

	plan := planner.Plan(change, service, environment, assessment, decisions)

	if !plan.ApprovalRequired {
		t.Fatal("expected approval to be required")
	}
	if !plan.AdditionalVerification {
		t.Fatal("expected additional verification")
	}
	if !plan.RollbackPrecheckRequired {
		t.Fatal("expected rollback precheck to be required")
	}
	if !plan.OffHoursPreferred {
		t.Fatal("expected off-hours preference")
	}
	if plan.Strategy != "canary" {
		t.Fatalf("expected canary strategy, got %s", plan.Strategy)
	}
	if len(plan.VerificationSignals) < 3 {
		t.Fatalf("expected multiple verification signals, got %d", len(plan.VerificationSignals))
	}
}
