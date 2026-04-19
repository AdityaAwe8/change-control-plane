package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/client"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type liveProofReport struct {
	Profile                    string                             `json:"profile"`
	EnvironmentClass           string                             `json:"environment_class"`
	ProofQuality               string                             `json:"proof_quality"`
	VerifiedAt                 string                             `json:"verified_at"`
	Warnings                   []string                           `json:"warnings,omitempty"`
	EvidenceSummary            []string                           `json:"evidence_summary,omitempty"`
	ConfigSummary              liveProofConfigSummary             `json:"config_summary"`
	Checks                     []liveProofCheck                   `json:"checks"`
	SCMKind                    string                             `json:"scm_kind"`
	Organization               types.Organization                 `json:"organization"`
	Project                    types.Project                      `json:"project"`
	Team                       types.Team                         `json:"team"`
	Service                    types.Service                      `json:"service"`
	Environment                types.Environment                  `json:"environment"`
	GitHubOnboardingStart      *types.GitHubOnboardingStartResult `json:"github_onboarding_start,omitempty"`
	GitHubOnboardingCompletion *types.Integration                 `json:"github_onboarding_completion,omitempty"`
	GitHubIntegration          *types.Integration                 `json:"github_integration,omitempty"`
	GitLabIntegration          *types.Integration                 `json:"gitlab_integration,omitempty"`
	KubernetesIntegration      types.Integration                  `json:"kubernetes_integration"`
	PrometheusIntegration      types.Integration                  `json:"prometheus_integration"`
	SCMWebhookRegistration     types.WebhookRegistrationResult    `json:"scm_webhook_registration"`
	SCMTestResult              types.IntegrationTestResult        `json:"scm_test_result"`
	SCMSyncResult              types.IntegrationSyncResult        `json:"scm_sync_result"`
	KubernetesTestResult       types.IntegrationTestResult        `json:"kubernetes_test_result"`
	KubernetesSyncResult       types.IntegrationSyncResult        `json:"kubernetes_sync_result"`
	PrometheusTestResult       types.IntegrationTestResult        `json:"prometheus_test_result"`
	PrometheusSyncResult       types.IntegrationSyncResult        `json:"prometheus_sync_result"`
	Repository                 types.Repository                   `json:"repository"`
	KubernetesResource         types.DiscoveredResource           `json:"kubernetes_resource"`
	PrometheusResource         types.DiscoveredResource           `json:"prometheus_resource"`
	CoverageSummary            types.CoverageSummary              `json:"coverage_summary"`
}

type liveProofConfigSummary struct {
	APIBaseURL liveProofEndpointSummary       `json:"api_base_url"`
	SCM        liveProofProviderConfigSummary `json:"scm"`
	Kubernetes liveProofProviderConfigSummary `json:"kubernetes"`
	Prometheus liveProofProviderConfigSummary `json:"prometheus"`
}

type liveProofProviderConfigSummary struct {
	Kind            string                    `json:"kind"`
	ScopeType       string                    `json:"scope_type,omitempty"`
	ScopeName       string                    `json:"scope_name,omitempty"`
	AuthStrategy    string                    `json:"auth_strategy,omitempty"`
	Endpoint        liveProofEndpointSummary  `json:"endpoint"`
	WebEndpoint     *liveProofEndpointSummary `json:"web_endpoint,omitempty"`
	SecretEnvs      []liveProofSecretSummary  `json:"secret_envs,omitempty"`
	RepositoryHint  string                    `json:"repository_hint,omitempty"`
	ResourceHint    string                    `json:"resource_hint,omitempty"`
	QueryName       string                    `json:"query_name,omitempty"`
	QueryWindowSecs int                       `json:"query_window_seconds,omitempty"`
	QueryStepSecs   int                       `json:"query_step_seconds,omitempty"`
}

type liveProofEndpointSummary struct {
	URL           string `json:"url"`
	Host          string `json:"host"`
	EndpointClass string `json:"endpoint_class"`
}

type liveProofSecretSummary struct {
	EnvName     string `json:"env_name"`
	Configured  bool   `json:"configured"`
	RequiredFor string `json:"required_for,omitempty"`
}

type liveProofCheck struct {
	Provider string   `json:"provider"`
	Stage    string   `json:"stage"`
	Status   string   `json:"status"`
	Summary  string   `json:"summary"`
	Details  []string `json:"details,omitempty"`
	Hint     string   `json:"hint,omitempty"`
}

type liveProofInput struct {
	APIBaseURL             string
	EnvironmentClass       string
	SCMKind                string
	GitLabBaseURL          string
	GitLabGroup            string
	GitLabTokenEnv         string
	GitLabWebhookSecretEnv string
	GitHubBaseURL          string
	GitHubWebBaseURL       string
	GitHubOwner            string
	GitHubAppID            string
	GitHubAppSlug          string
	GitHubPrivateKeyEnv    string
	GitHubWebhookSecretEnv string
	GitHubInstallationID   string
	KubernetesBaseURL      string
	KubernetesTokenEnv     string
	KubernetesNamespace    string
	KubernetesDeployment   string
	KubernetesStatusPath   string
	PrometheusBaseURL      string
	PrometheusTokenEnv     string
	PrometheusQueryName    string
	PrometheusQuery        string
	PrometheusThreshold    float64
	PrometheusComparator   string
	PrometheusUnit         string
	PrometheusSeverity     string
	PrometheusWindowSecs   int
	PrometheusStepSecs     int
}

const (
	proofProfileLive           = "live"
	proofEnvironmentHostedLike = "hosted_like"
	proofEnvironmentCustomer   = "customer_environment"
	proofEnvironmentHostedSaaS = "hosted_saas"
	proofQualityMeaningful     = "meaningful"
	proofQualityMeaningfulWarn = "meaningful_with_warnings"
	checkStatusPassed          = "passed"
	checkStatusWarning         = "warning"
)

var envVarNamePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("live-proof-verify", flag.ContinueOnError)
	flags.SetOutput(stderr)

	apiBaseURL := flags.String("api-base-url", valueOrDefault("CCP_LIVE_PROOF_API_BASE_URL", "http://127.0.0.1:8080"), "control plane api base url")
	adminEmail := flags.String("admin-email", valueOrDefault("CCP_LIVE_PROOF_ADMIN_EMAIL", "admin@changecontrolplane.local"), "proof admin email")
	adminPassword := flags.String("admin-password", valueOrDefault("CCP_LIVE_PROOF_ADMIN_PASSWORD", "ChangeMe123!"), "proof admin password")
	orgName := flags.String("org-name", valueOrDefault("CCP_LIVE_PROOF_ORG_NAME", "Live Proof"), "organization name")
	orgSlug := flags.String("org-slug", valueOrDefault("CCP_LIVE_PROOF_ORG_SLUG", "live-proof"), "organization slug")
	projectName := flags.String("project-name", valueOrDefault("CCP_LIVE_PROOF_PROJECT_NAME", "Proof Platform"), "project name")
	projectSlug := flags.String("project-slug", valueOrDefault("CCP_LIVE_PROOF_PROJECT_SLUG", "proof-platform"), "project slug")
	teamName := flags.String("team-name", valueOrDefault("CCP_LIVE_PROOF_TEAM_NAME", "Proof Team"), "team name")
	teamSlug := flags.String("team-slug", valueOrDefault("CCP_LIVE_PROOF_TEAM_SLUG", "proof-team"), "team slug")
	serviceName := flags.String("service-name", valueOrDefault("CCP_LIVE_PROOF_SERVICE_NAME", "Checkout"), "service name")
	serviceSlug := flags.String("service-slug", valueOrDefault("CCP_LIVE_PROOF_SERVICE_SLUG", "checkout"), "service slug")
	environmentName := flags.String("environment-name", valueOrDefault("CCP_LIVE_PROOF_ENV_NAME", "Production"), "environment name")
	environmentSlug := flags.String("environment-slug", valueOrDefault("CCP_LIVE_PROOF_ENV_SLUG", "production"), "environment slug")
	scmKind := flags.String("scm-kind", valueOrDefault("CCP_LIVE_PROOF_SCM_KIND", "gitlab"), "scm provider kind: gitlab or github")
	environmentClass := flags.String("environment-class", valueOrDefault("CCP_LIVE_PROOF_ENVIRONMENT_CLASS", proofEnvironmentHostedLike), "proof environment class: hosted_like, customer_environment, or hosted_saas")
	reportPath := flags.String("report", valueOrDefault("CCP_LIVE_PROOF_REPORT", ""), "optional path to write the proof report JSON")
	validateReportPath := flags.String("validate-report", valueOrDefault("CCP_LIVE_PROOF_VALIDATE_REPORT", ""), "validate an existing proof report JSON and print the normalized result")

	gitlabBaseURL := flags.String("gitlab-base-url", valueOrDefault("CCP_LIVE_PROOF_GITLAB_BASE_URL", "https://gitlab.com/api/v4"), "gitlab api base url")
	gitlabGroup := flags.String("gitlab-group", valueOrDefault("CCP_LIVE_PROOF_GITLAB_GROUP", ""), "gitlab group scope")
	gitlabTokenEnv := flags.String("gitlab-token-env", valueOrDefault("CCP_LIVE_PROOF_GITLAB_TOKEN_ENV", ""), "gitlab access token env name")
	gitlabWebhookSecretEnv := flags.String("gitlab-webhook-secret-env", valueOrDefault("CCP_LIVE_PROOF_GITLAB_WEBHOOK_SECRET_ENV", ""), "gitlab webhook secret env name")

	githubBaseURL := flags.String("github-base-url", valueOrDefault("CCP_LIVE_PROOF_GITHUB_API_BASE_URL", "https://api.github.com"), "github api base url")
	githubWebBaseURL := flags.String("github-web-base-url", valueOrDefault("CCP_LIVE_PROOF_GITHUB_WEB_BASE_URL", "https://github.com"), "github web base url")
	githubOwner := flags.String("github-owner", valueOrDefault("CCP_LIVE_PROOF_GITHUB_OWNER", ""), "github owner or organization")
	githubAppID := flags.String("github-app-id", valueOrDefault("CCP_LIVE_PROOF_GITHUB_APP_ID", ""), "github app id")
	githubAppSlug := flags.String("github-app-slug", valueOrDefault("CCP_LIVE_PROOF_GITHUB_APP_SLUG", ""), "github app slug")
	githubPrivateKeyEnv := flags.String("github-private-key-env", valueOrDefault("CCP_LIVE_PROOF_GITHUB_PRIVATE_KEY_ENV", ""), "github app private key env name")
	githubWebhookSecretEnv := flags.String("github-webhook-secret-env", valueOrDefault("CCP_LIVE_PROOF_GITHUB_WEBHOOK_SECRET_ENV", ""), "github webhook secret env name")
	githubInstallationID := flags.String("github-installation-id", valueOrDefault("CCP_LIVE_PROOF_GITHUB_INSTALLATION_ID", ""), "github installation id for non-interactive onboarding completion")

	kubernetesBaseURL := flags.String("kubernetes-base-url", valueOrDefault("CCP_LIVE_PROOF_KUBE_API_BASE_URL", ""), "kubernetes api base url")
	kubernetesTokenEnv := flags.String("kubernetes-token-env", valueOrDefault("CCP_LIVE_PROOF_KUBE_TOKEN_ENV", ""), "kubernetes bearer token env name")
	kubernetesNamespace := flags.String("kubernetes-namespace", valueOrDefault("CCP_LIVE_PROOF_KUBE_NAMESPACE", ""), "kubernetes namespace")
	kubernetesDeployment := flags.String("kubernetes-deployment", valueOrDefault("CCP_LIVE_PROOF_KUBE_DEPLOYMENT", ""), "kubernetes deployment name")
	kubernetesStatusPath := flags.String("kubernetes-status-path", valueOrDefault("CCP_LIVE_PROOF_KUBE_STATUS_PATH", ""), "optional kubernetes custom status path")

	prometheusBaseURL := flags.String("prometheus-base-url", valueOrDefault("CCP_LIVE_PROOF_PROMETHEUS_BASE_URL", ""), "prometheus api base url")
	prometheusTokenEnv := flags.String("prometheus-token-env", valueOrDefault("CCP_LIVE_PROOF_PROMETHEUS_TOKEN_ENV", ""), "prometheus bearer token env name")
	prometheusQueryName := flags.String("prometheus-query-name", valueOrDefault("CCP_LIVE_PROOF_PROMETHEUS_QUERY_NAME", "request_latency_ms"), "prometheus query display name")
	prometheusQuery := flags.String("prometheus-query", valueOrDefault("CCP_LIVE_PROOF_PROMETHEUS_QUERY", ""), "prometheus query expression")
	prometheusThreshold := flags.Float64("prometheus-threshold", valueOrDefaultFloat("CCP_LIVE_PROOF_PROMETHEUS_THRESHOLD", 200), "prometheus alert threshold")
	prometheusComparator := flags.String("prometheus-comparator", valueOrDefault("CCP_LIVE_PROOF_PROMETHEUS_COMPARATOR", ">="), "prometheus comparator")
	prometheusUnit := flags.String("prometheus-unit", valueOrDefault("CCP_LIVE_PROOF_PROMETHEUS_UNIT", "ms"), "prometheus unit")
	prometheusSeverity := flags.String("prometheus-severity", valueOrDefault("CCP_LIVE_PROOF_PROMETHEUS_SEVERITY", "critical"), "prometheus severity")
	prometheusWindowSeconds := flags.Int("prometheus-window-seconds", valueOrDefaultInt("CCP_LIVE_PROOF_PROMETHEUS_WINDOW_SECONDS", 300), "prometheus collection window seconds")
	prometheusStepSeconds := flags.Int("prometheus-step-seconds", valueOrDefaultInt("CCP_LIVE_PROOF_PROMETHEUS_STEP_SECONDS", 30), "prometheus collection step seconds")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*validateReportPath) != "" {
		report, err := readLiveProofReport(*validateReportPath)
		if err != nil {
			fmt.Fprintf(stderr, "read proof report failed: %v\n", err)
			return 1
		}
		if err := validateLiveProofReport(report); err != nil {
			fmt.Fprintf(stderr, "invalid proof report: %v\n", err)
			return 1
		}
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "marshal report failed: %v\n", err)
			return 1
		}
		_, _ = stdout.Write(append(body, '\n'))
		return 0
	}

	input := liveProofInput{
		APIBaseURL:             *apiBaseURL,
		EnvironmentClass:       *environmentClass,
		SCMKind:                *scmKind,
		GitLabBaseURL:          *gitlabBaseURL,
		GitLabGroup:            *gitlabGroup,
		GitLabTokenEnv:         *gitlabTokenEnv,
		GitLabWebhookSecretEnv: *gitlabWebhookSecretEnv,
		GitHubBaseURL:          *githubBaseURL,
		GitHubWebBaseURL:       *githubWebBaseURL,
		GitHubOwner:            *githubOwner,
		GitHubAppID:            *githubAppID,
		GitHubAppSlug:          *githubAppSlug,
		GitHubPrivateKeyEnv:    *githubPrivateKeyEnv,
		GitHubWebhookSecretEnv: *githubWebhookSecretEnv,
		GitHubInstallationID:   *githubInstallationID,
		KubernetesBaseURL:      *kubernetesBaseURL,
		KubernetesTokenEnv:     *kubernetesTokenEnv,
		KubernetesNamespace:    *kubernetesNamespace,
		KubernetesDeployment:   *kubernetesDeployment,
		KubernetesStatusPath:   *kubernetesStatusPath,
		PrometheusBaseURL:      *prometheusBaseURL,
		PrometheusTokenEnv:     *prometheusTokenEnv,
		PrometheusQueryName:    *prometheusQueryName,
		PrometheusQuery:        *prometheusQuery,
		PrometheusThreshold:    *prometheusThreshold,
		PrometheusComparator:   *prometheusComparator,
		PrometheusUnit:         *prometheusUnit,
		PrometheusSeverity:     *prometheusSeverity,
		PrometheusWindowSecs:   *prometheusWindowSeconds,
		PrometheusStepSecs:     *prometheusStepSeconds,
	}
	configSummary := buildLiveProofConfigSummary(input)
	preflightChecks, preflightWarnings, err := validateLiveProofInput(input)
	if err != nil {
		fmt.Fprintf(stderr, "proof preflight failed: %v\n", err)
		return 1
	}
	for _, check := range preflightChecks {
		fmt.Fprintf(stderr, "[proof:%s:%s] %s\n", check.Provider, check.Stage, check.Summary)
	}
	for _, warning := range preflightWarnings {
		fmt.Fprintf(stderr, "[proof:warning] %s\n", warning)
	}

	c := client.New(*apiBaseURL)
	authResult, err := c.SignIn(ctx, types.SignInRequest{
		Email:    *adminEmail,
		Password: *adminPassword,
	})
	if err != nil {
		fmt.Fprintf(stderr, "sign in failed: %v\n", err)
		return 1
	}
	c.SetToken(authResult.Token)

	org, err := ensureOrganization(ctx, c, *orgName, *orgSlug)
	if err != nil {
		fmt.Fprintf(stderr, "ensure organization failed: %v\n", err)
		return 1
	}
	c.SetOrganizationID(org.ID)

	project, err := ensureProject(ctx, c, org.ID, *projectName, *projectSlug)
	if err != nil {
		fmt.Fprintf(stderr, "ensure project failed: %v\n", err)
		return 1
	}
	team, err := ensureTeam(ctx, c, org.ID, project.ID, authResult.Session.ActorID, *teamName, *teamSlug)
	if err != nil {
		fmt.Fprintf(stderr, "ensure team failed: %v\n", err)
		return 1
	}
	service, err := ensureService(ctx, c, org.ID, project.ID, team.ID, *serviceName, *serviceSlug)
	if err != nil {
		fmt.Fprintf(stderr, "ensure service failed: %v\n", err)
		return 1
	}
	environment, err := ensureEnvironment(ctx, c, org.ID, project.ID, *environmentName, *environmentSlug)
	if err != nil {
		fmt.Fprintf(stderr, "ensure environment failed: %v\n", err)
		return 1
	}

	report := liveProofReport{
		Profile:          proofProfileLive,
		EnvironmentClass: normalizeProofEnvironmentClass(*environmentClass),
		ProofQuality:     proofQualityMeaningful,
		VerifiedAt:       time.Now().UTC().Format(time.RFC3339),
		Warnings:         append([]string{}, preflightWarnings...),
		ConfigSummary:    configSummary,
		Checks:           append([]liveProofCheck{}, preflightChecks...),
		SCMKind:          strings.ToLower(strings.TrimSpace(*scmKind)),
		Organization:     org,
		Project:          project,
		Team:             team,
		Service:          service,
		Environment:      environment,
	}
	appendEvidence(&report, fmt.Sprintf("organization=%s", org.Slug), fmt.Sprintf("project=%s", project.Slug), fmt.Sprintf("service=%s", service.Slug), fmt.Sprintf("environment=%s", environment.Slug))

	var repository types.Repository
	var webhookRegistration types.WebhookRegistrationResult
	var scmTestResult types.IntegrationTestResult
	var scmSyncResult types.IntegrationSyncResult

	switch strings.ToLower(strings.TrimSpace(*scmKind)) {
	case "gitlab":
		if strings.TrimSpace(*gitlabGroup) == "" || strings.TrimSpace(*gitlabTokenEnv) == "" || strings.TrimSpace(*gitlabWebhookSecretEnv) == "" {
			fmt.Fprintln(stderr, "gitlab-group, gitlab-token-env, and gitlab-webhook-secret-env are required for gitlab live proof")
			return 1
		}
		integration, err := ensureIntegration(ctx, c, types.CreateIntegrationRequest{
			OrganizationID: org.ID,
			Kind:           "gitlab",
			Name:           "Live Proof GitLab",
			InstanceKey:    "live-proof-gitlab",
			ScopeType:      "group",
			ScopeName:      *gitlabGroup,
			Mode:           "advisory",
			AuthStrategy:   "token",
		}, types.UpdateIntegrationRequest{
			Name:                    stringPtr("Live Proof GitLab"),
			ScopeType:               stringPtr("group"),
			ScopeName:               stringPtr(*gitlabGroup),
			Mode:                    stringPtr("advisory"),
			AuthStrategy:            stringPtr("token"),
			Enabled:                 boolPtr(true),
			ControlEnabled:          boolPtr(false),
			ScheduleEnabled:         boolPtr(true),
			ScheduleIntervalSeconds: intPtr(300),
			SyncStaleAfterSeconds:   intPtr(900),
			Metadata: types.Metadata{
				"api_base_url":       *gitlabBaseURL,
				"group":              *gitlabGroup,
				"access_token_env":   *gitlabTokenEnv,
				"webhook_secret_env": *gitlabWebhookSecretEnv,
			},
		})
		if err != nil {
			fmt.Fprintf(stderr, "ensure gitlab integration failed: %v\n", err)
			return 1
		}
		report.GitLabIntegration = &integration
		appendCheck(&report, "gitlab", "integration_configured", checkStatusPassed, "GitLab live-proof integration configured", []string{
			"scope=" + strings.TrimSpace(*gitlabGroup),
			"instance_key=" + integration.InstanceKey,
			"auth_strategy=" + integration.AuthStrategy,
		}, "")

		scmTestResult, err = c.TestIntegration(ctx, integration.ID)
		if err != nil {
			fmt.Fprintf(stderr, "gitlab connection test failed: %v\n", err)
			return 1
		}
		recordIntegrationRunCheck(&report, "gitlab", "connection_test", scmTestResult.Run, scmTestResult.Run.Details, "verify the GitLab token scope, base URL, and group path")
		webhookRegistration, err = c.SyncWebhookRegistration(ctx, integration.ID)
		if err != nil {
			fmt.Fprintf(stderr, "gitlab webhook registration failed: %v\n", err)
			return 1
		}
		recordWebhookCheck(&report, "gitlab", webhookRegistration, "verify the GitLab group webhook permissions and callback reachability")
		scmSyncResult, err = c.SyncIntegration(ctx, integration.ID)
		if err != nil {
			fmt.Fprintf(stderr, "gitlab sync failed: %v\n", err)
			return 1
		}
		recordIntegrationRunCheck(&report, "gitlab", "discovery_sync", scmSyncResult.Run, scmSyncResult.Run.Details, "verify GitLab project discovery and token access")
		repository, err = ensureRepositoryMapping(ctx, c, integration.ID, project.ID, service.ID, environment.ID)
		if err != nil {
			fmt.Fprintf(stderr, "ensure gitlab repository mapping failed: %v\n", err)
			return 1
		}
		appendEvidence(&report,
			"gitlab_scope="+strings.TrimSpace(*gitlabGroup),
			"gitlab_webhook_status="+webhookRegistration.Registration.Status,
			"gitlab_repository="+repository.Name,
		)
		recordRepositoryMappingCheck(&report, "gitlab", repository)
	case "github":
		if strings.TrimSpace(*githubOwner) == "" || strings.TrimSpace(*githubAppID) == "" || strings.TrimSpace(*githubAppSlug) == "" || strings.TrimSpace(*githubPrivateKeyEnv) == "" || strings.TrimSpace(*githubWebhookSecretEnv) == "" {
			fmt.Fprintln(stderr, "github-owner, github-app-id, github-app-slug, github-private-key-env, and github-webhook-secret-env are required for github live proof")
			return 1
		}
		integration, err := ensureIntegration(ctx, c, types.CreateIntegrationRequest{
			OrganizationID: org.ID,
			Kind:           "github",
			Name:           "Live Proof GitHub",
			InstanceKey:    "live-proof-github",
			ScopeType:      "organization",
			ScopeName:      *githubOwner,
			Mode:           "advisory",
			AuthStrategy:   "github_app",
		}, types.UpdateIntegrationRequest{
			Name:           stringPtr("Live Proof GitHub"),
			ScopeType:      stringPtr("organization"),
			ScopeName:      stringPtr(*githubOwner),
			Mode:           stringPtr("advisory"),
			AuthStrategy:   stringPtr("github_app"),
			Enabled:        boolPtr(false),
			ControlEnabled: boolPtr(false),
			Metadata: types.Metadata{
				"api_base_url":       *githubBaseURL,
				"web_base_url":       *githubWebBaseURL,
				"owner":              *githubOwner,
				"app_id":             *githubAppID,
				"app_slug":           *githubAppSlug,
				"private_key_env":    *githubPrivateKeyEnv,
				"webhook_secret_env": *githubWebhookSecretEnv,
			},
		})
		if err != nil {
			fmt.Fprintf(stderr, "ensure github integration failed: %v\n", err)
			return 1
		}
		appendCheck(&report, "github", "integration_configured", checkStatusPassed, "GitHub live-proof integration configured", []string{
			"owner=" + strings.TrimSpace(*githubOwner),
			"instance_key=" + integration.InstanceKey,
			"auth_strategy=" + integration.AuthStrategy,
		}, "")

		onboardingStart, err := c.StartGitHubOnboarding(ctx, integration.ID)
		if err != nil {
			fmt.Fprintf(stderr, "start github onboarding failed: %v\n", err)
			return 1
		}
		report.GitHubOnboardingStart = &onboardingStart
		appendCheck(&report, "github", "onboarding_start", checkStatusPassed, "GitHub App onboarding authorize URL generated", []string{
			"authorize_url_host=" + hostFromURL(onboardingStart.AuthorizeURL),
			"callback_url_host=" + hostFromURL(onboardingStart.CallbackURL),
			"expires_at=" + onboardingStart.ExpiresAt,
		}, "complete the installation against the intended GitHub owner and record the installation id")

		installationID := strings.TrimSpace(*githubInstallationID)
		if installationID == "" {
			installationID = strings.TrimSpace(fmt.Sprint(integration.Metadata["installation_id"]))
		}
		if installationID == "" {
			fmt.Fprintf(stderr, "github onboarding authorize URL: %s\n", onboardingStart.AuthorizeURL)
			fmt.Fprintln(stderr, "github-installation-id is required to complete the hosted github app onboarding flow non-interactively")
			return 1
		}

		completedIntegration, err := completeGitHubOnboarding(ctx, *apiBaseURL, onboardingStart.AuthorizeURL, installationID)
		if err != nil {
			fmt.Fprintf(stderr, "complete github onboarding failed: %v\n", err)
			return 1
		}
		report.GitHubOnboardingCompletion = &completedIntegration
		appendCheck(&report, "github", "onboarding_complete", checkStatusPassed, "GitHub App onboarding callback completed", []string{
			"installation_id=" + installationID,
			"onboarding_status=" + completedIntegration.OnboardingStatus,
		}, "")

		integration, err = c.UpdateIntegration(ctx, completedIntegration.ID, types.UpdateIntegrationRequest{
			Enabled:        boolPtr(true),
			ControlEnabled: boolPtr(false),
			Mode:           stringPtr("advisory"),
		})
		if err != nil {
			fmt.Fprintf(stderr, "enable github integration failed: %v\n", err)
			return 1
		}
		report.GitHubIntegration = &integration

		scmTestResult, err = c.TestIntegration(ctx, integration.ID)
		if err != nil {
			fmt.Fprintf(stderr, "github connection test failed: %v\n", err)
			return 1
		}
		recordIntegrationRunCheck(&report, "github", "connection_test", scmTestResult.Run, scmTestResult.Run.Details, "verify GitHub App permissions, installation id, and owner scope")
		webhookRegistration, err = c.SyncWebhookRegistration(ctx, integration.ID)
		if err != nil {
			fmt.Fprintf(stderr, "github webhook registration failed: %v\n", err)
			return 1
		}
		recordWebhookCheck(&report, "github", webhookRegistration, "verify GitHub organization webhook permissions and callback reachability")
		scmSyncResult, err = c.SyncIntegration(ctx, integration.ID)
		if err != nil {
			fmt.Fprintf(stderr, "github sync failed: %v\n", err)
			return 1
		}
		recordIntegrationRunCheck(&report, "github", "discovery_sync", scmSyncResult.Run, scmSyncResult.Run.Details, "verify GitHub repository discovery and installation-token access")
		repository, err = ensureRepositoryMapping(ctx, c, integration.ID, project.ID, service.ID, environment.ID)
		if err != nil {
			fmt.Fprintf(stderr, "ensure github repository mapping failed: %v\n", err)
			return 1
		}
		appendEvidence(&report,
			"github_owner="+strings.TrimSpace(*githubOwner),
			"github_webhook_status="+webhookRegistration.Registration.Status,
			"github_repository="+repository.Name,
		)
		recordRepositoryMappingCheck(&report, "github", repository)
	default:
		fmt.Fprintf(stderr, "unsupported scm-kind %q\n", *scmKind)
		return 1
	}

	kubernetesMetadata := types.Metadata{
		"api_base_url":    *kubernetesBaseURL,
		"namespace":       *kubernetesNamespace,
		"deployment_name": *kubernetesDeployment,
	}
	if strings.TrimSpace(*kubernetesStatusPath) != "" {
		kubernetesMetadata["status_path"] = *kubernetesStatusPath
	}
	if strings.TrimSpace(*kubernetesTokenEnv) != "" {
		kubernetesMetadata["bearer_token_env"] = *kubernetesTokenEnv
	}

	kubernetesAuthStrategy := "none"
	if strings.TrimSpace(*kubernetesTokenEnv) != "" {
		kubernetesAuthStrategy = "bearer_env"
	}
	kubernetesIntegration, err := ensureIntegration(ctx, c, types.CreateIntegrationRequest{
		OrganizationID: org.ID,
		Kind:           "kubernetes",
		Name:           "Live Proof Kubernetes",
		InstanceKey:    "live-proof-kubernetes",
		ScopeType:      "cluster",
		ScopeName:      *environmentSlug,
		Mode:           "advisory",
		AuthStrategy:   kubernetesAuthStrategy,
	}, types.UpdateIntegrationRequest{
		Name:                    stringPtr("Live Proof Kubernetes"),
		ScopeType:               stringPtr("cluster"),
		ScopeName:               stringPtr(*environmentSlug),
		Mode:                    stringPtr("advisory"),
		AuthStrategy:            stringPtr(kubernetesAuthStrategy),
		Enabled:                 boolPtr(true),
		ControlEnabled:          boolPtr(false),
		ScheduleEnabled:         boolPtr(true),
		ScheduleIntervalSeconds: intPtr(300),
		SyncStaleAfterSeconds:   intPtr(900),
		Metadata:                kubernetesMetadata,
	})
	if err != nil {
		fmt.Fprintf(stderr, "ensure kubernetes integration failed: %v\n", err)
		return 1
	}
	report.KubernetesIntegration = kubernetesIntegration
	appendCheck(&report, "kubernetes", "integration_configured", checkStatusPassed, "Kubernetes live-proof integration configured", []string{
		"namespace=" + strings.TrimSpace(*kubernetesNamespace),
		"deployment=" + strings.TrimSpace(*kubernetesDeployment),
		"auth_strategy=" + kubernetesIntegration.AuthStrategy,
	}, "")

	report.KubernetesTestResult, err = c.TestIntegration(ctx, kubernetesIntegration.ID)
	if err != nil {
		fmt.Fprintf(stderr, "kubernetes connection test failed: %v\n", err)
		return 1
	}
	recordIntegrationRunCheck(&report, "kubernetes", "connection_test", report.KubernetesTestResult.Run, report.KubernetesTestResult.Run.Details, "verify cluster reachability, auth, and any custom status path configuration")
	report.KubernetesSyncResult, err = c.SyncIntegration(ctx, kubernetesIntegration.ID)
	if err != nil {
		fmt.Fprintf(stderr, "kubernetes sync failed: %v\n", err)
		return 1
	}
	recordIntegrationRunCheck(&report, "kubernetes", "discovery_sync", report.KubernetesSyncResult.Run, report.KubernetesSyncResult.Run.Details, "verify namespace visibility and deployment discovery in the target cluster")
	report.KubernetesResource, err = ensureDiscoveredResourceMapping(ctx, c, kubernetesIntegration.ID, "kubernetes_workload", project.ID, service.ID, environment.ID, repository.ID)
	if err != nil {
		fmt.Fprintf(stderr, "ensure kubernetes resource mapping failed: %v\n", err)
		return 1
	}
	recordDiscoveredResourceCheck(&report, "kubernetes", report.KubernetesResource)
	appendEvidence(&report,
		"kubernetes_namespace="+strings.TrimSpace(*kubernetesNamespace),
		"kubernetes_deployment="+strings.TrimSpace(*kubernetesDeployment),
		"kubernetes_resource="+report.KubernetesResource.Name,
	)

	prometheusMetadata := types.Metadata{
		"api_base_url":   *prometheusBaseURL,
		"window_seconds": *prometheusWindowSeconds,
		"step_seconds":   *prometheusStepSeconds,
		"queries": []types.Metadata{
			{
				"name":           *prometheusQueryName,
				"query":          *prometheusQuery,
				"category":       "technical",
				"threshold":      *prometheusThreshold,
				"comparator":     *prometheusComparator,
				"unit":           *prometheusUnit,
				"severity":       *prometheusSeverity,
				"service_id":     service.ID,
				"environment_id": environment.ID,
				"resource_name":  service.Slug,
			},
		},
	}
	if strings.TrimSpace(*prometheusTokenEnv) != "" {
		prometheusMetadata["bearer_token_env"] = *prometheusTokenEnv
	}
	prometheusAuthStrategy := "none"
	if strings.TrimSpace(*prometheusTokenEnv) != "" {
		prometheusAuthStrategy = "bearer_env"
	}
	prometheusIntegration, err := ensureIntegration(ctx, c, types.CreateIntegrationRequest{
		OrganizationID: org.ID,
		Kind:           "prometheus",
		Name:           "Live Proof Prometheus",
		InstanceKey:    "live-proof-prometheus",
		ScopeType:      "environment",
		ScopeName:      environment.Slug,
		Mode:           "advisory",
		AuthStrategy:   prometheusAuthStrategy,
	}, types.UpdateIntegrationRequest{
		Name:                    stringPtr("Live Proof Prometheus"),
		ScopeType:               stringPtr("environment"),
		ScopeName:               stringPtr(environment.Slug),
		Mode:                    stringPtr("advisory"),
		AuthStrategy:            stringPtr(prometheusAuthStrategy),
		Enabled:                 boolPtr(true),
		ControlEnabled:          boolPtr(false),
		ScheduleEnabled:         boolPtr(true),
		ScheduleIntervalSeconds: intPtr(300),
		SyncStaleAfterSeconds:   intPtr(900),
		Metadata:                prometheusMetadata,
	})
	if err != nil {
		fmt.Fprintf(stderr, "ensure prometheus integration failed: %v\n", err)
		return 1
	}
	report.PrometheusIntegration = prometheusIntegration
	appendCheck(&report, "prometheus", "integration_configured", checkStatusPassed, "Prometheus live-proof integration configured", []string{
		"query_name=" + strings.TrimSpace(*prometheusQueryName),
		"window_seconds=" + fmt.Sprintf("%d", *prometheusWindowSeconds),
		"step_seconds=" + fmt.Sprintf("%d", *prometheusStepSeconds),
	}, "")

	report.PrometheusTestResult, err = c.TestIntegration(ctx, prometheusIntegration.ID)
	if err != nil {
		fmt.Fprintf(stderr, "prometheus connection test failed: %v\n", err)
		return 1
	}
	recordIntegrationRunCheck(&report, "prometheus", "connection_test", report.PrometheusTestResult.Run, report.PrometheusTestResult.Run.Details, "verify the Prometheus base URL, auth header, and API path")
	report.PrometheusSyncResult, err = c.SyncIntegration(ctx, prometheusIntegration.ID)
	if err != nil {
		fmt.Fprintf(stderr, "prometheus sync failed: %v\n", err)
		return 1
	}
	recordIntegrationRunCheck(&report, "prometheus", "signal_sync", report.PrometheusSyncResult.Run, report.PrometheusSyncResult.Run.Details, "verify the query, range window, and signal freshness in the target Prometheus environment")
	report.PrometheusResource, err = ensureDiscoveredResourceMapping(ctx, c, prometheusIntegration.ID, "prometheus_signal_target", project.ID, service.ID, environment.ID, repository.ID)
	if err != nil {
		fmt.Fprintf(stderr, "ensure prometheus resource mapping failed: %v\n", err)
		return 1
	}
	recordDiscoveredResourceCheck(&report, "prometheus", report.PrometheusResource)
	appendEvidence(&report,
		"prometheus_query_name="+strings.TrimSpace(*prometheusQueryName),
		"prometheus_resource="+report.PrometheusResource.Name,
	)

	report.SCMWebhookRegistration = webhookRegistration
	report.SCMTestResult = scmTestResult
	report.SCMSyncResult = scmSyncResult
	report.Repository = repository
	report.CoverageSummary, err = c.IntegrationCoverageSummary(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "coverage summary failed: %v\n", err)
		return 1
	}
	recordCoverageCheck(&report)
	if len(report.Warnings) > 0 {
		report.ProofQuality = proofQualityMeaningfulWarn
	}
	if err := validateLiveProofReport(report); err != nil {
		fmt.Fprintf(stderr, "invalid proof report: %v\n", err)
		return 1
	}

	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "marshal report failed: %v\n", err)
		return 1
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(stderr, "[proof:warning] %s\n", warning)
	}
	fmt.Fprintf(stderr, "[proof] environment_class=%s proof_quality=%s checks=%d warnings=%d\n", report.EnvironmentClass, report.ProofQuality, len(report.Checks), len(report.Warnings))
	if strings.TrimSpace(*reportPath) != "" {
		if err := os.WriteFile(*reportPath, append(body, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "write report failed: %v\n", err)
			return 1
		}
	}
	_, _ = stdout.Write(append(body, '\n'))
	return 0
}

