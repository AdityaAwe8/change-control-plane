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

type CreateConfigSetRequest struct {
	OrganizationID string        `json:"organization_id"`
	ProjectID      string        `json:"project_id"`
	EnvironmentID  string        `json:"environment_id"`
	ServiceID      string        `json:"service_id,omitempty"`
	Name           string        `json:"name"`
	Version        string        `json:"version"`
	Entries        []ConfigEntry `json:"entries"`
	Metadata       Metadata      `json:"metadata,omitempty"`
}

type UpdateConfigSetRequest struct {
	Name     *string        `json:"name,omitempty"`
	Version  *string        `json:"version,omitempty"`
	Status   *string        `json:"status,omitempty"`
	Entries  *[]ConfigEntry `json:"entries,omitempty"`
	Metadata Metadata       `json:"metadata,omitempty"`
}

type CreateReleaseRequest struct {
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	EnvironmentID  string   `json:"environment_id"`
	Name           string   `json:"name"`
	Summary        string   `json:"summary"`
	ChangeSetIDs   []string `json:"change_set_ids"`
	ConfigSetIDs   []string `json:"config_set_ids,omitempty"`
	Version        string   `json:"version"`
	Metadata       Metadata `json:"metadata,omitempty"`
}

type UpdateReleaseRequest struct {
	Name         *string   `json:"name,omitempty"`
	Summary      *string   `json:"summary,omitempty"`
	ChangeSetIDs *[]string `json:"change_set_ids,omitempty"`
	ConfigSetIDs *[]string `json:"config_set_ids,omitempty"`
	Version      *string   `json:"version,omitempty"`
	Status       *string   `json:"status,omitempty"`
	Metadata     Metadata  `json:"metadata,omitempty"`
}

type CreateDatabaseChangeRequest struct {
	OrganizationID         string    `json:"organization_id"`
	ProjectID              string    `json:"project_id"`
	EnvironmentID          string    `json:"environment_id"`
	ServiceID              string    `json:"service_id,omitempty"`
	ChangeSetID            string    `json:"change_set_id"`
	Name                   string    `json:"name"`
	Datastore              string    `json:"datastore"`
	OperationType          string    `json:"operation_type"`
	ExecutionIntent        string    `json:"execution_intent"`
	Compatibility          string    `json:"compatibility"`
	Reversibility          string    `json:"reversibility"`
	RiskLevel              RiskLevel `json:"risk_level"`
	LockRisk               bool      `json:"lock_risk,omitempty"`
	ManualApprovalRequired bool      `json:"manual_approval_required,omitempty"`
	Summary                string    `json:"summary"`
	Evidence               []string  `json:"evidence,omitempty"`
	Metadata               Metadata  `json:"metadata,omitempty"`
}

type UpdateDatabaseChangeRequest struct {
	Name                   *string    `json:"name,omitempty"`
	Datastore              *string    `json:"datastore,omitempty"`
	OperationType          *string    `json:"operation_type,omitempty"`
	ExecutionIntent        *string    `json:"execution_intent,omitempty"`
	Compatibility          *string    `json:"compatibility,omitempty"`
	Reversibility          *string    `json:"reversibility,omitempty"`
	RiskLevel              *RiskLevel `json:"risk_level,omitempty"`
	LockRisk               *bool      `json:"lock_risk,omitempty"`
	ManualApprovalRequired *bool      `json:"manual_approval_required,omitempty"`
	Status                 *string    `json:"status,omitempty"`
	Summary                *string    `json:"summary,omitempty"`
	Evidence               *[]string  `json:"evidence,omitempty"`
	Metadata               Metadata   `json:"metadata,omitempty"`
}

