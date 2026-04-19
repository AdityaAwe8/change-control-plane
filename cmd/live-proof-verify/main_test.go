package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestRunLiveProofVerifyWithGitLab(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_LIVE_TEST_GITLAB_TOKEN", "glpat-test")
	t.Setenv("CCP_LIVE_TEST_GITLAB_WEBHOOK_SECRET", "gitlab-webhook-secret")
	t.Setenv("CCP_LIVE_TEST_KUBE_TOKEN", "kube-secret")
	t.Setenv("CCP_LIVE_TEST_PROM_TOKEN", "prom-secret")

	apiURL := startAPIServer(t)
	signUpUser(t, apiURL, "gitlab-proof@acme.local", "ProofPass123!")

	var webhookID int
	gitlabServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("PRIVATE-TOKEN"); got != "glpat-test" {
			t.Fatalf("expected gitlab token header, got %q", got)
		}
		switch {
		case r.URL.Path == "/user":
			_ = json.NewEncoder(w).Encode(map[string]any{"username": "proof-bot"})
		case r.URL.Path == "/groups/acme":
			_ = json.NewEncoder(w).Encode(map[string]any{"full_path": "acme"})
		case r.URL.Path == "/groups/acme/projects":
			_ = json.NewEncoder(w).Encode([]map[string]any{{
				"id":                  101,
				"name":                "checkout",
				"path_with_namespace": "acme/checkout",
				"web_url":             "https://gitlab.example.com/acme/checkout",
				"default_branch":      "main",
				"visibility":          "private",
				"namespace": map[string]any{
					"full_path": "acme",
					"path":      "acme",
					"name":      "Acme",
				},
			}})
		case r.URL.Path == "/groups/acme/hooks" && r.Method == http.MethodGet:
			if webhookID == 0 {
				_ = json.NewEncoder(w).Encode([]map[string]any{})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": webhookID, "url": "http://control-plane.local/api/v1/integrations/live-proof-gitlab/webhooks/gitlab"}})
		case r.URL.Path == "/groups/acme/hooks" && r.Method == http.MethodPost:
			webhookID = 202
			_ = json.NewEncoder(w).Encode(map[string]any{"id": webhookID})
		case r.URL.Path == "/groups/acme/hooks/202" && r.Method == http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 202})
		case strings.HasPrefix(r.URL.Path, "/projects/101/repository/files/") && strings.Contains(r.URL.Path, "CODEOWNERS"):
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "404 File Not Found"})
		default:
			t.Fatalf("unexpected gitlab request %s %s", r.Method, r.URL.String())
		}
	}))
	defer gitlabServer.Close()

	kubeServer := newKubeProofServer(t)
	defer kubeServer.Close()
	promServer := newPromProofServer(t)
	defer promServer.Close()

	reportPath := filepath.Join(t.TempDir(), "gitlab-live-proof.json")
	var stdout, stderr bytes.Buffer
	exitCode := run(context.Background(), []string{
		"--api-base-url", apiURL,
		"--admin-email", "gitlab-proof@acme.local",
		"--admin-password", "ProofPass123!",
		"--scm-kind", "gitlab",
		"--gitlab-base-url", gitlabServer.URL,
		"--gitlab-group", "acme",
		"--gitlab-token-env", "CCP_LIVE_TEST_GITLAB_TOKEN",
		"--gitlab-webhook-secret-env", "CCP_LIVE_TEST_GITLAB_WEBHOOK_SECRET",
		"--kubernetes-base-url", kubeServer.URL,
		"--kubernetes-token-env", "CCP_LIVE_TEST_KUBE_TOKEN",
		"--kubernetes-namespace", "prod",
		"--kubernetes-deployment", "checkout",
		"--kubernetes-status-path", "/custom/status/checkout",
		"--prometheus-base-url", promServer.URL,
		"--prometheus-token-env", "CCP_LIVE_TEST_PROM_TOKEN",
		"--prometheus-query", `request_latency_ms{service="checkout",environment="production"}`,
		"--report", reportPath,
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success, got exit code %d stderr=%s", exitCode, stderr.String())
	}

	var report liveProofReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Profile != "live" || report.SCMKind != "gitlab" {
		t.Fatalf("expected live gitlab report, got %+v", report)
	}
	if report.EnvironmentClass != proofEnvironmentHostedLike || report.ProofQuality != proofQualityMeaningful {
		t.Fatalf("expected hosted-like meaningful proof classification, got %+v", report)
	}
	if report.ConfigSummary.SCM.Kind != "gitlab" || report.ConfigSummary.SCM.Endpoint.EndpointClass != "local" {
		t.Fatalf("expected gitlab config summary, got %+v", report.ConfigSummary)
	}
	if len(report.Checks) == 0 || len(report.EvidenceSummary) == 0 {
		t.Fatalf("expected checks and evidence summary, got %+v", report)
	}
	if report.GitLabIntegration == nil || report.Repository.Provider != "gitlab" {
		t.Fatalf("expected gitlab integration and repository proof, got %+v", report)
	}
	if report.KubernetesResource.ResourceType != "kubernetes_workload" || report.PrometheusResource.ResourceType != "prometheus_signal_target" {
		t.Fatalf("expected mapped runtime resources, got %+v", report)
	}
	if _, err := os.ReadFile(reportPath); err != nil {
		t.Fatalf("expected report file: %v", err)
	}
	assertNoSecretLeak(t, stdout.String(), "glpat-test", "kube-secret", "prom-secret")
	fileBody, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected report file: %v", err)
	}
	assertNoSecretLeak(t, string(fileBody), "glpat-test", "kube-secret", "prom-secret")

	var validated bytes.Buffer
	exitCode = run(context.Background(), []string{
		"--validate-report", reportPath,
	}, &validated, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected validation success, got exit code %d stderr=%s", exitCode, stderr.String())
	}
}

