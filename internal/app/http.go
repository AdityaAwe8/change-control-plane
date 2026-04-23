package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type HTTPServer struct {
	app *Application
	mux *http.ServeMux
}

func NewHTTPServer(app *Application) *HTTPServer {
	server := &HTTPServer{
		app: app,
		mux: http.NewServeMux(),
	}
	server.routes()
	return server
}

func (s *HTTPServer) Handler() http.Handler {
	return s.withCORS(s.mux)
}

func (s *HTTPServer) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)

	s.mux.HandleFunc("POST /api/v1/auth/sign-up", s.handleSignUp)
	s.mux.HandleFunc("POST /api/v1/auth/sign-in", s.handleSignIn)
	s.mux.HandleFunc("POST /api/v1/auth/dev/login", s.handleDevLogin)
	s.mux.HandleFunc("GET /api/v1/auth/providers", s.listPublicIdentityProviders)
	s.mux.HandleFunc("POST /api/v1/auth/providers/{id}/start", s.startIdentityProviderSignIn)
	s.mux.HandleFunc("GET /api/v1/auth/providers/callback", s.completeIdentityProviderSignIn)
	s.mux.HandleFunc("GET /api/v1/auth/session", s.withOptionalAuth(s.handleSession))
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.withOptionalAuth(s.handleLogout))
	s.mux.HandleFunc("GET /api/v1/catalog", s.withAuth(s.handleCatalog))
	s.mux.HandleFunc("GET /api/v1/metrics/basics", s.withAuth(s.handleMetrics))
	s.mux.HandleFunc("GET /api/v1/incidents", s.withAuth(s.handleIncidents))
	s.mux.HandleFunc("GET /api/v1/incidents/{id}", s.withAuth(s.getIncident))
	s.mux.HandleFunc("GET /api/v1/organizations", s.withAuth(s.listOrganizations))
	s.mux.HandleFunc("POST /api/v1/organizations", s.withAuth(s.createOrganization))
	s.mux.HandleFunc("GET /api/v1/organizations/{id}", s.withAuth(s.getOrganization))
	s.mux.HandleFunc("PATCH /api/v1/organizations/{id}", s.withAuth(s.updateOrganization))
	s.mux.HandleFunc("GET /api/v1/projects", s.withAuth(s.listProjects))
	s.mux.HandleFunc("POST /api/v1/projects", s.withAuth(s.createProject))
	s.mux.HandleFunc("GET /api/v1/projects/{id}", s.withAuth(s.getProject))
	s.mux.HandleFunc("PATCH /api/v1/projects/{id}", s.withAuth(s.updateProject))
	s.mux.HandleFunc("POST /api/v1/projects/{id}/archive", s.withAuth(s.archiveProject))
	s.mux.HandleFunc("GET /api/v1/teams", s.withAuth(s.listTeams))
	s.mux.HandleFunc("POST /api/v1/teams", s.withAuth(s.createTeam))
	s.mux.HandleFunc("GET /api/v1/teams/{id}", s.withAuth(s.getTeam))
	s.mux.HandleFunc("PATCH /api/v1/teams/{id}", s.withAuth(s.updateTeam))
	s.mux.HandleFunc("POST /api/v1/teams/{id}/archive", s.withAuth(s.archiveTeam))
	s.mux.HandleFunc("GET /api/v1/services", s.withAuth(s.listServices))
	s.mux.HandleFunc("POST /api/v1/services", s.withAuth(s.createService))
	s.mux.HandleFunc("GET /api/v1/services/{id}", s.withAuth(s.getService))
	s.mux.HandleFunc("PATCH /api/v1/services/{id}", s.withAuth(s.updateService))
	s.mux.HandleFunc("POST /api/v1/services/{id}/archive", s.withAuth(s.archiveService))
	s.mux.HandleFunc("GET /api/v1/environments", s.withAuth(s.listEnvironments))
	s.mux.HandleFunc("POST /api/v1/environments", s.withAuth(s.createEnvironment))
	s.mux.HandleFunc("GET /api/v1/environments/{id}", s.withAuth(s.getEnvironment))
	s.mux.HandleFunc("PATCH /api/v1/environments/{id}", s.withAuth(s.updateEnvironment))
	s.mux.HandleFunc("POST /api/v1/environments/{id}/archive", s.withAuth(s.archiveEnvironment))
	s.mux.HandleFunc("GET /api/v1/changes", s.withAuth(s.listChanges))
	s.mux.HandleFunc("POST /api/v1/changes", s.withAuth(s.createChange))
	s.mux.HandleFunc("GET /api/v1/changes/{id}", s.withAuth(s.getChange))
	s.mux.HandleFunc("GET /api/v1/risk-assessments", s.withAuth(s.listRiskAssessments))
	s.mux.HandleFunc("POST /api/v1/risk-assessments", s.withAuth(s.createRiskAssessment))
	s.mux.HandleFunc("GET /api/v1/rollout-plans", s.withAuth(s.listRolloutPlans))
	s.mux.HandleFunc("POST /api/v1/rollout-plans", s.withAuth(s.createRolloutPlan))
	s.mux.HandleFunc("GET /api/v1/rollout-executions", s.withAuth(s.listRolloutExecutions))
	s.mux.HandleFunc("POST /api/v1/rollout-executions", s.withAuth(s.createRolloutExecution))
	s.mux.HandleFunc("GET /api/v1/rollout-executions/{id}", s.withAuth(s.getRolloutExecution))
	s.mux.HandleFunc("GET /api/v1/rollout-executions/{id}/evidence-pack", s.withAuth(s.getRolloutExecutionEvidencePack))
	s.mux.HandleFunc("POST /api/v1/rollout-executions/{id}/advance", s.withAuth(s.advanceRolloutExecution))
	s.mux.HandleFunc("POST /api/v1/rollout-executions/{id}/pause", s.withAuth(s.pauseRolloutExecution))
	s.mux.HandleFunc("POST /api/v1/rollout-executions/{id}/resume", s.withAuth(s.resumeRolloutExecution))
	s.mux.HandleFunc("POST /api/v1/rollout-executions/{id}/rollback", s.withAuth(s.rollbackRolloutExecution))
	s.mux.HandleFunc("POST /api/v1/rollout-executions/{id}/reconcile", s.withAuth(s.reconcileRolloutExecution))
	s.mux.HandleFunc("POST /api/v1/rollout-executions/{id}/signal-snapshots", s.withAuth(s.createSignalSnapshot))
	s.mux.HandleFunc("POST /api/v1/rollout-executions/{id}/verification", s.withAuth(s.recordVerificationResult))
	s.mux.HandleFunc("GET /api/v1/rollout-executions/{id}/timeline", s.withAuth(s.listRolloutExecutionTimeline))
	s.mux.HandleFunc("GET /api/v1/page-state/rollout", s.withAuth(s.getRolloutPageState))
	s.mux.HandleFunc("GET /api/v1/policies", s.withAuth(s.listPolicies))
	s.mux.HandleFunc("POST /api/v1/policies", s.withAuth(s.createPolicy))
	s.mux.HandleFunc("GET /api/v1/policies/{id}", s.withAuth(s.getPolicy))
	s.mux.HandleFunc("PATCH /api/v1/policies/{id}", s.withAuth(s.updatePolicy))
	s.mux.HandleFunc("GET /api/v1/policy-decisions", s.withAuth(s.listPolicyDecisions))
	s.mux.HandleFunc("GET /api/v1/audit-events", s.withAuth(s.listAuditEvents))
	s.mux.HandleFunc("GET /api/v1/status-events", s.withAuth(s.listStatusEvents))
	s.mux.HandleFunc("GET /api/v1/status-events/search", s.withAuth(s.searchStatusEvents))
	s.mux.HandleFunc("GET /api/v1/page-state/deployments", s.withAuth(s.getDeploymentsPageState))
	s.mux.HandleFunc("GET /api/v1/status-events/{id}", s.withAuth(s.getStatusEvent))
	s.mux.HandleFunc("GET /api/v1/projects/{id}/status-events", s.withAuth(s.listProjectStatusEvents))
	s.mux.HandleFunc("GET /api/v1/services/{id}/status-events", s.withAuth(s.listServiceStatusEvents))
	s.mux.HandleFunc("GET /api/v1/environments/{id}/status-events", s.withAuth(s.listEnvironmentStatusEvents))
	s.mux.HandleFunc("GET /api/v1/page-state/graph", s.withAuth(s.getGraphPageState))
	s.mux.HandleFunc("GET /api/v1/page-state/simulation", s.withAuth(s.getSimulationPageState))
	s.mux.HandleFunc("GET /api/v1/rollback-policies", s.withAuth(s.listRollbackPolicies))
	s.mux.HandleFunc("POST /api/v1/rollback-policies", s.withAuth(s.createRollbackPolicy))
	s.mux.HandleFunc("PATCH /api/v1/rollback-policies/{id}", s.withAuth(s.updateRollbackPolicy))
	s.mux.HandleFunc("GET /api/v1/integrations", s.withAuth(s.listIntegrations))
	s.mux.HandleFunc("GET /api/v1/page-state/integrations", s.withAuth(s.getIntegrationsPageState))
	s.mux.HandleFunc("POST /api/v1/integrations", s.withAuth(s.createIntegration))
	s.mux.HandleFunc("GET /api/v1/integrations/coverage", s.withAuth(s.integrationCoverageSummary))
	s.mux.HandleFunc("PATCH /api/v1/integrations/{id}", s.withAuth(s.updateIntegration))
	s.mux.HandleFunc("POST /api/v1/integrations/{id}/test", s.withAuth(s.testIntegration))
	s.mux.HandleFunc("POST /api/v1/integrations/{id}/sync", s.withAuth(s.syncIntegration))
	s.mux.HandleFunc("GET /api/v1/integrations/{id}/sync-runs", s.withAuth(s.listIntegrationSyncRuns))
	s.mux.HandleFunc("GET /api/v1/integrations/{id}/webhook-registration", s.withAuth(s.getWebhookRegistration))
	s.mux.HandleFunc("POST /api/v1/integrations/{id}/webhook-registration/sync", s.withAuth(s.syncWebhookRegistration))
	s.mux.HandleFunc("POST /api/v1/integrations/{id}/github/onboarding/start", s.withAuth(s.startGitHubOnboarding))
	s.mux.HandleFunc("POST /api/v1/integrations/{id}/graph-ingest", s.withAuth(s.ingestIntegrationGraph))
	s.mux.HandleFunc("POST /api/v1/integrations/{id}/webhooks/github", s.handleGitHubWebhook)
	s.mux.HandleFunc("POST /api/v1/integrations/{id}/webhooks/gitlab", s.handleGitLabWebhook)
	s.mux.HandleFunc("GET /api/v1/integrations/github/callback", s.completeGitHubOnboarding)
	s.mux.HandleFunc("GET /api/v1/graph/relationships", s.withAuth(s.listGraphRelationships))
	s.mux.HandleFunc("GET /api/v1/repositories", s.withAuth(s.listRepositories))
	s.mux.HandleFunc("PATCH /api/v1/repositories/{id}", s.withAuth(s.updateRepository))
	s.mux.HandleFunc("GET /api/v1/discovered-resources", s.withAuth(s.listDiscoveredResources))
	s.mux.HandleFunc("PATCH /api/v1/discovered-resources/{id}", s.withAuth(s.updateDiscoveredResource))
	s.mux.HandleFunc("GET /api/v1/identity-providers", s.withAuth(s.listIdentityProviders))
	s.mux.HandleFunc("GET /api/v1/browser-sessions", s.withAuth(s.listBrowserSessions))
	s.mux.HandleFunc("GET /api/v1/page-state/enterprise", s.withAuth(s.getEnterprisePageState))
	s.mux.HandleFunc("POST /api/v1/identity-providers", s.withAuth(s.createIdentityProvider))
	s.mux.HandleFunc("PATCH /api/v1/identity-providers/{id}", s.withAuth(s.updateIdentityProvider))
	s.mux.HandleFunc("POST /api/v1/identity-providers/{id}/test", s.withAuth(s.testIdentityProvider))
	s.mux.HandleFunc("POST /api/v1/browser-sessions/{id}/revoke", s.withAuth(s.revokeBrowserSessionByID))
	s.mux.HandleFunc("GET /api/v1/outbox-events", s.withAuth(s.listOutboxEvents))
	s.mux.HandleFunc("POST /api/v1/outbox-events/{id}/retry", s.withAuth(s.retryOutboxEvent))
	s.mux.HandleFunc("POST /api/v1/outbox-events/{id}/requeue", s.withAuth(s.requeueOutboxEvent))
	s.mux.HandleFunc("GET /api/v1/service-accounts", s.withAuth(s.listServiceAccounts))
	s.mux.HandleFunc("POST /api/v1/service-accounts", s.withAuth(s.createServiceAccount))
	s.mux.HandleFunc("POST /api/v1/service-accounts/{id}/deactivate", s.withAuth(s.deactivateServiceAccount))
	s.mux.HandleFunc("GET /api/v1/service-accounts/{id}/tokens", s.withAuth(s.listServiceAccountTokens))
	s.mux.HandleFunc("POST /api/v1/service-accounts/{id}/tokens", s.withAuth(s.issueServiceAccountToken))
	s.mux.HandleFunc("POST /api/v1/service-accounts/{id}/tokens/{token_id}/revoke", s.withAuth(s.revokeServiceAccountToken))
	s.mux.HandleFunc("POST /api/v1/service-accounts/{id}/tokens/{token_id}/rotate", s.withAuth(s.rotateServiceAccountToken))
}

