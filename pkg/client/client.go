package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Client struct {
	baseURL    string
	token      string
	orgID      string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) SetToken(token string) {
	c.token = strings.TrimSpace(token)
}

func (c *Client) SetOrganizationID(orgID string) {
	c.orgID = strings.TrimSpace(orgID)
}

func (c *Client) OrganizationID() string {
	return c.orgID
}

func (c *Client) CreateOrganization(ctx context.Context, req types.CreateOrganizationRequest) (types.Organization, error) {
	return doItem[types.Organization](ctx, c, http.MethodPost, "/api/v1/organizations", req)
}

func (c *Client) ListOrganizations(ctx context.Context) ([]types.Organization, error) {
	return doList[types.Organization](ctx, c, http.MethodGet, "/api/v1/organizations")
}

func (c *Client) CreateProject(ctx context.Context, req types.CreateProjectRequest) (types.Project, error) {
	return doItem[types.Project](ctx, c, http.MethodPost, "/api/v1/projects", req)
}

func (c *Client) CreateTeam(ctx context.Context, req types.CreateTeamRequest) (types.Team, error) {
	return doItem[types.Team](ctx, c, http.MethodPost, "/api/v1/teams", req)
}

func (c *Client) GetTeam(ctx context.Context, id string) (types.Team, error) {
	return doItem[types.Team](ctx, c, http.MethodGet, "/api/v1/teams/"+id, nil)
}

func (c *Client) ListTeams(ctx context.Context) ([]types.Team, error) {
	return doList[types.Team](ctx, c, http.MethodGet, "/api/v1/teams")
}

func (c *Client) UpdateTeam(ctx context.Context, id string, req types.UpdateTeamRequest) (types.Team, error) {
	return doItem[types.Team](ctx, c, http.MethodPatch, "/api/v1/teams/"+id, req)
}

func (c *Client) ArchiveTeam(ctx context.Context, id string) (types.Team, error) {
	return doItem[types.Team](ctx, c, http.MethodPost, "/api/v1/teams/"+id+"/archive", struct{}{})
}

func (c *Client) UpdateProject(ctx context.Context, id string, req types.UpdateProjectRequest) (types.Project, error) {
	return doItem[types.Project](ctx, c, http.MethodPatch, "/api/v1/projects/"+id, req)
}

func (c *Client) ArchiveProject(ctx context.Context, id string) (types.Project, error) {
	return doItem[types.Project](ctx, c, http.MethodPost, "/api/v1/projects/"+id+"/archive", struct{}{})
}

func (c *Client) ListProjects(ctx context.Context) ([]types.Project, error) {
	return doList[types.Project](ctx, c, http.MethodGet, "/api/v1/projects")
}

func (c *Client) CreateService(ctx context.Context, req types.CreateServiceRequest) (types.Service, error) {
	return doItem[types.Service](ctx, c, http.MethodPost, "/api/v1/services", req)
}

func (c *Client) GetService(ctx context.Context, id string) (types.Service, error) {
	return doItem[types.Service](ctx, c, http.MethodGet, "/api/v1/services/"+id, nil)
}

func (c *Client) UpdateService(ctx context.Context, id string, req types.UpdateServiceRequest) (types.Service, error) {
	return doItem[types.Service](ctx, c, http.MethodPatch, "/api/v1/services/"+id, req)
}

func (c *Client) ArchiveService(ctx context.Context, id string) (types.Service, error) {
	return doItem[types.Service](ctx, c, http.MethodPost, "/api/v1/services/"+id+"/archive", struct{}{})
}

func (c *Client) ListServices(ctx context.Context) ([]types.Service, error) {
	return doList[types.Service](ctx, c, http.MethodGet, "/api/v1/services")
}

func (c *Client) CreateEnvironment(ctx context.Context, req types.CreateEnvironmentRequest) (types.Environment, error) {
	return doItem[types.Environment](ctx, c, http.MethodPost, "/api/v1/environments", req)
}

func (c *Client) GetEnvironment(ctx context.Context, id string) (types.Environment, error) {
	return doItem[types.Environment](ctx, c, http.MethodGet, "/api/v1/environments/"+id, nil)
}

