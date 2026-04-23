package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/audit"
	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/delivery"
	"github.com/change-control-plane/change-control-plane/internal/events"
	"github.com/change-control-plane/change-control-plane/internal/integrations"
	"github.com/change-control-plane/change-control-plane/internal/intelligence"
	policylib "github.com/change-control-plane/change-control-plane/internal/policies"
	"github.com/change-control-plane/change-control-plane/internal/risk"
	"github.com/change-control-plane/change-control-plane/internal/rollouts"
	"github.com/change-control-plane/change-control-plane/internal/status"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/internal/verification"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Application struct {
	Config        common.Config
	Store         storage.Store
	Events        events.Bus
	Audit         *audit.Service
	Status        *status.Service
	RiskEngine    *risk.Engine
	Planner       *rollouts.Planner
	Integrations  *integrations.Registry
	Intelligence  *intelligence.Client
	Orchestrators *delivery.Registry
	Signals       *verification.Registry
	Verifier      *verification.Engine
	Auth          *auth.Service
	Authorizer    *auth.Authorizer
}

const minPasswordLength = 8

const (
	demoAdminEmail             = "admin@changecontrolplane.local"
	demoAdminPassword          = "ChangeMe123!"
	demoAdminDisplayName       = "Sample Admin"
	demoAdminOrgName           = "Change Control Plane Demo"
	demoAdminOrgSlug           = "change-control-plane-demo"
	demoProjectSlug            = "control-plane-demo-platform"
	demoTeamSlug               = "control-plane-core"
	demoPrimaryServiceSlug     = "checkout-api"
	demoSupportServiceSlug     = "ledger-worker"
	demoProdEnvironmentSlug    = "production"
	demoStagingEnvironmentSlug = "staging"
	demoChangeSummary          = "Checkout reliability hardening and guarded rollout rehearsal"
)

func NewApplication(cfg common.Config) (*Application, error) {
	var store storage.Store
	switch strings.ToLower(cfg.StorageDriver) {
	case "", "memory":
		store = NewInMemoryStore()
	case "postgres":
		postgresStore, err := storage.NewPostgresStore(cfg)
		if err != nil {
			return nil, err
		}
		store = postgresStore
	default:
		return nil, fmt.Errorf("unsupported storage driver %q", cfg.StorageDriver)
	}
	return NewApplicationWithStore(cfg, store), nil
}

func NewApplicationWithStore(cfg common.Config, store storage.Store) *Application {
	bus := events.NewDurableBus(store)
	tokenService := auth.NewTokenService(cfg.AuthTokenSecret, time.Duration(cfg.AuthTokenTTL)*time.Minute)
	application := &Application{
		Config:        cfg,
		Store:         store,
		Events:        bus,
		Audit:         audit.NewService(store),
		Status:        status.NewService(store),
		RiskEngine:    risk.NewEngine(),
		Planner:       rollouts.NewPlanner(),
		Integrations:  integrations.NewRegistry(),
		Intelligence:  intelligence.NewClient(cfg),
		Orchestrators: delivery.NewRegistry(),
		Signals:       verification.NewRegistry(),
		Verifier:      verification.NewEngine(),
		Auth:          auth.NewService(store, tokenService),
		Authorizer:    auth.NewAuthorizer(),
	}
	_ = application.ensureDemoAdminSeeded(context.Background())
	return application
}

func (a *Application) Close() error {
	return a.Store.Close()
}

func (a *Application) SignUp(ctx context.Context, req types.SignUpRequest) (types.AuthResponse, error) {
	if a.Config.AuthMode != "dev" {
		return types.AuthResponse{}, ErrForbidden
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		return types.AuthResponse{}, fmt.Errorf("%w: email is required", ErrValidation)
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		return types.AuthResponse{}, fmt.Errorf("%w: display_name is required", ErrValidation)
	}
	if err := validatePassword(req.Password, req.PasswordConfirmation); err != nil {
		return types.AuthResponse{}, err
	}

	salt, hash, iterations, err := auth.HashPassword(req.Password)
	if err != nil {
		return types.AuthResponse{}, err
	}

	now := time.Now().UTC()
	user, err := a.Store.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
		if user.PasswordHash != "" {
			return types.AuthResponse{}, fmt.Errorf("%w: email already has an account", ErrValidation)
		}
		user.DisplayName = displayName
		user.Status = "active"
		user.PasswordSalt = salt
		user.PasswordHash = hash
		user.PasswordIterations = iterations
		user.UpdatedAt = now
		if err := a.Store.UpdateUser(ctx, user); err != nil {
			return types.AuthResponse{}, err
		}
	case errors.Is(err, storage.ErrNotFound):
		user = types.User{
			BaseRecord: types.BaseRecord{
				ID:        common.NewID("user"),
				CreatedAt: now,
				UpdatedAt: now,
			},
			Email:              email,
			DisplayName:        displayName,
			Status:             "active",
			PasswordSalt:       salt,
			PasswordHash:       hash,
			PasswordIterations: iterations,
		}
		if err := a.Store.CreateUser(ctx, user); err != nil {
			return types.AuthResponse{}, err
		}
	default:
		return types.AuthResponse{}, err
	}

	return a.issueAuthResponse(ctx, user, "auth.sign_up", []string{"password account created"}, "password", "", "")
}

func (a *Application) SignIn(ctx context.Context, req types.SignInRequest) (types.AuthResponse, error) {
	if a.Config.AuthMode != "dev" {
		return types.AuthResponse{}, ErrForbidden
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		return types.AuthResponse{}, fmt.Errorf("%w: email is required", ErrValidation)
	}
	if req.Password == "" {
		return types.AuthResponse{}, fmt.Errorf("%w: password is required", ErrValidation)
	}

	user, err := a.Store.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return types.AuthResponse{}, fmt.Errorf("%w: invalid email or password", ErrUnauthorized)
		}
		return types.AuthResponse{}, err
	}
	if user.Status != "active" || !auth.VerifyPassword(req.Password, user.PasswordSalt, user.PasswordHash, user.PasswordIterations) {
		return types.AuthResponse{}, fmt.Errorf("%w: invalid email or password", ErrUnauthorized)
	}

	return a.issueAuthResponse(ctx, user, "auth.sign_in", []string{"password login issued"}, "password", "", "")
}

