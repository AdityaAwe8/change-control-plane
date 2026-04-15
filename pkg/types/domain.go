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
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email"`
	DisplayName    string `json:"display_name"`
	Status         string `json:"status"`
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
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	Name           string `json:"name"`
	Provider       string `json:"provider"`
	URL            string `json:"url"`
	DefaultBranch  string `json:"default_branch"`
}

type Integration struct {
	BaseRecord
	OrganizationID string     `json:"organization_id,omitempty"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind"`
	Mode           string     `json:"mode"`
	Status         string     `json:"status"`
	Capabilities   []string   `json:"capabilities"`
	Description    string     `json:"description"`
	LastSyncedAt   *time.Time `json:"last_synced_at,omitempty"`
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

type Release struct {
	BaseRecord
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	ChangeSetIDs   []string `json:"change_set_ids"`
	Version        string   `json:"version"`
	Status         string   `json:"status"`
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
	Summary                string   `json:"summary"`
	Explanation            []string `json:"explanation,omitempty"`
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
	OrganizationID string   `json:"organization_id,omitempty"`
	Name           string   `json:"name"`
	Code           string   `json:"code"`
	Scope          string   `json:"scope"`
	Mode           string   `json:"mode"`
	Enabled        bool     `json:"enabled"`
	Description    string   `json:"description"`
	Triggers       []string `json:"triggers,omitempty"`
}

type PolicyDecision struct {
	BaseRecord
	OrganizationID string   `json:"organization_id"`
	ProjectID      string   `json:"project_id"`
	PolicyID       string   `json:"policy_id"`
	ChangeSetID    string   `json:"change_set_id"`
	Outcome        string   `json:"outcome"`
	Summary        string   `json:"summary"`
	Reasons        []string `json:"reasons"`
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

type RolloutExecution struct {
	BaseRecord
	OrganizationID         string     `json:"organization_id"`
	ProjectID              string     `json:"project_id"`
	RolloutPlanID          string     `json:"rollout_plan_id"`
	ChangeSetID            string     `json:"change_set_id"`
	ServiceID              string     `json:"service_id"`
	EnvironmentID          string     `json:"environment_id"`
	Status                 string     `json:"status"`
	CurrentStep            string     `json:"current_step"`
	LastDecision           string     `json:"last_decision,omitempty"`
	LastDecisionReason     string     `json:"last_decision_reason,omitempty"`
	LastVerificationResult string     `json:"last_verification_result,omitempty"`
	StartedAt              *time.Time `json:"started_at,omitempty"`
	CompletedAt            *time.Time `json:"completed_at,omitempty"`
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