func (c *Client) UpdateEnvironment(ctx context.Context, id string, req types.UpdateEnvironmentRequest) (types.Environment, error) {
	return doItem[types.Environment](ctx, c, http.MethodPatch, "/api/v1/environments/"+id, req)
}

func (c *Client) ArchiveEnvironment(ctx context.Context, id string) (types.Environment, error) {
	return doItem[types.Environment](ctx, c, http.MethodPost, "/api/v1/environments/"+id+"/archive", struct{}{})
}

func (c *Client) ListEnvironments(ctx context.Context) ([]types.Environment, error) {
	return doList[types.Environment](ctx, c, http.MethodGet, "/api/v1/environments")
}

func (c *Client) CreateChangeSet(ctx context.Context, req types.CreateChangeSetRequest) (types.ChangeSet, error) {
	return doItem[types.ChangeSet](ctx, c, http.MethodPost, "/api/v1/changes", req)
}

func (c *Client) ListChangeSets(ctx context.Context) ([]types.ChangeSet, error) {
	return doList[types.ChangeSet](ctx, c, http.MethodGet, "/api/v1/changes")
}

func (c *Client) GetChangeSet(ctx context.Context, id string) (types.ChangeSet, error) {
	return doItem[types.ChangeSet](ctx, c, http.MethodGet, "/api/v1/changes/"+id, nil)
}

func (c *Client) AssessRisk(ctx context.Context, req types.CreateRiskAssessmentRequest) (types.RiskAssessmentResult, error) {
	return doItem[types.RiskAssessmentResult](ctx, c, http.MethodPost, "/api/v1/risk-assessments", req)
}

func (c *Client) ListRiskAssessments(ctx context.Context) ([]types.RiskAssessment, error) {
	return doList[types.RiskAssessment](ctx, c, http.MethodGet, "/api/v1/risk-assessments")
}

func (c *Client) CreateRolloutPlan(ctx context.Context, req types.CreateRolloutPlanRequest) (types.RolloutPlanResult, error) {
	return doItem[types.RolloutPlanResult](ctx, c, http.MethodPost, "/api/v1/rollout-plans", req)
}

func (c *Client) ListRolloutPlans(ctx context.Context) ([]types.RolloutPlan, error) {
	return doList[types.RolloutPlan](ctx, c, http.MethodGet, "/api/v1/rollout-plans")
}

func (c *Client) ListPolicies(ctx context.Context) ([]types.Policy, error) {
	return doList[types.Policy](ctx, c, http.MethodGet, "/api/v1/policies")
}

func (c *Client) GetPolicy(ctx context.Context, id string) (types.Policy, error) {
	return doItem[types.Policy](ctx, c, http.MethodGet, "/api/v1/policies/"+id, nil)
}

func (c *Client) CreatePolicy(ctx context.Context, req types.CreatePolicyRequest) (types.Policy, error) {
	return doItem[types.Policy](ctx, c, http.MethodPost, "/api/v1/policies", req)
}

func (c *Client) UpdatePolicy(ctx context.Context, id string, req types.UpdatePolicyRequest) (types.Policy, error) {
	return doItem[types.Policy](ctx, c, http.MethodPatch, "/api/v1/policies/"+id, req)
}

func (c *Client) ListPolicyDecisions(ctx context.Context, rawQuery string) ([]types.PolicyDecision, error) {
	path := "/api/v1/policy-decisions"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.PolicyDecision](ctx, c, http.MethodGet, path)
}

func (c *Client) ListRolloutExecutions(ctx context.Context) ([]types.RolloutExecution, error) {
	return doList[types.RolloutExecution](ctx, c, http.MethodGet, "/api/v1/rollout-executions")
}

func (c *Client) CreateRolloutExecution(ctx context.Context, req types.CreateRolloutExecutionRequest) (types.RolloutExecution, error) {
	return doItem[types.RolloutExecution](ctx, c, http.MethodPost, "/api/v1/rollout-executions", req)
}

func (c *Client) GetRolloutExecution(ctx context.Context, id string) (types.RolloutExecutionDetail, error) {
	return doItem[types.RolloutExecutionDetail](ctx, c, http.MethodGet, "/api/v1/rollout-executions/"+id, nil)
}

