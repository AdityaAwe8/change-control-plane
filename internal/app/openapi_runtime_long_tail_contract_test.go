package app_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestOpenAPIAuthSummaryAndIdentityProviderRuntimeResponsesMatchDocumentedSchemas(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_OIDC_CLIENT_SECRET_TEST", "super-secret")

	oidcServer := newRuntimeOIDCServer(t)
	defer oidcServer.Close()

	doc := loadOpenAPIDocument(t)
	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
	defer server.Close()
	application.Config.APIBaseURL = server.URL

	healthBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/healthz", nil, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/healthz", http.MethodGet, http.StatusOK, healthBody)

	readyBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/readyz", nil, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/readyz", http.MethodGet, http.StatusOK, readyBody)

	signUpBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/auth/sign-up", types.SignUpRequest{
		Email:                "signup-openapi@acme.local",
		DisplayName:          "Signup User",
		Password:             "ChangeMe123!",
		PasswordConfirmation: "ChangeMe123!",
	}, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/auth/sign-up", http.MethodPost, http.StatusOK, signUpBody)

	signInBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/auth/sign-in", types.SignInRequest{
		Email:    "signup-openapi@acme.local",
		Password: "ChangeMe123!",
	}, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/auth/sign-in", http.MethodPost, http.StatusOK, signInBody)

	sessionBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/auth/session", nil, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/auth/session", http.MethodGet, http.StatusOK, sessionBody)

	logoutBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/auth/logout", nil, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/auth/logout", http.MethodPost, http.StatusOK, logoutBody)

	devLoginBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/auth/dev/login", types.DevLoginRequest{
		Email:            "owner-auth-runtime@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Auth Runtime",
		OrganizationSlug: "auth-runtime",
	}, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/auth/dev/login", http.MethodPost, http.StatusOK, devLoginBody)

	adminAuth := decodeItemResponse[types.AuthResponse](t, devLoginBody)
	orgID := adminAuth.Session.ActiveOrganizationID

	authenticatedSessionBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/auth/session", nil, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/auth/session", http.MethodGet, http.StatusOK, authenticatedSessionBody)

	orgListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/organizations", nil, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/organizations", http.MethodGet, http.StatusOK, orgListBody)

	orgCreateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/organizations", types.CreateOrganizationRequest{
		Name: "Secondary Org",
		Slug: "secondary-org",
		Tier: "growth",
		Mode: "advisory",
	}, adminAuth.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/organizations", http.MethodPost, http.StatusCreated, orgCreateBody)

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: orgID,
		Name:           "Auth Runtime Platform",
		Slug:           "auth-runtime-platform",
		AdoptionMode:   "advisory",
	}, adminAuth.Token, orgID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Identity",
		Slug:           "identity",
		OwnerUserIDs:   []string{adminAuth.Session.ActorID},
	}, adminAuth.Token, orgID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Session Gateway",
		Slug:           "session-gateway",
		Criticality:    "high",
	}, adminAuth.Token, orgID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Staging",
		Slug:           "staging",
		Type:           "staging",
		Region:         "us-central1",
	}, adminAuth.Token, orgID)

	catalogBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/catalog", nil, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/catalog", http.MethodGet, http.StatusOK, catalogBody)

	metricsBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/metrics/basics", nil, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/metrics/basics", http.MethodGet, http.StatusOK, metricsBody)

	policiesBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/policies", nil, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/policies", http.MethodGet, http.StatusOK, policiesBody)

	policyCreateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/policies", types.CreatePolicyRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Name:           "Runtime Production Review",
		Code:           "runtime-production-review",
		AppliesTo:      "rollout_plan",
		Mode:           "require_manual_review",
		Priority:       120,
		Description:    "Require manual review for high-risk production rollout plans.",
		Conditions: types.PolicyCondition{
			MinRiskLevel:   "high",
			ProductionOnly: true,
		},
	}, adminAuth.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/policies", http.MethodPost, http.StatusCreated, policyCreateBody)

	policy := decodeItemResponse[types.Policy](t, policyCreateBody)

	policyGetBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/policies/"+policy.ID, nil, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/policies/{id}", http.MethodGet, http.StatusOK, policyGetBody)

	updatedDescription := "Require manual review before production rollout planning when risk remains high."
	policyUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/policies/"+policy.ID, types.UpdatePolicyRequest{
		Description: &updatedDescription,
	}, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/policies/{id}", http.MethodPatch, http.StatusOK, policyUpdateBody)

	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "openapi policy runtime change",
		ChangeTypes:    []string{"code", "dependency"},
		FileCount:      6,
		ResourceCount:  1,
	}, adminAuth.Token, orgID)
	assessment := postItemAuth[types.RiskAssessmentResult](t, server.URL+"/api/v1/risk-assessments", types.CreateRiskAssessmentRequest{
		ChangeSetID: change.ID,
	}, adminAuth.Token, orgID)
	plan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, adminAuth.Token, orgID)

	policyDecisionBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/policy-decisions?policy_id="+policy.ID+"&risk_assessment_id="+assessment.Assessment.ID+"&rollout_plan_id="+plan.Plan.ID, nil, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/policy-decisions", http.MethodGet, http.StatusOK, policyDecisionBody)

	providerCreateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/identity-providers", types.CreateIdentityProviderRequest{
		OrganizationID:  orgID,
		Name:            "Acme OIDC",
		Kind:            "oidc",
		IssuerURL:       oidcServer.URL + "/oidc",
		ClientID:        "oidc-client-123",
		ClientSecretEnv: "CCP_OIDC_CLIENT_SECRET_TEST",
		AllowedDomains:  []string{"acme.com"},
		DefaultRole:     "org_member",
		Enabled:         true,
	}, adminAuth.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/identity-providers", http.MethodPost, http.StatusCreated, providerCreateBody)

	provider := decodeItemResponse[types.IdentityProvider](t, providerCreateBody)

	providerListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/identity-providers", nil, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/identity-providers", http.MethodGet, http.StatusOK, providerListBody)

	providerUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/identity-providers/"+provider.ID, types.UpdateIdentityProviderRequest{
		Name:        stringPtr("Acme OIDC Updated"),
		DefaultRole: stringPtr("viewer"),
	}, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/identity-providers/{id}", http.MethodPatch, http.StatusOK, providerUpdateBody)

	providerTestBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/identity-providers/"+provider.ID+"/test", struct{}{}, adminAuth.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/identity-providers/{id}/test", http.MethodPost, http.StatusOK, providerTestBody)

	publicProvidersBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/auth/providers", nil, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/auth/providers", http.MethodGet, http.StatusOK, publicProvidersBody)

	providerStartBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/auth/providers/"+provider.ID+"/start", types.IdentityProviderStartRequest{}, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/auth/providers/{id}/start", http.MethodPost, http.StatusOK, providerStartBody)

	startResult := decodeItemResponse[types.IdentityProviderStartResult](t, providerStartBody)
	authorizeURL, err := url.Parse(startResult.AuthorizeURL)
	if err != nil {
		t.Fatal(err)
	}
	state := authorizeURL.Query().Get("state")
	if strings.TrimSpace(state) == "" {
		t.Fatalf("expected signed OIDC state, got %+v", startResult)
	}

	redirectClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/auth/providers/callback?state="+url.QueryEscape(state)+"&code=good-code", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := redirectClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected OIDC callback redirect, got %d", resp.StatusCode)
	}
	if _, ok := doc.Paths["/api/v1/auth/providers/callback"]["get"].Responses["302"]; !ok {
		t.Fatal("expected OpenAPI to document the OIDC callback redirect response")
	}
}

