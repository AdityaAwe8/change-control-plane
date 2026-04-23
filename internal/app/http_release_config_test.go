package app_test

import (
	"net/http"
	"testing"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestConfigSetReleaseEvidenceAndIncidentRoutes(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-release@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Release",
		OrganizationSlug: "acme-release",
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
		Criticality:      "high",
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
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "Checkout API and governed config coordination",
		ChangeTypes:    []string{"code", "config"},
		FileCount:      9,
		ResourceCount:  2,
		TouchesSecrets: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	assessment := postItemAuth[types.RiskAssessmentResult](t, server.URL+"/api/v1/risk-assessments", types.CreateRiskAssessmentRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	plan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if assessment.Assessment.ID == "" || plan.Plan.ID == "" {
		t.Fatal("expected risk assessment and rollout plan to be created")
	}

	configSet := postItemAuth[types.ConfigSetDetail](t, server.URL+"/api/v1/config-sets", types.CreateConfigSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		ServiceID:      service.ID,
		Name:           "production-app",
		Version:        "v1",
		Entries: []types.ConfigEntry{
			{Key: "DB_PASSWORD_REF", Value: "prod/checkout/db/password", ValueType: "secret_ref", Required: true},
			{Key: "FEATURE_FLAG_CHECKOUT_GUARD", Value: "enabled", ValueType: "literal"},
		},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if configSet.Validation.Status != "valid" {
		t.Fatalf("expected valid config set validation, got %+v", configSet.Validation)
	}

	release := postItemAuth[types.ReleaseAnalysis](t, server.URL+"/api/v1/releases", types.CreateReleaseRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		Name:           "April Production Bundle",
		Summary:        "Checkout release bundle with governed config",
		ChangeSetIDs:   []string{change.ID},
		ConfigSetIDs:   []string{configSet.ConfigSet.ID},
		Version:        "2026.04.23",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if release.Release.ID == "" {
		t.Fatal("expected release id to be populated")
	}
	if len(release.ConfigValidation) != 1 {
		t.Fatalf("expected one config validation entry, got %d", len(release.ConfigValidation))
	}
	if len(release.ReadinessReview) == 0 {
		t.Fatal("expected readiness review to be generated for governed release")
	}

	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID:      plan.Plan.ID,
		ReleaseID:          release.Release.ID,
		BackendType:        "simulated",
		SignalProviderType: "simulated",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.ReleaseID != release.Release.ID {
		t.Fatalf("expected rollout execution to retain release id %s, got %s", release.Release.ID, execution.ReleaseID)
	}
	switch execution.Status {
	case "awaiting_approval":
		execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
			Action: "approve",
			Reason: "approve governed release bundle",
		}, admin.Token, admin.Session.ActiveOrganizationID)
		fallthrough
	case "approved", "planned":
		execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
			Action: "start",
			Reason: "start governed release bundle",
		}, admin.Token, admin.Session.ActiveOrganizationID)
	}

	pack := getItemAuth[types.RolloutEvidencePack](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/evidence-pack", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if pack.Release == nil || pack.Release.ID != release.Release.ID {
		t.Fatalf("expected evidence pack to include release context, got %+v", pack.Release)
	}
	if pack.ReleaseAnalysis == nil || pack.Summary.ReleaseID != release.Release.ID {
		t.Fatalf("expected evidence pack summary to include release mapping, got %+v", pack.Summary)
	}

	paused := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/pause?reason=browser+correlated+incident", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if paused.Status != "paused" {
		t.Fatalf("expected paused rollout execution, got %s", paused.Status)
	}

	incident := getItemAuth[types.IncidentDetail](t, server.URL+"/api/v1/incidents/incident_"+execution.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if incident.AssistantSummary == nil {
		t.Fatal("expected incident detail to include assistant summary")
	}
	if incident.AssistantSummary.LikelyCause == "" {
		t.Fatalf("expected incident assistant summary to explain likely cause, got %+v", incident.AssistantSummary)
	}
}
