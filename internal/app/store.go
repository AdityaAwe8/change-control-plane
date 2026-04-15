package app

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type InMemoryStore struct {
	mu                  sync.RWMutex
	organizations       map[string]types.Organization
	projects            map[string]types.Project
	teams               map[string]types.Team
	services            map[string]types.Service
	environments        map[string]types.Environment
	changeSets          map[string]types.ChangeSet
	riskAssessments     map[string]types.RiskAssessment
	rolloutPlans        map[string]types.RolloutPlan
	rolloutExecutions   map[string]types.RolloutExecution
	verificationResults map[string]types.VerificationResult
	auditEvents         map[string]types.AuditEvent
	integrations        map[string]types.Integration
	repositories        map[string]types.Repository
	graphRelationships  map[string]types.GraphRelationship
	users               map[string]types.User
	usersByEmail        map[string]string
	orgMemberships      map[string]types.OrganizationMembership
	projectMemberships  map[string]types.ProjectMembership
	serviceAccounts     map[string]types.ServiceAccount
	apiTokens           map[string]types.APIToken
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		organizations:       make(map[string]types.Organization),
		projects:            make(map[string]types.Project),
		teams:               make(map[string]types.Team),
		services:            make(map[string]types.Service),
		environments:        make(map[string]types.Environment),
		changeSets:          make(map[string]types.ChangeSet),
		riskAssessments:     make(map[string]types.RiskAssessment),
		rolloutPlans:        make(map[string]types.RolloutPlan),
		rolloutExecutions:   make(map[string]types.RolloutExecution),
		verificationResults: make(map[string]types.VerificationResult),
		auditEvents:         make(map[string]types.AuditEvent),
		integrations:        make(map[string]types.Integration),
		repositories:        make(map[string]types.Repository),
		graphRelationships:  make(map[string]types.GraphRelationship),
		users:               make(map[string]types.User),
		usersByEmail:        make(map[string]string),
		orgMemberships:      make(map[string]types.OrganizationMembership),
		projectMemberships:  make(map[string]types.ProjectMembership),
		serviceAccounts:     make(map[string]types.ServiceAccount),
		apiTokens:           make(map[string]types.APIToken),
	}
}

func (s *InMemoryStore) Close() error {
	return nil
}

func (s *InMemoryStore) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (s *InMemoryStore) CreateOrganization(_ context.Context, org types.Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.organizations[org.ID] = org
	return nil
}

func (s *InMemoryStore) GetOrganization(_ context.Context, id string) (types.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	org, ok := s.organizations[id]
	if !ok {
		return types.Organization{}, storage.ErrNotFound
	}
	return org, nil
}

func (s *InMemoryStore) GetOrganizationBySlug(_ context.Context, slug string) (types.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, org := range s.organizations {
		if org.Slug == slug {
			return org, nil
		}
	}
	return types.Organization{}, storage.ErrNotFound
}

