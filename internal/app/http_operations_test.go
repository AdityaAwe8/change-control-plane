package app_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestServiceAccountTokenLifecycleAndAuth(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
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
	_ = postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Ledger",
		Slug:           "ledger",
		Criticality:    "high",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	serviceAccount := postItemAuth[types.ServiceAccount](t, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "deployment-agent",
		Role:           "org_member",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	issued := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name: "primary",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	services := getListAuth[types.Service](t, server.URL+"/api/v1/services", issued.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(services) != 1 {
		t.Fatalf("expected one service through machine actor, got %d", len(services))
	}

	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-b@acme.local",
		DisplayName:      "Owner B",
		OrganizationName: "Other",
		OrganizationSlug: "other",
	})
	getListAuth[types.Service](t, server.URL+"/api/v1/services", issued.Token, otherOrg.Session.ActiveOrganizationID, http.StatusForbidden)

	_ = postItemAuth[types.APIToken](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens/"+issued.Entry.ID+"/revoke", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	getListAuth[types.Service](t, server.URL+"/api/v1/services", issued.Token, admin.Session.ActiveOrganizationID, http.StatusUnauthorized)
}

func TestGraphIngestionIsIdempotent(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
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
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
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
		Summary:        "Ship release",
		ChangeTypes:    []string{"code"},
	}, admin.Token, admin.Session.ActiveOrganizationID)

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var githubIntegration types.Integration
	for _, integration := range integrations {
		if integration.Kind == "github" {
			githubIntegration = integration
			break
		}
	}
	if githubIntegration.ID == "" {
		t.Fatal("expected github integration")
	}

	payload := types.IntegrationGraphIngestRequest{
		Repositories: []types.IntegrationRepositoryInput{
			{
				ServiceID:     service.ID,
				Name:          "checkout",
				Provider:      "github",
				URL:           "https://github.com/acme/checkout",
				DefaultBranch: "main",
			},
		},
		ChangeRepositories: []types.ChangeRepositoryBindingInput{
			{
				ChangeSetID:   change.ID,
				RepositoryURL: "https://github.com/acme/checkout",
			},
		},
		ServiceEnvironments: []types.ServiceEnvironmentBindingInput{
			{ServiceID: service.ID, EnvironmentID: environment.ID},
		},
	}

	first := postListAuth[types.GraphRelationship](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/graph-ingest", payload, admin.Token, admin.Session.ActiveOrganizationID)
	second := postListAuth[types.GraphRelationship](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/graph-ingest", payload, admin.Token, admin.Session.ActiveOrganizationID)
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("expected relationships to be ingested")
	}

	relationships := getListAuth[types.GraphRelationship](t, server.URL+"/api/v1/graph/relationships", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(relationships) != 4 {
		t.Fatalf("expected four unique relationships after repeated ingest, got %d", len(relationships))
	}
}

func TestRolloutExecutionLifecycleAndVerification(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
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
		Summary:               "Update payment routing",
		ChangeTypes:           []string{"code", "iam"},
		FileCount:             8,
		TouchesInfrastructure: true,
		TouchesIAM:            true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolloutPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rolloutPlan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "awaiting_approval" {
		t.Fatalf("expected awaiting_approval, got %s", execution.Status)
	}

	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approval granted",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "begin rollout",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "in_progress" {
		t.Fatalf("expected in_progress, got %s", execution.Status)
	}

	_ = postItemAuth[types.VerificationResult](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/verification", types.RecordVerificationResultRequest{
		Outcome:  "fail",
		Decision: "pause",
		Summary:  "latency regression detected",
		Signals:  []string{"latency", "error-rate"},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "continue",
		Reason: "manual approval after mitigation",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "in_progress" {
		t.Fatalf("expected in_progress after continue, got %s", execution.Status)
	}

	_ = postItemAuth[types.VerificationResult](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/verification", types.RecordVerificationResultRequest{
		Outcome:  "pass",
		Decision: "continue",
		Summary:  "signals recovered",
		Signals:  []string{"latency"},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "complete",
		Reason: "promotion finished",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "completed" {
		t.Fatalf("expected completed, got %s", execution.Status)
	}

	detail := getItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+execution.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(detail.VerificationResults) != 2 {
		t.Fatalf("expected two verification results, got %d", len(detail.VerificationResults))
	}
}

func TestOrgMemberCannotArchiveService(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	member := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "member@acme.local",
		DisplayName:      "Member",
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
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/services/"+service.ID+"/archive", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+member.Token)
	req.Header.Set("X-CCP-Organization-ID", admin.Session.ActiveOrganizationID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func postItemAuth[T any](t *testing.T, url string, body any, token, organizationID string) T {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if organizationID != "" {
		req.Header.Set("X-CCP-Organization-ID", organizationID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
	var envelope types.ItemResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func postListAuth[T any](t *testing.T, url string, body any, token, organizationID string) []T {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-CCP-Organization-ID", organizationID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
	var envelope types.ListResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func getItemAuth[T any](t *testing.T, url, token, organizationID string, expectedStatus int) T {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-CCP-Organization-ID", organizationID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}
	var envelope types.ItemResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func getListAuth[T any](t *testing.T, url, token, organizationID string, expectedStatus int) []T {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if organizationID != "" {
		req.Header.Set("X-CCP-Organization-ID", organizationID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}
	if expectedStatus >= http.StatusBadRequest {
		return nil
	}
	var envelope types.ListResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}