func (c *Client) GetRolloutEvidencePack(ctx context.Context, id string) (types.RolloutEvidencePack, error) {
	return doItem[types.RolloutEvidencePack](ctx, c, http.MethodGet, "/api/v1/rollout-executions/"+id+"/evidence-pack", nil)
}

func (c *Client) AdvanceRolloutExecution(ctx context.Context, id string, req types.AdvanceRolloutExecutionRequest) (types.RolloutExecution, error) {
	return doItem[types.RolloutExecution](ctx, c, http.MethodPost, "/api/v1/rollout-executions/"+id+"/advance", req)
}

func (c *Client) ReconcileRolloutExecution(ctx context.Context, id string) (types.RolloutExecutionDetail, error) {
	return doItem[types.RolloutExecutionDetail](ctx, c, http.MethodPost, "/api/v1/rollout-executions/"+id+"/reconcile", struct{}{})
}

func (c *Client) PauseRolloutExecution(ctx context.Context, id, reason string) (types.RolloutExecution, error) {
	return doItem[types.RolloutExecution](ctx, c, http.MethodPost, "/api/v1/rollout-executions/"+id+"/pause?reason="+url.QueryEscape(reason), nil)
}

func (c *Client) ResumeRolloutExecution(ctx context.Context, id, reason string) (types.RolloutExecution, error) {
	return doItem[types.RolloutExecution](ctx, c, http.MethodPost, "/api/v1/rollout-executions/"+id+"/resume?reason="+url.QueryEscape(reason), nil)
}

func (c *Client) RollbackRolloutExecution(ctx context.Context, id, reason string) (types.RolloutExecution, error) {
	return doItem[types.RolloutExecution](ctx, c, http.MethodPost, "/api/v1/rollout-executions/"+id+"/rollback?reason="+url.QueryEscape(reason), nil)
}

func (c *Client) CreateSignalSnapshot(ctx context.Context, executionID string, req types.CreateSignalSnapshotRequest) (types.SignalSnapshot, error) {
	return doItem[types.SignalSnapshot](ctx, c, http.MethodPost, "/api/v1/rollout-executions/"+executionID+"/signal-snapshots", req)
}

func (c *Client) RecordVerificationResult(ctx context.Context, executionID string, req types.RecordVerificationResultRequest) (types.VerificationResult, error) {
	return doItem[types.VerificationResult](ctx, c, http.MethodPost, "/api/v1/rollout-executions/"+executionID+"/verification", req)
}

func (c *Client) ListIntegrations(ctx context.Context) ([]types.Integration, error) {
	return c.ListIntegrationsWithQuery(ctx, "")
}

func (c *Client) ListIntegrationsWithQuery(ctx context.Context, rawQuery string) ([]types.Integration, error) {
	path := "/api/v1/integrations"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.Integration](ctx, c, http.MethodGet, path)
}

func (c *Client) CreateIntegration(ctx context.Context, req types.CreateIntegrationRequest) (types.Integration, error) {
	return doItem[types.Integration](ctx, c, http.MethodPost, "/api/v1/integrations", req)
}

func (c *Client) UpdateIntegration(ctx context.Context, id string, req types.UpdateIntegrationRequest) (types.Integration, error) {
	return doItem[types.Integration](ctx, c, http.MethodPatch, "/api/v1/integrations/"+id, req)
}

func (c *Client) TestIntegration(ctx context.Context, id string) (types.IntegrationTestResult, error) {
	return doItem[types.IntegrationTestResult](ctx, c, http.MethodPost, "/api/v1/integrations/"+id+"/test", struct{}{})
}

func (c *Client) SyncIntegration(ctx context.Context, id string) (types.IntegrationSyncResult, error) {
	return doItem[types.IntegrationSyncResult](ctx, c, http.MethodPost, "/api/v1/integrations/"+id+"/sync", struct{}{})
}

func (c *Client) ListIntegrationSyncRuns(ctx context.Context, id string) ([]types.IntegrationSyncRun, error) {
	return doList[types.IntegrationSyncRun](ctx, c, http.MethodGet, "/api/v1/integrations/"+id+"/sync-runs")
}