func (a *Application) ensureDemoAdminSeeded(ctx context.Context) error {
	if a.Config.AuthMode != "dev" {
		return nil
	}

	salt, hash, iterations, err := auth.HashPassword(demoAdminPassword)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	var seededOrg types.Organization
	var seededUser types.User
	if err := a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
		org, err := a.Store.GetOrganizationBySlug(txCtx, demoAdminOrgSlug)
		switch {
		case err == nil:
		case errors.Is(err, storage.ErrNotFound):
			org = types.Organization{
				BaseRecord: types.BaseRecord{
					ID:        common.NewID("org"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name: demoAdminOrgName,
				Slug: demoAdminOrgSlug,
				Tier: "growth",
				Mode: "startup",
			}
			if err := a.Store.CreateOrganization(txCtx, org); err != nil {
				return err
			}
		default:
			return err
		}
		seededOrg = org

		user, err := a.Store.GetUserByEmail(txCtx, demoAdminEmail)
		switch {
		case err == nil:
			user.OrganizationID = org.ID
			user.DisplayName = demoAdminDisplayName
			user.Status = "active"
			user.PasswordSalt = salt
			user.PasswordHash = hash
			user.PasswordIterations = iterations
			user.UpdatedAt = now
			if err := a.Store.UpdateUser(txCtx, user); err != nil {
				return err
			}
		case errors.Is(err, storage.ErrNotFound):
			user = types.User{
				BaseRecord: types.BaseRecord{
					ID:        common.NewID("user"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				OrganizationID:     org.ID,
				Email:              demoAdminEmail,
				DisplayName:        demoAdminDisplayName,
				Status:             "active",
				PasswordSalt:       salt,
				PasswordHash:       hash,
				PasswordIterations: iterations,
			}
			if err := a.Store.CreateUser(txCtx, user); err != nil {
				return err
			}
		default:
			return err
		}
		seededUser = user

		if _, err := a.Store.GetOrganizationMembership(txCtx, user.ID, org.ID); err != nil {
			if !errors.Is(err, storage.ErrNotFound) {
				return err
			}
			membership := types.OrganizationMembership{
				BaseRecord: types.BaseRecord{
					ID:        common.NewID("orgm"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				UserID:         user.ID,
				OrganizationID: org.ID,
				Role:           "org_admin",
				Status:         "active",
			}
			if err := a.Store.CreateOrganizationMembership(txCtx, membership); err != nil {
				return err
			}
		}

		if err := a.seedDefaultIntegrations(txCtx, org.ID); err != nil {
			return err
		}
		return a.seedDefaultPolicies(txCtx, org.ID)
	}); err != nil {
		return err
	}
	return a.ensureDemoWorkspaceSeeded(ctx, seededUser, seededOrg)
}

func (a *Application) ensureDemoWorkspaceSeeded(ctx context.Context, user types.User, org types.Organization) error {
	if user.ID == "" || org.ID == "" {
		return nil
	}

	identity := auth.Identity{
		Authenticated:        true,
		ActorID:              user.ID,
		ActorType:            types.ActorTypeUser,
		User:                 user,
		ActiveOrganizationID: org.ID,
		OrganizationMemberships: map[string]types.OrganizationMembership{
			org.ID: {
				UserID:         user.ID,
				OrganizationID: org.ID,
				Role:           "org_admin",
				Status:         "active",
			},
		},
		ProjectMemberships: map[string]types.ProjectMembership{},
		OrganizationRoles: map[string]string{
			org.ID: "org_admin",
		},
		ProjectRoles: map[string]string{},
	}
	seedCtx := auth.WithIdentity(ctx, identity)

	projects, err := a.Store.ListProjects(seedCtx, storage.ProjectQuery{OrganizationID: org.ID})
	if err != nil {
		return err
	}
	project, ok := findProjectBySlug(projects, demoProjectSlug)
	if !ok {
		project, err = a.CreateProject(seedCtx, types.CreateProjectRequest{
			OrganizationID: org.ID,
			Name:           "Control Plane Demo Platform",
			Slug:           demoProjectSlug,
			Description:    "Seeded demo workspace for product walkthroughs and operational testing.",
			AdoptionMode:   "governed",
		})
		if err != nil {
			return err
		}
	}

	teams, err := a.Store.ListTeams(seedCtx, storage.TeamQuery{OrganizationID: org.ID, ProjectID: project.ID})
	if err != nil {
		return err
	}
	team, ok := findTeamBySlug(teams, demoTeamSlug)
	if !ok {
		team, err = a.CreateTeam(seedCtx, types.CreateTeamRequest{
			OrganizationID: org.ID,
			ProjectID:      project.ID,
			Name:           "Control Plane Core",
			Slug:           demoTeamSlug,
			OwnerUserIDs:   []string{user.ID},
		})
		if err != nil {
			return err
		}
	}

	services, err := a.Store.ListServices(seedCtx, storage.ServiceQuery{OrganizationID: org.ID, ProjectID: project.ID})
	if err != nil {
		return err
	}
	primaryService, ok := findServiceBySlug(services, demoPrimaryServiceSlug)
	if !ok {
		primaryService, err = a.CreateService(seedCtx, types.CreateServiceRequest{
			OrganizationID:         org.ID,
			ProjectID:              project.ID,
			TeamID:                 team.ID,
			Name:                   "Checkout API",
			Slug:                   demoPrimaryServiceSlug,
			Description:            "Customer-facing checkout orchestration with strict reliability guardrails.",
			Criticality:            "mission_critical",
			CustomerFacing:         true,
			HasSLO:                 true,
			HasObservability:       true,
			RegulatedZone:          true,
			DependentServicesCount: 3,
		})
		if err != nil {
			return err
		}
	}
	supportService, ok := findServiceBySlug(services, demoSupportServiceSlug)
	if !ok {
		supportService, err = a.CreateService(seedCtx, types.CreateServiceRequest{
			OrganizationID:         org.ID,
			ProjectID:              project.ID,
			TeamID:                 team.ID,
			Name:                   "Ledger Worker",
			Slug:                   demoSupportServiceSlug,
			Description:            "Settlement and ledger fan-out processing for completed orders.",
			Criticality:            "high",
			HasSLO:                 true,
			HasObservability:       true,
			DependentServicesCount: 1,
		})
		if err != nil {
			return err
		}
	}

	environments, err := a.Store.ListEnvironments(seedCtx, storage.EnvironmentQuery{OrganizationID: org.ID, ProjectID: project.ID})
	if err != nil {
		return err
	}
	stagingEnvironment, ok := findEnvironmentBySlug(environments, demoStagingEnvironmentSlug)
	if !ok {
		stagingEnvironment, err = a.CreateEnvironment(seedCtx, types.CreateEnvironmentRequest{
			OrganizationID: org.ID,
			ProjectID:      project.ID,
			Name:           "Staging",
			Slug:           demoStagingEnvironmentSlug,
			Type:           "staging",
			Region:         "us-central1",
			ComplianceZone: "internal",
		})
		if err != nil {
			return err
		}
	}
	prodEnvironment, ok := findEnvironmentBySlug(environments, demoProdEnvironmentSlug)
	if !ok {
		prodEnvironment, err = a.CreateEnvironment(seedCtx, types.CreateEnvironmentRequest{
			OrganizationID: org.ID,
			ProjectID:      project.ID,
			Name:           "Production",
			Slug:           demoProdEnvironmentSlug,
			Type:           "production",
			Region:         "us-central1",
			Production:     true,
			ComplianceZone: "pci",
		})
		if err != nil {
			return err
		}
	}

	changes, err := a.Store.ListChangeSets(seedCtx, storage.ChangeSetQuery{OrganizationID: org.ID, ProjectID: project.ID, ServiceID: primaryService.ID})
	if err != nil {
		return err
	}
	change, ok := findChangeBySummary(changes, demoChangeSummary)
	if !ok {
		change, err = a.CreateChangeSet(seedCtx, types.CreateChangeSetRequest{
			OrganizationID:          org.ID,
			ProjectID:               project.ID,
			ServiceID:               primaryService.ID,
			EnvironmentID:           prodEnvironment.ID,
			Summary:                 demoChangeSummary,
			ChangeTypes:             []string{"code", "config", "dependency"},
			FileCount:               14,
			ResourceCount:           5,
			TouchesInfrastructure:   true,
			TouchesSecrets:          true,
			DependencyChanges:       true,
			HistoricalIncidentCount: 2,
			PoorRollbackHistory:     true,
		})
		if err != nil {
			return err
		}
	}

	plans, err := a.Store.ListRolloutPlans(seedCtx, storage.RolloutPlanQuery{OrganizationID: org.ID, ProjectID: project.ID, ChangeSetID: change.ID})
	if err != nil {
		return err
	}
	var plan types.RolloutPlan
	if len(plans) == 0 {
		planned, err := a.CreateRolloutPlan(seedCtx, types.CreateRolloutPlanRequest{ChangeSetID: change.ID})
		if err != nil {
			return err
		}
		plan = planned.Plan
	} else {
		plan = plans[0]
	}

	rollbackPolicies, err := a.Store.ListRollbackPolicies(seedCtx, storage.RollbackPolicyQuery{OrganizationID: org.ID})
	if err != nil {
		return err
	}
	if len(rollbackPolicies) == 0 {
		if _, err := a.CreateRollbackPolicy(seedCtx, types.CreateRollbackPolicyRequest{
			OrganizationID:            org.ID,
			ProjectID:                 project.ID,
			ServiceID:                 primaryService.ID,
			EnvironmentID:             prodEnvironment.ID,
			Name:                      "Production Checkout Safeguard",
			Description:               "Rollback automatically when latency or error thresholds are breached during production rollout.",
			Priority:                  100,
			MaxErrorRate:              1.5,
			MaxLatencyMs:              350,
			MaxVerificationFailures:   1,
			RollbackOnProviderFailure: boolPtr(true),
			RollbackOnCriticalSignals: boolPtr(true),
		}); err != nil {
			return err
		}
	}

	policies, err := a.Store.ListPolicies(seedCtx, storage.PolicyQuery{OrganizationID: org.ID})
	if err != nil {
		return err
	}
	if len(policies) == 0 {
		if err := a.seedDefaultPolicies(seedCtx, org.ID); err != nil {
			return err
		}
	}

	serviceAccounts, err := a.Store.ListServiceAccounts(seedCtx, storage.ServiceAccountQuery{OrganizationID: org.ID})
	if err != nil {
		return err
	}
	var serviceAccount types.ServiceAccount
	if len(serviceAccounts) == 0 {
		serviceAccount, err = a.CreateServiceAccount(seedCtx, types.CreateServiceAccountRequest{
			OrganizationID: org.ID,
			Name:           "demo-rollout-automation",
			Description:    "Seeded machine actor for rollout automation and graph ingestion demos.",
			Role:           "org_member",
		})
		if err != nil {
			return err
		}
	} else {
		serviceAccount = serviceAccounts[0]
	}
	tokens, err := a.Store.ListAPITokens(seedCtx, storage.APITokenQuery{OrganizationID: org.ID, ServiceAccountID: serviceAccount.ID})
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		if _, err := a.IssueServiceAccountToken(seedCtx, serviceAccount.ID, types.IssueAPITokenRequest{
			Name: "demo-primary",
		}); err != nil {
			return err
		}
	}

	graphRelationships, err := a.Store.ListGraphRelationships(seedCtx, storage.GraphRelationshipQuery{OrganizationID: org.ID})
	if err != nil {
		return err
	}
	if len(graphRelationships) == 0 {
		integrations, err := a.Store.ListIntegrations(seedCtx, storage.IntegrationQuery{OrganizationID: org.ID})
		if err != nil {
			return err
		}
		githubIntegration, ok := findIntegrationByKind(integrations, "github")
		if ok {
			if _, err := a.IngestIntegrationGraph(seedCtx, githubIntegration.ID, types.IntegrationGraphIngestRequest{
				Repositories: []types.IntegrationRepositoryInput{
					{
						ProjectID:     project.ID,
						ServiceID:     primaryService.ID,
						Name:          "checkout-api",
						Provider:      "github",
						URL:           "https://github.com/change-control-plane/checkout-api",
						DefaultBranch: "main",
					},
					{
						ProjectID:     project.ID,
						ServiceID:     supportService.ID,
						Name:          "ledger-worker",
						Provider:      "github",
						URL:           "https://github.com/change-control-plane/ledger-worker",
						DefaultBranch: "main",
					},
				},
				ServiceDependencies: []types.ServiceDependencyInput{
					{
						ServiceID:          primaryService.ID,
						DependsOnServiceID: supportService.ID,
						CriticalDependency: true,
					},
				},
				ServiceEnvironments: []types.ServiceEnvironmentBindingInput{
					{ServiceID: primaryService.ID, EnvironmentID: stagingEnvironment.ID},
					{ServiceID: primaryService.ID, EnvironmentID: prodEnvironment.ID},
					{ServiceID: supportService.ID, EnvironmentID: prodEnvironment.ID},
				},
				ChangeRepositories: []types.ChangeRepositoryBindingInput{
					{
						ChangeSetID:   change.ID,
						RepositoryURL: "https://github.com/change-control-plane/checkout-api",
					},
				},
			}); err != nil {
				return err
			}
		}
	}

	executions, err := a.Store.ListRolloutExecutions(seedCtx, storage.RolloutExecutionQuery{OrganizationID: org.ID, ProjectID: project.ID})
	if err != nil {
		return err
	}
	if len(executions) == 0 {
		execution, err := a.CreateRolloutExecution(seedCtx, types.CreateRolloutExecutionRequest{
			RolloutPlanID:      plan.ID,
			BackendType:        "simulated",
			SignalProviderType: "simulated",
		})
		if err != nil {
			return err
		}
		if _, err := a.AdvanceRolloutExecution(seedCtx, execution.ID, types.AdvanceRolloutExecutionRequest{
			Action: "approve",
			Reason: "seeded demo approval",
		}); err != nil {
			return err
		}
		if _, err := a.AdvanceRolloutExecution(seedCtx, execution.ID, types.AdvanceRolloutExecutionRequest{
			Action: "start",
			Reason: "seeded demo start",
		}); err != nil {
			return err
		}
		if _, err := a.ReconcileRolloutExecution(seedCtx, execution.ID); err != nil {
			return err
		}
		if _, err := a.CreateSignalSnapshot(seedCtx, execution.ID, types.CreateSignalSnapshotRequest{
			Health:  "critical",
			Summary: "Synthetic checkout degradation triggered automated rollback safeguards.",
			Signals: []types.SignalValue{
				{Name: "latency_p95", Category: "latency", Value: 742, Unit: "ms", Status: "critical", Threshold: 350, Comparator: "gt"},
				{Name: "error_rate", Category: "reliability", Value: 4.2, Unit: "%", Status: "critical", Threshold: 1.5, Comparator: "gt"},
				{Name: "checkout_conversion", Category: "business", Value: -7.4, Unit: "%", Status: "warning", Threshold: -2, Comparator: "lt"},
			},
			Explanation: []string{
				"seeded demo signal snapshot models a production regression",
				"critical latency and error signals should trigger rollback posture",
			},
		}); err != nil {
			return err
		}
		if _, err := a.ReconcileRolloutExecution(seedCtx, execution.ID); err != nil {
			return err
		}
	}

	return nil
}

func boolPtr(value bool) *bool {
	return &value
}

func findProjectBySlug(projects []types.Project, slug string) (types.Project, bool) {
	for _, project := range projects {
		if project.Slug == slug {
			return project, true
		}
	}
	return types.Project{}, false
}

func findTeamBySlug(teams []types.Team, slug string) (types.Team, bool) {
	for _, team := range teams {
		if team.Slug == slug {
			return team, true
		}
	}
	return types.Team{}, false
}

func findServiceBySlug(services []types.Service, slug string) (types.Service, bool) {
	for _, service := range services {
		if service.Slug == slug {
			return service, true
		}
	}
	return types.Service{}, false
}

func findEnvironmentBySlug(environments []types.Environment, slug string) (types.Environment, bool) {
	for _, environment := range environments {
		if environment.Slug == slug {
			return environment, true
		}
	}
	return types.Environment{}, false
}

func findChangeBySummary(changes []types.ChangeSet, summary string) (types.ChangeSet, bool) {
	for _, change := range changes {
		if change.Summary == summary {
			return change, true
		}
	}
	return types.ChangeSet{}, false
}

func findIntegrationByKind(integrations []types.Integration, kind string) (types.Integration, bool) {
	for _, integration := range integrations {
		if integration.Kind == kind {
			return integration, true
		}
	}
	return types.Integration{}, false
}

func (a *Application) DevLogin(ctx context.Context, req types.DevLoginRequest) (types.DevLoginResponse, error) {
	if a.Config.AuthMode != "dev" {
		return types.DevLoginResponse{}, ErrForbidden
	}

	email := normalizeEmail(req.Email)
	if email == "" {
		return types.DevLoginResponse{}, fmt.Errorf("%w: email is required", ErrValidation)
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = email
	}

	user, err := a.Store.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
	case errors.Is(err, storage.ErrNotFound):
		if strings.TrimSpace(req.OrganizationSlug) == "" {
			return types.DevLoginResponse{}, fmt.Errorf("%w: organization_slug is required for first login", ErrValidation)
		}
		var createdUser types.User
		err = a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
			org, orgErr := a.Store.GetOrganizationBySlug(txCtx, req.OrganizationSlug)
			orgCreated := false
			if orgErr != nil {
				if !errors.Is(orgErr, storage.ErrNotFound) {
					return orgErr
				}
				if strings.TrimSpace(req.OrganizationName) == "" {
					return fmt.Errorf("%w: organization_name is required when bootstrapping a new organization", ErrValidation)
				}
				orgCreated = true
				now := time.Now().UTC()
				org = types.Organization{
					BaseRecord: types.BaseRecord{
						ID:        common.NewID("org"),
						CreatedAt: now,
						UpdatedAt: now,
					},
					Name: req.OrganizationName,
					Slug: req.OrganizationSlug,
					Tier: "growth",
					Mode: "startup",
				}
				if err := a.Store.CreateOrganization(txCtx, org); err != nil {
					return err
				}
			}

			now := time.Now().UTC()
			createdUser = types.User{
				BaseRecord: types.BaseRecord{
					ID:        common.NewID("user"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				OrganizationID: org.ID,
				Email:          email,
				DisplayName:    displayName,
				Status:         "active",
			}
			if err := a.Store.CreateUser(txCtx, createdUser); err != nil {
				return err
			}

			role := defaultOrgRole(req.Roles, orgCreated)
			membership := types.OrganizationMembership{
				BaseRecord: types.BaseRecord{
					ID:        common.NewID("orgm"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				UserID:         createdUser.ID,
				OrganizationID: org.ID,
				Role:           role,
				Status:         "active",
			}
			if err := a.Store.CreateOrganizationMembership(txCtx, membership); err != nil {
				return err
			}
			if orgCreated {
				if err := a.seedDefaultIntegrations(txCtx, org.ID); err != nil {
					return err
				}
				if err := a.seedDefaultPolicies(txCtx, org.ID); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return types.DevLoginResponse{}, err
		}
		user = createdUser
	default:
		return types.DevLoginResponse{}, err
	}

	authResponse, err := a.issueAuthResponse(ctx, user, "auth.dev_login", []string{"dev login issued"}, "dev_bootstrap", "", "")
	if err != nil {
		return types.DevLoginResponse{}, err
	}
	return types.DevLoginResponse(authResponse), nil
}

func (a *Application) Session(ctx context.Context) types.SessionInfo {
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || !identity.Authenticated {
		return types.SessionInfo{
			Authenticated: false,
			Mode:          a.Config.AuthMode,
			Actor:         "anonymous",
		}
	}
	session, err := a.buildSession(ctx, identity)
	if err != nil {
		return types.SessionInfo{
			Authenticated:  true,
			Mode:           a.Config.AuthMode,
			Actor:          identity.ActorLabel(),
			ActorID:        identity.ActorID,
			ActorType:      string(identity.ActorType),
			AuthMethod:     identity.AuthMethod,
			AuthProviderID: identity.AuthProviderID,
			AuthProvider:   identity.AuthProvider,
			IssuedAt:       formatSessionTime(identity.TokenIssuedAt),
			ExpiresAt:      formatSessionTime(identity.TokenExpiresAt),
			Email:          identity.User.Email,
			DisplayName:    identity.User.DisplayName,
		}
	}
	return session
}

func (a *Application) issueAuthResponse(ctx context.Context, user types.User, action string, details []string, authMethod, authProviderID, authProvider string) (types.AuthResponse, error) {
	token, err := a.Auth.TokenService().SignDetailed(user.ID, types.ActorTypeUser, authMethod, authProviderID, authProvider)
	if err != nil {
		return types.AuthResponse{}, err
	}
	identity, err := a.Auth.LoadIdentity(ctx, "Bearer "+token, "")
	if err != nil {
		return types.AuthResponse{}, err
	}
	session, err := a.buildSession(ctx, identity)
	if err != nil {
		return types.AuthResponse{}, err
	}
	_, _ = a.Audit.Record(ctx, auditActorFromIdentity(identity), action, "session", user.ID, "success", identity.ActiveOrganizationID, "", details)
	return types.AuthResponse{
		Token:   token,
		Session: session,
	}, nil
}

func normalizeEmail(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

func validatePassword(password, confirmation string) error {
	if password == "" {
		return fmt.Errorf("%w: password is required", ErrValidation)
	}
	if confirmation == "" {
		return fmt.Errorf("%w: password_confirmation is required", ErrValidation)
	}
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: password must be at least %d characters", ErrValidation, minPasswordLength)
	}
	if password != confirmation {
		return fmt.Errorf("%w: password confirmation must match", ErrValidation)
	}
	return nil
}

func (a *Application) CreateOrganization(ctx context.Context, req types.CreateOrganizationRequest) (types.Organization, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Organization{}, err
	}
	if !a.Authorizer.CanCreateOrganization(identity) {
		return types.Organization{}, a.forbidden(ctx, identity, "organization.create.denied", "organization", "", "", "", []string{"actor lacks create organization permission"})
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" {
		return types.Organization{}, fmt.Errorf("%w: name and slug are required", ErrValidation)
	}

	now := time.Now().UTC()
	org := types.Organization{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("org"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		Name: req.Name,
		Slug: req.Slug,
		Tier: valueOrDefault(req.Tier, "growth"),
		Mode: valueOrDefault(req.Mode, "startup"),
	}

	err = a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := a.Store.CreateOrganization(txCtx, org); err != nil {
			return err
		}
		membership := types.OrganizationMembership{
			BaseRecord: types.BaseRecord{
				ID:        common.NewID("orgm"),
				CreatedAt: now,
				UpdatedAt: now,
			},
			UserID:         identity.ActorID,
			OrganizationID: org.ID,
			Role:           "org_admin",
			Status:         "active",
		}
		if err := a.Store.CreateOrganizationMembership(txCtx, membership); err != nil {
			return err
		}
		if err := a.seedDefaultIntegrations(txCtx, org.ID); err != nil {
			return err
		}
		return a.seedDefaultPolicies(txCtx, org.ID)
	})
	if err != nil {
		return types.Organization{}, err
	}
	if err := a.record(ctx, identity, "organization.created", "organization", org.ID, org.ID, "", []string{"organization created"}); err != nil {
		return types.Organization{}, err
	}
	return org, nil
}

func (a *Application) ListOrganizations(ctx context.Context) ([]types.Organization, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	organizationIDs := identity.OrganizationIDs()
	if len(organizationIDs) == 0 {
		return []types.Organization{}, nil
	}
	return a.Store.ListOrganizations(ctx, storage.OrganizationQuery{IDs: organizationIDs})
}

func (a *Application) CreateProject(ctx context.Context, req types.CreateProjectRequest) (types.Project, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Project{}, err
	}
	if strings.TrimSpace(req.OrganizationID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" {
		return types.Project{}, fmt.Errorf("%w: organization_id, name, and slug are required", ErrValidation)
	}
	if _, err := a.Store.GetOrganization(ctx, req.OrganizationID); err != nil {
		return types.Project{}, fmt.Errorf("%w: organization %s", storage.ErrNotFound, req.OrganizationID)
	}
	if !a.Authorizer.CanCreateProject(identity, req.OrganizationID) {
		return types.Project{}, a.forbidden(ctx, identity, "project.create.denied", "project", "", req.OrganizationID, "", []string{"actor lacks project creation permission"})
	}

	now := time.Now().UTC()
	project := types.Project{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("proj"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Slug:           req.Slug,
		Description:    req.Description,
		AdoptionMode:   valueOrDefault(req.AdoptionMode, "advisory"),
		Status:         "active",
	}

	err = a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := a.Store.CreateProject(txCtx, project); err != nil {
			return err
		}
		membership := types.ProjectMembership{
			BaseRecord: types.BaseRecord{
				ID:        common.NewID("prjm"),
				CreatedAt: now,
				UpdatedAt: now,
			},
			UserID:         identity.ActorID,
			OrganizationID: req.OrganizationID,
			ProjectID:      project.ID,
			Role:           "project_admin",
			Status:         "active",
		}
		return a.Store.CreateProjectMembership(txCtx, membership)
	})
	if err != nil {
		return types.Project{}, err
	}
	if err := a.record(ctx, identity, "project.created", "project", project.ID, req.OrganizationID, project.ID, []string{"project created"}); err != nil {
		return types.Project{}, err
	}
	return project, nil
}

func (a *Application) ListProjects(ctx context.Context) ([]types.Project, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanViewOrganization(identity, orgID) {
		return nil, ErrForbidden
	}
	return a.Store.ListProjects(ctx, storage.ProjectQuery{OrganizationID: orgID})
}

func (a *Application) CreateTeam(ctx context.Context, req types.CreateTeamRequest) (types.Team, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Team{}, err
	}
	if strings.TrimSpace(req.OrganizationID) == "" || strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" {
		return types.Team{}, fmt.Errorf("%w: organization_id, project_id, name, and slug are required", ErrValidation)
	}
	project, err := a.Store.GetProject(ctx, req.ProjectID)
	if err != nil {
		return types.Team{}, fmt.Errorf("%w: project %s", storage.ErrNotFound, req.ProjectID)
	}
	if project.OrganizationID != req.OrganizationID {
		return types.Team{}, fmt.Errorf("%w: project %s does not belong to organization %s", ErrValidation, req.ProjectID, req.OrganizationID)
	}
	if !a.Authorizer.CanCreateTeam(identity, req.OrganizationID, req.ProjectID) {
		return types.Team{}, a.forbidden(ctx, identity, "team.create.denied", "team", "", req.OrganizationID, req.ProjectID, []string{"actor lacks team creation permission"})
	}
	now := time.Now().UTC()
	team := types.Team{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("team"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID: req.OrganizationID,
		ProjectID:      req.ProjectID,
		Name:           req.Name,
		Slug:           req.Slug,
		OwnerUserIDs:   req.OwnerUserIDs,
		Status:         "active",
	}
	if err := a.Store.CreateTeam(ctx, team); err != nil {
		return types.Team{}, err
	}
	if err := a.record(ctx, identity, "team.created", "team", team.ID, req.OrganizationID, req.ProjectID, []string{"team created"}); err != nil {
		return types.Team{}, err
	}
	return team, nil
}

func (a *Application) ListTeams(ctx context.Context) ([]types.Team, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListTeams(ctx, storage.TeamQuery{OrganizationID: orgID})
}

func (a *Application) CreateService(ctx context.Context, req types.CreateServiceRequest) (types.Service, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Service{}, err
	}
	if strings.TrimSpace(req.OrganizationID) == "" || strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.TeamID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" {
		return types.Service{}, fmt.Errorf("%w: organization_id, project_id, team_id, name, and slug are required", ErrValidation)
	}
	project, err := a.Store.GetProject(ctx, req.ProjectID)
	if err != nil {
		return types.Service{}, fmt.Errorf("%w: project %s", storage.ErrNotFound, req.ProjectID)
	}
	if project.OrganizationID != req.OrganizationID {
		return types.Service{}, fmt.Errorf("%w: project %s does not belong to organization %s", ErrValidation, req.ProjectID, req.OrganizationID)
	}
	team, err := a.Store.GetTeam(ctx, req.TeamID)
	if err != nil {
		return types.Service{}, fmt.Errorf("%w: team %s", storage.ErrNotFound, req.TeamID)
	}
	if team.ProjectID != req.ProjectID {
		return types.Service{}, fmt.Errorf("%w: team %s does not belong to project %s", ErrValidation, req.TeamID, req.ProjectID)
	}
	if !a.Authorizer.CanManageProject(identity, req.OrganizationID, req.ProjectID) && !contains(team.OwnerUserIDs, identity.ActorID) {
		return types.Service{}, a.forbidden(ctx, identity, "service.create.denied", "service", "", req.OrganizationID, req.ProjectID, []string{"actor lacks service registration permission"})
	}
	now := time.Now().UTC()
	service := types.Service{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("svc"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:         req.OrganizationID,
		ProjectID:              req.ProjectID,
		TeamID:                 req.TeamID,
		Name:                   req.Name,
		Slug:                   req.Slug,
		Description:            req.Description,
		Criticality:            valueOrDefault(req.Criticality, "medium"),
		Tier:                   valueOrDefault(req.Tier, "service"),
		CustomerFacing:         req.CustomerFacing,
		HasSLO:                 req.HasSLO,
		HasObservability:       req.HasObservability,
		RegulatedZone:          req.RegulatedZone,
		DependentServicesCount: req.DependentServicesCount,
		Status:                 "active",
	}
	if err := a.Store.CreateService(ctx, service); err != nil {
		return types.Service{}, err
	}
	if err := a.record(ctx, identity, "service.registered", "service", service.ID, req.OrganizationID, req.ProjectID, []string{"service registered"}); err != nil {
		return types.Service{}, err
	}
	return service, nil
}

