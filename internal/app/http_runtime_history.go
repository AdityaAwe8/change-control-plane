package app

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *HTTPServer) pauseRolloutExecution(w http.ResponseWriter, r *http.Request) {
	reason, err := decodeRolloutActionReason(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.AdvanceRolloutExecution(r.Context(), r.PathValue("id"), types.AdvanceRolloutExecutionRequest{
		Action: "pause",
		Reason: reason,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.RolloutExecution]{Data: result})
}

func (s *HTTPServer) resumeRolloutExecution(w http.ResponseWriter, r *http.Request) {
	reason, err := decodeRolloutActionReason(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.AdvanceRolloutExecution(r.Context(), r.PathValue("id"), types.AdvanceRolloutExecutionRequest{
		Action: "resume",
		Reason: reason,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.RolloutExecution]{Data: result})
}

func (s *HTTPServer) rollbackRolloutExecution(w http.ResponseWriter, r *http.Request) {
	reason, err := decodeRolloutActionReason(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.AdvanceRolloutExecution(r.Context(), r.PathValue("id"), types.AdvanceRolloutExecutionRequest{
		Action: "rollback",
		Reason: reason,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.RolloutExecution]{Data: result})
}

func (s *HTTPServer) listStatusEvents(w http.ResponseWriter, r *http.Request) {
	query, err := decodeStatusEventQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.ListStatusEvents(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.StatusEvent]{Data: result})
}

func (s *HTTPServer) searchStatusEvents(w http.ResponseWriter, r *http.Request) {
	query, err := decodeStatusEventQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.QueryStatusEvents(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.StatusEventQueryResult]{Data: result})
}

func (s *HTTPServer) getStatusEvent(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetStatusEvent(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.StatusEvent]{Data: result})
}

func (s *HTTPServer) listRolloutExecutionTimeline(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListRolloutExecutionStatusEvents(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.StatusEvent]{Data: result})
}

func (s *HTTPServer) listProjectStatusEvents(w http.ResponseWriter, r *http.Request) {
	query, err := decodeStatusEventQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	query.ProjectID = r.PathValue("id")
	result, err := s.app.ListStatusEvents(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.StatusEvent]{Data: result})
}

func (s *HTTPServer) listServiceStatusEvents(w http.ResponseWriter, r *http.Request) {
	query, err := decodeStatusEventQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	query.ServiceID = r.PathValue("id")
	result, err := s.app.ListStatusEvents(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.StatusEvent]{Data: result})
}

func (s *HTTPServer) listEnvironmentStatusEvents(w http.ResponseWriter, r *http.Request) {
	query, err := decodeStatusEventQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	query.EnvironmentID = r.PathValue("id")
	result, err := s.app.ListStatusEvents(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.StatusEvent]{Data: result})
}

func (s *HTTPServer) listRollbackPolicies(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListRollbackPolicies(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.RollbackPolicy]{Data: result})
}

func (s *HTTPServer) createRollbackPolicy(w http.ResponseWriter, r *http.Request) {
	var req types.CreateRollbackPolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateRollbackPolicy(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.RollbackPolicy]{Data: result})
}

func (s *HTTPServer) updateRollbackPolicy(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateRollbackPolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateRollbackPolicy(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.RollbackPolicy]{Data: result})
}

func decodeStatusEventQuery(r *http.Request) (storage.StatusEventQuery, error) {
	query := storage.StatusEventQuery{
		ProjectID:          strings.TrimSpace(r.URL.Query().Get("project_id")),
		TeamID:             strings.TrimSpace(r.URL.Query().Get("team_id")),
		ServiceID:          strings.TrimSpace(r.URL.Query().Get("service_id")),
		EnvironmentID:      strings.TrimSpace(r.URL.Query().Get("environment_id")),
		RolloutExecutionID: strings.TrimSpace(r.URL.Query().Get("rollout_execution_id")),
		ChangeSetID:        strings.TrimSpace(r.URL.Query().Get("change_set_id")),
		ResourceType:       strings.TrimSpace(r.URL.Query().Get("resource_type")),
		ResourceID:         strings.TrimSpace(r.URL.Query().Get("resource_id")),
		ActorType:          strings.TrimSpace(r.URL.Query().Get("actor_type")),
		ActorID:            strings.TrimSpace(r.URL.Query().Get("actor_id")),
		Source:             strings.TrimSpace(r.URL.Query().Get("source")),
		Outcome:            strings.TrimSpace(r.URL.Query().Get("outcome")),
		Search:             strings.TrimSpace(r.URL.Query().Get("search")),
		RollbackOnly:       parseBoolQuery(r, "rollback_only"),
		Limit:              parseIntQuery(r, "limit", 100),
		Offset:             parseIntQuery(r, "offset", 0),
	}
	if rawEventTypes := strings.TrimSpace(r.URL.Query().Get("event_type")); rawEventTypes != "" {
		query.EventTypes = splitCSV(rawEventTypes)
	}
	if rawAutomated := strings.TrimSpace(r.URL.Query().Get("automated")); rawAutomated != "" {
		value := parseBoolQuery(r, "automated")
		query.Automated = &value
	}
	if rawSince := strings.TrimSpace(r.URL.Query().Get("since")); rawSince != "" {
		parsed, err := time.Parse(time.RFC3339, rawSince)
		if err != nil {
			return storage.StatusEventQuery{}, err
		}
		query.Since = &parsed
	}
	if rawUntil := strings.TrimSpace(r.URL.Query().Get("until")); rawUntil != "" {
		parsed, err := time.Parse(time.RFC3339, rawUntil)
		if err != nil {
			return storage.StatusEventQuery{}, err
		}
		query.Until = &parsed
	}
	return query, nil
}

func parseIntQuery(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseBoolQuery(r *http.Request, key string) bool {
	raw := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	return raw == "1" || raw == "true" || raw == "yes"
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

func decodeRolloutActionReason(r *http.Request) (string, error) {
	queryReason := strings.TrimSpace(r.URL.Query().Get("reason"))
	if r.ContentLength == 0 {
		return queryReason, nil
	}
	if strings.TrimSpace(r.Header.Get("Content-Type")) == "" && queryReason != "" {
		return queryReason, nil
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		if errors.Is(err, http.ErrBodyNotAllowed) {
			return queryReason, nil
		}
		return "", err
	}
	if reason := strings.TrimSpace(payload.Reason); reason != "" {
		return reason, nil
	}
	return queryReason, nil
}
