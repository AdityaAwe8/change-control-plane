package app_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestGitHubIntegrationSyncAndWebhookIngestsMappedChangeSet(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_GITHUB_TOKEN_TEST", "ghs_test")
	t.Setenv("CCP_GITHUB_WEBHOOK_SECRET_TEST", "hook-secret")

	githubServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/acme":
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "acme"})
		case "/orgs/acme/repos":
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"name":           "checkout",
				"full_name":      "acme/checkout",
				"html_url":       "https://github.com/acme/checkout",
				"default_branch": "main",
				"owner":          map[string]any{"login": "acme"},
			}})
		case "/orgs/acme/hooks":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]map[string]any{})
				return
			}
			if r.Method == http.MethodPost {
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 101})
				return
			}
			t.Fatalf("unexpected github hooks method %s", r.Method)
		case "/repos/acme/checkout/contents/.github/CODEOWNERS":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":  "KiBAYWNtZS9wbGF0Zm9ybSBwbGF0Zm9ybUBhY21lLmxvY2FsCi9hcHBzL2NoZWNrb3V0LyogQGFjbWUvY2hlY2tvdXQK",
				"encoding": "base64",
				"sha":      "sha-checkout-codeowners",
			})
		case "/repos/acme/checkout/contents/CODEOWNERS", "/repos/acme/checkout/contents/docs/CODEOWNERS":
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("unexpected github path %s", r.URL.Path)
		}
	}))
	defer githubServer.Close()

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var githubIntegration types.Integration
	for _, integration := range integrations {
		if integration.Kind == "github" {
			githubIntegration = integration
			break
		}
	}
	if githubIntegration.ID == "" {
		t.Fatal("expected github integration")
	}

	mode := "advisory"
	enabled := true
	updated := patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID, types.UpdateIntegrationRequest{
		Mode:    &mode,
		Enabled: &enabled,
		Metadata: types.Metadata{
			"api_base_url":       githubServer.URL,
			"owner":              "acme",
			"access_token_env":   "CCP_GITHUB_TOKEN_TEST",
			"webhook_secret_env": "CCP_GITHUB_WEBHOOK_SECRET_TEST",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if !updated.Enabled {
		t.Fatal("expected github integration to be enabled")
	}

	_ = postItemAuth[types.IntegrationTestResult](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/test", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	syncResult := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if len(syncResult.Repositories) != 1 {
		t.Fatalf("expected one discovered repository, got %d", len(syncResult.Repositories))
	}

	repositories := getListAuth[types.Repository](t, server.URL+"/api/v1/repositories", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(repositories) != 1 {
		t.Fatalf("expected one repository from sync, got %d", len(repositories))
	}
	ownership, _ := repositories[0].Metadata["ownership"].(map[string]any)
	if ownership["status"] != "imported" {
		t.Fatalf("expected repository ownership import to persist, got %+v", repositories[0].Metadata)
	}
	repository := patchItemAuth[types.Repository](t, server.URL+"/api/v1/repositories/"+repositories[0].ID, types.UpdateRepositoryRequest{
		ServiceID:     &service.ID,
		EnvironmentID: &environment.ID,
		Status:        stringPtr("mapped"),
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if repository.ServiceID != service.ID || repository.EnvironmentID != environment.ID {
		t.Fatalf("expected repository mapping to persist, got %+v", repository)
	}
	inferredOwner, _ := repository.Metadata["inferred_owner"].(map[string]any)
	if inferredOwner["team_id"] != team.ID {
		t.Fatalf("expected repository mapping to infer owning team, got %+v", repository.Metadata)
	}

	payload := []byte(`{
		"ref":"refs/heads/main",
		"after":"abc123",
		"compare":"https://github.com/acme/checkout/compare/one...two",
		"repository":{
			"name":"checkout",
			"full_name":"acme/checkout",
			"html_url":"https://github.com/acme/checkout",
			"default_branch":"main",
			"owner":{"login":"acme"}
		},
		"head_commit":{
			"id":"abc123",
			"message":"CCP-101 ship stronger rollback posture",
			"added":["infra/deployment.yaml"],
			"removed":[],
			"modified":["db/migrations/20260416.sql","go.mod"]
		}
	}`)
	mac := hmac.New(sha256.New, []byte("hook-secret"))
	_, _ = mac.Write(payload)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/webhooks/github", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "delivery-1")
	req.Header.Set("X-Hub-Signature-256", signature)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 from webhook, got %d", resp.StatusCode)
	}

	changes := getListAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(changes) != 1 {
		t.Fatalf("expected one change set from mapped webhook ingest, got %d", len(changes))
	}
	if !strings.Contains(changes[0].Summary, "CCP-101") {
		t.Fatalf("expected webhook change summary to be preserved, got %+v", changes[0])
	}

	runs := getListAuth[types.IntegrationSyncRun](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/sync-runs", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(runs) < 3 {
		t.Fatalf("expected test, sync, and webhook runs, got %d", len(runs))
	}
}

func TestGitLabIntegrationSyncAndWebhookIngestsMappedChangeSet(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_GITLAB_TOKEN_TEST", "glpat_test")
	t.Setenv("CCP_GITLAB_WEBHOOK_SECRET_TEST", "gitlab-hook-secret")

	gitlabServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "glpat_test" {
			t.Fatalf("expected gitlab private token, got %q", r.Header.Get("PRIVATE-TOKEN"))
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
		case "/groups/acme/hooks":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]map[string]any{})
				return
			}
			if r.Method == http.MethodPost {
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 202})
				return
			}
			t.Fatalf("unexpected gitlab hooks method %s", r.Method)
		case "/projects/42/merge_requests/7/changes":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"changes": []map[string]any{
					{"old_path": "deploy/values.yaml", "new_path": "deploy/values.yaml"},
					{"new_path": "db/migrations/20260416.sql", "new_file": true},
				},
			})
		case "/projects/42/repository/files/.github/CODEOWNERS":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":  "KiBAYWNtZS9wbGF0Zm9ybQovYXBwcy9jaGVja291dC8qIEBhY21lL2NoZWNrb3V0Ci9kYi8qIGRiYUBhY21lLmxvY2FsCg==",
				"encoding": "base64",
				"blob_id":  "blob-checkout-codeowners",
			})
		case "/projects/42/repository/files/CODEOWNERS", "/projects/42/repository/files/docs/CODEOWNERS":
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("unexpected gitlab path %s", r.URL.Path)
		}
	}))
	defer gitlabServer.Close()

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-gitlab",
	})
	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	created := postItemAuth[types.Integration](t, server.URL+"/api/v1/integrations", types.CreateIntegrationRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Kind:           "gitlab",
		Name:           "GitLab Acme",
		InstanceKey:    "gitlab-acme",
		ScopeType:      "repository_group",
		ScopeName:      "Acme GitLab",
		AuthStrategy:   "personal_access_token",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if created.Kind != "gitlab" || created.InstanceKey != "gitlab-acme" {
		t.Fatalf("expected gitlab instance to be created, got %+v", created)
	}

	mode := "advisory"
	enabled := true
	configured := patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+created.ID, types.UpdateIntegrationRequest{
		Mode:    &mode,
		Enabled: &enabled,
		Metadata: types.Metadata{
			"api_base_url":       gitlabServer.URL,
			"group":              "acme",
			"access_token_env":   "CCP_GITLAB_TOKEN_TEST",
			"webhook_secret_env": "CCP_GITLAB_WEBHOOK_SECRET_TEST",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if configured.OnboardingStatus != "configured" {
		t.Fatalf("expected gitlab onboarding status to become configured, got %+v", configured)
	}

	testResult := postItemAuth[types.IntegrationTestResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/test", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if !containsDetail(testResult.Run.Details, "resolved gitlab principal acme-owner") {
		t.Fatalf("expected gitlab connection test details, got %+v", testResult.Run)
	}

	syncResult := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if len(syncResult.Repositories) != 1 {
		t.Fatalf("expected one discovered gitlab repository, got %+v", syncResult)
	}
	if syncResult.Repositories[0].Provider != "gitlab" || syncResult.Repositories[0].SourceIntegrationID != created.ID {
		t.Fatalf("expected repository to stay scoped to gitlab instance, got %+v", syncResult.Repositories[0])
	}

	filteredRepositories := getListAuth[types.Repository](t, server.URL+"/api/v1/repositories?provider=gitlab&source_integration_id="+created.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(filteredRepositories) != 1 {
		t.Fatalf("expected instance-scoped gitlab repository listing, got %+v", filteredRepositories)
	}
	ownership, _ := filteredRepositories[0].Metadata["ownership"].(map[string]any)
	if ownership["status"] != "imported" {
		t.Fatalf("expected gitlab repository ownership import to persist, got %+v", filteredRepositories[0].Metadata)
	}

	repository := patchItemAuth[types.Repository](t, server.URL+"/api/v1/repositories/"+filteredRepositories[0].ID, types.UpdateRepositoryRequest{
		ServiceID:     &service.ID,
		EnvironmentID: &environment.ID,
		Status:        stringPtr("mapped"),
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if repository.ServiceID != service.ID || repository.EnvironmentID != environment.ID {
		t.Fatalf("expected gitlab repository mapping to persist, got %+v", repository)
	}
	inferredOwner, _ := repository.Metadata["inferred_owner"].(map[string]any)
	if inferredOwner["team_id"] != team.ID {
		t.Fatalf("expected gitlab repository mapping to infer owning team, got %+v", repository.Metadata)
	}

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
		"labels":[{"title":"backend"}],
		"assignees":[{"username":"maintainer"}],
		"reviewers":[{"username":"reviewer-one"}],
		"user":{"username":"author"}
	}`)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/integrations/"+created.ID+"/webhooks/gitlab", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gitlab-Event", "Merge Request Hook")
	req.Header.Set("X-Gitlab-Event-UUID", "delivery-1")
	req.Header.Set("X-Gitlab-Token", "gitlab-hook-secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 from gitlab webhook, got %d", resp.StatusCode)
	}

	changes := getListAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(changes) != 1 {
		t.Fatalf("expected one gitlab change set from mapped webhook ingest, got %d", len(changes))
	}
	if !strings.Contains(changes[0].Summary, "CCP-301") {
		t.Fatalf("expected gitlab webhook summary to be preserved, got %+v", changes[0])
	}
	if stringMetadata(changes[0].Metadata, "scm_provider") != "gitlab" {
		t.Fatalf("expected gitlab provider metadata to be preserved, got %+v", changes[0].Metadata)
	}

	filteredIntegrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations?kind=gitlab&instance_key=gitlab-acme", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(filteredIntegrations) != 1 || filteredIntegrations[0].ID != created.ID {
		t.Fatalf("expected instance-scoped gitlab filter result, got %+v", filteredIntegrations)
	}

	runs := getListAuth[types.IntegrationSyncRun](t, server.URL+"/api/v1/integrations/"+created.ID+"/sync-runs", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(runs) < 3 {
		t.Fatalf("expected test, sync, and webhook runs for gitlab, got %d", len(runs))
	}
}

func TestGitHubAppOnboardingSupportsMultiInstanceSync(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_GITHUB_APP_PRIVATE_KEY_TEST", marshalRSAPrivateKeyPEM(t))
	t.Setenv("CCP_GITHUB_WEBHOOK_SECRET_TEST", "hook-secret")

	githubServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/987654/access_tokens":
			if r.Method != http.MethodPost {
				t.Fatalf("expected post for installation token, got %s", r.Method)
			}
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				t.Fatalf("expected bearer app jwt, got %q", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "ghs_installation_token",
				"expires_at": "2026-04-16T21:00:00Z",
			})
		case "/orgs/acme":
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "acme"})
		case "/orgs/acme/repos":
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"name":           "control-plane",
				"full_name":      "acme/control-plane",
				"html_url":       "https://github.com/acme/control-plane",
				"default_branch": "main",
				"private":        true,
				"archived":       false,
				"owner":          map[string]any{"login": "acme"},
			}})
		case "/orgs/acme/hooks":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]map[string]any{})
				return
			}
			if r.Method == http.MethodPost {
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 303})
				return
			}
			t.Fatalf("unexpected github app hooks method %s", r.Method)
		case "/repos/acme/control-plane/contents/.github/CODEOWNERS", "/repos/acme/control-plane/contents/CODEOWNERS", "/repos/acme/control-plane/contents/docs/CODEOWNERS":
			http.Error(w, "not found", http.StatusNotFound)
		default:
			t.Fatalf("unexpected github app path %s", r.URL.Path)
		}
	}))
	defer githubServer.Close()

	cfg := common.LoadConfig()
	cfg.APIBaseURL = "http://control-plane.local"
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-github-app",
	})

	created := postItemAuth[types.Integration](t, server.URL+"/api/v1/integrations", types.CreateIntegrationRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Kind:           "github",
		Name:           "GitHub App Acme",
		InstanceKey:    "github-app-acme",
		ScopeType:      "organization",
		ScopeName:      "Acme GitHub",
		AuthStrategy:   "github_app",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if created.Kind != "github" || created.InstanceKey != "github-app-acme" {
		t.Fatalf("expected github app instance to be created, got %+v", created)
	}

	enabled := patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+created.ID, types.UpdateIntegrationRequest{
		Enabled:      boolPtr(false),
		AuthStrategy: stringPtr("github_app"),
		Metadata: types.Metadata{
			"api_base_url":       githubServer.URL,
			"web_base_url":       githubServer.URL,
			"owner":              "acme",
			"app_id":             "123456",
			"app_slug":           "change-control-plane",
			"private_key_env":    "CCP_GITHUB_APP_PRIVATE_KEY_TEST",
			"webhook_secret_env": "CCP_GITHUB_WEBHOOK_SECRET_TEST",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if enabled.AuthStrategy != "github_app" {
		t.Fatalf("expected github_app auth strategy, got %+v", enabled)
	}

	start := postItemAuth[types.GitHubOnboardingStartResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/github/onboarding/start", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if !strings.Contains(start.AuthorizeURL, "/apps/change-control-plane/installations/new") {
		t.Fatalf("expected github app install url, got %+v", start)
	}
	startURL, err := url.Parse(start.AuthorizeURL)
	if err != nil {
		t.Fatal(err)
	}
	callbackResp, err := http.Get(server.URL + "/api/v1/integrations/github/callback?state=" + url.QueryEscape(startURL.Query().Get("state")) + "&installation_id=987654&setup_action=install")
	if err != nil {
		t.Fatal(err)
	}
	defer callbackResp.Body.Close()
	if callbackResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from callback, got %d", callbackResp.StatusCode)
	}

	testResult := postItemAuth[types.IntegrationTestResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/test", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if !containsDetail(testResult.Run.Details, "resolved github principal acme") {
		t.Fatalf("expected github app connection test details, got %+v", testResult.Run)
	}
	syncResult := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if len(syncResult.Repositories) != 1 {
		t.Fatalf("expected one discovered github repository, got %+v", syncResult)
	}
	if syncResult.Repositories[0].SourceIntegrationID != created.ID {
		t.Fatalf("expected repository to stay scoped to github app instance, got %+v", syncResult.Repositories[0])
	}

	filtered := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations?kind=github&instance_key=github-app-acme", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(filtered) != 1 || filtered[0].ID != created.ID || filtered[0].OnboardingStatus != "installed" {
		t.Fatalf("expected instance-scoped github filter result, got %+v", filtered)
	}
}

func TestKubernetesAndPrometheusIntegrationRunsExposePilotEvidence(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_KUBE_TOKEN_TEST", "kube-token")
	t.Setenv("CCP_PROM_TOKEN_TEST", "prom-token")

	kubeServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apis/apps/v1/namespaces/prod/deployments/checkout":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": false},
				"status": map[string]any{
					"replicas":            4,
					"updatedReplicas":     4,
					"availableReplicas":   4,
					"unavailableReplicas": 0,
					"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
				},
			})
		case "/apis/apps/v1/namespaces/prod/deployments":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
					"spec":     map[string]any{"paused": false},
					"status": map[string]any{
						"replicas":            4,
						"updatedReplicas":     4,
						"availableReplicas":   4,
						"unavailableReplicas": 0,
						"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
					},
				}},
			})
		default:
			t.Fatalf("unexpected kubernetes path %s", r.URL.Path)
		}
	}))
	defer kubeServer.Close()

	promServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query_range" {
			t.Fatalf("unexpected prometheus path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result": []map[string]any{{
					"values": [][]any{
						{float64(1), "0.1"},
						{float64(2), "0.4"},
					},
				}},
			},
		})
	}))
	defer promServer.Close()

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-pilot",
	})

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var kubeIntegration types.Integration
	var promIntegration types.Integration
	for _, integration := range integrations {
		switch integration.Kind {
		case "kubernetes":
			kubeIntegration = integration
		case "prometheus":
			promIntegration = integration
		}
	}
	if kubeIntegration.ID == "" || promIntegration.ID == "" {
		t.Fatalf("expected kubernetes and prometheus integrations, got %+v", integrations)
	}

	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":    kubeServer.URL,
			"namespace":       "prod",
			"deployment_name": "checkout",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+promIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":     promServer.URL,
			"query_path":       "/api/v1/query_range",
			"window_seconds":   "300",
			"step_seconds":     "60",
			"bearer_token_env": "CCP_PROM_TOKEN_TEST",
			"queries": []map[string]any{
				{"name": "error_rate", "category": "technical", "query": "error_rate", "threshold": 1, "comparator": ">", "unit": "%"},
			},
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	kubeTest := postItemAuth[types.IntegrationTestResult](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID+"/test", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if !containsDetail(kubeTest.Run.Details, "backend_status=awaiting_verification") {
		t.Fatalf("expected normalized kubernetes status details, got %+v", kubeTest.Run)
	}
	kubeSync := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if !containsDetail(kubeSync.Run.Details, "available_replicas=4") {
		t.Fatalf("expected kubernetes replica evidence, got %+v", kubeSync.Run)
	}
	if len(kubeSync.DiscoveredResources) != 1 || kubeSync.DiscoveredResources[0].ResourceType != "kubernetes_workload" {
		t.Fatalf("expected discovered kubernetes workload evidence, got %+v", kubeSync.DiscoveredResources)
	}

	promTest := postItemAuth[types.IntegrationTestResult](t, server.URL+"/api/v1/integrations/"+promIntegration.ID+"/test", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if !containsDetail(promTest.Run.Details, "signal_count=1") {
		t.Fatalf("expected prometheus signal-count detail, got %+v", promTest.Run)
	}
	promSync := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+promIntegration.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if !strings.Contains(promSync.Run.Summary, "healthy across 1 signal") {
		t.Fatalf("expected prometheus summary to describe collected evidence, got %+v", promSync.Run)
	}
	if !containsDetail(promSync.Run.Details, "signal.error_rate=") {
		t.Fatalf("expected prometheus signal detail, got %+v", promSync.Run)
	}
}

func TestAdvisoryKubernetesIntegrationDoesNotExecuteControlActions(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	var getCalls atomic.Int32
	var mutatingCalls atomic.Int32
	kubeServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCalls.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": false},
				"status": map[string]any{
					"replicas":            3,
					"updatedReplicas":     3,
					"availableReplicas":   3,
					"unavailableReplicas": 0,
					"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
				},
			})
		default:
			mutatingCalls.Add(1)
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer kubeServer.Close()

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-advisory",
	})
	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "high",
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "Ship live advisory release",
		ChangeTypes:    []string{"code"},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	plan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var kubeIntegration types.Integration
	for _, integration := range integrations {
		if integration.Kind == "kubernetes" {
			kubeIntegration = integration
			break
		}
	}
	if kubeIntegration.ID == "" {
		t.Fatal("expected kubernetes integration")
	}
	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":    kubeServer.URL,
			"namespace":       "prod",
			"deployment_name": "checkout",
			"status_path":     "/apis/apps/v1/namespaces/prod/deployments/checkout",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID:        plan.Plan.ID,
		BackendType:          "kubernetes",
		BackendIntegrationID: kubeIntegration.ID,
		SignalProviderType:   "simulated",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approved",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start advisory rollout",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	pausePayload, err := json.Marshal(map[string]any{"reason": "operator pause"})
	if err != nil {
		t.Fatal(err)
	}
	pauseReq, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/pause", bytes.NewReader(pausePayload))
	if err != nil {
		t.Fatal(err)
	}
	pauseReq.Header.Set("Content-Type", "application/json")
	pauseReq.Header.Set("Authorization", "Bearer "+admin.Token)
	pauseReq.Header.Set("X-CCP-Organization-ID", admin.Session.ActiveOrganizationID)
	pauseResp, err := http.DefaultClient.Do(pauseReq)
	if err != nil {
		t.Fatal(err)
	}
	defer pauseResp.Body.Close()
	if pauseResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for advisory manual pause, got %d", pauseResp.StatusCode)
	}
	var pauseError types.ErrorResponse
	if err := json.NewDecoder(pauseResp.Body).Decode(&pauseError); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(pauseError.Error.Message), "advisory mode blocks manual pause") {
		t.Fatalf("expected advisory pause error, got %+v", pauseError)
	}

	detail := postItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/reconcile", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if mutatingCalls.Load() != 0 {
		t.Fatalf("expected advisory mode to avoid mutating kubernetes calls, got %d", mutatingCalls.Load())
	}
	if getCalls.Load() == 0 {
		t.Fatal("expected advisory mode to observe kubernetes status")
	}
	if detail.Execution.Status != "in_progress" {
		t.Fatalf("expected execution to remain in progress in advisory mode, got %s", detail.Execution.Status)
	}
	if !detail.RuntimeSummary.AdvisoryOnly || detail.RuntimeSummary.ControlMode != "advisory" {
		t.Fatalf("expected advisory runtime summary, got %+v", detail.RuntimeSummary)
	}
	if len(detail.VerificationResults) > 0 {
		latest := detail.VerificationResults[len(detail.VerificationResults)-1]
		if !strings.HasPrefix(latest.Decision, "advisory_") {
			t.Fatalf("expected advisory verification decision, got %+v", latest)
		}
		if latest.ActionState != "recommended" || latest.ControlMode != "advisory" {
			t.Fatalf("expected advisory verification fields, got %+v", latest)
		}
	}
}

func TestDiscoveredResourcesCanBeMappedAndSummarized(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	kubeServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apis/apps/v1/namespaces/prod/deployments/checkout":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": false},
				"status": map[string]any{
					"replicas":            2,
					"updatedReplicas":     2,
					"availableReplicas":   2,
					"unavailableReplicas": 0,
					"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
				},
			})
		case "/apis/apps/v1/namespaces/prod/deployments":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
					"spec":     map[string]any{"paused": false},
					"status": map[string]any{
						"replicas":            2,
						"updatedReplicas":     2,
						"availableReplicas":   2,
						"unavailableReplicas": 0,
						"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
					},
				}},
			})
		default:
			t.Fatalf("unexpected kubernetes path %s", r.URL.Path)
		}
	}))
	defer kubeServer.Close()

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-discovery",
	})
	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Payments",
		Slug:           "payments",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "production",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var kubeIntegration types.Integration
	for _, integration := range integrations {
		if integration.Kind == "kubernetes" {
			kubeIntegration = integration
			break
		}
	}
	if kubeIntegration.ID == "" {
		t.Fatal("expected kubernetes integration")
	}

	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":    kubeServer.URL,
			"namespace":       "prod",
			"deployment_name": "checkout",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	syncResult := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if len(syncResult.DiscoveredResources) == 0 {
		t.Fatal("expected discovered kubernetes resource from sync")
	}

	discovered := getListAuth[types.DiscoveredResource](t, server.URL+"/api/v1/discovered-resources?integration_id="+kubeIntegration.ID+"&unmapped_only=true", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(discovered) == 0 {
		t.Fatal("expected unmapped discovered resource")
	}
	if discovered[0].ServiceID != "" || discovered[0].EnvironmentID != "" {
		t.Fatalf("expected unmapped resource before manual mapping, got %+v", discovered[0])
	}

	coverageBefore := getItemAuth[types.CoverageSummary](t, server.URL+"/api/v1/integrations/coverage", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if coverageBefore.UnmappedDiscoveredResources == 0 {
		t.Fatalf("expected unmapped discovered resource coverage gap, got %+v", coverageBefore)
	}

	mapped := patchItemAuth[types.DiscoveredResource](t, server.URL+"/api/v1/discovered-resources/"+discovered[0].ID, types.UpdateDiscoveredResourceRequest{
		ServiceID:     &service.ID,
		EnvironmentID: &environment.ID,
		Status:        stringPtr("mapped"),
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if mapped.ServiceID != service.ID || mapped.EnvironmentID != environment.ID || mapped.Status != "mapped" {
		t.Fatalf("expected discovered resource mapping to persist, got %+v", mapped)
	}

	coverageAfter := getItemAuth[types.CoverageSummary](t, server.URL+"/api/v1/integrations/coverage", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if coverageAfter.WorkloadCoverageEnvironments == 0 || coverageAfter.UnmappedDiscoveredResources != 0 {
		t.Fatalf("expected coverage summary to reflect mapping, got %+v", coverageAfter)
	}
}

func TestKubernetesSyncMarksMissingWorkloadsAfterInventoryDisappears(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	var stage atomic.Int32
	kubeServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apis/apps/v1/namespaces/prod/deployments/gateway":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "gateway", "namespace": "prod"},
				"spec":     map[string]any{"paused": false},
				"status": map[string]any{
					"replicas":            2,
					"updatedReplicas":     2,
					"availableReplicas":   2,
					"unavailableReplicas": 0,
					"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
				},
			})
		case "/apis/apps/v1/namespaces/prod/deployments":
			items := []map[string]any{}
			if stage.Load() == 0 {
				items = append(items, map[string]any{
					"metadata": map[string]any{"name": "payments", "namespace": "prod"},
					"spec":     map[string]any{"paused": false},
					"status": map[string]any{
						"replicas":            3,
						"updatedReplicas":     3,
						"availableReplicas":   3,
						"unavailableReplicas": 0,
						"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
					},
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
		default:
			t.Fatalf("unexpected kubernetes path %s", r.URL.Path)
		}
	}))
	defer kubeServer.Close()

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-kube-missing",
	})
	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Payments",
		Slug:           "payments",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var kubeIntegration types.Integration
	for _, integration := range integrations {
		if integration.Kind == "kubernetes" {
			kubeIntegration = integration
			break
		}
	}
	if kubeIntegration.ID == "" {
		t.Fatal("expected kubernetes integration")
	}

	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":    kubeServer.URL,
			"namespace":       "prod",
			"deployment_name": "gateway",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	firstSync := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if len(firstSync.DiscoveredResources) != 1 {
		t.Fatalf("expected first sync to discover one workload, got %+v", firstSync.DiscoveredResources)
	}
	mapped := patchItemAuth[types.DiscoveredResource](t, server.URL+"/api/v1/discovered-resources/"+firstSync.DiscoveredResources[0].ID, types.UpdateDiscoveredResourceRequest{
		ServiceID:     &service.ID,
		EnvironmentID: &environment.ID,
		Status:        stringPtr("mapped"),
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if mapped.Status != "mapped" {
		t.Fatalf("expected mapped workload after explicit mapping, got %+v", mapped)
	}
	coverageBefore := getItemAuth[types.CoverageSummary](t, server.URL+"/api/v1/integrations/coverage", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if coverageBefore.WorkloadCoverageEnvironments != 1 {
		t.Fatalf("expected workload coverage before disappearance, got %+v", coverageBefore)
	}

	stage.Store(1)
	secondSync := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if !containsDetail(secondSync.Run.Details, "missing_workloads=1") {
		t.Fatalf("expected missing workload detail after second sync, got %+v", secondSync.Run)
	}

	discovered := getListAuth[types.DiscoveredResource](t, server.URL+"/api/v1/discovered-resources?integration_id="+kubeIntegration.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(discovered) != 1 || discovered[0].Status != "missing" {
		t.Fatalf("expected workload to be marked missing after disappearing from inventory, got %+v", discovered)
	}
	coverageAfter := getItemAuth[types.CoverageSummary](t, server.URL+"/api/v1/integrations/coverage", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if coverageAfter.WorkloadCoverageEnvironments != 0 {
		t.Fatalf("expected missing workload to stop counting as coverage, got %+v", coverageAfter)
	}
}

func TestPrometheusSyncSurfacesWarningWhenNoSamplesAreReturned(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	promServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query_range" {
			t.Fatalf("unexpected prometheus path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result":     []map[string]any{},
			},
		})
	}))
	defer promServer.Close()

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-prom-warning",
	})
	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Payments",
		Slug:           "payments",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var promIntegration types.Integration
	for _, integration := range integrations {
		if integration.Kind == "prometheus" {
			promIntegration = integration
			break
		}
	}
	if promIntegration.ID == "" {
		t.Fatal("expected prometheus integration")
	}

	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+promIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":   promServer.URL,
			"window_seconds": "120",
			"step_seconds":   "15",
			"queries": []map[string]any{
				{"name": "error_rate", "category": "technical", "query": "error_rate", "threshold": 1, "comparator": ">", "unit": "%", "service_id": service.ID},
			},
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	result := postItemAuth[types.IntegrationSyncResult](t, server.URL+"/api/v1/integrations/"+promIntegration.ID+"/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if !strings.Contains(result.Run.Summary, "warning") {
		t.Fatalf("expected warning summary when prometheus returns no samples, got %+v", result.Run)
	}
	if !containsDetail(result.Run.Details, "returned no samples") {
		t.Fatalf("expected explicit no-samples detail, got %+v", result.Run)
	}
	if len(result.DiscoveredResources) != 1 || result.DiscoveredResources[0].Health != "warning" {
		t.Fatalf("expected discovered signal target warning state, got %+v", result.DiscoveredResources)
	}
}

func TestGitLabWebhookRegistrationSyncRepairsExistingHostedWebhook(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_GITLAB_TOKEN_TEST", "glpat_test")
	t.Setenv("CCP_GITLAB_WEBHOOK_SECRET_TEST", "gitlab-hook-secret")

	var hookID int
	var callbackURL string
	gitlabServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("PRIVATE-TOKEN") != "glpat_test" {
			t.Fatalf("expected gitlab private token header, got %q", r.Header.Get("PRIVATE-TOKEN"))
		}
		switch r.URL.Path {
		case "/groups/acme/hooks":
			switch r.Method {
			case http.MethodGet:
				if hookID == 0 {
					_ = json.NewEncoder(w).Encode([]map[string]any{})
					return
				}
				_ = json.NewEncoder(w).Encode([]map[string]any{{
					"id":  hookID,
					"url": callbackURL,
				}})
				return
			case http.MethodPost:
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode gitlab webhook create payload: %v", err)
				}
				if payload["token"] != "gitlab-hook-secret" {
					t.Fatalf("expected webhook secret in create payload, got %+v", payload)
				}
				if payload["push_events"] != true || payload["merge_requests_events"] != true || payload["releases_events"] != true {
					t.Fatalf("expected gitlab webhook events in create payload, got %+v", payload)
				}
				callbackURL = strings.TrimSpace(payload["url"].(string))
				if !strings.Contains(callbackURL, "/api/v1/integrations/") || !strings.Contains(callbackURL, "/webhooks/gitlab") {
					t.Fatalf("expected callback url in create payload, got %q", callbackURL)
				}
				hookID = 202
				_ = json.NewEncoder(w).Encode(map[string]any{"id": hookID})
				return
			default:
				t.Fatalf("unexpected gitlab hooks method %s", r.Method)
			}
		case "/groups/acme/hooks/202":
			if r.Method != http.MethodPut {
				t.Fatalf("expected gitlab webhook update with PUT, got %s", r.Method)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode gitlab webhook update payload: %v", err)
			}
			callbackURL = strings.TrimSpace(payload["url"].(string))
			if !strings.Contains(callbackURL, "/api/v1/integrations/") || !strings.Contains(callbackURL, "/webhooks/gitlab") {
				t.Fatalf("expected callback url in update payload, got %q", callbackURL)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": hookID})
			return
		default:
			t.Fatalf("unexpected gitlab path %s", r.URL.Path)
		}
	}))
	defer gitlabServer.Close()

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-gitlab-hooks",
	})

	created := postItemAuth[types.Integration](t, server.URL+"/api/v1/integrations", types.CreateIntegrationRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Kind:           "gitlab",
		Name:           "GitLab Hosted Acme",
		InstanceKey:    "gitlab-hosted-acme",
		ScopeType:      "repository_group",
		ScopeName:      "Acme GitLab",
		AuthStrategy:   "personal_access_token",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+created.ID, types.UpdateIntegrationRequest{
		Mode:    stringPtr("advisory"),
		Enabled: boolPtr(true),
		Metadata: types.Metadata{
			"api_base_url":       gitlabServer.URL,
			"group":              "acme",
			"access_token_env":   "CCP_GITLAB_TOKEN_TEST",
			"webhook_secret_env": "CCP_GITLAB_WEBHOOK_SECRET_TEST",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	preexisting := getItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/webhook-registration", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if preexisting.Registration.ExternalHookID != "202" || preexisting.Registration.ProviderKind != "gitlab" {
		t.Fatalf("expected integration update to auto-register gitlab hosted hook, got %+v", preexisting)
	}
	if preexisting.Registration.CallbackURL != callbackURL {
		t.Fatalf("expected stored gitlab callback to match hosted registration, got %+v", preexisting)
	}

	first := postItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/webhook-registration/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if first.Registration.ExternalHookID != "202" || first.Registration.ProviderKind != "gitlab" {
		t.Fatalf("expected first gitlab webhook registration to persist hosted hook details, got %+v", first)
	}
	if !containsDetail(first.Details, "existing gitlab group webhook updated") {
		t.Fatalf("expected first explicit gitlab webhook sync to repair existing hosted hook, got %+v", first)
	}

	second := postItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/webhook-registration/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if second.Registration.ExternalHookID != "202" {
		t.Fatalf("expected second gitlab webhook registration to keep hook id, got %+v", second)
	}
	if !containsDetail(second.Details, "existing gitlab group webhook updated") {
		t.Fatalf("expected second gitlab webhook registration to repair existing hook, got %+v", second)
	}

	fetched := getItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/webhook-registration", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if fetched.Registration.CallbackURL != callbackURL || fetched.Registration.ExternalHookID != "202" {
		t.Fatalf("expected stored gitlab webhook registration to match hosted callback, got %+v", fetched)
	}
}

func TestGitHubAppWebhookRegistrationSyncRepairsExistingHostedWebhook(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_GITHUB_APP_PRIVATE_KEY_TEST", marshalRSAPrivateKeyPEM(t))
	t.Setenv("CCP_GITHUB_WEBHOOK_SECRET_TEST", "hook-secret")

	var hookID int
	var callbackURL string
	githubServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/app/installations/987654/access_tokens":
			if r.Method != http.MethodPost {
				t.Fatalf("expected post for installation token, got %s", r.Method)
			}
			if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				t.Fatalf("expected bearer app jwt, got %q", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "ghs_installation_token",
				"expires_at": "2026-04-16T21:00:00Z",
			})
		case "/orgs/acme/hooks":
			if got := r.Header.Get("Authorization"); got != "Bearer ghs_installation_token" {
				t.Fatalf("expected installation token header for github hooks, got %q", got)
			}
			switch r.Method {
			case http.MethodGet:
				if hookID == 0 {
					_ = json.NewEncoder(w).Encode([]map[string]any{})
					return
				}
				_ = json.NewEncoder(w).Encode([]map[string]any{{
					"id":     hookID,
					"active": true,
					"config": map[string]any{"url": callbackURL},
				}})
				return
			case http.MethodPost:
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatalf("decode github webhook create payload: %v", err)
				}
				config := payload["config"].(map[string]any)
				if config["secret"] != "hook-secret" || config["content_type"] != "json" {
					t.Fatalf("expected webhook secret and content type in github create payload, got %+v", payload)
				}
				callbackURL = strings.TrimSpace(config["url"].(string))
				if !strings.Contains(callbackURL, "/api/v1/integrations/") || !strings.Contains(callbackURL, "/webhooks/github") {
					t.Fatalf("expected callback url in github create payload, got %q", callbackURL)
				}
				hookID = 303
				_ = json.NewEncoder(w).Encode(map[string]any{"id": hookID})
				return
			default:
				t.Fatalf("unexpected github hooks method %s", r.Method)
			}
		case "/orgs/acme/hooks/303":
			if got := r.Header.Get("Authorization"); got != "Bearer ghs_installation_token" {
				t.Fatalf("expected installation token header for github hook update, got %q", got)
			}
			if r.Method != http.MethodPatch {
				t.Fatalf("expected github webhook update with PATCH, got %s", r.Method)
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode github webhook update payload: %v", err)
			}
			config := payload["config"].(map[string]any)
			callbackURL = strings.TrimSpace(config["url"].(string))
			if !strings.Contains(callbackURL, "/api/v1/integrations/") || !strings.Contains(callbackURL, "/webhooks/github") {
				t.Fatalf("expected callback url in github update payload, got %q", callbackURL)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": hookID})
		default:
			t.Fatalf("unexpected github path %s", r.URL.Path)
		}
	}))
	defer githubServer.Close()

	cfg := common.LoadConfig()
	cfg.APIBaseURL = "http://control-plane.local"
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-github-hooks",
	})

	created := postItemAuth[types.Integration](t, server.URL+"/api/v1/integrations", types.CreateIntegrationRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Kind:           "github",
		Name:           "GitHub App Acme",
		InstanceKey:    "github-app-hosted-acme",
		ScopeType:      "organization",
		ScopeName:      "Acme GitHub",
		AuthStrategy:   "github_app",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+created.ID, types.UpdateIntegrationRequest{
		Enabled:      boolPtr(false),
		AuthStrategy: stringPtr("github_app"),
		Metadata: types.Metadata{
			"api_base_url":       githubServer.URL,
			"web_base_url":       githubServer.URL,
			"owner":              "acme",
			"app_id":             "123456",
			"app_slug":           "change-control-plane",
			"private_key_env":    "CCP_GITHUB_APP_PRIVATE_KEY_TEST",
			"webhook_secret_env": "CCP_GITHUB_WEBHOOK_SECRET_TEST",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	start := postItemAuth[types.GitHubOnboardingStartResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/github/onboarding/start", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	startURL, err := url.Parse(start.AuthorizeURL)
	if err != nil {
		t.Fatal(err)
	}
	callbackResp, err := http.Get(server.URL + "/api/v1/integrations/github/callback?state=" + url.QueryEscape(startURL.Query().Get("state")) + "&installation_id=987654&setup_action=install")
	if err != nil {
		t.Fatal(err)
	}
	defer callbackResp.Body.Close()
	if callbackResp.StatusCode != http.StatusOK {
		t.Fatalf("expected github onboarding callback to succeed, got %d", callbackResp.StatusCode)
	}

	preexisting := getItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/webhook-registration", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if preexisting.Registration.ProviderKind != "github" || preexisting.Registration.Status != "disabled" {
		t.Fatalf("expected github onboarding callback to preserve disabled placeholder registration until explicit sync, got %+v", preexisting)
	}
	if preexisting.Registration.ExternalHookID != "" {
		t.Fatalf("expected github placeholder registration to remain unbound before sync, got %+v", preexisting)
	}
	if !containsDetail(preexisting.Details, "github app_id and installation_id are required") {
		t.Fatalf("expected github placeholder registration to retain pre-onboarding error detail, got %+v", preexisting)
	}

	first := postItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/webhook-registration/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if first.Registration.ExternalHookID != "303" || first.Registration.ProviderKind != "github" {
		t.Fatalf("expected first github webhook registration to persist hosted hook details, got %+v", first)
	}
	if first.Registration.CallbackURL != callbackURL {
		t.Fatalf("expected first github webhook registration to persist hosted callback url, got %+v", first)
	}
	if !containsDetail(first.Details, "github organization webhook registered automatically") {
		t.Fatalf("expected first explicit github webhook sync to create hosted hook after onboarding, got %+v", first)
	}

	second := postItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/webhook-registration/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if second.Registration.ExternalHookID != "303" {
		t.Fatalf("expected second github webhook registration to keep hook id, got %+v", second)
	}
	if !containsDetail(second.Details, "existing github organization webhook updated") {
		t.Fatalf("expected second github webhook registration to repair existing hook, got %+v", second)
	}

	fetched := getItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+created.ID+"/webhook-registration", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if fetched.Registration.CallbackURL != callbackURL || fetched.Registration.ExternalHookID != "303" {
		t.Fatalf("expected stored github webhook registration to match hosted callback, got %+v", fetched)
	}
}

func TestKubernetesAndPrometheusIntegrationRoutesHonorConfiguredAuthHeadersAndPaths(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_KUBE_TOKEN_TEST", "kube-secret")
	t.Setenv("CCP_PROM_TOKEN_TEST", "prom-secret")

	var kubeCalls atomic.Int32
	kubeServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kubeCalls.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer kube-secret" {
			t.Fatalf("expected kubernetes bearer token header, got %q", got)
		}
		if r.URL.Path != "/custom/status/checkout" {
			t.Fatalf("expected custom kubernetes status path, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
			"spec":     map[string]any{"paused": false},
			"status": map[string]any{
				"replicas":            2,
				"updatedReplicas":     2,
				"availableReplicas":   2,
				"unavailableReplicas": 0,
				"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
			},
		})
	}))
	defer kubeServer.Close()

	var promCalls atomic.Int32
	promServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		promCalls.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer prom-secret" {
			t.Fatalf("expected prometheus bearer token header, got %q", got)
		}
		if r.URL.Path != "/tenant/query_range" {
			t.Fatalf("expected custom prometheus query path, got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result": []map[string]any{{
					"values": [][]any{
						{float64(1), "0.2"},
						{float64(2), "0.4"},
					},
				}},
			},
		})
	}))
	defer promServer.Close()

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-auth-paths",
	})

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var kubeIntegration types.Integration
	var promIntegration types.Integration
	for _, integration := range integrations {
		switch integration.Kind {
		case "kubernetes":
			kubeIntegration = integration
		case "prometheus":
			promIntegration = integration
		}
	}
	if kubeIntegration.ID == "" || promIntegration.ID == "" {
		t.Fatalf("expected kubernetes and prometheus integrations, got %+v", integrations)
	}

	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":     kubeServer.URL,
			"status_path":      "/custom/status/checkout",
			"namespace":        "prod",
			"deployment_name":  "checkout",
			"bearer_token_env": "CCP_KUBE_TOKEN_TEST",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+promIntegration.ID, types.UpdateIntegrationRequest{
		Mode:           stringPtr("advisory"),
		Enabled:        boolPtr(true),
		ControlEnabled: boolPtr(false),
		Metadata: types.Metadata{
			"api_base_url":     promServer.URL,
			"query_path":       "/tenant/query_range",
			"window_seconds":   "300",
			"step_seconds":     "60",
			"bearer_token_env": "CCP_PROM_TOKEN_TEST",
			"queries": []map[string]any{
				{"name": "latency_p95_ms", "category": "technical", "query": "latency", "threshold": 500, "comparator": ">", "unit": "ms"},
			},
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	kubeTest := postItemAuth[types.IntegrationTestResult](t, server.URL+"/api/v1/integrations/"+kubeIntegration.ID+"/test", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if kubeCalls.Load() == 0 {
		t.Fatal("expected kubernetes test route to hit the custom hosted status endpoint")
	}
	if !containsDetail(kubeTest.Run.Details, "deployment_name=checkout") {
		t.Fatalf("expected kubernetes test details after custom hosted proof, got %+v", kubeTest.Run)
	}

	promTest := postItemAuth[types.IntegrationTestResult](t, server.URL+"/api/v1/integrations/"+promIntegration.ID+"/test", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if promCalls.Load() == 0 {
		t.Fatal("expected prometheus test route to hit the custom hosted query endpoint")
	}
	if !containsDetail(promTest.Run.Details, "signal.latency_p95_ms=") {
		t.Fatalf("expected prometheus test details after custom hosted proof, got %+v", promTest.Run)
	}
}

func TestIntegrationMutationRoutesEnforceScopeAndRBAC(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	member := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "member@acme.local",
		DisplayName:      "Member",
		OrganizationSlug: "acme",
		Roles:            []string{"org_member"},
	})
	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "other-owner@acme.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other Org",
		OrganizationSlug: "other-org",
	})

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(integrations) == 0 {
		t.Fatal("expected seeded integrations")
	}
	targetIntegration := integrations[0]

	now := time.Now().UTC()
	repository := types.Repository{
		BaseRecord:          types.BaseRecord{ID: "repo_rbac_test", CreatedAt: now, UpdatedAt: now},
		OrganizationID:      admin.Session.ActiveOrganizationID,
		SourceIntegrationID: targetIntegration.ID,
		Name:                "checkout",
		Provider:            targetIntegration.Kind,
		URL:                 "https://example.com/acme/checkout",
		DefaultBranch:       "main",
		Status:              "discovered",
	}
	if err := application.Store.UpsertRepository(t.Context(), repository); err != nil {
		t.Fatal(err)
	}
	resource := types.DiscoveredResource{
		BaseRecord:     types.BaseRecord{ID: "res_rbac_test", CreatedAt: now, UpdatedAt: now},
		OrganizationID: admin.Session.ActiveOrganizationID,
		IntegrationID:  targetIntegration.ID,
		ResourceType:   "deployment",
		Provider:       targetIntegration.Kind,
		ExternalID:     "checkout-deployment",
		Name:           "checkout",
		Status:         "discovered",
	}
	if err := application.Store.UpsertDiscoveredResource(t.Context(), resource); err != nil {
		t.Fatal(err)
	}

	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/integrations/"+targetIntegration.ID, types.UpdateIntegrationRequest{
		Name: stringPtr("Denied Integration"),
	}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member integration update to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/integrations/"+targetIntegration.ID+"/test", struct{}{}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member integration test to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/integrations/"+targetIntegration.ID+"/sync", struct{}{}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member integration sync to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/repositories/"+repository.ID, types.UpdateRepositoryRequest{
		Status: stringPtr("mapped"),
	}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member repository mapping to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/discovered-resources/"+resource.ID, types.UpdateDiscoveredResourceRequest{
		Status: stringPtr("mapped"),
	}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member discovered-resource mapping to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/integrations/"+targetIntegration.ID+"/sync", struct{}{}, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org integration sync to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/repositories/"+repository.ID, types.UpdateRepositoryRequest{
		Status: stringPtr("cross-org"),
	}, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org repository mapping to be forbidden, got %d", status)
	}

	updated := patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+targetIntegration.ID, types.UpdateIntegrationRequest{
		Name: stringPtr(targetIntegration.Name + " Updated"),
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if !strings.Contains(updated.Name, "Updated") {
		t.Fatalf("expected admin integration update to succeed, got %+v", updated)
	}
}

func patchItemAuth[T any](t *testing.T, url string, body any, token, organizationID string, expectedStatus int) T {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-CCP-Organization-ID", organizationID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}
	var envelope types.ItemResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func stringPtr(value string) *string { return &value }
func boolPtr(value bool) *bool       { return &value }

func containsDetail(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}

func stringMetadata(metadata types.Metadata, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func marshalRSAPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
}
