package app

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type InMemoryStore struct {
	mu                   sync.RWMutex
	organizations        map[string]types.Organization
	projects             map[string]types.Project
	teams                map[string]types.Team
	services             map[string]types.Service
	environments         map[string]types.Environment
	changeSets           map[string]types.ChangeSet
	riskAssessments      map[string]types.RiskAssessment
	rolloutPlans         map[string]types.RolloutPlan
	rolloutExecutions    map[string]types.RolloutExecution
	verificationResults  map[string]types.VerificationResult
	signalSnapshots      map[string]types.SignalSnapshot
	auditEvents          map[string]types.AuditEvent
	integrations         map[string]types.Integration
	integrationSyncRuns  map[string]types.IntegrationSyncRun
	repositories         map[string]types.Repository
	discoveredResources  map[string]types.DiscoveredResource
	graphRelationships   map[string]types.GraphRelationship
	users                map[string]types.User
	usersByEmail         map[string]string
	identityProviders    map[string]types.IdentityProvider
	identityLinks        map[string]types.IdentityLink
	orgMemberships       map[string]types.OrganizationMembership
	projectMemberships   map[string]types.ProjectMembership
	serviceAccounts      map[string]types.ServiceAccount
	apiTokens            map[string]types.APIToken
	browserSessions      map[string]types.BrowserSession
	webhookRegistrations map[string]types.WebhookRegistration
	policies             map[string]types.Policy
	policyDecisions      map[string]types.PolicyDecision
	rollbackPolicies     map[string]types.RollbackPolicy
	statusEvents         map[string]types.StatusEvent
	outboxEvents         map[string]types.OutboxEvent
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		organizations:        make(map[string]types.Organization),
		projects:             make(map[string]types.Project),
		teams:                make(map[string]types.Team),
		services:             make(map[string]types.Service),
		environments:         make(map[string]types.Environment),
		changeSets:           make(map[string]types.ChangeSet),
		riskAssessments:      make(map[string]types.RiskAssessment),
		rolloutPlans:         make(map[string]types.RolloutPlan),
		rolloutExecutions:    make(map[string]types.RolloutExecution),
		verificationResults:  make(map[string]types.VerificationResult),
		signalSnapshots:      make(map[string]types.SignalSnapshot),
		auditEvents:          make(map[string]types.AuditEvent),
		integrations:         make(map[string]types.Integration),
		integrationSyncRuns:  make(map[string]types.IntegrationSyncRun),
		repositories:         make(map[string]types.Repository),
		discoveredResources:  make(map[string]types.DiscoveredResource),
		graphRelationships:   make(map[string]types.GraphRelationship),
		users:                make(map[string]types.User),
		usersByEmail:         make(map[string]string),
		identityProviders:    make(map[string]types.IdentityProvider),
		identityLinks:        make(map[string]types.IdentityLink),
		orgMemberships:       make(map[string]types.OrganizationMembership),
		projectMemberships:   make(map[string]types.ProjectMembership),
		serviceAccounts:      make(map[string]types.ServiceAccount),
		apiTokens:            make(map[string]types.APIToken),
		browserSessions:      make(map[string]types.BrowserSession),
		webhookRegistrations: make(map[string]types.WebhookRegistration),
		policies:             make(map[string]types.Policy),
		policyDecisions:      make(map[string]types.PolicyDecision),
		rollbackPolicies:     make(map[string]types.RollbackPolicy),
		statusEvents:         make(map[string]types.StatusEvent),
		outboxEvents:         make(map[string]types.OutboxEvent),
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
		if query.ResourceType != "" && item.ResourceType != query.ResourceType {
			return false
		}
		if query.ResourceID != "" && item.ResourceID != query.ResourceID {
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
		if query.Kind != "" && item.Kind != query.Kind {
			return false
		}
		if query.InstanceKey != "" && item.InstanceKey != query.InstanceKey {
			return false
		}
		if query.ScopeType != "" && item.ScopeType != query.ScopeType {
			return false
		}
		if query.AuthStrategy != "" && item.AuthStrategy != query.AuthStrategy {
			return false
		}
		if query.Enabled != nil && item.Enabled != *query.Enabled {
			return false
		}
		if query.Search != "" {
			haystack := strings.ToLower(item.Name + " " + item.Kind + " " + item.ScopeName)
			if !strings.Contains(haystack, strings.ToLower(query.Search)) {
				return false
			}
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

func (s *InMemoryStore) ClaimIntegrationSync(_ context.Context, id string, dueBefore, staleClaimBefore, claimedAt time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	integration, ok := s.integrations[id]
	if !ok {
		return false, storage.ErrNotFound
	}
	if !integration.Enabled || !integration.ScheduleEnabled || integration.NextScheduledSyncAt == nil || integration.NextScheduledSyncAt.After(dueBefore) {
		return false, nil
	}
	if integration.SyncClaimedAt != nil && !integration.SyncClaimedAt.Before(staleClaimBefore) {
		return false, nil
	}
	integration.SyncClaimedAt = &claimedAt
	integration.UpdatedAt = claimedAt
	s.integrations[id] = integration
	return true, nil
}

func (s *InMemoryStore) CreateIntegrationSyncRun(_ context.Context, run types.IntegrationSyncRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.integrationSyncRuns[run.ID] = run
	return nil
}

func (s *InMemoryStore) ListIntegrationSyncRuns(_ context.Context, query storage.IntegrationSyncRunQuery) ([]types.IntegrationSyncRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.integrationSyncRuns, func(item types.IntegrationSyncRun) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.IntegrationID != "" && item.IntegrationID != query.IntegrationID {
			return false
		}
		if query.Operation != "" && item.Operation != query.Operation {
			return false
		}
		if query.Trigger != "" && item.Trigger != query.Trigger {
			return false
		}
		if query.Status != "" && item.Status != query.Status {
			return false
		}
		if query.ExternalEventID != "" && item.ExternalEventID != query.ExternalEventID {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) CreateWebhookRegistration(_ context.Context, registration types.WebhookRegistration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhookRegistrations[registration.ID] = registration
	return nil
}

func (s *InMemoryStore) GetWebhookRegistrationByIntegration(_ context.Context, integrationID string) (types.WebhookRegistration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, registration := range s.webhookRegistrations {
		if registration.IntegrationID == integrationID {
			return registration, nil
		}
	}
	return types.WebhookRegistration{}, storage.ErrNotFound
}

func (s *InMemoryStore) ListWebhookRegistrations(_ context.Context, query storage.WebhookRegistrationQuery) ([]types.WebhookRegistration, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.webhookRegistrations, func(item types.WebhookRegistration) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.IntegrationID != "" && item.IntegrationID != query.IntegrationID {
			return false
		}
		if query.ProviderKind != "" && item.ProviderKind != query.ProviderKind {
			return false
		}
		if query.Status != "" && item.Status != query.Status {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateWebhookRegistration(_ context.Context, registration types.WebhookRegistration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.webhookRegistrations[registration.ID]; !ok {
		return storage.ErrNotFound
	}
	s.webhookRegistrations[registration.ID] = registration
	return nil
}

func (s *InMemoryStore) UpsertRepository(_ context.Context, repository types.Repository) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repositories[repository.ID] = repository
	return nil
}

func (s *InMemoryStore) GetRepository(_ context.Context, id string) (types.Repository, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	repository, ok := s.repositories[id]
	if !ok {
		return types.Repository{}, storage.ErrNotFound
	}
	return repository, nil
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
		if query.ServiceID != "" && item.ServiceID != query.ServiceID {
			return false
		}
		if query.EnvironmentID != "" && item.EnvironmentID != query.EnvironmentID {
			return false
		}
		if query.SourceIntegrationID != "" && item.SourceIntegrationID != query.SourceIntegrationID {
			return false
		}
		if query.Provider != "" && item.Provider != query.Provider {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateRepository(_ context.Context, repository types.Repository) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.repositories[repository.ID]; !ok {
		return storage.ErrNotFound
	}
	s.repositories[repository.ID] = repository
	return nil
}

func (s *InMemoryStore) UpsertDiscoveredResource(_ context.Context, resource types.DiscoveredResource) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discoveredResources[resource.ID] = resource
	return nil
}

func (s *InMemoryStore) GetDiscoveredResource(_ context.Context, id string) (types.DiscoveredResource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resource, ok := s.discoveredResources[id]
	if !ok {
		return types.DiscoveredResource{}, storage.ErrNotFound
	}
	return resource, nil
}

func (s *InMemoryStore) ListDiscoveredResources(_ context.Context, query storage.DiscoveredResourceQuery) ([]types.DiscoveredResource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.discoveredResources, func(item types.DiscoveredResource) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.IntegrationID != "" && item.IntegrationID != query.IntegrationID {
			return false
		}
		if query.ResourceType != "" && item.ResourceType != query.ResourceType {
			return false
		}
		if query.Provider != "" && item.Provider != query.Provider {
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
		if query.RepositoryID != "" && item.RepositoryID != query.RepositoryID {
			return false
		}
		if query.Status != "" && item.Status != query.Status {
			return false
		}
		if query.UnmappedOnly && (item.ServiceID != "" && item.EnvironmentID != "") {
			return false
		}
		if query.Search != "" {
			haystack := strings.ToLower(item.Name + " " + item.Namespace + " " + item.ExternalID + " " + item.Summary)
			if !strings.Contains(haystack, strings.ToLower(query.Search)) {
				return false
			}
		}
		return true
	})
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateDiscoveredResource(_ context.Context, resource types.DiscoveredResource) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.discoveredResources[resource.ID]; !ok {
		return storage.ErrNotFound
	}
	s.discoveredResources[resource.ID] = resource
	return nil
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

func (s *InMemoryStore) UpdateUser(_ context.Context, user types.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[user.ID]; !ok {
		return storage.ErrNotFound
	}
	s.users[user.ID] = user
	s.usersByEmail[user.Email] = user.ID
	return nil
}

func (s *InMemoryStore) CreateIdentityProvider(_ context.Context, provider types.IdentityProvider) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identityProviders[provider.ID] = provider
	return nil
}

func (s *InMemoryStore) GetIdentityProvider(_ context.Context, id string) (types.IdentityProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	provider, ok := s.identityProviders[id]
	if !ok {
		return types.IdentityProvider{}, storage.ErrNotFound
	}
	return provider, nil
}

func (s *InMemoryStore) ListIdentityProviders(_ context.Context, query storage.IdentityProviderQuery) ([]types.IdentityProvider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.identityProviders, func(item types.IdentityProvider) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.Kind != "" && item.Kind != query.Kind {
			return false
		}
		if query.Enabled != nil && item.Enabled != *query.Enabled {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateIdentityProvider(_ context.Context, provider types.IdentityProvider) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.identityProviders[provider.ID]; !ok {
		return storage.ErrNotFound
	}
	s.identityProviders[provider.ID] = provider
	return nil
}

func (s *InMemoryStore) CreateIdentityLink(_ context.Context, link types.IdentityLink) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identityLinks[link.ID] = link
	return nil
}

func (s *InMemoryStore) GetIdentityLinkBySubject(_ context.Context, providerID, subject string) (types.IdentityLink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, link := range s.identityLinks {
		if link.ProviderID == providerID && link.ExternalSubject == subject {
			return link, nil
		}
	}
	return types.IdentityLink{}, storage.ErrNotFound
}

func (s *InMemoryStore) ListIdentityLinksByUser(_ context.Context, userID string) ([]types.IdentityLink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.identityLinks, func(item types.IdentityLink) bool {
		return item.UserID == userID
	})
	return items, nil
}

func (s *InMemoryStore) UpdateIdentityLink(_ context.Context, link types.IdentityLink) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.identityLinks[link.ID]; !ok {
		return storage.ErrNotFound
	}
	s.identityLinks[link.ID] = link
	return nil
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

func (s *InMemoryStore) CreateBrowserSession(_ context.Context, session types.BrowserSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.browserSessions[session.ID] = session
	return nil
}

func (s *InMemoryStore) GetBrowserSessionByHash(_ context.Context, sessionHash string) (types.BrowserSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, session := range s.browserSessions {
		if session.SessionHash == sessionHash {
			return session, nil
		}
	}
	return types.BrowserSession{}, storage.ErrNotFound
}

func (s *InMemoryStore) UpdateBrowserSession(_ context.Context, session types.BrowserSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.browserSessions[session.ID]; !ok {
		return storage.ErrNotFound
	}
	s.browserSessions[session.ID] = session
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

func (s *InMemoryStore) ClaimRolloutExecution(_ context.Context, id string, staleBefore, claimedAt time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	execution, ok := s.rolloutExecutions[id]
	if !ok {
		return false, storage.ErrNotFound
	}
	if execution.LastReconciledAt != nil && !execution.LastReconciledAt.Before(staleBefore) {
		return false, nil
	}
	execution.LastReconciledAt = &claimedAt
	execution.UpdatedAt = claimedAt
	s.rolloutExecutions[id] = execution
	return true, nil
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

func (s *InMemoryStore) CreateSignalSnapshot(_ context.Context, snapshot types.SignalSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.signalSnapshots[snapshot.ID] = snapshot
	return nil
}

func (s *InMemoryStore) ListSignalSnapshots(_ context.Context, query storage.SignalSnapshotQuery) ([]types.SignalSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.signalSnapshots, func(item types.SignalSnapshot) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		if query.RolloutExecutionID != "" && item.RolloutExecutionID != query.RolloutExecutionID {
			return false
		}
		if query.ProviderType != "" && item.ProviderType != query.ProviderType {
			return false
		}
		return true
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) CreatePolicy(_ context.Context, policy types.Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[policy.ID] = policy
	return nil
}

func (s *InMemoryStore) GetPolicy(_ context.Context, id string) (types.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, ok := s.policies[id]
	if !ok {
		return types.Policy{}, storage.ErrNotFound
	}
	return policy, nil
}

func (s *InMemoryStore) ListPolicies(_ context.Context, query storage.PolicyQuery) ([]types.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.policies, func(item types.Policy) bool {
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
		if query.AppliesTo != "" && item.AppliesTo != query.AppliesTo {
			return false
		}
		if query.EnabledOnly && !item.Enabled {
			return false
		}
		return true
	})
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Priority == items[j].Priority {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].Priority > items[j].Priority
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdatePolicy(_ context.Context, policy types.Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policies[policy.ID]; !ok {
		return storage.ErrNotFound
	}
	s.policies[policy.ID] = policy
	return nil
}

func (s *InMemoryStore) CreatePolicyDecision(_ context.Context, decision types.PolicyDecision) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policyDecisions[decision.ID] = decision
	return nil
}

func (s *InMemoryStore) ListPolicyDecisions(_ context.Context, query storage.PolicyDecisionQuery) ([]types.PolicyDecision, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.policyDecisions, func(item types.PolicyDecision) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		if query.PolicyID != "" && item.PolicyID != query.PolicyID {
			return false
		}
		if query.ChangeSetID != "" && item.ChangeSetID != query.ChangeSetID {
			return false
		}
		if query.RiskAssessmentID != "" && item.RiskAssessmentID != query.RiskAssessmentID {
			return false
		}
		if query.RolloutPlanID != "" && item.RolloutPlanID != query.RolloutPlanID {
			return false
		}
		if query.RolloutExecutionID != "" && item.RolloutExecutionID != query.RolloutExecutionID {
			return false
		}
		if query.AppliesTo != "" && item.AppliesTo != query.AppliesTo {
			return false
		}
		return true
	})
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) CreateRollbackPolicy(_ context.Context, policy types.RollbackPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rollbackPolicies[policy.ID] = policy
	return nil
}

func (s *InMemoryStore) GetRollbackPolicy(_ context.Context, id string) (types.RollbackPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, ok := s.rollbackPolicies[id]
	if !ok {
		return types.RollbackPolicy{}, storage.ErrNotFound
	}
	return policy, nil
}

func (s *InMemoryStore) ListRollbackPolicies(_ context.Context, query storage.RollbackPolicyQuery) ([]types.RollbackPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.rollbackPolicies, func(item types.RollbackPolicy) bool {
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
		if query.EnabledOnly && !item.Enabled {
			return false
		}
		return true
	})
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) UpdateRollbackPolicy(_ context.Context, policy types.RollbackPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.rollbackPolicies[policy.ID]; !ok {
		return storage.ErrNotFound
	}
	s.rollbackPolicies[policy.ID] = policy
	return nil
}

func (s *InMemoryStore) CreateStatusEvent(_ context.Context, event types.StatusEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusEvents[event.ID] = event
	return nil
}

func (s *InMemoryStore) GetStatusEvent(_ context.Context, id string) (types.StatusEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.statusEvents[id]
	if !ok {
		return types.StatusEvent{}, storage.ErrNotFound
	}
	return event, nil
}

func (s *InMemoryStore) ListStatusEvents(_ context.Context, query storage.StatusEventQuery) ([]types.StatusEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.statusEvents, func(item types.StatusEvent) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.ProjectID != "" && item.ProjectID != query.ProjectID {
			return false
		}
		if query.TeamID != "" && item.TeamID != query.TeamID {
			return false
		}
		if query.ServiceID != "" && item.ServiceID != query.ServiceID {
			return false
		}
		if query.EnvironmentID != "" && item.EnvironmentID != query.EnvironmentID {
			return false
		}
		if query.RolloutExecutionID != "" && item.RolloutExecutionID != query.RolloutExecutionID {
			return false
		}
		if query.ChangeSetID != "" && item.ChangeSetID != query.ChangeSetID {
			return false
		}
		if query.ResourceType != "" && item.ResourceType != query.ResourceType {
			return false
		}
		if query.ResourceID != "" && item.ResourceID != query.ResourceID {
			return false
		}
		if len(query.EventTypes) > 0 && !matchIDs(query.EventTypes, item.EventType) {
			return false
		}
		if query.ActorType != "" && item.ActorType != query.ActorType {
			return false
		}
		if query.ActorID != "" && item.ActorID != query.ActorID {
			return false
		}
		if query.Source != "" && item.Source != query.Source {
			return false
		}
		if query.Outcome != "" && item.Outcome != query.Outcome {
			return false
		}
		if query.Automated != nil && item.Automated != *query.Automated {
			return false
		}
		if query.RollbackOnly && !strings.Contains(strings.ToLower(item.EventType), "rollback") && item.NewState != "rolled_back" && !strings.Contains(strings.ToLower(item.Summary), "rollback") {
			return false
		}
		if query.Since != nil && item.CreatedAt.Before(*query.Since) {
			return false
		}
		if query.Until != nil && item.CreatedAt.After(*query.Until) {
			return false
		}
		if query.Search != "" {
			haystack := strings.ToLower(item.EventType + " " + item.Summary + " " + strings.Join(item.Explanation, " "))
			if !strings.Contains(haystack, strings.ToLower(query.Search)) {
				return false
			}
		}
		return true
	})
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) CountStatusEvents(ctx context.Context, query storage.StatusEventQuery) (int, error) {
	items, err := s.ListStatusEvents(ctx, storage.StatusEventQuery{
		OrganizationID:     query.OrganizationID,
		ProjectID:          query.ProjectID,
		TeamID:             query.TeamID,
		ServiceID:          query.ServiceID,
		EnvironmentID:      query.EnvironmentID,
		RolloutExecutionID: query.RolloutExecutionID,
		ChangeSetID:        query.ChangeSetID,
		ResourceType:       query.ResourceType,
		ResourceID:         query.ResourceID,
		EventTypes:         query.EventTypes,
		ActorType:          query.ActorType,
		ActorID:            query.ActorID,
		Source:             query.Source,
		Outcome:            query.Outcome,
		Automated:          query.Automated,
		RollbackOnly:       query.RollbackOnly,
		Search:             query.Search,
		Since:              query.Since,
		Until:              query.Until,
	})
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *InMemoryStore) CreateOutboxEvent(_ context.Context, event types.OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outboxEvents[event.ID] = event
	return nil
}

