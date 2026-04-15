package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/rollouts"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) GetOrganization(ctx context.Context, id string) (types.Organization, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Organization{}, err
	}
	if !a.Authorizer.CanViewOrganization(identity, id) {
		return types.Organization{}, ErrForbidden
	}
	return a.Store.GetOrganization(ctx, id)
}

func (a *Application) UpdateOrganization(ctx context.Context, id string, req types.UpdateOrganizationRequest) (types.Organization, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Organization{}, err
	}
	organization, err := a.Store.GetOrganization(ctx, id)
	if err != nil {
		return types.Organization{}, err
	}
	if !a.Authorizer.CanManageOrganization(identity, organization.ID) {
		return types.Organization{}, a.forbidden(ctx, identity, "organization.update.denied", "organization", organization.ID, organization.ID, "", []string{"actor lacks organization management permission"})
	}
	if req.Name != nil {
		organization.Name = strings.TrimSpace(*req.Name)
	}
	if req.Tier != nil {
		organization.Tier = strings.TrimSpace(*req.Tier)
	}
	if req.Mode != nil {
		organization.Mode = strings.TrimSpace(*req.Mode)
	}
	if req.Metadata != nil {
		organization.Metadata = req.Metadata
	}
	organization.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateOrganization(ctx, organization); err != nil {
		return types.Organization{}, err
	}
	if err := a.record(ctx, identity, "organization.updated", "organization", organization.ID, organization.ID, "", []string{"organization updated"}); err != nil {
		return types.Organization{}, err
	}
	return organization, nil
}

func (a *Application) GetProject(ctx context.Context, id string) (types.Project, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Project{}, err
	}
	project, err := a.Store.GetProject(ctx, id)
	if err != nil {
		return types.Project{}, err
	}
	if !a.Authorizer.CanReadProject(identity, project.OrganizationID, project.ID) {
		return types.Project{}, ErrForbidden
	}
	return project, nil
}

func (a *Application) UpdateProject(ctx context.Context, id string, req types.UpdateProjectRequest) (types.Project, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Project{}, err
	}
	project, err := a.Store.GetProject(ctx, id)
	if err != nil {
		return types.Project{}, err
	}
	if !a.Authorizer.CanManageProject(identity, project.OrganizationID, project.ID) {
		return types.Project{}, a.forbidden(ctx, identity, "project.update.denied", "project", project.ID, project.OrganizationID, project.ID, []string{"actor lacks project management permission"})
	}
	if req.Name != nil {
		project.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		project.Slug = strings.TrimSpace(*req.Slug)
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.AdoptionMode != nil {
		project.AdoptionMode = strings.TrimSpace(*req.AdoptionMode)
	}
	if req.Status != nil {
		project.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		project.Metadata = req.Metadata
	}
	project.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateProject(ctx, project); err != nil {
		return types.Project{}, err
	}
	if err := a.record(ctx, identity, "project.updated", "project", project.ID, project.OrganizationID, project.ID, []string{"project updated"}); err != nil {
		return types.Project{}, err
	}
	return project, nil
}

func (a *Application) ArchiveProject(ctx context.Context, id string) (types.Project, error) {
	req := types.UpdateProjectRequest{}
	status := "archived"
	req.Status = &status
	return a.UpdateProject(ctx, id, req)
}

func (a *Application) GetTeam(ctx context.Context, id string) (types.Team, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Team{}, err
	}
	team, err := a.Store.GetTeam(ctx, id)
	if err != nil {
		return types.Team{}, err
	}
	if !a.Authorizer.CanReadProject(identity, team.OrganizationID, team.ProjectID) {
		return types.Team{}, ErrForbidden
	}
	return team, nil
}

func (a *Application) UpdateTeam(ctx context.Context, id string, req types.UpdateTeamRequest) (types.Team, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Team{}, err
	}
	team, err := a.Store.GetTeam(ctx, id)
	if err != nil {
		return types.Team{}, err
	}
	if !a.Authorizer.CanManageTeam(identity, team.OrganizationID, team.ProjectID) {
		return types.Team{}, a.forbidden(ctx, identity, "team.update.denied", "team", team.ID, team.OrganizationID, team.ProjectID, []string{"actor lacks team management permission"})
	}
	if req.Name != nil {
		team.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		team.Slug = strings.TrimSpace(*req.Slug)
	}
	if req.OwnerUserIDs != nil {
		team.OwnerUserIDs = *req.OwnerUserIDs
	}
	if req.Status != nil {
		team.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		team.Metadata = req.Metadata
	}
	team.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateTeam(ctx, team); err != nil {
		return types.Team{}, err
	}
	if err := a.record(ctx, identity, "team.updated", "team", team.ID, team.OrganizationID, team.ProjectID, []string{"team updated"}); err != nil {
		return types.Team{}, err
	}
	return team, nil
}

