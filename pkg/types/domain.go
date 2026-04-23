package types

import "time"

type Metadata map[string]any

type BaseRecord struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Metadata  Metadata  `json:"metadata,omitempty"`
}

func (b BaseRecord) CreatedTime() time.Time {
	return b.CreatedAt
}

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type Organization struct {
	BaseRecord
	Name string `json:"name"`
	Slug string `json:"slug"`
	Tier string `json:"tier"`
	Mode string `json:"mode"`
}

type Project struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Description    string `json:"description"`
	AdoptionMode   string `json:"adoption_mode"`
	Status         string `json:"status"`
}

type Team struct {
	BaseRecord
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	Name           string   `json:"name"`
	Slug           string   `json:"slug"`
	OwnerUserIDs   []string `json:"owner_user_ids,omitempty"`
	Status         string   `json:"status"`
}

type User struct {
	BaseRecord
	OrganizationID     string `json:"organization_id"`
	Email              string `json:"email"`
	DisplayName        string `json:"display_name"`
	Status             string `json:"status"`
	PasswordSalt       string `json:"-"`
	PasswordHash       string `json:"-"`
	PasswordIterations int    `json:"-"`
}

type IdentityProvider struct {
	BaseRecord
	OrganizationID        string     `json:"organization_id"`
	Name                  string     `json:"name"`
	Kind                  string     `json:"kind"`
	IssuerURL             string     `json:"issuer_url,omitempty"`
	AuthorizationEndpoint string     `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string     `json:"token_endpoint,omitempty"`
	UserInfoEndpoint      string     `json:"userinfo_endpoint,omitempty"`
	JWKSURI               string     `json:"jwks_uri,omitempty"`
	ClientID              string     `json:"client_id,omitempty"`
	ClientSecretEnv       string     `json:"client_secret_env,omitempty"`
	Scopes                []string   `json:"scopes,omitempty"`
	ClaimMappings         Metadata   `json:"claim_mappings,omitempty"`
	RoleMappings          Metadata   `json:"role_mappings,omitempty"`
	AllowedDomains        []string   `json:"allowed_domains,omitempty"`
	DefaultRole           string     `json:"default_role,omitempty"`
	Enabled               bool       `json:"enabled"`
	Status                string     `json:"status"`
	ConnectionHealth      string     `json:"connection_health,omitempty"`
	LastTestedAt          *time.Time `json:"last_tested_at,omitempty"`
	LastError             string     `json:"last_error,omitempty"`
	LastAuthenticatedAt   *time.Time `json:"last_authenticated_at,omitempty"`
}

type IdentityLink struct {
	BaseRecord
	OrganizationID  string     `json:"organization_id"`
	ProviderID      string     `json:"provider_id"`
	UserID          string     `json:"user_id"`
	ExternalSubject string     `json:"external_subject"`
	Email           string     `json:"email,omitempty"`
	Status          string     `json:"status"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
}

type Role struct {
	BaseRecord
	OrganizationID string   `json:"organization_id"`
	Name           string   `json:"name"`
	Scope          string   `json:"scope"`
	Permissions    []string `json:"permissions"`
}

type Environment struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Type           string `json:"type"`
	Region         string `json:"region"`
	Production     bool   `json:"production"`
	ComplianceZone string `json:"compliance_zone"`
	Status         string `json:"status"`
}

type Service struct {
	BaseRecord
	OrganizationID         string `json:"organization_id"`
	ProjectID              string `json:"project_id"`
	TeamID                 string `json:"team_id"`
	Name                   string `json:"name"`
	Slug                   string `json:"slug"`
	Description            string `json:"description"`
	Criticality            string `json:"criticality"`
	Tier                   string `json:"tier"`
	CustomerFacing         bool   `json:"customer_facing"`
	HasSLO                 bool   `json:"has_slo"`
	HasObservability       bool   `json:"has_observability"`
	RegulatedZone          bool   `json:"regulated_zone"`
	DependentServicesCount int    `json:"dependent_services_count"`
	Status                 string `json:"status"`
}