func (s *HTTPServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, types.ItemResponse[types.HealthResponse]{Data: types.HealthResponse{Status: "ok", Service: "change-control-plane-api"}})
}

func (s *HTTPServer) handleReady(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, types.ItemResponse[types.HealthResponse]{Data: types.HealthResponse{Status: "ready", Service: "change-control-plane-api"}})
}

func (s *HTTPServer) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	var req types.DevLoginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.DevLogin(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if err := s.issueBrowserSessionCookie(r.Context(), w, result.Session); err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DevLoginResponse]{Data: result})
}

func (s *HTTPServer) handleSignUp(w http.ResponseWriter, r *http.Request) {
	var req types.SignUpRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.SignUp(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if err := s.issueBrowserSessionCookie(r.Context(), w, result.Session); err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.AuthResponse]{Data: result})
}

func (s *HTTPServer) handleSignIn(w http.ResponseWriter, r *http.Request) {
	var req types.SignInRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.SignIn(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	if err := s.issueBrowserSessionCookie(r.Context(), w, result.Session); err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.AuthResponse]{Data: result})
}

func (s *HTTPServer) handleSession(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, types.ItemResponse[types.SessionInfo]{Data: s.app.Session(r.Context())})
}

func (s *HTTPServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := s.app.RevokeBrowserSession(r.Context(), s.readBrowserSessionCookie(r)); err != nil {
		writeAppError(w, err)
		return
	}
	s.clearBrowserSessionCookie(w)
	writeJSON(w, http.StatusOK, types.ItemResponse[types.SessionInfo]{Data: types.SessionInfo{
		Authenticated: false,
		Mode:          s.app.Config.AuthMode,
		Actor:         "anonymous",
	}})
}

