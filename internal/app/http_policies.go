package app

import (
	"net/http"
	"strings"

	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *HTTPServer) listPolicies(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListPolicies(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Policy]{Data: result})
}

func (s *HTTPServer) createPolicy(w http.ResponseWriter, r *http.Request) {
	var req types.CreatePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreatePolicy(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.Policy]{Data: result})
}

func (s *HTTPServer) getPolicy(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetPolicy(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Policy]{Data: result})
}

func (s *HTTPServer) updatePolicy(w http.ResponseWriter, r *http.Request) {
	var req types.UpdatePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdatePolicy(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.Policy]{Data: result})
}

func (s *HTTPServer) listPolicyDecisions(w http.ResponseWriter, r *http.Request) {
	query := storage.PolicyDecisionQuery{
		ProjectID:          strings.TrimSpace(r.URL.Query().Get("project_id")),
		PolicyID:           strings.TrimSpace(r.URL.Query().Get("policy_id")),
		ChangeSetID:        strings.TrimSpace(r.URL.Query().Get("change_set_id")),
		RiskAssessmentID:   strings.TrimSpace(r.URL.Query().Get("risk_assessment_id")),
		RolloutPlanID:      strings.TrimSpace(r.URL.Query().Get("rollout_plan_id")),
		RolloutExecutionID: strings.TrimSpace(r.URL.Query().Get("rollout_execution_id")),
		AppliesTo:          strings.TrimSpace(r.URL.Query().Get("applies_to")),
		Limit:              parseIntQuery(r, "limit", 50),
		Offset:             parseIntQuery(r, "offset", 0),
	}
	result, err := s.app.ListPolicyDecisions(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.PolicyDecision]{Data: result})
}