func (a *Application) ArchiveTeam(ctx context.Context, id string) (types.Team, error) {
	status := "archived"
	return a.UpdateTeam(ctx, id, types.UpdateTeamRequest{Status: &status})
}

func (a *Application) GetService(ctx context.Context, id string) (types.Service, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Service{}, err
	}
	service, err := a.Store.GetService(ctx, id)
	if err != nil {
		return types.Service{}, err
	}
	if !a.Authorizer.CanReadProject(identity, service.OrganizationID, service.ProjectID) {
		return types.Service{}, ErrForbidden
	}
	return service, nil
}

func (a *Application) UpdateService(ctx context.Context, id string, req types.UpdateServiceRequest) (types.Service, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Service{}, err
	}
	service, err := a.Store.GetService(ctx, id)
	if err != nil {
		return types.Service{}, err
	}
	team, err := a.Store.GetTeam(ctx, service.TeamID)
	if err != nil {
		return types.Service{}, err
	}
	if !a.Authorizer.CanManageService(identity, service, team) {
		return types.Service{}, a.forbidden(ctx, identity, "service.update.denied", "service", service.ID, service.OrganizationID, service.ProjectID, []string{"actor lacks service management permission"})
	}
	if req.Name != nil {
		service.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		service.Slug = strings.TrimSpace(*req.Slug)
	}
	if req.Description != nil {
		service.Description = *req.Description
	}
	if req.Criticality != nil {
		service.Criticality = strings.TrimSpace(*req.Criticality)
	}
	if req.Tier != nil {
		service.Tier = strings.TrimSpace(*req.Tier)
	}
	if req.CustomerFacing != nil {
		service.CustomerFacing = *req.CustomerFacing
	}
	if req.HasSLO != nil {
		service.HasSLO = *req.HasSLO
	}
	if req.HasObservability != nil {
		service.HasObservability = *req.HasObservability
	}
	if req.RegulatedZone != nil {
		service.RegulatedZone = *req.RegulatedZone
	}
	if req.DependentServicesCount != nil {
		service.DependentServicesCount = *req.DependentServicesCount
	}
	if req.Status != nil {
		service.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		service.Metadata = req.Metadata
	}
	service.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateService(ctx, service); err != nil {
		return types.Service{}, err
	}
	if err := a.record(ctx, identity, "service.updated", "service", service.ID, service.OrganizationID, service.ProjectID, []string{"service updated"}); err != nil {
		return types.Service{}, err
	}
	return service, nil
}

func (a *Application) ArchiveService(ctx context.Context, id string) (types.Service, error) {
	status := "archived"
	return a.UpdateService(ctx, id, types.UpdateServiceRequest{Status: &status})
}

func (a *Application) GetEnvironment(ctx context.Context, id string) (types.Environment, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Environment{}, err
	}
	environment, err := a.Store.GetEnvironment(ctx, id)
	if err != nil {
		return types.Environment{}, err
	}
	if !a.Authorizer.CanReadProject(identity, environment.OrganizationID, environment.ProjectID) {
		return types.Environment{}, ErrForbidden
	}
	return environment, nil
}