func TestOpenAPIStatusIncidentAndRemainingRolloutRuntimeResponsesMatchDocumentedSchemas(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	doc := loadOpenAPIDocument(t)
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-status-runtime@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Status Runtime",
		OrganizationSlug: "status-runtime",
	})
	orgID := admin.Session.ActiveOrganizationID

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: orgID,
		Name:           "Rollout Runtime",
		Slug:           "rollout-runtime",
	}, admin.Token, orgID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Runtime",
		Slug:           "runtime",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, orgID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   orgID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "mission_critical",
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, orgID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, orgID)

	_ = postItemAuth[types.RollbackPolicy](t, server.URL+"/api/v1/rollback-policies", types.CreateRollbackPolicyRequest{
		OrganizationID:            orgID,
		ProjectID:                 project.ID,
		ServiceID:                 service.ID,
		EnvironmentID:             environment.ID,
		Name:                      "Prod strict",
		MaxErrorRate:              1,
		RollbackOnCriticalSignals: ptrBool(true),
	}, admin.Token, orgID)

	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "runtime rollback validation",
		ChangeTypes:    []string{"code"},
		FileCount:      3,
	}, admin.Token, orgID)
	plan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, orgID)

	executionBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID:      plan.Plan.ID,
		BackendType:        "simulated",
		SignalProviderType: "simulated",
	}, admin.Token, orgID, http.StatusCreated)
	execution := decodeItemResponse[types.RolloutExecution](t, executionBody)

	if execution.Status == "awaiting_approval" {
		approveBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
			Action: "approve",
			Reason: "approve runtime test",
		}, admin.Token, orgID, http.StatusOK)
		assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/advance", http.MethodPost, http.StatusOK, approveBody)
		execution = decodeItemResponse[types.RolloutExecution](t, approveBody)
	}

	startBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start runtime test",
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/advance", http.MethodPost, http.StatusOK, startBody)

	reconcileBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/reconcile", struct{}{}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/reconcile", http.MethodPost, http.StatusOK, reconcileBody)

	pauseBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/pause?reason="+url.QueryEscape("hold rollout"), nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/pause", http.MethodPost, http.StatusOK, pauseBody)

	resumeBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/resume?reason="+url.QueryEscape("resume rollout"), nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/resume", http.MethodPost, http.StatusOK, resumeBody)

	rollbackBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/rollback?reason="+url.QueryEscape("rollback rollout"), nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/rollout-executions/{id}/rollback", http.MethodPost, http.StatusOK, rollbackBody)

	serviceAccount := postItemAuth[types.ServiceAccount](t, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: orgID,
		Name:           "worker-bot",
		Role:           "org_member",
	}, admin.Token, orgID)
	issued := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name: "worker",
	}, admin.Token, orgID)

	autoChange := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "runtime status search candidate",
		ChangeTypes:    []string{"code"},
		FileCount:      2,
	}, admin.Token, orgID)
	autoPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: autoChange.ID,
	}, admin.Token, orgID)
	autoExecution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID:      autoPlan.Plan.ID,
		BackendType:        "simulated",
		SignalProviderType: "simulated",
	}, admin.Token, orgID)
	if autoExecution.Status == "awaiting_approval" {
		autoExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+autoExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
			Action: "approve",
			Reason: "approval granted",
		}, admin.Token, orgID)
	}
	_ = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+autoExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "begin rollout",
	}, issued.Token, orgID)
	_ = postItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+autoExecution.ID+"/reconcile", struct{}{}, issued.Token, orgID)
	_ = postItemAuth[types.SignalSnapshot](t, server.URL+"/api/v1/rollout-executions/"+autoExecution.ID+"/signal-snapshots", types.CreateSignalSnapshotRequest{
		ProviderType: "simulated",
		Health:       "critical",
		Summary:      "error rate crossed rollback threshold",
		Signals: []types.SignalValue{
			{Name: "error_rate", Category: "technical", Value: 2.7, Unit: "%", Status: "critical", Threshold: 1, Comparator: ">"},
		},
	}, issued.Token, orgID)
	_ = postItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+autoExecution.ID+"/reconcile", struct{}{}, issued.Token, orgID)

	incidentsBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/incidents?service_id="+service.ID+"&limit=5", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/incidents", http.MethodGet, http.StatusOK, incidentsBody)

	statusListBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/status-events?search=rollback&rollback_only=true&service_id="+service.ID+"&limit=10", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/status-events", http.MethodGet, http.StatusOK, statusListBody)
	statusEvents := decodeListResponse[types.StatusEvent](t, statusListBody)
	if len(statusEvents) == 0 {
		t.Fatal("expected rollback-related status events for runtime validation")
	}

	statusSearchBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/status-events/search?search=rollback&rollback_only=true&service_id="+service.ID+"&limit=1&offset=0", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/status-events/search", http.MethodGet, http.StatusOK, statusSearchBody)

	statusGetBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/status-events/"+statusEvents[0].ID, nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/status-events/{id}", http.MethodGet, http.StatusOK, statusGetBody)

	projectTimelineBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/projects/"+project.ID+"/status-events", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/projects/{id}/status-events", http.MethodGet, http.StatusOK, projectTimelineBody)

	serviceTimelineBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/services/"+service.ID+"/status-events", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/services/{id}/status-events", http.MethodGet, http.StatusOK, serviceTimelineBody)

	environmentTimelineBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/environments/"+environment.ID+"/status-events", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/environments/{id}/status-events", http.MethodGet, http.StatusOK, environmentTimelineBody)
}

