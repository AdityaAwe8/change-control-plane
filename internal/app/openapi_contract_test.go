package app_test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestOpenAPIIncludesPilotHardeningRoutesAndSchemas(t *testing.T) {
	t.Parallel()

	content := openAPIContent(t)

	requiredFragments := []string{
		"/api/v1/auth/sign-up:",
		"/api/v1/auth/sign-in:",
		"/api/v1/auth/logout:",
		"/api/v1/auth/providers:",
		"/api/v1/auth/providers/{id}/start:",
		"/api/v1/auth/providers/callback:",
		"/api/v1/incidents:",
		"/api/v1/incidents/{id}:",
		"/api/v1/integrations:",
		"/api/v1/page-state/integrations:",
		"/api/v1/integrations/{id}/test:",
		"/api/v1/integrations/{id}/sync:",
		"/api/v1/integrations/{id}/sync-runs:",
		"/api/v1/integrations/{id}/webhook-registration:",
		"/api/v1/integrations/{id}/webhook-registration/sync:",
		"/api/v1/integrations/{id}/github/onboarding/start:",
		"/api/v1/integrations/github/callback:",
		"/api/v1/integrations/coverage:",
		"/api/v1/integrations/{id}/webhooks/github:",
		"/api/v1/integrations/{id}/webhooks/gitlab:",
		"/api/v1/repositories:",
		"/api/v1/repositories/{id}:",
		"/api/v1/discovered-resources:",
		"/api/v1/discovered-resources/{id}:",
		"/api/v1/identity-providers:",
		"/api/v1/identity-providers/{id}:",
		"/api/v1/identity-providers/{id}/test:",
		"/api/v1/outbox-events:",
		"/api/v1/outbox-events/{id}/retry:",
		"/api/v1/outbox-events/{id}/requeue:",
		"/api/v1/page-state/enterprise:",
		"/api/v1/page-state/graph:",
		"/api/v1/page-state/rollout:",
		"/api/v1/page-state/deployments:",
		"/api/v1/page-state/simulation:",
		"/api/v1/service-accounts:",
		"/api/v1/service-accounts/{id}/deactivate:",
		"/api/v1/service-accounts/{id}/tokens:",
		"/api/v1/service-accounts/{id}/tokens/{token_id}/revoke:",
		"/api/v1/service-accounts/{id}/tokens/{token_id}/rotate:",
		"/api/v1/status-events/search:",
		"IdentityProvider:",
		"PublicIdentityProvider:",
		"IdentityProviderStartResult:",
		"IdentityProviderTestResult:",
		"WebhookRegistration:",
		"WebhookRegistrationResult:",
		"WebhookRegistrationResultItemResponse:",
		"OutboxEvent:",
		"OutboxEventListResponse:",
		"OutboxEventItemResponse:",
		"ServiceAccount:",
		"APIToken:",
		"IssuedAPITokenResponse:",
		"CreateServiceAccountRequest:",
		"IssueAPITokenRequest:",
		"RotateAPITokenRequest:",
		"IntegrationTestResult:",
		"IntegrationSyncResult:",
		"IntegrationListResponse:",
		"IntegrationItemResponse:",
		"IntegrationSyncRun:",
		"IntegrationSyncRunListResponse:",
		"IntegrationSyncRunItemResponse:",
		"IntegrationTestResultItemResponse:",
		"IntegrationSyncResultItemResponse:",
		"CreateIntegrationRequest:",
		"GitHubOnboardingStartResult:",
		"GitHubOnboardingStartResultItemResponse:",
		"Organization:",
		"OrganizationListResponse:",
		"OrganizationItemResponse:",
		"Project:",
		"ProjectListResponse:",
		"ProjectItemResponse:",
		"Team:",
		"TeamListResponse:",
		"TeamItemResponse:",
		"Service:",
		"ServiceListResponse:",
		"ServiceItemResponse:",
		"Environment:",
		"EnvironmentListResponse:",
		"EnvironmentItemResponse:",
		"Incident:",
		"IncidentListResponse:",
		"IncidentDetail:",
		"IncidentDetailItemResponse:",
		"ChangeSet:",
		"ChangeSetListResponse:",
		"ChangeSetItemResponse:",
		"BlastRadius:",
		"RiskAssessment:",
		"RiskAssessmentListResponse:",
		"RiskAssessmentResult:",
		"RiskAssessmentResultItemResponse:",
		"PolicyDecision:",
		"RolloutStep:",
		"RolloutPlan:",
		"RolloutPlanListResponse:",
		"RolloutPlanResult:",
		"RolloutPlanResultItemResponse:",
		"AuditEvent:",
		"AuditEventListResponse:",
		"RollbackPolicyListResponse:",
		"RollbackPolicyItemResponse:",
		"VerificationResultItemResponse:",
		"Repository:",
		"RepositoryListResponse:",
		"RepositoryItemResponse:",
		"DiscoveredResource:",
		"DiscoveredResourceListResponse:",
		"DiscoveredResourceItemResponse:",
		"CoverageSummary:",
		"CoverageSummaryItemResponse:",
		"IntegrationsPageState:",
		"IntegrationsPageStateItemResponse:",
		"EnterprisePageState:",
		"EnterprisePageStateItemResponse:",
		"GraphPageState:",
		"GraphPageStateItemResponse:",
		"RolloutPageState:",
		"RolloutPageStateItemResponse:",
		"DeploymentsPageState:",
		"DeploymentsPageStateItemResponse:",
		"SimulationPageState:",
		"SimulationPageStateItemResponse:",
		"StatusEventQueryResult:",
		"GraphRelationshipListResponse:",
		"RolloutExecutionRuntimeSummary:",
		"ErrorResponse:",
	}

	for _, fragment := range requiredFragments {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected openapi contract to include %q", fragment)
		}
	}
}