func (a *Application) UpdateEnvironment(ctx context.Context, id string, req types.UpdateEnvironmentRequest) (types.Environment, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Environment{}, err
	}
	environment, err := a.Store.GetEnvironment(ctx, id)
	if err != nil {
		return types.Environment{}, err
	}
	if !a.Authorizer.CanCreateEnvironment(identity, environment.OrganizationID, environment.ProjectID) {
		return types.Environment{}, a.forbidden(ctx, identity, "environment.update.denied", "environment", environment.ID, environment.OrganizationID, environment.ProjectID, []string{"actor lacks environment mutation permission"})
	}
	if req.Name != nil {
		environment.Name = strings.TrimSpace(*req.Name)
	}
	if req.Slug != nil {
		environment.Slug = strings.TrimSpace(*req.Slug)
	}
	if req.Type != nil {
		environment.Type = strings.TrimSpace(*req.Type)
	}
	if req.Region != nil {
		environment.Region = strings.TrimSpace(*req.Region)
	}
	if req.Production != nil {
		environment.Production = *req.Production
	}
	if req.ComplianceZone != nil {
		environment.ComplianceZone = *req.ComplianceZone
	}
	if req.Status != nil {
		environment.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		environment.Metadata = req.Metadata
	}
	environment.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateEnvironment(ctx, environment); err != nil {
		return types.Environment{}, err
	}
	if err := a.record(ctx, identity, "environment.updated", "environment", environment.ID, environment.OrganizationID, environment.ProjectID, []string{"environment updated"}); err != nil {
		return types.Environment{}, err
	}
	return environment, nil
}

func (a *Application) ArchiveEnvironment(ctx context.Context, id string) (types.Environment, error) {
	status := "archived"
	return a.UpdateEnvironment(ctx, id, types.UpdateEnvironmentRequest{Status: &status})
}

func (a *Application) GetChangeSet(ctx context.Context, id string) (types.ChangeSet, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ChangeSet{}, err
	}
	change, err := a.Store.GetChangeSet(ctx, id)
	if err != nil {
		return types.ChangeSet{}, err
	}
	if !a.Authorizer.CanReadProject(identity, change.OrganizationID, change.ProjectID) {
		return types.ChangeSet{}, ErrForbidden
	}
	return change, nil
}

func (a *Application) UpdateIntegration(ctx context.Context, id string, req types.UpdateIntegrationRequest) (types.Integration, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Integration{}, err
	}
	integration, err := a.Store.GetIntegration(ctx, id)
	if err != nil {
		return types.Integration{}, err
	}
	if !a.Authorizer.CanManageIntegrations(identity, integration.OrganizationID) {
		return types.Integration{}, a.forbidden(ctx, identity, "integration.update.denied", "integration", integration.ID, integration.OrganizationID, "", []string{"actor lacks integration management permission"})
	}
	if req.Name != nil {
		integration.Name = strings.TrimSpace(*req.Name)
	}
	if req.Mode != nil {
		integration.Mode = strings.TrimSpace(*req.Mode)
	}
	if req.Status != nil {
		integration.Status = strings.TrimSpace(*req.Status)
	}
	if req.Description != nil {
		integration.Description = *req.Description
	}
	if req.Capabilities != nil {
		integration.Capabilities = *req.Capabilities
	}
	if req.Metadata != nil {
		integration.Metadata = req.Metadata
	}
	integration.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateIntegration(ctx, integration); err != nil {
		return types.Integration{}, err
	}
	if err := a.record(ctx, identity, "integration.updated", "integration", integration.ID, integration.OrganizationID, "", []string{"integration updated"}); err != nil {
		return types.Integration{}, err
	}
	return integration, nil
}

func (a *Application) CreateServiceAccount(ctx context.Context, req types.CreateServiceAccountRequest) (types.ServiceAccount, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ServiceAccount{}, err
	}
	if strings.TrimSpace(req.OrganizationID) == "" || strings.TrimSpace(req.Name) == "" {
		return types.ServiceAccount{}, fmt.Errorf("%w: organization_id and name are required", ErrValidation)
	}
	if !a.Authorizer.CanManageServiceAccounts(identity, req.OrganizationID) {
		return types.ServiceAccount{}, a.forbidden(ctx, identity, "service_account.create.denied", "service_account", "", req.OrganizationID, "", []string{"actor lacks service account management permission"})
	}
	now := time.Now().UTC()
	serviceAccount := types.ServiceAccount{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("svcacct"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:  req.OrganizationID,
		Name:            req.Name,
		Description:     req.Description,
		Role:            valueOrDefault(req.Role, "viewer"),
		CreatedByUserID: identity.ActorID,
		Status:          "active",
	}
	if err := a.Store.CreateServiceAccount(ctx, serviceAccount); err != nil {
		return types.ServiceAccount{}, err
	}
	if err := a.record(ctx, identity, "service_account.created", "service_account", serviceAccount.ID, serviceAccount.OrganizationID, "", []string{serviceAccount.Role}); err != nil {
		return types.ServiceAccount{}, err
	}
	return serviceAccount, nil
}