func readLiveProofReport(path string) (liveProofReport, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return liveProofReport{}, err
	}
	var report liveProofReport
	if err := json.Unmarshal(body, &report); err != nil {
		return liveProofReport{}, err
	}
	return report, nil
}

func validateLiveProofReport(report liveProofReport) error {
	if strings.TrimSpace(report.Profile) != proofProfileLive {
		return fmt.Errorf("profile must be live")
	}
	if normalizeProofEnvironmentClass(report.EnvironmentClass) == "" {
		return fmt.Errorf("environment_class must be one of %s, %s, or %s", proofEnvironmentHostedLike, proofEnvironmentCustomer, proofEnvironmentHostedSaaS)
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(report.VerifiedAt)); err != nil {
		return fmt.Errorf("verified_at must be rfc3339: %w", err)
	}
	switch strings.TrimSpace(report.ProofQuality) {
	case proofQualityMeaningful, proofQualityMeaningfulWarn:
	default:
		return fmt.Errorf("proof_quality must be %s or %s", proofQualityMeaningful, proofQualityMeaningfulWarn)
	}
	if len(report.Checks) == 0 {
		return fmt.Errorf("checks are required")
	}
	if strings.TrimSpace(report.ConfigSummary.APIBaseURL.URL) == "" || strings.TrimSpace(report.ConfigSummary.APIBaseURL.EndpointClass) == "" {
		return fmt.Errorf("config_summary.api_base_url is required")
	}
	switch strings.TrimSpace(report.SCMKind) {
	case "gitlab":
		if report.GitLabIntegration == nil || strings.TrimSpace(report.GitLabIntegration.ID) == "" {
			return fmt.Errorf("gitlab report requires gitlab_integration")
		}
	case "github":
		if report.GitHubIntegration == nil || strings.TrimSpace(report.GitHubIntegration.ID) == "" {
			return fmt.Errorf("github report requires github_integration")
		}
		if report.GitHubOnboardingStart == nil || strings.TrimSpace(report.GitHubOnboardingStart.AuthorizeURL) == "" {
			return fmt.Errorf("github report requires github_onboarding_start evidence")
		}
		if report.GitHubOnboardingCompletion == nil || strings.TrimSpace(report.GitHubOnboardingCompletion.ID) == "" {
			return fmt.Errorf("github report requires github_onboarding_completion evidence")
		}
	default:
		return fmt.Errorf("unsupported scm_kind %q", report.SCMKind)
	}
	if strings.TrimSpace(report.Organization.ID) == "" || strings.TrimSpace(report.Project.ID) == "" || strings.TrimSpace(report.Team.ID) == "" || strings.TrimSpace(report.Service.ID) == "" || strings.TrimSpace(report.Environment.ID) == "" {
		return fmt.Errorf("organization, project, team, service, and environment evidence are required")
	}
	if strings.TrimSpace(report.SCMWebhookRegistration.Registration.ID) == "" {
		return fmt.Errorf("scm webhook registration evidence is required")
	}
	if strings.TrimSpace(report.SCMTestResult.Integration.ID) == "" || strings.TrimSpace(report.SCMTestResult.Run.ID) == "" {
		return fmt.Errorf("scm test result evidence is required")
	}
	if strings.TrimSpace(report.SCMSyncResult.Integration.ID) == "" || strings.TrimSpace(report.SCMSyncResult.Run.ID) == "" {
		return fmt.Errorf("scm sync result evidence is required")
	}
	if strings.TrimSpace(report.Repository.ID) == "" || strings.TrimSpace(report.Repository.Provider) == "" {
		return fmt.Errorf("repository evidence is required")
	}
	if strings.TrimSpace(report.KubernetesIntegration.ID) == "" || strings.TrimSpace(report.KubernetesTestResult.Run.ID) == "" || strings.TrimSpace(report.KubernetesSyncResult.Run.ID) == "" || strings.TrimSpace(report.KubernetesResource.ID) == "" {
		return fmt.Errorf("kubernetes integration, test, sync, and resource evidence are required")
	}
	if strings.TrimSpace(report.PrometheusIntegration.ID) == "" || strings.TrimSpace(report.PrometheusTestResult.Run.ID) == "" || strings.TrimSpace(report.PrometheusSyncResult.Run.ID) == "" || strings.TrimSpace(report.PrometheusResource.ID) == "" {
		return fmt.Errorf("prometheus integration, test, sync, and resource evidence are required")
	}
	if len(report.EvidenceSummary) == 0 {
		return fmt.Errorf("evidence_summary is required")
	}
	return nil
}