func (s *HTTPServer) handleCatalog(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.Catalog(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.CatalogSummary]{Data: result})
}

func (s *HTTPServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.Metrics(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.BasicMetrics]{Data: result})
}

func (s *HTTPServer) handleIncidents(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.Incidents(r.Context(), decodeIncidentQuery(r))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Incident]{Data: result})
}

func (s *HTTPServer) getIncident(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetIncidentDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.IncidentDetail]{Data: result})
}

func decodeIncidentQuery(r *http.Request) IncidentQuery {
	return IncidentQuery{
		ProjectID:     strings.TrimSpace(r.URL.Query().Get("project_id")),
		ServiceID:     strings.TrimSpace(r.URL.Query().Get("service_id")),
		EnvironmentID: strings.TrimSpace(r.URL.Query().Get("environment_id")),
		ChangeSetID:   strings.TrimSpace(r.URL.Query().Get("change_set_id")),
		Severity:      strings.TrimSpace(r.URL.Query().Get("severity")),
		Status:        strings.TrimSpace(r.URL.Query().Get("status")),
		Search:        strings.TrimSpace(r.URL.Query().Get("search")),
		Limit:         parseIntQuery(r, "limit", 0),
	}
}

func (s *HTTPServer) listOrganizations(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListOrganizations(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Organization]{Data: result})
}

