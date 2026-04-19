package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/delivery"
	liveintegrations "github.com/change-control-plane/change-control-plane/internal/integrations"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/internal/verification"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) ListRepositories(ctx context.Context, query storage.RepositoryQuery) ([]types.Repository, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	query.OrganizationID = orgID
	return a.Store.ListRepositories(ctx, query)
}

func (a *Application) UpdateRepository(ctx context.Context, id string, req types.UpdateRepositoryRequest) (types.Repository, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.Repository{}, err
	}
	repository, err := a.Store.GetRepository(ctx, id)
	if err != nil {
		return types.Repository{}, err
	}
	if !a.Authorizer.CanManageIntegrations(identity, repository.OrganizationID) {
		return types.Repository{}, a.forbidden(ctx, identity, "repository.update.denied", "repository", repository.ID, repository.OrganizationID, repository.ProjectID, []string{"actor lacks repository mapping permission"})
	}
	repository.Metadata = ensureMetadata(repository.Metadata)
	var service types.Service
	var environment types.Environment
	var serviceSet bool
	var environmentSet bool
	now := time.Now().UTC()

	if req.Name != nil {
		repository.Name = strings.TrimSpace(*req.Name)
	}
	if req.DefaultBranch != nil {
		repository.DefaultBranch = valueOrDefault(strings.TrimSpace(*req.DefaultBranch), "main")
	}
	if req.Status != nil {
		repository.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		repository.Metadata = mergeMetadata(repository.Metadata, req.Metadata)
	}
	if req.ServiceID != nil {
		repository.ServiceID = strings.TrimSpace(*req.ServiceID)
		if repository.ServiceID != "" {
			service, err = a.Store.GetService(ctx, repository.ServiceID)
			if err != nil {
				return types.Repository{}, err
			}
			if service.OrganizationID != repository.OrganizationID {
				return types.Repository{}, fmt.Errorf("%w: repository service scope mismatch", ErrValidation)
			}
			repository.ProjectID = service.ProjectID
			serviceSet = true
			setMappingProvenance(repository.Metadata, "service", mappingSourceManual, now, "repository mapped to service by operator")
		}
		if repository.ServiceID == "" {
			clearMappingProvenance(repository.Metadata, "service")
		}
	}
	if req.ProjectID != nil && !serviceSet {
		repository.ProjectID = strings.TrimSpace(*req.ProjectID)
		if repository.ProjectID != "" {
			project, err := a.Store.GetProject(ctx, repository.ProjectID)
			if err != nil {
				return types.Repository{}, err
			}
			if project.OrganizationID != repository.OrganizationID {
				return types.Repository{}, fmt.Errorf("%w: repository project scope mismatch", ErrValidation)
			}
			setMappingProvenance(repository.Metadata, "project", mappingSourceManual, now, "repository project set by operator")
		}
		if repository.ProjectID == "" {
			clearMappingProvenance(repository.Metadata, "project")
		}
	}
	if req.EnvironmentID != nil {
		repository.EnvironmentID = strings.TrimSpace(*req.EnvironmentID)
		if repository.EnvironmentID != "" {
			environment, err = a.Store.GetEnvironment(ctx, repository.EnvironmentID)
			if err != nil {
				return types.Repository{}, err
			}
			if environment.OrganizationID != repository.OrganizationID {
				return types.Repository{}, fmt.Errorf("%w: repository environment scope mismatch", ErrValidation)
			}
			environmentSet = true
			if repository.ProjectID == "" {
				repository.ProjectID = environment.ProjectID
			}
			setMappingProvenance(repository.Metadata, "environment", mappingSourceManual, now, "repository mapped to environment by operator")
		}
		if repository.EnvironmentID == "" {
			clearMappingProvenance(repository.Metadata, "environment")
		}
	}
	if repository.ProjectID == "" && repository.ServiceID != "" {
		service, err = a.Store.GetService(ctx, repository.ServiceID)
		if err != nil {
			return types.Repository{}, err
		}
		repository.ProjectID = service.ProjectID
		serviceSet = true
	}
	if repository.ProjectID != "" && environmentSet && environment.ProjectID != repository.ProjectID {
		return types.Repository{}, fmt.Errorf("%w: repository environment project mismatch", ErrValidation)
	}
	if repository.ProjectID != "" && serviceSet && service.ProjectID != repository.ProjectID {
		return types.Repository{}, fmt.Errorf("%w: repository service project mismatch", ErrValidation)
	}
	if repository.Status == "" {
		if repository.ServiceID != "" || repository.EnvironmentID != "" {
			repository.Status = "mapped"
		} else {
			repository.Status = "discovered"
		}
	}
	repository, err = a.applyRepositoryInferredOwnership(ctx, repository, now)
	if err != nil {
		return types.Repository{}, err
	}
	repository.UpdatedAt = now
	if err := a.Store.UpdateRepository(ctx, repository); err != nil {
		return types.Repository{}, err
	}
	if err := a.ensureRepositoryGraphMappings(ctx, repository); err != nil {
		return types.Repository{}, err
	}
	if err := a.record(ctx, identity, "repository.updated", "repository", repository.ID, repository.OrganizationID, repository.ProjectID, []string{repository.URL, repository.ServiceID, repository.EnvironmentID}); err != nil {
		return types.Repository{}, err
	}
	return repository, nil
}

