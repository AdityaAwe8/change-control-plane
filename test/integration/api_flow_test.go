package integration_test

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

func TestControlPlaneAPIFlow(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
	defer server.Close()

	login := postItem[types.DevLoginResponse](t, server.URL+"/api/v1/auth/dev/login", types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Acme Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	}, "", "")

	token := login.Token
	orgID := login.Session.ActiveOrganizationID

	orgs := getList[types.Organization](t, server.URL+"/api/v1/organizations", token, orgID)
	if len(orgs) != 1 {
		t.Fatalf("expected one accessible organization, got %d", len(orgs))
	}

	project := postItem[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: orgID,
		Name:           "Core Platform",
		Slug:           "core-platform",
	}, token, orgID)
	team := postItem[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Platform",
		Slug:           "platform",
		OwnerUserIDs:   []string{login.Session.ActorID},
	}, token, orgID)
	service := postItem[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   orgID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout API",
		Slug:             "checkout-api",
		Criticality:      "mission_critical",
		CustomerFacing:   true,
		HasSLO:           true,
		HasObservability: true,
	}, token, orgID)
	environment := postItem[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
	}, token, orgID)
	change := postItem[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID:        orgID,
		ProjectID:             project.ID,
		ServiceID:             service.ID,
		EnvironmentID:         environment.ID,
		Summary:               "Introduce a new payments retry path",
		ChangeTypes:           []string{"code", "iam"},
		FileCount:             18,
		ResourceCount:         2,
		TouchesInfrastructure: true,
		TouchesIAM:            true,
	}, token, orgID)

	risk := postItem[types.RiskAssessmentResult](t, server.URL+"/api/v1/risk-assessments", types.CreateRiskAssessmentRequest{
		ChangeSetID: change.ID,
	}, token, orgID)
	if risk.Assessment.Score == 0 {
		t.Fatal("expected non-zero risk score")
	}

	rollout := postItem[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, token, orgID)
	if rollout.Plan.Strategy == "" {
		t.Fatal("expected rollout strategy")
	}
	if !rollout.Plan.ApprovalRequired {
		t.Fatal("expected approval requirement for production IAM change")
	}

	auditEvents := getList[types.AuditEvent](t, server.URL+"/api/v1/audit-events", token, orgID)
	if len(auditEvents) < 7 {
		t.Fatalf("expected audit events to be recorded, got %d", len(auditEvents))
	}
	if auditEvents[0].ActorType == "" {
		t.Fatal("expected audit event actor type to be present")
	}

	metrics := getItem[types.BasicMetrics](t, server.URL+"/api/v1/metrics/basics", token, orgID)
	if metrics.Organizations != 1 || metrics.Services != 1 || metrics.RolloutPlans != 1 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

func postItem[T any](t *testing.T, url string, body any, token, organizationID string) T {
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

func getItem[T any](t *testing.T, url, token, organizationID string) T {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
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

func getList[T any](t *testing.T, url, token, organizationID string) []T {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
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

	var envelope types.ListResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}