func (s *HTTPServer) createOrganization(w http.ResponseWriter, r *http.Request) {
	var req types.CreateOrganizationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	org, err := s.app.CreateOrganization(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.Organization]{Data: org})
}

func (s *HTTPServer) listProjects(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListProjects(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Project]{Data: result})
}

func (s *HTTPServer) createProject(w http.ResponseWriter, r *http.Request) {
	var req types.CreateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	project, err := s.app.CreateProject(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.Project]{Data: project})
}

func (s *HTTPServer) listTeams(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListTeams(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Team]{Data: result})
}

func (s *HTTPServer) createTeam(w http.ResponseWriter, r *http.Request) {
	var req types.CreateTeamRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	team, err := s.app.CreateTeam(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.Team]{Data: team})
}

func (s *HTTPServer) listServices(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListServices(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Service]{Data: result})
}

func (s *HTTPServer) createService(w http.ResponseWriter, r *http.Request) {
	var req types.CreateServiceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	service, err := s.app.CreateService(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.Service]{Data: service})
}

func (s *HTTPServer) listEnvironments(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListEnvironments(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Environment]{Data: result})
}

func (s *HTTPServer) createEnvironment(w http.ResponseWriter, r *http.Request) {
	var req types.CreateEnvironmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	environment, err := s.app.CreateEnvironment(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.Environment]{Data: environment})
}

