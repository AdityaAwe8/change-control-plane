package app

import (
	"net/http"

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

func (s *HTTPServer) listGraphRelationships(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListGraphRelationships(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.GraphRelationship]{Data: result})
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