func (s *InMemoryStore) GetOutboxEvent(_ context.Context, id string) (types.OutboxEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, ok := s.outboxEvents[id]
	if !ok {
		return types.OutboxEvent{}, storage.ErrNotFound
	}
	return event, nil
}

func (s *InMemoryStore) ListOutboxEvents(_ context.Context, query storage.OutboxEventQuery) ([]types.OutboxEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := filterSortedValues(s.outboxEvents, func(item types.OutboxEvent) bool {
		if query.OrganizationID != "" && item.OrganizationID != query.OrganizationID {
			return false
		}
		if query.EventType != "" && item.EventType != query.EventType {
			return false
		}
		if query.Status != "" && item.Status != query.Status {
			return false
		}
		return true
	})
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return paginate(items, query.Offset, query.Limit), nil
}

func (s *InMemoryStore) ClaimOutboxEvents(_ context.Context, now time.Time, limit int, staleClaimBefore time.Time) ([]types.OutboxEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	type candidate struct {
		id    string
		event types.OutboxEvent
	}
	candidates := make([]candidate, 0, len(s.outboxEvents))
	for id, event := range s.outboxEvents {
		if event.Status == "processed" || event.Status == "dead_letter" {
			continue
		}
		if event.NextAttemptAt != nil && event.NextAttemptAt.After(now) {
			continue
		}
		if event.ClaimedAt != nil && !event.ClaimedAt.Before(staleClaimBefore) {
			continue
		}
		candidates = append(candidates, candidate{id: id, event: event})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].event.CreatedAt.Before(candidates[j].event.CreatedAt)
	})
	items := make([]types.OutboxEvent, 0, limit)
	for _, candidate := range candidates {
		event := candidate.event
		event.Status = "processing"
		event.ClaimedAt = &now
		event.UpdatedAt = now
		s.outboxEvents[candidate.id] = event
		items = append(items, event)
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items, nil
}

func (s *InMemoryStore) UpdateOutboxEvent(_ context.Context, event types.OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.outboxEvents[event.ID]; !ok {
		return storage.ErrNotFound
	}
	s.outboxEvents[event.ID] = event
	return nil
}

func (s *InMemoryStore) UpdateOutboxEventIfStatus(_ context.Context, event types.OutboxEvent, expectedStatus string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.outboxEvents[event.ID]
	if !ok {
		return false, nil
	}
	if current.Status != expectedStatus {
		return false, nil
	}
	s.outboxEvents[event.ID] = event
	return true, nil
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