func (c *Client) StartGitHubOnboarding(ctx context.Context, id string) (types.GitHubOnboardingStartResult, error) {
	return doItem[types.GitHubOnboardingStartResult](ctx, c, http.MethodPost, "/api/v1/integrations/"+id+"/github/onboarding/start", struct{}{})
}

func (c *Client) IntegrationCoverageSummary(ctx context.Context) (types.CoverageSummary, error) {
	return doItem[types.CoverageSummary](ctx, c, http.MethodGet, "/api/v1/integrations/coverage", nil)
}

func (c *Client) GetWebhookRegistration(ctx context.Context, id string) (types.WebhookRegistrationResult, error) {
	return doItem[types.WebhookRegistrationResult](ctx, c, http.MethodGet, "/api/v1/integrations/"+id+"/webhook-registration", nil)
}

func (c *Client) SyncWebhookRegistration(ctx context.Context, id string) (types.WebhookRegistrationResult, error) {
	return doItem[types.WebhookRegistrationResult](ctx, c, http.MethodPost, "/api/v1/integrations/"+id+"/webhook-registration/sync", struct{}{})
}

func (c *Client) ListIdentityProviders(ctx context.Context) ([]types.IdentityProvider, error) {
	return doList[types.IdentityProvider](ctx, c, http.MethodGet, "/api/v1/identity-providers")
}

func (c *Client) CreateIdentityProvider(ctx context.Context, req types.CreateIdentityProviderRequest) (types.IdentityProvider, error) {
	return doItem[types.IdentityProvider](ctx, c, http.MethodPost, "/api/v1/identity-providers", req)
}

func (c *Client) UpdateIdentityProvider(ctx context.Context, id string, req types.UpdateIdentityProviderRequest) (types.IdentityProvider, error) {
	return doItem[types.IdentityProvider](ctx, c, http.MethodPatch, "/api/v1/identity-providers/"+id, req)
}

func (c *Client) TestIdentityProvider(ctx context.Context, id string) (types.IdentityProviderTestResult, error) {
	return doItem[types.IdentityProviderTestResult](ctx, c, http.MethodPost, "/api/v1/identity-providers/"+id+"/test", struct{}{})
}

func (c *Client) ListBrowserSessions(ctx context.Context, rawQuery string) ([]types.BrowserSessionInfo, error) {
	path := "/api/v1/browser-sessions"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.BrowserSessionInfo](ctx, c, http.MethodGet, path)
}

func (c *Client) RevokeBrowserSession(ctx context.Context, id string) (types.BrowserSessionInfo, error) {
	return doItem[types.BrowserSessionInfo](ctx, c, http.MethodPost, "/api/v1/browser-sessions/"+id+"/revoke", struct{}{})
}

func (c *Client) ListOutboxEvents(ctx context.Context, rawQuery string) ([]types.OutboxEvent, error) {
	path := "/api/v1/outbox-events"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.OutboxEvent](ctx, c, http.MethodGet, path)
}

func (c *Client) RetryOutboxEvent(ctx context.Context, id string) (types.OutboxEvent, error) {
	return doItem[types.OutboxEvent](ctx, c, http.MethodPost, "/api/v1/outbox-events/"+id+"/retry", struct{}{})
}

func (c *Client) RequeueOutboxEvent(ctx context.Context, id string) (types.OutboxEvent, error) {
	return doItem[types.OutboxEvent](ctx, c, http.MethodPost, "/api/v1/outbox-events/"+id+"/requeue", struct{}{})
}

func (c *Client) IngestIntegrationGraph(ctx context.Context, id string, req types.IntegrationGraphIngestRequest) ([]types.GraphRelationship, error) {
	return doListBody[types.GraphRelationship](ctx, c, http.MethodPost, "/api/v1/integrations/"+id+"/graph-ingest", req)
}

func (c *Client) ListGraphRelationships(ctx context.Context, rawQuery string) ([]types.GraphRelationship, error) {
	path := "/api/v1/graph/relationships"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.GraphRelationship](ctx, c, http.MethodGet, path)
}

func (c *Client) ListRepositories(ctx context.Context, rawQuery string) ([]types.Repository, error) {
	path := "/api/v1/repositories"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.Repository](ctx, c, http.MethodGet, path)
}