func completeGitHubOnboarding(ctx context.Context, apiBaseURL, authorizeURL, installationID string) (types.Integration, error) {
	parsed, err := url.Parse(authorizeURL)
	if err != nil {
		return types.Integration{}, err
	}
	state := strings.TrimSpace(parsed.Query().Get("state"))
	if state == "" {
		return types.Integration{}, errors.New("github onboarding authorize url did not include state")
	}
	callbackURL := strings.TrimRight(apiBaseURL, "/") + "/api/v1/integrations/github/callback?state=" + url.QueryEscape(state) + "&installation_id=" + url.QueryEscape(installationID) + "&setup_action=install"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, callbackURL, nil)
	if err != nil {
		return types.Integration{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return types.Integration{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.Integration{}, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return types.Integration{}, fmt.Errorf("github callback failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload types.ItemResponse[types.Integration]
	if err := json.Unmarshal(body, &payload); err != nil {
		return types.Integration{}, err
	}
	return payload.Data, nil
}

func ensureOrganization(ctx context.Context, c *client.Client, name, slug string) (types.Organization, error) {
	items, err := c.ListOrganizations(ctx)
	if err != nil {
		return types.Organization{}, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Slug), strings.TrimSpace(slug)) {
			return item, nil
		}
	}
	return c.CreateOrganization(ctx, types.CreateOrganizationRequest{Name: name, Slug: slug, Tier: "enterprise"})
}

func ensureProject(ctx context.Context, c *client.Client, organizationID, name, slug string) (types.Project, error) {
	items, err := c.ListProjects(ctx)
	if err != nil {
		return types.Project{}, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Slug), strings.TrimSpace(slug)) {
			return item, nil
		}
	}
	return c.CreateProject(ctx, types.CreateProjectRequest{
		OrganizationID: organizationID,
		Name:           name,
		Slug:           slug,
		AdoptionMode:   "advisory",
		Description:    "Live proof project for hosted SCM and runtime integrations.",
	})
}

