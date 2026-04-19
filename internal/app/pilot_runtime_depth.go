package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/delivery"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/internal/verification"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

const integrationClaimTTL = 2 * time.Minute

func defaultIntegrationScheduleInterval(kind string) int {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "kubernetes", "prometheus":
		return 300
	case "github", "gitlab":
		return 900
	default:
		return 600
	}
}

func defaultIntegrationStaleAfter(kind string, intervalSeconds int) int {
	interval := intervalSeconds
	if interval <= 0 {
		interval = defaultIntegrationScheduleInterval(kind)
	}
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "github", "gitlab":
		return maxInt(interval*2, 1800)
	default:
		return maxInt(interval*2, 600)
	}
}

func hydrateIntegrationRuntimeState(integration types.Integration, now time.Time) types.Integration {
	if integration.ScheduleIntervalSeconds <= 0 && integration.ScheduleEnabled {
		integration.ScheduleIntervalSeconds = defaultIntegrationScheduleInterval(integration.Kind)
	}
	if integration.SyncStaleAfterSeconds <= 0 {
		integration.SyncStaleAfterSeconds = defaultIntegrationStaleAfter(integration.Kind, integration.ScheduleIntervalSeconds)
	}
	integration.FreshnessState, integration.Stale, integration.SyncLagSeconds = integrationFreshnessState(integration, now)
	return integration
}

func applyIntegrationScheduleDefaults(integration types.Integration, now time.Time) types.Integration {
	if integration.Enabled && integration.ScheduleIntervalSeconds <= 0 {
		integration.ScheduleIntervalSeconds = defaultIntegrationScheduleInterval(integration.Kind)
	}
	if integration.Enabled && integration.ScheduleEnabled && integration.SyncStaleAfterSeconds <= 0 {
		integration.SyncStaleAfterSeconds = defaultIntegrationStaleAfter(integration.Kind, integration.ScheduleIntervalSeconds)
	}
	if !integration.Enabled || !integration.ScheduleEnabled {
		integration.NextScheduledSyncAt = nil
		integration.SyncClaimedAt = nil
		if !integration.Enabled {
			integration.SyncConsecutiveFailures = 0
		}
		return hydrateIntegrationRuntimeState(integration, now)
	}
	if integration.NextScheduledSyncAt == nil {
		if integration.LastSyncSucceededAt != nil {
			next := integration.LastSyncSucceededAt.Add(time.Duration(integration.ScheduleIntervalSeconds) * time.Second)
			integration.NextScheduledSyncAt = &next
		} else {
			next := now
			integration.NextScheduledSyncAt = &next
		}
	}
	return hydrateIntegrationRuntimeState(integration, now)
}

func integrationFreshnessState(integration types.Integration, now time.Time) (string, bool, int) {
	if !integration.Enabled {
		return "disabled", false, 0
	}
	if !integration.ScheduleEnabled {
		return "manual_only", false, 0
	}
	reference := integration.LastSyncSucceededAt
	if reference == nil {
		reference = integration.LastSyncAttemptedAt
	}
	lagSeconds := 0
	if reference != nil {
		lagSeconds = int(now.Sub(*reference).Seconds())
		if lagSeconds < 0 {
			lagSeconds = 0
		}
	}
	staleThreshold := integration.SyncStaleAfterSeconds
	if staleThreshold <= 0 {
		staleThreshold = defaultIntegrationStaleAfter(integration.Kind, integration.ScheduleIntervalSeconds)
	}
	if integration.LastSyncFailedAt != nil && (integration.LastSyncSucceededAt == nil || integration.LastSyncFailedAt.After(*integration.LastSyncSucceededAt)) {
		stale := lagSeconds > staleThreshold
		if stale {
			return "stale_error", true, lagSeconds
		}
		return "error", false, lagSeconds
	}
	if integration.LastSyncSucceededAt == nil {
		if integration.NextScheduledSyncAt != nil && now.After(integration.NextScheduledSyncAt.Add(time.Duration(staleThreshold)*time.Second)) {
			return "stale_pending", true, lagSeconds
		}
		return "scheduled", false, lagSeconds
	}
	if lagSeconds > staleThreshold {
		return "stale", true, lagSeconds
	}
	return "fresh", false, lagSeconds
}