type Repository struct {
	BaseRecord
	OrganizationID      string     `json:"organization_id"`
	ProjectID           string     `json:"project_id,omitempty"`
	ServiceID           string     `json:"service_id,omitempty"`
	EnvironmentID       string     `json:"environment_id,omitempty"`
	SourceIntegrationID string     `json:"source_integration_id,omitempty"`
	Name                string     `json:"name"`
	Provider            string     `json:"provider"`
	URL                 string     `json:"url"`
	DefaultBranch       string     `json:"default_branch"`
	Status              string     `json:"status"`
	LastSyncedAt        *time.Time `json:"last_synced_at,omitempty"`
}

type DiscoveredResource struct {
	BaseRecord
	OrganizationID string     `json:"organization_id"`
	IntegrationID  string     `json:"integration_id"`
	ProjectID      string     `json:"project_id,omitempty"`
	ServiceID      string     `json:"service_id,omitempty"`
	EnvironmentID  string     `json:"environment_id,omitempty"`
	RepositoryID   string     `json:"repository_id,omitempty"`
	ResourceType   string     `json:"resource_type"`
	Provider       string     `json:"provider"`
	ExternalID     string     `json:"external_id"`
	Namespace      string     `json:"namespace,omitempty"`
	Name           string     `json:"name"`
	Status         string     `json:"status"`
	Health         string     `json:"health,omitempty"`
	Summary        string     `json:"summary,omitempty"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
}

type Integration struct {
	BaseRecord
	OrganizationID          string     `json:"organization_id,omitempty"`
	Name                    string     `json:"name"`
	Kind                    string     `json:"kind"`
	InstanceKey             string     `json:"instance_key,omitempty"`
	ScopeType               string     `json:"scope_type,omitempty"`
	ScopeName               string     `json:"scope_name,omitempty"`
	Mode                    string     `json:"mode"`
	AuthStrategy            string     `json:"auth_strategy,omitempty"`
	OnboardingStatus        string     `json:"onboarding_status,omitempty"`
	Status                  string     `json:"status"`
	Enabled                 bool       `json:"enabled"`
	ControlEnabled          bool       `json:"control_enabled"`
	ConnectionHealth        string     `json:"connection_health"`
	Capabilities            []string   `json:"capabilities"`
	Description             string     `json:"description"`
	LastTestedAt            *time.Time `json:"last_tested_at,omitempty"`
	LastSyncedAt            *time.Time `json:"last_synced_at,omitempty"`
	LastError               string     `json:"last_error,omitempty"`
	ScheduleEnabled         bool       `json:"schedule_enabled"`
	ScheduleIntervalSeconds int        `json:"schedule_interval_seconds,omitempty"`
	SyncStaleAfterSeconds   int        `json:"sync_stale_after_seconds,omitempty"`
	NextScheduledSyncAt     *time.Time `json:"next_scheduled_sync_at,omitempty"`
	LastSyncAttemptedAt     *time.Time `json:"last_sync_attempted_at,omitempty"`
	LastSyncSucceededAt     *time.Time `json:"last_sync_succeeded_at,omitempty"`
	LastSyncFailedAt        *time.Time `json:"last_sync_failed_at,omitempty"`
	SyncClaimedAt           *time.Time `json:"-"`
	SyncConsecutiveFailures int        `json:"sync_consecutive_failures,omitempty"`
	FreshnessState          string     `json:"freshness_state,omitempty"`
	Stale                   bool       `json:"stale,omitempty"`
	SyncLagSeconds          int        `json:"sync_lag_seconds,omitempty"`
}

type IntegrationSyncRun struct {
	BaseRecord
	OrganizationID  string     `json:"organization_id"`
	IntegrationID   string     `json:"integration_id"`
	Operation       string     `json:"operation"`
	Trigger         string     `json:"trigger,omitempty"`
	Status          string     `json:"status"`
	Summary         string     `json:"summary"`
	Details         []string   `json:"details,omitempty"`
	ResourceCount   int        `json:"resource_count"`
	ExternalEventID string     `json:"external_event_id,omitempty"`
	ErrorClass      string     `json:"error_class,omitempty"`
	ScheduledFor    *time.Time `json:"scheduled_for,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

type WebhookRegistration struct {
	BaseRecord
	OrganizationID   string     `json:"organization_id"`
	IntegrationID    string     `json:"integration_id"`
	ProviderKind     string     `json:"provider_kind"`
	ScopeIdentifier  string     `json:"scope_identifier,omitempty"`
	CallbackURL      string     `json:"callback_url"`
	ExternalHookID   string     `json:"external_hook_id,omitempty"`
	Status           string     `json:"status"`
	DeliveryHealth   string     `json:"delivery_health,omitempty"`
	AutoManaged      bool       `json:"auto_managed"`
	LastRegisteredAt *time.Time `json:"last_registered_at,omitempty"`
	LastValidatedAt  *time.Time `json:"last_validated_at,omitempty"`
	LastDeliveryAt   *time.Time `json:"last_delivery_at,omitempty"`
	LastError        string     `json:"last_error,omitempty"`
	FailureCount     int        `json:"failure_count,omitempty"`
}

type Deployment struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ServiceID      string `json:"service_id"`
	EnvironmentID  string `json:"environment_id"`
	ReleaseID      string `json:"release_id"`
	Status         string `json:"status"`
	Strategy       string `json:"strategy"`
}