func (a *Application) ListIntegrationSyncRuns(ctx context.Context, integrationID string) ([]types.IntegrationSyncRun, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return nil, err
	}
	if !identity.HasOrganizationAccess(integration.OrganizationID) {
		return nil, ErrForbidden
	}
	return a.Store.ListIntegrationSyncRuns(ctx, storage.IntegrationSyncRunQuery{
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		Limit:          50,
	})
}

func (a *Application) TestIntegrationConnection(ctx context.Context, integrationID string) (types.IntegrationTestResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.IntegrationTestResult{}, err
	}
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return types.IntegrationTestResult{}, err
	}
	if !a.Authorizer.CanManageIntegrations(identity, integration.OrganizationID) {
		return types.IntegrationTestResult{}, a.forbidden(ctx, identity, "integration.test.denied", "integration", integration.ID, integration.OrganizationID, "", []string{"actor lacks integration test permission"})
	}
	if err := validateIntegrationConfiguration(integration, false); err != nil {
		run := a.integrationRunForError(integration, "test_connection", err)
		integration, _ = a.persistIntegrationRun(ctx, integration, run)
		return types.IntegrationTestResult{Integration: integration, Run: run}, err
	}

	now := time.Now().UTC()
	run := types.IntegrationSyncRun{
		BaseRecord: types.BaseRecord{
			ID:        types.BaseRecord{}.ID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		Operation:      "test_connection",
		Trigger:        "manual",
		Status:         "success",
		Summary:        fmt.Sprintf("%s connection test succeeded", integration.Name),
		StartedAt:      now,
		CompletedAt:    &now,
	}
	run.ID = commonID("isr")
	details, err := a.executeIntegrationTest(ctx, integration)
	if err != nil {
		run = a.integrationRunForError(integration, "test_connection", err)
		integration, _ = a.persistIntegrationRun(ctx, integration, run)
		return types.IntegrationTestResult{Integration: integration, Run: run}, err
	}
	if isSCMIntegrationKind(integration.Kind) {
		if registration, registrationErr := a.ensureWebhookRegistration(ctx, integration, false); registrationErr == nil {
			details = append(details, registration.Details...)
		} else {
			details = append(details, "webhook_registration="+truncateString(registrationErr.Error(), 200))
		}
	}
	run.Details = details
	integration, err = a.persistIntegrationRun(ctx, integration, run)
	if err != nil {
		return types.IntegrationTestResult{}, err
	}
	if err := a.record(ctx, identity, "integration.tested", "integration", integration.ID, integration.OrganizationID, "", details); err != nil {
		return types.IntegrationTestResult{}, err
	}
	return types.IntegrationTestResult{Integration: hydrateIntegrationRuntimeState(integration, time.Now().UTC()), Run: run}, nil
}

func (a *Application) SyncIntegration(ctx context.Context, integrationID string) (types.IntegrationSyncResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.IntegrationSyncResult{}, err
	}
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return types.IntegrationSyncResult{}, err
	}
	if !a.Authorizer.CanManageIntegrations(identity, integration.OrganizationID) {
		return types.IntegrationSyncResult{}, a.forbidden(ctx, identity, "integration.sync.denied", "integration", integration.ID, integration.OrganizationID, "", []string{"actor lacks integration sync permission"})
	}
	result, err := a.syncIntegrationWithTrigger(ctx, integration, "manual", nil)
	if err != nil {
		return result, err
	}
	if isSCMIntegrationKind(result.Integration.Kind) {
		if registration, registrationErr := a.ensureWebhookRegistration(ctx, result.Integration, false); registrationErr == nil {
			result.Run.Details = append(result.Run.Details, registration.Details...)
		} else {
			result.Run.Details = append(result.Run.Details, "webhook_registration="+truncateString(registrationErr.Error(), 200))
		}
	}
	if err := a.record(ctx, identity, "integration.synced", "integration", result.Integration.ID, result.Integration.OrganizationID, "", compactDetailList(result.Run.Details)); err != nil {
		return types.IntegrationSyncResult{}, err
	}
	return result, nil
}

