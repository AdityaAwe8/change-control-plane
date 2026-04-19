package integrations

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateGitHubAppInstallationToken(t *testing.T) {
	t.Parallel()

	privateKey := marshalPrivateKeyPEM(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/app/installations/123456/access_tokens" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			t.Fatalf("expected bearer jwt, got %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":      "ghs_installation",
			"expires_at": "2026-04-16T22:00:00Z",
		})
	}))
	defer server.Close()

	token, expiresAt, err := CreateGitHubAppInstallationToken(t.Context(), server.URL, "12345", "123456", privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if token != "ghs_installation" {
		t.Fatalf("expected installation token, got %q", token)
	}
	if expiresAt == nil {
		t.Fatal("expected parsed expiration")
	}
}

func TestGitHubClientConnectionDiscoveryAndWebhookRegistration(t *testing.T) {
	t.Parallel()

	var webhookCreates int
	var webhookUpdates int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ghs_installation" {
			t.Fatalf("expected github installation token, got %q", got)
		}
		switch {
		case r.URL.Path == "/orgs/acme" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "acme"})
		case r.URL.Path == "/orgs/acme/repos" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"name":           "checkout",
				"full_name":      "acme/checkout",
				"html_url":       "https://github.example.com/acme/checkout",
				"default_branch": "main",
				"private":        true,
				"archived":       false,
				"owner": map[string]any{
					"login": "acme",
				},
			}})
		case r.URL.Path == "/orgs/acme/hooks" && r.Method == http.MethodGet:
			if webhookCreates == 0 {
				_ = json.NewEncoder(w).Encode([]map[string]any{})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id":     303,
				"active": true,
				"config": map[string]any{"url": "https://ccp.example.com/api/v1/integrations/github/webhooks/github"},
			}})
		case r.URL.Path == "/orgs/acme/hooks" && r.Method == http.MethodPost:
			webhookCreates++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 303})
		case r.URL.Path == "/orgs/acme/hooks/303" && r.Method == http.MethodPatch:
			webhookUpdates++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 303})
		default:
			t.Fatalf("unexpected github request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "ghs_installation")
	details, err := client.TestConnection(t.Context(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if !containsGitHubString(details, "resolved github principal acme") {
		t.Fatalf("expected github principal detail, got %+v", details)
	}

	repositories, err := client.DiscoverRepositories(t.Context(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(repositories) != 1 || repositories[0].Provider != "github" || repositories[0].FullName != "acme/checkout" {
		t.Fatalf("expected normalized github repository, got %+v", repositories)
	}

	registration, err := client.EnsureOrganizationWebhook(t.Context(), "acme", "https://ccp.example.com/api/v1/integrations/github/webhooks/github", "hook-secret")
	if err != nil {
		t.Fatal(err)
	}
	if registration.ExternalHookID != "303" || webhookCreates != 1 {
		t.Fatalf("expected webhook create evidence, got registration=%+v creates=%d", registration, webhookCreates)
	}

	registration, err = client.EnsureOrganizationWebhook(t.Context(), "acme", "https://ccp.example.com/api/v1/integrations/github/webhooks/github", "hook-secret")
	if err != nil {
		t.Fatal(err)
	}
	if registration.ExternalHookID != "303" || webhookUpdates != 1 {
		t.Fatalf("expected webhook update evidence, got registration=%+v updates=%d", registration, webhookUpdates)
	}
}

func TestGitHubClientLoadCODEOWNERS(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ghs_installation" {
			t.Fatalf("expected github installation token, got %q", got)
		}
		switch r.URL.Path {
		case "/repos/acme/checkout/contents/.github/CODEOWNERS":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content":  "IyB0b3AgbGV2ZWwKKiBAYWNtZS9wbGF0Zm9ybSBwbGF0Zm9ybUBhY21lLmxvY2FsCi9hcHBzL2NoZWNrb3V0LyogQGFjbWUvY2hlY2tvdXQK",
				"encoding": "base64",
				"sha":      "sha-codeowners",
			})
		default:
			t.Fatalf("unexpected github request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "ghs_installation")
	result, err := client.LoadCODEOWNERS(t.Context(), SCMRepository{
		Provider:      "github",
		Owner:         "acme",
		Name:          "checkout",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "imported" || result.FilePath != ".github/CODEOWNERS" || result.Revision != "sha-codeowners" {
		t.Fatalf("expected imported codeowners result, got %+v", result)
	}
	if len(result.Rules) != 2 || len(result.Owners) != 3 {
		t.Fatalf("expected parsed rules and owners, got %+v", result)
	}
}

func TestGitHubClientLoadCODEOWNERSNotFound(t *testing.T) {
	t.Parallel()

	var requestedPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewGitHubClient(server.URL, "ghs_installation")
	result, err := client.LoadCODEOWNERS(t.Context(), SCMRepository{
		Provider:      "github",
		Owner:         "acme",
		Name:          "checkout",
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

func marshalPrivateKeyPEM(t *testing.T) string {
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

func containsGitHubString(items []string, needle string) bool {
	for _, item := range items {
		if strings.Contains(item, needle) {
			return true
		}
	}
	return false
}