type CreateDatabaseValidationCheckRequest struct {
	OrganizationID   string   `json:"organization_id"`
	ProjectID        string   `json:"project_id"`
	EnvironmentID    string   `json:"environment_id"`
	ServiceID        string   `json:"service_id,omitempty"`
	ChangeSetID      string   `json:"change_set_id"`
	DatabaseChangeID string   `json:"database_change_id,omitempty"`
	ConnectionRefID  string   `json:"connection_ref_id,omitempty"`
	Name             string   `json:"name"`
	Phase            string   `json:"phase"`
	CheckType        string   `json:"check_type"`
	ReadOnly         bool     `json:"read_only"`
	Required         bool     `json:"required"`
	ExecutionMode    string   `json:"execution_mode"`
	Specification    string   `json:"specification"`
	Status           string   `json:"status,omitempty"`
	Summary          string   `json:"summary"`
	Evidence         []string `json:"evidence,omitempty"`
	Metadata         Metadata `json:"metadata,omitempty"`
}

type UpdateDatabaseValidationCheckRequest struct {
	DatabaseChangeID  *string    `json:"database_change_id,omitempty"`
	ConnectionRefID   *string    `json:"connection_ref_id,omitempty"`
	Name              *string    `json:"name,omitempty"`
	Phase             *string    `json:"phase,omitempty"`
	CheckType         *string    `json:"check_type,omitempty"`
	ReadOnly          *bool      `json:"read_only,omitempty"`
	Required          *bool      `json:"required,omitempty"`
	ExecutionMode     *string    `json:"execution_mode,omitempty"`
	Specification     *string    `json:"specification,omitempty"`
	Status            *string    `json:"status,omitempty"`
	Summary           *string    `json:"summary,omitempty"`
	LastRunAt         *time.Time `json:"last_run_at,omitempty"`
	LastResultSummary *string    `json:"last_result_summary,omitempty"`
	Evidence          *[]string  `json:"evidence,omitempty"`
	Metadata          Metadata   `json:"metadata,omitempty"`
}

type CreateDatabaseConnectionReferenceRequest struct {
	OrganizationID  string   `json:"organization_id"`
	ProjectID       string   `json:"project_id"`
	EnvironmentID   string   `json:"environment_id"`
	ServiceID       string   `json:"service_id,omitempty"`
	Name            string   `json:"name"`
	Datastore       string   `json:"datastore"`
	Driver          string   `json:"driver"`
	SourceType      string   `json:"source_type,omitempty"`
	DSNEnv          string   `json:"dsn_env,omitempty"`
	SecretRef       string   `json:"secret_ref,omitempty"`
	SecretRefEnv    string   `json:"secret_ref_env,omitempty"`
	ReadOnlyCapable bool     `json:"read_only_capable,omitempty"`
	Summary         string   `json:"summary"`
	Metadata        Metadata `json:"metadata,omitempty"`
}

type UpdateDatabaseConnectionReferenceRequest struct {
	Name            *string  `json:"name,omitempty"`
	Datastore       *string  `json:"datastore,omitempty"`
	Driver          *string  `json:"driver,omitempty"`
	SourceType      *string  `json:"source_type,omitempty"`
	DSNEnv          *string  `json:"dsn_env,omitempty"`
	SecretRef       *string  `json:"secret_ref,omitempty"`
	SecretRefEnv    *string  `json:"secret_ref_env,omitempty"`
	ReadOnlyCapable *bool    `json:"read_only_capable,omitempty"`
	Summary         *string  `json:"summary,omitempty"`
	Metadata        Metadata `json:"metadata,omitempty"`
}

