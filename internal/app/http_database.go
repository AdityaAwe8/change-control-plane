package app

import (
	"net/http"
	"strings"

	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *HTTPServer) listDatabaseConnectionReferences(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListDatabaseConnectionReferences(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.DatabaseConnectionReference]{Data: result})
}

func (s *HTTPServer) createDatabaseConnectionReference(w http.ResponseWriter, r *http.Request) {
	var req types.CreateDatabaseConnectionReferenceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateDatabaseConnectionReference(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.DatabaseConnectionReferenceDetail]{Data: result})
}

func (s *HTTPServer) getDatabaseConnectionReference(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetDatabaseConnectionReferenceDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DatabaseConnectionReferenceDetail]{Data: result})
}

func (s *HTTPServer) updateDatabaseConnectionReference(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateDatabaseConnectionReferenceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateDatabaseConnectionReference(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DatabaseConnectionReferenceDetail]{Data: result})
}

func (s *HTTPServer) testDatabaseConnectionReference(w http.ResponseWriter, r *http.Request) {
	var req types.TestDatabaseConnectionReferenceRequest
	if err := decodeJSON(r, &req); err != nil && !strings.Contains(strings.ToLower(err.Error()), "eof") {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.TestDatabaseConnectionReference(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.DatabaseConnectionTestDetail]{Data: result})
}

func (s *HTTPServer) listDatabaseConnectionTests(w http.ResponseWriter, r *http.Request) {
	query := storage.DatabaseConnectionTestQuery{
		ProjectID:       strings.TrimSpace(r.URL.Query().Get("project_id")),
		EnvironmentID:   strings.TrimSpace(r.URL.Query().Get("environment_id")),
		ServiceID:       strings.TrimSpace(r.URL.Query().Get("service_id")),
		ConnectionRefID: strings.TrimSpace(r.URL.Query().Get("connection_ref_id")),
		Status:          strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:           500,
	}
	result, err := s.app.ListDatabaseConnectionTests(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.DatabaseConnectionTest]{Data: result})
}

func (s *HTTPServer) getDatabaseConnectionTest(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetDatabaseConnectionTestDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DatabaseConnectionTestDetail]{Data: result})
}

func (s *HTTPServer) listDatabaseChanges(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListDatabaseChanges(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.DatabaseChange]{Data: result})
}

func (s *HTTPServer) createDatabaseChange(w http.ResponseWriter, r *http.Request) {
	var req types.CreateDatabaseChangeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateDatabaseChange(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.DatabaseChangeDetail]{Data: result})
}

func (s *HTTPServer) getDatabaseChange(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetDatabaseChangeDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DatabaseChangeDetail]{Data: result})
}

func (s *HTTPServer) updateDatabaseChange(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateDatabaseChangeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateDatabaseChange(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DatabaseChangeDetail]{Data: result})
}

func (s *HTTPServer) listDatabaseChecks(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListDatabaseValidationChecks(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.DatabaseValidationCheck]{Data: result})
}

func (s *HTTPServer) createDatabaseCheck(w http.ResponseWriter, r *http.Request) {
	var req types.CreateDatabaseValidationCheckRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateDatabaseValidationCheck(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.DatabaseValidationCheckDetail]{Data: result})
}

func (s *HTTPServer) getDatabaseCheck(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetDatabaseValidationCheckDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DatabaseValidationCheckDetail]{Data: result})
}

func (s *HTTPServer) updateDatabaseCheck(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateDatabaseValidationCheckRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateDatabaseValidationCheck(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DatabaseValidationCheckDetail]{Data: result})
}

func (s *HTTPServer) executeDatabaseCheck(w http.ResponseWriter, r *http.Request) {
	var req types.ExecuteDatabaseValidationCheckRequest
	if err := decodeJSON(r, &req); err != nil && !strings.Contains(strings.ToLower(err.Error()), "eof") {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.ExecuteDatabaseValidationCheck(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.DatabaseValidationExecutionDetail]{Data: result})
}

func (s *HTTPServer) listDatabaseExecutions(w http.ResponseWriter, r *http.Request) {
	query := storage.DatabaseValidationExecutionQuery{
		ProjectID:         strings.TrimSpace(r.URL.Query().Get("project_id")),
		EnvironmentID:     strings.TrimSpace(r.URL.Query().Get("environment_id")),
		ServiceID:         strings.TrimSpace(r.URL.Query().Get("service_id")),
		ChangeSetID:       strings.TrimSpace(r.URL.Query().Get("change_set_id")),
		DatabaseChangeID:  strings.TrimSpace(r.URL.Query().Get("database_change_id")),
		ValidationCheckID: strings.TrimSpace(r.URL.Query().Get("validation_check_id")),
		ConnectionRefID:   strings.TrimSpace(r.URL.Query().Get("connection_ref_id")),
		Status:            strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:             500,
	}
	result, err := s.app.ListDatabaseValidationExecutions(r.Context(), query)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.DatabaseValidationExecution]{Data: result})
}

func (s *HTTPServer) getDatabaseExecution(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetDatabaseValidationExecutionDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.DatabaseValidationExecutionDetail]{Data: result})
}