func TestOpenAPIOlderCRUDSurfacesUseDocumentedResponseSchemas(t *testing.T) {
	t.Parallel()

	content := openAPIContent(t)

	requiredFragments := []string{
		"/api/v1/organizations:",
		"$ref: '#/components/schemas/OrganizationListResponse'",
		"/api/v1/organizations/{id}:",
		"$ref: '#/components/schemas/OrganizationItemResponse'",
		"/api/v1/projects:",
		"$ref: '#/components/schemas/ProjectListResponse'",
		"/api/v1/projects/{id}:",
		"$ref: '#/components/schemas/ProjectItemResponse'",
		"/api/v1/teams:",
		"$ref: '#/components/schemas/TeamListResponse'",
		"/api/v1/teams/{id}:",
		"$ref: '#/components/schemas/TeamItemResponse'",
		"/api/v1/services:",
		"$ref: '#/components/schemas/ServiceListResponse'",
		"/api/v1/services/{id}:",
		"$ref: '#/components/schemas/ServiceItemResponse'",
		"/api/v1/environments:",
		"$ref: '#/components/schemas/EnvironmentListResponse'",
		"/api/v1/environments/{id}:",
		"$ref: '#/components/schemas/EnvironmentItemResponse'",
		"$ref: '#/components/responses/UnauthorizedError'",
		"$ref: '#/components/responses/ValidationError'",
		"$ref: '#/components/responses/NotFoundError'",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected openapi truth pass fragment %q", fragment)
		}
	}
}

