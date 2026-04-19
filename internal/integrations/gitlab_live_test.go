package integrations

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateGitLabWebhookToken(t *testing.T) {
	t.Parallel()

	if !ValidateGitLabWebhookToken("hook-secret", "hook-secret") {
		t.Fatal("expected matching webhook token to validate")
	}
	if ValidateGitLabWebhookToken("hook-secret", "wrong-secret") {
		t.Fatal("expected mismatched webhook token to fail validation")
	}
	if ValidateGitLabWebhookToken("", "hook-secret") {
		t.Fatal("expected empty configured secret to fail validation")
	}
}

func TestGitLabClientConnectionDiscoveryAndMergeRequestChanges(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "glpat_test" {
			t.Fatalf("expected private token header, got %q", r.Header.Get("PRIVATE-TOKEN"))
		}
		switch r.URL.Path {
		case "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"username": "acme-owner",
				"name":     "Acme Owner",
			})
		case "/groups/acme":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"full_path": "acme",
				"name":      "Acme",
			})
		case "/groups/acme/projects":
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id":                  42,
				"name":                "checkout",
				"path_with_namespace": "acme/checkout",
				"web_url":             "https://gitlab.example.com/acme/checkout",
				"default_branch":      "main",
				"archived":            false,
				"visibility":          "private",
				"namespace": map[string]any{
					"full_path": "acme",
					"path":      "acme",
					"name":      "Acme",
				},
			}})
		case "/projects/42/merge_requests/7/changes":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"changes": []map[string]any{
					{"old_path": "deploy/values.yaml", "new_path": "deploy/values.yaml"},
					{"new_path": "db/migrations/20260416.sql", "new_file": true},
				},
			})
		default:
			t.Fatalf("unexpected gitlab path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewGitLabClient(server.URL, "glpat_test")
	details, err := client.TestConnection(t.Context(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(details, "resolved gitlab principal acme-owner") {
		t.Fatalf("expected gitlab principal detail, got %+v", details)
	}
	if !containsString(details, "resolved gitlab scope acme") {
		t.Fatalf("expected gitlab scope detail, got %+v", details)
	}

	repositories, err := client.DiscoverRepositories(t.Context(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(repositories) != 1 {
		t.Fatalf("expected one discovered repository, got %d", len(repositories))
	}
	if repositories[0].Provider != "gitlab" || repositories[0].ExternalID != "42" || repositories[0].FullName != "acme/checkout" {
		t.Fatalf("expected gitlab repository normalization, got %+v", repositories[0])
	}

	files, err := client.MergeRequestChanges(t.Context(), "42", 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected two normalized merge-request files, got %+v", files)
	}
	if files[1].Status != "added" {
		t.Fatalf("expected new file to normalize as added, got %+v", files[1])
	}
}

func TestGitLabEnsureGroupWebhookRegistersAndUpdates(t *testing.T) {
	t.Parallel()

	var webhookCreates int
	var webhookUpdates int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "glpat_test" {
			t.Fatalf("expected private token header, got %q", r.Header.Get("PRIVATE-TOKEN"))
		}
		switch {
		case r.URL.Path == "/groups/acme/hooks" && r.Method == http.MethodGet:
			if webhookCreates == 0 {
				_ = json.NewEncoder(w).Encode([]map[string]any{})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id":  202,
				"url": "https://ccp.example.com/api/v1/integrations/gitlab/webhooks/gitlab",
			}})
		case r.URL.Path == "/groups/acme/hooks" && r.Method == http.MethodPost:
			webhookCreates++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 202})
		case r.URL.Path == "/groups/acme/hooks/202" && r.Method == http.MethodPut:
			webhookUpdates++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 202})
		default:
			t.Fatalf("unexpected gitlab request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewGitLabClient(server.URL, "glpat_test")
	registration, err := client.EnsureGroupWebhook(t.Context(), "acme", "https://ccp.example.com/api/v1/integrations/gitlab/webhooks/gitlab", "hook-secret")
	if err != nil {
		t.Fatal(err)
	}
	if registration.ExternalHookID != "202" || webhookCreates != 1 {
		t.Fatalf("expected webhook create evidence, got registration=%+v creates=%d", registration, webhookCreates)
	}

	registration, err = client.EnsureGroupWebhook(t.Context(), "acme", "https://ccp.example.com/api/v1/integrations/gitlab/webhooks/gitlab", "hook-secret")
	if err != nil {
		t.Fatal(err)
	}
	if registration.ExternalHookID != "202" || webhookUpdates != 1 {
		t.Fatalf("expected webhook update evidence, got registration=%+v updates=%d", registration, webhookUpdates)
	}
}

