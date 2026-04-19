package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type webhook struct {
	ID                 int    `json:"id"`
	URL                string `json:"url"`
	Token              string `json:"token,omitempty"`
	PushEvents         bool   `json:"push_events"`
	MergeRequestEvents bool   `json:"merge_requests_events"`
	TagPushEvents      bool   `json:"tag_push_events"`
	ReleasesEvents     bool   `json:"releases_events"`
	SSLVerification    bool   `json:"enable_ssl_verification"`
}

type fixtureState struct {
	mu     sync.Mutex
	hooks  []webhook
	nextID int
}

func main() {
	port := valueOrDefault("PORT", "39480")
	token := valueOrDefault("REFERENCE_PILOT_GITLAB_TOKEN", "reference-pilot-token")
	group := valueOrDefault("REFERENCE_PILOT_GITLAB_GROUP", "acme")
	projectID := valueOrDefault("REFERENCE_PILOT_GITLAB_PROJECT_ID", "101")
	projectName := valueOrDefault("REFERENCE_PILOT_GITLAB_PROJECT_NAME", "checkout-service")
	projectPath := strings.Trim(strings.Join([]string{group, projectName}, "/"), "/")
	baseURL := strings.TrimRight(valueOrDefault("REFERENCE_PILOT_GITLAB_WEB_URL", "http://127.0.0.1:"+port), "/")

	state := &fixtureState{nextID: 1}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/v4/user", requireToken(token, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"id":       7,
			"username": "reference-bot",
			"name":     "Reference Pilot Bot",
		})
	}))
	mux.HandleFunc("GET /api/v4/groups/"+group, requireToken(token, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"id":        42,
			"name":      "Acme",
			"path":      group,
			"full_path": group,
		})
	}))
	mux.HandleFunc("GET /api/v4/groups/"+group+"/projects", requireToken(token, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, []map[string]any{{
			"id":                  mustAtoi(projectID),
			"name":                projectName,
			"path_with_namespace": projectPath,
			"web_url":             baseURL + "/" + projectPath,
			"default_branch":      "main",
			"archived":            false,
			"visibility":          "private",
			"namespace": map[string]any{
				"full_path": group,
				"path":      group,
				"name":      "Acme",
			},
		}})
	}))
	mux.HandleFunc("GET /api/v4/groups/"+group+"/hooks", requireToken(token, func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		writeJSON(w, http.StatusOK, state.hooks)
	}))
	mux.HandleFunc("POST /api/v4/groups/"+group+"/hooks", requireToken(token, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var request webhook
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		request.ID = state.nextID
		state.nextID++
		state.hooks = append(state.hooks, request)
		writeJSON(w, http.StatusCreated, request)
	}))
	mux.HandleFunc("PUT /api/v4/groups/"+group+"/hooks/", requireToken(token, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		id, ok := trailingInt(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		var request webhook
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		for idx, hook := range state.hooks {
			if hook.ID != id {
				continue
			}
			request.ID = id
			state.hooks[idx] = request
			writeJSON(w, http.StatusOK, request)
			return
		}
		http.NotFound(w, r)
	}))
	mux.HandleFunc("GET /api/v4/projects/"+projectID+"/merge_requests/7/changes", requireToken(token, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"changes": []map[string]any{
				{"old_path": "deploy/reference-pilot/k8s/reference-pilot.yaml", "new_path": "deploy/reference-pilot/k8s/reference-pilot.yaml", "new_file": false, "renamed_file": false, "deleted_file": false},
				{"old_path": "cmd/reference-pilot-workload/main.go", "new_path": "cmd/reference-pilot-workload/main.go", "new_file": false, "renamed_file": false, "deleted_file": false},
			},
		})
	}))

	log.Printf("reference pilot gitlab fixture listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func requireToken(expected string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.Header.Get("PRIVATE-TOKEN")) != strings.TrimSpace(expected) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func valueOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func trailingInt(path string) (int, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return 0, false
	}
	value, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, false
	}
	return value, true
}

func mustAtoi(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}