func ensureTeam(ctx context.Context, c *client.Client, organizationID, projectID, ownerUserID, name, slug string) (types.Team, error) {
	items, err := c.ListTeams(ctx)
	if err != nil {
		return types.Team{}, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Slug), strings.TrimSpace(slug)) {
			return item, nil
		}
	}
	ownerIDs := []string{}
	if strings.TrimSpace(ownerUserID) != "" {
		ownerIDs = append(ownerIDs, ownerUserID)
	}
	return c.CreateTeam(ctx, types.CreateTeamRequest{
		OrganizationID: organizationID,
		ProjectID:      projectID,
		Name:           name,
		Slug:           slug,
		OwnerUserIDs:   ownerIDs,
	})
}

func ensureService(ctx context.Context, c *client.Client, organizationID, projectID, teamID, name, slug string) (types.Service, error) {
	items, err := c.ListServices(ctx)
	if err != nil {
		return types.Service{}, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Slug), strings.TrimSpace(slug)) {
			return item, nil
		}
	}
	return c.CreateService(ctx, types.CreateServiceRequest{
		OrganizationID:   organizationID,
		ProjectID:        projectID,
		TeamID:           teamID,
		Name:             name,
		Slug:             slug,
		Criticality:      "high",
		CustomerFacing:   true,
		HasSLO:           true,
		HasObservability: true,
		Description:      "Live proof service target.",
	})
}

