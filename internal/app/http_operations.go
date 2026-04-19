package app

import (
	"io"
	"net/http"
	"strings"

	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *HTTPServer) getOrganization(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetOrganization(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Organization]{Data: result})
}

func (s *HTTPServer) updateOrganization(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateOrganizationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateOrganization(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Organization]{Data: result})
}

func (s *HTTPServer) getProject(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetProject(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Project]{Data: result})
}

func (s *HTTPServer) updateProject(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateProjectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateProject(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Project]{Data: result})
}

func (s *HTTPServer) archiveProject(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ArchiveProject(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Project]{Data: result})
}

func (s *HTTPServer) getTeam(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetTeam(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Team]{Data: result})
}

func (s *HTTPServer) updateTeam(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateTeamRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateTeam(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Team]{Data: result})
}

func (s *HTTPServer) archiveTeam(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ArchiveTeam(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Team]{Data: result})
}

func (s *HTTPServer) getService(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetService(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Service]{Data: result})
}

func (s *HTTPServer) updateService(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateServiceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateService(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Service]{Data: result})
}

func (s *HTTPServer) archiveService(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ArchiveService(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Service]{Data: result})
}

func (s *HTTPServer) getEnvironment(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetEnvironment(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Environment]{Data: result})
}

func (s *HTTPServer) updateEnvironment(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateEnvironmentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateEnvironment(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Environment]{Data: result})
}

func (s *HTTPServer) archiveEnvironment(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ArchiveEnvironment(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Environment]{Data: result})
}

func (s *HTTPServer) getChange(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetChangeSet(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.ChangeSet]{Data: result})
}

func (s *HTTPServer) createIntegration(w http.ResponseWriter, r *http.Request) {
	var req types.CreateIntegrationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateIntegration(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.Integration]{Data: result})
}

func (s *HTTPServer) updateIntegration(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateIntegrationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateIntegration(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Integration]{Data: result})
}

func (s *HTTPServer) integrationCoverageSummary(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.CoverageSummary(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.CoverageSummary]{Data: result})
}

func (s *HTTPServer) testIntegration(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.TestIntegrationConnection(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.IntegrationTestResult]{Data: result})
}

func (s *HTTPServer) syncIntegration(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.SyncIntegration(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.IntegrationSyncResult]{Data: result})
}

func (s *HTTPServer) listIntegrationSyncRuns(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListIntegrationSyncRuns(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.IntegrationSyncRun]{Data: result})
}

func (s *HTTPServer) startGitHubOnboarding(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.StartGitHubOnboarding(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.GitHubOnboardingStartResult]{Data: result})
}

func (s *HTTPServer) completeGitHubOnboarding(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.CompleteGitHubOnboarding(r.Context(), r.URL.Query().Get("state"), r.URL.Query())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Integration]{Data: result})
}

func (s *HTTPServer) ingestIntegrationGraph(w http.ResponseWriter, r *http.Request) {
	var req types.IntegrationGraphIngestRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.IngestIntegrationGraph(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.GraphRelationship]{Data: result})
}

func (s *HTTPServer) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.HandleGitHubWebhook(r.Context(), r.PathValue("id"), r.Header, payload)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, types.ItemResponse[types.IntegrationSyncRun]{Data: result})
}

func (s *HTTPServer) handleGitLabWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.HandleGitLabWebhook(r.Context(), r.PathValue("id"), r.Header, payload)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, types.ItemResponse[types.IntegrationSyncRun]{Data: result})
}

func (s *HTTPServer) listGraphRelationships(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListGraphRelationships(r.Context(), storage.GraphRelationshipQuery{
		SourceIntegrationID: strings.TrimSpace(r.URL.Query().Get("source_integration_id")),
		RelationshipType:    strings.TrimSpace(r.URL.Query().Get("relationship_type")),
		FromResourceID:      strings.TrimSpace(r.URL.Query().Get("from_resource_id")),
		ToResourceID:        strings.TrimSpace(r.URL.Query().Get("to_resource_id")),
		Limit:               parseIntQuery(r, "limit", 200),
		Offset:              parseIntQuery(r, "offset", 0),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.GraphRelationship]{Data: result})
}