func (a *Application) syncIntegrationWithTrigger(ctx context.Context, integration types.Integration, trigger string, scheduledFor *time.Time) (types.IntegrationSyncResult, error) {
	if err := validateIntegrationConfiguration(integration, false); err != nil {
		run := a.integrationRunForError(integration, "sync", err)
		run.Trigger = trigger
		run.ScheduledFor = scheduledFor
		integration, _ = a.persistIntegrationRun(ctx, integration, run)
		return types.IntegrationSyncResult{Integration: hydrateIntegrationRuntimeState(integration, time.Now().UTC()), Run: run}, err
	}
	var (
		repositories        []types.Repository
		discoveredResources []types.DiscoveredResource
		relationships       []types.GraphRelationship
		run                 types.IntegrationSyncRun
		err                 error
	)
	switch strings.ToLower(strings.TrimSpace(integration.Kind)) {
	case "github", "gitlab":
		repositories, run, err = a.syncSCMIntegration(ctx, integration)
	case "kubernetes":
		run, discoveredResources, err = a.syncKubernetesIntegration(ctx, integration)
	case "prometheus":
		run, discoveredResources, err = a.syncPrometheusIntegration(ctx, integration)
	default:
		err = fmt.Errorf("%w: sync is not implemented for integration kind %s", ErrValidation, integration.Kind)
		run = a.integrationRunForError(integration, "sync", err)
	}
	run.Trigger = valueOrDefault(strings.TrimSpace(trigger), "manual")
	run.ScheduledFor = scheduledFor
	if err != nil {
		integration, _ = a.persistIntegrationRun(ctx, integration, run)
		return types.IntegrationSyncResult{Integration: hydrateIntegrationRuntimeState(integration, time.Now().UTC()), Run: run}, err
	}
	integration, err = a.persistIntegrationRun(ctx, integration, run)
	if err != nil {
		return types.IntegrationSyncResult{}, err
	}
	if isSCMIntegrationKind(integration.Kind) {
		for _, repository := range repositories {
			if err := a.ensureRepositoryGraphMappings(ctx, repository); err != nil {
				return types.IntegrationSyncResult{}, err
			}
		}
		relationships, _ = a.Store.ListGraphRelationships(ctx, storage.GraphRelationshipQuery{
			OrganizationID:      integration.OrganizationID,
			SourceIntegrationID: integration.ID,
			Limit:               200,
		})
	}
	for _, resource := range discoveredResources {
		if err := a.ensureDiscoveredResourceGraphMappings(ctx, resource); err != nil {
			return types.IntegrationSyncResult{}, err
		}
	}
	return types.IntegrationSyncResult{
		Integration:         hydrateIntegrationRuntimeState(integration, time.Now().UTC()),
		Run:                 run,
		Repositories:        repositories,
		DiscoveredResources: discoveredResources,
		Relationships:       relationships,
	}, nil
}

func (a *Application) executeIntegrationTest(ctx context.Context, integration types.Integration) ([]string, error) {
	switch strings.ToLower(strings.TrimSpace(integration.Kind)) {
	case "github", "gitlab":
		client, scope, err := scmClientFromIntegration(ctx, integration)
		if err != nil {
			return nil, err
		}
		return client.TestConnection(ctx, scope)
	case "kubernetes":
		_, details, err := observeKubernetesIntegration(ctx, integration)
		return details, err
	case "prometheus":
		_, details, err := collectPrometheusIntegration(ctx, integration)
		return details, err
	default:
		return nil, fmt.Errorf("%w: connection test is not implemented for integration kind %s", ErrValidation, integration.Kind)
	}
}