func TestOpenAPIIntegrationDiscoveryAndMachineAuthRuntimeResponsesMatchDocumentedSchemas(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_GITLAB_TOKEN_TEST", "gitlab-token")
	t.Setenv("CCP_GITLAB_WEBHOOK_SECRET_TEST", "gitlab-hook-secret")
	t.Setenv("CCP_GITHUB_APP_PRIVATE_KEY_TEST", marshalRSAPrivateKeyPEM(t))
	t.Setenv("CCP_GITHUB_WEBHOOK_SECRET_TEST", "github-hook-secret")
	t.Setenv("CCP_KUBE_TOKEN_TEST", "kube-token")
	t.Setenv("CCP_PROM_TOKEN_TEST", "prom-token")

	gitLabServer := newRuntimeGitLabServer(t)
	defer gitLabServer.Close()
	gitHubServer := newRuntimeGitHubServer(t)
	defer gitHubServer.Close()
	kubeServer := newRuntimeKubernetesServer(t)
	defer kubeServer.Close()
	promServer := newRuntimePrometheusServer(t)
	defer promServer.Close()

	doc := loadOpenAPIDocument(t)
	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
	defer server.Close()
	application.Config.APIBaseURL = server.URL

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-integration-runtime@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Integration Runtime",
		OrganizationSlug: "integration-runtime",
	})
	orgID := admin.Session.ActiveOrganizationID

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: orgID,
		Name:           "Integration Runtime",
		Slug:           "integration-runtime",
	}, admin.Token, orgID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Platform",
		Slug:           "platform",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, orgID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
		Criticality:    "high",
	}, admin.Token, orgID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "production",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, orgID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: orgID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "graph ingest candidate",
		ChangeTypes:    []string{"code"},
		FileCount:      1,
	}, admin.Token, orgID)

	integrationCreateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/integrations", types.CreateIntegrationRequest{
		OrganizationID: orgID,
		Kind:           "gitlab",
		Name:           "GitLab Runtime",
		InstanceKey:    "gitlab-runtime",
		ScopeType:      "group",
		ScopeName:      "acme",
		AuthStrategy:   "token",
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations", http.MethodPost, http.StatusCreated, integrationCreateBody)
	gitLabIntegration := decodeItemResponse[types.Integration](t, integrationCreateBody)

	integrationUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/integrations/"+gitLabIntegration.ID, types.UpdateIntegrationRequest{
		Enabled:                 boolPtr(true),
		Mode:                    stringPtr("advisory"),
		ScheduleEnabled:         boolPtr(true),
		ScheduleIntervalSeconds: intPtr(300),
		Metadata: types.Metadata{
			"api_base_url":       gitLabServer.URL,
			"group":              "acme",
			"access_token_env":   "CCP_GITLAB_TOKEN_TEST",
			"webhook_secret_env": "CCP_GITLAB_WEBHOOK_SECRET_TEST",
		},
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}", http.MethodPatch, http.StatusOK, integrationUpdateBody)

	_ = postItemAuth[types.IntegrationTestResult](t, server.URL+"/api/v1/integrations/"+gitLabIntegration.ID+"/test", struct{}{}, admin.Token, orgID)
	syncResult := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+gitLabIntegration.ID+"/sync", struct{}{}, admin.Token, orgID)

	integrationCoverageBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/integrations/coverage", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/coverage", http.MethodGet, http.StatusOK, integrationCoverageBody)

	syncRunsBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/integrations/"+gitLabIntegration.ID+"/sync-runs", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}/sync-runs", http.MethodGet, http.StatusOK, syncRunsBody)

	webhookSyncBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/integrations/"+gitLabIntegration.ID+"/webhook-registration/sync", struct{}{}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}/webhook-registration/sync", http.MethodPost, http.StatusOK, webhookSyncBody)

	webhookGetBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/integrations/"+gitLabIntegration.ID+"/webhook-registration", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}/webhook-registration", http.MethodGet, http.StatusOK, webhookGetBody)

	graphIngestBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/integrations/"+gitLabIntegration.ID+"/graph-ingest", types.IntegrationGraphIngestRequest{
		Repositories: []types.IntegrationRepositoryInput{{
			ProjectID:     project.ID,
			ServiceID:     service.ID,
			Name:          "control-plane-shadow",
			Provider:      "gitlab",
			URL:           "https://gitlab.example.com/acme/control-plane-shadow",
			DefaultBranch: "main",
		}},
		ServiceEnvironments: []types.ServiceEnvironmentBindingInput{{
			ServiceID:     service.ID,
			EnvironmentID: environment.ID,
		}},
		ChangeRepositories: []types.ChangeRepositoryBindingInput{{
			ChangeSetID:   change.ID,
			RepositoryURL: syncResult.Repositories[0].URL,
		}},
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}/graph-ingest", http.MethodPost, http.StatusOK, graphIngestBody)

	graphRelationshipsBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/graph/relationships", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/graph/relationships", http.MethodGet, http.StatusOK, graphRelationshipsBody)

	repositoriesBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/repositories?provider=gitlab&source_integration_id="+gitLabIntegration.ID, nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/repositories", http.MethodGet, http.StatusOK, repositoriesBody)
	repositories := decodeListResponse[types.Repository](t, repositoriesBody)
	if len(repositories) == 0 {
		t.Fatal("expected discovered repository for runtime contract validation")
	}

	repositoryUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/repositories/"+repositories[0].ID, types.UpdateRepositoryRequest{
		ServiceID:     stringPtr(service.ID),
		EnvironmentID: stringPtr(environment.ID),
		Status:        stringPtr("mapped"),
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/repositories/{id}", http.MethodPatch, http.StatusOK, repositoryUpdateBody)

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, orgID, http.StatusOK)
	var kubeIntegration types.Integration
	var promIntegration types.Integration
	for _, integration := range integrations {
		switch integration.Kind {
		case "kubernetes":
			kubeIntegration = integration
		case "prometheus":
			promIntegration = integration
		}
	}
	if kubeIntegration.ID == "" || promIntegration.ID == "" {
		t.Fatalf("expected built-in kubernetes and prometheus integrations, got %+v", integrations)
	}

	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":      kubeServer.URL,
			"namespace":         "prod",
			"deployment_name":   "checkout",
			"bearer_token_env":  "CCP_KUBE_TOKEN_TEST",
			"inventory_enabled": true,
			"status_path":       "/apis/apps/v1/namespaces/prod/deployments/checkout",
		},
	}, admin.Token, orgID, http.StatusOK)
	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+promIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":     promServer.URL,
			"query_path":       "/api/v1/query_range",
			"window_seconds":   "300",
			"step_seconds":     "60",
			"bearer_token_env": "CCP_PROM_TOKEN_TEST",
			"queries": []map[string]any{
				{"name": "latency_ms", "category": "technical", "query": "latency_ms", "threshold": 200, "comparator": ">=", "unit": "ms"},
			},
		},
	}, admin.Token, orgID, http.StatusOK)
	_ = postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID+"/sync", struct{}{}, admin.Token, orgID)
	_ = postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+promIntegration.ID+"/sync", struct{}{}, admin.Token, orgID)

	discoveredResourcesBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/discovered-resources?service_id="+service.ID+"&limit=10", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/discovered-resources", http.MethodGet, http.StatusOK, discoveredResourcesBody)
	discoveredResources := decodeListResponse[types.DiscoveredResource](t, discoveredResourcesBody)
	if len(discoveredResources) == 0 {
		t.Fatal("expected discovered resources for runtime contract validation")
	}

	discoveredResourceUpdateBody := doAuthenticatedJSON(t, http.MethodPatch, server.URL+"/api/v1/discovered-resources/"+discoveredResources[0].ID, types.UpdateDiscoveredResourceRequest{
		ServiceID:     stringPtr(service.ID),
		EnvironmentID: stringPtr(environment.ID),
		RepositoryID:  stringPtr(repositories[0].ID),
		Status:        stringPtr("mapped"),
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/discovered-resources/{id}", http.MethodPatch, http.StatusOK, discoveredResourceUpdateBody)

	githubCreateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/integrations", types.CreateIntegrationRequest{
		OrganizationID: orgID,
		Kind:           "github",
		Name:           "GitHub Runtime",
		InstanceKey:    "github-runtime",
		ScopeType:      "organization",
		ScopeName:      "Acme GitHub",
		AuthStrategy:   "github_app",
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations", http.MethodPost, http.StatusCreated, githubCreateBody)
	githubIntegration := decodeItemResponse[types.Integration](t, githubCreateBody)

	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID, types.UpdateIntegrationRequest{
		Enabled:      boolPtr(false),
		AuthStrategy: stringPtr("github_app"),
		Metadata: types.Metadata{
			"api_base_url":       gitHubServer.URL,
			"web_base_url":       gitHubServer.URL,
			"owner":              "acme",
			"app_id":             "123456",
			"app_slug":           "change-control-plane",
			"private_key_env":    "CCP_GITHUB_APP_PRIVATE_KEY_TEST",
			"webhook_secret_env": "CCP_GITHUB_WEBHOOK_SECRET_TEST",
		},
	}, admin.Token, orgID, http.StatusOK)

	githubStartBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/github/onboarding/start", struct{}{}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}/github/onboarding/start", http.MethodPost, http.StatusOK, githubStartBody)
	githubStart := decodeItemResponse[types.GitHubOnboardingStartResult](t, githubStartBody)
	githubAuthorizeURL, err := url.Parse(githubStart.AuthorizeURL)
	if err != nil {
		t.Fatal(err)
	}
	githubState := githubAuthorizeURL.Query().Get("state")
	if strings.TrimSpace(githubState) == "" {
		t.Fatalf("expected github onboarding state, got %+v", githubStart)
	}

	githubCallbackBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/integrations/github/callback?state="+url.QueryEscape(githubState)+"&installation_id=987654&setup_action=install", nil, "", "", http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/github/callback", http.MethodGet, http.StatusOK, githubCallbackBody)

	_ = postItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/webhook-registration/sync", struct{}{}, admin.Token, orgID)

	githubPayload := []byte(`{"ref":"refs/heads/main","after":"abc123","repository":{"name":"control-plane","full_name":"acme/control-plane","html_url":"https://github.com/acme/control-plane","default_branch":"main","owner":{"login":"acme"}}}`)
	githubMAC := hmac.New(sha256.New, []byte("github-hook-secret"))
	_, _ = githubMAC.Write(githubPayload)
	githubWebhookBody := doRequestBody(t, http.MethodPost, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/webhooks/github", githubPayload, map[string]string{
		"Content-Type":        "application/json",
		"X-GitHub-Event":      "push",
		"X-GitHub-Delivery":   "delivery-openapi",
		"X-Hub-Signature-256": "sha256=" + hex.EncodeToString(githubMAC.Sum(nil)),
	}, http.StatusAccepted)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}/webhooks/github", http.MethodPost, http.StatusAccepted, githubWebhookBody)

	gitLabPayload := []byte(`{"object_kind":"merge_request","project":{"id":42,"name":"control-plane","web_url":"https://gitlab.example.com/acme/control-plane","default_branch":"main","path_with_namespace":"acme/control-plane","namespace":"acme"},"object_attributes":{"iid":7,"title":"CCP-401 tighten hooks","description":"Tracks webhook runtime validation","source_branch":"feature/ccp-401","target_branch":"main","action":"open","state":"opened","url":"https://gitlab.example.com/acme/control-plane/-/merge_requests/7","merge_status":"can_be_merged","last_commit":{"id":"abc123"}}}`)
	gitLabWebhookBody := doRequestBody(t, http.MethodPost, server.URL+"/api/v1/integrations/"+gitLabIntegration.ID+"/webhooks/gitlab", gitLabPayload, map[string]string{
		"Content-Type":        "application/json",
		"X-Gitlab-Event":      "Merge Request Hook",
		"X-Gitlab-Event-UUID": "delivery-openapi",
		"X-Gitlab-Token":      "gitlab-hook-secret",
	}, http.StatusAccepted)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/integrations/{id}/webhooks/gitlab", http.MethodPost, http.StatusAccepted, gitLabWebhookBody)

	serviceAccountCreateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: orgID,
		Name:           "runtime-operator",
		Role:           "org_member",
	}, admin.Token, orgID, http.StatusCreated)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/service-accounts", http.MethodPost, http.StatusCreated, serviceAccountCreateBody)
	serviceAccount := decodeItemResponse[types.ServiceAccount](t, serviceAccountCreateBody)

	serviceAccountsBody := doAuthenticatedJSON(t, http.MethodGet, server.URL+"/api/v1/service-accounts", nil, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/service-accounts", http.MethodGet, http.StatusOK, serviceAccountsBody)

	issuedPrimary := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name:           "primary",
		ExpiresInHours: 12,
	}, admin.Token, orgID)
	revokeBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens/"+issuedPrimary.Entry.ID+"/revoke", struct{}{}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/service-accounts/{id}/tokens/{token_id}/revoke", http.MethodPost, http.StatusOK, revokeBody)

	issuedRotate := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name:           "rotate-me",
		ExpiresInHours: 24,
	}, admin.Token, orgID)
	rotateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens/"+issuedRotate.Entry.ID+"/rotate", types.RotateAPITokenRequest{
		Name:           "rotated",
		ExpiresInHours: 48,
	}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/service-accounts/{id}/tokens/{token_id}/rotate", http.MethodPost, http.StatusOK, rotateBody)

	deactivateBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/deactivate", struct{}{}, admin.Token, orgID, http.StatusOK)
	assertRouteResponseMatchesOpenAPI(t, doc, "/api/v1/service-accounts/{id}/deactivate", http.MethodPost, http.StatusOK, deactivateBody)
}