func (a *Application) ListServiceAccounts(ctx context.Context) ([]types.ServiceAccount, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanReadServiceAccounts(identity, orgID) {
		return nil, ErrForbidden
	}
	return a.Store.ListServiceAccounts(ctx, storage.ServiceAccountQuery{OrganizationID: orgID})
}

func (a *Application) DeactivateServiceAccount(ctx context.Context, id string) (types.ServiceAccount, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ServiceAccount{}, err
	}
	serviceAccount, err := a.Store.GetServiceAccount(ctx, id)
	if err != nil {
		return types.ServiceAccount{}, err
	}
	if !a.Authorizer.CanManageServiceAccounts(identity, serviceAccount.OrganizationID) {
		return types.ServiceAccount{}, a.forbidden(ctx, identity, "service_account.deactivate.denied", "service_account", serviceAccount.ID, serviceAccount.OrganizationID, "", []string{"actor lacks service account management permission"})
	}
	serviceAccount.Status = "inactive"
	serviceAccount.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateServiceAccount(ctx, serviceAccount); err != nil {
		return types.ServiceAccount{}, err
	}
	if err := a.record(ctx, identity, "service_account.deactivated", "service_account", serviceAccount.ID, serviceAccount.OrganizationID, "", []string{"service account deactivated"}); err != nil {
		return types.ServiceAccount{}, err
	}
	return serviceAccount, nil
}

func (a *Application) IssueServiceAccountToken(ctx context.Context, serviceAccountID string, req types.IssueAPITokenRequest) (types.IssuedAPITokenResponse, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	serviceAccount, err := a.Store.GetServiceAccount(ctx, serviceAccountID)
	if err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	if !a.Authorizer.CanManageServiceAccounts(identity, serviceAccount.OrganizationID) {
		return types.IssuedAPITokenResponse{}, a.forbidden(ctx, identity, "api_token.issue.denied", "api_token", "", serviceAccount.OrganizationID, "", []string{"actor lacks token issuance permission"})
	}
	rawToken, tokenPrefix, tokenHash, err := a.Auth.TokenService().GenerateAPIToken()
	if err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	now := time.Now().UTC()
	var expiresAt *time.Time
	if req.ExpiresInHours > 0 {
		expiry := now.Add(time.Duration(req.ExpiresInHours) * time.Hour)
		expiresAt = &expiry
	}
	token := types.APIToken{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("token"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID:   serviceAccount.OrganizationID,
		ServiceAccountID: serviceAccount.ID,
		Name:             valueOrDefault(req.Name, serviceAccount.Name+" automation token"),
		TokenPrefix:      tokenPrefix,
		TokenHash:        tokenHash,
		Status:           "active",
		ExpiresAt:        expiresAt,
	}
	if err := a.Store.CreateAPIToken(ctx, token); err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	if err := a.record(ctx, identity, "api_token.issued", "api_token", token.ID, serviceAccount.OrganizationID, "", []string{token.Name, token.TokenPrefix}); err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	return types.IssuedAPITokenResponse{Token: rawToken, Entry: token}, nil
}

func (a *Application) ListServiceAccountTokens(ctx context.Context, serviceAccountID string) ([]types.APIToken, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	serviceAccount, err := a.Store.GetServiceAccount(ctx, serviceAccountID)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanManageServiceAccounts(identity, serviceAccount.OrganizationID) {
		return nil, ErrForbidden
	}
	return a.Store.ListAPITokens(ctx, storage.APITokenQuery{OrganizationID: serviceAccount.OrganizationID, ServiceAccountID: serviceAccountID})
}

func (a *Application) RevokeAPIToken(ctx context.Context, serviceAccountID, tokenID string) (types.APIToken, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.APIToken{}, err
	}
	serviceAccount, err := a.Store.GetServiceAccount(ctx, serviceAccountID)
	if err != nil {
		return types.APIToken{}, err
	}
	if !a.Authorizer.CanManageServiceAccounts(identity, serviceAccount.OrganizationID) {
		return types.APIToken{}, a.forbidden(ctx, identity, "api_token.revoke.denied", "api_token", tokenID, serviceAccount.OrganizationID, "", []string{"actor lacks token revocation permission"})
	}
	token, err := a.Store.GetAPIToken(ctx, tokenID)
	if err != nil {
		return types.APIToken{}, err
	}
	if token.ServiceAccountID != serviceAccountID {
		return types.APIToken{}, fmt.Errorf("%w: token does not belong to service account", ErrValidation)
	}
	now := time.Now().UTC()
	token.Status = "revoked"
	token.RevokedAt = &now
	token.UpdatedAt = now
	if err := a.Store.UpdateAPIToken(ctx, token); err != nil {
		return types.APIToken{}, err
	}
	if err := a.record(ctx, identity, "api_token.revoked", "api_token", token.ID, serviceAccount.OrganizationID, "", []string{token.TokenPrefix}); err != nil {
		return types.APIToken{}, err
	}
	return token, nil
}