type ConfigEntry struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	Required    bool   `json:"required,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
}

type ConfigSet struct {
	BaseRecord
	OrganizationID string        `json:"organization_id"`
	ProjectID      string        `json:"project_id"`
	EnvironmentID  string        `json:"environment_id"`
	ServiceID      string        `json:"service_id,omitempty"`
	Name           string        `json:"name"`
	Version        string        `json:"version"`
	Status         string        `json:"status"`
	Entries        []ConfigEntry `json:"entries"`
}

type Release struct {
	BaseRecord
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	EnvironmentID  string   `json:"environment_id"`
	Name           string   `json:"name"`
	Summary        string   `json:"summary"`
	ChangeSetIDs   []string `json:"change_set_ids"`
	ConfigSetIDs   []string `json:"config_set_ids,omitempty"`
	Version        string   `json:"version"`
	Status         string   `json:"status"`
}

type DatabaseChange struct {
	BaseRecord
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
	Status                 string    `json:"status"`
	Summary                string    `json:"summary"`
	Evidence               []string  `json:"evidence,omitempty"`
}

type DatabaseValidationCheck struct {
	BaseRecord
	OrganizationID    string     `json:"organization_id"`
	ProjectID         string     `json:"project_id"`
	EnvironmentID     string     `json:"environment_id"`
	ServiceID         string     `json:"service_id,omitempty"`
	ChangeSetID       string     `json:"change_set_id"`
	DatabaseChangeID  string     `json:"database_change_id,omitempty"`
	ConnectionRefID   string     `json:"connection_ref_id,omitempty"`
	Name              string     `json:"name"`
	Phase             string     `json:"phase"`
	CheckType         string     `json:"check_type"`
	ReadOnly          bool       `json:"read_only"`
	Required          bool       `json:"required"`
	ExecutionMode     string     `json:"execution_mode"`
	Specification     string     `json:"specification"`
	Status            string     `json:"status"`
	Summary           string     `json:"summary"`
	LastRunAt         *time.Time `json:"last_run_at,omitempty"`
	LastResultSummary string     `json:"last_result_summary,omitempty"`
	Evidence          []string   `json:"evidence,omitempty"`
}

type DatabaseConnectionReference struct {
	BaseRecord
	OrganizationID   string     `json:"organization_id"`
	ProjectID        string     `json:"project_id"`
	EnvironmentID    string     `json:"environment_id"`
	ServiceID        string     `json:"service_id,omitempty"`
	Name             string     `json:"name"`
	Datastore        string     `json:"datastore"`
	Driver           string     `json:"driver"`
	SourceType       string     `json:"source_type"`
	DSNEnv           string     `json:"dsn_env"`
	SecretRef        string     `json:"secret_ref,omitempty"`
	SecretRefEnv     string     `json:"secret_ref_env,omitempty"`
	ReadOnlyCapable  bool       `json:"read_only_capable,omitempty"`
	Status           string     `json:"status"`
	Summary          string     `json:"summary"`
	LastTestedAt     *time.Time `json:"last_tested_at,omitempty"`
	LastHealthyAt    *time.Time `json:"last_healthy_at,omitempty"`
	LastErrorClass   string     `json:"last_error_class,omitempty"`
	LastErrorSummary string     `json:"last_error_summary,omitempty"`
}

type DatabaseConnectionTest struct {
	BaseRecord
	OrganizationID   string     `json:"organization_id"`
	ProjectID        string     `json:"project_id"`
	EnvironmentID    string     `json:"environment_id"`
	ServiceID        string     `json:"service_id,omitempty"`
	ConnectionRefID  string     `json:"connection_ref_id"`
	Trigger          string     `json:"trigger"`
	Status           string     `json:"status"`
	Summary          string     `json:"summary"`
	Details          []string   `json:"details,omitempty"`
	ErrorClass       string     `json:"error_class,omitempty"`
	ActorType        string     `json:"actor_type,omitempty"`
	ActorID          string     `json:"actor_id,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