func (a *Application) syncGitHubIntegration(ctx context.Context, integration types.Integration) ([]types.Repository, types.IntegrationSyncRun, error) {
	client, owner, err := githubClientFromIntegration(ctx, integration)
	if err != nil {
		return nil, a.integrationRunForError(integration, "sync", err), err
	}
	items, err := client.DiscoverRepositories(ctx, owner)
	if err != nil {
		return nil, a.integrationRunForError(integration, "sync", err), err
	}
	now := time.Now().UTC()
	repositories := make([]types.Repository, 0, len(items))
	for _, discovered := range items {
		existing, lookupErr := a.Store.GetRepositoryByURL(ctx, integration.OrganizationID, discovered.HTMLURL)
		repository := types.Repository{
			BaseRecord: types.BaseRecord{
				ID:        stableResourceID("repo", integration.OrganizationID, integration.Kind, discovered.HTMLURL),
				CreatedAt: now,
				UpdatedAt: now,
				Metadata: types.Metadata{
					"full_name":             discovered.FullName,
					"owner":                 discovered.Owner,
					"private":               discovered.Private,
					"archived":              discovered.Archived,
					"source_integration_id": integration.ID,
				},
			},
			OrganizationID:      integration.OrganizationID,
			ProjectID:           "",
			SourceIntegrationID: integration.ID,
			Name:                discovered.Name,
			Provider:            "github",
			URL:                 discovered.HTMLURL,
			DefaultBranch:       valueOrDefault(discovered.DefaultBranch, "main"),
			Status:              "discovered",
			LastSyncedAt:        &now,
		}
		if lookupErr == nil {
			repository = existing
			if repository.Metadata == nil {
				repository.Metadata = types.Metadata{}
			}
			repository.Name = valueOrDefault(discovered.Name, repository.Name)
			repository.Provider = "github"
			repository.URL = discovered.HTMLURL
			repository.DefaultBranch = valueOrDefault(discovered.DefaultBranch, repository.DefaultBranch)
			if repository.SourceIntegrationID == "" {
				repository.SourceIntegrationID = integration.ID
			}
			repository.Status = valueOrDefault(repository.Status, "discovered")
			repository.LastSyncedAt = &now
			repository.Metadata["full_name"] = discovered.FullName
			repository.Metadata["owner"] = discovered.Owner
			repository.Metadata["private"] = discovered.Private
			repository.Metadata["archived"] = discovered.Archived
			repository.Metadata["source_integration_id"] = integration.ID
			repository.Metadata["integration_instances"] = appendUniqueMetadataStrings(repository.Metadata["integration_instances"], integration.ID)
			repository.UpdatedAt = now
		} else {
			repository.Metadata["integration_instances"] = []string{integration.ID}
		}
		if err := a.Store.UpsertRepository(ctx, repository); err != nil {
			return nil, a.integrationRunForError(integration, "sync", err), err
		}
		repositories = append(repositories, repository)
	}
	run := types.IntegrationSyncRun{
		BaseRecord: types.BaseRecord{
			ID:        commonID("isr"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  types.Metadata{"owner": owner},
		},
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		Operation:      "sync",
		Status:         "success",
		Summary:        fmt.Sprintf("discovered %d github repositories", len(repositories)),
		Details:        []string{fmt.Sprintf("owner=%s", valueOrDefault(owner, "authenticated-principal"))},
		ResourceCount:  len(repositories),
		StartedAt:      now,
		CompletedAt:    &now,
	}
	return repositories, run, nil
}

func (a *Application) syncKubernetesIntegration(ctx context.Context, integration types.Integration) (types.IntegrationSyncRun, []types.DiscoveredResource, error) {
	result, details, err := observeKubernetesIntegration(ctx, integration)
	if err != nil {
		return a.integrationRunForError(integration, "sync", err), nil, err
	}
	discoveredResources, discoveryDetails, discoveryErr := a.discoverKubernetesResources(ctx, integration)
	if discoveryErr != nil {
		details = append(details, "discovery_warning="+truncateString(discoveryErr.Error(), 200))
	} else {
		details = append(details, discoveryDetails...)
	}
	now := time.Now().UTC()
	return types.IntegrationSyncRun{
		BaseRecord: types.BaseRecord{
			ID:        commonID("isr"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		Operation:      "sync",
		Status:         "success",
		Summary:        fmt.Sprintf("observed kubernetes workload state: %s", valueOrDefault(result.BackendStatus, "unknown")),
		Details:        details,
		ResourceCount:  maxInt(1, len(discoveredResources)),
		StartedAt:      now,
		CompletedAt:    &now,
	}, discoveredResources, nil
}

func (a *Application) syncPrometheusIntegration(ctx context.Context, integration types.Integration) (types.IntegrationSyncRun, []types.DiscoveredResource, error) {
	collection, details, err := collectPrometheusIntegration(ctx, integration)
	if err != nil {
		return a.integrationRunForError(integration, "sync", err), nil, err
	}
	discoveredResources, discoveryDetails, discoveryErr := a.persistPrometheusDiscoveredResources(ctx, integration, collection)
	if discoveryErr != nil {
		details = append(details, "coverage_warning="+truncateString(discoveryErr.Error(), 200))
	} else {
		details = append(details, discoveryDetails...)
	}
	resourceCount := 0
	if len(collection.Snapshots) > 0 {
		resourceCount = len(collection.Snapshots[0].Signals)
	}
	now := time.Now().UTC()
	return types.IntegrationSyncRun{
		BaseRecord: types.BaseRecord{
			ID:        commonID("isr"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		Operation:      "sync",
		Status:         "success",
		Summary:        fmt.Sprintf("collected prometheus signal evidence: %s", summarizePrometheusCollection(collection)),
		Details:        details,
		ResourceCount:  maxInt(maxInt(1, resourceCount), len(discoveredResources)),
		StartedAt:      now,
		CompletedAt:    &now,
	}, discoveredResources, nil
}

func (a *Application) persistIntegrationRun(ctx context.Context, integration types.Integration, run types.IntegrationSyncRun) (types.Integration, error) {
	now := time.Now().UTC()
	if strings.TrimSpace(run.Trigger) == "" {
		if strings.HasPrefix(run.Operation, "github.webhook") {
			run.Trigger = "webhook"
		} else {
			run.Trigger = "manual"
		}
	}
	integration.UpdatedAt = now
	switch run.Operation {
	case "test_connection":
		integration.LastTestedAt = &now
	default:
		integration.LastSyncedAt = &now
		integration.LastSyncAttemptedAt = &run.StartedAt
	}
	if run.Status == "success" || run.Status == "duplicate" || run.Status == "skipped" {
		integration.ConnectionHealth = "healthy"
		integration.LastError = ""
		if run.Operation != "test_connection" {
			integration.LastSyncSucceededAt = &now
			integration.SyncConsecutiveFailures = 0
		}
		if integration.Enabled {
			integration.Status = "connected"
		}
	} else {
		integration.ConnectionHealth = "error"
		integration.LastError = run.Summary
		if run.Operation != "test_connection" {
			integration.LastSyncFailedAt = &now
			integration.SyncConsecutiveFailures++
		}
		if integration.Enabled {
			integration.Status = "error"
		}
	}
	integration.SyncClaimedAt = nil
	integration.NextScheduledSyncAt = nextIntegrationSyncTime(integration, run, now)
	integration = hydrateIntegrationRuntimeState(integration, now)
	err := a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := a.Store.CreateIntegrationSyncRun(txCtx, run); err != nil {
			return err
		}
		return a.Store.UpdateIntegration(txCtx, integration)
	})
	if err != nil {
		return types.Integration{}, err
	}
	return integration, nil
}

func (a *Application) integrationRunForError(integration types.Integration, operation string, err error) types.IntegrationSyncRun {
	now := time.Now().UTC()
	return types.IntegrationSyncRun{
		BaseRecord: types.BaseRecord{
			ID:        commonID("isr"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		Operation:      operation,
		Trigger:        "manual",
		Status:         "error",
		Summary:        err.Error(),
		Details:        []string{err.Error()},
		ErrorClass:     classifyIntegrationError(err),
		StartedAt:      now,
		CompletedAt:    &now,
	}
}

func (a *Application) ensureRepositoryGraphMappings(ctx context.Context, repository types.Repository) error {
	sourceIntegrationID := stringMetadataValue(repository.Metadata, "source_integration_id")
	if sourceIntegrationID == "" {
		return nil
	}
	now := time.Now().UTC()
	if repository.ServiceID != "" {
		service, err := a.Store.GetService(ctx, repository.ServiceID)
		if err != nil {
			return err
		}
		relationship := newGraphRelationship(now, sourceIntegrationID, repository.OrganizationID, service.ProjectID, "service_repository", "service", service.ID, "repository", repository.ID)
		relationship.Metadata = relationshipProvenance(mappingProvenanceSource(repository.Metadata, "service", mappingSourceManual), repository.URL, "service_repository")
		if err := a.Store.UpsertGraphRelationship(ctx, relationship); err != nil {
			return err
		}
		source := newGraphRelationship(now, sourceIntegrationID, repository.OrganizationID, service.ProjectID, "service_integration_source", "service", service.ID, "integration", sourceIntegrationID)
		source.Metadata = relationshipProvenance(discoverySourceIntegrationSync, repository.URL, "service_integration_source")
		if err := a.Store.UpsertGraphRelationship(ctx, source); err != nil {
			return err
		}
		if teamID := ownershipTeamID(repository.Metadata); teamID != "" {
			ownerRelationship := newGraphRelationship(now, sourceIntegrationID, repository.OrganizationID, service.ProjectID, "team_repository_owner", "team", teamID, "repository", repository.ID)
			ownerRelationship.Metadata = relationshipProvenance("inferred_owner", repository.ServiceID, repository.URL)
			if err := a.Store.UpsertGraphRelationship(ctx, ownerRelationship); err != nil {
				return err
			}
		}
	}
	if repository.ServiceID != "" && repository.EnvironmentID != "" {
		service, err := a.Store.GetService(ctx, repository.ServiceID)
		if err != nil {
			return err
		}
		relationship := newGraphRelationship(now, sourceIntegrationID, repository.OrganizationID, service.ProjectID, "service_environment", "service", repository.ServiceID, "environment", repository.EnvironmentID)
		relationship.Metadata = relationshipProvenance(mappingProvenanceSource(repository.Metadata, "environment", mappingSourceManual), repository.ServiceID, repository.EnvironmentID)
		if err := a.Store.UpsertGraphRelationship(ctx, relationship); err != nil {
			return err
		}
	}
	return nil
}

func validateIntegrationConfiguration(integration types.Integration, strict bool) error {
	kind := strings.ToLower(strings.TrimSpace(integration.Kind))
	mode := normalizeIntegrationMode(integration.Mode)
	integration.AuthStrategy = normalizeIntegrationAuthStrategy(integration.Kind, integration.AuthStrategy, integration.Metadata)
	if mode == "" {
		return fmt.Errorf("%w: integration mode is required", ErrValidation)
	}
	if !integration.Enabled {
		if integration.ControlEnabled {
			return fmt.Errorf("%w: disabled integrations cannot enable active control", ErrValidation)
		}
		return nil
	}
	switch kind {
	case "github":
		switch integration.AuthStrategy {
		case "github_app":
			if stringMetadataValue(integration.Metadata, "app_id") == "" {
				return fmt.Errorf("%w: github app integrations require app_id", ErrValidation)
			}
			if stringMetadataValue(integration.Metadata, "app_slug") == "" {
				return fmt.Errorf("%w: github app integrations require app_slug", ErrValidation)
			}
			if stringMetadataValue(integration.Metadata, "private_key_env") == "" {
				return fmt.Errorf("%w: github app integrations require private_key_env", ErrValidation)
			}
			if stringMetadataValue(integration.Metadata, "installation_id") == "" {
				return fmt.Errorf("%w: github app integrations require installation_id after onboarding completes", ErrValidation)
			}
		default:
			if stringMetadataValue(integration.Metadata, "access_token_env") == "" {
				return fmt.Errorf("%w: github integrations require access_token_env", ErrValidation)
			}
		}
		if strict && stringMetadataValue(integration.Metadata, "webhook_secret_env") == "" {
			return fmt.Errorf("%w: github integrations require webhook_secret_env for authenticated webhook ingest", ErrValidation)
		}
	case "gitlab":
		if stringMetadataValue(integration.Metadata, "access_token_env") == "" {
			return fmt.Errorf("%w: gitlab integrations require access_token_env", ErrValidation)
		}
		if strict && stringMetadataValue(integration.Metadata, "webhook_secret_env") == "" {
			return fmt.Errorf("%w: gitlab integrations require webhook_secret_env for authenticated webhook ingest", ErrValidation)
		}
	case "kubernetes":
		if stringMetadataValue(integration.Metadata, "api_base_url") == "" {
			return fmt.Errorf("%w: kubernetes integrations require api_base_url", ErrValidation)
		}
		if stringMetadataValue(integration.Metadata, "status_path") == "" && (stringMetadataValue(integration.Metadata, "namespace") == "" || stringMetadataValue(integration.Metadata, "deployment_name") == "") {
			return fmt.Errorf("%w: kubernetes integrations require status_path or namespace + deployment_name", ErrValidation)
		}
	case "prometheus":
		if stringMetadataValue(integration.Metadata, "api_base_url") == "" {
			return fmt.Errorf("%w: prometheus integrations require api_base_url", ErrValidation)
		}
		if len(decodePrometheusQueryTemplates(integration.Metadata["queries"])) == 0 {
			return fmt.Errorf("%w: prometheus integrations require at least one query template", ErrValidation)
		}
	default:
		return fmt.Errorf("%w: unsupported integration kind %s", ErrValidation, integration.Kind)
	}
	if integration.ControlEnabled && mode != "active_control" {
		return fmt.Errorf("%w: control_enabled requires mode active_control", ErrValidation)
	}
	if integration.ScheduleEnabled {
		if integration.ScheduleIntervalSeconds <= 0 {
			return fmt.Errorf("%w: schedule_interval_seconds must be greater than zero when schedule_enabled is true", ErrValidation)
		}
		if integration.SyncStaleAfterSeconds < 0 {
			return fmt.Errorf("%w: sync_stale_after_seconds cannot be negative", ErrValidation)
		}
	}
	return nil
}

func normalizeIntegrationMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "advisory-ready", "advisory", "read_only", "read-only":
		return "advisory"
	case "active_control", "control", "governance":
		return "active_control"
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}

func normalizeIntegrationInstanceKey(raw, fallback string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(fallback))
	}
	if value == "" {
		return "default"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && builder.Len() > 0 {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "default"
	}
	return result
}

func normalizeIntegrationScopeType(scopeType string) string {
	switch strings.ToLower(strings.TrimSpace(scopeType)) {
	case "", "organization", "org":
		return "organization"
	case "environment", "service", "repository_group", "repository", "team":
		return strings.ToLower(strings.TrimSpace(scopeType))
	default:
		return strings.ToLower(strings.TrimSpace(scopeType))
	}
}

func normalizeIntegrationAuthStrategy(kind, strategy string, metadata types.Metadata) string {
	normalized := strings.ToLower(strings.TrimSpace(strategy))
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "github":
		switch normalized {
		case "", "pat", "token", "personal_access_token", "personal-access-token":
			if stringMetadataValue(metadata, "app_id") != "" || stringMetadataValue(metadata, "installation_id") != "" || stringMetadataValue(metadata, "private_key_env") != "" {
				return "github_app"
			}
			return "personal_access_token"
		case "app", "github_app", "github-app":
			return "github_app"
		default:
			return normalized
		}
	case "gitlab":
		switch normalized {
		case "", "pat", "token", "personal_access_token", "personal-access-token":
			return "personal_access_token"
		default:
			return normalized
		}
	case "kubernetes", "prometheus":
		if normalized == "" {
			return "bearer_token_env"
		}
		return normalized
	default:
		return normalized
	}
}

func initialIntegrationOnboardingStatus(integration types.Integration) string {
	switch strings.ToLower(strings.TrimSpace(integration.Kind)) {
	case "github":
		switch normalizeIntegrationAuthStrategy(integration.Kind, integration.AuthStrategy, integration.Metadata) {
		case "github_app":
			if stringMetadataValue(integration.Metadata, "installation_id") != "" {
				return "installed"
			}
			return "not_started"
		case "personal_access_token":
			if stringMetadataValue(integration.Metadata, "access_token_env") != "" {
				return "legacy_configured"
			}
		}
	case "gitlab":
		if stringMetadataValue(integration.Metadata, "access_token_env") != "" {
			return "configured"
		}
	}
	return "not_started"
}

func integrationAllowsActiveControl(integration *types.Integration) bool {
	if integration == nil {
		return false
	}
	return integration.Enabled && integration.ControlEnabled && normalizeIntegrationMode(integration.Mode) == "active_control"
}

func githubClientFromIntegration(ctx context.Context, integration types.Integration) (liveintegrations.GitHubClient, string, error) {
	baseURL := stringMetadataValue(integration.Metadata, "api_base_url")
	switch normalizeIntegrationAuthStrategy(integration.Kind, integration.AuthStrategy, integration.Metadata) {
	case "github_app":
		privateKeyEnv := stringMetadataValue(integration.Metadata, "private_key_env")
		if privateKeyEnv == "" {
			return liveintegrations.GitHubClient{}, "", fmt.Errorf("%w: github app private_key_env is required", ErrValidation)
		}
		privateKey := strings.TrimSpace(os.Getenv(privateKeyEnv))
		if privateKey == "" {
			return liveintegrations.GitHubClient{}, "", fmt.Errorf("%w: github app private key env %s is empty", ErrValidation, privateKeyEnv)
		}
		appID := stringMetadataValue(integration.Metadata, "app_id")
		installationID := stringMetadataValue(integration.Metadata, "installation_id")
		token, _, err := liveintegrations.CreateGitHubAppInstallationToken(ctx, baseURL, appID, installationID, privateKey)
		if err != nil {
			return liveintegrations.GitHubClient{}, "", err
		}
		return liveintegrations.NewGitHubClient(baseURL, token), stringMetadataValue(integration.Metadata, "owner"), nil
	default:
		tokenEnv := stringMetadataValue(integration.Metadata, "access_token_env")
		if tokenEnv == "" {
			return liveintegrations.GitHubClient{}, "", fmt.Errorf("%w: github access_token_env is required", ErrValidation)
		}
		token := strings.TrimSpace(os.Getenv(tokenEnv))
		if token == "" {
			return liveintegrations.GitHubClient{}, "", fmt.Errorf("%w: github token env %s is empty", ErrValidation, tokenEnv)
		}
		return liveintegrations.NewGitHubClient(baseURL, token), stringMetadataValue(integration.Metadata, "owner"), nil
	}
}

func executeHTTPConnectivityCheck(ctx context.Context, method, baseURL, path, bearerTokenEnv string, headers map[string]string) ([]string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("%w: api_base_url is required", ErrValidation)
	}
	endpoint := strings.TrimRight(strings.TrimSpace(baseURL), "/") + path
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if bearerTokenEnv != "" {
		token := strings.TrimSpace(os.Getenv(bearerTokenEnv))
		if token == "" {
			return nil, fmt.Errorf("%w: bearer token env %s is empty", ErrValidation, bearerTokenEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("remote request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return compactDetailList([]string{
		fmt.Sprintf("endpoint=%s", endpoint),
		fmt.Sprintf("status=%d", resp.StatusCode),
		fmt.Sprintf("body=%s", truncateString(strings.TrimSpace(string(body)), 160)),
	}), nil
}

type prometheusQueryTemplate struct {
	Name          string `json:"name"`
	Query         string `json:"query"`
	Category      string `json:"category,omitempty"`
	ServiceID     string `json:"service_id,omitempty"`
	EnvironmentID string `json:"environment_id,omitempty"`
	ResourceName  string `json:"resource_name,omitempty"`
}

func decodePrometheusQueryTemplates(raw any) []prometheusQueryTemplate {
	metadata := types.Metadata{"queries": raw}
	queriesRaw, ok := metadata["queries"]
	if !ok || queriesRaw == nil {
		return nil
	}
	payload, err := jsonMarshal(queriesRaw)
	if err != nil {
		return nil
	}
	var queries []prometheusQueryTemplate
	if err := jsonUnmarshal(payload, &queries); err != nil {
		return nil
	}
	return queries
}

func observeKubernetesIntegration(ctx context.Context, integration types.Integration) (delivery.SyncResult, []string, error) {
	provider := delivery.NewKubernetesDeploymentProvider()
	result, err := provider.Sync(ctx, kubernetesIntegrationRuntimeContext(integration))
	if err != nil {
		return delivery.SyncResult{}, nil, err
	}
	details := compactDetailList([]string{
		fmt.Sprintf("connection=ok"),
		fmt.Sprintf("backend_status=%s", valueOrDefault(result.BackendStatus, "unknown")),
		fmt.Sprintf("progress_percent=%d", result.ProgressPercent),
		fmt.Sprintf("current_step=%s", valueOrDefault(result.CurrentStep, "unknown")),
		fmt.Sprintf("namespace=%s", metadataString(result.Metadata, "namespace")),
		fmt.Sprintf("deployment_name=%s", metadataString(result.Metadata, "deployment_name")),
		fmt.Sprintf("replicas=%s", metadataString(result.Metadata, "replicas")),
		fmt.Sprintf("updated_replicas=%s", metadataString(result.Metadata, "updated_replicas")),
		fmt.Sprintf("available_replicas=%s", metadataString(result.Metadata, "available_replicas")),
		fmt.Sprintf("unavailable_replicas=%s", metadataString(result.Metadata, "unavailable_replicas")),
		fmt.Sprintf("paused=%s", metadataString(result.Metadata, "paused")),
	})
	return result, details, nil
}

func kubernetesIntegrationRuntimeContext(integration types.Integration) types.RolloutExecutionRuntimeContext {
	namespace := valueOrDefault(stringMetadataValue(integration.Metadata, "namespace"), "default")
	deploymentName := valueOrDefault(stringMetadataValue(integration.Metadata, "deployment_name"), "service")
	return types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord:           types.BaseRecord{ID: stableResourceID("exec", integration.OrganizationID, integration.ID, "kubernetes")},
			OrganizationID:       integration.OrganizationID,
			BackendType:          "kubernetes",
			Status:               "in_progress",
			BackendIntegrationID: integration.ID,
		},
		Service:            types.Service{Slug: deploymentName},
		Environment:        types.Environment{Slug: namespace},
		BackendIntegration: &integration,
	}
}

func collectPrometheusIntegration(ctx context.Context, integration types.Integration) (verification.Collection, []string, error) {
	provider := verification.NewPrometheusProvider()
	collection, err := provider.Collect(ctx, prometheusIntegrationRuntimeContext(integration))
	if err != nil {
		return verification.Collection{}, nil, err
	}
	details := []string{
		"connection=ok",
		fmt.Sprintf("snapshot_count=%d", len(collection.Snapshots)),
		fmt.Sprintf("source=%s", valueOrDefault(collection.Source, "prometheus")),
	}
	if len(collection.Snapshots) > 0 {
		snapshot := collection.Snapshots[len(collection.Snapshots)-1]
		details = append(details,
			fmt.Sprintf("health=%s", snapshot.Health),
			fmt.Sprintf("window_start=%s", snapshot.WindowStart.Format(time.RFC3339)),
			fmt.Sprintf("window_end=%s", snapshot.WindowEnd.Format(time.RFC3339)),
			fmt.Sprintf("signal_count=%d", len(snapshot.Signals)),
			fmt.Sprintf("summary=%s", snapshot.Summary),
		)
		for _, signal := range snapshot.Signals {
			details = append(details, fmt.Sprintf("signal.%s=%0.3f%s (%s)", signal.Name, signal.Value, signal.Unit, signal.Status))
		}
	}
	return collection, compactDetailList(append(details, collection.Explanation...)), nil
}

func prometheusIntegrationRuntimeContext(integration types.Integration) types.RolloutExecutionRuntimeContext {
	return types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord:          types.BaseRecord{ID: stableResourceID("exec", integration.OrganizationID, integration.ID, "prometheus")},
			OrganizationID:      integration.OrganizationID,
			SignalProviderType:  "prometheus",
			Status:              "in_progress",
			SignalIntegrationID: integration.ID,
		},
		SignalIntegration: &integration,
	}
}