func (a *Application) RotateAPIToken(ctx context.Context, serviceAccountID, tokenID string, req types.RotateAPITokenRequest) (types.IssuedAPITokenResponse, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	serviceAccount, err := a.Store.GetServiceAccount(ctx, serviceAccountID)
	if err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	if !a.Authorizer.CanManageServiceAccounts(identity, serviceAccount.OrganizationID) {
		return types.IssuedAPITokenResponse{}, a.forbidden(ctx, identity, "api_token.rotate.denied", "api_token", tokenID, serviceAccount.OrganizationID, "", []string{"actor lacks token rotation permission"})
	}
	currentToken, err := a.Store.GetAPIToken(ctx, tokenID)
	if err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	if currentToken.ServiceAccountID != serviceAccountID {
		return types.IssuedAPITokenResponse{}, fmt.Errorf("%w: token does not belong to service account", ErrValidation)
	}

	rawToken, tokenPrefix, tokenHash, err := a.Auth.TokenService().GenerateAPIToken()
	if err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	now := time.Now().UTC()
	var expiresAt *time.Time
	if req.ExpiresInHours > 0 {
		expiry := now.Add(time.Duration(req.ExpiresInHours) * time.Hour)
		expiresAt = &expiry
	}
	newToken := types.APIToken{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("token"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID:   serviceAccount.OrganizationID,
		ServiceAccountID: serviceAccountID,
		Name:             valueOrDefault(req.Name, currentToken.Name),
		TokenPrefix:      tokenPrefix,
		TokenHash:        tokenHash,
		Status:           "active",
		ExpiresAt:        expiresAt,
	}
	err = a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
		currentToken.Status = "revoked"
		currentToken.RevokedAt = &now
		currentToken.UpdatedAt = now
		if err := a.Store.UpdateAPIToken(txCtx, currentToken); err != nil {
			return err
		}
		return a.Store.CreateAPIToken(txCtx, newToken)
	})
	if err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	if err := a.record(ctx, identity, "api_token.rotated", "api_token", newToken.ID, serviceAccount.OrganizationID, "", []string{currentToken.ID, newToken.TokenPrefix}); err != nil {
		return types.IssuedAPITokenResponse{}, err
	}
	return types.IssuedAPITokenResponse{Token: rawToken, Entry: newToken}, nil
}