func ensureEnvironment(ctx context.Context, c *client.Client, organizationID, projectID, name, slug string) (types.Environment, error) {
	items, err := c.ListEnvironments(ctx)
	if err != nil {
		return types.Environment{}, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Slug), strings.TrimSpace(slug)) {
			return item, nil
		}
	}
	return c.CreateEnvironment(ctx, types.CreateEnvironmentRequest{
		OrganizationID: organizationID,
		ProjectID:      projectID,
		Name:           name,
		Slug:           slug,
		Type:           "production",
		Region:         "customer-like",
		Production:     true,
		ComplianceZone: "proof",
	})
}

func ensureIntegration(ctx context.Context, c *client.Client, createReq types.CreateIntegrationRequest, updateReq types.UpdateIntegrationRequest) (types.Integration, error) {
	query := fmt.Sprintf("kind=%s&instance_key=%s", createReq.Kind, createReq.InstanceKey)
	items, err := c.ListIntegrationsWithQuery(ctx, query)
	if err != nil {
		return types.Integration{}, err
	}
	var integration types.Integration
	if len(items) == 0 {
		integration, err = c.CreateIntegration(ctx, createReq)
		if err != nil {
			return types.Integration{}, err
		}
	} else {
		integration = items[0]
	}
	return c.UpdateIntegration(ctx, integration.ID, updateReq)
}

