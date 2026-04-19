package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
	"gopkg.in/yaml.v3"
)

type openAPIDocument struct {
	Paths      map[string]map[string]openAPIOperation `yaml:"paths"`
	Components openAPIComponents                      `yaml:"components"`
}

type openAPIComponents struct {
	Schemas map[string]*openAPISchema `yaml:"schemas"`
}

type openAPIOperation struct {
	Responses map[string]openAPIResponse `yaml:"responses"`
}

type openAPIResponse struct {
	Content map[string]openAPIMediaType `yaml:"content"`
}

type openAPIMediaType struct {
	Schema *openAPISchema `yaml:"schema"`
}

type openAPISchema struct {
	Ref                  string                    `yaml:"$ref"`
	Type                 string                    `yaml:"type"`
	Required             []string                  `yaml:"required"`
	Properties           map[string]*openAPISchema `yaml:"properties"`
	Items                *openAPISchema            `yaml:"items"`
	AdditionalProperties any                       `yaml:"additionalProperties"`
	Nullable             bool                      `yaml:"nullable"`
}

func TestOpenAPISchemasStayAlignedWithHighValueGoTypes(t *testing.T) {
	t.Parallel()

	doc := loadOpenAPIDocument(t)
	cases := []struct {
		schema string
		value  any
	}{
		{schema: "Organization", value: types.Organization{}},
		{schema: "Project", value: types.Project{}},
		{schema: "Team", value: types.Team{}},
		{schema: "Service", value: types.Service{}},
		{schema: "Environment", value: types.Environment{}},
		{schema: "ChangeSet", value: types.ChangeSet{}},
		{schema: "RiskAssessment", value: types.RiskAssessment{}},
		{schema: "Policy", value: types.Policy{}},
		{schema: "PolicyCondition", value: types.PolicyCondition{}},
		{schema: "PolicyDecision", value: types.PolicyDecision{}},
		{schema: "RolloutPlan", value: types.RolloutPlan{}},
		{schema: "AuditEvent", value: types.AuditEvent{}},
		{schema: "RollbackPolicy", value: types.RollbackPolicy{}},
		{schema: "Incident", value: types.Incident{}},
		{schema: "IncidentDetail", value: types.IncidentDetail{}},
		{schema: "Integration", value: types.Integration{}},
		{schema: "IntegrationSyncRun", value: types.IntegrationSyncRun{}},
		{schema: "IntegrationTestResult", value: types.IntegrationTestResult{}},
		{schema: "IntegrationSyncResult", value: types.IntegrationSyncResult{}},
		{schema: "ServiceAccount", value: types.ServiceAccount{}},
		{schema: "APIToken", value: types.APIToken{}},
		{schema: "IssuedAPITokenResponse", value: types.IssuedAPITokenResponse{}},
		{schema: "VerificationResult", value: types.VerificationResult{}},
		{schema: "RolloutExecution", value: types.RolloutExecution{}},
		{schema: "SignalSnapshot", value: types.SignalSnapshot{}},
		{schema: "RolloutExecutionRuntimeSummary", value: types.RolloutExecutionRuntimeSummary{}},
		{schema: "RolloutExecutionDetail", value: types.RolloutExecutionDetail{}},
		{schema: "IdentityProvider", value: types.IdentityProvider{}},
		{schema: "OutboxEvent", value: types.OutboxEvent{}},
	}

	for _, tc := range cases {
		t.Run(tc.schema, func(t *testing.T) {
			assertSchemaMatchesStructJSON(t, doc, tc.schema, reflect.TypeOf(tc.value))
		})
	}
}

