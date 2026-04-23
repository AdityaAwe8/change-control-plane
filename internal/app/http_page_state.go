package app

import (
	"net/http"

	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *HTTPServer) getRolloutPageState(w http.ResponseWriter, r *http.Request) {
	rolloutPlans, err := s.app.ListRolloutPlans(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	rolloutExecutions, err := s.app.ListRolloutExecutions(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	integrations, err := s.app.IntegrationsList(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	var rolloutExecutionDetail *types.RolloutExecutionDetail
	if len(rolloutExecutions) > 0 {
		detail, err := s.app.GetRolloutExecutionDetail(r.Context(), rolloutExecutions[0].ID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		rolloutExecutionDetail = &detail
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.RolloutPageState]{Data: types.RolloutPageState{
		RolloutPlans:           rolloutPlans,
		RolloutExecutions:      rolloutExecutions,
		RolloutExecutionDetail: rolloutExecutionDetail,
		Integrations:           integrations,
	}})
}

func (s *HTTPServer) getDeploymentsPageState(w http.ResponseWriter, r *http.Request) {
	query, err := decodeStatusEventQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	catalog, err := s.app.Catalog(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	rollbackPolicies, err := s.app.ListRollbackPolicies(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	statusDashboard, err := s.app.QueryStatusEvents(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	coverageSummary, err := s.app.CoverageSummary(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DeploymentsPageState]{Data: types.DeploymentsPageState{
		Catalog:          catalog,
		RollbackPolicies: rollbackPolicies,
		StatusDashboard:  statusDashboard,
		CoverageSummary:  coverageSummary,
	}})
}

func (s *HTTPServer) getIntegrationsPageState(w http.ResponseWriter, r *http.Request) {
	catalog, err := s.app.Catalog(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	teams, err := s.app.ListTeams(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	integrations, err := s.app.IntegrationsList(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	coverageSummary, err := s.app.CoverageSummary(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	repositories, err := s.app.ListRepositories(r.Context(), storage.RepositoryQuery{Limit: 500})
	if err != nil {
		writeAppError(w, err)
		return
	}
	discoveredResources, err := s.app.ListDiscoveredResources(r.Context(), storage.DiscoveredResourceQuery{Limit: 500})
	if err != nil {
		writeAppError(w, err)
		return
	}
	integrationSyncRuns, err := s.integrationSyncRunsByID(r, integrations)
	if err != nil {
		writeAppError(w, err)
		return
	}
	webhookRegistrations, err := s.webhookRegistrationsByID(r, integrations)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.IntegrationsPageState]{Data: types.IntegrationsPageState{
		Catalog:              catalog,
		Teams:                teams,
		Integrations:         integrations,
		CoverageSummary:      coverageSummary,
		Repositories:         repositories,
		DiscoveredResources:  discoveredResources,
		IntegrationSyncRuns:  integrationSyncRuns,
		WebhookRegistrations: webhookRegistrations,
	}})
}

func (s *HTTPServer) getEnterprisePageState(w http.ResponseWriter, r *http.Request) {
	identityProviders, err := s.app.ListIdentityProviders(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	integrations, err := s.app.IntegrationsList(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	webhookRegistrations, err := s.webhookRegistrationsByID(r, integrations)
	if err != nil {
		writeAppError(w, err)
		return
	}
	outboxEvents, err := s.app.ListOutboxEvents(r.Context(), storage.OutboxEventQuery{Limit: 25})
	if err != nil {
		writeAppError(w, err)
		return
	}
	browserSessions, err := s.app.ListBrowserSessions(r.Context(), storage.BrowserSessionQuery{Limit: 25})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.EnterprisePageState]{Data: types.EnterprisePageState{
		IdentityProviders:    identityProviders,
		Integrations:         integrations,
		WebhookRegistrations: webhookRegistrations,
		OutboxEvents:         outboxEvents,
		BrowserSessions:      browserSessions,
	}})
}

func (s *HTTPServer) getGraphPageState(w http.ResponseWriter, r *http.Request) {
	graphRelationships, err := s.app.ListGraphRelationships(r.Context(), storage.GraphRelationshipQuery{Limit: 500})
	if err != nil {
		writeAppError(w, err)
		return
	}
	catalog, err := s.app.Catalog(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	integrations, err := s.app.IntegrationsList(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	projects, err := s.app.ListProjects(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	changes, err := s.app.ListChangeSets(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	teams, err := s.app.ListTeams(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	repositories, err := s.app.ListRepositories(r.Context(), storage.RepositoryQuery{Limit: 500})
	if err != nil {
		writeAppError(w, err)
		return
	}
	discoveredResources, err := s.app.ListDiscoveredResources(r.Context(), storage.DiscoveredResourceQuery{Limit: 500})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.GraphPageState]{Data: types.GraphPageState{
		GraphRelationships: graphRelationships,
		Catalog:            catalog,
		Integrations:       integrations,
		Projects:           projects,
		Teams:              teams,
		Repositories:       repositories,
		DiscoveredResources: discoveredResources,
		Changes:            changes,
	}})
}

func (s *HTTPServer) getSimulationPageState(w http.ResponseWriter, r *http.Request) {
	changes, err := s.app.ListChangeSets(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	riskAssessments, err := s.app.ListRiskAssessments(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	rolloutPlans, err := s.app.ListRolloutPlans(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	rolloutExecutions, err := s.app.ListRolloutExecutions(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	rollbackPolicies, err := s.app.ListRollbackPolicies(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	statusEvents, err := s.app.ListStatusEvents(r.Context(), storage.StatusEventQuery{Limit: 200})
	if err != nil {
		writeAppError(w, err)
		return
	}
	var rolloutExecutionDetail *types.RolloutExecutionDetail
	if len(rolloutExecutions) > 0 {
		detail, err := s.app.GetRolloutExecutionDetail(r.Context(), rolloutExecutions[0].ID)
		if err != nil {
			writeAppError(w, err)
			return
		}
		rolloutExecutionDetail = &detail
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.SimulationPageState]{Data: types.SimulationPageState{
		Changes:                changes,
		RiskAssessments:        riskAssessments,
		RolloutPlans:           rolloutPlans,
		RolloutExecutions:      rolloutExecutions,
		RolloutExecutionDetail: rolloutExecutionDetail,
		RollbackPolicies:       rollbackPolicies,
		StatusEvents:           statusEvents,
	}})
}

func (s *HTTPServer) integrationSyncRunsByID(r *http.Request, integrations []types.Integration) (map[string][]types.IntegrationSyncRun, error) {
	integrationSyncRuns := make(map[string][]types.IntegrationSyncRun, len(integrations))
	for _, integration := range integrations {
		runs, err := s.app.ListIntegrationSyncRuns(r.Context(), integration.ID)
		if err != nil {
			return nil, err
		}
		integrationSyncRuns[integration.ID] = runs
	}
	return integrationSyncRuns, nil
}

func (s *HTTPServer) webhookRegistrationsByID(r *http.Request, integrations []types.Integration) (map[string]*types.WebhookRegistration, error) {
	webhookRegistrations := make(map[string]*types.WebhookRegistration)
	for _, integration := range integrations {
		if !supportsPageStateWebhookRegistration(integration.Kind) {
			continue
		}
		result, err := s.app.GetWebhookRegistration(r.Context(), integration.ID)
		if err != nil {
			return nil, err
		}
		registration := result.Registration
		webhookRegistrations[integration.ID] = &registration
	}
	return webhookRegistrations, nil
}

func supportsPageStateWebhookRegistration(kind string) bool {
	return kind == "github" || kind == "gitlab"
}