func (s *HTTPServer) listChanges(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListChangeSets(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.ChangeSet]{Data: result})
}

func (s *HTTPServer) createChange(w http.ResponseWriter, r *http.Request) {
	var req types.CreateChangeSetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	change, err := s.app.CreateChangeSet(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.ChangeSet]{Data: change})
}

func (s *HTTPServer) listRiskAssessments(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListRiskAssessments(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.RiskAssessment]{Data: result})
}

func (s *HTTPServer) createRiskAssessment(w http.ResponseWriter, r *http.Request) {
	var req types.CreateRiskAssessmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.AssessRisk(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.RiskAssessmentResult]{Data: result})
}

func (s *HTTPServer) listRolloutPlans(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListRolloutPlans(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.RolloutPlan]{Data: result})
}

func (s *HTTPServer) createRolloutPlan(w http.ResponseWriter, r *http.Request) {
	var req types.CreateRolloutPlanRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateRolloutPlan(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.RolloutPlanResult]{Data: result})
}

func (s *HTTPServer) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.AuditEvents(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.AuditEvent]{Data: result})
}

func (s *HTTPServer) listIntegrations(w http.ResponseWriter, r *http.Request) {
	query := storage.IntegrationQuery{
		Kind:         strings.TrimSpace(r.URL.Query().Get("kind")),
		InstanceKey:  strings.TrimSpace(r.URL.Query().Get("instance_key")),
		ScopeType:    strings.TrimSpace(r.URL.Query().Get("scope_type")),
		AuthStrategy: strings.TrimSpace(r.URL.Query().Get("auth_strategy")),
		Search:       strings.TrimSpace(r.URL.Query().Get("search")),
	}
	if enabledRaw := strings.TrimSpace(r.URL.Query().Get("enabled")); enabledRaw != "" {
		if enabled, err := strconv.ParseBool(enabledRaw); err == nil {
			query.Enabled = &enabled
		}
	}
	result, err := s.app.ListIntegrationsWithQuery(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Integration]{Data: result})
}

func (s *HTTPServer) withAuth(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, source, err := s.resolveIdentity(r)
		if err != nil {
			writeAppError(w, err)
			return
		}
		if source == authSourceBrowserSession {
			if err := s.validateCookieMutationOrigin(r); err != nil {
				writeAppError(w, err)
				return
			}
		}
		next(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	}
}

func (s *HTTPServer) withOptionalAuth(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.requestCarriesAuth(r) {
			next(w, r)
			return
		}
		identity, source, err := s.resolveIdentity(r)
		if err != nil {
			writeAppError(w, err)
			return
		}
		if source == authSourceBrowserSession {
			if err := s.validateCookieMutationOrigin(r); err != nil {
				writeAppError(w, err)
				return
			}
		}
		next(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	}
}

type authSource string

const (
	authSourceAnonymous      authSource = "anonymous"
	authSourceBearer         authSource = "bearer"
	authSourceBrowserSession authSource = "browser_session"
)

func (s *HTTPServer) resolveIdentity(r *http.Request) (auth.Identity, authSource, error) {
	scope := requestOrganizationScope(r)
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		identity, err := s.app.Auth.LoadIdentity(r.Context(), authorization, scope)
		return identity, authSourceBearer, err
	}
	if rawSession := s.readBrowserSessionCookie(r); rawSession != "" {
		identity, err := s.app.Auth.LoadIdentityFromBrowserSession(r.Context(), rawSession, scope)
		return identity, authSourceBrowserSession, err
	}
	return auth.Identity{}, authSourceAnonymous, ErrUnauthorized
}

func (s *HTTPServer) requestCarriesAuth(r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		return true
	}
	return s.readBrowserSessionCookie(r) != ""
}

func requestOrganizationScope(r *http.Request) string {
	if scoped := strings.TrimSpace(r.Header.Get("X-CCP-Organization-ID")); scoped != "" {
		return scoped
	}
	return strings.TrimSpace(r.URL.Query().Get("organization_id"))
}