func (a *Application) IngestIntegrationGraph(ctx context.Context, integrationID string, req types.IntegrationGraphIngestRequest) ([]types.GraphRelationship, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanManageIntegrations(identity, integration.OrganizationID) {
		return nil, a.forbidden(ctx, identity, "integration.ingest.denied", "integration", integration.ID, integration.OrganizationID, "", []string{"actor lacks integration ingestion permission"})
	}

	now := time.Now().UTC()
	relationships := make([]types.GraphRelationship, 0)
	err = a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
		touchedServiceIDs := make(map[string]string)

		for _, repositoryInput := range req.Repositories {
			projectID, err := a.resolveRepositoryProject(txCtx, integration.OrganizationID, repositoryInput.ProjectID, repositoryInput.ServiceID)
			if err != nil {
				return err
			}
			repository := types.Repository{
				BaseRecord: types.BaseRecord{
					ID:        stableResourceID("repo", integration.OrganizationID, repositoryInput.Provider, repositoryInput.URL),
					CreatedAt: now,
					UpdatedAt: now,
					Metadata:  repositoryInput.Metadata,
				},
				OrganizationID: integration.OrganizationID,
				ProjectID:      projectID,
				Name:           repositoryInput.Name,
				Provider:       valueOrDefault(repositoryInput.Provider, integration.Kind),
				URL:            repositoryInput.URL,
				DefaultBranch:  valueOrDefault(repositoryInput.DefaultBranch, "main"),
			}
			if err := a.Store.UpsertRepository(txCtx, repository); err != nil {
				return err
			}
			if repositoryInput.ServiceID != "" {
				service, err := a.Store.GetService(txCtx, repositoryInput.ServiceID)
				if err != nil {
					return err
				}
				if service.OrganizationID != integration.OrganizationID {
					return fmt.Errorf("%w: service scope mismatch for repository ingest", ErrValidation)
				}
				touchedServiceIDs[service.ID] = service.ProjectID
				relationship := newGraphRelationship(now, integration.ID, integration.OrganizationID, service.ProjectID, "service_repository", "service", service.ID, "repository", repository.ID)
				if err := a.Store.UpsertGraphRelationship(txCtx, relationship); err != nil {
					return err
				}
				relationships = append(relationships, relationship)
			}
		}

		for _, dependency := range req.ServiceDependencies {
			service, err := a.Store.GetService(txCtx, dependency.ServiceID)
			if err != nil {
				return err
			}
			dependsOnService, err := a.Store.GetService(txCtx, dependency.DependsOnServiceID)
			if err != nil {
				return err
			}
			if service.OrganizationID != integration.OrganizationID || dependsOnService.OrganizationID != integration.OrganizationID {
				return fmt.Errorf("%w: dependency scope mismatch", ErrValidation)
			}
			touchedServiceIDs[service.ID] = service.ProjectID
			relationship := newGraphRelationship(now, integration.ID, integration.OrganizationID, service.ProjectID, "service_dependency", "service", service.ID, "service", dependsOnService.ID)
			relationship.Metadata = types.Metadata{"critical_dependency": dependency.CriticalDependency}
			if err := a.Store.UpsertGraphRelationship(txCtx, relationship); err != nil {
				return err
			}
			relationships = append(relationships, relationship)
		}

		for _, binding := range req.ServiceEnvironments {
			service, err := a.Store.GetService(txCtx, binding.ServiceID)
			if err != nil {
				return err
			}
			environment, err := a.Store.GetEnvironment(txCtx, binding.EnvironmentID)
			if err != nil {
				return err
			}
			if service.OrganizationID != integration.OrganizationID || environment.OrganizationID != integration.OrganizationID {
				return fmt.Errorf("%w: environment binding scope mismatch", ErrValidation)
			}
			touchedServiceIDs[service.ID] = service.ProjectID
			relationship := newGraphRelationship(now, integration.ID, integration.OrganizationID, service.ProjectID, "service_environment", "service", service.ID, "environment", environment.ID)
			if err := a.Store.UpsertGraphRelationship(txCtx, relationship); err != nil {
				return err
			}
			relationships = append(relationships, relationship)
		}

		for _, binding := range req.ChangeRepositories {
			change, err := a.Store.GetChangeSet(txCtx, binding.ChangeSetID)
			if err != nil {
				return err
			}
			repository, err := a.Store.GetRepositoryByURL(txCtx, integration.OrganizationID, binding.RepositoryURL)
			if err != nil {
				return err
			}
			relationship := newGraphRelationship(now, integration.ID, integration.OrganizationID, change.ProjectID, "change_repository", "change_set", change.ID, "repository", repository.ID)
			if err := a.Store.UpsertGraphRelationship(txCtx, relationship); err != nil {
				return err
			}
			relationships = append(relationships, relationship)
		}

		for serviceID, projectID := range touchedServiceIDs {
			relationship := newGraphRelationship(now, integration.ID, integration.OrganizationID, projectID, "service_integration_source", "service", serviceID, "integration", integration.ID)
			if err := a.Store.UpsertGraphRelationship(txCtx, relationship); err != nil {
				return err
			}
			relationships = append(relationships, relationship)
		}

		integration.LastSyncedAt = &now
		integration.UpdatedAt = now
		return a.Store.UpdateIntegration(txCtx, integration)
	})
	if err != nil {
		return nil, err
	}
	if err := a.record(ctx, identity, "integration.graph.ingested", "integration", integration.ID, integration.OrganizationID, "", []string{fmt.Sprintf("relationships=%d", len(relationships))}); err != nil {
		return nil, err
	}
	return relationships, nil
}