func TestOpenAPIHighValueRuntimeResponsesMatchDocumentedSchemas(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_KUBE_TOKEN_TEST", "kube-secret")

	kubeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer kube-secret" {
			t.Fatalf("expected kubernetes bearer token header, got %q", got)
		}
		switch r.URL.Path {
		case "/custom/status/checkout":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": false},
				"status": map[string]any{
					"replicas":            2,
					"updatedReplicas":     2,
					"availableReplicas":   2,
					"unavailableReplicas": 0,
					"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
				},
			})
		case "/apis/apps/v1/namespaces/prod/deployments":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
					"spec":     map[string]any{"paused": false},
					"status": map[string]any{
						"replicas":            2,
						"updatedReplicas":     2,
						"availableReplicas":   2,
						"unavailableReplicas": 0,
						"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
					},
				}},
			})
		default:
			t.Fatalf("unexpected kubernetes path %s", r.URL.Path)
		}
	}))
	defer kubeServer.Close()

	doc := loadOpenAPIDocument(t)
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-contract@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Contract",
		OrganizationSlug: "acme-contract",
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
		Criticality:    "high",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "contract validation change",
		ChangeTypes:    []string{"code"},
		FileCount:      2,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolloutPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	executionBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rolloutPlan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions", http.MethodPost, http.StatusCreated, executionBody)

	var executionResponse types.ItemResponse[types.RolloutExecution]
	if err := json.Unmarshal(executionBody, &executionResponse); err != nil {
		t.Fatal(err)
	}
	executionID := executionResponse.Data.ID

	executionsBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/rollout-executions", nil, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions", http.MethodGet, http.StatusOK, executionsBody)

	advanceBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+executionID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "contract validation approve",
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/advance", http.MethodPost, http.StatusOK, advanceBody)

	startBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+executionID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "contract validation start",
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/advance", http.MethodPost, http.StatusOK, startBody)

	snapshotBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+executionID+"/signal-snapshots", types.CreateSignalSnapshotRequest{
		ProviderType: "manual",
		Health:       "healthy",
		Summary:      "manual snapshot",
		Signals: []types.SignalValue{{
			Name:   "error_rate",
			Value:  0.2,
			Unit:   "%",
			Status: "healthy",
		}},
		WindowSeconds: 300,
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/signal-snapshots", http.MethodPost, http.StatusCreated, snapshotBody)

	verificationBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+executionID+"/verification", types.RecordVerificationResultRequest{
		Outcome:        "passed",
		Decision:       "continue",
		Summary:        "contract validation verification",
		DecisionSource: "operator",
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/verification", http.MethodPost, http.StatusCreated, verificationBody)

	detailBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/rollout-executions/"+executionID, nil, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}", http.MethodGet, http.StatusOK, detailBody)

	timelineBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/rollout-executions/"+executionID+"/timeline", nil, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/timeline", http.MethodGet, http.StatusOK, timelineBody)

	paused := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+executionID+"/pause", struct {
		Reason string `json:"reason"`
	}{Reason: "incident pause"}, admin.Token, admin.Session.ActiveOrganizationID)
	if paused.Status != "paused" {
		t.Fatalf("expected paused rollout execution, got %+v", paused)
	}

	incidentID := "incident_" + executionID
	incidentBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/incidents/"+incidentID, nil, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/incidents/{id}", http.MethodGet, http.StatusOK, incidentBody)

	integrationsBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/integrations", nil, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations", http.MethodGet, http.StatusOK, integrationsBody)

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var kubeIntegration types.Integration
	for _, integration := range integrations {
		if integration.Kind == "kubernetes" {
			kubeIntegration = integration
			break
		}
	}
	if kubeIntegration.ID == "" {
		t.Fatal("expected built-in kubernetes integration")
	}

	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":      kubeServer.URL,
			"namespace":         "prod",
			"deployment_name":   "checkout",
			"status_path":       "/custom/status/checkout",
			"bearer_token_env":  "CCP_KUBE_TOKEN_TEST",
			"inventory_enabled": true,
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	integrationTestBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/integrations/"+kubeIntegration.ID+"/test", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}/test", http.MethodPost, http.StatusOK, integrationTestBody)

	integrationSyncBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/integrations/"+kubeIntegration.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}/sync", http.MethodPost, http.StatusOK, integrationSyncBody)

	issuedTokenBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "contract-agent",
		Role:           "org_member",
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/service-accounts", http.MethodPost, http.StatusCreated, issuedTokenBody)

	var serviceAccountResponse types.ItemResponse[types.ServiceAccount]
	if err := json.Unmarshal(issuedTokenBody, &serviceAccountResponse); err != nil {
		t.Fatal(err)
	}
	serviceAccountID := serviceAccountResponse.Data.ID

	tokenIssueBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/service-accounts/"+serviceAccountID+"/tokens", types.IssueAPITokenRequest{
		Name:           "primary",
		ExpiresInHours: 12,
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/service-accounts/{id}/tokens", http.MethodPost, http.StatusCreated, tokenIssueBody)

	tokenListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/service-accounts/"+serviceAccountID+"/tokens", nil, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/service-accounts/{id}/tokens", http.MethodGet, http.StatusOK, tokenListBody)

	now := time.Now().UTC()
	retryNextAttempt := now.Add(10 * time.Minute)
	if err := application.Store.CreateOutboxEvent(context.Background(), types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        "evt_contract_retry",
			CreatedAt: now.Add(-2 * time.Minute),
			UpdatedAt: now.Add(-1 * time.Minute),
			Metadata: types.Metadata{
				"last_error_class": "temporary",
				"recovery_hint":    "check upstream dependency health before forcing an immediate retry",
			},
		},
		EventType:      "contract.retry.test",
		OrganizationID: admin.Session.ActiveOrganizationID,
		ResourceType:   "integration",
		ResourceID:     "integration_contract_retry",
		Status:         "error",
		Attempts:       2,
		LastError:      "temporary dispatch failure",
		NextAttemptAt:  &retryNextAttempt,
	}); err != nil {
		t.Fatal(err)
	}
	if err := application.Store.CreateOutboxEvent(context.Background(), types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        "evt_contract_requeue",
			CreatedAt: now.Add(-3 * time.Minute),
			UpdatedAt: now.Add(-90 * time.Second),
			Metadata: types.Metadata{
				"last_error_class": "permanent",
				"dead_lettered_at": now.Add(-90 * time.Second).Format(time.RFC3339Nano),
				"recovery_hint":    "fix the handler or payload before replaying this event",
			},
		},
		EventType:      "contract.requeue.test",
		OrganizationID: admin.Session.ActiveOrganizationID,
		ResourceType:   "integration",
		ResourceID:     "integration_contract_requeue",
		Status:         "dead_letter",
		Attempts:       5,
		LastError:      "permanent payload failure",
	}); err != nil {
		t.Fatal(err)
	}

	outboxBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/outbox-events", nil, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/outbox-events", http.MethodGet, http.StatusOK, outboxBody)

	outboxRetryBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/outbox-events/evt_contract_retry/retry", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/outbox-events/{id}/retry", http.MethodPost, http.StatusOK, outboxRetryBody)

	outboxRequeueBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/outbox-events/evt_contract_requeue/requeue", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/outbox-events/{id}/requeue", http.MethodPost, http.StatusOK, outboxRequeueBody)
}

