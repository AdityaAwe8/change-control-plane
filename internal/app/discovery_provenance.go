package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	liveintegrations "github.com/change-control-plane/change-control-plane/internal/integrations"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

const (
	discoverySourceIntegrationSync    = "integration_sync"
	discoverySourceRuntimeObservation = "runtime_observation"
	mappingSourceManual               = "manual"
	mappingSourceIntegrationIngest    = "integration_graph_ingest"
	mappingSourceInferredName         = "inferred_name_match"
	mappingSourceInferredNamespace    = "inferred_namespace_match"
	mappingSourceQueryTemplate        = "query_template"
	mappingSourceRepositoryMapping    = "repository_mapping"
	ownershipSourceCodeowners         = "codeowners_import"
	ownershipSourceServiceMapping     = "service_mapping"
)

func ensureMetadata(metadata types.Metadata) types.Metadata {
	if metadata == nil {
		return types.Metadata{}
	}
	return metadata
}

func mergeMetadata(base, incoming types.Metadata) types.Metadata {
	result := ensureMetadata(base)
	for key, value := range ensureMetadata(incoming) {
		result[key] = value
	}
	return result
}

func metadataObject(value any) types.Metadata {
	switch typed := value.(type) {
	case nil:
		return types.Metadata{}
	case types.Metadata:
		return ensureMetadata(typed)
	case map[string]any:
		return types.Metadata(typed)
	default:
		return types.Metadata{}
	}
}

func metadataStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if value := strings.TrimSpace(fmt.Sprint(item)); value != "" {
				result = append(result, value)
			}
		}
		return result
	default:
		return nil
	}
}

func setMappingProvenance(metadata types.Metadata, field, source string, now time.Time, details ...string) {
	if metadata == nil || strings.TrimSpace(field) == "" {
		return
	}
	mapping := metadataObject(metadata["mapping_provenance"])
	entry := types.Metadata{
		"source":     strings.TrimSpace(source),
		"updated_at": now.UTC().Format(time.RFC3339),
	}
	if len(details) > 0 {
		entry["details"] = compactDetailList(details)
	}
	mapping[field] = entry
	metadata["mapping_provenance"] = map[string]any(mapping)
}

func clearMappingProvenance(metadata types.Metadata, fields ...string) {
	if metadata == nil {
		return
	}
	mapping := metadataObject(metadata["mapping_provenance"])
	for _, field := range fields {
		delete(mapping, field)
	}
	if len(mapping) == 0 {
		delete(metadata, "mapping_provenance")
		return
	}
	metadata["mapping_provenance"] = map[string]any(mapping)
}

func mappingProvenanceSource(metadata types.Metadata, field, fallback string) string {
	entry := metadataObject(metadataObject(metadata["mapping_provenance"])[field])
	if source := stringMetadataValue(entry, "source"); source != "" {
		return source
	}
	return strings.TrimSpace(fallback)
}

func setRepositoryDiscoveryProvenance(repository *types.Repository, integration types.Integration, discovered liveintegrations.SCMRepository, now time.Time) {
	repository.Metadata = ensureMetadata(repository.Metadata)
	repository.Metadata["provenance"] = map[string]any{
		"source":                discoverySourceIntegrationSync,
		"provider":              integration.Kind,
		"source_integration_id": integration.ID,
		"external_id":           discovered.ExternalID,
		"namespace":             discovered.Namespace,
		"owner":                 discovered.Owner,
		"full_name":             discovered.FullName,
		"recorded_at":           now.UTC().Format(time.RFC3339),
	}
}

func setDiscoveredResourceProvenance(resource *types.DiscoveredResource, source string, now time.Time, details types.Metadata) {
	resource.Metadata = ensureMetadata(resource.Metadata)
	provenance := types.Metadata{
		"source":       source,
		"provider":     resource.Provider,
		"integration":  resource.IntegrationID,
		"external_id":  resource.ExternalID,
		"resource_type": resource.ResourceType,
		"recorded_at":  now.UTC().Format(time.RFC3339),
	}
	for key, value := range details {
		provenance[key] = value
	}
	resource.Metadata["provenance"] = map[string]any(provenance)
}