func (a *Application) ListGraphRelationships(ctx context.Context) ([]types.GraphRelationship, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListGraphRelationships(ctx, storage.GraphRelationshipQuery{OrganizationID: orgID})
}

func (a *Application) CreateRolloutExecution(ctx context.Context, req types.CreateRolloutExecutionRequest) (types.RolloutExecution, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RolloutExecution{}, err
	}
	plan, err := a.Store.GetRolloutPlan(ctx, req.RolloutPlanID)
	if err != nil {
		return types.RolloutExecution{}, err
	}
	if !a.Authorizer.CanExecuteRollout(identity, plan.OrganizationID, plan.ProjectID) {
		return types.RolloutExecution{}, a.forbidden(ctx, identity, "rollout.execution.create.denied", "rollout_execution", "", plan.OrganizationID, plan.ProjectID, []string{"actor lacks rollout execution permission"})
	}
	change, err := a.Store.GetChangeSet(ctx, plan.ChangeSetID)
	if err != nil {
		return types.RolloutExecution{}, err
	}
	now := time.Now().UTC()
	execution := types.RolloutExecution{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("exec"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: plan.OrganizationID,
		ProjectID:      plan.ProjectID,
		RolloutPlanID:  plan.ID,
		ChangeSetID:    change.ID,
		ServiceID:      change.ServiceID,
		EnvironmentID:  change.EnvironmentID,
		Status:         rollouts.InitialExecutionStatus(plan),
		CurrentStep:    rollouts.InitialExecutionStep(plan),
	}
	if err := a.Store.CreateRolloutExecution(ctx, execution); err != nil {
		return types.RolloutExecution{}, err
	}
	if err := a.record(ctx, identity, "rollout.execution.created", "rollout_execution", execution.ID, execution.OrganizationID, execution.ProjectID, []string{execution.Status}); err != nil {
		return types.RolloutExecution{}, err
	}
	return execution, nil
}

func (a *Application) ListRolloutExecutions(ctx context.Context) ([]types.RolloutExecution, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListRolloutExecutions(ctx, storage.RolloutExecutionQuery{OrganizationID: orgID})
}

func (a *Application) GetRolloutExecutionDetail(ctx context.Context, id string) (types.RolloutExecutionDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RolloutExecutionDetail{}, err
	}
	execution, err := a.Store.GetRolloutExecution(ctx, id)
	if err != nil {
		return types.RolloutExecutionDetail{}, err
	}
	if !a.Authorizer.CanReadProject(identity, execution.OrganizationID, execution.ProjectID) {
		return types.RolloutExecutionDetail{}, ErrForbidden
	}
	results, err := a.Store.ListVerificationResults(ctx, storage.VerificationResultQuery{OrganizationID: execution.OrganizationID, ProjectID: execution.ProjectID, RolloutExecutionID: execution.ID})
	if err != nil {
		return types.RolloutExecutionDetail{}, err
	}
	return types.RolloutExecutionDetail{Execution: execution, VerificationResults: results}, nil
}

func (a *Application) AdvanceRolloutExecution(ctx context.Context, id string, req types.AdvanceRolloutExecutionRequest) (types.RolloutExecution, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.RolloutExecution{}, err
	}
	execution, err := a.Store.GetRolloutExecution(ctx, id)
	if err != nil {
		return types.RolloutExecution{}, err
	}
	if !a.Authorizer.CanExecuteRollout(identity, execution.OrganizationID, execution.ProjectID) {
		return types.RolloutExecution{}, a.forbidden(ctx, identity, "rollout.execution.transition.denied", "rollout_execution", execution.ID, execution.OrganizationID, execution.ProjectID, []string{"actor lacks rollout transition permission"})
	}
	execution, err = rollouts.AdvanceExecution(execution, strings.TrimSpace(req.Action), req.Reason, time.Now().UTC())
	if err != nil {
		return types.RolloutExecution{}, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	execution.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateRolloutExecution(ctx, execution); err != nil {
		return types.RolloutExecution{}, err
	}
	if err := a.record(ctx, identity, "rollout.execution.transitioned", "rollout_execution", execution.ID, execution.OrganizationID, execution.ProjectID, []string{execution.Status, req.Action}); err != nil {
		return types.RolloutExecution{}, err
	}
	return execution, nil
}

