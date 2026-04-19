package intelligence_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/intelligence"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestPythonClientAugmentRiskAndSimulateRollout(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	workspace := pythonWorkspace(t)
	client := intelligence.NewClient(common.Config{
		EnablePythonIntelligence: true,
		PythonExecutable:         "python3",
		PythonWorkspace:          workspace,
	})

	now := time.Now().UTC()
	change := types.ChangeSet{
		BaseRecord:              types.BaseRecord{ID: "chg_test", CreatedAt: now, UpdatedAt: now},
		OrganizationID:          "org_test",
		ProjectID:               "proj_test",
		ServiceID:               "svc_test",
		EnvironmentID:           "env_test",
		Summary:                 "update payment routing",
		ChangeTypes:             []string{"code", "iam"},
		FileCount:               12,
		ResourceCount:           2,
		TouchesInfrastructure:   true,
		TouchesIAM:              true,
		DependencyChanges:       true,
		HistoricalIncidentCount: 2,
		PoorRollbackHistory:     false,
	}
	service := types.Service{
		BaseRecord:             types.BaseRecord{ID: "svc_test", CreatedAt: now, UpdatedAt: now},
		OrganizationID:         "org_test",
		ProjectID:              "proj_test",
		Name:                   "Checkout",
		Criticality:            "mission_critical",
		CustomerFacing:         true,
		HasSLO:                 true,
		HasObservability:       true,
		DependentServicesCount: 2,
	}
	environment := types.Environment{
		BaseRecord:     types.BaseRecord{ID: "env_test", CreatedAt: now, UpdatedAt: now},
		OrganizationID: "org_test",
		ProjectID:      "proj_test",
		Name:           "Production",
		Type:           "production",
		Production:     true,
	}
	assessment := types.RiskAssessment{
		BaseRecord:                  types.BaseRecord{ID: "risk_test", CreatedAt: now, UpdatedAt: now},
		OrganizationID:              "org_test",
		ProjectID:                   "proj_test",
		ChangeSetID:                 change.ID,
		ServiceID:                   service.ID,
		EnvironmentID:               environment.ID,
		Score:                       72,
		Level:                       types.RiskLevelHigh,
		ConfidenceScore:             0.82,
		RecommendedRolloutStrategy:  "canary",
		RecommendedGuardrails:       []string{"health-check-gates"},
		RecommendedDeploymentWindow: "off-hours-preferred",
	}
	plan := types.RolloutPlan{
		BaseRecord:          types.BaseRecord{ID: "roll_test", CreatedAt: now, UpdatedAt: now},
		OrganizationID:      "org_test",
		ProjectID:           "proj_test",
		ChangeSetID:         change.ID,
		RiskAssessmentID:    assessment.ID,
		Strategy:            "canary",
		ApprovalRequired:    true,
		VerificationSignals: []string{"latency", "error-rate"},
	}

	augmentation, err := client.AugmentRisk(context.Background(), change, service, environment, assessment)
	if err != nil {
		t.Fatal(err)
	}
	if augmentation.ChangeCluster == "" {
		t.Fatal("expected change cluster from python augmentation")
	}
	if len(augmentation.NormalizedFactors) == 0 {
		t.Fatal("expected normalized factors from python augmentation")
	}

	simulation, err := client.SimulateRollout(context.Background(), change, service, environment, assessment, plan)
	if err != nil {
		t.Fatal(err)
	}
	if simulation.RecommendedNextAction == "" {
		t.Fatal("expected recommended next action from python simulation")
	}
	if len(simulation.VerificationFocus) == 0 {
		t.Fatal("expected verification focus from python simulation")
	}
}

func pythonWorkspace(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "python"))
}