func (a *Application) ListServices(ctx context.Context) ([]types.Service, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListServices(ctx, storage.ServiceQuery{OrganizationID: orgID})
}

func (a *Application) CreateEnvironment(ctx context.Context, req types.CreateEnvironmentRequest) (types.Environment, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Environment{}, err
	}
	if strings.TrimSpace(req.OrganizationID) == "" || strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" || strings.TrimSpace(req.Type) == "" {
		return types.Environment{}, fmt.Errorf("%w: organization_id, project_id, name, slug, and type are required", ErrValidation)
	}
	project, err := a.Store.GetProject(ctx, req.ProjectID)
	if err != nil {
		return types.Environment{}, fmt.Errorf("%w: project %s", storage.ErrNotFound, req.ProjectID)
	}
	if project.OrganizationID != req.OrganizationID {
		return types.Environment{}, fmt.Errorf("%w: project %s does not belong to organization %s", ErrValidation, req.ProjectID, req.OrganizationID)
	}
	if !a.Authorizer.CanCreateEnvironment(identity, req.OrganizationID, req.ProjectID) {
		return types.Environment{}, a.forbidden(ctx, identity, "environment.create.denied", "environment", "", req.OrganizationID, req.ProjectID, []string{"actor lacks environment mutation permission"})
	}
	now := time.Now().UTC()
	environment := types.Environment{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("env"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID: req.OrganizationID,
		ProjectID:      req.ProjectID,
		Name:           req.Name,
		Slug:           req.Slug,
		Type:           req.Type,
		Region:         valueOrDefault(req.Region, "us-central1"),
		Production:     req.Production,
		ComplianceZone: req.ComplianceZone,
		Status:         "active",
	}
	if err := a.Store.CreateEnvironment(ctx, environment); err != nil {
		return types.Environment{}, err
	}
	if err := a.record(ctx, identity, "environment.created", "environment", environment.ID, req.OrganizationID, req.ProjectID, []string{"environment created"}); err != nil {
		return types.Environment{}, err
	}
	return environment, nil
}