func (c *Client) UpdateRepository(ctx context.Context, id string, req types.UpdateRepositoryRequest) (types.Repository, error) {
	return doItem[types.Repository](ctx, c, http.MethodPatch, "/api/v1/repositories/"+id, req)
}

func (c *Client) ListDiscoveredResources(ctx context.Context, rawQuery string) ([]types.DiscoveredResource, error) {
	path := "/api/v1/discovered-resources"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.DiscoveredResource](ctx, c, http.MethodGet, path)
}

func (c *Client) UpdateDiscoveredResource(ctx context.Context, id string, req types.UpdateDiscoveredResourceRequest) (types.DiscoveredResource, error) {
	return doItem[types.DiscoveredResource](ctx, c, http.MethodPatch, "/api/v1/discovered-resources/"+id, req)
}

func (c *Client) CreateServiceAccount(ctx context.Context, req types.CreateServiceAccountRequest) (types.ServiceAccount, error) {
	return doItem[types.ServiceAccount](ctx, c, http.MethodPost, "/api/v1/service-accounts", req)
}

func (c *Client) ListServiceAccounts(ctx context.Context) ([]types.ServiceAccount, error) {
	return doList[types.ServiceAccount](ctx, c, http.MethodGet, "/api/v1/service-accounts")
}

func (c *Client) DeactivateServiceAccount(ctx context.Context, id string) (types.ServiceAccount, error) {
	return doItem[types.ServiceAccount](ctx, c, http.MethodPost, "/api/v1/service-accounts/"+id+"/deactivate", struct{}{})
}

func (c *Client) IssueServiceAccountToken(ctx context.Context, id string, req types.IssueAPITokenRequest) (types.IssuedAPITokenResponse, error) {
	return doItem[types.IssuedAPITokenResponse](ctx, c, http.MethodPost, "/api/v1/service-accounts/"+id+"/tokens", req)
}

func (c *Client) ListServiceAccountTokens(ctx context.Context, id string) ([]types.APIToken, error) {
	return doList[types.APIToken](ctx, c, http.MethodGet, "/api/v1/service-accounts/"+id+"/tokens")
}

func (c *Client) RevokeServiceAccountToken(ctx context.Context, serviceAccountID, tokenID string) (types.APIToken, error) {
	return doItem[types.APIToken](ctx, c, http.MethodPost, "/api/v1/service-accounts/"+serviceAccountID+"/tokens/"+tokenID+"/revoke", struct{}{})
}

func (c *Client) RotateServiceAccountToken(ctx context.Context, serviceAccountID, tokenID string, req types.RotateAPITokenRequest) (types.IssuedAPITokenResponse, error) {
	return doItem[types.IssuedAPITokenResponse](ctx, c, http.MethodPost, "/api/v1/service-accounts/"+serviceAccountID+"/tokens/"+tokenID+"/rotate", req)
}

func (c *Client) ListIncidents(ctx context.Context, rawQuery string) ([]types.Incident, error) {
	path := "/api/v1/incidents"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.Incident](ctx, c, http.MethodGet, path)
}

func (c *Client) GetIncidentDetail(ctx context.Context, id string) (types.IncidentDetail, error) {
	return doItem[types.IncidentDetail](ctx, c, http.MethodGet, "/api/v1/incidents/"+id, nil)
}

func (c *Client) ListAuditEvents(ctx context.Context) ([]types.AuditEvent, error) {
	return doList[types.AuditEvent](ctx, c, http.MethodGet, "/api/v1/audit-events")
}

func (c *Client) ListStatusEvents(ctx context.Context, rawQuery string) ([]types.StatusEvent, error) {
	path := "/api/v1/status-events"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.StatusEvent](ctx, c, http.MethodGet, path)
}

func (c *Client) SearchStatusEvents(ctx context.Context, rawQuery string) (types.StatusEventQueryResult, error) {
	path := "/api/v1/status-events/search"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doItem[types.StatusEventQueryResult](ctx, c, http.MethodGet, path, nil)
}

func (c *Client) GetStatusEvent(ctx context.Context, id string) (types.StatusEvent, error) {
	return doItem[types.StatusEvent](ctx, c, http.MethodGet, "/api/v1/status-events/"+id, nil)
}