type DatabaseValidationExecution struct {
	BaseRecord
	OrganizationID     string     `json:"organization_id"`
	ProjectID          string     `json:"project_id"`
	EnvironmentID      string     `json:"environment_id"`
	ServiceID          string     `json:"service_id,omitempty"`
	ChangeSetID        string     `json:"change_set_id"`
	DatabaseChangeID   string     `json:"database_change_id,omitempty"`
	ValidationCheckID  string     `json:"validation_check_id"`
	ConnectionRefID    string     `json:"connection_ref_id"`
	Trigger            string     `json:"trigger"`
	ExecutionMode      string     `json:"execution_mode"`
	Status             string     `json:"status"`
	Summary            string     `json:"summary"`
	ResultDetails      []string   `json:"result_details,omitempty"`
	Evidence           []string   `json:"evidence,omitempty"`
	ErrorClass         string     `json:"error_class,omitempty"`
	ActorType          string     `json:"actor_type,omitempty"`
	ActorID            string     `json:"actor_id,omitempty"`
	StartedAt          time.Time  `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
}

type ChangeSet struct {
	BaseRecord
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
	Status                  string   `json:"status"`
}

type ChangeArtifact struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ChangeSetID    string `json:"change_set_id"`
	Kind           string `json:"kind"`
	Name           string `json:"name"`
	Digest         string `json:"digest"`
}

type BlastRadius struct {
	Scope                string   `json:"scope"`
	ServicesImpacted     int      `json:"services_impacted"`
	ResourcesImpacted    int      `json:"resources_impacted"`
	CustomerJourneys     []string `json:"customer_journeys,omitempty"`
	RegulatedSystems     bool     `json:"regulated_systems"`
	ProductionImpact     bool     `json:"production_impact"`
	CustomerFacingImpact bool     `json:"customer_facing_impact"`
	Summary              string   `json:"summary"`
}

type RiskAssessment struct {
	BaseRecord
	OrganizationID              string      `json:"organization_id"`
	ProjectID                   string      `json:"project_id"`
	ChangeSetID                 string      `json:"change_set_id"`
	ServiceID                   string      `json:"service_id"`
	EnvironmentID               string      `json:"environment_id"`
	Score                       int         `json:"score"`
	Level                       RiskLevel   `json:"level"`
	ConfidenceScore             float64     `json:"confidence_score"`
	Explanation                 []string    `json:"explanation"`
	BlastRadius                 BlastRadius `json:"blast_radius"`
	RecommendedApprovalLevel    string      `json:"recommended_approval_level"`
	RecommendedRolloutStrategy  string      `json:"recommended_rollout_strategy"`
	RecommendedDeploymentWindow string      `json:"recommended_deployment_window"`
	RecommendedGuardrails       []string    `json:"recommended_guardrails"`
}

type RolloutStep struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Guards      []string `json:"guards,omitempty"`
}

type RolloutPlan struct {
	BaseRecord
	OrganizationID           string        `json:"organization_id"`
	ProjectID                string        `json:"project_id"`
	ChangeSetID              string        `json:"change_set_id"`
	RiskAssessmentID         string        `json:"risk_assessment_id"`
	Strategy                 string        `json:"strategy"`
	ApprovalRequired         bool          `json:"approval_required"`
	ApprovalLevel            string        `json:"approval_level"`
	DeploymentWindow         string        `json:"deployment_window"`
	AdditionalVerification   bool          `json:"additional_verification"`
	RollbackPrecheckRequired bool          `json:"rollback_precheck_required"`
	BusinessHoursRestriction bool          `json:"business_hours_restriction"`
	OffHoursPreferred        bool          `json:"off_hours_preferred"`
	VerificationSignals      []string      `json:"verification_signals"`
	RollbackConditions       []string      `json:"rollback_conditions"`
	Guardrails               []string      `json:"guardrails"`
	Steps                    []RolloutStep `json:"steps"`
	Explanation              []string      `json:"explanation"`
}

type VerificationPlan struct {
	BaseRecord
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	ServiceID      string   `json:"service_id"`
	EnvironmentID  string   `json:"environment_id"`
	Signals        []string `json:"signals"`
}

type VerificationResult struct {
	BaseRecord
	OrganizationID         string   `json:"organization_id"`
	ProjectID              string   `json:"project_id"`
	RolloutExecutionID     string   `json:"rollout_execution_id"`
	RolloutPlanID          string   `json:"rollout_plan_id"`
	ChangeSetID            string   `json:"change_set_id"`
	ServiceID              string   `json:"service_id"`
	EnvironmentID          string   `json:"environment_id"`
	Status                 string   `json:"status"`
	Outcome                string   `json:"outcome"`
	Decision               string   `json:"decision"`
	Signals                []string `json:"signals"`
	TechnicalSignalSummary Metadata `json:"technical_signal_summary,omitempty"`
	BusinessSignalSummary  Metadata `json:"business_signal_summary,omitempty"`
	Automated              bool     `json:"automated"`
	DecisionSource         string   `json:"decision_source"`
	SignalSnapshotIDs      []string `json:"signal_snapshot_ids,omitempty"`
	Summary                string   `json:"summary"`
	Explanation            []string `json:"explanation,omitempty"`
	ActionState            string   `json:"action_state,omitempty"`
	ControlMode            string   `json:"control_mode,omitempty"`
}

type Incident struct {
	BaseRecord
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	ServiceID      string   `json:"service_id"`
	EnvironmentID  string   `json:"environment_id"`
	Title          string   `json:"title"`
	Severity       string   `json:"severity"`
	Status         string   `json:"status"`
	RelatedChange  string   `json:"related_change"`
	ImpactedPaths  []string `json:"impacted_paths,omitempty"`
}

type IncidentDetail struct {
	Incident           Incident             `json:"incident"`
	RolloutExecutionID string               `json:"rollout_execution_id"`
	StatusTimeline     []StatusEvent        `json:"status_timeline,omitempty"`
	AssistantSummary   *OpsAssistantSummary `json:"assistant_summary,omitempty"`
}

type Runbook struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ServiceID      string `json:"service_id"`
	Name           string `json:"name"`
	Link           string `json:"link"`
	ExecutionMode  string `json:"execution_mode"`
}

type Policy struct {
	BaseRecord
	OrganizationID string          `json:"organization_id,omitempty"`
	ProjectID      string          `json:"project_id,omitempty"`
	ServiceID      string          `json:"service_id,omitempty"`
	EnvironmentID  string          `json:"environment_id,omitempty"`
	Name           string          `json:"name"`
	Code           string          `json:"code"`
	Scope          string          `json:"scope"`
	AppliesTo      string          `json:"applies_to"`
	Mode           string          `json:"mode"`
	Enabled        bool            `json:"enabled"`
	Priority       int             `json:"priority"`
	Description    string          `json:"description"`
	Conditions     PolicyCondition `json:"conditions,omitempty"`
	Triggers       []string        `json:"triggers,omitempty"`
}

type PolicyCondition struct {
	MinRiskLevel        string   `json:"min_risk_level,omitempty"`
	ProductionOnly      bool     `json:"production_only,omitempty"`
	RegulatedOnly       bool     `json:"regulated_only,omitempty"`
	RequiredChangeTypes []string `json:"required_change_types,omitempty"`
	RequiredTouches     []string `json:"required_touches,omitempty"`
	MissingCapabilities []string `json:"missing_capabilities,omitempty"`
}

type PolicyDecision struct {
	BaseRecord
	OrganizationID     string   `json:"organization_id"`
	ProjectID          string   `json:"project_id"`
	ServiceID          string   `json:"service_id,omitempty"`
	EnvironmentID      string   `json:"environment_id,omitempty"`
	PolicyID           string   `json:"policy_id"`
	PolicyName         string   `json:"policy_name"`
	PolicyCode         string   `json:"policy_code"`
	PolicyScope        string   `json:"policy_scope"`
	AppliesTo          string   `json:"applies_to"`
	Mode               string   `json:"mode"`
	ChangeSetID        string   `json:"change_set_id,omitempty"`
	RiskAssessmentID   string   `json:"risk_assessment_id,omitempty"`
	RolloutPlanID      string   `json:"rollout_plan_id,omitempty"`
	RolloutExecutionID string   `json:"rollout_execution_id,omitempty"`
	Outcome            string   `json:"outcome"`
	Summary            string   `json:"summary"`
	Reasons            []string `json:"reasons"`
}

type AuditEvent struct {
	BaseRecord
	OrganizationID string   `json:"organization_id,omitempty"`
	ProjectID      string   `json:"project_id,omitempty"`
	ActorID        string   `json:"actor_id"`
	ActorType      string   `json:"actor_type"`
	Actor          string   `json:"actor"`
	Action         string   `json:"action"`
	ResourceType   string   `json:"resource_type"`
	ResourceID     string   `json:"resource_id"`
	Outcome        string   `json:"outcome"`
	Details        []string `json:"details,omitempty"`
}

type CostBaseline struct {
	BaseRecord
	OrganizationID string  `json:"organization_id"`
	ProjectID      string  `json:"project_id"`
	ServiceID      string  `json:"service_id"`
	MonthlyUSD     float64 `json:"monthly_usd"`
}

type ServiceDependency struct {
	BaseRecord
	OrganizationID     string `json:"organization_id"`
	ProjectID          string `json:"project_id"`
	ServiceID          string `json:"service_id"`
	DependsOnServiceID string `json:"depends_on_service_id"`
	CriticalDependency bool   `json:"critical_dependency"`
}

type Resource struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	EnvironmentID  string `json:"environment_id"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	URN            string `json:"urn"`
}

