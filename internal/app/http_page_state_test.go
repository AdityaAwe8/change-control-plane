package app_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestIntegrationsPageStateRouteBundlesOperationalReads(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-integrations@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-integrations",
	})
	project, _, service, environment, _, github := seedGraphContext(t, server.URL, admin.Token, admin.Session.ActiveOrganizationID, "integrations")

	_ = postListAuth[types.GraphRelationship](t, server.URL+"/api/v1/integrations/"+github.ID+"/graph-ingest", types.IntegrationGraphIngestRequest{
		Repositories: []types.IntegrationRepositoryInput{
			{
				ProjectID:     project.ID,
				ServiceID:     service.ID,
				Name:          "checkout-integrations",
				Provider:      "github",
				URL:           "https://github.com/acme/checkout-integrations",
				DefaultBranch: "main",
			},
		},
		ServiceEnvironments: []types.ServiceEnvironmentBindingInput{
			{
				ServiceID:     service.ID,
				EnvironmentID: environment.ID,
			},
		},
	}, admin.Token, admin.Session.ActiveOrganizationID)

	data := getItemAuth[types.IntegrationsPageState](t, server.URL+"/api/v1/page-state/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(data.Integrations) == 0 {
		t.Fatal("expected integrations page state to include integrations")
	}
	if len(data.Catalog.Services) == 0 {
		t.Fatal("expected integrations page state catalog to include seeded service")
	}
	if len(data.Repositories) == 0 {
		t.Fatal("expected integrations page state to include ingested repository")
	}
	if len(data.Teams) == 0 {
		t.Fatal("expected integrations page state to include teams for ownership labels")
	}
	if _, ok := data.IntegrationSyncRuns[github.ID]; !ok {
		t.Fatalf("expected integrations page state to include sync-run bucket for %s", github.ID)
	}
	if data.WebhookRegistrations[github.ID] == nil {
		t.Fatalf("expected integrations page state to include webhook registration for %s", github.ID)
	}
}