func TestOpenAPICatalogAndGovernanceRuntimeResponsesMatchDocumentedSchemas(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	doc := loadOpenAPIDocument(t)
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-catalog-contract@acme.local",
		DisplayName:      "Catalog Owner",
		OrganizationName: "Acme Catalog Contract",
		OrganizationSlug: "acme-catalog-contract",
	})
	orgID := admin.Session.ActiveOrganizationID

	orgBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/organizations/"+orgID, nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/organizations/{id}", http.MethodGet, http.StatusOK, orgBody)

	updatedOrgBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/organizations/"+orgID, types.UpdateOrganizationRequest{
		Name: stringPtr("Acme Catalog Contract Updated"),
		Tier: stringPtr("enterprise"),
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/organizations/{id}", http.MethodPatch, http.StatusOK, updatedOrgBody)

	projectBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: orgID,
		Name:           "Payments Platform",
		Slug:           "payments-platform",
		AdoptionMode:   "advisory",
		Description:    "Project used for runtime contract validation.",
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/projects", http.MethodPost, http.StatusCreated, projectBody)

	var projectResponse types.ItemResponse[types.Project]
	if err := json.Unmarshal(projectBody, &projectResponse); err != nil {
		t.Fatal(err)
	}
	projectID := projectResponse.Data.ID

	projectListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/projects", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/projects", http.MethodGet, http.StatusOK, projectListBody)

	projectGetBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/projects/"+projectID, nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/projects/{id}", http.MethodGet, http.StatusOK, projectGetBody)

	projectUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/projects/"+projectID, types.UpdateProjectRequest{
		Description:  stringPtr("Updated runtime contract project."),
		AdoptionMode: stringPtr("managed"),
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/projects/{id}", http.MethodPatch, http.StatusOK, projectUpdateBody)

	teamBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: orgID,
		ProjectID:      projectID,
		Name:           "Payments Core",
		Slug:           "payments-core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/teams", http.MethodPost, http.StatusCreated, teamBody)

	var teamResponse types.ItemResponse[types.Team]
	if err := json.Unmarshal(teamBody, &teamResponse); err != nil {
		t.Fatal(err)
	}
	teamID := teamResponse.Data.ID

	teamListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/teams", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/teams", http.MethodGet, http.StatusOK, teamListBody)

	teamGetBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/teams/"+teamID, nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/teams/{id}", http.MethodGet, http.StatusOK, teamGetBody)

	updatedOwnerUserIDs := []string{admin.Session.ActorID}
	teamUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/teams/"+teamID, types.UpdateTeamRequest{
		Name:         stringPtr("Payments Core Updated"),
		OwnerUserIDs: &updatedOwnerUserIDs,
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/teams/{id}", http.MethodPatch, http.StatusOK, teamUpdateBody)

	serviceBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   orgID,
		ProjectID:        projectID,
		TeamID:           teamID,
		Name:             "Checkout API",
		Slug:             "checkout-api",
		Description:      "Runtime contract service.",
		Criticality:      "high",
		HasSLO:           true,
		HasObservability: true,
		CustomerFacing:   true,
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/services", http.MethodPost, http.StatusCreated, serviceBody)

	var serviceResponse types.ItemResponse[types.Service]
	if err := json.Unmarshal(serviceBody, &serviceResponse); err != nil {
		t.Fatal(err)
	}
	serviceID := serviceResponse.Data.ID

	serviceListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/services", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/services", http.MethodGet, http.StatusOK, serviceListBody)

	serviceGetBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/services/"+serviceID, nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/services/{id}", http.MethodGet, http.StatusOK, serviceGetBody)

	serviceUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/services/"+serviceID, types.UpdateServiceRequest{
		Description:      stringPtr("Updated runtime contract service."),
		Criticality:      stringPtr("critical"),
		HasObservability: boolPtr(true),
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/services/{id}", http.MethodPatch, http.StatusOK, serviceUpdateBody)

	environmentBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: orgID,
		ProjectID:      projectID,
		Name:           "Production",
		Slug:           "production",
		Type:           "kubernetes",
		Region:         "us-central1",
		Production:     true,
		ComplianceZone: "pci",
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/environments", http.MethodPost, http.StatusCreated, environmentBody)

	var environmentResponse types.ItemResponse[types.Environment]
	if err := json.Unmarshal(environmentBody, &environmentResponse); err != nil {
		t.Fatal(err)
	}
	environmentID := environmentResponse.Data.ID

	environmentListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/environments", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/environments", http.MethodGet, http.StatusOK, environmentListBody)

	environmentGetBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/environments/"+environmentID, nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/environments/{id}", http.MethodGet, http.StatusOK, environmentGetBody)

	environmentUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/environments/"+environmentID, types.UpdateEnvironmentRequest{
		Region:         stringPtr("us-east1"),
		ComplianceZone: stringPtr("sox"),
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/environments/{id}", http.MethodPatch, http.StatusOK, environmentUpdateBody)

	changeBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: orgID,
		ProjectID:      projectID,
		ServiceID:      serviceID,
		EnvironmentID:  environmentID,
		Summary:        "contract rollout candidate",
		ChangeTypes:    []string{"code", "config"},
		FileCount:      5,
		ResourceCount:  2,
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/changes", http.MethodPost, http.StatusCreated, changeBody)

	var changeResponse types.ItemResponse[types.ChangeSet]
	if err := json.Unmarshal(changeBody, &changeResponse); err != nil {
		t.Fatal(err)
	}
	changeID := changeResponse.Data.ID

	changeListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/changes", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/changes", http.MethodGet, http.StatusOK, changeListBody)

	changeGetBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/changes/"+changeID, nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/changes/{id}", http.MethodGet, http.StatusOK, changeGetBody)

	riskCreateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/risk-assessments", types.CreateRiskAssessmentRequest{
		ChangeSetID: changeID,
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/risk-assessments", http.MethodPost, http.StatusCreated, riskCreateBody)

	riskListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/risk-assessments", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/risk-assessments", http.MethodGet, http.StatusOK, riskListBody)

	planCreateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: changeID,
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-plans", http.MethodPost, http.StatusCreated, planCreateBody)

	planListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/rollout-plans", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-plans", http.MethodGet, http.StatusOK, planListBody)

	rollbackCreateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollback-policies", types.CreateRollbackPolicyRequest{
		OrganizationID:            orgID,
		ProjectID:                 projectID,
		ServiceID:                 serviceID,
		EnvironmentID:             environmentID,
		Name:                      "Protect checkout",
		Description:               "Contract validation rollback policy.",
		Enabled:                   boolPtr(true),
		Priority:                  10,
		MaxErrorRate:              0.03,
		MaxLatencyMs:              500,
		MinimumThroughput:         100,
		MaxVerificationFailures:   1,
		RollbackOnProviderFailure: boolPtr(true),
		RollbackOnCriticalSignals: boolPtr(true),
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollback-policies", http.MethodPost, http.StatusCreated, rollbackCreateBody)

	var rollbackResponse types.ItemResponse[types.RollbackPolicy]
	if err := json.Unmarshal(rollbackCreateBody, &rollbackResponse); err != nil {
		t.Fatal(err)
	}
	rollbackPolicyID := rollbackResponse.Data.ID

	rollbackListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/rollback-policies", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollback-policies", http.MethodGet, http.StatusOK, rollbackListBody)

	rollbackUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/rollback-policies/"+rollbackPolicyID, types.UpdateRollbackPolicyRequest{
		Description:             stringPtr("Updated contract rollback policy."),
		MaxLatencyMs:            float64Ptr(650),
		MaxVerificationFailures: intPtr(2),
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollback-policies/{id}", http.MethodPatch, http.StatusOK, rollbackUpdateBody)

	auditListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/audit-events", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/audit-events", http.MethodGet, http.StatusOK, auditListBody)

	environmentArchiveBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/environments/"+environmentID+"/archive", struct{}{}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/environments/{id}/archive", http.MethodPost, http.StatusOK, environmentArchiveBody)

	serviceArchiveBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/services/"+serviceID+"/archive", struct{}{}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/services/{id}/archive", http.MethodPost, http.StatusOK, serviceArchiveBody)

	teamArchiveBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/teams/"+teamID+"/archive", struct{}{}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/teams/{id}/archive", http.MethodPost, http.StatusOK, teamArchiveBody)

	projectArchiveBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/projects/"+projectID+"/archive", struct{}{}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/projects/{id}/archive", http.MethodPost, http.StatusOK, projectArchiveBody)
}