func (a *Application) ListEnvironments(ctx context.Context) ([]types.Environment, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListEnvironments(ctx, storage.EnvironmentQuery{OrganizationID: orgID})
}

func (a *Application) CreateChangeSet(ctx context.Context, req types.CreateChangeSetRequest) (types.ChangeSet, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ChangeSet{}, err
	}
	if strings.TrimSpace(req.OrganizationID) == "" || strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.ServiceID) == "" || strings.TrimSpace(req.EnvironmentID) == "" || strings.TrimSpace(req.Summary) == "" {
		return types.ChangeSet{}, fmt.Errorf("%w: organization_id, project_id, service_id, environment_id, and summary are required", ErrValidation)
	}
	service, err := a.Store.GetService(ctx, req.ServiceID)
	if err != nil {
		return types.ChangeSet{}, fmt.Errorf("%w: service %s", storage.ErrNotFound, req.ServiceID)
	}
	if service.OrganizationID != req.OrganizationID || service.ProjectID != req.ProjectID {
		return types.ChangeSet{}, fmt.Errorf("%w: service scope mismatch", ErrValidation)
	}
	environment, err := a.Store.GetEnvironment(ctx, req.EnvironmentID)
	if err != nil {
		return types.ChangeSet{}, fmt.Errorf("%w: environment %s", storage.ErrNotFound, req.EnvironmentID)
	}
	if environment.OrganizationID != req.OrganizationID || environment.ProjectID != req.ProjectID {
		return types.ChangeSet{}, fmt.Errorf("%w: environment scope mismatch", ErrValidation)
	}
	team, err := a.Store.GetTeam(ctx, service.TeamID)
	if err != nil {
		return types.ChangeSet{}, fmt.Errorf("%w: team %s", storage.ErrNotFound, service.TeamID)
	}
	if !a.Authorizer.CanIngestChange(identity, req.OrganizationID, req.ProjectID, team) {
		return types.ChangeSet{}, a.forbidden(ctx, identity, "change.ingest.denied", "change_set", "", req.OrganizationID, req.ProjectID, []string{"actor lacks change ingestion permission"})
	}

	now := time.Now().UTC()
	change := types.ChangeSet{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("chg"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:          req.OrganizationID,
		ProjectID:               req.ProjectID,
		ServiceID:               req.ServiceID,
		EnvironmentID:           req.EnvironmentID,
		Summary:                 req.Summary,
		ChangeTypes:             req.ChangeTypes,
		FileCount:               req.FileCount,
		ResourceCount:           req.ResourceCount,
		TouchesInfrastructure:   req.TouchesInfrastructure,
		TouchesIAM:              req.TouchesIAM,
		TouchesSecrets:          req.TouchesSecrets,
		TouchesSchema:           req.TouchesSchema,
		DependencyChanges:       req.DependencyChanges,
		HistoricalIncidentCount: req.HistoricalIncidentCount,
		PoorRollbackHistory:     req.PoorRollbackHistory,
		Status:                  "ingested",
	}
	if err := a.Store.CreateChangeSet(ctx, change); err != nil {
		return types.ChangeSet{}, err
	}
	if err := a.record(ctx, identity, "change.ingested", "change_set", change.ID, req.OrganizationID, req.ProjectID, []string{change.Summary}); err != nil {
		return types.ChangeSet{}, err
	}
	return change, nil
}