func TestRunLiveProofVerifyWithGitHub(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_LIVE_TEST_GITHUB_PRIVATE_KEY", marshalRSAPrivateKeyPEM(t))
	t.Setenv("CCP_LIVE_TEST_GITHUB_WEBHOOK_SECRET", "github-webhook-secret")
	t.Setenv("CCP_LIVE_TEST_KUBE_TOKEN", "kube-secret")
	t.Setenv("CCP_LIVE_TEST_PROM_TOKEN", "prom-secret")

	apiURL := startAPIServer(t)
	signUpUser(t, apiURL, "github-proof@acme.local", "ProofPass123!")

	var webhookID int
	githubServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/app/installations/987654/access_tokens":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token":      "ghs_installation_token",
				"expires_at": "2026-04-16T21:00:00Z",
			})
		case r.URL.Path == "/orgs/acme":
			if got := r.Header.Get("Authorization"); got != "Bearer ghs_installation_token" {
				t.Fatalf("expected github installation token, got %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "acme"})
		case r.URL.Path == "/orgs/acme/repos":
			if got := r.Header.Get("Authorization"); got != "Bearer ghs_installation_token" {
				t.Fatalf("expected github installation token, got %q", got)
			}
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
		case r.URL.Path == "/orgs/acme/hooks":
			if got := r.Header.Get("Authorization"); got != "Bearer ghs_installation_token" {
				t.Fatalf("expected github installation token, got %q", got)
			}
			switch r.Method {
			case http.MethodGet:
				if webhookID == 0 {
					_ = json.NewEncoder(w).Encode([]map[string]any{})
					return
				}
				_ = json.NewEncoder(w).Encode([]map[string]any{{
					"id":     webhookID,
					"active": true,
					"config": map[string]any{"url": "http://control-plane.local/api/v1/integrations/live-proof-github/webhooks/github"},
				}})
			case http.MethodPost:
				webhookID = 303
				_ = json.NewEncoder(w).Encode(map[string]any{"id": webhookID})
			default:
				t.Fatalf("unexpected github hook method %s", r.Method)
			}
		case strings.HasPrefix(r.URL.Path, "/repos/acme/checkout/contents/") && strings.Contains(r.URL.Path, "CODEOWNERS"):
			if got := r.Header.Get("Authorization"); got != "Bearer ghs_installation_token" {
				t.Fatalf("expected github installation token, got %q", got)
			}
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
		default:
			t.Fatalf("unexpected github request %s %s", r.Method, r.URL.String())
		}
	}))
	defer githubServer.Close()

	kubeServer := newKubeProofServer(t)
	defer kubeServer.Close()
	promServer := newPromProofServer(t)
	defer promServer.Close()

	reportPath := filepath.Join(t.TempDir(), "github-live-proof.json")
	var stdout, stderr bytes.Buffer
	exitCode := run(context.Background(), []string{
		"--api-base-url", apiURL,
		"--admin-email", "github-proof@acme.local",
		"--admin-password", "ProofPass123!",
		"--scm-kind", "github",
		"--github-base-url", githubServer.URL,
		"--github-web-base-url", githubServer.URL,
		"--github-owner", "acme",
		"--github-app-id", "123456",
		"--github-app-slug", "change-control-plane",
		"--github-private-key-env", "CCP_LIVE_TEST_GITHUB_PRIVATE_KEY",
		"--github-webhook-secret-env", "CCP_LIVE_TEST_GITHUB_WEBHOOK_SECRET",
		"--github-installation-id", "987654",
		"--kubernetes-base-url", kubeServer.URL,
		"--kubernetes-token-env", "CCP_LIVE_TEST_KUBE_TOKEN",
		"--kubernetes-namespace", "prod",
		"--kubernetes-deployment", "checkout",
		"--kubernetes-status-path", "/custom/status/checkout",
		"--prometheus-base-url", promServer.URL,
		"--prometheus-token-env", "CCP_LIVE_TEST_PROM_TOKEN",
		"--prometheus-query", `request_latency_ms{service="checkout",environment="production"}`,
		"--report", reportPath,
	}, &stdout, &stderr)
	if exitCode != 0 {
		t.Fatalf("expected success, got exit code %d stderr=%s", exitCode, stderr.String())
	}

	var report liveProofReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.Profile != "live" || report.SCMKind != "github" {
		t.Fatalf("expected live github report, got %+v", report)
	}
	if report.EnvironmentClass != proofEnvironmentHostedLike || report.ProofQuality != proofQualityMeaningful {
		t.Fatalf("expected hosted-like meaningful proof classification, got %+v", report)
	}
	if report.ConfigSummary.SCM.Kind != "github" || report.ConfigSummary.SCM.Endpoint.EndpointClass != "local" {
		t.Fatalf("expected github config summary, got %+v", report.ConfigSummary)
	}
	if report.GitHubIntegration == nil || report.Repository.Provider != "github" {
		t.Fatalf("expected github integration and repository proof, got %+v", report)
	}
	if report.GitHubOnboardingStart == nil || report.GitHubOnboardingCompletion == nil {
		t.Fatalf("expected github onboarding evidence, got %+v", report)
	}
	if _, err := os.ReadFile(reportPath); err != nil {
		t.Fatalf("expected report file: %v", err)
	}
	assertNoSecretLeak(t, stdout.String(), "github-webhook-secret", "kube-secret", "prom-secret", "BEGIN RSA PRIVATE KEY")
}