func summarizePrometheusCollection(collection verification.Collection) string {
	if len(collection.Snapshots) == 0 {
		return "no snapshots collected"
	}
	snapshot := collection.Snapshots[len(collection.Snapshots)-1]
	return fmt.Sprintf("%s across %d signal(s)", snapshot.Health, len(snapshot.Signals))
}

func stringMetadataValue(metadata types.Metadata, key string) string {
	if metadata == nil {
		return ""
	}
	raw, ok := metadata[key]
	if !ok {
		return ""
	}
	typed, _ := raw.(string)
	return strings.TrimSpace(typed)
}

func truncateString(value string, max int) string {
	if len(value) <= max {
		return value
	}
	if max <= 3 {
		return value[:max]
	}
	return value[:max-3] + "..."
}

func appendUniqueMetadataStrings(raw any, values ...string) []string {
	seen := map[string]struct{}{}
	items := []string{}
	appendValue := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	switch typed := raw.(type) {
	case []string:
		for _, value := range typed {
			appendValue(value)
		}
	case []any:
		for _, value := range typed {
			if stringValue, ok := value.(string); ok {
				appendValue(stringValue)
			}
		}
	case string:
		appendValue(typed)
	}
	for _, value := range values {
		appendValue(value)
	}
	return items
}

func commonID(prefix string) string {
	return common.NewID(strings.ToLower(prefix))
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func jsonMarshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func jsonUnmarshal(data []byte, target any) error {
	return json.Unmarshal(data, target)
}