func (a *Application) ListChangeSets(ctx context.Context) ([]types.ChangeSet, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListChangeSets(ctx, storage.ChangeSetQuery{OrganizationID: orgID})
}

func (a *Application) AssessRisk(ctx context.Context, req types.CreateRiskAssessmentRequest) (types.RiskAssessmentResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RiskAssessmentResult{}, err
	}
	change, err := a.Store.GetChangeSet(ctx, req.ChangeSetID)
	if err != nil {
		return types.RiskAssessmentResult{}, fmt.Errorf("%w: change_set %s", storage.ErrNotFound, req.ChangeSetID)
	}
	service, err := a.Store.GetService(ctx, change.ServiceID)
	if err != nil {
		return types.RiskAssessmentResult{}, fmt.Errorf("%w: service %s", storage.ErrNotFound, change.ServiceID)
	}
	environment, err := a.Store.GetEnvironment(ctx, change.EnvironmentID)
	if err != nil {
		return types.RiskAssessmentResult{}, fmt.Errorf("%w: environment %s", storage.ErrNotFound, change.EnvironmentID)
	}
	if !a.Authorizer.CanAssessRisk(identity, change.OrganizationID, change.ProjectID) {
		return types.RiskAssessmentResult{}, a.forbidden(ctx, identity, "risk.assess.denied", "risk_assessment", "", change.OrganizationID, change.ProjectID, []string{"actor lacks risk assessment permission"})
	}

	assessment := a.RiskEngine.Assess(change, service, environment)
	a.applyRiskIntelligence(ctx, change, service, environment, &assessment)
	if err := a.Store.CreateRiskAssessment(ctx, assessment); err != nil {
		return types.RiskAssessmentResult{}, err
	}
	decisions, err := a.evaluateAndPersistPolicies(ctx, policylib.AppliesToRiskAssessment, change, service, environment, assessment, policyEvaluationReference{
		riskAssessmentID: assessment.ID,
		metadata:         policyDecisionMetadata("evaluated", false),
	})
	if err != nil {
		return types.RiskAssessmentResult{}, err
	}
	if err := a.record(ctx, identity, "risk.assessed", "risk_assessment", assessment.ID, change.OrganizationID, change.ProjectID, []string{fmt.Sprintf("risk=%d", assessment.Score), string(assessment.Level)}); err != nil {
		return types.RiskAssessmentResult{}, err
	}

	return types.RiskAssessmentResult{
		Assessment:      assessment,
		PolicyDecisions: decisions,
	}, nil
}