func TestOpenAPIChangeRiskRolloutAuditAndRollbackRoutesUseDocumentedResponseSchemas(t *testing.T) {
	t.Parallel()

	content := openAPIContent(t)

	sections := []struct {
		name      string
		path      string
		method    string
		fragments []string
	}{
		{
			name:   "change list",
			path:   "/api/v1/changes",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/ChangeSetListResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			name:   "change detail",
			path:   "/api/v1/changes/{id}",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/ChangeSetItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			name:   "risk assessments",
			path:   "/api/v1/risk-assessments",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/RiskAssessmentListResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			name:   "rollout plans",
			path:   "/api/v1/rollout-plans",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/RolloutPlanListResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			name:   "rollback policies",
			path:   "/api/v1/rollback-policies",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/RollbackPolicyListResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			name:   "audit events",
			path:   "/api/v1/audit-events",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/AuditEventListResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
	}

	for _, section := range sections {
		requireOpenAPIFragments(t, openAPIMethodSection(t, content, section.path, section.method), section.fragments...)
	}
}

func TestOpenAPIIncidentListDocumentsSupportedFilters(t *testing.T) {
	t.Parallel()

	section := openAPIMethodSection(t, openAPIContent(t), "/api/v1/incidents", "GET")
	requireOpenAPIFragments(t, section,
		"name: project_id",
		"name: service_id",
		"name: environment_id",
		"name: change_set_id",
		"name: severity",
		"name: status",
		"name: search",
		"name: limit",
		"$ref: '#/components/schemas/IncidentListResponse'",
	)
}

func TestOpenAPIVerificationRouteUsesDocumentedRequestAndResponseSchemas(t *testing.T) {
	t.Parallel()

	section := openAPIMethodSection(t, openAPIContent(t), "/api/v1/rollout-executions/{id}/verification", "POST")
	requireOpenAPIFragments(t, section,
		"name: id",
		"$ref: '#/components/schemas/RecordVerificationResultRequest'",
		"$ref: '#/components/schemas/VerificationResultItemResponse'",
		"$ref: '#/components/responses/UnauthorizedError'",
		"$ref: '#/components/responses/ForbiddenError'",
		"$ref: '#/components/responses/NotFoundError'",
		"$ref: '#/components/responses/ValidationError'",
	)
}

func TestOpenAPIRegisteredHTTPRoutesHavePathAndMethodEntries(t *testing.T) {
	t.Parallel()

	content := openAPIContent(t)
	for _, route := range registeredHTTPRoutes(t) {
		if openAPIPathSection(content, route.Path) == "" {
			t.Fatalf("expected openapi to document registered path %s %s", route.Method, route.Path)
		}
		if openAPIMethodSection(t, content, route.Path, route.Method) == "" {
			t.Fatalf("expected openapi to document registered method %s %s", route.Method, route.Path)
		}
	}
}

func TestOpenAPIIntegrationAndDiagnosticsRoutesUseDocumentedResponseSchemas(t *testing.T) {
	t.Parallel()

	content := openAPIContent(t)
	sections := []struct {
		path      string
		method    string
		fragments []string
	}{
		{
			path:   "/api/v1/page-state/rollout",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/RolloutPageStateItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			path:   "/api/v1/page-state/deployments",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/DeploymentsPageStateItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"name: search",
				"name: rollback_only",
				"name: service_id",
				"name: environment_id",
				"name: source",
				"name: event_type",
				"name: automated",
				"name: limit",
				"name: offset",
			},
		},
		{
			path:   "/api/v1/page-state/integrations",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/IntegrationsPageStateItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			path:   "/api/v1/integrations",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/IntegrationListResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"name: kind",
				"name: instance_key",
				"name: search",
			},
		},
		{
			path:   "/api/v1/integrations",
			method: "POST",
			fragments: []string{
				"$ref: '#/components/schemas/CreateIntegrationRequest'",
				"$ref: '#/components/schemas/IntegrationItemResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			path:   "/api/v1/integrations/coverage",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/CoverageSummaryItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}",
			method: "PATCH",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/UpdateIntegrationRequest'",
				"$ref: '#/components/schemas/IntegrationItemResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}/test",
			method: "POST",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/IntegrationTestResultItemResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}/sync",
			method: "POST",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/IntegrationSyncResultItemResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}/sync-runs",
			method: "GET",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/IntegrationSyncRunListResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}/webhook-registration",
			method: "GET",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/WebhookRegistrationResultItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}/webhook-registration/sync",
			method: "POST",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/WebhookRegistrationResultItemResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}/github/onboarding/start",
			method: "POST",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/GitHubOnboardingStartResultItemResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/integrations/github/callback",
			method: "GET",
			fragments: []string{
				"security: []",
				"name: state",
				"name: installation_id",
				"$ref: '#/components/schemas/IntegrationItemResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}/graph-ingest",
			method: "POST",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/IntegrationGraphIngestRequest'",
				"$ref: '#/components/schemas/GraphRelationshipListResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}/webhooks/github",
			method: "POST",
			fragments: []string{
				"security: []",
				"name: id",
				"$ref: '#/components/schemas/IntegrationSyncRunItemResponse'",
			},
		},
		{
			path:   "/api/v1/integrations/{id}/webhooks/gitlab",
			method: "POST",
			fragments: []string{
				"security: []",
				"name: id",
				"$ref: '#/components/schemas/IntegrationSyncRunItemResponse'",
			},
		},
		{
			path:   "/api/v1/repositories",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/RepositoryListResponse'",
				"name: source_integration_id",
				"name: provider",
			},
		},
		{
			path:   "/api/v1/repositories/{id}",
			method: "PATCH",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/UpdateRepositoryRequest'",
				"$ref: '#/components/schemas/RepositoryItemResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/discovered-resources",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/DiscoveredResourceListResponse'",
				"name: integration_id",
				"name: unmapped_only",
				"name: limit",
			},
		},
		{
			path:   "/api/v1/discovered-resources/{id}",
			method: "PATCH",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/UpdateDiscoveredResourceRequest'",
				"$ref: '#/components/schemas/DiscoveredResourceItemResponse'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/graph/relationships",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/GraphRelationshipListResponse'",
			},
		},
		{
			path:   "/api/v1/page-state/graph",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/GraphPageStateItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			path:   "/api/v1/identity-providers",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/IdentityProviderListResponse'",
			},
		},
		{
			path:   "/api/v1/page-state/enterprise",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/EnterprisePageStateItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
		{
			path:   "/api/v1/identity-providers",
			method: "POST",
			fragments: []string{
				"$ref: '#/components/schemas/CreateIdentityProviderRequest'",
				"$ref: '#/components/schemas/IdentityProviderItemResponse'",
			},
		},
		{
			path:   "/api/v1/identity-providers/{id}",
			method: "PATCH",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/UpdateIdentityProviderRequest'",
				"$ref: '#/components/schemas/IdentityProviderItemResponse'",
			},
		},
		{
			path:   "/api/v1/identity-providers/{id}/test",
			method: "POST",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/IdentityProviderTestResultItemResponse'",
			},
		},
		{
			path:   "/api/v1/outbox-events",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/OutboxEventListResponse'",
				"name: event_type",
				"name: status",
			},
		},
		{
			path:   "/api/v1/outbox-events/{id}/retry",
			method: "POST",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/OutboxEventItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/outbox-events/{id}/requeue",
			method: "POST",
			fragments: []string{
				"name: id",
				"$ref: '#/components/schemas/OutboxEventItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
				"$ref: '#/components/responses/ValidationError'",
				"$ref: '#/components/responses/NotFoundError'",
			},
		},
		{
			path:   "/api/v1/page-state/simulation",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/SimulationPageStateItemResponse'",
				"$ref: '#/components/responses/UnauthorizedError'",
				"$ref: '#/components/responses/ForbiddenError'",
			},
		},
	}

	for _, section := range sections {
		requireOpenAPIFragments(t, openAPIMethodSection(t, content, section.path, section.method), section.fragments...)
	}
}