type TestDatabaseConnectionReferenceRequest struct {
	Trigger  string   `json:"trigger,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

type ExecuteDatabaseValidationCheckRequest struct {
	Trigger  string   `json:"trigger,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

type CreateRolloutExecutionRequest struct {
	RolloutPlanID        string `json:"rollout_plan_id"`
	ReleaseID            string `json:"release_id,omitempty"`
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
	Catalog                CatalogSummary            `json:"catalog"`
	RolloutPlans           []RolloutPlan             `json:"rollout_plans"`
	RolloutExecutions      []RolloutExecution        `json:"rollout_executions"`
	RolloutExecutionDetail *RolloutExecutionDetail   `json:"rollout_execution_detail,omitempty"`
	Integrations           []Integration             `json:"integrations"`
	Releases               []Release                 `json:"releases,omitempty"`
	ReleaseAnalysis        *ReleaseAnalysis          `json:"release_analysis,omitempty"`
	ConfigSets             []ConfigSet               `json:"config_sets,omitempty"`
	DatabaseConnections    []DatabaseConnectionReference `json:"database_connections,omitempty"`
	DatabaseConnectionTests []DatabaseConnectionTest `json:"database_connection_tests,omitempty"`
	DatabaseChanges        []DatabaseChange          `json:"database_changes,omitempty"`
	DatabaseChecks         []DatabaseValidationCheck `json:"database_checks,omitempty"`
	DatabaseExecutions     []DatabaseValidationExecution `json:"database_executions,omitempty"`
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

type ConfigSetValidation struct {
	ConfigSetID         string   `json:"config_set_id"`
	Status              string   `json:"status"`
	MissingRequiredKeys []string `json:"missing_required_keys,omitempty"`
	DeprecatedKeys      []string `json:"deprecated_keys,omitempty"`
	InvalidSecretRefs   []string `json:"invalid_secret_refs,omitempty"`
	SecretReferenceKeys []string `json:"secret_reference_keys,omitempty"`
	DiffSummary         []string `json:"diff_summary,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
}

type ReadinessReviewItem struct {
	Severity               string   `json:"severity"`
	Category               string   `json:"category"`
	Question               string   `json:"question"`
	Reason                 string   `json:"reason"`
	Evidence               []string `json:"evidence,omitempty"`
	AcknowledgmentRequired bool     `json:"acknowledgment_required,omitempty"`
}

type ReleaseDependency struct {
	ServiceID            string `json:"service_id"`
	ServiceName          string `json:"service_name"`
	DependsOnServiceID   string `json:"depends_on_service_id"`
	DependsOnServiceName string `json:"depends_on_service_name"`
	Critical             bool   `json:"critical"`
	Summary              string `json:"summary"`
}

type RollbackGuidance struct {
	Safe     bool     `json:"safe"`
	Strategy string   `json:"strategy"`
	Summary  string   `json:"summary"`
	Steps    []string `json:"steps,omitempty"`
	Blockers []string `json:"blockers,omitempty"`
}

type OpsAssistantSummary struct {
	Status            string   `json:"status"`
	LikelyCause       string   `json:"likely_cause"`
	SuspiciousChanges []string `json:"suspicious_changes,omitempty"`
	Guidance          []string `json:"guidance,omitempty"`
}

type TeamMemoryInsight struct {
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Evidence []string `json:"evidence,omitempty"`
}

type CommunicationDrafts struct {
	ReleaseNotes      string `json:"release_notes,omitempty"`
	ApproverSummary   string `json:"approver_summary,omitempty"`
	StakeholderUpdate string `json:"stakeholder_update,omitempty"`
	MaintenanceNotice string `json:"maintenance_notice,omitempty"`
	IncidentHandoff   string `json:"incident_handoff,omitempty"`
	PostmortemStarter string `json:"postmortem_starter,omitempty"`
}

type ConfigSetDetail struct {
	ConfigSet       ConfigSet           `json:"config_set"`
	Validation      ConfigSetValidation `json:"validation"`
	RelatedReleases []Release           `json:"related_releases,omitempty"`
}

type DatabaseChangeDetail struct {
	DatabaseChange   DatabaseChange            `json:"database_change"`
	ValidationChecks []DatabaseValidationCheck `json:"validation_checks,omitempty"`
}

type DatabaseConnectionReferenceDetail struct {
	ConnectionReference DatabaseConnectionReference `json:"connection_reference"`
	ValidationChecks    []DatabaseValidationCheck   `json:"validation_checks,omitempty"`
	ConnectionTests     []DatabaseConnectionTest    `json:"connection_tests,omitempty"`
}

type DatabaseValidationCheckDetail struct {
	ValidationCheck    DatabaseValidationCheck       `json:"validation_check"`
	DatabaseChange     *DatabaseChange               `json:"database_change,omitempty"`
	ConnectionReference *DatabaseConnectionReference `json:"connection_reference,omitempty"`
	Executions         []DatabaseValidationExecution `json:"executions,omitempty"`
}

type DatabaseConnectionTestDetail struct {
	ConnectionTest      DatabaseConnectionTest      `json:"connection_test"`
	ConnectionReference DatabaseConnectionReference `json:"connection_reference"`
}

type DatabaseValidationExecutionDetail struct {
	Execution           DatabaseValidationExecution  `json:"execution"`
	ValidationCheck     DatabaseValidationCheck      `json:"validation_check"`
	DatabaseChange      *DatabaseChange              `json:"database_change,omitempty"`
	ConnectionReference *DatabaseConnectionReference `json:"connection_reference,omitempty"`
}

type DatabasePosture struct {
	Status                 string   `json:"status"`
	Summary                string   `json:"summary"`
	Compatibility          string   `json:"compatibility"`
	RollbackSafety         string   `json:"rollback_safety"`
	ManualApprovalRequired bool     `json:"manual_approval_required,omitempty"`
	ChangeCount            int      `json:"change_count"`
	RequiredCheckCount     int      `json:"required_check_count"`
	PendingCheckCount      int      `json:"pending_check_count"`
	BlockingFindings       []string `json:"blocking_findings,omitempty"`
	WarningFindings        []string `json:"warning_findings,omitempty"`
}

type ReleaseAnalysis struct {
	Release                 Release                   `json:"release"`
	ChangeSets              []ChangeSet               `json:"change_sets"`
	Assessments             []RiskAssessment          `json:"assessments"`
	ConfigSets              []ConfigSet               `json:"config_sets,omitempty"`
	DatabaseConnections     []DatabaseConnectionReference `json:"database_connections,omitempty"`
	DatabaseConnectionTests []DatabaseConnectionTest  `json:"database_connection_tests,omitempty"`
	DatabaseChanges         []DatabaseChange          `json:"database_changes,omitempty"`
	DatabaseChecks          []DatabaseValidationCheck `json:"database_checks,omitempty"`
	DatabaseExecutions      []DatabaseValidationExecution `json:"database_executions,omitempty"`
	LinkedRolloutExecutions []RolloutExecution        `json:"linked_rollout_executions,omitempty"`
	CombinedRiskScore       int                       `json:"combined_risk_score"`
	CombinedRiskLevel       RiskLevel                 `json:"combined_risk_level"`
	BlastRadius             BlastRadius               `json:"blast_radius"`
	ReleaseSummary          string                    `json:"release_summary"`
	DependencyPlan          []ReleaseDependency       `json:"dependency_plan,omitempty"`
	ConfigValidation        []ConfigSetValidation     `json:"config_validation,omitempty"`
	DatabasePosture         DatabasePosture           `json:"database_posture"`
	DatabaseFindings        []string                  `json:"database_findings,omitempty"`
	WindowFindings          []string                  `json:"window_findings,omitempty"`
	PolicyHighlights        []string                  `json:"policy_highlights,omitempty"`
	Warnings                []string                  `json:"warnings,omitempty"`
	Blockers                []string                  `json:"blockers,omitempty"`
	ReadinessReview         []ReadinessReviewItem     `json:"readiness_review,omitempty"`
	RollbackGuidance        RollbackGuidance          `json:"rollback_guidance"`
	OpsAssistant            OpsAssistantSummary       `json:"ops_assistant"`
	TeamMemory              []TeamMemoryInsight       `json:"team_memory,omitempty"`
	Communications          CommunicationDrafts       `json:"communications"`
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
	ReleaseID                 string    `json:"release_id,omitempty"`
	ReleaseName               string    `json:"release_name,omitempty"`
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
	Release             *Release                   `json:"release,omitempty"`
	ReleaseAnalysis     *ReleaseAnalysis           `json:"release_analysis,omitempty"`
	DatabaseConnections []DatabaseConnectionReference `json:"database_connections,omitempty"`
	DatabaseConnectionTests []DatabaseConnectionTest `json:"database_connection_tests,omitempty"`
	DatabaseChanges     []DatabaseChange           `json:"database_changes,omitempty"`
	DatabaseChecks      []DatabaseValidationCheck  `json:"database_checks,omitempty"`
	DatabaseExecutions  []DatabaseValidationExecution `json:"database_executions,omitempty"`
	DatabasePosture     *DatabasePosture           `json:"database_posture,omitempty"`
}