func classifyIntegrationError(err error) string {
	if err == nil {
		return ""
	}
	var providerErr *delivery.ProviderError
	if errors.As(err, &providerErr) {
		if providerErr.Temporary {
			return "temporary_provider"
		}
		return "provider"
	}
	var signalErr *verification.SignalProviderError
	if errors.As(err, &signalErr) {
		if signalErr.Temporary {
			return "temporary_signal_provider"
		}
		return "signal_provider"
	}
	switch {
	case errors.Is(err, ErrValidation):
		return "validation"
	case errors.Is(err, ErrForbidden):
		return "forbidden"
	case errors.Is(err, delivery.ErrProviderUnavailable), errors.Is(err, verification.ErrSignalProviderUnavailable):
		return "provider_unavailable"
	default:
		return "runtime"
	}
}

func nextIntegrationSyncTime(integration types.Integration, run types.IntegrationSyncRun, now time.Time) *time.Time {
	if !integration.Enabled || !integration.ScheduleEnabled {
		return nil
	}
	interval := integration.ScheduleIntervalSeconds
	if interval <= 0 {
		interval = defaultIntegrationScheduleInterval(integration.Kind)
	}
	nextDelay := time.Duration(interval) * time.Second
	if run.Status == "error" {
		failures := maxInt(1, integration.SyncConsecutiveFailures)
		backoffSeconds := 60
		for step := 1; step < failures; step++ {
			backoffSeconds *= 2
			if backoffSeconds >= 900 {
				backoffSeconds = 900
				break
			}
		}
		if retryDelay := time.Duration(backoffSeconds) * time.Second; retryDelay < nextDelay {
			nextDelay = retryDelay
		}
	}
	next := now.Add(nextDelay)
	return &next
}

func (a *Application) ListDiscoveredResources(ctx context.Context, query storage.DiscoveredResourceQuery) ([]types.DiscoveredResource, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	query.OrganizationID = orgID
	return a.Store.ListDiscoveredResources(ctx, query)
}

