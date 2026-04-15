package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/audit"
	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/events"
	"github.com/change-control-plane/change-control-plane/internal/integrations"
	"github.com/change-control-plane/change-control-plane/internal/policies"
	"github.com/change-control-plane/change-control-plane/internal/risk"
	"github.com/change-control-plane/change-control-plane/internal/rollouts"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Application struct {
	Config       common.Config
	Store        storage.Store
	Events       events.Bus
	Audit        *audit.Service
	RiskEngine   *risk.Engine
	Planner      *rollouts.Planner
	Policies     policies.Evaluator
	Integrations *integrations.Registry
	Auth         *auth.Service
	Authorizer   *auth.Authorizer
}

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
	bus := events.NewInMemoryBus()
	tokenService := auth.NewTokenService(cfg.AuthTokenSecret, time.Duration(cfg.AuthTokenTTL)*time.Minute)
	return &Application{
		Config:       cfg,
		Store:        store,
		Events:       bus,
		Audit:        audit.NewService(store),
		RiskEngine:   risk.NewEngine(),
		Planner:      rollouts.NewPlanner(),
		Policies:     policies.NewDefaultEvaluator(),
		Integrations: integrations.NewRegistry(),
		Auth:         auth.NewService(store, tokenService),
		Authorizer:   auth.NewAuthorizer(),
	}
}

func (a *Application) Close() error {
	return a.Store.Close()
}

func (a *Application) DevLogin(ctx context.Context, req types.DevLoginRequest) (types.DevLoginResponse, error) {
	if a.Config.AuthMode != "dev" {
		return types.DevLoginResponse{}, ErrForbidden
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
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

	token, err := a.Auth.TokenService().Sign(user.ID, types.ActorTypeUser)
	if err != nil {
		return types.DevLoginResponse{}, err
	}
	identity, err := a.Auth.LoadIdentity(ctx, "Bearer "+token, user.OrganizationID)
	if err != nil {
		return types.DevLoginResponse{}, err
	}
	session, err := a.buildSession(ctx, identity)
	if err != nil {
		return types.DevLoginResponse{}, err
	}
	_, _ = a.Audit.Record(ctx, auditActorFromIdentity(identity), "auth.dev_login", "session", user.ID, "success", identity.ActiveOrganizationID, "", []string{"dev login issued"})

	return types.DevLoginResponse{
		Token:   token,
		Session: session,
	}, nil
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
			Authenticated: true,
			Mode:          a.Config.AuthMode,
			Actor:         identity.ActorLabel(),
			ActorID:       identity.ActorID,
			ActorType:     string(identity.ActorType),
			Email:         identity.User.Email,
			DisplayName:   identity.User.DisplayName,
		}
	}
	return session
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
		return a.seedDefaultIntegrations(txCtx, org.ID)
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
	return a.Store.ListOrganizations(ctx, storage.OrganizationQuery{IDs: identity.OrganizationIDs()})
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
	if err := a.Store.CreateRiskAssessment(ctx, assessment); err != nil {
		return types.RiskAssessmentResult{}, err
	}
	decisions := a.Policies.Evaluate(change, service, environment, assessment)
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
	plan := a.Planner.Plan(change, service, environment, assessmentResult.Assessment, assessmentResult.PolicyDecisions)
	if err := a.Store.CreateRolloutPlan(ctx, plan); err != nil {
		return types.RolloutPlanResult{}, err
	}
	if err := a.record(ctx, identity, "rollout.planned", "rollout_plan", plan.ID, change.OrganizationID, change.ProjectID, []string{plan.Strategy}); err != nil {
		return types.RolloutPlanResult{}, err
	}

	return types.RolloutPlanResult{
		Assessment:      assessmentResult.Assessment,
		Plan:            plan,
		PolicyDecisions: assessmentResult.PolicyDecisions,
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
	if _, err := a.requireIdentity(ctx); err != nil {
		return nil, err
	}
	return a.Policies.Policies(), nil
}

func (a *Application) IntegrationsList(ctx context.Context) ([]types.Integration, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListIntegrations(ctx, storage.IntegrationQuery{OrganizationID: orgID})
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

func (a *Application) Incidents(ctx context.Context) ([]types.Incident, error) {
	if _, err := a.requireIdentity(ctx); err != nil {
		return nil, err
	}
	return []types.Incident{}, nil
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
	organizations, err := a.Store.ListOrganizations(ctx, storage.OrganizationQuery{IDs: identity.OrganizationIDs()})
	if err != nil {
		return types.SessionInfo{}, err
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
	projects, err := a.Store.ListProjects(ctx, storage.ProjectQuery{IDs: projectIDs})
	if err != nil {
		return types.SessionInfo{}, err
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
		Email:                identity.User.Email,
		DisplayName:          identity.User.DisplayName,
		ActiveOrganizationID: identity.ActiveOrganizationID,
		Organizations:        make([]types.SessionOrganization, 0, len(identity.OrganizationMemberships)),
		ProjectMemberships:   make([]types.SessionProjectScope, 0, len(projectMemberships)),
	}

	for _, organizationID := range identity.OrganizationIDs() {
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

func (a *Application) record(ctx context.Context, identity auth.Identity, eventType, resourceType, resourceID, organizationID, projectID string, details []string) error {
	event, err := a.Audit.Record(ctx, auditActorFromIdentity(identity), eventType, resourceType, resourceID, "success", organizationID, projectID, details)
	if err != nil {
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
		descriptor.ID = fmt.Sprintf("integration_%s_%s", organizationID, descriptor.Kind)
		if err := a.Store.UpsertIntegration(ctx, descriptor); err != nil {
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