func newRuntimeOIDCServer(t *testing.T) *httptest.Server {
	t.Helper()

	var oidcServer *httptest.Server
	oidcServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oidc/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                 oidcServer.URL + "/oidc",
				"authorization_endpoint": oidcServer.URL + "/oidc/authorize",
				"token_endpoint":         oidcServer.URL + "/oidc/token",
				"userinfo_endpoint":      oidcServer.URL + "/oidc/userinfo",
				"jwks_uri":               oidcServer.URL + "/oidc/jwks",
			})
		case "/oidc/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if got := r.Form.Get("client_id"); got != "oidc-client-123" {
				t.Fatalf("expected client_id oidc-client-123, got %q", got)
			}
			if got := r.Form.Get("client_secret"); got != "super-secret" {
				t.Fatalf("expected client secret from env, got %q", got)
			}
			if got := r.Form.Get("code"); got != "good-code" {
				t.Fatalf("expected authorization code good-code, got %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "access-token-123",
				"token_type":   "Bearer",
				"id_token":     "not-used-in-runtime-contract-tests",
			})
		case "/oidc/userinfo":
			if got := r.Header.Get("Authorization"); got != "Bearer access-token-123" {
				t.Fatalf("expected bearer access token, got %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub":                "oidc-user-123",
				"email":              "owner@acme.com",
				"name":               "Acme Owner",
				"preferred_username": "owner@acme.com",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	return oidcServer
}

func newRuntimeGitLabServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"username": "gitlab-owner"})
		case "/groups/acme":
			_ = json.NewEncoder(w).Encode(map[string]any{"full_path": "acme"})
		case "/groups/acme/projects":
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id":                  42,
				"name":                "control-plane",
				"web_url":             "https://gitlab.example.com/acme/control-plane",
				"default_branch":      "main",
				"path_with_namespace": "acme/control-plane",
				"namespace": map[string]any{
					"full_path": "acme",
				},
			}})
		case "/groups/acme/hooks":
			switch r.Method {
			case http.MethodGet:
				_ = json.NewEncoder(w).Encode([]map[string]any{})
			case http.MethodPost:
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 202})
			default:
				t.Fatalf("unexpected gitlab hooks method %s", r.Method)
			}
		case "/projects/42/merge_requests/7/changes":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"changes": []map[string]any{
					{"old_path": "deploy/values.yaml", "new_path": "deploy/values.yaml"},
					{"new_path": "db/migrations/20260416.sql", "new_file": true},
				},
			})
		case "/projects/42/repository/files/.github/CODEOWNERS", "/projects/42/repository/files/CODEOWNERS", "/projects/42/repository/files/docs/CODEOWNERS":
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("unexpected gitlab path %s", r.URL.Path)
		}
	}))
}

func newRuntimeGitHubServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/987654/access_tokens":
			if r.Method != http.MethodPost {
				t.Fatalf("expected post for github installation token, got %s", r.Method)
			}
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				t.Fatalf("expected bearer app jwt, got %q", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "ghs_installation_token",
				"expires_at": "2026-04-16T21:00:00Z",
			})
		case "/orgs/acme":
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "acme"})
		case "/orgs/acme/repos":
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"name":           "control-plane",
				"full_name":      "acme/control-plane",
				"html_url":       "https://github.com/acme/control-plane",
				"default_branch": "main",
				"private":        true,
				"archived":       false,
				"owner":          map[string]any{"login": "acme"},
			}})
		case "/repos/acme/control-plane/contents/.github/CODEOWNERS", "/repos/acme/control-plane/contents/CODEOWNERS", "/repos/acme/control-plane/contents/docs/CODEOWNERS":
			http.Error(w, "not found", http.StatusNotFound)
		case "/orgs/acme/hooks":
			switch r.Method {
			case http.MethodGet:
				_ = json.NewEncoder(w).Encode([]map[string]any{})
			case http.MethodPost:
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 303})
			default:
				t.Fatalf("unexpected github hooks method %s", r.Method)
			}
		default:
			t.Fatalf("unexpected github path %s", r.URL.Path)
		}
	}))
}

func newRuntimeKubernetesServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer kube-token" {
			t.Fatalf("expected kubernetes bearer token, got %q", got)
		}
		switch r.URL.Path {
		case "/apis/apps/v1/namespaces/prod/deployments/checkout":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": false},
				"status": map[string]any{
					"replicas":            4,
					"updatedReplicas":     4,
					"availableReplicas":   4,
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
						"replicas":            4,
						"updatedReplicas":     4,
						"availableReplicas":   4,
						"unavailableReplicas": 0,
						"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
					},
				}},
			})
		default:
			t.Fatalf("unexpected kubernetes path %s", r.URL.Path)
		}
	}))
}

func newRuntimePrometheusServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer prom-token" {
			t.Fatalf("expected prometheus bearer token, got %q", got)
		}
		if r.URL.Path != "/api/v1/query_range" {
			t.Fatalf("unexpected prometheus path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result": []map[string]any{{
					"values": [][]any{
						{float64(1), "100"},
						{float64(2), "250"},
					},
				}},
			},
		})
	}))
}

func decodeItemResponse[T any](t *testing.T, body []byte) T {
	t.Helper()

	var envelope types.ItemResponse[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func decodeListResponse[T any](t *testing.T, body []byte) []T {
	t.Helper()

	var envelope types.ListResponse[T]
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func doRequestBody(t *testing.T, method, requestURL string, body []byte, headers map[string]string, expectedStatus int) []byte {
	t.Helper()

	req, err := http.NewRequest(method, requestURL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	data := readAll(t, resp)
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d for %s %s, got %d: %s", expectedStatus, method, requestURL, resp.StatusCode, string(data))
	}
	return data
}

func readAll(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