func (a *Application) UpdateDiscoveredResource(ctx context.Context, id string, req types.UpdateDiscoveredResourceRequest) (types.DiscoveredResource, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DiscoveredResource{}, err
	}
	resource, err := a.Store.GetDiscoveredResource(ctx, id)
	if err != nil {
		return types.DiscoveredResource{}, err
	}
	if !a.Authorizer.CanManageIntegrations(identity, resource.OrganizationID) {
		return types.DiscoveredResource{}, a.forbidden(ctx, identity, "discovered_resource.update.denied", "discovered_resource", resource.ID, resource.OrganizationID, resource.ProjectID, []string{"actor lacks discovered-resource mapping permission"})
	}
	resource.Metadata = ensureMetadata(resource.Metadata)
	now := time.Now().UTC()
	if req.ProjectID != nil {
		resource.ProjectID = strings.TrimSpace(*req.ProjectID)
		if resource.ProjectID != "" {
			project, err := a.Store.GetProject(ctx, resource.ProjectID)
			if err != nil {
				return types.DiscoveredResource{}, err
			}
			if project.OrganizationID != resource.OrganizationID {
				return types.DiscoveredResource{}, fmt.Errorf("%w: discovered resource project scope mismatch", ErrValidation)
			}
			setMappingProvenance(resource.Metadata, "project", mappingSourceManual, now, "discovered resource project set by operator")
		}
		if resource.ProjectID == "" {
			clearMappingProvenance(resource.Metadata, "project")
		}
	}
	if req.ServiceID != nil {
		resource.ServiceID = strings.TrimSpace(*req.ServiceID)
		if resource.ServiceID != "" {
			service, err := a.Store.GetService(ctx, resource.ServiceID)
			if err != nil {
				return types.DiscoveredResource{}, err
			}
			if service.OrganizationID != resource.OrganizationID {
				return types.DiscoveredResource{}, fmt.Errorf("%w: discovered resource service scope mismatch", ErrValidation)
			}
			resource.ProjectID = service.ProjectID
			setMappingProvenance(resource.Metadata, "service", mappingSourceManual, now, "discovered resource mapped to service by operator")
		}
		if resource.ServiceID == "" {
			clearMappingProvenance(resource.Metadata, "service")
		}
	}
	if req.EnvironmentID != nil {
		resource.EnvironmentID = strings.TrimSpace(*req.EnvironmentID)
		if resource.EnvironmentID != "" {
			environment, err := a.Store.GetEnvironment(ctx, resource.EnvironmentID)
			if err != nil {
				return types.DiscoveredResource{}, err
			}
			if environment.OrganizationID != resource.OrganizationID {
				return types.DiscoveredResource{}, fmt.Errorf("%w: discovered resource environment scope mismatch", ErrValidation)
			}
			if resource.ProjectID == "" {
				resource.ProjectID = environment.ProjectID
			}
			setMappingProvenance(resource.Metadata, "environment", mappingSourceManual, now, "discovered resource mapped to environment by operator")
		}
		if resource.EnvironmentID == "" {
			clearMappingProvenance(resource.Metadata, "environment")
		}
	}
	if req.RepositoryID != nil {
		resource.RepositoryID = strings.TrimSpace(*req.RepositoryID)
		if resource.RepositoryID != "" {
			repository, err := a.Store.GetRepository(ctx, resource.RepositoryID)
			if err != nil {
				return types.DiscoveredResource{}, err
			}
			if repository.OrganizationID != resource.OrganizationID {
				return types.DiscoveredResource{}, fmt.Errorf("%w: discovered resource repository scope mismatch", ErrValidation)
			}
			if resource.ProjectID == "" {
				resource.ProjectID = repository.ProjectID
			}
			setMappingProvenance(resource.Metadata, "repository", mappingSourceManual, now, "discovered resource mapped to repository by operator")
		}
		if resource.RepositoryID == "" {
			clearMappingProvenance(resource.Metadata, "repository")
		}
	}
	if req.Status != nil {
		resource.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		resource.Metadata = mergeMetadata(resource.Metadata, req.Metadata)
	}
	if resource.Status == "" {
		if resource.ServiceID != "" && resource.EnvironmentID != "" {
			resource.Status = "mapped"
		} else {
			resource.Status = "candidate"
		}
	}
	resource, err = a.applyDiscoveredResourceInferredOwnership(ctx, resource, now)
	if err != nil {
		return types.DiscoveredResource{}, err
	}
	resource.UpdatedAt = now
	if err := a.Store.UpdateDiscoveredResource(ctx, resource); err != nil {
		return types.DiscoveredResource{}, err
	}
	if err := a.ensureDiscoveredResourceGraphMappings(ctx, resource); err != nil {
		return types.DiscoveredResource{}, err
	}
	if err := a.record(ctx, identity, "discovered_resource.updated", "discovered_resource", resource.ID, resource.OrganizationID, resource.ProjectID, []string{resource.ResourceType, resource.Name, resource.ServiceID, resource.EnvironmentID}); err != nil {
		return types.DiscoveredResource{}, err
	}
	return resource, nil
}