func decodeJSON(r *http.Request, v any) error {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") && r.Header.Get("Content-Type") != "" {
		return errors.New("content type must be application/json")
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, types.ErrorResponse{Error: types.ErrorDetail{Code: code, Message: message}})
}

func writeAppError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized), errors.Is(err, auth.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, ErrForbidden), errors.Is(err, auth.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, storage.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, ErrValidation):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func (s *HTTPServer) withCORS(next http.Handler) http.Handler {
	allowedOrigins := parseAllowedOrigins(s.app.Config.AllowedOrigins)
	allowAllDevOrigins := len(allowedOrigins) == 0 && strings.EqualFold(strings.TrimSpace(s.app.Config.Environment), "development")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if allowAllDevOrigins || originAllowed(origin, allowedOrigins) {
				setCORSHeaders(w.Header(), origin)
			} else if r.Method == http.MethodOptions {
				writeError(w, http.StatusForbidden, "forbidden", "origin is not allowed")
				return
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseAllowedOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	allowed := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	return allowed
}

func originAllowed(origin string, allowed []string) bool {
	for _, candidate := range allowed {
		if candidate == "*" || strings.EqualFold(origin, candidate) {
			return true
		}
	}
	return false
}

func setCORSHeaders(header http.Header, origin string) {
	header.Set("Access-Control-Allow-Origin", origin)
	appendVary(header, "Origin")
	header.Set("Access-Control-Allow-Headers", strings.Join([]string{
		"Authorization",
		"Content-Type",
		"X-CCP-Organization-ID",
	}, ", "))
	header.Set("Access-Control-Allow-Methods", strings.Join([]string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPatch,
		http.MethodOptions,
	}, ", "))
	header.Set("Access-Control-Allow-Credentials", "true")
	header.Set("Access-Control-Max-Age", "600")
}

func appendVary(header http.Header, value string) {
	current := header.Values("Vary")
	for _, existing := range current {
		for _, part := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) {
				return
			}
		}
	}
	header.Add("Vary", value)
}

func (s *HTTPServer) issueBrowserSessionCookie(ctx context.Context, w http.ResponseWriter, session types.SessionInfo) error {
	rawToken, err := s.app.IssueBrowserSession(ctx, session)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     s.browserSessionCookieName(),
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.browserSessionCookieSecure(),
		Expires:  time.Now().UTC().Add(s.app.browserSessionTTL()),
		MaxAge:   int(s.app.browserSessionTTL().Seconds()),
	})
	return nil
}

func (s *HTTPServer) clearBrowserSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.browserSessionCookieName(),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.browserSessionCookieSecure(),
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	})
}

func (s *HTTPServer) readBrowserSessionCookie(r *http.Request) string {
	cookie, err := r.Cookie(s.browserSessionCookieName())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func (s *HTTPServer) browserSessionCookieName() string {
	if strings.TrimSpace(s.app.Config.SessionCookieName) != "" {
		return strings.TrimSpace(s.app.Config.SessionCookieName)
	}
	return "ccp_session"
}

func (s *HTTPServer) browserSessionCookieSecure() bool {
	apiBaseURL := strings.ToLower(strings.TrimSpace(s.app.Config.APIBaseURL))
	if strings.HasPrefix(apiBaseURL, "https://") {
		return true
	}
	return !strings.EqualFold(strings.TrimSpace(s.app.Config.Environment), "development")
}

func (s *HTTPServer) validateCookieMutationOrigin(r *http.Request) error {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return nil
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		referer := strings.TrimSpace(r.Header.Get("Referer"))
		if referer != "" {
			parsed, err := url.Parse(referer)
			if err == nil && parsed.Scheme != "" && parsed.Host != "" {
				origin = parsed.Scheme + "://" + parsed.Host
			}
		}
	}
	if origin == "" {
		return ErrForbidden
	}
	if strings.EqualFold(origin, apiOrigin(s.app.Config.APIBaseURL)) {
		return nil
	}
	allowedOrigins := parseAllowedOrigins(s.app.Config.AllowedOrigins)
	allowAllDevOrigins := len(allowedOrigins) == 0 && strings.EqualFold(strings.TrimSpace(s.app.Config.Environment), "development")
	if allowAllDevOrigins || originAllowed(origin, allowedOrigins) {
		return nil
	}
	return ErrForbidden
}

func apiOrigin(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}