type InfrastructureStack struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	EnvironmentID  string `json:"environment_id"`
	Name           string `json:"name"`
	Engine         string `json:"engine"`
	Status         string `json:"status"`
}

type SecretReference struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ServiceID      string `json:"service_id"`
	Name           string `json:"name"`
	Backend        string `json:"backend"`
	Path           string `json:"path"`
}

type ComplianceZone struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	Description    string `json:"description"`
}

type DataClassification struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Level          string `json:"level"`
}

type SLO struct {
	BaseRecord
	OrganizationID string  `json:"organization_id"`
	ProjectID      string  `json:"project_id"`
	ServiceID      string  `json:"service_id"`
	Name           string  `json:"name"`
	Target         float64 `json:"target"`
}

type MetricSource struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
}

type BusinessMetric struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ServiceID      string `json:"service_id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	Unit           string `json:"unit"`
}

type GraphRelationship struct {
	BaseRecord
	OrganizationID      string    `json:"organization_id"`
	ProjectID           string    `json:"project_id,omitempty"`
	SourceIntegrationID string    `json:"source_integration_id,omitempty"`
	RelationshipType    string    `json:"relationship_type"`
	FromResourceType    string    `json:"from_resource_type"`
	FromResourceID      string    `json:"from_resource_id"`
	ToResourceType      string    `json:"to_resource_type"`
	ToResourceID        string    `json:"to_resource_id"`
	Status              string    `json:"status"`
	LastObservedAt      time.Time `json:"last_observed_at"`
}

type SignalValue struct {
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit,omitempty"`
	Status     string  `json:"status"`
	Threshold  float64 `json:"threshold,omitempty"`
	Comparator string  `json:"comparator,omitempty"`
}