func loadOpenAPIDocument(t *testing.T) openAPIDocument {
	t.Helper()
	var doc openAPIDocument
	if err := yaml.Unmarshal([]byte(openAPIContent(t)), &doc); err != nil {
		t.Fatalf("unmarshal openapi: %v", err)
	}
	if len(doc.Paths) == 0 || len(doc.Components.Schemas) == 0 {
		t.Fatal("expected parsed openapi paths and schemas")
	}
	return doc
}

func assertSchemaMatchesStructJSON(t *testing.T, doc openAPIDocument, schemaName string, structType reflect.Type) {
	t.Helper()
	schema := resolveNamedSchema(t, doc, schemaName)
	schemaFields := slices.Collect(mapKeys(schema.Properties))
	structFields := slices.Collect(mapKeys(jsonFieldTypes(structType)))
	slices.Sort(schemaFields)
	slices.Sort(structFields)
	if !reflect.DeepEqual(schemaFields, structFields) {
		t.Fatalf("expected schema %s fields %v to match struct fields %v", schemaName, schemaFields, structFields)
	}
}

func assertRouteResponseMatchesOpenAPI(t *testing.T, doc openAPIDocument, path, method string, status int, body []byte) {
	t.Helper()
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode response for %s %s: %v", method, path, err)
	}
	schema := responseSchema(t, doc, path, method, status)
	validateValueAgainstSchema(t, doc, schema, payload, fmt.Sprintf("%s %s", method, path))
}