func (a *Application) ListRiskAssessments(ctx context.Context) ([]types.RiskAssessment, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListRiskAssessments(ctx, storage.RiskAssessmentQuery{OrganizationID: orgID})
}

func (a *Application) CreateRolloutPlan(ctx context.Context, req types.CreateRolloutPlanRequest) (types.RolloutPlanResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RolloutPlanResult{}, err
	}
	change, err := a.Store.GetChangeSet(ctx, req.ChangeSetID)
	if err != nil {
		return types.RolloutPlanResult{}, fmt.Errorf("%w: change_set %s", storage.ErrNotFound, req.ChangeSetID)
	}
	if !a.Authorizer.CanPlanRollout(identity, change.OrganizationID, change.ProjectID) {
		return types.RolloutPlanResult{}, a.forbidden(ctx, identity, "rollout.plan.denied", "rollout_plan", "", change.OrganizationID, change.ProjectID, []string{"actor lacks rollout planning permission"})
	}
	assessmentResult, err := a.AssessRisk(ctx, types.CreateRiskAssessmentRequest{ChangeSetID: req.ChangeSetID})
	if err != nil {
		return types.RolloutPlanResult{}, err
	}
	service, _ := a.Store.GetService(ctx, change.ServiceID)
	environment, _ := a.Store.GetEnvironment(ctx, change.EnvironmentID)
	rolloutDecisions, err := a.evaluatePolicies(ctx, policylib.AppliesToRolloutPlan, change, service, environment, assessmentResult.Assessment, policyEvaluationReference{
		riskAssessmentID: assessmentResult.Assessment.ID,
		metadata:         policyDecisionMetadata("evaluated", false),
	})
	if err != nil {
		return types.RolloutPlanResult{}, err
	}
	allDecisions := append(append([]types.PolicyDecision{}, assessmentResult.PolicyDecisions...), rolloutDecisions...)
	if isPolicyBlocked(allDecisions) {
		for index := range rolloutDecisions {
			if rolloutDecisions[index].Metadata == nil {
				rolloutDecisions[index].Metadata = types.Metadata{}
			}
			for key, value := range policyDecisionMetadata("blocked", true) {
				rolloutDecisions[index].Metadata[key] = value
			}
		}
		if err := a.persistPolicyDecisions(ctx, rolloutDecisions); err != nil {
			return types.RolloutPlanResult{}, err
		}
		if err := a.recordPolicyBlockedOutcome(ctx, identity, change, allDecisions); err != nil {
			return types.RolloutPlanResult{}, err
		}
		return types.RolloutPlanResult{}, wrapPolicyBlockError(allDecisions)
	}
	plan := a.Planner.Plan(change, service, environment, assessmentResult.Assessment, allDecisions)
	if isPolicyReviewRequired(allDecisions) && plan.ApprovalLevel == "self-serve" {
		plan.ApprovalLevel = "policy-review"
	}
	plan.Explanation = append(plan.Explanation, decisionSummaries(allDecisions)...)
	a.applyRolloutSimulation(ctx, change, service, environment, assessmentResult.Assessment, &plan)
	if err := a.Store.CreateRolloutPlan(ctx, plan); err != nil {
		return types.RolloutPlanResult{}, err
	}
	for index := range rolloutDecisions {
		rolloutDecisions[index].RolloutPlanID = plan.ID
	}
	if err := a.persistPolicyDecisions(ctx, rolloutDecisions); err != nil {
		return types.RolloutPlanResult{}, err
	}
	if err := a.record(ctx, identity, "rollout.planned", "rollout_plan", plan.ID, change.OrganizationID, change.ProjectID, append([]string{plan.Strategy}, decisionSummaries(allDecisions)...), withStatusCategory("governance"), withStatusSummary("rollout plan created with policy evaluation")); err != nil {
		return types.RolloutPlanResult{}, err
	}

	return types.RolloutPlanResult{
		Assessment:      assessmentResult.Assessment,
		Plan:            plan,
		PolicyDecisions: allDecisions,
	}, nil
}