func applyRepositoryOwnershipImport(repository types.Repository, result liveintegrations.RepositoryOwnershipImport, now time.Time) types.Repository {
	repository.Metadata = ensureMetadata(repository.Metadata)
	ownership := types.Metadata{
		"source":     ownershipSourceCodeowners,
		"provider":   result.Provider,
		"status":     result.Status,
		"checked_at": now.UTC().Format(time.RFC3339),
	}
	if result.FilePath != "" {
		ownership["file_path"] = result.FilePath
	}
	if result.Ref != "" {
		ownership["ref"] = result.Ref
	}
	if result.Revision != "" {
		ownership["revision"] = result.Revision
	}
	if len(result.Owners) > 0 {
		ownership["owners"] = result.Owners
	}
	if len(result.Rules) > 0 {
		rules := make([]map[string]any, 0, len(result.Rules))
		for _, rule := range result.Rules {
			rules = append(rules, map[string]any{
				"pattern": rule.Pattern,
				"owners":  rule.Owners,
			})
		}
		ownership["rules"] = rules
		ownership["imported_at"] = now.UTC().Format(time.RFC3339)
	}
	if result.Error != "" {
		ownership["error"] = result.Error
	}
	repository.Metadata["ownership"] = map[string]any(ownership)
	repository.UpdatedAt = now
	return repository
}

func (a *Application) applyRepositoryInferredOwnership(ctx context.Context, repository types.Repository, now time.Time) (types.Repository, error) {
	repository.Metadata = ensureMetadata(repository.Metadata)
	if strings.TrimSpace(repository.ServiceID) == "" {
		delete(repository.Metadata, "inferred_owner")
		return repository, nil
	}
	service, err := a.Store.GetService(ctx, repository.ServiceID)
	if err != nil {
		return repository, err
	}
	if strings.TrimSpace(service.TeamID) == "" {
		delete(repository.Metadata, "inferred_owner")
		return repository, nil
	}
	team, err := a.Store.GetTeam(ctx, service.TeamID)
	if err != nil {
		return repository, err
	}
	repository.Metadata["inferred_owner"] = map[string]any{
		"source":       ownershipSourceServiceMapping,
		"mode":         "inferred",
		"team_id":      team.ID,
		"team_name":    team.Name,
		"service_id":   service.ID,
		"service_name": service.Name,
		"updated_at":   now.UTC().Format(time.RFC3339),
	}
	return repository, nil
}

func (a *Application) applyDiscoveredResourceInferredOwnership(ctx context.Context, resource types.DiscoveredResource, now time.Time) (types.DiscoveredResource, error) {
	resource.Metadata = ensureMetadata(resource.Metadata)
	serviceID := strings.TrimSpace(resource.ServiceID)
	source := ownershipSourceServiceMapping
	if serviceID == "" && strings.TrimSpace(resource.RepositoryID) != "" {
		repository, err := a.Store.GetRepository(ctx, resource.RepositoryID)
		if err != nil {
			return resource, err
		}
		serviceID = strings.TrimSpace(repository.ServiceID)
		source = mappingSourceRepositoryMapping
	}
	if serviceID == "" {
		delete(resource.Metadata, "inferred_owner")
		return resource, nil
	}
	service, err := a.Store.GetService(ctx, serviceID)
	if err != nil {
		return resource, err
	}
	if strings.TrimSpace(service.TeamID) == "" {
		delete(resource.Metadata, "inferred_owner")
		return resource, nil
	}
	team, err := a.Store.GetTeam(ctx, service.TeamID)
	if err != nil {
		return resource, err
	}
	resource.Metadata["inferred_owner"] = map[string]any{
		"source":       source,
		"mode":         "inferred",
		"team_id":      team.ID,
		"team_name":    team.Name,
		"service_id":   service.ID,
		"service_name": service.Name,
		"updated_at":   now.UTC().Format(time.RFC3339),
	}
	return resource, nil
}

func relationshipProvenance(source string, details ...string) types.Metadata {
	metadata := types.Metadata{
		"provenance_source": strings.TrimSpace(source),
	}
	if len(details) > 0 {
		metadata["evidence"] = compactDetailList(details)
	}
	return metadata
}

func ownershipTeamID(metadata types.Metadata) string {
	return stringMetadataValue(metadataObject(metadata["inferred_owner"]), "team_id")
}
