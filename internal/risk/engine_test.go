package risk

import (
	"strings"
	"testing"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestAssessCriticalProductionChange(t *testing.T) {
	engine := NewEngine()

	change := types.ChangeSet{
		OrganizationID:          "org_123",
		ProjectID:               "proj_123",
		ServiceID:               "svc_123",
		EnvironmentID:           "env_123",
		ChangeTypes:             []string{"code", "schema", "iam"},
		FileCount:               42,
		ResourceCount:           3,
		TouchesInfrastructure:   true,
		TouchesIAM:              true,
		TouchesSecrets:          true,
		TouchesSchema:           true,
		DependencyChanges:       true,
		HistoricalIncidentCount: 4,
		PoorRollbackHistory:     true,
	}

	service := types.Service{
		Criticality:            "mission_critical",
		CustomerFacing:         true,
		HasSLO:                 false,
		HasObservability:       false,
		RegulatedZone:          true,
		DependentServicesCount: 4,
	}

	environment := types.Environment{
		Type:           "production",
		Production:     true,
		ComplianceZone: "pci",
	}

	assessment := engine.Assess(change, service, environment)

	if assessment.Level != types.RiskLevelCritical {
		t.Fatalf("expected critical risk level, got %s", assessment.Level)
	}
	if assessment.Score < 80 {
		t.Fatalf("expected score >= 80, got %d", assessment.Score)
	}
	if assessment.RecommendedApprovalLevel != "change-advisory-board" {
		t.Fatalf("expected CAB approval, got %s", assessment.RecommendedApprovalLevel)
	}
	if assessment.RecommendedRolloutStrategy != "phased-rollout" {
		t.Fatalf("expected phased rollout, got %s", assessment.RecommendedRolloutStrategy)
	}
	if assessment.BlastRadius.Scope != "broad" {
		t.Fatalf("expected broad blast radius, got %s", assessment.BlastRadius.Scope)
	}
	if len(assessment.RecommendedGuardrails) == 0 {
		t.Fatal("expected non-empty guardrails")
	}

	foundSchemaGuard := false
	foundSecurityGuard := false
	for _, guard := range assessment.RecommendedGuardrails {
		if strings.Contains(guard, "schema") {
			foundSchemaGuard = true
		}
		if strings.Contains(guard, "security") {
			foundSecurityGuard = true
		}
	}

	if !foundSchemaGuard {
		t.Fatal("expected schema guardrail")
	}
	if !foundSecurityGuard {
		t.Fatal("expected security guardrail")
	}
}