func (a *Application) ListRolloutPlans(ctx context.Context) ([]types.RolloutPlan, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListRolloutPlans(ctx, storage.RolloutPlanQuery{OrganizationID: orgID})
}

func (a *Application) PoliciesList(ctx context.Context) ([]types.Policy, error) {
	return a.ListPolicies(ctx)
}

func (a *Application) IntegrationsList(ctx context.Context) ([]types.Integration, error) {
	return a.ListIntegrationsWithQuery(ctx, storage.IntegrationQuery{})
}

func (a *Application) ListIntegrationsWithQuery(ctx context.Context, query storage.IntegrationQuery) ([]types.Integration, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	query.OrganizationID = orgID
	integrations, err := a.Store.ListIntegrations(ctx, query)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	for idx := range integrations {
		integrations[idx] = hydrateIntegrationRuntimeState(integrations[idx], now)
	}
	return integrations, nil
}

func (a *Application) AuditEvents(ctx context.Context) ([]types.AuditEvent, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanViewAudit(identity, orgID) {
		return nil, ErrForbidden
	}
	return a.Store.ListAuditEvents(ctx, storage.AuditEventQuery{OrganizationID: orgID})
}

func (a *Application) Catalog(ctx context.Context) (types.CatalogSummary, error) {
	services, err := a.ListServices(ctx)
	if err != nil {
		return types.CatalogSummary{}, err
	}
	environments, err := a.ListEnvironments(ctx)
	if err != nil {
		return types.CatalogSummary{}, err
	}
	return types.CatalogSummary{
		Services:     services,
		Environments: environments,
	}, nil
}

