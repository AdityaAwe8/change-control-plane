package types

import "time"

type ItemResponse[T any] struct {
	Data T `json:"data"`
}

type ListResponse[T any] struct {
	Data []T `json:"data"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

type SessionInfo struct {
	Authenticated        bool                  `json:"authenticated"`
	Mode                 string                `json:"mode"`
	Actor                string                `json:"actor"`
	ActorID              string                `json:"actor_id,omitempty"`
	ActorType            string                `json:"actor_type,omitempty"`
	AuthMethod           string                `json:"auth_method,omitempty"`
	AuthProviderID       string                `json:"auth_provider_id,omitempty"`
	AuthProvider         string                `json:"auth_provider,omitempty"`
	IssuedAt             string                `json:"issued_at,omitempty"`
	ExpiresAt            string                `json:"expires_at,omitempty"`
	Email                string                `json:"email,omitempty"`
	DisplayName          string                `json:"display_name,omitempty"`
	ActiveOrganizationID string                `json:"active_organization_id,omitempty"`
	Organizations        []SessionOrganization `json:"organizations,omitempty"`
	ProjectMemberships   []SessionProjectScope `json:"project_memberships,omitempty"`
}

type BrowserSessionInfo struct {
	BaseRecord
	UserID          string     `json:"user_id"`
	UserEmail       string     `json:"user_email,omitempty"`
	UserDisplayName string     `json:"user_display_name,omitempty"`
	AuthMethod      string     `json:"auth_method,omitempty"`
	AuthProviderID  string     `json:"auth_provider_id,omitempty"`
	AuthProvider    string     `json:"auth_provider,omitempty"`
	LastSeenAt      *time.Time `json:"last_seen_at,omitempty"`
	ExpiresAt       time.Time  `json:"expires_at"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	Status          string     `json:"status"`
	Current         bool       `json:"current,omitempty"`
}

type SessionOrganization struct {
	OrganizationID string `json:"organization_id"`
	Organization   string `json:"organization"`
	Role           string `json:"role"`
}

type SessionProjectScope struct {
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	Project        string `json:"project"`
	Role           string `json:"role"`
}