type SignalSnapshot struct {
	BaseRecord
	OrganizationID      string        `json:"organization_id"`
	ProjectID           string        `json:"project_id"`
	RolloutExecutionID  string        `json:"rollout_execution_id"`
	RolloutPlanID       string        `json:"rollout_plan_id"`
	ChangeSetID         string        `json:"change_set_id"`
	ServiceID           string        `json:"service_id"`
	EnvironmentID       string        `json:"environment_id"`
	ProviderType        string        `json:"provider_type"`
	SourceIntegrationID string        `json:"source_integration_id,omitempty"`
	Health              string        `json:"health"`
	Summary             string        `json:"summary"`
	Signals             []SignalValue `json:"signals"`
	WindowStart         time.Time     `json:"window_start"`
	WindowEnd           time.Time     `json:"window_end"`
}

type RolloutExecution struct {
	BaseRecord
	OrganizationID         string     `json:"organization_id"`
	ProjectID              string     `json:"project_id"`
	RolloutPlanID          string     `json:"rollout_plan_id"`
	ReleaseID              string     `json:"release_id,omitempty"`
	ChangeSetID            string     `json:"change_set_id"`
	ServiceID              string     `json:"service_id"`
	EnvironmentID          string     `json:"environment_id"`
	BackendType            string     `json:"backend_type"`
	BackendIntegrationID   string     `json:"backend_integration_id,omitempty"`
	SignalProviderType     string     `json:"signal_provider_type"`
	SignalIntegrationID    string     `json:"signal_integration_id,omitempty"`
	BackendExecutionID     string     `json:"backend_execution_id,omitempty"`
	BackendStatus          string     `json:"backend_status,omitempty"`
	ProgressPercent        int        `json:"progress_percent"`
	Status                 string     `json:"status"`
	CurrentStep            string     `json:"current_step"`
	LastDecision           string     `json:"last_decision,omitempty"`
	LastDecisionReason     string     `json:"last_decision_reason,omitempty"`
	LastVerificationResult string     `json:"last_verification_result,omitempty"`
	SubmittedAt            *time.Time `json:"submitted_at,omitempty"`
	StartedAt              *time.Time `json:"started_at,omitempty"`
	CompletedAt            *time.Time `json:"completed_at,omitempty"`
	LastReconciledAt       *time.Time `json:"last_reconciled_at,omitempty"`
	LastBackendSyncAt      *time.Time `json:"last_backend_sync_at,omitempty"`
	LastSignalSyncAt       *time.Time `json:"last_signal_sync_at,omitempty"`
	LastError              string     `json:"last_error,omitempty"`
}

