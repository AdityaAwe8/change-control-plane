package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

func (c *Client) CreateOrganization(ctx context.Context, req types.CreateOrganizationRequest) (types.Organization, error) {
	return doItem[types.Organization](ctx, c, http.MethodPost, "/api/v1/organizations", req)
}

func (c *Client) ListOrganizations(ctx context.Context) ([]types.Organization, error) {
	return doList[types.Organization](ctx, c, http.MethodGet, "/api/v1/organizations")
}

func (c *Client) CreateProject(ctx context.Context, req types.CreateProjectRequest) (types.Project, error) {
	return doItem[types.Project](ctx, c, http.MethodPost, "/api/v1/projects", req)
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

func (c *Client) AssessRisk(ctx context.Context, req types.CreateRiskAssessmentRequest) (types.RiskAssessmentResult, error) {
	return doItem[types.RiskAssessmentResult](ctx, c, http.MethodPost, "/api/v1/risk-assessments", req)
}

func (c *Client) CreateRolloutPlan(ctx context.Context, req types.CreateRolloutPlanRequest) (types.RolloutPlanResult, error) {
	return doItem[types.RolloutPlanResult](ctx, c, http.MethodPost, "/api/v1/rollout-plans", req)
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

func (c *Client) AdvanceRolloutExecution(ctx context.Context, id string, req types.AdvanceRolloutExecutionRequest) (types.RolloutExecution, error) {
	return doItem[types.RolloutExecution](ctx, c, http.MethodPost, "/api/v1/rollout-executions/"+id+"/advance", req)
}

func (c *Client) RecordVerificationResult(ctx context.Context, executionID string, req types.RecordVerificationResultRequest) (types.VerificationResult, error) {
	return doItem[types.VerificationResult](ctx, c, http.MethodPost, "/api/v1/rollout-executions/"+executionID+"/verification", req)
}

func (c *Client) ListIntegrations(ctx context.Context) ([]types.Integration, error) {
	return doList[types.Integration](ctx, c, http.MethodGet, "/api/v1/integrations")
}

func (c *Client) UpdateIntegration(ctx context.Context, id string, req types.UpdateIntegrationRequest) (types.Integration, error) {
	return doItem[types.Integration](ctx, c, http.MethodPatch, "/api/v1/integrations/"+id, req)
}

func (c *Client) IngestIntegrationGraph(ctx context.Context, id string, req types.IntegrationGraphIngestRequest) ([]types.GraphRelationship, error) {
	return doListBody[types.GraphRelationship](ctx, c, http.MethodPost, "/api/v1/integrations/"+id+"/graph-ingest", req)
}

func (c *Client) ListGraphRelationships(ctx context.Context) ([]types.GraphRelationship, error) {
	return doList[types.GraphRelationship](ctx, c, http.MethodGet, "/api/v1/graph/relationships")
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

func (c *Client) ListIncidents(ctx context.Context) ([]types.Incident, error) {
	return doList[types.Incident](ctx, c, http.MethodGet, "/api/v1/incidents")
}

func (c *Client) ListAuditEvents(ctx context.Context) ([]types.AuditEvent, error) {
	return doList[types.AuditEvent](ctx, c, http.MethodGet, "/api/v1/audit-events")
}

func (c *Client) Session(ctx context.Context) (types.SessionInfo, error) {
	return doItem[types.SessionInfo](ctx, c, http.MethodGet, "/api/v1/auth/session", nil)
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
