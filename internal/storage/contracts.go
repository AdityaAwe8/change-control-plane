package storage

import (
	"context"
	"errors"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

var ErrNotFound = errors.New("resource not found")

type OrganizationQuery struct {
	IDs    []string
	Limit  int
	Offset int
}

type IdentityProviderQuery struct {
	OrganizationID string
	Kind           string
	Enabled        *bool
	Limit          int
	Offset         int
}

type ProjectQuery struct {
	OrganizationID string
	IDs            []string
	Limit          int
	Offset         int
}

type TeamQuery struct {
	OrganizationID string
	ProjectID      string
	IDs            []string
	Limit          int
	Offset         int
}

type ServiceQuery struct {
	OrganizationID string
	ProjectID      string
	TeamID         string
	IDs            []string
	Limit          int
	Offset         int
}

type EnvironmentQuery struct {
	OrganizationID string
	ProjectID      string
	IDs            []string
	Limit          int
	Offset         int
}

type ChangeSetQuery struct {
	OrganizationID string
	ProjectID      string
	ServiceID      string
	Limit          int
	Offset         int
}

type RiskAssessmentQuery struct {
	OrganizationID string
	ProjectID      string
	ChangeSetID    string
	Limit          int
	Offset         int
}

type RolloutPlanQuery struct {
	OrganizationID string
	ProjectID      string
	ChangeSetID    string
	Limit          int
	Offset         int
}

type ConfigSetQuery struct {
	OrganizationID string
	ProjectID      string
	EnvironmentID  string
	ServiceID      string
	Status         string
	Limit          int
	Offset         int
}

type ReleaseQuery struct {
	OrganizationID string
	ProjectID      string
	EnvironmentID  string
	Status         string
	Limit          int
	Offset         int
}

type DatabaseChangeQuery struct {
	OrganizationID string
	ProjectID      string
	EnvironmentID  string
	ServiceID      string
	ChangeSetID    string
	Datastore      string
	Status         string
	Limit          int
	Offset         int
}

type DatabaseValidationCheckQuery struct {
	OrganizationID   string
	ProjectID        string
	EnvironmentID    string
	ServiceID        string
	ChangeSetID      string
	DatabaseChangeID string
	ConnectionRefID  string
	Phase            string
	Status           string
	Limit            int
	Offset           int
}

type DatabaseConnectionReferenceQuery struct {
	OrganizationID string
	ProjectID      string
	EnvironmentID  string
	ServiceID      string
	Datastore      string
	Status         string
	Limit          int
	Offset         int
}

type DatabaseValidationExecutionQuery struct {
	OrganizationID    string
	ProjectID         string
	EnvironmentID     string
	ServiceID         string
	ChangeSetID       string
	DatabaseChangeID  string
	ValidationCheckID string
	ConnectionRefID   string
	Status            string
	Limit             int
	Offset            int
}

type DatabaseConnectionTestQuery struct {
	OrganizationID  string
	ProjectID       string
	EnvironmentID   string
	ServiceID       string
	ConnectionRefID string
	Status          string
	Limit           int
	Offset          int
}

type AuditEventQuery struct {
	OrganizationID string
	ProjectID      string
	ResourceType   string
	ResourceID     string
	Limit          int
	Offset         int
}

type IntegrationQuery struct {
	OrganizationID string
	Kind           string
	InstanceKey    string
	ScopeType      string
	AuthStrategy   string
	Enabled        *bool
	Search         string
	Limit          int
	Offset         int
}

type IntegrationSyncRunQuery struct {
	OrganizationID  string
	IntegrationID   string
	Operation       string
	Trigger         string
	Status          string
	ExternalEventID string
	Limit           int
	Offset          int
}

type WebhookRegistrationQuery struct {
	OrganizationID string
	IntegrationID  string
	ProviderKind   string
	Status         string
	Limit          int
	Offset         int
}

type ServiceAccountQuery struct {
	OrganizationID string
	Status         string
	Limit          int
	Offset         int
}

type APITokenQuery struct {
	OrganizationID   string
	ServiceAccountID string
	Status           string
	Limit            int
	Offset           int
}

type BrowserSessionQuery struct {
	OrganizationID string
	UserID         string
	Status         string
	Limit          int
	Offset         int
}

type RepositoryQuery struct {
	OrganizationID      string
	ProjectID           string
	ServiceID           string
	EnvironmentID       string
	SourceIntegrationID string
	Provider            string
	Limit               int
	Offset              int
}

type DiscoveredResourceQuery struct {
	OrganizationID string
	IntegrationID  string
	ResourceType   string
	Provider       string
	ProjectID      string
	ServiceID      string
	EnvironmentID  string
	RepositoryID   string
	Status         string
	Search         string
	UnmappedOnly   bool
	Limit          int
	Offset         int
}

type GraphRelationshipQuery struct {
	OrganizationID      string
	SourceIntegrationID string
	RelationshipType    string
	FromResourceID      string
	ToResourceID        string
	Limit               int
	Offset              int
}

type RolloutExecutionQuery struct {
	OrganizationID string
	ProjectID      string
	ServiceID      string
	EnvironmentID  string
	Status         string
	Limit          int
	Offset         int
}

type VerificationResultQuery struct {
	OrganizationID     string
	ProjectID          string
	RolloutExecutionID string
	Limit              int
	Offset             int
}

type SignalSnapshotQuery struct {
	OrganizationID     string
	ProjectID          string
	RolloutExecutionID string
	ProviderType       string
	Limit              int
	Offset             int
}

type RollbackPolicyQuery struct {
	OrganizationID string
	ProjectID      string
	ServiceID      string
	EnvironmentID  string
	EnabledOnly    bool
	Limit          int
	Offset         int
}

type PolicyQuery struct {
	OrganizationID string
	ProjectID      string
	ServiceID      string
	EnvironmentID  string
	AppliesTo      string
	EnabledOnly    bool
	Limit          int
	Offset         int
}

type PolicyDecisionQuery struct {
	OrganizationID     string
	ProjectID          string
	PolicyID           string
	ChangeSetID        string
	RiskAssessmentID   string
	RolloutPlanID      string
	RolloutExecutionID string
	AppliesTo          string
	Limit              int
	Offset             int
}

type StatusEventQuery struct {
	OrganizationID     string
	ProjectID          string
	TeamID             string
	ServiceID          string
	EnvironmentID      string
	RolloutExecutionID string
	ChangeSetID        string
	ResourceType       string
	ResourceID         string
	EventTypes         []string
	ActorType          string
	ActorID            string
	Source             string
	Outcome            string
	Automated          *bool
	RollbackOnly       bool
	Search             string
	Since              *time.Time
	Until              *time.Time
	Limit              int
	Offset             int
}

type OutboxEventQuery struct {
	OrganizationID string
	EventType      string
	Status         string
	Limit          int
	Offset         int
}

type Store interface {
	Close() error
	WithinTransaction(context.Context, func(context.Context) error) error

	CreateOrganization(context.Context, types.Organization) error
	GetOrganization(context.Context, string) (types.Organization, error)
	GetOrganizationBySlug(context.Context, string) (types.Organization, error)
	ListOrganizations(context.Context, OrganizationQuery) ([]types.Organization, error)
	UpdateOrganization(context.Context, types.Organization) error

	CreateProject(context.Context, types.Project) error
	GetProject(context.Context, string) (types.Project, error)
	ListProjects(context.Context, ProjectQuery) ([]types.Project, error)
	UpdateProject(context.Context, types.Project) error

	CreateTeam(context.Context, types.Team) error
	GetTeam(context.Context, string) (types.Team, error)
	ListTeams(context.Context, TeamQuery) ([]types.Team, error)
	UpdateTeam(context.Context, types.Team) error

	CreateService(context.Context, types.Service) error
	GetService(context.Context, string) (types.Service, error)
	ListServices(context.Context, ServiceQuery) ([]types.Service, error)
	UpdateService(context.Context, types.Service) error

	CreateEnvironment(context.Context, types.Environment) error
	GetEnvironment(context.Context, string) (types.Environment, error)
	ListEnvironments(context.Context, EnvironmentQuery) ([]types.Environment, error)
	UpdateEnvironment(context.Context, types.Environment) error

	CreateChangeSet(context.Context, types.ChangeSet) error
	GetChangeSet(context.Context, string) (types.ChangeSet, error)
	ListChangeSets(context.Context, ChangeSetQuery) ([]types.ChangeSet, error)

	CreateRiskAssessment(context.Context, types.RiskAssessment) error
	GetRiskAssessment(context.Context, string) (types.RiskAssessment, error)
	ListRiskAssessments(context.Context, RiskAssessmentQuery) ([]types.RiskAssessment, error)

	CreateRolloutPlan(context.Context, types.RolloutPlan) error
	GetRolloutPlan(context.Context, string) (types.RolloutPlan, error)
	ListRolloutPlans(context.Context, RolloutPlanQuery) ([]types.RolloutPlan, error)

	CreateConfigSet(context.Context, types.ConfigSet) error
	GetConfigSet(context.Context, string) (types.ConfigSet, error)
	ListConfigSets(context.Context, ConfigSetQuery) ([]types.ConfigSet, error)
	UpdateConfigSet(context.Context, types.ConfigSet) error

	CreateRelease(context.Context, types.Release) error
	GetRelease(context.Context, string) (types.Release, error)
	ListReleases(context.Context, ReleaseQuery) ([]types.Release, error)
	UpdateRelease(context.Context, types.Release) error

	CreateDatabaseChange(context.Context, types.DatabaseChange) error
	GetDatabaseChange(context.Context, string) (types.DatabaseChange, error)
	ListDatabaseChanges(context.Context, DatabaseChangeQuery) ([]types.DatabaseChange, error)
	UpdateDatabaseChange(context.Context, types.DatabaseChange) error

	CreateDatabaseValidationCheck(context.Context, types.DatabaseValidationCheck) error
	GetDatabaseValidationCheck(context.Context, string) (types.DatabaseValidationCheck, error)
	ListDatabaseValidationChecks(context.Context, DatabaseValidationCheckQuery) ([]types.DatabaseValidationCheck, error)
	UpdateDatabaseValidationCheck(context.Context, types.DatabaseValidationCheck) error

	CreateDatabaseConnectionReference(context.Context, types.DatabaseConnectionReference) error
	GetDatabaseConnectionReference(context.Context, string) (types.DatabaseConnectionReference, error)
	ListDatabaseConnectionReferences(context.Context, DatabaseConnectionReferenceQuery) ([]types.DatabaseConnectionReference, error)
	UpdateDatabaseConnectionReference(context.Context, types.DatabaseConnectionReference) error
	CreateDatabaseConnectionTest(context.Context, types.DatabaseConnectionTest) error
	GetDatabaseConnectionTest(context.Context, string) (types.DatabaseConnectionTest, error)
	ListDatabaseConnectionTests(context.Context, DatabaseConnectionTestQuery) ([]types.DatabaseConnectionTest, error)
	UpdateDatabaseConnectionTest(context.Context, types.DatabaseConnectionTest) error

	CreateDatabaseValidationExecution(context.Context, types.DatabaseValidationExecution) error
	GetDatabaseValidationExecution(context.Context, string) (types.DatabaseValidationExecution, error)
	ListDatabaseValidationExecutions(context.Context, DatabaseValidationExecutionQuery) ([]types.DatabaseValidationExecution, error)
	UpdateDatabaseValidationExecution(context.Context, types.DatabaseValidationExecution) error

	CreateAuditEvent(context.Context, types.AuditEvent) error
	ListAuditEvents(context.Context, AuditEventQuery) ([]types.AuditEvent, error)

	CreateIntegration(context.Context, types.Integration) error
	GetIntegration(context.Context, string) (types.Integration, error)
	UpsertIntegration(context.Context, types.Integration) error
	ListIntegrations(context.Context, IntegrationQuery) ([]types.Integration, error)
	UpdateIntegration(context.Context, types.Integration) error
	ClaimIntegrationSync(context.Context, string, time.Time, time.Time, time.Time) (bool, error)
	CreateIntegrationSyncRun(context.Context, types.IntegrationSyncRun) error
	ListIntegrationSyncRuns(context.Context, IntegrationSyncRunQuery) ([]types.IntegrationSyncRun, error)

	CreateWebhookRegistration(context.Context, types.WebhookRegistration) error
	GetWebhookRegistrationByIntegration(context.Context, string) (types.WebhookRegistration, error)
	ListWebhookRegistrations(context.Context, WebhookRegistrationQuery) ([]types.WebhookRegistration, error)
	UpdateWebhookRegistration(context.Context, types.WebhookRegistration) error

	UpsertRepository(context.Context, types.Repository) error
	GetRepository(context.Context, string) (types.Repository, error)
	GetRepositoryByURL(context.Context, string, string) (types.Repository, error)
	ListRepositories(context.Context, RepositoryQuery) ([]types.Repository, error)
	UpdateRepository(context.Context, types.Repository) error

	UpsertDiscoveredResource(context.Context, types.DiscoveredResource) error
	GetDiscoveredResource(context.Context, string) (types.DiscoveredResource, error)
	ListDiscoveredResources(context.Context, DiscoveredResourceQuery) ([]types.DiscoveredResource, error)
	UpdateDiscoveredResource(context.Context, types.DiscoveredResource) error

	UpsertGraphRelationship(context.Context, types.GraphRelationship) error
	ListGraphRelationships(context.Context, GraphRelationshipQuery) ([]types.GraphRelationship, error)

	CreateUser(context.Context, types.User) error
	GetUser(context.Context, string) (types.User, error)
	GetUserByEmail(context.Context, string) (types.User, error)
	UpdateUser(context.Context, types.User) error

	CreateIdentityProvider(context.Context, types.IdentityProvider) error
	GetIdentityProvider(context.Context, string) (types.IdentityProvider, error)
	ListIdentityProviders(context.Context, IdentityProviderQuery) ([]types.IdentityProvider, error)
	UpdateIdentityProvider(context.Context, types.IdentityProvider) error

	CreateIdentityLink(context.Context, types.IdentityLink) error
	GetIdentityLinkBySubject(context.Context, string, string) (types.IdentityLink, error)
	ListIdentityLinksByUser(context.Context, string) ([]types.IdentityLink, error)
	UpdateIdentityLink(context.Context, types.IdentityLink) error

	CreateOrganizationMembership(context.Context, types.OrganizationMembership) error
	GetOrganizationMembership(context.Context, string, string) (types.OrganizationMembership, error)
	ListOrganizationMembershipsByUser(context.Context, string) ([]types.OrganizationMembership, error)

	CreateProjectMembership(context.Context, types.ProjectMembership) error
	GetProjectMembership(context.Context, string, string) (types.ProjectMembership, error)
	ListProjectMembershipsByUser(context.Context, string) ([]types.ProjectMembership, error)

	CreateServiceAccount(context.Context, types.ServiceAccount) error
	GetServiceAccount(context.Context, string) (types.ServiceAccount, error)
	ListServiceAccounts(context.Context, ServiceAccountQuery) ([]types.ServiceAccount, error)
	UpdateServiceAccount(context.Context, types.ServiceAccount) error

	CreateAPIToken(context.Context, types.APIToken) error
	GetAPIToken(context.Context, string) (types.APIToken, error)
	GetAPITokenByPrefix(context.Context, string) (types.APIToken, error)
	ListAPITokens(context.Context, APITokenQuery) ([]types.APIToken, error)
	UpdateAPIToken(context.Context, types.APIToken) error

	CreateBrowserSession(context.Context, types.BrowserSession) error
	GetBrowserSession(context.Context, string) (types.BrowserSession, error)
	GetBrowserSessionByHash(context.Context, string) (types.BrowserSession, error)
	ListBrowserSessions(context.Context, BrowserSessionQuery) ([]types.BrowserSession, error)
	UpdateBrowserSession(context.Context, types.BrowserSession) error

	CreateRolloutExecution(context.Context, types.RolloutExecution) error
	GetRolloutExecution(context.Context, string) (types.RolloutExecution, error)
	ListRolloutExecutions(context.Context, RolloutExecutionQuery) ([]types.RolloutExecution, error)
	UpdateRolloutExecution(context.Context, types.RolloutExecution) error
	ClaimRolloutExecution(context.Context, string, time.Time, time.Time) (bool, error)

	CreateVerificationResult(context.Context, types.VerificationResult) error
	ListVerificationResults(context.Context, VerificationResultQuery) ([]types.VerificationResult, error)

	CreateSignalSnapshot(context.Context, types.SignalSnapshot) error
	ListSignalSnapshots(context.Context, SignalSnapshotQuery) ([]types.SignalSnapshot, error)

	CreateRollbackPolicy(context.Context, types.RollbackPolicy) error
	GetRollbackPolicy(context.Context, string) (types.RollbackPolicy, error)
	ListRollbackPolicies(context.Context, RollbackPolicyQuery) ([]types.RollbackPolicy, error)
	UpdateRollbackPolicy(context.Context, types.RollbackPolicy) error

	CreatePolicy(context.Context, types.Policy) error
	GetPolicy(context.Context, string) (types.Policy, error)
	ListPolicies(context.Context, PolicyQuery) ([]types.Policy, error)
	UpdatePolicy(context.Context, types.Policy) error

	CreatePolicyDecision(context.Context, types.PolicyDecision) error
	ListPolicyDecisions(context.Context, PolicyDecisionQuery) ([]types.PolicyDecision, error)

	CreateStatusEvent(context.Context, types.StatusEvent) error
	GetStatusEvent(context.Context, string) (types.StatusEvent, error)
	ListStatusEvents(context.Context, StatusEventQuery) ([]types.StatusEvent, error)
	CountStatusEvents(context.Context, StatusEventQuery) (int, error)

	CreateOutboxEvent(context.Context, types.OutboxEvent) error
	GetOutboxEvent(context.Context, string) (types.OutboxEvent, error)
	ListOutboxEvents(context.Context, OutboxEventQuery) ([]types.OutboxEvent, error)
	ClaimOutboxEvents(context.Context, time.Time, int, time.Time) ([]types.OutboxEvent, error)
	UpdateOutboxEventIfStatus(context.Context, types.OutboxEvent, string) (bool, error)
	UpdateOutboxEvent(context.Context, types.OutboxEvent) error
}