func TestOpenAPISystemAuthAndStatusRoutesUseDocumentedResponseSchemas(t *testing.T) {
	t.Parallel()

	content := openAPIContent(t)
	sections := []struct {
		path      string
		method    string
		fragments []string
	}{
		{
			path:   "/healthz",
			method: "GET",
			fragments: []string{
				"security: []",
				"$ref: '#/components/schemas/HealthResponseItemResponse'",
			},
		},
		{
			path:   "/readyz",
			method: "GET",
			fragments: []string{
				"security: []",
				"$ref: '#/components/schemas/HealthResponseItemResponse'",
			},
		},
		{
			path:   "/api/v1/auth/dev/login",
			method: "POST",
			fragments: []string{
				"security: []",
				"$ref: '#/components/schemas/DevLoginRequest'",
				"$ref: '#/components/schemas/AuthResponseItemResponse'",
			},
		},
		{
			path:   "/api/v1/auth/sign-up",
			method: "POST",
			fragments: []string{
				"security: []",
				"$ref: '#/components/schemas/SignUpRequest'",
				"$ref: '#/components/schemas/AuthResponseItemResponse'",
			},
		},
		{
			path:   "/api/v1/auth/sign-in",
			method: "POST",
			fragments: []string{
				"security: []",
				"$ref: '#/components/schemas/SignInRequest'",
				"$ref: '#/components/schemas/AuthResponseItemResponse'",
			},
		},
		{
			path:   "/api/v1/auth/providers",
			method: "GET",
			fragments: []string{
				"security: []",
				"$ref: '#/components/schemas/PublicIdentityProviderListResponse'",
			},
		},
		{
			path:   "/api/v1/auth/providers/{id}/start",
			method: "POST",
			fragments: []string{
				"security: []",
				"$ref: '#/components/schemas/IdentityProviderStartRequest'",
				"$ref: '#/components/schemas/IdentityProviderStartResultItemResponse'",
			},
		},
		{
			path:   "/api/v1/auth/providers/callback",
			method: "GET",
			fragments: []string{
				"security: []",
				"name: state",
				"'302':",
				"$ref: '#/components/schemas/AuthResponseItemResponse'",
			},
		},
		{
			path:   "/api/v1/auth/session",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/SessionInfoItemResponse'",
			},
		},
		{
			path:   "/api/v1/auth/logout",
			method: "POST",
			fragments: []string{
				"security: []",
				"$ref: '#/components/schemas/SessionInfoItemResponse'",
			},
		},
		{
			path:   "/api/v1/catalog",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/CatalogSummaryItemResponse'",
			},
		},
		{
			path:   "/api/v1/metrics/basics",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/BasicMetricsItemResponse'",
			},
		},
		{
			path:   "/api/v1/policies",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/PolicyListResponse'",
			},
		},
		{
			path:   "/api/v1/policies",
			method: "POST",
			fragments: []string{
				"$ref: '#/components/schemas/CreatePolicyRequest'",
				"$ref: '#/components/schemas/PolicyItemResponse'",
			},
		},
		{
			path:   "/api/v1/policies/{id}",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/PolicyItemResponse'",
			},
		},
		{
			path:   "/api/v1/policies/{id}",
			method: "PATCH",
			fragments: []string{
				"$ref: '#/components/schemas/UpdatePolicyRequest'",
				"$ref: '#/components/schemas/PolicyItemResponse'",
			},
		},
		{
			path:   "/api/v1/policy-decisions",
			method: "GET",
			fragments: []string{
				"name: policy_id",
				"name: risk_assessment_id",
				"name: rollout_plan_id",
				"name: applies_to",
				"$ref: '#/components/schemas/PolicyDecisionListResponse'",
			},
		},
		{
			path:   "/api/v1/status-events",
			method: "GET",
			fragments: []string{
				"name: rollback_only",
				"name: since",
				"name: until",
				"$ref: '#/components/schemas/StatusEventListResponse'",
			},
		},
		{
			path:   "/api/v1/status-events/search",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/StatusEventQueryResultItemResponse'",
			},
		},
		{
			path:   "/api/v1/status-events/{id}",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/StatusEventItemResponse'",
			},
		},
		{
			path:   "/api/v1/projects/{id}/status-events",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/StatusEventListResponse'",
			},
		},
		{
			path:   "/api/v1/services/{id}/status-events",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/StatusEventListResponse'",
			},
		},
		{
			path:   "/api/v1/environments/{id}/status-events",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/StatusEventListResponse'",
			},
		},
	}

	for _, section := range sections {
		requireOpenAPIFragments(t, openAPIMethodSection(t, content, section.path, section.method), section.fragments...)
	}
}

