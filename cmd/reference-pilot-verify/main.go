package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/client"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type verificationReport struct {
	Profile                string                          `json:"profile"`
	VerifiedAt             string                          `json:"verified_at"`
	ProofQuality           string                          `json:"proof_quality"`
	EvidenceSummary        []string                        `json:"evidence_summary,omitempty"`
	Organization           types.Organization              `json:"organization"`
	Project                types.Project                   `json:"project"`
	Team                   types.Team                      `json:"team"`
	Service                types.Service                   `json:"service"`
	Environment            types.Environment               `json:"environment"`
	GitLabIntegration      types.Integration               `json:"gitlab_integration"`
	KubernetesIntegration  types.Integration               `json:"kubernetes_integration"`
	PrometheusIntegration  types.Integration               `json:"prometheus_integration"`
	WebhookRegistration    types.WebhookRegistrationResult `json:"webhook_registration"`
	Repository             types.Repository                `json:"repository"`
	KubernetesResource     types.DiscoveredResource        `json:"kubernetes_resource"`
	PrometheusResource     types.DiscoveredResource        `json:"prometheus_resource"`
	ChangeSet              types.ChangeSet                 `json:"change_set"`
	RolloutPlan            types.RolloutPlan               `json:"rollout_plan"`
	Execution              types.RolloutExecution          `json:"execution"`
	ExecutionDetail        types.RolloutExecutionDetail    `json:"execution_detail"`
	StatusEventCount       int                             `json:"status_event_count"`
	TimelineEventCount     int                             `json:"timeline_event_count"`
	AuditEventCount        int                             `json:"audit_event_count"`
	WorkloadStateUpdatedAt string                          `json:"workload_state_updated_at,omitempty"`
}