type RolloutExecutionRuntimeSummary struct {
	BackendType               string `json:"backend_type"`
	BackendStatus             string `json:"backend_status"`
	ProgressPercent           int    `json:"progress_percent"`
	LatestSignalHealth        string `json:"latest_signal_health,omitempty"`
	LatestSignalSummary       string `json:"latest_signal_summary,omitempty"`
	LatestDecision            string `json:"latest_decision,omitempty"`
	LatestDecisionMode        string `json:"latest_decision_mode,omitempty"`
	ControlMode               string `json:"control_mode,omitempty"`
	ControlEnabled            bool   `json:"control_enabled,omitempty"`
	AdvisoryOnly              bool   `json:"advisory_only,omitempty"`
	RecommendedAction         string `json:"recommended_action,omitempty"`
	LastProviderAction        string `json:"last_provider_action,omitempty"`
	LastActionDisposition     string `json:"last_action_disposition,omitempty"`
	LastProviderActionSummary string `json:"last_provider_action_summary,omitempty"`
	ControlRationale          string `json:"control_rationale,omitempty"`
}

type RolloutExecutionRuntimeContext struct {
	Execution               RolloutExecution     `json:"execution"`
	Plan                    RolloutPlan          `json:"plan"`
	Assessment              RiskAssessment       `json:"assessment"`
	ChangeSet               ChangeSet            `json:"change_set"`
	Service                 Service              `json:"service"`
	Environment             Environment          `json:"environment"`
	BackendIntegration      *Integration         `json:"backend_integration,omitempty"`
	SignalIntegration       *Integration         `json:"signal_integration,omitempty"`
	EffectiveRollbackPolicy *RollbackPolicy      `json:"effective_rollback_policy,omitempty"`
	VerificationResults     []VerificationResult `json:"verification_results"`
	SignalSnapshots         []SignalSnapshot     `json:"signal_snapshots"`
}