func (a *Application) ensureDiscoveredResourceGraphMappings(ctx context.Context, resource types.DiscoveredResource) error {
	if resource.IntegrationID == "" {
		return nil
	}
	now := time.Now().UTC()
	if resource.RepositoryID != "" {
		repositoryRelationship := newGraphRelationship(now, resource.IntegrationID, resource.OrganizationID, resource.ProjectID, "discovered_resource_repository", "discovered_resource", resource.ID, "repository", resource.RepositoryID)
		repositoryRelationship.Metadata = relationshipProvenance(mappingProvenanceSource(resource.Metadata, "repository", mappingSourceRepositoryMapping), resource.ID, resource.RepositoryID)
		if err := a.Store.UpsertGraphRelationship(ctx, repositoryRelationship); err != nil {
			return err
		}
	}
	if resource.ServiceID == "" {
		return nil
	}
	service, err := a.Store.GetService(ctx, resource.ServiceID)
	if err != nil {
		return err
	}
	workloadRelationship := newGraphRelationship(now, resource.IntegrationID, resource.OrganizationID, service.ProjectID, "service_runtime_workload", "service", service.ID, "discovered_resource", resource.ID)
	workloadRelationship.Metadata = relationshipProvenance(mappingProvenanceSource(resource.Metadata, "service", mappingSourceManual), resource.Name, resource.ResourceType)
	if err := a.Store.UpsertGraphRelationship(ctx, workloadRelationship); err != nil {
		return err
	}
	sourceRelationship := newGraphRelationship(now, resource.IntegrationID, resource.OrganizationID, service.ProjectID, "service_integration_source", "service", service.ID, "integration", resource.IntegrationID)
	sourceRelationship.Metadata = relationshipProvenance(discoverySourceRuntimeObservation, resource.ResourceType, resource.Name)
	if err := a.Store.UpsertGraphRelationship(ctx, sourceRelationship); err != nil {
		return err
	}
	if resource.EnvironmentID != "" {
		environmentRelationship := newGraphRelationship(now, resource.IntegrationID, resource.OrganizationID, service.ProjectID, "service_environment", "service", service.ID, "environment", resource.EnvironmentID)
		environmentRelationship.Metadata = relationshipProvenance(mappingProvenanceSource(resource.Metadata, "environment", mappingSourceManual), service.ID, resource.EnvironmentID)
		if err := a.Store.UpsertGraphRelationship(ctx, environmentRelationship); err != nil {
			return err
		}
	}
	if teamID := ownershipTeamID(resource.Metadata); teamID != "" {
		ownerRelationship := newGraphRelationship(now, resource.IntegrationID, resource.OrganizationID, service.ProjectID, "team_discovered_resource_owner", "team", teamID, "discovered_resource", resource.ID)
		ownerRelationship.Metadata = relationshipProvenance("inferred_owner", resource.ServiceID, resource.Name)
		if err := a.Store.UpsertGraphRelationship(ctx, ownerRelationship); err != nil {
			return err
		}
	}
	return nil
}

func (a *Application) CoverageSummary(ctx context.Context) (types.CoverageSummary, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.CoverageSummary{}, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return types.CoverageSummary{}, err
	}
	integrations, err := a.Store.ListIntegrations(ctx, storage.IntegrationQuery{OrganizationID: orgID})
	if err != nil {
		return types.CoverageSummary{}, err
	}
	repositories, err := a.Store.ListRepositories(ctx, storage.RepositoryQuery{OrganizationID: orgID, Limit: 1000})
	if err != nil {
		return types.CoverageSummary{}, err
	}
	discoveredResources, err := a.Store.ListDiscoveredResources(ctx, storage.DiscoveredResourceQuery{OrganizationID: orgID, Limit: 2000})
	if err != nil {
		return types.CoverageSummary{}, err
	}
	summary := types.CoverageSummary{}
	servicesWithSignalCoverage := map[string]struct{}{}
	environmentsWithWorkloadCoverage := map[string]struct{}{}
	now := time.Now().UTC()
	for _, integration := range integrations {
		integration = hydrateIntegrationRuntimeState(integration, now)
		if integration.Enabled {
			summary.EnabledIntegrations++
		}
		switch strings.ToLower(strings.TrimSpace(integration.Kind)) {
		case "github":
			summary.GitHubIntegrations++
		case "gitlab":
			summary.GitLabIntegrations++
		case "kubernetes":
			summary.KubernetesIntegrations++
		case "prometheus":
			summary.PrometheusIntegrations++
		}
		if integration.ConnectionHealth == "healthy" {
			summary.HealthyIntegrations++
		}
		if integration.Stale {
			summary.StaleIntegrations++
		}
	}
	summary.Repositories = len(repositories)
	for _, repository := range repositories {
		if repository.ServiceID == "" || repository.EnvironmentID == "" {
			summary.UnmappedRepositories++
		}
	}
	summary.DiscoveredResources = len(discoveredResources)
	for _, resource := range discoveredResources {
		if resource.ServiceID == "" || resource.EnvironmentID == "" {
			summary.UnmappedDiscoveredResources++
		}
		if resource.ResourceType == "kubernetes_workload" && resource.EnvironmentID != "" && resource.Status != "missing" {
			environmentsWithWorkloadCoverage[resource.EnvironmentID] = struct{}{}
		}
		if resource.ResourceType == "prometheus_signal_target" && resource.ServiceID != "" && resource.Status != "missing" {
			servicesWithSignalCoverage[resource.ServiceID] = struct{}{}
		}
	}
	summary.WorkloadCoverageEnvironments = len(environmentsWithWorkloadCoverage)
	summary.SignalCoverageServices = len(servicesWithSignalCoverage)
	return summary, nil
}