const (
	referencePilotProfile      = "reference_pilot"
	referencePilotProofQuality = "meaningful"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("reference-pilot-verify", flag.ExitOnError)
	apiBaseURL := flags.String("api-base-url", valueOrDefault("CCP_REFERENCE_PILOT_API_BASE_URL", "http://127.0.0.1:38080"), "control plane api base url")
	adminEmail := flags.String("admin-email", valueOrDefault("CCP_REFERENCE_PILOT_ADMIN_EMAIL", "admin@changecontrolplane.local"), "pilot admin email")
	adminPassword := flags.String("admin-password", valueOrDefault("CCP_REFERENCE_PILOT_ADMIN_PASSWORD", "ChangeMe123!"), "pilot admin password")
	orgName := flags.String("org-name", valueOrDefault("CCP_REFERENCE_PILOT_ORG_NAME", "Reference Pilot"), "organization name")
	orgSlug := flags.String("org-slug", valueOrDefault("CCP_REFERENCE_PILOT_ORG_SLUG", "reference-pilot"), "organization slug")
	projectName := flags.String("project-name", valueOrDefault("CCP_REFERENCE_PILOT_PROJECT_NAME", "Checkout Platform"), "project name")
	projectSlug := flags.String("project-slug", valueOrDefault("CCP_REFERENCE_PILOT_PROJECT_SLUG", "checkout-platform"), "project slug")
	teamName := flags.String("team-name", valueOrDefault("CCP_REFERENCE_PILOT_TEAM_NAME", "Checkout Team"), "team name")
	teamSlug := flags.String("team-slug", valueOrDefault("CCP_REFERENCE_PILOT_TEAM_SLUG", "checkout-team"), "team slug")
	serviceName := flags.String("service-name", valueOrDefault("CCP_REFERENCE_PILOT_SERVICE_NAME", "Checkout"), "service name")
	serviceSlug := flags.String("service-slug", valueOrDefault("CCP_REFERENCE_PILOT_SERVICE_SLUG", "checkout"), "service slug")
	environmentName := flags.String("environment-name", valueOrDefault("CCP_REFERENCE_PILOT_ENV_NAME", "Pilot"), "environment name")
	environmentSlug := flags.String("environment-slug", valueOrDefault("CCP_REFERENCE_PILOT_ENV_SLUG", "ccp-pilot"), "environment slug")
	gitlabBaseURL := flags.String("gitlab-base-url", valueOrDefault("CCP_REFERENCE_PILOT_GITLAB_BASE_URL", "http://127.0.0.1:39480/api/v4"), "gitlab fixture api base url")
	gitlabGroup := flags.String("gitlab-group", valueOrDefault("CCP_REFERENCE_PILOT_GITLAB_GROUP", "acme"), "gitlab group scope")
	gitlabTokenEnv := flags.String("gitlab-token-env", valueOrDefault("CCP_REFERENCE_PILOT_GITLAB_TOKEN_ENV", "CCP_REFERENCE_PILOT_GITLAB_TOKEN"), "gitlab access token env name")
	gitlabWebhookSecretEnv := flags.String("gitlab-webhook-secret-env", valueOrDefault("CCP_REFERENCE_PILOT_GITLAB_WEBHOOK_SECRET_ENV", "CCP_REFERENCE_PILOT_GITLAB_WEBHOOK_SECRET"), "gitlab webhook secret env name")
	kubernetesBaseURL := flags.String("kubernetes-base-url", valueOrDefault("CCP_REFERENCE_PILOT_KUBE_API_BASE_URL", "http://127.0.0.1:18091"), "kubernetes api proxy base url")
	prometheusBaseURL := flags.String("prometheus-base-url", valueOrDefault("CCP_REFERENCE_PILOT_PROMETHEUS_BASE_URL", "http://127.0.0.1:19090"), "prometheus api base url")
	workloadAdminURL := flags.String("workload-admin-url", valueOrDefault("CCP_REFERENCE_PILOT_WORKLOAD_ADMIN_URL", "http://127.0.0.1:18092/admin/state"), "reference workload admin url")
	reportPath := flags.String("report", "", "optional path to write the proof report JSON")
	validateReportPath := flags.String("validate-report", valueOrDefault("CCP_REFERENCE_PILOT_VALIDATE_REPORT", ""), "validate an existing reference pilot report JSON and print the normalized result")
	if err := flags.Parse(args); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if strings.TrimSpace(*validateReportPath) != "" {
		report, err := readVerificationReport(*validateReportPath)
		if err != nil {
			fmt.Fprintf(stderr, "read reference pilot report failed: %v\n", err)
			return 1
		}
		if err := validateVerificationReport(report); err != nil {
			fmt.Fprintf(stderr, "invalid reference pilot report: %v\n", err)
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

	gitlabIntegration, err := ensureIntegration(ctx, c, types.CreateIntegrationRequest{
		OrganizationID: org.ID,
		Kind:           "gitlab",
		Name:           "Reference Pilot GitLab",
		InstanceKey:    "reference-pilot-gitlab",
		ScopeType:      "group",
		ScopeName:      *gitlabGroup,
		Mode:           "advisory",
		AuthStrategy:   "token",
	}, types.UpdateIntegrationRequest{
		Name:                    stringPtr("Reference Pilot GitLab"),
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

	kubernetesIntegration, err := ensureIntegration(ctx, c, types.CreateIntegrationRequest{
		OrganizationID: org.ID,
		Kind:           "kubernetes",
		Name:           "Reference Pilot Kubernetes",
		InstanceKey:    "reference-pilot-kubernetes",
		ScopeType:      "cluster",
		ScopeName:      "local-k3s",
		Mode:           "advisory",
		AuthStrategy:   "bearer_env",
	}, types.UpdateIntegrationRequest{
		Name:                    stringPtr("Reference Pilot Kubernetes"),
		ScopeType:               stringPtr("cluster"),
		ScopeName:               stringPtr("local-k3s"),
		Mode:                    stringPtr("advisory"),
		AuthStrategy:            stringPtr("bearer_env"),
		Enabled:                 boolPtr(true),
		ControlEnabled:          boolPtr(false),
		ScheduleEnabled:         boolPtr(true),
		ScheduleIntervalSeconds: intPtr(120),
		SyncStaleAfterSeconds:   intPtr(480),
		Metadata: types.Metadata{
			"api_base_url":          *kubernetesBaseURL,
			"namespace":             environment.Slug,
			"deployment_name":       service.Slug,
			"container_name":        service.Slug,
			"rollback_target_image": "ccp-reference-pilot-workload:local",
		},
	})
	if err != nil {
		fmt.Fprintf(stderr, "ensure kubernetes integration failed: %v\n", err)
		return 1
	}

	prometheusIntegration, err := ensureIntegration(ctx, c, types.CreateIntegrationRequest{
		OrganizationID: org.ID,
		Kind:           "prometheus",
		Name:           "Reference Pilot Prometheus",
		InstanceKey:    "reference-pilot-prometheus",
		ScopeType:      "environment",
		ScopeName:      environment.Slug,
		Mode:           "advisory",
		AuthStrategy:   "none",
	}, types.UpdateIntegrationRequest{
		Name:                    stringPtr("Reference Pilot Prometheus"),
		ScopeType:               stringPtr("environment"),
		ScopeName:               stringPtr(environment.Slug),
		Mode:                    stringPtr("advisory"),
		AuthStrategy:            stringPtr("none"),
		Enabled:                 boolPtr(true),
		ControlEnabled:          boolPtr(false),
		ScheduleEnabled:         boolPtr(true),
		ScheduleIntervalSeconds: intPtr(120),
		SyncStaleAfterSeconds:   intPtr(480),
		Metadata: types.Metadata{
			"api_base_url":   *prometheusBaseURL,
			"window_seconds": 120,
			"step_seconds":   15,
			"queries": []types.Metadata{
				{
					"name":           "reference_pilot_request_latency_ms",
					"query":          `reference_pilot_request_latency_ms{service="checkout",environment="pilot"}`,
					"category":       "technical",
					"threshold":      200,
					"comparator":     ">=",
					"unit":           "ms",
					"severity":       "critical",
					"service_id":     service.ID,
					"environment_id": environment.ID,
					"resource_name":  service.Slug,
				},
				{
					"name":           "reference_pilot_error_ratio",
					"query":          `reference_pilot_error_ratio{service="checkout",environment="pilot"}`,
					"category":       "technical",
					"threshold":      0.05,
					"comparator":     ">=",
					"unit":           "ratio",
					"severity":       "critical",
					"service_id":     service.ID,
					"environment_id": environment.ID,
					"resource_name":  service.Slug,
				},
			},
		},
	})
	if err != nil {
		fmt.Fprintf(stderr, "ensure prometheus integration failed: %v\n", err)
		return 1
	}

	if _, err := c.TestIntegration(ctx, gitlabIntegration.ID); err != nil {
		fmt.Fprintf(stderr, "gitlab test failed: %v\n", err)
		return 1
	}
	webhookRegistration, err := c.SyncWebhookRegistration(ctx, gitlabIntegration.ID)
	if err != nil {
		fmt.Fprintf(stderr, "gitlab webhook registration failed: %v\n", err)
		return 1
	}
	if _, err := c.TestIntegration(ctx, kubernetesIntegration.ID); err != nil {
		fmt.Fprintf(stderr, "kubernetes test failed: %v\n", err)
		return 1
	}
	if _, err := c.TestIntegration(ctx, prometheusIntegration.ID); err != nil {
		fmt.Fprintf(stderr, "prometheus test failed: %v\n", err)
		return 1
	}

	if _, err := c.SyncIntegration(ctx, gitlabIntegration.ID); err != nil {
		fmt.Fprintf(stderr, "gitlab sync failed: %v\n", err)
		return 1
	}
	repository, err := ensureRepositoryMapping(ctx, c, gitlabIntegration.ID, project.ID, service.ID, environment.ID)
	if err != nil {
		fmt.Fprintf(stderr, "ensure repository mapping failed: %v\n", err)
		return 1
	}
	if _, err := c.SyncIntegration(ctx, kubernetesIntegration.ID); err != nil {
		fmt.Fprintf(stderr, "kubernetes sync failed: %v\n", err)
		return 1
	}
	kubernetesResource, err := ensureDiscoveredResourceMapping(ctx, c, kubernetesIntegration.ID, "kubernetes_workload", project.ID, service.ID, environment.ID, repository.ID)
	if err != nil {
		fmt.Fprintf(stderr, "ensure kubernetes discovered resource mapping failed: %v\n", err)
		return 1
	}
	if _, err := c.SyncIntegration(ctx, prometheusIntegration.ID); err != nil {
		fmt.Fprintf(stderr, "prometheus sync failed: %v\n", err)
		return 1
	}
	prometheusResource, err := ensureDiscoveredResourceMapping(ctx, c, prometheusIntegration.ID, "prometheus_signal_target", project.ID, service.ID, environment.ID, repository.ID)
	if err != nil {
		fmt.Fprintf(stderr, "ensure prometheus discovered resource mapping failed: %v\n", err)
		return 1
	}

	workloadUpdatedAt, err := updateReferenceWorkload(*workloadAdminURL, workloadStateRequest{
		Version:    "v2",
		LatencyMS:  480,
		ErrorRatio: 0.12,
	})
	if err != nil {
		fmt.Fprintf(stderr, "update workload state failed: %v\n", err)
		return 1
	}

	changeTitle := fmt.Sprintf("REFPILOT-%d degrade checkout canary latency", time.Now().Unix())
	if err := postGitLabMergeRequestWebhook(*apiBaseURL, authResult.Token, org.ID, gitlabIntegration.ID, os.Getenv(*gitlabWebhookSecretEnv), changeTitle); err != nil {
		fmt.Fprintf(stderr, "post gitlab webhook failed: %v\n", err)
		return 1
	}

	changeSet, err := waitForChangeSet(ctx, c, changeTitle)
	if err != nil {
		fmt.Fprintf(stderr, "wait for change set failed: %v\n", err)
		return 1
	}

	assessment, err := c.AssessRisk(ctx, types.CreateRiskAssessmentRequest{ChangeSetID: changeSet.ID})
	if err != nil {
		fmt.Fprintf(stderr, "risk assessment failed: %v\n", err)
		return 1
	}
	planResult, err := c.CreateRolloutPlan(ctx, types.CreateRolloutPlanRequest{ChangeSetID: changeSet.ID})
	if err != nil {
		fmt.Fprintf(stderr, "create rollout plan failed: %v\n", err)
		return 1
	}
	execution, err := c.CreateRolloutExecution(ctx, types.CreateRolloutExecutionRequest{
		RolloutPlanID:        planResult.Plan.ID,
		BackendType:          "kubernetes",
		BackendIntegrationID: kubernetesIntegration.ID,
		SignalProviderType:   "prometheus",
		SignalIntegrationID:  prometheusIntegration.ID,
	})
	if err != nil {
		fmt.Fprintf(stderr, "create rollout execution failed: %v\n", err)
		return 1
	}

	execution, err = ensureExecutionStarted(ctx, c, execution)
	if err != nil {
		fmt.Fprintf(stderr, "start rollout execution failed: %v\n", err)
		return 1
	}

	time.Sleep(12 * time.Second)
	detail, err := reconcileUntilRecommendation(ctx, c, execution.ID)
	if err != nil {
		fmt.Fprintf(stderr, "reconcile reference pilot flow failed: %v\n", err)
		return 1
	}

	if !detail.RuntimeSummary.AdvisoryOnly {
		fmt.Fprintln(stderr, "expected advisory runtime summary for the reference pilot execution")
		return 1
	}
	if !strings.HasPrefix(detail.RuntimeSummary.LatestDecision, "advisory_") {
		fmt.Fprintf(stderr, "expected advisory recommendation, got %s\n", detail.RuntimeSummary.LatestDecision)
		return 1
	}

	statusResult, err := c.SearchStatusEvents(ctx, "rollout_execution_id="+execution.ID+"&limit=50")
	if err != nil {
		fmt.Fprintf(stderr, "search status events failed: %v\n", err)
		return 1
	}
	auditEvents, err := c.ListAuditEvents(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "list audit events failed: %v\n", err)
		return 1
	}

	report := verificationReport{
		Profile:                referencePilotProfile,
		VerifiedAt:             time.Now().UTC().Format(time.RFC3339),
		ProofQuality:           referencePilotProofQuality,
		Organization:           org,
		Project:                project,
		Team:                   team,
		Service:                service,
		Environment:            environment,
		GitLabIntegration:      gitlabIntegration,
		KubernetesIntegration:  kubernetesIntegration,
		PrometheusIntegration:  prometheusIntegration,
		WebhookRegistration:    webhookRegistration,
		Repository:             repository,
		KubernetesResource:     kubernetesResource,
		PrometheusResource:     prometheusResource,
		ChangeSet:              changeSet,
		RolloutPlan:            planResult.Plan,
		Execution:              execution,
		ExecutionDetail:        detail,
		StatusEventCount:       len(statusResult.Events),
		TimelineEventCount:     len(detail.StatusTimeline),
		AuditEventCount:        len(auditEvents),
		WorkloadStateUpdatedAt: workloadUpdatedAt,
	}
	report.EvidenceSummary = buildReferencePilotEvidenceSummary(report)

	if err := validateVerificationReport(report); err != nil {
		fmt.Fprintf(stderr, "invalid reference pilot report: %v\n", err)
		return 1
	}

	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "marshal report failed: %v\n", err)
		return 1
	}
	if strings.TrimSpace(*reportPath) != "" {
		if err := os.WriteFile(*reportPath, append(body, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "write report failed: %v\n", err)
			return 1
		}
	}
	_, _ = stdout.Write(append(body, '\n'))
	_ = assessment
	return 0
}

func readVerificationReport(path string) (verificationReport, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return verificationReport{}, err
	}
	var report verificationReport
	if err := json.Unmarshal(body, &report); err != nil {
		return verificationReport{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return verificationReport{}, err
	}
	return normalizeVerificationReport(report, info.ModTime().UTC()), nil
}

func validateVerificationReport(report verificationReport) error {
	if strings.TrimSpace(report.Profile) != referencePilotProfile {
		return fmt.Errorf("profile must be %s", referencePilotProfile)
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(report.VerifiedAt)); err != nil {
		return fmt.Errorf("verified_at must be rfc3339: %w", err)
	}
	if strings.TrimSpace(report.ProofQuality) != referencePilotProofQuality {
		return fmt.Errorf("proof_quality must be %s", referencePilotProofQuality)
	}
	if strings.TrimSpace(report.Organization.ID) == "" || strings.TrimSpace(report.Project.ID) == "" || strings.TrimSpace(report.Team.ID) == "" || strings.TrimSpace(report.Service.ID) == "" || strings.TrimSpace(report.Environment.ID) == "" {
		return errors.New("organization, project, team, service, and environment evidence are required")
	}
	if strings.TrimSpace(report.GitLabIntegration.ID) == "" || strings.TrimSpace(report.KubernetesIntegration.ID) == "" || strings.TrimSpace(report.PrometheusIntegration.ID) == "" {
		return errors.New("gitlab, kubernetes, and prometheus integration evidence are required")
	}
	if strings.TrimSpace(report.WebhookRegistration.Registration.ID) == "" {
		return errors.New("webhook registration evidence is required")
	}
	if strings.TrimSpace(report.Repository.ID) == "" || !strings.EqualFold(strings.TrimSpace(report.Repository.Provider), "gitlab") {
		return errors.New("gitlab repository evidence is required")
	}
	if !strings.EqualFold(strings.TrimSpace(report.Repository.Status), "mapped") {
		return errors.New("reference pilot repository must be mapped")
	}
	if strings.TrimSpace(report.KubernetesResource.ID) == "" || strings.TrimSpace(report.KubernetesResource.ResourceType) != "kubernetes_workload" {
		return errors.New("kubernetes workload evidence is required")
	}
	if !strings.EqualFold(strings.TrimSpace(report.KubernetesResource.Status), "mapped") {
		return errors.New("kubernetes workload must be mapped")
	}
	if strings.TrimSpace(report.PrometheusResource.ID) == "" || strings.TrimSpace(report.PrometheusResource.ResourceType) != "prometheus_signal_target" {
		return errors.New("prometheus signal target evidence is required")
	}
	if strings.TrimSpace(report.ChangeSet.ID) == "" || strings.TrimSpace(report.RolloutPlan.ID) == "" || strings.TrimSpace(report.Execution.ID) == "" {
		return errors.New("change, rollout plan, and execution evidence are required")
	}
	if !report.ExecutionDetail.RuntimeSummary.AdvisoryOnly {
		return errors.New("reference pilot report must preserve advisory_only runtime evidence")
	}
	if !strings.HasPrefix(strings.TrimSpace(report.ExecutionDetail.RuntimeSummary.LatestDecision), "advisory_") {
		return errors.New("reference pilot report must preserve an advisory latest_decision")
	}
	if strings.TrimSpace(report.ExecutionDetail.RuntimeSummary.LastActionDisposition) != "suppressed" {
		return errors.New("reference pilot report must preserve last_action_disposition=suppressed")
	}
	if report.StatusEventCount <= 0 || report.TimelineEventCount <= 0 || report.AuditEventCount <= 0 {
		return errors.New("status, timeline, and audit counts must be greater than zero")
	}
	if len(report.EvidenceSummary) == 0 {
		return errors.New("evidence_summary is required")
	}
	return nil
}

func normalizeVerificationReport(report verificationReport, fallbackTime time.Time) verificationReport {
	if strings.TrimSpace(report.Profile) == "" {
		report.Profile = referencePilotProfile
	}
	if strings.TrimSpace(report.ProofQuality) == "" {
		report.ProofQuality = referencePilotProofQuality
	}
	if strings.TrimSpace(report.VerifiedAt) == "" {
		switch {
		case strings.TrimSpace(report.WorkloadStateUpdatedAt) != "":
			report.VerifiedAt = report.WorkloadStateUpdatedAt
		case !report.Execution.CreatedAt.IsZero():
			report.VerifiedAt = report.Execution.CreatedAt.UTC().Format(time.RFC3339)
		case !fallbackTime.IsZero():
			report.VerifiedAt = fallbackTime.UTC().Format(time.RFC3339)
		}
	}
	if len(report.EvidenceSummary) == 0 {
		report.EvidenceSummary = buildReferencePilotEvidenceSummary(report)
	}
	return report
}

func buildReferencePilotEvidenceSummary(report verificationReport) []string {
	summary := []string{
		fmt.Sprintf("gitlab repository %s is mapped through the reference fixture", strings.TrimSpace(report.Repository.Name)),
		fmt.Sprintf("kubernetes resource %s is mapped as %s", strings.TrimSpace(report.KubernetesResource.Name), strings.TrimSpace(report.KubernetesResource.ResourceType)),
		fmt.Sprintf("prometheus resource %s is mapped as %s", strings.TrimSpace(report.PrometheusResource.Name), strings.TrimSpace(report.PrometheusResource.ResourceType)),
		fmt.Sprintf("runtime advisory evidence retained latest_decision=%s disposition=%s", strings.TrimSpace(report.ExecutionDetail.RuntimeSummary.LatestDecision), strings.TrimSpace(report.ExecutionDetail.RuntimeSummary.LastActionDisposition)),
	}
	filtered := make([]string, 0, len(summary))
	for _, item := range summary {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}

func ensureOrganization(ctx context.Context, c *client.Client, name, slug string) (types.Organization, error) {
	organizations, err := c.ListOrganizations(ctx)
	if err != nil {
		return types.Organization{}, err
	}
	for _, org := range organizations {
		if strings.EqualFold(strings.TrimSpace(org.Slug), strings.TrimSpace(slug)) {
			return org, nil
		}
	}
	return c.CreateOrganization(ctx, types.CreateOrganizationRequest{Name: name, Slug: slug, Tier: "growth"})
}

func ensureProject(ctx context.Context, c *client.Client, organizationID, name, slug string) (types.Project, error) {
	projects, err := c.ListProjects(ctx)
	if err != nil {
		return types.Project{}, err
	}
	for _, project := range projects {
		if strings.EqualFold(strings.TrimSpace(project.Slug), strings.TrimSpace(slug)) {
			return project, nil
		}
	}
	return c.CreateProject(ctx, types.CreateProjectRequest{
		OrganizationID: organizationID,
		Name:           name,
		Slug:           slug,
		AdoptionMode:   "advisory",
		Description:    "Reference pilot project for real cluster and metrics proof",
	})
}

func ensureTeam(ctx context.Context, c *client.Client, organizationID, projectID, ownerUserID, name, slug string) (types.Team, error) {
	teams, err := c.ListTeams(ctx)
	if err != nil {
		return types.Team{}, err
	}
	for _, team := range teams {
		if strings.EqualFold(strings.TrimSpace(team.Slug), strings.TrimSpace(slug)) {
			return team, nil
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
	services, err := c.ListServices(ctx)
	if err != nil {
		return types.Service{}, err
	}
	for _, service := range services {
		if strings.EqualFold(strings.TrimSpace(service.Slug), strings.TrimSpace(slug)) {
			return service, nil
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
		Description:      "Reference pilot checkout service",
	})
}

func ensureEnvironment(ctx context.Context, c *client.Client, organizationID, projectID, name, slug string) (types.Environment, error) {
	environments, err := c.ListEnvironments(ctx)
	if err != nil {
		return types.Environment{}, err
	}
	for _, environment := range environments {
		if strings.EqualFold(strings.TrimSpace(environment.Slug), strings.TrimSpace(slug)) {
			return environment, nil
		}
	}
	return c.CreateEnvironment(ctx, types.CreateEnvironmentRequest{
		OrganizationID: organizationID,
		ProjectID:      projectID,
		Name:           name,
		Slug:           slug,
		Type:           "kubernetes",
		Region:         "local",
		Production:     true,
		ComplianceZone: "pilot",
	})
}

func ensureIntegration(ctx context.Context, c *client.Client, createReq types.CreateIntegrationRequest, updateReq types.UpdateIntegrationRequest) (types.Integration, error) {
	query := fmt.Sprintf("kind=%s&instance_key=%s", createReq.Kind, createReq.InstanceKey)
	integrations, err := c.ListIntegrationsWithQuery(ctx, query)
	if err != nil {
		return types.Integration{}, err
	}
	var integration types.Integration
	if len(integrations) == 0 {
		integration, err = c.CreateIntegration(ctx, createReq)
		if err != nil {
			return types.Integration{}, err
		}
	} else {
		integration = integrations[0]
	}
	updated, err := c.UpdateIntegration(ctx, integration.ID, updateReq)
	if err != nil {
		return types.Integration{}, err
	}
	return updated, nil
}

func ensureRepositoryMapping(ctx context.Context, c *client.Client, integrationID, projectID, serviceID, environmentID string) (types.Repository, error) {
	repositories, err := c.ListRepositories(ctx, "source_integration_id="+integrationID)
	if err != nil {
		return types.Repository{}, err
	}
	if len(repositories) == 0 {
		return types.Repository{}, errors.New("expected gitlab discovery to return at least one repository")
	}
	sort.SliceStable(repositories, func(i, j int) bool {
		return repositories[i].CreatedAt.After(repositories[j].CreatedAt)
	})
	repository := repositories[0]
	updated, err := c.UpdateRepository(ctx, repository.ID, types.UpdateRepositoryRequest{
		ProjectID:     stringPtr(projectID),
		ServiceID:     stringPtr(serviceID),
		EnvironmentID: stringPtr(environmentID),
		Status:        stringPtr("mapped"),
	})
	if err != nil {
		return types.Repository{}, err
	}
	return updated, nil
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
	resource := items[0]
	updated, err := c.UpdateDiscoveredResource(ctx, resource.ID, types.UpdateDiscoveredResourceRequest{
		ProjectID:     stringPtr(projectID),
		ServiceID:     stringPtr(serviceID),
		EnvironmentID: stringPtr(environmentID),
		RepositoryID:  stringPtr(repositoryID),
		Status:        stringPtr("mapped"),
	})
	if err != nil {
		return types.DiscoveredResource{}, err
	}
	return updated, nil
}

type workloadStateRequest struct {
	Version    string  `json:"version,omitempty"`
	LatencyMS  float64 `json:"latency_ms,omitempty"`
	ErrorRatio float64 `json:"error_ratio,omitempty"`
}

func updateReferenceWorkload(adminURL string, state workloadStateRequest) (string, error) {
	body, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, adminURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		payload, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("workload admin request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	var response struct {
		UpdatedAt string `json:"updated_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}
	return response.UpdatedAt, nil
}

func postGitLabMergeRequestWebhook(apiBaseURL, token, organizationID, integrationID, secret, title string) error {
	payload := map[string]any{
		"object_kind": "merge_request",
		"project": map[string]any{
			"id":                  101,
			"name":                "checkout-service",
			"web_url":             "http://127.0.0.1:39480/acme/checkout-service",
			"default_branch":      "main",
			"path_with_namespace": "acme/checkout-service",
			"namespace":           "acme",
		},
		"object_attributes": map[string]any{
			"iid":           7,
			"title":         title,
			"description":   "Reference pilot merge request to prove advisory-only runtime behavior.",
			"source_branch": "pilot/canary-proof",
			"target_branch": "main",
			"action":        "open",
			"state":         "opened",
			"url":           "http://127.0.0.1:39480/acme/checkout-service/-/merge_requests/7",
			"merge_status":  "can_be_merged",
			"last_commit": map[string]any{
				"id": fmt.Sprintf("refpilot-%d", time.Now().Unix()),
			},
		},
		"labels": []map[string]any{
			{"title": "pilot-proof"},
			{"title": "reference-environment"},
		},
		"reviewers": []map[string]any{
			{"username": "reviewer-bot"},
		},
		"user": map[string]any{
			"username": "reference-bot",
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, strings.TrimRight(apiBaseURL, "/")+"/api/v1/integrations/"+integrationID+"/webhooks/gitlab", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-CCP-Organization-ID", organizationID)
	req.Header.Set("X-Gitlab-Event", "Merge Request Hook")
	req.Header.Set("X-Gitlab-Event-UUID", fmt.Sprintf("refpilot-%d", time.Now().UnixNano()))
	req.Header.Set("X-Gitlab-Token", secret)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gitlab webhook failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	return nil
}

func waitForChangeSet(ctx context.Context, c *client.Client, summary string) (types.ChangeSet, error) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		items, err := c.ListChangeSets(ctx)
		if err != nil {
			return types.ChangeSet{}, err
		}
		for i := len(items) - 1; i >= 0; i-- {
			if strings.Contains(items[i].Summary, summary) {
				return items[i], nil
			}
		}
		time.Sleep(1500 * time.Millisecond)
	}
	return types.ChangeSet{}, fmt.Errorf("change set containing %q was not created", summary)
}

func ensureExecutionStarted(ctx context.Context, c *client.Client, execution types.RolloutExecution) (types.RolloutExecution, error) {
	var err error
	switch execution.Status {
	case "awaiting_approval":
		execution, err = c.AdvanceRolloutExecution(ctx, execution.ID, types.AdvanceRolloutExecutionRequest{
			Action: "approve",
			Reason: "reference pilot approval",
		})
		if err != nil {
			return types.RolloutExecution{}, err
		}
		fallthrough
	case "approved", "planned":
		return c.AdvanceRolloutExecution(ctx, execution.ID, types.AdvanceRolloutExecutionRequest{
			Action: "start",
			Reason: "reference pilot advisory verification",
		})
	default:
		return execution, nil
	}
}

func reconcileUntilRecommendation(ctx context.Context, c *client.Client, executionID string) (types.RolloutExecutionDetail, error) {
	deadline := time.Now().Add(30 * time.Second)
	var last types.RolloutExecutionDetail
	for time.Now().Before(deadline) {
		detail, err := c.ReconcileRolloutExecution(ctx, executionID)
		if err != nil {
			return types.RolloutExecutionDetail{}, err
		}
		last = detail
		if strings.HasPrefix(detail.RuntimeSummary.LatestDecision, "advisory_") {
			return detail, nil
		}
		time.Sleep(5 * time.Second)
	}
	return last, fmt.Errorf("reconcile did not produce an advisory recommendation before timeout")
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