func (a *Application) Metrics(ctx context.Context) (types.BasicMetrics, error) {
	organizations, err := a.ListOrganizations(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	projects, err := a.ListProjects(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	teams, err := a.ListTeams(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	services, err := a.ListServices(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	environments, err := a.ListEnvironments(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	changes, err := a.ListChangeSets(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	risks, err := a.ListRiskAssessments(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	rollouts, err := a.ListRolloutPlans(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	audits, err := a.AuditEvents(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	integrations, err := a.IntegrationsList(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	policies, err := a.PoliciesList(ctx)
	if err != nil {
		return types.BasicMetrics{}, err
	}
	return types.BasicMetrics{
		Organizations:   len(organizations),
		Projects:        len(projects),
		Teams:           len(teams),
		Services:        len(services),
		Environments:    len(environments),
		Changes:         len(changes),
		RiskAssessments: len(risks),
		RolloutPlans:    len(rollouts),
		AuditEvents:     len(audits),
		Policies:        len(policies),
		Integrations:    len(integrations),
	}, nil
}

type IncidentQuery struct {
	ProjectID     string
	ServiceID     string
	EnvironmentID string
	ChangeSetID   string
	Severity      string
	Status        string
	Search        string
	Limit         int
}

func (a *Application) Incidents(ctx context.Context, query IncidentQuery) ([]types.Incident, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}

	executions, err := a.Store.ListRolloutExecutions(ctx, storage.RolloutExecutionQuery{OrganizationID: orgID})
	if err != nil {
		return nil, err
	}

	incidents := make([]types.Incident, 0, len(executions))
	for _, execution := range executions {
		incident, ok := a.incidentFromRolloutExecution(ctx, execution)
		if !ok {
			continue
		}
		if incident.OrganizationID == "" {
			incident.OrganizationID = orgID
		}
		if !incidentMatchesQuery(incident, query) {
			continue
		}
		incidents = append(incidents, incident)
	}

	sort.Slice(incidents, func(i, j int) bool {
		return incidents[i].UpdatedAt.After(incidents[j].UpdatedAt)
	})
	if query.Limit > 0 && len(incidents) > query.Limit {
		incidents = incidents[:query.Limit]
	}
	return incidents, nil
}

func (a *Application) GetIncidentDetail(ctx context.Context, id string) (types.IncidentDetail, error) {
	executionID := incidentExecutionID(id)
	if executionID == "" {
		return types.IncidentDetail{}, fmt.Errorf("%w: incident id is required", ErrValidation)
	}

	detail, err := a.GetRolloutExecutionDetail(ctx, executionID)
	if err != nil {
		return types.IncidentDetail{}, err
	}

	incident, ok := a.incidentFromRolloutExecution(ctx, detail.Execution)
	if !ok {
		return types.IncidentDetail{}, storage.ErrNotFound
	}

	var releaseAnalysis *types.ReleaseAnalysis
	if strings.TrimSpace(detail.Execution.ReleaseID) != "" {
		analysis, err := a.GetReleaseAnalysis(ctx, detail.Execution.ReleaseID)
		if err == nil {
			releaseAnalysis = &analysis
		}
	}
	assistantSummary := buildIncidentAssistantSummary(detail, releaseAnalysis)

	return types.IncidentDetail{
		Incident:           incident,
		RolloutExecutionID: detail.Execution.ID,
		StatusTimeline:     detail.StatusTimeline,
		AssistantSummary:   &assistantSummary,
	}, nil
}

func (a *Application) incidentFromRolloutExecution(ctx context.Context, execution types.RolloutExecution) (types.Incident, bool) {
	severity, status := derivedIncidentState(execution)
	if severity == "" {
		return types.Incident{}, false
	}

	serviceName := execution.ServiceID
	if service, err := a.Store.GetService(ctx, execution.ServiceID); err == nil && service.Name != "" {
		serviceName = service.Name
	}
	environmentName := execution.EnvironmentID
	if environment, err := a.Store.GetEnvironment(ctx, execution.EnvironmentID); err == nil && environment.Name != "" {
		environmentName = environment.Name
	}

	impactedPaths := []string{serviceName, environmentName}
	if execution.CurrentStep != "" {
		impactedPaths = append(impactedPaths, execution.CurrentStep)
	}
	if execution.LastDecisionReason != "" {
		impactedPaths = append(impactedPaths, execution.LastDecisionReason)
	}

	title := fmt.Sprintf("%s rollout %s in %s", serviceName, humanizeIncidentStatus(execution.Status), environmentName)
	return types.Incident{
		BaseRecord: types.BaseRecord{
			ID:        incidentIDForExecution(execution.ID),
			CreatedAt: execution.UpdatedAt,
			UpdatedAt: execution.UpdatedAt,
		},
		OrganizationID: execution.OrganizationID,
		ProjectID:      execution.ProjectID,
		ServiceID:      execution.ServiceID,
		EnvironmentID:  execution.EnvironmentID,
		Title:          title,
		Severity:       severity,
		Status:         status,
		RelatedChange:  execution.ChangeSetID,
		ImpactedPaths:  impactedPaths,
	}, true
}

func incidentIDForExecution(executionID string) string {
	return "incident_" + strings.TrimSpace(executionID)
}

func incidentExecutionID(id string) string {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return ""
	}
	return strings.TrimPrefix(trimmed, "incident_")
}

func derivedIncidentState(execution types.RolloutExecution) (severity string, status string) {
	switch execution.Status {
	case "rolled_back":
		return "critical", "mitigated"
	case "failed":
		return "critical", "open"
	case "paused":
		return "high", "monitoring"
	case "awaiting_approval":
		if execution.LastVerificationResult == "fail" {
			return "medium", "investigating"
		}
	}
	return "", ""
}

func humanizeIncidentStatus(status string) string {
	switch status {
	case "rolled_back":
		return "rolled back"
	case "awaiting_approval":
		return "awaiting approval"
	default:
		return strings.ReplaceAll(status, "_", " ")
	}
}

func incidentMatchesQuery(incident types.Incident, query IncidentQuery) bool {
	if query.ProjectID != "" && incident.ProjectID != query.ProjectID {
		return false
	}
	if query.ServiceID != "" && incident.ServiceID != query.ServiceID {
		return false
	}
	if query.EnvironmentID != "" && incident.EnvironmentID != query.EnvironmentID {
		return false
	}
	if query.ChangeSetID != "" && incident.RelatedChange != query.ChangeSetID {
		return false
	}
	if query.Severity != "" && !strings.EqualFold(incident.Severity, query.Severity) {
		return false
	}
	if query.Status != "" && !strings.EqualFold(incident.Status, query.Status) {
		return false
	}
	if query.Search != "" {
		search := strings.ToLower(strings.TrimSpace(query.Search))
		haystack := strings.ToLower(strings.Join(append([]string{
			incident.ID,
			incident.Title,
			incident.RelatedChange,
			incident.Severity,
			incident.Status,
		}, incident.ImpactedPaths...), " "))
		if !strings.Contains(haystack, search) {
			return false
		}
	}
	return true
}

func (a *Application) requireIdentity(ctx context.Context) (auth.Identity, error) {
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || !identity.Authenticated {
		return auth.Identity{}, ErrUnauthorized
	}
	return identity, nil
}

func (a *Application) requireActiveOrganization(identity auth.Identity) (string, error) {
	if identity.ActiveOrganizationID == "" {
		return "", fmt.Errorf("%w: active organization scope is required", ErrValidation)
	}
	if !identity.HasOrganizationAccess(identity.ActiveOrganizationID) {
		return "", ErrForbidden
	}
	return identity.ActiveOrganizationID, nil
}

func (a *Application) buildSession(ctx context.Context, identity auth.Identity) (types.SessionInfo, error) {
	organizationIDs := identity.OrganizationIDs()
	organizations := []types.Organization{}
	var err error
	if len(organizationIDs) > 0 {
		organizations, err = a.Store.ListOrganizations(ctx, storage.OrganizationQuery{IDs: organizationIDs})
		if err != nil {
			return types.SessionInfo{}, err
		}
	}
	orgByID := make(map[string]types.Organization, len(organizations))
	for _, organization := range organizations {
		orgByID[organization.ID] = organization
	}

	projectMemberships, err := a.Store.ListProjectMembershipsByUser(ctx, identity.ActorID)
	if err != nil {
		return types.SessionInfo{}, err
	}
	projectIDs := make([]string, 0, len(projectMemberships))
	for _, membership := range projectMemberships {
		projectIDs = append(projectIDs, membership.ProjectID)
	}
	projects := []types.Project{}
	if len(projectIDs) > 0 {
		projects, err = a.Store.ListProjects(ctx, storage.ProjectQuery{IDs: projectIDs})
		if err != nil {
			return types.SessionInfo{}, err
		}
	}
	projectByID := make(map[string]types.Project, len(projects))
	for _, project := range projects {
		projectByID[project.ID] = project
	}

	session := types.SessionInfo{
		Authenticated:        true,
		Mode:                 a.Config.AuthMode,
		Actor:                identity.ActorLabel(),
		ActorID:              identity.ActorID,
		ActorType:            string(identity.ActorType),
		AuthMethod:           identity.AuthMethod,
		AuthProviderID:       identity.AuthProviderID,
		AuthProvider:         identity.AuthProvider,
		IssuedAt:             formatSessionTime(identity.TokenIssuedAt),
		ExpiresAt:            formatSessionTime(identity.TokenExpiresAt),
		Email:                identity.User.Email,
		DisplayName:          identity.User.DisplayName,
		ActiveOrganizationID: identity.ActiveOrganizationID,
		Organizations:        make([]types.SessionOrganization, 0, len(identity.OrganizationMemberships)),
		ProjectMemberships:   make([]types.SessionProjectScope, 0, len(projectMemberships)),
	}

	for _, organizationID := range organizationIDs {
		organization := orgByID[organizationID]
		session.Organizations = append(session.Organizations, types.SessionOrganization{
			OrganizationID: organization.ID,
			Organization:   organization.Name,
			Role:           identity.OrganizationRole(organization.ID),
		})
	}
	for _, membership := range projectMemberships {
		project := projectByID[membership.ProjectID]
		session.ProjectMemberships = append(session.ProjectMemberships, types.SessionProjectScope{
			OrganizationID: membership.OrganizationID,
			ProjectID:      membership.ProjectID,
			Project:        project.Name,
			Role:           membership.Role,
		})
	}

	return session, nil
}

func formatSessionTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (a *Application) record(ctx context.Context, identity auth.Identity, eventType, resourceType, resourceID, organizationID, projectID string, details []string, options ...statusRecordOption) error {
	event, err := a.Audit.Record(ctx, auditActorFromIdentity(identity), eventType, resourceType, resourceID, "success", organizationID, projectID, details)
	if err != nil {
		return err
	}
	if err := a.emitStatusEventFromAudit(ctx, identity, event, details, options...); err != nil {
		return err
	}
	return a.Events.Publish(ctx, types.DomainEvent{
		ID:             event.ID,
		Type:           eventType,
		OrganizationID: organizationID,
		ProjectID:      projectID,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		OccurredAt:     event.CreatedAt,
		Payload:        types.Metadata{"details": details},
	})
}

func (a *Application) forbidden(ctx context.Context, identity auth.Identity, eventType, resourceType, resourceID, organizationID, projectID string, details []string) error {
	_, _ = a.Audit.Record(ctx, auditActorFromIdentity(identity), eventType, resourceType, resourceID, "denied", organizationID, projectID, details)
	return ErrForbidden
}

func (a *Application) seedDefaultIntegrations(ctx context.Context, organizationID string) error {
	for _, descriptor := range a.Integrations.List() {
		descriptor.OrganizationID = organizationID
		descriptor.InstanceKey = "default"
		descriptor.ScopeType = "organization"
		if descriptor.ScopeName == "" {
			descriptor.ScopeName = descriptor.Name
		}
		descriptor.ID = fmt.Sprintf("integration_%s_%s", organizationID, descriptor.Kind)
		if _, err := a.Store.GetIntegration(ctx, descriptor.ID); err == nil {
			continue
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
		if err := a.Store.CreateIntegration(ctx, descriptor); err != nil {
			return err
		}
	}
	return nil
}

func auditActorFromIdentity(identity auth.Identity) audit.Actor {
	return audit.Actor{
		ID:    identity.ActorID,
		Type:  string(identity.ActorType),
		Label: identity.ActorLabel(),
	}
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func defaultOrgRole(roles []string, orgCreated bool) string {
	if len(roles) > 0 && strings.TrimSpace(roles[0]) != "" {
		return roles[0]
	}
	if orgCreated {
		return "org_admin"
	}
	return "org_member"
}