type SignUpRequest struct {
	Email                string `json:"email"`
	DisplayName          string `json:"display_name"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token   string      `json:"token"`
	Session SessionInfo `json:"session"`
}

type CatalogSummary struct {
	Services     []Service     `json:"services"`
	Environments []Environment `json:"environments"`
}

type BasicMetrics struct {
	Organizations   int `json:"organizations"`
	Projects        int `json:"projects"`
	Teams           int `json:"teams"`
	Services        int `json:"services"`
	Environments    int `json:"environments"`
	Changes         int `json:"changes"`
	RiskAssessments int `json:"risk_assessments"`
	RolloutPlans    int `json:"rollout_plans"`
	AuditEvents     int `json:"audit_events"`
	Policies        int `json:"policies"`
	Integrations    int `json:"integrations"`
}

type CreateOrganizationRequest struct {
	Name     string   `json:"name"`
	Slug     string   `json:"slug"`
	Tier     string   `json:"tier"`
	Mode     string   `json:"mode"`
	Metadata Metadata `json:"metadata,omitempty"`
}

type UpdateOrganizationRequest struct {
	Name     *string  `json:"name,omitempty"`
	Tier     *string  `json:"tier,omitempty"`
	Mode     *string  `json:"mode,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

type CreateProjectRequest struct {
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	Slug           string   `json:"slug"`
	Description    string   `json:"description"`
	AdoptionMode   string   `json:"adoption_mode"`
	Metadata       Metadata `json:"metadata,omitempty"`
}

type UpdateProjectRequest struct {
	Name         *string  `json:"name,omitempty"`
	Slug         *string  `json:"slug,omitempty"`
	Description  *string  `json:"description,omitempty"`
	AdoptionMode *string  `json:"adoption_mode,omitempty"`
	Status       *string  `json:"status,omitempty"`
	Metadata     Metadata `json:"metadata,omitempty"`
}

type CreateTeamRequest struct {
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	Name           string   `json:"name"`
	Slug           string   `json:"slug"`
	OwnerUserIDs   []string `json:"owner_user_ids,omitempty"`
	Metadata       Metadata `json:"metadata,omitempty"`
}

type UpdateTeamRequest struct {
	Name         *string   `json:"name,omitempty"`
	Slug         *string   `json:"slug,omitempty"`
	OwnerUserIDs *[]string `json:"owner_user_ids,omitempty"`
	Status       *string   `json:"status,omitempty"`
	Metadata     Metadata  `json:"metadata,omitempty"`
}

type CreateServiceRequest struct {
	OrganizationID         string   `json:"organization_id"`
	ProjectID              string   `json:"project_id"`
	TeamID                 string   `json:"team_id"`
	Name                   string   `json:"name"`
	Slug                   string   `json:"slug"`
	Description            string   `json:"description"`
	Criticality            string   `json:"criticality"`
	Tier                   string   `json:"tier"`
	CustomerFacing         bool     `json:"customer_facing"`
	HasSLO                 bool     `json:"has_slo"`
	HasObservability       bool     `json:"has_observability"`
	RegulatedZone          bool     `json:"regulated_zone"`
	DependentServicesCount int      `json:"dependent_services_count"`
	Metadata               Metadata `json:"metadata,omitempty"`
}

type UpdateServiceRequest struct {
	Name                   *string  `json:"name,omitempty"`
	Slug                   *string  `json:"slug,omitempty"`
	Description            *string  `json:"description,omitempty"`
	Criticality            *string  `json:"criticality,omitempty"`
	Tier                   *string  `json:"tier,omitempty"`
	CustomerFacing         *bool    `json:"customer_facing,omitempty"`
	HasSLO                 *bool    `json:"has_slo,omitempty"`
	HasObservability       *bool    `json:"has_observability,omitempty"`
	RegulatedZone          *bool    `json:"regulated_zone,omitempty"`
	DependentServicesCount *int     `json:"dependent_services_count,omitempty"`
	Status                 *string  `json:"status,omitempty"`
	Metadata               Metadata `json:"metadata,omitempty"`
}

type CreateEnvironmentRequest struct {
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	Name           string   `json:"name"`
	Slug           string   `json:"slug"`
	Type           string   `json:"type"`
	Region         string   `json:"region"`
	Production     bool     `json:"production"`
	ComplianceZone string   `json:"compliance_zone"`
	Metadata       Metadata `json:"metadata,omitempty"`
}

type UpdateEnvironmentRequest struct {
	Name           *string  `json:"name,omitempty"`
	Slug           *string  `json:"slug,omitempty"`
	Type           *string  `json:"type,omitempty"`
	Region         *string  `json:"region,omitempty"`
	Production     *bool    `json:"production,omitempty"`
	ComplianceZone *string  `json:"compliance_zone,omitempty"`
	Status         *string  `json:"status,omitempty"`
	Metadata       Metadata `json:"metadata,omitempty"`
}

type CreateChangeSetRequest struct {
	OrganizationID          string   `json:"organization_id"`
	ProjectID               string   `json:"project_id"`
	ServiceID               string   `json:"service_id"`
	EnvironmentID           string   `json:"environment_id"`
	Summary                 string   `json:"summary"`
	ChangeTypes             []string `json:"change_types"`
	FileCount               int      `json:"file_count"`
	ResourceCount           int      `json:"resource_count"`
	TouchesInfrastructure   bool     `json:"touches_infrastructure"`
	TouchesIAM              bool     `json:"touches_iam"`
	TouchesSecrets          bool     `json:"touches_secrets"`
	TouchesSchema           bool     `json:"touches_schema"`
	DependencyChanges       bool     `json:"dependency_changes"`
	HistoricalIncidentCount int      `json:"historical_incident_count"`
	PoorRollbackHistory     bool     `json:"poor_rollback_history"`
	Metadata                Metadata `json:"metadata,omitempty"`
}

type CreateRiskAssessmentRequest struct {
	ChangeSetID string `json:"change_set_id"`
}

type CreateRolloutPlanRequest struct {
	ChangeSetID string `json:"change_set_id"`
}

type CreateRolloutExecutionRequest struct {
	RolloutPlanID        string `json:"rollout_plan_id"`
	BackendType          string `json:"backend_type,omitempty"`
	BackendIntegrationID string `json:"backend_integration_id,omitempty"`
	SignalProviderType   string `json:"signal_provider_type,omitempty"`
	SignalIntegrationID  string `json:"signal_integration_id,omitempty"`
}

type AdvanceRolloutExecutionRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type RecordVerificationResultRequest struct {
	Outcome                string   `json:"outcome"`
	Decision               string   `json:"decision"`
	Signals                []string `json:"signals,omitempty"`
	TechnicalSignalSummary Metadata `json:"technical_signal_summary,omitempty"`
	BusinessSignalSummary  Metadata `json:"business_signal_summary,omitempty"`
	Automated              bool     `json:"automated,omitempty"`
	DecisionSource         string   `json:"decision_source,omitempty"`
	SignalSnapshotIDs      []string `json:"signal_snapshot_ids,omitempty"`
	Summary                string   `json:"summary"`
	Explanation            []string `json:"explanation,omitempty"`
	Metadata               Metadata `json:"metadata,omitempty"`
}

type CreateSignalSnapshotRequest struct {
	ProviderType        string        `json:"provider_type,omitempty"`
	SourceIntegrationID string        `json:"source_integration_id,omitempty"`
	Health              string        `json:"health"`
	Summary             string        `json:"summary"`
	Signals             []SignalValue `json:"signals"`
	WindowSeconds       int           `json:"window_seconds,omitempty"`
	Explanation         []string      `json:"explanation,omitempty"`
	Metadata            Metadata      `json:"metadata,omitempty"`
}

type CreateServiceAccountRequest struct {
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Role           string   `json:"role"`
	Metadata       Metadata `json:"metadata,omitempty"`
}

type UpdateServiceAccountRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Role        *string  `json:"role,omitempty"`
	Status      *string  `json:"status,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
}

type IssueAPITokenRequest struct {
	Name           string `json:"name"`
	ExpiresInHours int    `json:"expires_in_hours,omitempty"`
}

type RotateAPITokenRequest struct {
	Name           string `json:"name,omitempty"`
	ExpiresInHours int    `json:"expires_in_hours,omitempty"`
}

type IssuedAPITokenResponse struct {
	Token string   `json:"token"`
	Entry APIToken `json:"entry"`
}

type UpdateIntegrationRequest struct {
	Name                    *string   `json:"name,omitempty"`
	InstanceKey             *string   `json:"instance_key,omitempty"`
	ScopeType               *string   `json:"scope_type,omitempty"`
	ScopeName               *string   `json:"scope_name,omitempty"`
	Mode                    *string   `json:"mode,omitempty"`
	AuthStrategy            *string   `json:"auth_strategy,omitempty"`
	OnboardingStatus        *string   `json:"onboarding_status,omitempty"`
	Status                  *string   `json:"status,omitempty"`
	Enabled                 *bool     `json:"enabled,omitempty"`
	ControlEnabled          *bool     `json:"control_enabled,omitempty"`
	ScheduleEnabled         *bool     `json:"schedule_enabled,omitempty"`
	ScheduleIntervalSeconds *int      `json:"schedule_interval_seconds,omitempty"`
	SyncStaleAfterSeconds   *int      `json:"sync_stale_after_seconds,omitempty"`
	Description             *string   `json:"description,omitempty"`
	Capabilities            *[]string `json:"capabilities,omitempty"`
	Metadata                Metadata  `json:"metadata,omitempty"`
}

type CreateIntegrationRequest struct {
	OrganizationID          string   `json:"organization_id"`
	Kind                    string   `json:"kind"`
	Name                    string   `json:"name"`
	InstanceKey             string   `json:"instance_key,omitempty"`
	ScopeType               string   `json:"scope_type,omitempty"`
	ScopeName               string   `json:"scope_name,omitempty"`
	Mode                    string   `json:"mode,omitempty"`
	AuthStrategy            string   `json:"auth_strategy,omitempty"`
	Enabled                 bool     `json:"enabled,omitempty"`
	ControlEnabled          bool     `json:"control_enabled,omitempty"`
	ScheduleEnabled         bool     `json:"schedule_enabled,omitempty"`
	ScheduleIntervalSeconds int      `json:"schedule_interval_seconds,omitempty"`
	SyncStaleAfterSeconds   int      `json:"sync_stale_after_seconds,omitempty"`
	Description             string   `json:"description,omitempty"`
	Metadata                Metadata `json:"metadata,omitempty"`
}

type GitHubOnboardingStartResult struct {
	Integration  Integration `json:"integration"`
	AuthorizeURL string      `json:"authorize_url"`
	CallbackURL  string      `json:"callback_url"`
	ExpiresAt    string      `json:"expires_at"`
	Strategy     string      `json:"strategy"`
	StatePreview string      `json:"state_preview,omitempty"`
}

type CreateIdentityProviderRequest struct {
	OrganizationID        string   `json:"organization_id"`
	Name                  string   `json:"name"`
	Kind                  string   `json:"kind"`
	IssuerURL             string   `json:"issuer_url,omitempty"`
	AuthorizationEndpoint string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string   `json:"token_endpoint,omitempty"`
	UserInfoEndpoint      string   `json:"userinfo_endpoint,omitempty"`
	JWKSURI               string   `json:"jwks_uri,omitempty"`
	ClientID              string   `json:"client_id,omitempty"`
	ClientSecretEnv       string   `json:"client_secret_env,omitempty"`
	Scopes                []string `json:"scopes,omitempty"`
	ClaimMappings         Metadata `json:"claim_mappings,omitempty"`
	RoleMappings          Metadata `json:"role_mappings,omitempty"`
	AllowedDomains        []string `json:"allowed_domains,omitempty"`
	DefaultRole           string   `json:"default_role,omitempty"`
	Enabled               bool     `json:"enabled,omitempty"`
	Metadata              Metadata `json:"metadata,omitempty"`
}

type UpdateIdentityProviderRequest struct {
	Name                  *string   `json:"name,omitempty"`
	IssuerURL             *string   `json:"issuer_url,omitempty"`
	AuthorizationEndpoint *string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         *string   `json:"token_endpoint,omitempty"`
	UserInfoEndpoint      *string   `json:"userinfo_endpoint,omitempty"`
	JWKSURI               *string   `json:"jwks_uri,omitempty"`
	ClientID              *string   `json:"client_id,omitempty"`
	ClientSecretEnv       *string   `json:"client_secret_env,omitempty"`
	Scopes                *[]string `json:"scopes,omitempty"`
	ClaimMappings         Metadata  `json:"claim_mappings,omitempty"`
	RoleMappings          Metadata  `json:"role_mappings,omitempty"`
	AllowedDomains        *[]string `json:"allowed_domains,omitempty"`
	DefaultRole           *string   `json:"default_role,omitempty"`
	Enabled               *bool     `json:"enabled,omitempty"`
	Status                *string   `json:"status,omitempty"`
	Metadata              Metadata  `json:"metadata,omitempty"`
}

type PublicIdentityProvider struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
}

type IdentityProviderStartRequest struct {
	ReturnTo string `json:"return_to,omitempty"`
}

type IdentityProviderStartResult struct {
	Provider     IdentityProvider `json:"provider"`
	AuthorizeURL string           `json:"authorize_url"`
	CallbackURL  string           `json:"callback_url"`
	ExpiresAt    string           `json:"expires_at"`
	Strategy     string           `json:"strategy"`
	StatePreview string           `json:"state_preview,omitempty"`
}

type IdentityProviderTestResult struct {
	Provider IdentityProvider `json:"provider"`
	Status   string           `json:"status"`
	Details  []string         `json:"details,omitempty"`
}

type WebhookRegistrationResult struct {
	Registration WebhookRegistration `json:"registration"`
	Details      []string            `json:"details,omitempty"`
}

type UpdateRepositoryRequest struct {
	ProjectID     *string  `json:"project_id,omitempty"`
	ServiceID     *string  `json:"service_id,omitempty"`
	EnvironmentID *string  `json:"environment_id,omitempty"`
	Name          *string  `json:"name,omitempty"`
	DefaultBranch *string  `json:"default_branch,omitempty"`
	Status        *string  `json:"status,omitempty"`
	Metadata      Metadata `json:"metadata,omitempty"`
}

type UpdateDiscoveredResourceRequest struct {
	ProjectID     *string  `json:"project_id,omitempty"`
	ServiceID     *string  `json:"service_id,omitempty"`
	EnvironmentID *string  `json:"environment_id,omitempty"`
	RepositoryID  *string  `json:"repository_id,omitempty"`
	Status        *string  `json:"status,omitempty"`
	Metadata      Metadata `json:"metadata,omitempty"`
}

type IntegrationRepositoryInput struct {
	ProjectID     string   `json:"project_id,omitempty"`
	ServiceID     string   `json:"service_id,omitempty"`
	Name          string   `json:"name"`
	Provider      string   `json:"provider"`
	URL           string   `json:"url"`
	DefaultBranch string   `json:"default_branch"`
	Metadata      Metadata `json:"metadata,omitempty"`
}

type ServiceDependencyInput struct {
	ServiceID          string `json:"service_id"`
	DependsOnServiceID string `json:"depends_on_service_id"`
	CriticalDependency bool   `json:"critical_dependency"`
}

type ServiceEnvironmentBindingInput struct {
	ServiceID     string `json:"service_id"`
	EnvironmentID string `json:"environment_id"`
}

type ChangeRepositoryBindingInput struct {
	ChangeSetID   string `json:"change_set_id"`
	RepositoryURL string `json:"repository_url"`
}

type IntegrationGraphIngestRequest struct {
	Repositories        []IntegrationRepositoryInput     `json:"repositories,omitempty"`
	ServiceDependencies []ServiceDependencyInput         `json:"service_dependencies,omitempty"`
	ServiceEnvironments []ServiceEnvironmentBindingInput `json:"service_environments,omitempty"`
	ChangeRepositories  []ChangeRepositoryBindingInput   `json:"change_repositories,omitempty"`
}

type IntegrationTestResult struct {
	Integration Integration        `json:"integration"`
	Run         IntegrationSyncRun `json:"run"`
}

type IntegrationSyncResult struct {
	Integration         Integration          `json:"integration"`
	Run                 IntegrationSyncRun   `json:"run"`
	Repositories        []Repository         `json:"repositories,omitempty"`
	DiscoveredResources []DiscoveredResource `json:"discovered_resources,omitempty"`
	Relationships       []GraphRelationship  `json:"relationships,omitempty"`
}

type StatusEventQuerySummary struct {
	Total           int     `json:"total"`
	Returned        int     `json:"returned"`
	Limit           int     `json:"limit"`
	Offset          int     `json:"offset"`
	RollbackEvents  int     `json:"rollback_events"`
	AutomatedEvents int     `json:"automated_events"`
	LatestEventAt   *string `json:"latest_event_at,omitempty"`
	OldestEventAt   *string `json:"oldest_event_at,omitempty"`
}

type StatusEventQueryResult struct {
	Events  []StatusEvent           `json:"events"`
	Summary StatusEventQuerySummary `json:"summary"`
	Filters Metadata                `json:"filters,omitempty"`
}

type CoverageSummary struct {
	EnabledIntegrations          int `json:"enabled_integrations"`
	StaleIntegrations            int `json:"stale_integrations"`
	HealthyIntegrations          int `json:"healthy_integrations"`
	GitHubIntegrations           int `json:"github_integrations,omitempty"`
	GitLabIntegrations           int `json:"gitlab_integrations,omitempty"`
	KubernetesIntegrations       int `json:"kubernetes_integrations,omitempty"`
	PrometheusIntegrations       int `json:"prometheus_integrations,omitempty"`
	Repositories                 int `json:"repositories"`
	UnmappedRepositories         int `json:"unmapped_repositories"`
	DiscoveredResources          int `json:"discovered_resources"`
	UnmappedDiscoveredResources  int `json:"unmapped_discovered_resources"`
	WorkloadCoverageEnvironments int `json:"workload_coverage_environments"`
	SignalCoverageServices       int `json:"signal_coverage_services"`
}

type RolloutPageState struct {
	RolloutPlans           []RolloutPlan           `json:"rollout_plans"`
	RolloutExecutions      []RolloutExecution      `json:"rollout_executions"`
	RolloutExecutionDetail *RolloutExecutionDetail `json:"rollout_execution_detail,omitempty"`
	Integrations           []Integration           `json:"integrations"`
}

type DeploymentsPageState struct {
	Catalog          CatalogSummary         `json:"catalog"`
	RollbackPolicies []RollbackPolicy       `json:"rollback_policies"`
	StatusDashboard  StatusEventQueryResult `json:"status_dashboard"`
	CoverageSummary  CoverageSummary        `json:"coverage_summary"`
}

type IntegrationsPageState struct {
	Catalog              CatalogSummary                  `json:"catalog"`
	Teams                []Team                          `json:"teams"`
	Integrations         []Integration                   `json:"integrations"`
	CoverageSummary      CoverageSummary                 `json:"coverage_summary"`
	Repositories         []Repository                    `json:"repositories"`
	DiscoveredResources  []DiscoveredResource            `json:"discovered_resources"`
	IntegrationSyncRuns  map[string][]IntegrationSyncRun `json:"integration_sync_runs"`
	WebhookRegistrations map[string]*WebhookRegistration `json:"webhook_registrations"`
}

type EnterprisePageState struct {
	IdentityProviders    []IdentityProvider              `json:"identity_providers"`
	Integrations         []Integration                   `json:"integrations"`
	WebhookRegistrations map[string]*WebhookRegistration `json:"webhook_registrations"`
	OutboxEvents         []OutboxEvent                   `json:"outbox_events"`
	BrowserSessions      []BrowserSessionInfo            `json:"browser_sessions"`
}

type GraphPageState struct {
	GraphRelationships  []GraphRelationship  `json:"graph_relationships"`
	Catalog             CatalogSummary       `json:"catalog"`
	Integrations        []Integration        `json:"integrations"`
	Projects            []Project            `json:"projects"`
	Teams               []Team               `json:"teams"`
	Repositories        []Repository         `json:"repositories"`
	DiscoveredResources []DiscoveredResource `json:"discovered_resources"`
	Changes             []ChangeSet          `json:"changes"`
}

type SimulationPageState struct {
	Changes                []ChangeSet             `json:"changes"`
	RiskAssessments        []RiskAssessment        `json:"risk_assessments"`
	RolloutPlans           []RolloutPlan           `json:"rollout_plans"`
	RolloutExecutions      []RolloutExecution      `json:"rollout_executions"`
	RolloutExecutionDetail *RolloutExecutionDetail `json:"rollout_execution_detail,omitempty"`
	RollbackPolicies       []RollbackPolicy        `json:"rollback_policies"`
	StatusEvents           []StatusEvent           `json:"status_events"`
}

type DevLoginRequest struct {
	Email            string   `json:"email"`
	DisplayName      string   `json:"display_name"`
	OrganizationName string   `json:"organization_name,omitempty"`
	OrganizationSlug string   `json:"organization_slug,omitempty"`
	Roles            []string `json:"roles,omitempty"`
}

type DevLoginResponse = AuthResponse

type RiskAssessmentResult struct {
	Assessment      RiskAssessment   `json:"assessment"`
	PolicyDecisions []PolicyDecision `json:"policy_decisions"`
}

type RolloutPlanResult struct {
	Assessment      RiskAssessment   `json:"assessment"`
	Plan            RolloutPlan      `json:"plan"`
	PolicyDecisions []PolicyDecision `json:"policy_decisions"`
}

type RolloutExecutionDetail struct {
	Execution               RolloutExecution               `json:"execution"`
	VerificationResults     []VerificationResult           `json:"verification_results"`
	SignalSnapshots         []SignalSnapshot               `json:"signal_snapshots"`
	Timeline                []AuditEvent                   `json:"timeline"`
	StatusTimeline          []StatusEvent                  `json:"status_timeline"`
	EffectiveRollbackPolicy *RollbackPolicy                `json:"effective_rollback_policy,omitempty"`
	RuntimeSummary          RolloutExecutionRuntimeSummary `json:"runtime_summary"`
}

type RolloutEvidencePackSummary struct {
	GeneratedAt               time.Time `json:"generated_at"`
	ApprovalState             string    `json:"approval_state"`
	RiskLevel                 RiskLevel `json:"risk_level"`
	RiskScore                 int       `json:"risk_score"`
	BlastRadiusScope          string    `json:"blast_radius_scope"`
	BlastRadiusSummary        string    `json:"blast_radius_summary"`
	RolloutStrategy           string    `json:"rollout_strategy"`
	LatestDecision            string    `json:"latest_decision,omitempty"`
	LatestVerificationOutcome string    `json:"latest_verification_outcome,omitempty"`
	ControlMode               string    `json:"control_mode,omitempty"`
	IncidentCount             int       `json:"incident_count"`
	RepositoryCount           int       `json:"repository_count"`
	DiscoveredResourceCount   int       `json:"discovered_resource_count"`
	BlockingPolicyCount       int       `json:"blocking_policy_count"`
	ManualReviewPolicyCount   int       `json:"manual_review_policy_count"`
	EvidenceHighlights        []string  `json:"evidence_highlights,omitempty"`
}

type RolloutEvidencePack struct {
	Summary             RolloutEvidencePackSummary `json:"summary"`
	Organization        Organization               `json:"organization"`
	Project             Project                    `json:"project"`
	Service             Service                    `json:"service"`
	Environment         Environment                `json:"environment"`
	ChangeSet           ChangeSet                  `json:"change_set"`
	Assessment          RiskAssessment             `json:"assessment"`
	Plan                RolloutPlan                `json:"plan"`
	ExecutionDetail     RolloutExecutionDetail     `json:"execution_detail"`
	BackendIntegration  *Integration               `json:"backend_integration,omitempty"`
	SignalIntegration   *Integration               `json:"signal_integration,omitempty"`
	PolicyDecisions     []PolicyDecision           `json:"policy_decisions,omitempty"`
	Incidents           []Incident                 `json:"incidents,omitempty"`
	Repositories        []Repository               `json:"repositories,omitempty"`
	DiscoveredResources []DiscoveredResource       `json:"discovered_resources,omitempty"`
	GraphRelationships  []GraphRelationship        `json:"graph_relationships,omitempty"`
	AuditTrail          []AuditEvent               `json:"audit_trail,omitempty"`
}