func TestOpenAPIMachineAuthRoutesUseDocumentedResponseSchemas(t *testing.T) {
	t.Parallel()

	content := openAPIContent(t)
	sections := []struct {
		path      string
		method    string
		fragments []string
	}{
		{
			path:   "/api/v1/service-accounts",
			method: "GET",
			fragments: []string{
				"$ref: '#/components/schemas/ServiceAccountListResponse'",
			},
		},
		{
			path:   "/api/v1/service-accounts/{id}/deactivate",
			method: "POST",
			fragments: []string{
				"$ref: '#/components/schemas/ServiceAccountItemResponse'",
			},
		},
		{
			path:   "/api/v1/service-accounts/{id}/tokens/{token_id}/revoke",
			method: "POST",
			fragments: []string{
				"$ref: '#/components/schemas/APITokenItemResponse'",
			},
		},
		{
			path:   "/api/v1/service-accounts/{id}/tokens/{token_id}/rotate",
			method: "POST",
			fragments: []string{
				"$ref: '#/components/schemas/RotateAPITokenRequest'",
				"$ref: '#/components/schemas/IssuedAPITokenItemResponse'",
			},
		},
	}

	for _, section := range sections {
		requireOpenAPIFragments(t, openAPIMethodSection(t, content, section.path, section.method), section.fragments...)
	}
}