func ensureRepositoryMapping(ctx context.Context, c *client.Client, integrationID, projectID, serviceID, environmentID string) (types.Repository, error) {
	items, err := c.ListRepositories(ctx, "source_integration_id="+integrationID)
	if err != nil {
		return types.Repository{}, err
	}
	if len(items) == 0 {
		return types.Repository{}, errors.New("expected scm discovery to return at least one repository")
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	item := items[0]
	return c.UpdateRepository(ctx, item.ID, types.UpdateRepositoryRequest{
		ProjectID:     stringPtr(projectID),
		ServiceID:     stringPtr(serviceID),
		EnvironmentID: stringPtr(environmentID),
		Status:        stringPtr("mapped"),
	})
}

func ensureDiscoveredResourceMapping(ctx context.Context, c *client.Client, integrationID, resourceType, projectID, serviceID, environmentID, repositoryID string) (types.DiscoveredResource, error) {
	items, err := c.ListDiscoveredResources(ctx, "integration_id="+integrationID+"&resource_type="+resourceType)
	if err != nil {
		return types.DiscoveredResource{}, err
	}
	if len(items) == 0 {
		return types.DiscoveredResource{}, fmt.Errorf("expected discovered resource for %s", resourceType)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	item := items[0]
	return c.UpdateDiscoveredResource(ctx, item.ID, types.UpdateDiscoveredResourceRequest{
		ProjectID:     stringPtr(projectID),
		ServiceID:     stringPtr(serviceID),
		EnvironmentID: stringPtr(environmentID),
		RepositoryID:  stringPtr(repositoryID),
		Status:        stringPtr("mapped"),
	})
}

func buildLiveProofConfigSummary(input liveProofInput) liveProofConfigSummary {
	scmKind := strings.ToLower(strings.TrimSpace(input.SCMKind))
	summary := liveProofConfigSummary{
		APIBaseURL: summarizeEndpoint(input.APIBaseURL),
		Kubernetes: liveProofProviderConfigSummary{
			Kind:         "kubernetes",
			ScopeType:    "cluster",
			ScopeName:    strings.TrimSpace(input.KubernetesNamespace),
			AuthStrategy: authStrategyForOptionalEnv(input.KubernetesTokenEnv),
			Endpoint:     summarizeEndpoint(input.KubernetesBaseURL),
			SecretEnvs:   compactSecretSummaries(summarizeSecretEnv(input.KubernetesTokenEnv, "kubernetes_api_auth")),
			ResourceHint: strings.TrimSpace(input.KubernetesDeployment),
		},
		Prometheus: liveProofProviderConfigSummary{
			Kind:            "prometheus",
			ScopeType:       "environment",
			ScopeName:       "query_range",
			AuthStrategy:    authStrategyForOptionalEnv(input.PrometheusTokenEnv),
			Endpoint:        summarizeEndpoint(input.PrometheusBaseURL),
			SecretEnvs:      compactSecretSummaries(summarizeSecretEnv(input.PrometheusTokenEnv, "prometheus_api_auth")),
			ResourceHint:    strings.TrimSpace(input.PrometheusQuery),
			QueryName:       strings.TrimSpace(input.PrometheusQueryName),
			QueryWindowSecs: input.PrometheusWindowSecs,
			QueryStepSecs:   input.PrometheusStepSecs,
		},
	}
	switch scmKind {
	case "gitlab":
		summary.SCM = liveProofProviderConfigSummary{
			Kind:         "gitlab",
			ScopeType:    "group",
			ScopeName:    strings.TrimSpace(input.GitLabGroup),
			AuthStrategy: "token",
			Endpoint:     summarizeEndpoint(input.GitLabBaseURL),
			SecretEnvs: compactSecretSummaries(
				summarizeSecretEnv(input.GitLabTokenEnv, "gitlab_access_token"),
				summarizeSecretEnv(input.GitLabWebhookSecretEnv, "gitlab_webhook_secret"),
			),
			RepositoryHint: strings.TrimSpace(input.GitLabGroup),
		}
	case "github":
		webEndpoint := summarizeEndpoint(input.GitHubWebBaseURL)
		summary.SCM = liveProofProviderConfigSummary{
			Kind:         "github",
			ScopeType:    "organization",
			ScopeName:    strings.TrimSpace(input.GitHubOwner),
			AuthStrategy: "github_app",
			Endpoint:     summarizeEndpoint(input.GitHubBaseURL),
			WebEndpoint:  &webEndpoint,
			SecretEnvs: compactSecretSummaries(
				summarizeSecretEnv(input.GitHubPrivateKeyEnv, "github_app_private_key"),
				summarizeSecretEnv(input.GitHubWebhookSecretEnv, "github_webhook_secret"),
			),
			RepositoryHint: strings.TrimSpace(input.GitHubOwner),
		}
	}
	return summary
}

func validateLiveProofInput(input liveProofInput) ([]liveProofCheck, []string, error) {
	checks := make([]liveProofCheck, 0, 12)
	warnings := make([]string, 0, 4)

	environmentClass := normalizeProofEnvironmentClass(input.EnvironmentClass)
	if environmentClass == "" {
		return nil, nil, fmt.Errorf("environment-class must be one of %s, %s, or %s", proofEnvironmentHostedLike, proofEnvironmentCustomer, proofEnvironmentHostedSaaS)
	}
	if _, err := parseAndValidateURL(input.APIBaseURL, "api-base-url"); err != nil {
		return nil, nil, err
	}
	checks = append(checks, liveProofCheck{
		Provider: "control_plane",
		Stage:    "config_validation",
		Status:   checkStatusPassed,
		Summary:  "Control-plane API base URL validated",
		Details:  []string{"environment_class=" + environmentClass, "api_base_url=" + strings.TrimSpace(input.APIBaseURL)},
	})

	if strings.TrimSpace(input.KubernetesBaseURL) == "" || strings.TrimSpace(input.KubernetesNamespace) == "" || strings.TrimSpace(input.KubernetesDeployment) == "" {
		return nil, nil, fmt.Errorf("kubernetes-base-url, kubernetes-namespace, and kubernetes-deployment are required")
	}
	if _, err := parseAndValidateURL(input.KubernetesBaseURL, "kubernetes-base-url"); err != nil {
		return nil, nil, err
	}
	if input.KubernetesStatusPath != "" && !strings.HasPrefix(strings.TrimSpace(input.KubernetesStatusPath), "/") {
		return nil, nil, fmt.Errorf("kubernetes-status-path must start with / when provided")
	}
	if err := validateOptionalSecretEnv(input.KubernetesTokenEnv, "kubernetes-token-env"); err != nil {
		return nil, nil, err
	}
	checks = append(checks, liveProofCheck{
		Provider: "kubernetes",
		Stage:    "config_validation",
		Status:   checkStatusPassed,
		Summary:  "Kubernetes proof target validated",
		Details:  []string{"namespace=" + strings.TrimSpace(input.KubernetesNamespace), "deployment=" + strings.TrimSpace(input.KubernetesDeployment)},
	})

	if strings.TrimSpace(input.PrometheusBaseURL) == "" || strings.TrimSpace(input.PrometheusQuery) == "" {
		return nil, nil, fmt.Errorf("prometheus-base-url and prometheus-query are required")
	}
	if _, err := parseAndValidateURL(input.PrometheusBaseURL, "prometheus-base-url"); err != nil {
		return nil, nil, err
	}
	if err := validateOptionalSecretEnv(input.PrometheusTokenEnv, "prometheus-token-env"); err != nil {
		return nil, nil, err
	}
	if err := validatePrometheusWindow(input.PrometheusWindowSecs, input.PrometheusStepSecs); err != nil {
		return nil, nil, err
	}
	if !isAllowedComparator(input.PrometheusComparator) {
		return nil, nil, fmt.Errorf("prometheus-comparator must be one of >, >=, <, or <=")
	}
	checks = append(checks, liveProofCheck{
		Provider: "prometheus",
		Stage:    "config_validation",
		Status:   checkStatusPassed,
		Summary:  "Prometheus proof target validated",
		Details: []string{
			"query_name=" + strings.TrimSpace(input.PrometheusQueryName),
			fmt.Sprintf("window_seconds=%d", input.PrometheusWindowSecs),
			fmt.Sprintf("step_seconds=%d", input.PrometheusStepSecs),
		},
	})

	switch strings.ToLower(strings.TrimSpace(input.SCMKind)) {
	case "gitlab":
		if strings.TrimSpace(input.GitLabGroup) == "" || strings.TrimSpace(input.GitLabTokenEnv) == "" || strings.TrimSpace(input.GitLabWebhookSecretEnv) == "" {
			return nil, nil, fmt.Errorf("gitlab-group, gitlab-token-env, and gitlab-webhook-secret-env are required for gitlab live proof")
		}
		if _, err := parseAndValidateURL(input.GitLabBaseURL, "gitlab-base-url"); err != nil {
			return nil, nil, err
		}
		if err := validateRequiredSecretEnv(input.GitLabTokenEnv, "gitlab-token-env"); err != nil {
			return nil, nil, err
		}
		if err := validateRequiredSecretEnv(input.GitLabWebhookSecretEnv, "gitlab-webhook-secret-env"); err != nil {
			return nil, nil, err
		}
		if environmentClass == proofEnvironmentHostedSaaS && summarizeEndpoint(input.GitLabBaseURL).EndpointClass != "public" {
			return nil, nil, fmt.Errorf("gitlab-base-url must be publicly hosted when environment-class is %s", proofEnvironmentHostedSaaS)
		}
		checks = append(checks, liveProofCheck{
			Provider: "gitlab",
			Stage:    "config_validation",
			Status:   checkStatusPassed,
			Summary:  "GitLab live-proof inputs validated",
			Details:  []string{"group=" + strings.TrimSpace(input.GitLabGroup), "api_base_url=" + strings.TrimSpace(input.GitLabBaseURL)},
		})
	case "github":
		if strings.TrimSpace(input.GitHubOwner) == "" || strings.TrimSpace(input.GitHubAppID) == "" || strings.TrimSpace(input.GitHubAppSlug) == "" || strings.TrimSpace(input.GitHubPrivateKeyEnv) == "" || strings.TrimSpace(input.GitHubWebhookSecretEnv) == "" {
			return nil, nil, fmt.Errorf("github-owner, github-app-id, github-app-slug, github-private-key-env, and github-webhook-secret-env are required for github live proof")
		}
		if _, err := parseAndValidateURL(input.GitHubBaseURL, "github-base-url"); err != nil {
			return nil, nil, err
		}
		if _, err := parseAndValidateURL(input.GitHubWebBaseURL, "github-web-base-url"); err != nil {
			return nil, nil, err
		}
		if strings.Contains(strings.TrimSpace(input.GitHubOwner), "/") {
			return nil, nil, fmt.Errorf("github-owner must be an owner or organization name, not a repository path")
		}
		if err := validateRequiredSecretEnv(input.GitHubPrivateKeyEnv, "github-private-key-env"); err != nil {
			return nil, nil, err
		}
		if err := validateRequiredSecretEnv(input.GitHubWebhookSecretEnv, "github-webhook-secret-env"); err != nil {
			return nil, nil, err
		}
		if strings.TrimSpace(input.GitHubInstallationID) == "" {
			warnings = append(warnings, "github-installation-id was not provided; hosted GitHub proof requires installation evidence to complete non-interactively")
		}
		if environmentClass == proofEnvironmentHostedSaaS && summarizeEndpoint(input.GitHubBaseURL).EndpointClass != "public" {
			return nil, nil, fmt.Errorf("github-base-url must be publicly hosted when environment-class is %s", proofEnvironmentHostedSaaS)
		}
		checks = append(checks, liveProofCheck{
			Provider: "github",
			Stage:    "config_validation",
			Status:   checkStatusPassed,
			Summary:  "GitHub live-proof inputs validated",
			Details:  []string{"owner=" + strings.TrimSpace(input.GitHubOwner), "api_base_url=" + strings.TrimSpace(input.GitHubBaseURL), "web_base_url=" + strings.TrimSpace(input.GitHubWebBaseURL)},
		})
	default:
		return nil, nil, fmt.Errorf("unsupported scm-kind %q", input.SCMKind)
	}

	return checks, warnings, nil
}

func parseAndValidateURL(raw, field string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("%s is required", field)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%s must be an absolute http(s) url", field)
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return nil, fmt.Errorf("%s must use http or https", field)
	}
	return parsed, nil
}

func validateRequiredSecretEnv(envName, field string) error {
	trimmed := strings.TrimSpace(envName)
	if trimmed == "" {
		return fmt.Errorf("%s is required", field)
	}
	if !envVarNamePattern.MatchString(trimmed) {
		return fmt.Errorf("%s must be an uppercase environment variable name", field)
	}
	if strings.TrimSpace(os.Getenv(trimmed)) == "" {
		return fmt.Errorf("%s references env %s, but it is not set", field, trimmed)
	}
	return nil
}

func validateOptionalSecretEnv(envName, field string) error {
	trimmed := strings.TrimSpace(envName)
	if trimmed == "" {
		return nil
	}
	if !envVarNamePattern.MatchString(trimmed) {
		return fmt.Errorf("%s must be an uppercase environment variable name", field)
	}
	if strings.TrimSpace(os.Getenv(trimmed)) == "" {
		return fmt.Errorf("%s references env %s, but it is not set", field, trimmed)
	}
	return nil
}

func validatePrometheusWindow(windowSeconds, stepSeconds int) error {
	if windowSeconds <= 0 {
		return fmt.Errorf("prometheus-window-seconds must be > 0")
	}
	if stepSeconds <= 0 {
		return fmt.Errorf("prometheus-step-seconds must be > 0")
	}
	if stepSeconds > windowSeconds {
		return fmt.Errorf("prometheus-step-seconds must be <= prometheus-window-seconds")
	}
	return nil
}

func isAllowedComparator(value string) bool {
	switch strings.TrimSpace(value) {
	case ">", ">=", "<", "<=":
		return true
	default:
		return false
	}
}

func normalizeProofEnvironmentClass(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case proofEnvironmentHostedLike:
		return proofEnvironmentHostedLike
	case proofEnvironmentCustomer:
		return proofEnvironmentCustomer
	case proofEnvironmentHostedSaaS:
		return proofEnvironmentHostedSaaS
	default:
		return ""
	}
}