func (a *Application) RecordVerificationResult(ctx context.Context, executionID string, req types.RecordVerificationResultRequest) (types.VerificationResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.VerificationResult{}, err
	}
	execution, err := a.Store.GetRolloutExecution(ctx, executionID)
	if err != nil {
		return types.VerificationResult{}, err
	}
	if !a.Authorizer.CanRecordVerification(identity, execution.OrganizationID, execution.ProjectID) {
		return types.VerificationResult{}, a.forbidden(ctx, identity, "verification.record.denied", "verification_result", "", execution.OrganizationID, execution.ProjectID, []string{"actor lacks verification permission"})
	}
	now := time.Now().UTC()
	result := types.VerificationResult{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("verify"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID:         execution.OrganizationID,
		ProjectID:              execution.ProjectID,
		RolloutExecutionID:     execution.ID,
		RolloutPlanID:          execution.RolloutPlanID,
		ChangeSetID:            execution.ChangeSetID,
		ServiceID:              execution.ServiceID,
		EnvironmentID:          execution.EnvironmentID,
		Status:                 "recorded",
		Outcome:                strings.TrimSpace(req.Outcome),
		Decision:               strings.TrimSpace(req.Decision),
		Signals:                req.Signals,
		TechnicalSignalSummary: req.TechnicalSignalSummary,
		BusinessSignalSummary:  req.BusinessSignalSummary,
		Summary:                req.Summary,
		Explanation:            req.Explanation,
	}
	if result.Outcome == "" || result.Decision == "" {
		return types.VerificationResult{}, fmt.Errorf("%w: outcome and decision are required", ErrValidation)
	}

	err = a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := a.Store.CreateVerificationResult(txCtx, result); err != nil {
			return err
		}
		updatedExecution, err := rollouts.ApplyVerificationDecision(execution, result, now)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrValidation, err.Error())
		}
		updatedExecution.UpdatedAt = now
		return a.Store.UpdateRolloutExecution(txCtx, updatedExecution)
	})
	if err != nil {
		return types.VerificationResult{}, err
	}
	if err := a.record(ctx, identity, "verification.recorded", "verification_result", result.ID, result.OrganizationID, result.ProjectID, []string{result.Outcome, result.Decision}); err != nil {
		return types.VerificationResult{}, err
	}
	return result, nil
}

func (a *Application) resolveRepositoryProject(ctx context.Context, organizationID, projectID, serviceID string) (string, error) {
	if strings.TrimSpace(projectID) != "" {
		project, err := a.Store.GetProject(ctx, projectID)
		if err != nil {
			return "", err
		}
		if project.OrganizationID != organizationID {
			return "", fmt.Errorf("%w: repository project scope mismatch", ErrValidation)
		}
		return project.ID, nil
	}
	if strings.TrimSpace(serviceID) == "" {
		return "", fmt.Errorf("%w: project_id or service_id is required for repository ingest", ErrValidation)
	}
	service, err := a.Store.GetService(ctx, serviceID)
	if err != nil {
		return "", err
	}
	if service.OrganizationID != organizationID {
		return "", fmt.Errorf("%w: repository service scope mismatch", ErrValidation)
	}
	return service.ProjectID, nil
}

func newGraphRelationship(now time.Time, integrationID, organizationID, projectID, relationshipType, fromType, fromID, toType, toID string) types.GraphRelationship {
	return types.GraphRelationship{
		BaseRecord: types.BaseRecord{
			ID:        stableResourceID("graph", organizationID, integrationID, relationshipType, fromType, fromID, toType, toID),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID:      organizationID,
		ProjectID:           projectID,
		SourceIntegrationID: integrationID,
		RelationshipType:    relationshipType,
		FromResourceType:    fromType,
		FromResourceID:      fromID,
		ToResourceType:      toType,
		ToResourceID:        toID,
		Status:              "active",
		LastObservedAt:      now,
	}
}

func stableResourceID(prefix string, parts ...string) string {
	normalized := strings.Join(parts, "::")
	sum := sha256.Sum256([]byte(normalized))
	return prefix + "_" + hex.EncodeToString(sum[:10])
}

func (a *Application) machineActorIdentity(ctx context.Context) (auth.Identity, bool) {
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || !identity.Authenticated {
		return auth.Identity{}, false
	}
	return identity, true
}

func notFoundAsValidation(err error, format string, args ...any) error {
	if errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("%w: %s", ErrValidation, fmt.Sprintf(format, args...))
	}
	return err
}
