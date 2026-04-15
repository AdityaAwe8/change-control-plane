package storage

import (
	"context"
	"errors"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

var ErrNotFound = errors.New("resource not found")

type OrganizationQuery struct {
	IDs    []string
	Limit  int
	Offset int
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

type AuditEventQuery struct {
	OrganizationID string
	ProjectID      string
	Limit          int
	Offset         int
}

type IntegrationQuery struct {
	OrganizationID string
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

type RepositoryQuery struct {
	OrganizationID string
	ProjectID      string
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

	CreateAuditEvent(context.Context, types.AuditEvent) error
	ListAuditEvents(context.Context, AuditEventQuery) ([]types.AuditEvent, error)

	CreateIntegration(context.Context, types.Integration) error
	GetIntegration(context.Context, string) (types.Integration, error)
	UpsertIntegration(context.Context, types.Integration) error
	ListIntegrations(context.Context, IntegrationQuery) ([]types.Integration, error)
	UpdateIntegration(context.Context, types.Integration) error

	UpsertRepository(context.Context, types.Repository) error
	GetRepositoryByURL(context.Context, string, string) (types.Repository, error)
	ListRepositories(context.Context, RepositoryQuery) ([]types.Repository, error)

	UpsertGraphRelationship(context.Context, types.GraphRelationship) error
	ListGraphRelationships(context.Context, GraphRelationshipQuery) ([]types.GraphRelationship, error)

	CreateUser(context.Context, types.User) error
	GetUser(context.Context, string) (types.User, error)
	GetUserByEmail(context.Context, string) (types.User, error)

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

	CreateRolloutExecution(context.Context, types.RolloutExecution) error
	GetRolloutExecution(context.Context, string) (types.RolloutExecution, error)
	ListRolloutExecutions(context.Context, RolloutExecutionQuery) ([]types.RolloutExecution, error)
	UpdateRolloutExecution(context.Context, types.RolloutExecution) error

	CreateVerificationResult(context.Context, types.VerificationResult) error
	ListVerificationResults(context.Context, VerificationResultQuery) ([]types.VerificationResult, error)
}