func TestRolloutPageStateRouteBundlesExecutionReads(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-rollout@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-rollout",
	})
	project, _, service, environment, change, _ := seedGraphContext(t, server.URL, admin.Token, admin.Session.ActiveOrganizationID, "rollout")

	configSet := postItemAuth[types.ConfigSetDetail](t, server.URL+"/api/v1/config-sets", types.CreateConfigSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		ServiceID:      service.ID,
		Name:           "browser-rollout",
		Version:        "v1",
		Entries: []types.ConfigEntry{
			{Key: "DB_PASSWORD_REF", Value: "prod/checkout/db/password", ValueType: "secret_ref", Required: true},
		},
	}, admin.Token, admin.Session.ActiveOrganizationID)

	rollout := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	release := postItemAuth[types.ReleaseAnalysis](t, server.URL+"/api/v1/releases", types.CreateReleaseRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		Name:           "Browser Rollout Bundle",
		Summary:        "Release bundle for rollout page-state proof",
		ChangeSetIDs:   []string{change.ID},
		ConfigSetIDs:   []string{configSet.ConfigSet.ID},
		Version:        "2026.04.23",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	databaseChange := postItemAuth[types.DatabaseChangeDetail](t, server.URL+"/api/v1/database-changes", types.CreateDatabaseChangeRequest{
		OrganizationID:  admin.Session.ActiveOrganizationID,
		ProjectID:       project.ID,
		EnvironmentID:   environment.ID,
		ServiceID:       service.ID,
		ChangeSetID:     change.ID,
		Name:            "Rollout page schema expansion",
		Datastore:       "checkout-primary",
		OperationType:   "schema_change",
		ExecutionIntent: "pre_deploy",
		Compatibility:   "expand_contract",
		Reversibility:   "reversible",
		RiskLevel:       types.RiskLevelHigh,
		Summary:         "Page-state proof database change",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	databaseConnection := postItemAuth[types.DatabaseConnectionReferenceDetail](t, server.URL+"/api/v1/database-connection-references", types.CreateDatabaseConnectionReferenceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		ServiceID:      service.ID,
		Name:           "page-state-secret-ref",
		Datastore:      "checkout-primary",
		Driver:         "postgres",
		SourceType:     "secret_ref_dsn",
		SecretRef:      "prod/checkout/page-state/runtime_dsn",
		Summary:        "Route-local proof connection reference",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	connectionTest := postItemAuth[types.DatabaseConnectionTestDetail](t, server.URL+"/api/v1/database-connection-references/"+databaseConnection.ConnectionReference.ID+"/test", types.TestDatabaseConnectionReferenceRequest{
		Trigger: "manual",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	databaseCheck := postItemAuth[types.DatabaseValidationCheckDetail](t, server.URL+"/api/v1/database-validation-checks", types.CreateDatabaseValidationCheckRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		EnvironmentID:    environment.ID,
		ServiceID:        service.ID,
		ChangeSetID:      change.ID,
		DatabaseChangeID: databaseChange.DatabaseChange.ID,
		ConnectionRefID:  databaseConnection.ConnectionReference.ID,
		Name:             "Rollout page pre-check",
		Phase:            "pre_deploy",
		CheckType:        "compatibility_check",
		ReadOnly:         true,
		Required:         true,
		ExecutionMode:    "manual_attestation",
		Specification:    "Confirm rollout page state includes DB checks.",
		Summary:          "Pending page-state test proof",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rollout.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	data := getItemAuth[types.RolloutPageState](t, server.URL+"/api/v1/page-state/rollout", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(data.RolloutPlans) != 1 || data.RolloutPlans[0].ID != rollout.Plan.ID {
		t.Fatalf("expected rollout page state to include seeded rollout plan, got %+v", data.RolloutPlans)
	}
	if len(data.RolloutExecutions) != 1 || data.RolloutExecutions[0].ID != execution.ID {
		t.Fatalf("expected rollout page state to include seeded rollout execution, got %+v", data.RolloutExecutions)
	}
	if data.RolloutExecutionDetail == nil || data.RolloutExecutionDetail.Execution.ID != execution.ID {
		t.Fatalf("expected rollout page state to include rollout execution detail, got %+v", data.RolloutExecutionDetail)
	}
	if len(data.Integrations) == 0 {
		t.Fatal("expected rollout page state to include backend integration context")
	}
	if len(data.Catalog.Services) == 0 || len(data.Catalog.Environments) == 0 {
		t.Fatalf("expected rollout page state to include catalog context, got %+v", data.Catalog)
	}
	if len(data.ConfigSets) != 1 || data.ConfigSets[0].ID != configSet.ConfigSet.ID {
		t.Fatalf("expected rollout page state to include config sets, got %+v", data.ConfigSets)
	}
	if len(data.DatabaseChanges) != 1 || data.DatabaseChanges[0].ID != databaseChange.DatabaseChange.ID {
		t.Fatalf("expected rollout page state to include database changes, got %+v", data.DatabaseChanges)
	}
	if len(data.DatabaseChecks) != 1 || data.DatabaseChecks[0].ID != databaseCheck.ValidationCheck.ID {
		t.Fatalf("expected rollout page state to include database checks, got %+v", data.DatabaseChecks)
	}
	if len(data.DatabaseConnections) != 1 || data.DatabaseConnections[0].ID != databaseConnection.ConnectionReference.ID {
		t.Fatalf("expected rollout page state to include database connections, got %+v", data.DatabaseConnections)
	}
	if len(data.DatabaseConnectionTests) != 1 || data.DatabaseConnectionTests[0].ID != connectionTest.ConnectionTest.ID {
		t.Fatalf("expected rollout page state to include database connection tests, got %+v", data.DatabaseConnectionTests)
	}
	if len(data.Releases) != 1 || data.Releases[0].ID != release.Release.ID {
		t.Fatalf("expected rollout page state to include release bundles, got %+v", data.Releases)
	}
	if data.ReleaseAnalysis == nil || data.ReleaseAnalysis.Release.ID != release.Release.ID {
		t.Fatalf("expected rollout page state to include latest release analysis, got %+v", data.ReleaseAnalysis)
	}
}

func TestDeploymentsPageStateRouteBundlesOperationalDashboardReads(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-deployments@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-deployments",
	})
	_, _, service, environment, change, _ := seedGraphContext(t, server.URL, admin.Token, admin.Session.ActiveOrganizationID, "deployments")

	rollout := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rollout.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approve deployments route test",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	policy := postItemAuth[types.RollbackPolicy](t, server.URL+"/api/v1/rollback-policies", types.CreateRollbackPolicyRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Name:           "Deployments guardrail",
		Priority:       60,
		MaxErrorRate:   1.5,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	data := getItemAuth[types.DeploymentsPageState](t, server.URL+"/api/v1/page-state/deployments?limit=5&search=approve", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(data.Catalog.Services) != 1 || data.Catalog.Services[0].ID != service.ID {
		t.Fatalf("expected deployments page state catalog to include seeded service, got %+v", data.Catalog.Services)
	}
	if len(data.RollbackPolicies) != 1 || data.RollbackPolicies[0].ID != policy.ID {
		t.Fatalf("expected deployments page state to include rollback policy, got %+v", data.RollbackPolicies)
	}
	if data.StatusDashboard.Summary.Limit != 5 {
		t.Fatalf("expected deployments page state to honor requested limit, got %+v", data.StatusDashboard.Summary)
	}
	if data.StatusDashboard.Summary.Total == 0 || len(data.StatusDashboard.Events) == 0 {
		t.Fatalf("expected deployments page state to include matching status events, got %+v", data.StatusDashboard)
	}
	if data.CoverageSummary.GitHubIntegrations+data.CoverageSummary.GitLabIntegrations+data.CoverageSummary.KubernetesIntegrations+data.CoverageSummary.PrometheusIntegrations == 0 {
		t.Fatalf("expected deployments page state to include coverage summary, got %+v", data.CoverageSummary)
	}
}

func TestEnterprisePageStateRouteBundlesAdminReads(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_OIDC_CLIENT_SECRET_TEST", "secret-value")

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-enterprise@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-enterprise",
	})
	provider := postItemAuth[types.IdentityProvider](t, server.URL+"/api/v1/identity-providers", types.CreateIdentityProviderRequest{
		OrganizationID:  admin.Session.ActiveOrganizationID,
		Name:            "Acme Okta",
		Kind:            "oidc",
		IssuerURL:       "https://issuer.example.com",
		ClientID:        "oidc-client-123",
		ClientSecretEnv: "CCP_OIDC_CLIENT_SECRET_TEST",
		AllowedDomains:  []string{"acme.local"},
		DefaultRole:     "org_member",
		Enabled:         true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = mustCreateBrowserSessionCookie(t, application, admin.Session.ActorID, "oidc", provider.ID, provider.Name, time.Now().UTC().Add(2*time.Hour), nil)

	data := getItemAuth[types.EnterprisePageState](t, server.URL+"/api/v1/page-state/enterprise", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(data.IdentityProviders) != 1 || data.IdentityProviders[0].ID != provider.ID {
		t.Fatalf("expected enterprise page state to include created provider, got %+v", data.IdentityProviders)
	}
	if len(data.Integrations) == 0 {
		t.Fatal("expected enterprise page state to include integrations")
	}
	if len(data.WebhookRegistrations) == 0 {
		t.Fatal("expected enterprise page state to include webhook diagnostics for scm integrations")
	}
	if len(data.BrowserSessions) == 0 || data.BrowserSessions[0].UserEmail != admin.Session.Email {
		t.Fatalf("expected enterprise page state to include browser session diagnostics, got %+v", data.BrowserSessions)
	}
}

func TestGraphPageStateRouteBundlesTopologyReads(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-graph@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-graph",
	})
	project, _, service, environment, change, github := seedGraphContext(t, server.URL, admin.Token, admin.Session.ActiveOrganizationID, "graph")

	relationships := postListAuth[types.GraphRelationship](t, server.URL+"/api/v1/integrations/"+github.ID+"/graph-ingest", types.IntegrationGraphIngestRequest{
		Repositories: []types.IntegrationRepositoryInput{
			{
				ProjectID:     project.ID,
				ServiceID:     service.ID,
				Name:          "checkout-graph",
				Provider:      "github",
				URL:           "https://github.com/acme/checkout-graph",
				DefaultBranch: "main",
			},
		},
		ServiceEnvironments: []types.ServiceEnvironmentBindingInput{
			{
				ServiceID:     service.ID,
				EnvironmentID: environment.ID,
			},
		},
		ChangeRepositories: []types.ChangeRepositoryBindingInput{
			{
				ChangeSetID:   change.ID,
				RepositoryURL: "https://github.com/acme/checkout-graph",
			},
		},
	}, admin.Token, admin.Session.ActiveOrganizationID)

	data := getItemAuth[types.GraphPageState](t, server.URL+"/api/v1/page-state/graph", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(data.GraphRelationships) != len(relationships) {
		t.Fatalf("expected graph page state to include %d relationships, got %d", len(relationships), len(data.GraphRelationships))
	}
	if len(data.Projects) != 1 || data.Projects[0].ID != project.ID {
		t.Fatalf("expected graph page state to include seeded project, got %+v", data.Projects)
	}
	if len(data.Teams) == 0 {
		t.Fatal("expected graph page state to include teams for ownership edges")
	}
	if len(data.Repositories) == 0 {
		t.Fatal("expected graph page state to include repositories for repository labels")
	}
	if len(data.Changes) != 1 || data.Changes[0].ID != change.ID {
		t.Fatalf("expected graph page state to include seeded change, got %+v", data.Changes)
	}
}

func TestSimulationPageStateRouteBundlesScenarioPlanningReads(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-simulation@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-simulation",
	})
	_, _, service, environment, change, _ := seedGraphContext(t, server.URL, admin.Token, admin.Session.ActiveOrganizationID, "simulation")

	rollout := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rollout.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = postItemAuth[types.RollbackPolicy](t, server.URL+"/api/v1/rollback-policies", types.CreateRollbackPolicyRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Name:           "Simulation guardrail",
		Priority:       80,
		MaxErrorRate:   1.2,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approve simulation route test",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	data := getItemAuth[types.SimulationPageState](t, server.URL+"/api/v1/page-state/simulation", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(data.Changes) != 1 || data.Changes[0].ID != change.ID {
		t.Fatalf("expected simulation page state to include seeded change, got %+v", data.Changes)
	}
	if len(data.RiskAssessments) == 0 {
		t.Fatal("expected simulation page state to include generated risk assessment")
	}
	if len(data.RolloutPlans) != 1 || data.RolloutPlans[0].ID != rollout.Plan.ID {
		t.Fatalf("expected simulation page state to include rollout plan, got %+v", data.RolloutPlans)
	}
	if len(data.RolloutExecutions) != 1 || data.RolloutExecutions[0].ID != execution.ID {
		t.Fatalf("expected simulation page state to include rollout execution, got %+v", data.RolloutExecutions)
	}
	if data.RolloutExecutionDetail == nil || data.RolloutExecutionDetail.Execution.ID != execution.ID {
		t.Fatalf("expected simulation page state to include rollout execution detail, got %+v", data.RolloutExecutionDetail)
	}
	if len(data.RollbackPolicies) != 1 {
		t.Fatalf("expected simulation page state to include rollback policy, got %+v", data.RollbackPolicies)
	}
}

func seedGraphContext(t *testing.T, serverURL, token, organizationID, suffix string) (types.Project, types.Team, types.Service, types.Environment, types.ChangeSet, types.Integration) {
	t.Helper()

	project := postItemAuth[types.Project](t, serverURL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: organizationID,
		Name:           "Platform " + suffix,
		Slug:           "platform-" + suffix,
	}, token, organizationID)
	team := postItemAuth[types.Team](t, serverURL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: organizationID,
		ProjectID:      project.ID,
		Name:           "Core " + suffix,
		Slug:           "core-" + suffix,
	}, token, organizationID)
	service := postItemAuth[types.Service](t, serverURL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: organizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout " + suffix,
		Slug:           "checkout-" + suffix,
		Criticality:    "high",
	}, token, organizationID)
	environment := postItemAuth[types.Environment](t, serverURL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: organizationID,
		ProjectID:      project.ID,
		Name:           "Production " + suffix,
		Slug:           "prod-" + suffix,
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, token, organizationID)
	change := postItemAuth[types.ChangeSet](t, serverURL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: organizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "Browser rollout " + suffix,
		ChangeTypes:    []string{"code"},
		FileCount:      2,
	}, token, organizationID)

	integrations := getListAuth[types.Integration](t, serverURL+"/api/v1/integrations", token, organizationID, http.StatusOK)
	var github types.Integration
	for _, integration := range integrations {
		if integration.Kind == "github" {
			github = integration
			break
		}
	}
	if github.ID == "" {
		t.Fatal("expected seeded github integration")
	}
	return project, team, service, environment, change, github
}