func (s *HTTPServer) listRepositories(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListRepositories(r.Context(), storage.RepositoryQuery{
		ProjectID:           strings.TrimSpace(r.URL.Query().Get("project_id")),
		ServiceID:           strings.TrimSpace(r.URL.Query().Get("service_id")),
		EnvironmentID:       strings.TrimSpace(r.URL.Query().Get("environment_id")),
		SourceIntegrationID: strings.TrimSpace(r.URL.Query().Get("source_integration_id")),
		Provider:            strings.TrimSpace(r.URL.Query().Get("provider")),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Repository]{Data: result})
}

func (s *HTTPServer) updateRepository(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateRepositoryRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateRepository(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Repository]{Data: result})
}

func (s *HTTPServer) listDiscoveredResources(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListDiscoveredResources(r.Context(), storage.DiscoveredResourceQuery{
		IntegrationID: strings.TrimSpace(r.URL.Query().Get("integration_id")),
		ResourceType:  strings.TrimSpace(r.URL.Query().Get("resource_type")),
		Provider:      strings.TrimSpace(r.URL.Query().Get("provider")),
		ProjectID:     strings.TrimSpace(r.URL.Query().Get("project_id")),
		ServiceID:     strings.TrimSpace(r.URL.Query().Get("service_id")),
		EnvironmentID: strings.TrimSpace(r.URL.Query().Get("environment_id")),
		RepositoryID:  strings.TrimSpace(r.URL.Query().Get("repository_id")),
		Status:        strings.TrimSpace(r.URL.Query().Get("status")),
		Search:        strings.TrimSpace(r.URL.Query().Get("search")),
		UnmappedOnly:  parseBoolQuery(r, "unmapped_only"),
		Limit:         parseIntQuery(r, "limit", 200),
		Offset:        parseIntQuery(r, "offset", 0),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.DiscoveredResource]{Data: result})
}

func (s *HTTPServer) updateDiscoveredResource(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateDiscoveredResourceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateDiscoveredResource(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DiscoveredResource]{Data: result})
}

func (s *HTTPServer) listServiceAccounts(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListServiceAccounts(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.ServiceAccount]{Data: result})
}

func (s *HTTPServer) createServiceAccount(w http.ResponseWriter, r *http.Request) {
	var req types.CreateServiceAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateServiceAccount(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.ServiceAccount]{Data: result})
}

func (s *HTTPServer) deactivateServiceAccount(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.DeactivateServiceAccount(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.ServiceAccount]{Data: result})
}

func (s *HTTPServer) listServiceAccountTokens(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListServiceAccountTokens(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.APIToken]{Data: result})
}

func (s *HTTPServer) issueServiceAccountToken(w http.ResponseWriter, r *http.Request) {
	var req types.IssueAPITokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.IssueServiceAccountToken(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.IssuedAPITokenResponse]{Data: result})
}

func (s *HTTPServer) revokeServiceAccountToken(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.RevokeAPIToken(r.Context(), r.PathValue("id"), r.PathValue("token_id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.APIToken]{Data: result})
}

func (s *HTTPServer) rotateServiceAccountToken(w http.ResponseWriter, r *http.Request) {
	var req types.RotateAPITokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.RotateAPIToken(r.Context(), r.PathValue("id"), r.PathValue("token_id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.IssuedAPITokenResponse]{Data: result})
}

func (s *HTTPServer) listRolloutExecutions(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListRolloutExecutions(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.RolloutExecution]{Data: result})
}

func (s *HTTPServer) createRolloutExecution(w http.ResponseWriter, r *http.Request) {
	var req types.CreateRolloutExecutionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateRolloutExecution(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.RolloutExecution]{Data: result})
}

func (s *HTTPServer) getRolloutExecution(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetRolloutExecutionDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.RolloutExecutionDetail]{Data: result})
}

func (s *HTTPServer) advanceRolloutExecution(w http.ResponseWriter, r *http.Request) {
	var req types.AdvanceRolloutExecutionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.AdvanceRolloutExecution(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.RolloutExecution]{Data: result})
}

func (s *HTTPServer) reconcileRolloutExecution(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ReconcileRolloutExecution(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.RolloutExecutionDetail]{Data: result})
}

func (s *HTTPServer) createSignalSnapshot(w http.ResponseWriter, r *http.Request) {
	var req types.CreateSignalSnapshotRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateSignalSnapshot(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.SignalSnapshot]{Data: result})
}

func (s *HTTPServer) recordVerificationResult(w http.ResponseWriter, r *http.Request) {
	var req types.RecordVerificationResultRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.RecordVerificationResult(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.VerificationResult]{Data: result})
}
