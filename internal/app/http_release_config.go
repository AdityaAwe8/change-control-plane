package app

import (
	"net/http"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *HTTPServer) listConfigSets(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListConfigSets(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.ConfigSet]{Data: result})
}

func (s *HTTPServer) createConfigSet(w http.ResponseWriter, r *http.Request) {
	var req types.CreateConfigSetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateConfigSet(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.ConfigSetDetail]{Data: result})
}

func (s *HTTPServer) getConfigSet(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetConfigSetDetail(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.ConfigSetDetail]{Data: result})
}

func (s *HTTPServer) updateConfigSet(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateConfigSetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateConfigSet(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.ConfigSetDetail]{Data: result})
}

func (s *HTTPServer) listReleases(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListReleases(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.Release]{Data: result})
}

func (s *HTTPServer) createRelease(w http.ResponseWriter, r *http.Request) {
	var req types.CreateReleaseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateRelease(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.ReleaseAnalysis]{Data: result})
}

func (s *HTTPServer) getRelease(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetReleaseAnalysis(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.ReleaseAnalysis]{Data: result})
}

func (s *HTTPServer) updateRelease(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateReleaseRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateRelease(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.ReleaseAnalysis]{Data: result})
}