func (c *Client) ListProjectStatusEvents(ctx context.Context, id, rawQuery string) ([]types.StatusEvent, error) {
	path := "/api/v1/projects/" + id + "/status-events"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.StatusEvent](ctx, c, http.MethodGet, path)
}

func (c *Client) ListServiceStatusEvents(ctx context.Context, id, rawQuery string) ([]types.StatusEvent, error) {
	path := "/api/v1/services/" + id + "/status-events"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.StatusEvent](ctx, c, http.MethodGet, path)
}

func (c *Client) ListEnvironmentStatusEvents(ctx context.Context, id, rawQuery string) ([]types.StatusEvent, error) {
	path := "/api/v1/environments/" + id + "/status-events"
	if strings.TrimSpace(rawQuery) != "" {
		path += "?" + strings.TrimPrefix(rawQuery, "?")
	}
	return doList[types.StatusEvent](ctx, c, http.MethodGet, path)
}

func (c *Client) ListRolloutExecutionTimeline(ctx context.Context, id string) ([]types.StatusEvent, error) {
	return doList[types.StatusEvent](ctx, c, http.MethodGet, "/api/v1/rollout-executions/"+id+"/timeline")
}

func (c *Client) ListRollbackPolicies(ctx context.Context) ([]types.RollbackPolicy, error) {
	return doList[types.RollbackPolicy](ctx, c, http.MethodGet, "/api/v1/rollback-policies")
}

func (c *Client) CreateRollbackPolicy(ctx context.Context, req types.CreateRollbackPolicyRequest) (types.RollbackPolicy, error) {
	return doItem[types.RollbackPolicy](ctx, c, http.MethodPost, "/api/v1/rollback-policies", req)
}

func (c *Client) UpdateRollbackPolicy(ctx context.Context, id string, req types.UpdateRollbackPolicyRequest) (types.RollbackPolicy, error) {
	return doItem[types.RollbackPolicy](ctx, c, http.MethodPatch, "/api/v1/rollback-policies/"+id, req)
}

func (c *Client) Session(ctx context.Context) (types.SessionInfo, error) {
	return doItem[types.SessionInfo](ctx, c, http.MethodGet, "/api/v1/auth/session", nil)
}

func (c *Client) SignUp(ctx context.Context, req types.SignUpRequest) (types.AuthResponse, error) {
	return doItem[types.AuthResponse](ctx, c, http.MethodPost, "/api/v1/auth/sign-up", req)
}

func (c *Client) SignIn(ctx context.Context, req types.SignInRequest) (types.AuthResponse, error) {
	return doItem[types.AuthResponse](ctx, c, http.MethodPost, "/api/v1/auth/sign-in", req)
}

func (c *Client) DevLogin(ctx context.Context, req types.DevLoginRequest) (types.DevLoginResponse, error) {
	return doItem[types.DevLoginResponse](ctx, c, http.MethodPost, "/api/v1/auth/dev/login", req)
}

func doItem[T any](ctx context.Context, c *Client, method, path string, body any) (T, error) {
	var zero T

	req, err := newRequest(ctx, c, c.baseURL+path, method, body)
	if err != nil {
		return zero, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error.Message != "" {
			return zero, errors.New(apiErr.Error.Message)
		}
		return zero, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var envelope types.ItemResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return zero, err
	}
	return envelope.Data, nil
}

func doList[T any](ctx context.Context, c *Client, method, path string) ([]T, error) {
	req, err := newRequest(ctx, c, c.baseURL+path, method, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var envelope types.ListResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func doListBody[T any](ctx context.Context, c *Client, method, path string, body any) ([]T, error) {
	req, err := newRequest(ctx, c, c.baseURL+path, method, body)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error.Message != "" {
			return nil, errors.New(apiErr.Error.Message)
		}
		return nil, fmt.Errorf("request failed with status %d", resp.StatusCode)
	}

	var envelope types.ListResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return envelope.Data, nil
}

func newRequest(ctx context.Context, c *Client, url, method string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		buffer := &bytes.Buffer{}
		if err := json.NewEncoder(buffer).Encode(body); err != nil {
			return nil, err
		}
		reader = buffer
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if c.orgID != "" {
		req.Header.Set("X-CCP-Organization-ID", c.orgID)
	}
	return req, nil
}