func TestRunLiveProofVerifyRejectsIncompleteReport(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	reportPath := filepath.Join(t.TempDir(), "invalid-live-proof.json")
	body, err := json.Marshal(liveProofReport{
		Profile:          proofProfileLive,
		EnvironmentClass: proofEnvironmentHostedLike,
		ProofQuality:     proofQualityMeaningful,
		VerifiedAt:       "2026-04-18T18:00:00Z",
		SCMKind:          "gitlab",
		ConfigSummary: liveProofConfigSummary{
			APIBaseURL: liveProofEndpointSummary{URL: "http://127.0.0.1:8080", Host: "127.0.0.1", EndpointClass: "local"},
			SCM:        liveProofProviderConfigSummary{Kind: "gitlab", Endpoint: liveProofEndpointSummary{URL: "https://gitlab.example.com/api/v4", Host: "gitlab.example.com", EndpointClass: "public"}},
		},
		Checks:          []liveProofCheck{{Provider: "gitlab", Stage: "config_validation", Status: checkStatusPassed, Summary: "ok"}},
		EvidenceSummary: []string{"organization=live-proof"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, body, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exitCode := run(context.Background(), []string{
		"--validate-report", reportPath,
	}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("expected invalid report validation failure, stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "gitlab report requires gitlab_integration") {
		t.Fatalf("expected validation error details, got stderr=%s", stderr.String())
	}
}

func TestRunLiveProofVerifyRejectsInvalidEnvironmentClass(t *testing.T) {
	t.Setenv("CCP_LIVE_TEST_GITLAB_TOKEN", "glpat-test")
	t.Setenv("CCP_LIVE_TEST_GITLAB_WEBHOOK_SECRET", "gitlab-webhook-secret")
	t.Setenv("CCP_LIVE_TEST_KUBE_TOKEN", "kube-secret")
	t.Setenv("CCP_LIVE_TEST_PROM_TOKEN", "prom-secret")

	var stdout, stderr bytes.Buffer
	exitCode := run(context.Background(), []string{
		"--api-base-url", "http://127.0.0.1:18080",
		"--environment-class", "mystery",
		"--scm-kind", "gitlab",
		"--gitlab-base-url", "https://gitlab.example.com/api/v4",
		"--gitlab-group", "acme",
		"--gitlab-token-env", "CCP_LIVE_TEST_GITLAB_TOKEN",
		"--gitlab-webhook-secret-env", "CCP_LIVE_TEST_GITLAB_WEBHOOK_SECRET",
		"--kubernetes-base-url", "https://kubernetes.example.com",
		"--kubernetes-token-env", "CCP_LIVE_TEST_KUBE_TOKEN",
		"--kubernetes-namespace", "prod",
		"--kubernetes-deployment", "checkout",
		"--prometheus-base-url", "https://prometheus.example.com",
		"--prometheus-token-env", "CCP_LIVE_TEST_PROM_TOKEN",
		"--prometheus-query", "request_latency_ms",
	}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("expected invalid environment class failure")
	}
	if !strings.Contains(stderr.String(), "environment-class must be one of") {
		t.Fatalf("expected environment-class validation error, got %s", stderr.String())
	}
}

func TestRunLiveProofVerifyRejectsMissingSecretEnv(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := run(context.Background(), []string{
		"--api-base-url", "http://127.0.0.1:18080",
		"--scm-kind", "gitlab",
		"--gitlab-base-url", "https://gitlab.example.com/api/v4",
		"--gitlab-group", "acme",
		"--gitlab-token-env", "CCP_LIVE_TEST_GITLAB_TOKEN",
		"--gitlab-webhook-secret-env", "CCP_LIVE_TEST_GITLAB_WEBHOOK_SECRET",
		"--kubernetes-base-url", "https://kubernetes.example.com",
		"--kubernetes-namespace", "prod",
		"--kubernetes-deployment", "checkout",
		"--prometheus-base-url", "https://prometheus.example.com",
		"--prometheus-query", "request_latency_ms",
	}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("expected missing env validation failure")
	}
	if !strings.Contains(stderr.String(), "CCP_LIVE_TEST_GITLAB_TOKEN") {
		t.Fatalf("expected missing env name in validation error, got %s", stderr.String())
	}
}

func TestRunLiveProofVerifyRejectsHostedSaaSAgainstLocalSCMEndpoint(t *testing.T) {
	t.Setenv("CCP_LIVE_TEST_GITHUB_PRIVATE_KEY", marshalRSAPrivateKeyPEM(t))
	t.Setenv("CCP_LIVE_TEST_GITHUB_WEBHOOK_SECRET", "github-webhook-secret")
	t.Setenv("CCP_LIVE_TEST_KUBE_TOKEN", "kube-secret")
	t.Setenv("CCP_LIVE_TEST_PROM_TOKEN", "prom-secret")

	var stdout, stderr bytes.Buffer
	exitCode := run(context.Background(), []string{
		"--api-base-url", "http://127.0.0.1:18080",
		"--environment-class", proofEnvironmentHostedSaaS,
		"--scm-kind", "github",
		"--github-base-url", "http://127.0.0.1:8089",
		"--github-web-base-url", "http://127.0.0.1:8089",
		"--github-owner", "acme",
		"--github-app-id", "123456",
		"--github-app-slug", "change-control-plane",
		"--github-private-key-env", "CCP_LIVE_TEST_GITHUB_PRIVATE_KEY",
		"--github-webhook-secret-env", "CCP_LIVE_TEST_GITHUB_WEBHOOK_SECRET",
		"--github-installation-id", "987654",
		"--kubernetes-base-url", "https://kubernetes.example.com",
		"--kubernetes-token-env", "CCP_LIVE_TEST_KUBE_TOKEN",
		"--kubernetes-namespace", "prod",
		"--kubernetes-deployment", "checkout",
		"--prometheus-base-url", "https://prometheus.example.com",
		"--prometheus-token-env", "CCP_LIVE_TEST_PROM_TOKEN",
		"--prometheus-query", "request_latency_ms",
	}, &stdout, &stderr)
	if exitCode == 0 {
		t.Fatalf("expected hosted_saas/local github mismatch failure")
	}
	if !strings.Contains(stderr.String(), "github-base-url must be publicly hosted") {
		t.Fatalf("expected hosted_saas endpoint validation error, got %s", stderr.String())
	}
}

func startAPIServer(t *testing.T) string {
	t.Helper()
	cfg := common.LoadConfig()
	cfg.APIBaseURL = "http://control-plane.local"
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	t.Cleanup(server.Close)
	return server.URL
}

func signUpUser(t *testing.T, serverURL, email, password string) {
	t.Helper()
	body, err := json.Marshal(types.SignUpRequest{
		Email:                email,
		DisplayName:          "Proof User",
		Password:             password,
		PasswordConfirmation: password,
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(serverURL+"/api/v1/auth/sign-up", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected sign up success, got %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
}

func newKubeProofServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer kube-secret" {
			t.Fatalf("expected kubernetes bearer token, got %q", got)
		}
		switch r.URL.Path {
		case "/custom/status/checkout":
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
			t.Fatalf("unexpected kubernetes request %s", r.URL.String())
		}
	}))
}

func newPromProofServer(t *testing.T) *httptest.Server {
	t.Helper()
	return newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer prom-secret" {
			t.Fatalf("expected prometheus bearer token, got %q", got)
		}
		if r.URL.Path != "/api/v1/query_range" {
			t.Fatalf("unexpected prometheus path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result": []map[string]any{{
					"metric": map[string]any{"__name__": "request_latency_ms"},
					"values": [][]any{
						{1.0, "180"},
						{2.0, "220"},
					},
				}},
			},
		})
	}))
}

func newLocalIPv4Server(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(handler)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server.Listener = listener
	server.Start()
	return server
}

func marshalRSAPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	return string(pem.EncodeToMemory(block))
}

func assertNoSecretLeak(t *testing.T, body string, forbidden ...string) {
	t.Helper()
	for _, token := range forbidden {
		if token == "" {
			continue
		}
		if strings.Contains(body, token) {
			t.Fatalf("expected report output to redact secret value %q", token)
		}
	}
}