func summarizeEndpoint(raw string) liveProofEndpointSummary {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return liveProofEndpointSummary{}
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return liveProofEndpointSummary{URL: trimmed, EndpointClass: "invalid"}
	}
	host := parsed.Hostname()
	return liveProofEndpointSummary{
		URL:           trimmed,
		Host:          host,
		EndpointClass: classifyEndpointHost(host),
	}
}

func summarizeSecretEnv(envName, requiredFor string) liveProofSecretSummary {
	trimmed := strings.TrimSpace(envName)
	return liveProofSecretSummary{
		EnvName:     trimmed,
		Configured:  trimmed != "" && strings.TrimSpace(os.Getenv(trimmed)) != "",
		RequiredFor: requiredFor,
	}
}

func compactSecretSummaries(items ...liveProofSecretSummary) []liveProofSecretSummary {
	filtered := make([]liveProofSecretSummary, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.EnvName) == "" {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func classifyEndpointHost(host string) string {
	trimmed := strings.TrimSpace(strings.ToLower(host))
	if trimmed == "" {
		return "unknown"
	}
	if trimmed == "localhost" || trimmed == "::1" || strings.HasPrefix(trimmed, "127.") {
		return "local"
	}
	if ip := net.ParseIP(trimmed); ip != nil {
		if ip.IsPrivate() {
			return "private"
		}
		return "public"
	}
	if strings.HasSuffix(trimmed, ".local") || strings.HasSuffix(trimmed, ".internal") || strings.HasSuffix(trimmed, ".cluster.local") {
		return "private"
	}
	return "public"
}

func authStrategyForOptionalEnv(envName string) string {
	if strings.TrimSpace(envName) != "" {
		return "bearer_env"
	}
	return "none"
}

func appendCheck(report *liveProofReport, provider, stage, status, summary string, details []string, hint string) {
	report.Checks = append(report.Checks, liveProofCheck{
		Provider: provider,
		Stage:    stage,
		Status:   status,
		Summary:  summary,
		Details:  compactDetails(details),
		Hint:     hint,
	})
}

func appendEvidence(report *liveProofReport, values ...string) {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			report.EvidenceSummary = append(report.EvidenceSummary, trimmed)
		}
	}
}

func addWarning(report *liveProofReport, warning string) {
	trimmed := strings.TrimSpace(warning)
	if trimmed != "" {
		report.Warnings = append(report.Warnings, trimmed)
	}
}

func recordIntegrationRunCheck(report *liveProofReport, provider, stage string, run types.IntegrationSyncRun, details []string, hint string) {
	status := checkStatusPassed
	summary := fmt.Sprintf("%s %s completed with status %s", providerDisplayName(provider), strings.ReplaceAll(stage, "_", " "), strings.TrimSpace(run.Status))
	normalized := strings.ToLower(strings.TrimSpace(run.Status))
	if normalized == "warning" || normalized == "partial" {
		status = checkStatusWarning
		addWarning(report, summary)
	}
	if containsProblemDetail(details) {
		status = checkStatusWarning
		addWarning(report, fmt.Sprintf("%s reported warnings: %s", provider, strings.Join(compactDetails(details), "; ")))
	}
	appendCheck(report, provider, stage, status, summary, append(compactDetails(details), fmt.Sprintf("resource_count=%d", run.ResourceCount)), hint)
}

func recordWebhookCheck(report *liveProofReport, provider string, result types.WebhookRegistrationResult, hint string) {
	status := checkStatusPassed
	if strings.TrimSpace(result.Registration.ExternalHookID) == "" || strings.TrimSpace(result.Registration.Status) != "registered" {
		status = checkStatusWarning
		addWarning(report, fmt.Sprintf("%s webhook registration did not reach a fully registered state", provider))
	}
	appendCheck(report, provider, "webhook_registration", status, fmt.Sprintf("%s webhook registration state captured", providerDisplayName(provider)), append(compactDetails(result.Details),
		"status="+strings.TrimSpace(result.Registration.Status),
		"external_hook_id="+strings.TrimSpace(result.Registration.ExternalHookID),
		"scope="+strings.TrimSpace(result.Registration.ScopeIdentifier),
	), hint)
}

func recordRepositoryMappingCheck(report *liveProofReport, provider string, repository types.Repository) {
	status := checkStatusPassed
	if strings.TrimSpace(repository.Status) != "mapped" {
		status = checkStatusWarning
		addWarning(report, fmt.Sprintf("%s repository %s is not marked mapped", provider, repository.Name))
	}
	appendCheck(report, provider, "repository_mapping", status, fmt.Sprintf("%s repository discovery and mapping captured", providerDisplayName(provider)), []string{
		"repository=" + repository.Name,
		"provider=" + repository.Provider,
		"status=" + repository.Status,
		"source_integration_id=" + repository.SourceIntegrationID,
	}, "confirm the discovered repository is the intended source of change truth")
}

func recordDiscoveredResourceCheck(report *liveProofReport, provider string, resource types.DiscoveredResource) {
	status := checkStatusPassed
	if strings.TrimSpace(resource.Status) != "mapped" {
		status = checkStatusWarning
		addWarning(report, fmt.Sprintf("%s discovered resource %s is not marked mapped", provider, resource.Name))
	}
	appendCheck(report, provider, "resource_mapping", status, fmt.Sprintf("%s discovered resource evidence captured", providerDisplayName(provider)), []string{
		"resource_type=" + resource.ResourceType,
		"name=" + resource.Name,
		"namespace=" + resource.Namespace,
		"status=" + resource.Status,
		"health=" + resource.Health,
	}, "confirm the discovered runtime resource matches the intended service and environment")
}

func recordCoverageCheck(report *liveProofReport) {
	status := checkStatusPassed
	details := []string{
		fmt.Sprintf("repositories=%d", report.CoverageSummary.Repositories),
		fmt.Sprintf("unmapped_repositories=%d", report.CoverageSummary.UnmappedRepositories),
		fmt.Sprintf("discovered_resources=%d", report.CoverageSummary.DiscoveredResources),
		fmt.Sprintf("unmapped_discovered_resources=%d", report.CoverageSummary.UnmappedDiscoveredResources),
	}
	if report.CoverageSummary.UnmappedRepositories > 0 || report.CoverageSummary.UnmappedDiscoveredResources > 0 {
		status = checkStatusWarning
		addWarning(report, "coverage summary still reports unmapped repositories or discovered resources")
	}
	appendCheck(report, "control_plane", "coverage_summary", status, "Coverage summary captured after proof mapping", details, "map any remaining repositories or discovered resources before using this as deployment-readiness evidence")
}

func compactDetails(details []string) []string {
	items := make([]string, 0, len(details))
	seen := map[string]struct{}{}
	for _, detail := range details {
		trimmed := strings.TrimSpace(detail)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	return items
}

func containsProblemDetail(details []string) bool {
	for _, detail := range details {
		trimmed := strings.ToLower(strings.TrimSpace(detail))
		if strings.Contains(trimmed, "warning") || strings.Contains(trimmed, "no sample") || strings.Contains(trimmed, "stale") || strings.Contains(trimmed, "repair") || strings.Contains(trimmed, "manual") {
			return true
		}
	}
	return false
}

func hostFromURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

func providerDisplayName(provider string) string {
	trimmed := strings.TrimSpace(provider)
	if trimmed == "" {
		return "Provider"
	}
	switch strings.ToLower(trimmed) {
	case "github":
		return "GitHub"
	case "gitlab":
		return "GitLab"
	case "kubernetes":
		return "Kubernetes"
	case "prometheus":
		return "Prometheus"
	default:
		return strings.ToUpper(trimmed[:1]) + trimmed[1:]
	}
}

func stringPtr(value string) *string { return &value }
func boolPtr(value bool) *bool       { return &value }
func intPtr(value int) *int          { return &value }

func valueOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func valueOrDefaultInt(key string, fallback int) int {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		var parsed int
		if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

func valueOrDefaultFloat(key string, fallback float64) float64 {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		var parsed float64
		if _, err := fmt.Sscanf(value, "%f", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}