type registeredRoute struct {
	Method string
	Path   string
}

func openAPIContent(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../docs/api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func registeredHTTPRoutes(t *testing.T) []registeredRoute {
	t.Helper()
	data, err := os.ReadFile("../../internal/app/http.go")
	if err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`HandleFunc\("([A-Z]+) ([^"]+)"`).FindAllStringSubmatch(string(data), -1)
	if len(matches) == 0 {
		t.Fatal("expected route registrations in http.go")
	}
	routes := make([]registeredRoute, 0, len(matches))
	for _, match := range matches {
		routes = append(routes, registeredRoute{Method: match[1], Path: match[2]})
	}
	return routes
}

func openAPIPathSection(content, path string) string {
	lines := strings.Split(content, "\n")
	target := "  " + path + ":"
	start := -1
	for index, line := range lines {
		if strings.TrimRight(line, " ") == target {
			start = index
			break
		}
	}
	if start == -1 {
		return ""
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		if strings.HasPrefix(lines[index], "  /") {
			end = index
			break
		}
	}
	return strings.Join(lines[start+1:end], "\n")
}

func openAPIMethodSection(t *testing.T, content, path, method string) string {
	t.Helper()
	pathSection := openAPIPathSection(content, path)
	if pathSection == "" {
		t.Fatalf("expected openapi path section for %s", path)
	}
	lines := strings.Split(pathSection, "\n")
	target := "    " + strings.ToLower(method) + ":"
	start := -1
	for index, line := range lines {
		if strings.TrimRight(line, " ") == target {
			start = index
			break
		}
	}
	if start == -1 {
		return ""
	}
	end := len(lines)
	for index := start + 1; index < len(lines); index++ {
		line := lines[index]
		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "      ") && strings.HasSuffix(strings.TrimSpace(line), ":") {
			end = index
			break
		}
	}
	return strings.Join(lines[start+1:end], "\n")
}

func requireOpenAPIFragments(t *testing.T, section string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(section, fragment) {
			t.Fatalf("expected openapi section to include %q", fragment)
		}
	}
}
