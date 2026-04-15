package types

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
	Email                string                `json:"email,omitempty"`
	DisplayName          string                `json:"display_name,omitempty"`
	ActiveOrganizationID string                `json:"active_organization_id,omitempty"`
	Organizations        []SessionOrganization `json:"organizations,omitempty"`
	ProjectMemberships   []SessionProjectScope `json:"project_memberships,omitempty"`
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
	RolloutPlanID string `json:"rollout_plan_id"`
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
	Summary                string   `json:"summary"`
	Explanation            []string `json:"explanation,omitempty"`
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
	Name         *string   `json:"name,omitempty"`
	Mode         *string   `json:"mode,omitempty"`
	Status       *string   `json:"status,omitempty"`
	Description  *string   `json:"description,omitempty"`
	Capabilities *[]string `json:"capabilities,omitempty"`
	Metadata     Metadata  `json:"metadata,omitempty"`
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

type DevLoginRequest struct {
	Email            string   `json:"email"`
	DisplayName      string   `json:"display_name"`
	OrganizationName string   `json:"organization_name,omitempty"`
	OrganizationSlug string   `json:"organization_slug,omitempty"`
	Roles            []string `json:"roles,omitempty"`
}

type DevLoginResponse struct {
	Token   string      `json:"token"`
	Session SessionInfo `json:"session"`
}

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
	Execution           RolloutExecution     `json:"execution"`
	VerificationResults []VerificationResult `json:"verification_results"`
}