func (s *InMemoryStore) ListOrganizations(_ context.Context, query storage.OrganizationQuery) ([]types.Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.organizations, func(item types.Organization) bool {
		return matchIDs(query.IDs, item.ID)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateOrganization(_ context.Context, org types.Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.organizations[org.ID]; !ok {
		return storage.ErrNotFound
	}
	s.organizations[org.ID] = org
	return nil
}

func (s *InMemoryStore) CreateProject(_ context.Context, project types.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[project.ID] = project
	return nil
}

func (s *InMemoryStore) GetProject(_ context.Context, id string) (types.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	project, ok := s.projects[id]
	if !ok {
		return types.Project{}, storage.ErrNotFound
	}
	return project, nil
}

func (s *InMemoryStore) ListProjects(_ context.Context, query storage.ProjectQuery) ([]types.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.projects, func(item types.Project) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		return matchIDs(query.IDs, item.ID)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateProject(_ context.Context, project types.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[project.ID]; !ok {
		return storage.ErrNotFound
	}
	s.projects[project.ID] = project
	return nil
}

func (s *InMemoryStore) CreateTeam(_ context.Context, team types.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.teams[team.ID] = team
	return nil
}

func (s *InMemoryStore) GetTeam(_ context.Context, id string) (types.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	team, ok := s.teams[id]
	if !ok {
		return types.Team{}, storage.ErrNotFound
	}
	return team, nil
}

func (s *InMemoryStore) ListTeams(_ context.Context, query storage.TeamQuery) ([]types.Team, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.teams, func(item types.Team) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		return matchIDs(query.IDs, item.ID)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateTeam(_ context.Context, team types.Team) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.teams[team.ID]; !ok {
		return storage.ErrNotFound
	}
	s.teams[team.ID] = team
	return nil
}

func (s *InMemoryStore) CreateService(_ context.Context, service types.Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[service.ID] = service
	return nil
}

func (s *InMemoryStore) GetService(_ context.Context, id string) (types.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	service, ok := s.services[id]
	if !ok {
		return types.Service{}, storage.ErrNotFound
	}
	return service, nil
}

func (s *InMemoryStore) ListServices(_ context.Context, query storage.ServiceQuery) ([]types.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.services, func(item types.Service) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		if query.TeamID != "" && item.TeamID != query.TeamID {
			return false
		}
		return matchIDs(query.IDs, item.ID)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateService(_ context.Context, service types.Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[service.ID]; !ok {
		return storage.ErrNotFound
	}
	s.services[service.ID] = service
	return nil
}

func (s *InMemoryStore) CreateEnvironment(_ context.Context, environment types.Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.environments[environment.ID] = environment
	return nil
}

func (s *InMemoryStore) GetEnvironment(_ context.Context, id string) (types.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	environment, ok := s.environments[id]
	if !ok {
		return types.Environment{}, storage.ErrNotFound
	}
	return environment, nil
}

func (s *InMemoryStore) ListEnvironments(_ context.Context, query storage.EnvironmentQuery) ([]types.Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.environments, func(item types.Environment) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		return matchIDs(query.IDs, item.ID)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateEnvironment(_ context.Context, environment types.Environment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.environments[environment.ID]; !ok {
		return storage.ErrNotFound
	}
	s.environments[environment.ID] = environment
	return nil
}

func (s *InMemoryStore) CreateChangeSet(_ context.Context, change types.ChangeSet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changeSets[change.ID] = change
	return nil
}

func (s *InMemoryStore) GetChangeSet(_ context.Context, id string) (types.ChangeSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	change, ok := s.changeSets[id]
	if !ok {
		return types.ChangeSet{}, storage.ErrNotFound
	}
	return change, nil
}

func (s *InMemoryStore) ListChangeSets(_ context.Context, query storage.ChangeSetQuery) ([]types.ChangeSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.changeSets, func(item types.ChangeSet) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		if query.ServiceID != "" && item.ServiceID != query.ServiceID {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) CreateRiskAssessment(_ context.Context, assessment types.RiskAssessment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.riskAssessments[assessment.ID] = assessment
	return nil
}

func (s *InMemoryStore) GetRiskAssessment(_ context.Context, id string) (types.RiskAssessment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	assessment, ok := s.riskAssessments[id]
	if !ok {
		return types.RiskAssessment{}, storage.ErrNotFound
	}
	return assessment, nil
}

func (s *InMemoryStore) ListRiskAssessments(_ context.Context, query storage.RiskAssessmentQuery) ([]types.RiskAssessment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.riskAssessments, func(item types.RiskAssessment) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		if query.ChangeSetID != "" && item.ChangeSetID != query.ChangeSetID {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) CreateRolloutPlan(_ context.Context, plan types.RolloutPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rolloutPlans[plan.ID] = plan
	return nil
}

func (s *InMemoryStore) GetRolloutPlan(_ context.Context, id string) (types.RolloutPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	plan, ok := s.rolloutPlans[id]
	if !ok {
		return types.RolloutPlan{}, storage.ErrNotFound
	}
	return plan, nil
}

func (s *InMemoryStore) ListRolloutPlans(_ context.Context, query storage.RolloutPlanQuery) ([]types.RolloutPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.rolloutPlans, func(item types.RolloutPlan) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		if query.ChangeSetID != "" && item.ChangeSetID != query.ChangeSetID {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) CreateAuditEvent(_ context.Context, event types.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auditEvents[event.ID] = event
	return nil
}

func (s *InMemoryStore) ListAuditEvents(_ context.Context, query storage.AuditEventQuery) ([]types.AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.auditEvents, func(item types.AuditEvent) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) CreateIntegration(_ context.Context, integration types.Integration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.integrations[integration.ID] = integration
	return nil
}

func (s *InMemoryStore) GetIntegration(_ context.Context, id string) (types.Integration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	integration, ok := s.integrations[id]
	if !ok {
		return types.Integration{}, storage.ErrNotFound
	}
	return integration, nil
}

func (s *InMemoryStore) UpsertIntegration(_ context.Context, integration types.Integration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.integrations[integration.ID] = integration
	return nil
}

func (s *InMemoryStore) ListIntegrations(_ context.Context, query storage.IntegrationQuery) ([]types.Integration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.integrations, func(item types.Integration) bool {
		if query.OrganizationID != "" && item.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateIntegration(_ context.Context, integration types.Integration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.integrations[integration.ID]; !ok {
		return storage.ErrNotFound
	}
	s.integrations[integration.ID] = integration
	return nil
}

func (s *InMemoryStore) UpsertRepository(_ context.Context, repository types.Repository) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repositories[repository.ID] = repository
	return nil
}

func (s *InMemoryStore) GetRepositoryByURL(_ context.Context, organizationID, url string) (types.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, repository := range s.repositories {
		if repository.OrganizationID == organizationID && repository.URL == url {
			return repository, nil
		}
	}
	return types.Repository{}, storage.ErrNotFound
}

func (s *InMemoryStore) ListRepositories(_ context.Context, query storage.RepositoryQuery) ([]types.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.repositories, func(item types.Repository) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpsertGraphRelationship(_ context.Context, relationship types.GraphRelationship) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.graphRelationships[relationship.ID] = relationship
	return nil
}

func (s *InMemoryStore) ListGraphRelationships(_ context.Context, query storage.GraphRelationshipQuery) ([]types.GraphRelationship, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.graphRelationships, func(item types.GraphRelationship) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.SourceIntegrationID != "" && item.SourceIntegrationID != query.SourceIntegrationID {
			return false
		}
		if query.RelationshipType != "" && item.RelationshipType != query.RelationshipType {
			return false
		}
		if query.FromResourceID != "" && item.FromResourceID != query.FromResourceID {
			return false
		}
		if query.ToResourceID != "" && item.ToResourceID != query.ToResourceID {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) CreateUser(_ context.Context, user types.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.ID] = user
	s.usersByEmail[user.Email] = user.ID
	return nil
}

func (s *InMemoryStore) GetUser(_ context.Context, id string) (types.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return types.User{}, storage.ErrNotFound
	}
	return user, nil
}

func (s *InMemoryStore) GetUserByEmail(_ context.Context, email string) (types.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.usersByEmail[email]
	if !ok {
		return types.User{}, storage.ErrNotFound
	}
	return s.users[id], nil
}

func (s *InMemoryStore) CreateOrganizationMembership(_ context.Context, membership types.OrganizationMembership) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orgMemberships[membership.ID] = membership
	return nil
}

func (s *InMemoryStore) GetOrganizationMembership(_ context.Context, userID, organizationID string) (types.OrganizationMembership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, membership := range s.orgMemberships {
		if membership.UserID == userID && membership.OrganizationID == organizationID {
			return membership, nil
		}
	}
	return types.OrganizationMembership{}, storage.ErrNotFound
}

func (s *InMemoryStore) ListOrganizationMembershipsByUser(_ context.Context, userID string) ([]types.OrganizationMembership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.orgMemberships, func(item types.OrganizationMembership) bool {
		return item.UserID == userID
	})
	return items, nil
}

func (s *InMemoryStore) CreateProjectMembership(_ context.Context, membership types.ProjectMembership) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projectMemberships[membership.ID] = membership
	return nil
}

func (s *InMemoryStore) GetProjectMembership(_ context.Context, userID, projectID string) (types.ProjectMembership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, membership := range s.projectMemberships {
		if membership.UserID == userID && membership.ProjectID == projectID {
			return membership, nil
		}
	}
	return types.ProjectMembership{}, storage.ErrNotFound
}

func (s *InMemoryStore) ListProjectMembershipsByUser(_ context.Context, userID string) ([]types.ProjectMembership, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.projectMemberships, func(item types.ProjectMembership) bool {
		return item.UserID == userID
	})
	return items, nil
}

func (s *InMemoryStore) CreateServiceAccount(_ context.Context, serviceAccount types.ServiceAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serviceAccounts[serviceAccount.ID] = serviceAccount
	return nil
}

func (s *InMemoryStore) GetServiceAccount(_ context.Context, id string) (types.ServiceAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	serviceAccount, ok := s.serviceAccounts[id]
	if !ok {
		return types.ServiceAccount{}, storage.ErrNotFound
	}
	return serviceAccount, nil
}

func (s *InMemoryStore) ListServiceAccounts(_ context.Context, query storage.ServiceAccountQuery) ([]types.ServiceAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.serviceAccounts, func(item types.ServiceAccount) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.Status != "" && item.Status != query.Status {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateServiceAccount(_ context.Context, serviceAccount types.ServiceAccount) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.serviceAccounts[serviceAccount.ID]; !ok {
		return storage.ErrNotFound
	}
	s.serviceAccounts[serviceAccount.ID] = serviceAccount
	return nil
}

func (s *InMemoryStore) CreateAPIToken(_ context.Context, token types.APIToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiTokens[token.ID] = token
	return nil
}

func (s *InMemoryStore) GetAPIToken(_ context.Context, id string) (types.APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.apiTokens[id]
	if !ok {
		return types.APIToken{}, storage.ErrNotFound
	}
	return token, nil
}

func (s *InMemoryStore) GetAPITokenByPrefix(_ context.Context, prefix string) (types.APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, token := range s.apiTokens {
		if token.TokenPrefix == prefix {
			return token, nil
		}
	}
	return types.APIToken{}, storage.ErrNotFound
}

func (s *InMemoryStore) ListAPITokens(_ context.Context, query storage.APITokenQuery) ([]types.APIToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.apiTokens, func(item types.APIToken) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ServiceAccountID != "" && item.ServiceAccountID != query.ServiceAccountID {
			return false
		}
		if query.Status != "" && item.Status != query.Status {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateAPIToken(_ context.Context, token types.APIToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apiTokens[token.ID]; !ok {
		return storage.ErrNotFound
	}
	s.apiTokens[token.ID] = token
	return nil
}

func (s *InMemoryStore) CreateRolloutExecution(_ context.Context, execution types.RolloutExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rolloutExecutions[execution.ID] = execution
	return nil
}

func (s *InMemoryStore) GetRolloutExecution(_ context.Context, id string) (types.RolloutExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	execution, ok := s.rolloutExecutions[id]
	if !ok {
		return types.RolloutExecution{}, storage.ErrNotFound
	}
	return execution, nil
}

func (s *InMemoryStore) ListRolloutExecutions(_ context.Context, query storage.RolloutExecutionQuery) ([]types.RolloutExecution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.rolloutExecutions, func(item types.RolloutExecution) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		if query.ServiceID != "" && item.ServiceID != query.ServiceID {
			return false
		}
		if query.EnvironmentID != "" && item.EnvironmentID != query.EnvironmentID {
			return false
		}
		if query.Status != "" && item.Status != query.Status {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateRolloutExecution(_ context.Context, execution types.RolloutExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rolloutExecutions[execution.ID]; !ok {
		return storage.ErrNotFound
	}
	s.rolloutExecutions[execution.ID] = execution
	return nil
}

func (s *InMemoryStore) CreateVerificationResult(_ context.Context, result types.VerificationResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.verificationResults[result.ID] = result
	return nil
}

func (s *InMemoryStore) ListVerificationResults(_ context.Context, query storage.VerificationResultQuery) ([]types.VerificationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.verificationResults, func(item types.VerificationResult) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		if query.RolloutExecutionID != "" && item.RolloutExecutionID != query.RolloutExecutionID {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func sortedValues[T interface{ CreatedTime() time.Time }](items map[string]T) []T {
	return filterSortedValues(items, func(T) bool { return true })
}

func filterSortedValues[T interface{ CreatedTime() time.Time }](items map[string]T, match func(T) bool) []T {
	values := make([]T, 0, len(items))
	for _, item := range items {
		if match(item) {
			values = append(values, item)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].CreatedTime().Before(values[j].CreatedTime())
	})
	return values
}

func matchIDs(ids []string, id string) bool {
	if len(ids) == 0 {
		return true
	}
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func paginate[T any](items []T, offset, limit int) []T {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []T{}
	}
	if limit <= 0 {
		return items[offset:]
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}