func (a *Application) QueryStatusEvents(ctx context.Context, query storage.StatusEventQuery) (types.StatusEventQueryResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.StatusEventQueryResult{}, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return types.StatusEventQueryResult{}, err
	}
	query.OrganizationID = orgID
	events, err := a.Store.ListStatusEvents(ctx, query)
	if err != nil {
		return types.StatusEventQueryResult{}, err
	}
	total, err := a.Store.CountStatusEvents(ctx, query)
	if err != nil {
		return types.StatusEventQueryResult{}, err
	}
	result := types.StatusEventQueryResult{
		Events: events,
		Summary: types.StatusEventQuerySummary{
			Total:    total,
			Returned: len(events),
			Limit:    query.Limit,
			Offset:   query.Offset,
		},
		Filters: types.Metadata{
			"search":               query.Search,
			"rollback_only":        query.RollbackOnly,
			"project_id":           query.ProjectID,
			"service_id":           query.ServiceID,
			"environment_id":       query.EnvironmentID,
			"rollout_execution_id": query.RolloutExecutionID,
			"resource_type":        query.ResourceType,
			"resource_id":          query.ResourceID,
			"source":               query.Source,
			"event_type":           strings.Join(query.EventTypes, ","),
			"limit":                query.Limit,
			"offset":               query.Offset,
		},
	}
	if query.Automated != nil {
		result.Filters["automated"] = *query.Automated
	}
	for index, event := range events {
		if index == 0 {
			latest := event.CreatedAt.Format(time.RFC3339)
			result.Summary.LatestEventAt = &latest
		}
		oldest := event.CreatedAt.Format(time.RFC3339)
		result.Summary.OldestEventAt = &oldest
		if strings.Contains(strings.ToLower(event.EventType), "rollback") || event.NewState == "rolled_back" || strings.Contains(strings.ToLower(event.Summary), "rollback") {
			result.Summary.RollbackEvents++
		}
		if event.Automated {
			result.Summary.AutomatedEvents++
		}
	}
	return result, nil
}

func (a *Application) ClaimScheduledIntegrationSync(ctx context.Context, integrationID string, claimedAt time.Time) (bool, error) {
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return false, err
	}
	if identity, ok := currentIdentity(ctx); ok && identity.Authenticated && !identity.HasOrganizationAccess(integration.OrganizationID) {
		return false, ErrForbidden
	}
	return a.Store.ClaimIntegrationSync(ctx, integrationID, claimedAt, claimedAt.Add(-integrationClaimTTL), claimedAt)
}

func (a *Application) RunScheduledIntegrationSync(ctx context.Context, integrationID string) (types.IntegrationSyncResult, error) {
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return types.IntegrationSyncResult{}, err
	}
	if identity, ok := currentIdentity(ctx); ok && identity.Authenticated && !identity.HasOrganizationAccess(integration.OrganizationID) {
		return types.IntegrationSyncResult{}, ErrForbidden
	}
	trigger := "scheduled"
	if integration.SyncConsecutiveFailures > 0 {
		trigger = "retry"
	}
	result, syncErr := a.syncIntegrationWithTrigger(ctx, integration, trigger, integration.NextScheduledSyncAt)
	actor := systemIdentity()
	if identity, ok := currentIdentity(ctx); ok && identity.Authenticated {
		actor = identity
	}
	recordDetails := compactDetailList(append([]string{fmt.Sprintf("trigger=%s", trigger)}, result.Run.Details...))
	if syncErr == nil {
		if err := a.record(ctx, actor, "integration.sync.scheduled", "integration", integration.ID, integration.OrganizationID, "", recordDetails); err != nil {
			return types.IntegrationSyncResult{}, err
		}
		return result, nil
	}
	_ = a.record(ctx, actor, "integration.sync.scheduled_failed", "integration", integration.ID, integration.OrganizationID, "", compactDetailList(append(recordDetails, syncErr.Error())))
	return result, syncErr
}