type ApprovalRequest struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ChangeSetID    string `json:"change_set_id"`
	Level          string `json:"level"`
	Status         string `json:"status"`
}

type ApprovalDecision struct {
	BaseRecord
	OrganizationID    string `json:"organization_id"`
	ProjectID         string `json:"project_id"`
	ApprovalRequestID string `json:"approval_request_id"`
	Outcome           string `json:"outcome"`
	Approver          string `json:"approver"`
	Reason            string `json:"reason"`
}

type SimulationRun struct {
	BaseRecord
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	ChangeSetID    string `json:"change_set_id"`
	Status         string `json:"status"`
}

type SimulationResult struct {
	BaseRecord
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	SimulationRun  string   `json:"simulation_run"`
	Status         string   `json:"status"`
	Findings       []string `json:"findings"`
}

type DomainEvent struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`
	OrganizationID string    `json:"organization_id,omitempty"`
	ProjectID      string    `json:"project_id,omitempty"`
	ResourceType   string    `json:"resource_type"`
	ResourceID     string    `json:"resource_id"`
	OccurredAt     time.Time `json:"occurred_at"`
	Payload        Metadata  `json:"payload,omitempty"`
}

type OutboxEvent struct {
	BaseRecord
	EventType      string     `json:"event_type"`
	OrganizationID string     `json:"organization_id,omitempty"`
	ProjectID      string     `json:"project_id,omitempty"`
	ResourceType   string     `json:"resource_type"`
	ResourceID     string     `json:"resource_id"`
	Status         string     `json:"status"`
	Attempts       int        `json:"attempts,omitempty"`
	NextAttemptAt  *time.Time `json:"next_attempt_at,omitempty"`
	ClaimedAt      *time.Time `json:"claimed_at,omitempty"`
	ProcessedAt    *time.Time `json:"processed_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
}