func responseSchema(t *testing.T, doc openAPIDocument, path, method string, status int) *openAPISchema {
	t.Helper()
	operations, ok := doc.Paths[path]
	if !ok {
		t.Fatalf("expected openapi path %s", path)
	}
	operation, ok := operations[strings.ToLower(method)]
	if !ok {
		t.Fatalf("expected openapi method %s %s", method, path)
	}
	response, ok := operation.Responses[fmt.Sprintf("%d", status)]
	if !ok {
		t.Fatalf("expected openapi response %d for %s %s", status, method, path)
	}
	media, ok := response.Content["application/json"]
	if !ok || media.Schema == nil {
		t.Fatalf("expected application/json schema for %s %s %d", method, path, status)
	}
	return media.Schema
}

func resolveNamedSchema(t *testing.T, doc openAPIDocument, name string) *openAPISchema {
	t.Helper()
	schema, ok := doc.Components.Schemas[name]
	if !ok || schema == nil {
		t.Fatalf("expected schema %s in openapi components", name)
	}
	return schema
}

func resolveSchema(t *testing.T, doc openAPIDocument, schema *openAPISchema) *openAPISchema {
	t.Helper()
	if schema == nil {
		t.Fatal("expected schema")
	}
	if schema.Ref == "" {
		return schema
	}
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(schema.Ref, prefix) {
		t.Fatalf("unsupported schema ref %s", schema.Ref)
	}
	return resolveNamedSchema(t, doc, strings.TrimPrefix(schema.Ref, prefix))
}

