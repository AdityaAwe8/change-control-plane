package app_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestRiskAndRolloutIncludePythonIntelligenceMetadata(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENABLE_PYTHON_INTELLIGENCE", "true")
	t.Setenv("CCP_PYTHON_EXECUTABLE", "python3")
	t.Setenv("CCP_PYTHON_WORKSPACE", repoPythonWorkspace(t))

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "mission_critical",
		CustomerFacing:   true,
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID:        admin.Session.ActiveOrganizationID,
		ProjectID:             project.ID,
		ServiceID:             service.ID,
		EnvironmentID:         environment.ID,
		Summary:               "update payment routing",
		ChangeTypes:           []string{"code", "iam"},
		FileCount:             12,
		ResourceCount:         2,
		TouchesInfrastructure: true,
		TouchesIAM:            true,
		DependencyChanges:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	assessment := postItemAuth[types.RiskAssessmentResult](t, server.URL+"/api/v1/risk-assessments", types.CreateRiskAssessmentRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	metadata, ok := assessment.Assessment.Metadata["python_intelligence"].(map[string]any)
	if !ok {
		t.Fatalf("expected python_intelligence metadata, got %#v", assessment.Assessment.Metadata)
	}
	if metadata["status"] != "applied" {
		t.Fatalf("expected applied python_intelligence status, got %#v", metadata)
	}
	if len(assessment.Assessment.Explanation) == 0 {
		t.Fatal("expected explainable risk output")
	}

	plan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	simulation, ok := plan.Plan.Metadata["python_simulation"].(map[string]any)
	if !ok {
		t.Fatalf("expected python_simulation metadata, got %#v", plan.Plan.Metadata)
	}
	if simulation["status"] != "applied" {
		t.Fatalf("expected applied python_simulation status, got %#v", simulation)
	}
	if len(plan.Plan.VerificationSignals) < 3 {
		t.Fatalf("expected rollout plan verification signals to be enriched, got %#v", plan.Plan.VerificationSignals)
	}
}

func repoPythonWorkspace(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", "python"))
}