func TestGitLabClientLoadCODEOWNERS(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "glpat_test" {
			t.Fatalf("expected private token header, got %q", r.Header.Get("PRIVATE-TOKEN"))
		}
		switch r.URL.Path {
		case "/projects/42/repository/files/.github/CODEOWNERS":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":  "KiBAbWVAbWNlLmxvY2FsCi9hcHBzL2NoZWNrb3V0LyogQGFjbWUvcGxhdGZvcm0K",
				"encoding": "base64",
				"blob_id":  "blob-42",
			})
		default:
			t.Fatalf("unexpected gitlab request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewGitLabClient(server.URL, "glpat_test")
	result, err := client.LoadCODEOWNERS(t.Context(), SCMRepository{
		Provider:      "gitlab",
		ExternalID:    "42",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "imported" || result.FilePath != ".github/CODEOWNERS" || result.Revision != "blob-42" {
		t.Fatalf("expected imported codeowners result, got %+v", result)
	}
	if len(result.Rules) != 2 || len(result.Owners) != 2 {
		t.Fatalf("expected parsed rules and owners, got %+v", result)
	}
}

func TestGitLabClientLoadCODEOWNERSNotFound(t *testing.T) {
	t.Parallel()

	var requestedPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewGitLabClient(server.URL, "glpat_test")
	result, err := client.LoadCODEOWNERS(t.Context(), SCMRepository{
		Provider:      "gitlab",
		ExternalID:    "42",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "not_found" {
		t.Fatalf("expected not_found result, got %+v", result)
	}
	if len(requestedPaths) != 3 {
		t.Fatalf("expected all candidate paths to be checked, got %v", requestedPaths)
	}
}

func TestParseGitLabMergeRequestWebhookNormalizesChange(t *testing.T) {
	t.Parallel()

	payload := []byte(`{
		"object_kind":"merge_request",
		"project":{
			"id":42,
			"name":"checkout",
			"web_url":"https://gitlab.example.com/acme/checkout",
			"default_branch":"main",
			"path_with_namespace":"acme/checkout",
			"namespace":"acme"
		},
		"object_attributes":{
			"iid":7,
			"title":"CCP-301 tighten approvals",
			"description":"Tracks CCP-301 and rollout safety",
			"source_branch":"feature/ccp-301",
			"target_branch":"main",
			"action":"open",
			"state":"opened",
			"url":"https://gitlab.example.com/acme/checkout/-/merge_requests/7",
			"merge_status":"can_be_merged",
			"last_commit":{"id":"abc123"}
		},
		"labels":[{"title":"backend"},{"title":"rollout"}],
		"assignees":[{"username":"maintainer"}],
		"reviewers":[{"username":"reviewer-one"}],
		"user":{"username":"author"}
	}`)

	result, err := ParseGitLabWebhook("Merge Request Hook", "delivery-1", payload, func(projectID string, iid int) ([]SCMChangedFile, error) {
		if projectID != "42" || iid != 7 {
			t.Fatalf("expected merge request change fetch for 42/7, got %s/%d", projectID, iid)
		}
		return []SCMChangedFile{
			{Filename: "deploy/values.yaml", Status: "modified"},
			{Filename: "db/migrations/20260416.sql", Status: "added"},
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Operation != "gitlab.webhook.merge_request" {
		t.Fatalf("expected merge-request operation, got %+v", result)
	}
	if result.Change == nil {
		t.Fatal("expected normalized change payload")
	}
	if result.Change.Repository.Provider != "gitlab" || result.Change.Repository.FullName != "acme/checkout" {
		t.Fatalf("expected gitlab repository normalization, got %+v", result.Change.Repository)
	}
	if result.Change.ChangeType != "merge_request" || result.Change.FileCount != 2 {
		t.Fatalf("expected merge-request change metadata, got %+v", result.Change)
	}
	if !containsString(result.Change.IssueKeys, "CCP-301") {
		t.Fatalf("expected issue key extraction, got %+v", result.Change.IssueKeys)
	}
	if !containsString(result.Change.Reviewers, "maintainer") || !containsString(result.Change.Reviewers, "reviewer-one") {
		t.Fatalf("expected reviewers to include assignees and reviewers, got %+v", result.Change.Reviewers)
	}
	if len(result.Change.Files) != 2 || result.Change.Files[1].Status != "added" {
		t.Fatalf("expected fetched merge-request files, got %+v", result.Change.Files)
	}
	if strings.TrimSpace(stringValue(result.Change.Metadata["target_branch"])) != "main" {
		t.Fatalf("expected provider metadata to retain target branch, got %+v", result.Change.Metadata)
	}
}

func containsString(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}