func validateValueAgainstSchema(t *testing.T, doc openAPIDocument, schema *openAPISchema, value any, location string) {
	t.Helper()
	schema = resolveSchema(t, doc, schema)
	if value == nil {
		if schema.Nullable {
			return
		}
		t.Fatalf("expected non-null value at %s", location)
	}
	schemaType := schema.Type
	if schemaType == "" && len(schema.Properties) > 0 {
		schemaType = "object"
	}
	switch schemaType {
	case "object":
		object, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("expected object at %s, got %T", location, value)
		}
		for _, required := range schema.Required {
			if _, ok := object[required]; !ok {
				t.Fatalf("expected required field %s at %s", required, location)
			}
		}
		allowAdditional := false
		if additional, ok := schema.AdditionalProperties.(bool); ok {
			allowAdditional = additional
		}
		for key, nested := range object {
			propertySchema, ok := schema.Properties[key]
			if !ok {
				if allowAdditional {
					continue
				}
				t.Fatalf("unexpected field %s at %s", key, location)
			}
			validateValueAgainstSchema(t, doc, propertySchema, nested, location+"."+key)
		}
	case "array":
		items, ok := value.([]any)
		if !ok {
			t.Fatalf("expected array at %s, got %T", location, value)
		}
		for index, item := range items {
			validateValueAgainstSchema(t, doc, schema.Items, item, fmt.Sprintf("%s[%d]", location, index))
		}
	case "string":
		if _, ok := value.(string); !ok {
			t.Fatalf("expected string at %s, got %T", location, value)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			t.Fatalf("expected boolean at %s, got %T", location, value)
		}
	case "integer":
		number, ok := value.(float64)
		if !ok || number != float64(int64(number)) {
			t.Fatalf("expected integer at %s, got %#v", location, value)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			t.Fatalf("expected number at %s, got %T", location, value)
		}
	default:
		// OpenAPI schemas in this repo sometimes omit a type for object refs.
		if len(schema.Properties) == 0 && schema.AdditionalProperties != nil {
			if _, ok := value.(map[string]any); !ok {
				t.Fatalf("expected object at %s, got %T", location, value)
			}
			return
		}
		t.Fatalf("unsupported schema type %q at %s", schema.Type, location)
	}
}

func jsonFieldTypes(rt reflect.Type) map[string]reflect.Type {
	fields := make(map[string]reflect.Type)
	collectJSONFieldTypes(derefType(rt), fields)
	return fields
}

func collectJSONFieldTypes(rt reflect.Type, fields map[string]reflect.Type) {
	if rt.Kind() != reflect.Struct {
		return
	}
	for index := 0; index < rt.NumField(); index++ {
		field := rt.Field(index)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			collectJSONFieldTypes(derefType(field.Type), fields)
			continue
		}
		tag := field.Tag.Get("json")
		name := strings.TrimSpace(strings.Split(tag, ",")[0])
		switch name {
		case "", "-":
			continue
		default:
			fields[name] = field.Type
		}
	}
}

func derefType(rt reflect.Type) reflect.Type {
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	return rt
}

func mapKeys[V any](items map[string]V) func(func(string) bool) {
	return func(yield func(string) bool) {
		for key := range items {
			if !yield(key) {
				return
			}
		}
	}
}

func doAuthenticatedJSON(t *testing.T, method, url string, body any, token, organizationID string, expectedStatus int) []byte {
	t.Helper()

	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d for %s %s, got %d: %s", expectedStatus, method, url, resp.StatusCode, string(data))
	}
	return data
}

func intPtr(value int) *int { return &value }

func float64Ptr(value float64) *float64 { return &value }
