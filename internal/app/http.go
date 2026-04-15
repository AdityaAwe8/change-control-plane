package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
	return s.mux
}

func (s *HTTPServer) routes() {
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	s.mux.HandleFunc("GET /readyz", s.handleReady)

	s.mux.HandleFunc("POST /api/v1/auth/dev/login", s.handleDevLogin)
	s.mux.HandleFunc("GET /api/v1/auth/session", s.withOptionalAuth(s.handleSession))
	s.mux.HandleFunc("GET /api/v1/catalog", s.withAuth(s.handleCatalog))
	s.mux.HandleFunc("GET /api/v1/metrics/basics", s.withAuth(s.handleMetrics))
	s.mux.HandleFunc("GET /api/v1/incidents", s.withAuth(s.handleIncidents))
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
	s.mux.HandleFunc("POST /api/v1/rollout-executions/{id}/advance", s.withAuth(s.advanceRolloutExecution))
	s.mux.HandleFunc("POST /api/v1/rollout-executions/{id}/verification", s.withAuth(s.recordVerificationResult))
	s.mux.HandleFunc("GET /api/v1/policies", s.withAuth(s.listPolicies))
	s.mux.HandleFunc("GET /api/v1/audit-events", s.withAuth(s.listAuditEvents))
	s.mux.HandleFunc("GET /api/v1/integrations", s.withAuth(s.listIntegrations))
	s.mux.HandleFunc("PATCH /api/v1/integrations/{id}", s.withAuth(s.updateIntegration))
	s.mux.HandleFunc("POST /api/v1/integrations/{id}/graph-ingest", s.withAuth(s.ingestIntegrationGraph))
	s.mux.HandleFunc("GET /api/v1/graph/relationships", s.withAuth(s.listGraphRelationships))
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
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DevLoginResponse]{Data: result})
}

func (s *HTTPServer) handleSession(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, types.ItemResponse[types.SessionInfo]{Data: s.app.Session(r.Context())})
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
	result, err := s.app.Incidents(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Incident]{Data: result})
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

func (s *HTTPServer) listPolicies(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.PoliciesList(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Policy]{Data: result})
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
	result, err := s.app.IntegrationsList(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Integration]{Data: result})
}

func (s *HTTPServer) withAuth(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity, err := s.app.Auth.LoadIdentity(r.Context(), r.Header.Get("Authorization"), requestOrganizationScope(r))
		if err != nil {
			writeAppError(w, err)
			return
		}
		next(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	}
}

func (s *HTTPServer) withOptionalAuth(next func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.Header.Get("Authorization")) == "" {
			next(w, r)
			return
		}
		identity, err := s.app.Auth.LoadIdentity(r.Context(), r.Header.Get("Authorization"), requestOrganizationScope(r))
		if err != nil {
			writeAppError(w, err)
			return
		}
		next(w, r.WithContext(auth.WithIdentity(r.Context(), identity)))
	}
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
	return decoder.Decode(v)
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
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	}
}
