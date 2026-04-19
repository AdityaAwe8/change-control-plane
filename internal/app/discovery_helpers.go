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

	"github.com/change-control-plane/change-control-plane/internal/delivery"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/internal/verification"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) discoverKubernetesResources(ctx context.Context, integration types.Integration) ([]types.DiscoveredResource, []string, error) {
	baseURL := strings.TrimRight(stringMetadataValue(integration.Metadata, "api_base_url"), "/")
	namespace := valueOrDefault(stringMetadataValue(integration.Metadata, "namespace"), "default")
	if baseURL == "" {
		return nil, nil, fmt.Errorf("%w: kubernetes integrations require api_base_url for discovery", ErrValidation)
	}
	inventoryPath := stringMetadataValue(integration.Metadata, "inventory_path")
	if inventoryPath == "" {
		inventoryPath = fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments", namespace)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+inventoryPath, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if tokenEnv := stringMetadataValue(integration.Metadata, "bearer_token_env"); tokenEnv != "" {
		if token := strings.TrimSpace(os.Getenv(tokenEnv)); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, nil, fmt.Errorf("kubernetes inventory request failed with status %d: %s", resp.StatusCode, truncateString(strings.TrimSpace(string(body)), 200))
	}
	var payload struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, nil, fmt.Errorf("unable to parse kubernetes workload inventory payload")
	}
	services, err := a.Store.ListServices(ctx, storage.ServiceQuery{OrganizationID: integration.OrganizationID, Limit: 500})
	if err != nil {
		return nil, nil, err
	}
	environments, err := a.Store.ListEnvironments(ctx, storage.EnvironmentQuery{OrganizationID: integration.OrganizationID, Limit: 500})
	if err != nil {
		return nil, nil, err
	}
	now := time.Now().UTC()
	resources := make([]types.DiscoveredResource, 0, len(payload.Items))
	seenResourceIDs := make(map[string]struct{}, len(payload.Items))
	unmapped := 0
	for _, rawItem := range payload.Items {
		deploymentStatus, err := parseKubernetesDeploymentResource(rawItem)
		if err != nil {
			continue
		}
		resource, err := a.buildKubernetesDiscoveredResource(ctx, integration, deploymentStatus, services, environments, now)
		if err != nil {
			return nil, nil, err
		}
		if err := a.Store.UpsertDiscoveredResource(ctx, resource); err != nil {
			return nil, nil, err
		}
		seenResourceIDs[resource.ID] = struct{}{}
		if resource.ServiceID == "" || resource.EnvironmentID == "" {
			unmapped++
		}
		resources = append(resources, resource)
	}
	missingDetails, err := a.markMissingDiscoveredResources(ctx, integration, "kubernetes_workload", seenResourceIDs, now, "kubernetes workload was not present in the latest inventory refresh", "missing")
	if err != nil {
		return nil, nil, err
	}
	return resources, compactDetailList([]string{
		fmt.Sprintf("inventory_path=%s", inventoryPath),
		fmt.Sprintf("discovered_workloads=%d", len(resources)),
		fmt.Sprintf("unmapped_workloads=%d", unmapped),
		missingDetails,
	}), nil
}

func (a *Application) buildKubernetesDiscoveredResource(ctx context.Context, integration types.Integration, deploymentStatus delivery.KubernetesDeploymentStatus, services []types.Service, environments []types.Environment, now time.Time) (types.DiscoveredResource, error) {
	externalID := strings.Trim(strings.Join([]string{deploymentStatus.Namespace, deploymentStatus.DeploymentName}, "/"), "/")
	resourceID := stableResourceID("discovery", integration.OrganizationID, integration.ID, "kubernetes_workload", externalID)
	resource := types.DiscoveredResource{
		BaseRecord: types.BaseRecord{
			ID:        resourceID,
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  types.Metadata{},
		},
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		ResourceType:   "kubernetes_workload",
		Provider:       "kubernetes",
		ExternalID:     externalID,
		Namespace:      deploymentStatus.Namespace,
		Name:           deploymentStatus.DeploymentName,
		Health:         "healthy",
		LastSeenAt:     &now,
	}
	setDiscoveredResourceProvenance(&resource, discoverySourceRuntimeObservation, now, types.Metadata{
		"namespace": resource.Namespace,
		"name":      resource.Name,
	})
	normalized := delivery.NormalizeKubernetesDeploymentStatus(deploymentStatus)
	resource.Health = normalized.BackendStatus
	resource.Summary = normalized.Summary
	resource.Metadata["backend_status"] = normalized.BackendStatus
	resource.Metadata["progress_percent"] = normalized.ProgressPercent
	resource.Metadata["current_step"] = normalized.CurrentStep
	for key, value := range normalized.Metadata {
		resource.Metadata[key] = value
	}
	if existing, err := a.Store.GetDiscoveredResource(ctx, resourceID); err == nil {
		resource = existing
		if resource.Metadata == nil {
			resource.Metadata = types.Metadata{}
		}
		resource.Provider = "kubernetes"
		resource.ExternalID = externalID
		resource.Namespace = deploymentStatus.Namespace
		resource.Name = deploymentStatus.DeploymentName
		resource.Health = normalized.BackendStatus
		resource.Summary = normalized.Summary
		resource.LastSeenAt = &now
		resource.UpdatedAt = now
		for key, value := range normalized.Metadata {
			resource.Metadata[key] = value
		}
	}
	delete(resource.Metadata, "missing_in_latest_sync")
	delete(resource.Metadata, "missing_observed_at")
	serviceID, projectID := guessServiceMapping(services, deploymentStatus.DeploymentName)
	environmentID, environmentProjectID := guessEnvironmentMapping(environments, deploymentStatus.Namespace)
	if resource.ServiceID == "" && serviceID != "" {
		resource.ServiceID = serviceID
		setMappingProvenance(resource.Metadata, "service", mappingSourceInferredName, now, "kubernetes workload name matched a catalog service")
	}
	if resource.EnvironmentID == "" && environmentID != "" {
		resource.EnvironmentID = environmentID
		setMappingProvenance(resource.Metadata, "environment", mappingSourceInferredNamespace, now, "kubernetes namespace matched a catalog environment")
	}
	if resource.ProjectID == "" {
		switch {
		case resource.ServiceID != "" && projectID != "":
			resource.ProjectID = projectID
			setMappingProvenance(resource.Metadata, "project", mappingSourceInferredName, now, "project inferred from matched service")
		case resource.EnvironmentID != "" && environmentProjectID != "":
			resource.ProjectID = environmentProjectID
			setMappingProvenance(resource.Metadata, "project", mappingSourceInferredNamespace, now, "project inferred from matched environment")
		}
	}
	if resource.ServiceID != "" && resource.EnvironmentID != "" {
		resource.Status = "mapped"
	} else if resource.Status == "" || resource.Status == "missing" || resource.Status == "mapped" {
		resource.Status = "candidate"
	}
	return a.applyDiscoveredResourceInferredOwnership(ctx, resource, now)
}

func parseKubernetesDeploymentResource(raw json.RawMessage) (delivery.KubernetesDeploymentStatus, error) {
	var payload struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Paused bool `json:"paused"`
		} `json:"spec"`
		Status struct {
			ObservedGeneration  int64                                 `json:"observedGeneration"`
			Replicas            int                                   `json:"replicas"`
			UpdatedReplicas     int                                   `json:"updatedReplicas"`
			AvailableReplicas   int                                   `json:"availableReplicas"`
			UnavailableReplicas int                                   `json:"unavailableReplicas"`
			Conditions          []delivery.KubernetesDeploymentCondition `json:"conditions"`
		} `json:"status"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return delivery.KubernetesDeploymentStatus{}, err
	}
	if strings.TrimSpace(payload.Metadata.Name) == "" {
		return delivery.KubernetesDeploymentStatus{}, fmt.Errorf("workload inventory item missing metadata.name")
	}
	return delivery.KubernetesDeploymentStatus{
		Namespace:           payload.Metadata.Namespace,
		DeploymentName:      payload.Metadata.Name,
		ObservedGeneration:  payload.Status.ObservedGeneration,
		Paused:              payload.Spec.Paused,
		Replicas:            payload.Status.Replicas,
		UpdatedReplicas:     payload.Status.UpdatedReplicas,
		AvailableReplicas:   payload.Status.AvailableReplicas,
		UnavailableReplicas: payload.Status.UnavailableReplicas,
		Conditions:          payload.Status.Conditions,
	}, nil
}

func (a *Application) persistPrometheusDiscoveredResources(ctx context.Context, integration types.Integration, collection verification.Collection) ([]types.DiscoveredResource, []string, error) {
	if len(collection.Snapshots) == 0 {
		return nil, []string{"signal_targets=0"}, nil
	}
	snapshot := collection.Snapshots[len(collection.Snapshots)-1]
	templates := decodePrometheusQueryTemplates(integration.Metadata["queries"])
	now := time.Now().UTC()
	resources := make([]types.DiscoveredResource, 0, len(snapshot.Signals))
	seenResourceIDs := make(map[string]struct{}, len(snapshot.Signals))
	mappedTargets := 0
	for _, signal := range snapshot.Signals {
		template := findPrometheusQueryTemplate(templates, signal.Name)
		resource, err := a.buildPrometheusDiscoveredResource(ctx, integration, template, signal, snapshot, now)
		if err != nil {
			return nil, nil, err
		}
		if err := a.Store.UpsertDiscoveredResource(ctx, resource); err != nil {
			return nil, nil, err
		}
		seenResourceIDs[resource.ID] = struct{}{}
		if resource.ServiceID != "" || resource.EnvironmentID != "" {
			mappedTargets++
		}
		resources = append(resources, resource)
	}
	missingDetails, err := a.markMissingDiscoveredResources(ctx, integration, "prometheus_signal_target", seenResourceIDs, now, "prometheus did not return this signal target in the latest collection window", "warning")
	if err != nil {
		return nil, nil, err
	}
	return resources, compactDetailList([]string{
		fmt.Sprintf("signal_targets=%d", len(resources)),
		fmt.Sprintf("mapped_signal_targets=%d", mappedTargets),
		fmt.Sprintf("window_start=%s", snapshot.WindowStart.Format(time.RFC3339)),
		fmt.Sprintf("window_end=%s", snapshot.WindowEnd.Format(time.RFC3339)),
		missingDetails,
	}), nil
}

func (a *Application) buildPrometheusDiscoveredResource(ctx context.Context, integration types.Integration, template prometheusQueryTemplate, signal types.SignalValue, snapshot types.SignalSnapshot, now time.Time) (types.DiscoveredResource, error) {
	name := valueOrDefault(template.ResourceName, signal.Name)
	externalID := signal.Name
	if template.ServiceID != "" || template.EnvironmentID != "" {
		externalID = strings.Trim(strings.Join([]string{template.ServiceID, template.EnvironmentID, signal.Name}, "/"), "/")
	}
	resourceID := stableResourceID("discovery", integration.OrganizationID, integration.ID, "prometheus_signal_target", externalID)
	resource := types.DiscoveredResource{
		BaseRecord: types.BaseRecord{
			ID:        resourceID,
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  types.Metadata{},
		},
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		ResourceType:   "prometheus_signal_target",
		Provider:       "prometheus",
		ExternalID:     externalID,
		Name:           name,
		Status:         "candidate",
		Health:         signal.Status,
		Summary:        fmt.Sprintf("%s=%0.3f%s", signal.Name, signal.Value, signal.Unit),
		LastSeenAt:     &now,
	}
	setDiscoveredResourceProvenance(&resource, discoverySourceRuntimeObservation, now, types.Metadata{
		"query_name": signal.Name,
	})
	if existing, err := a.Store.GetDiscoveredResource(ctx, resourceID); err == nil {
		resource = existing
		if resource.Metadata == nil {
			resource.Metadata = types.Metadata{}
		}
		resource.Provider = "prometheus"
		resource.ExternalID = externalID
		resource.Name = name
		resource.Health = signal.Status
		resource.Summary = fmt.Sprintf("%s=%0.3f%s", signal.Name, signal.Value, signal.Unit)
		resource.LastSeenAt = &now
		resource.UpdatedAt = now
	}
	delete(resource.Metadata, "missing_in_latest_sync")
	delete(resource.Metadata, "missing_observed_at")
	if resource.ServiceID == "" && template.ServiceID != "" {
		service, err := a.Store.GetService(ctx, template.ServiceID)
		if err == nil && service.OrganizationID == integration.OrganizationID {
			resource.ServiceID = service.ID
			resource.ProjectID = service.ProjectID
			setMappingProvenance(resource.Metadata, "service", mappingSourceQueryTemplate, now, "prometheus query template pinned this target to a service")
			setMappingProvenance(resource.Metadata, "project", mappingSourceQueryTemplate, now, "project inferred from prometheus query template service binding")
		}
	}
	if resource.EnvironmentID == "" && template.EnvironmentID != "" {
		environment, err := a.Store.GetEnvironment(ctx, template.EnvironmentID)
		if err == nil && environment.OrganizationID == integration.OrganizationID {
			resource.EnvironmentID = environment.ID
			if resource.ProjectID == "" {
				resource.ProjectID = environment.ProjectID
			}
			setMappingProvenance(resource.Metadata, "environment", mappingSourceQueryTemplate, now, "prometheus query template pinned this target to an environment")
			if resource.ProjectID == environment.ProjectID {
				setMappingProvenance(resource.Metadata, "project", mappingSourceQueryTemplate, now, "project inferred from prometheus query template environment binding")
			}
		}
	}
	if resource.ServiceID != "" || resource.EnvironmentID != "" {
		resource.Status = "mapped"
	} else if resource.Status == "" || resource.Status == "missing" || resource.Status == "mapped" {
		resource.Status = "candidate"
	}
	resource.Metadata["signal_name"] = signal.Name
	resource.Metadata["signal_value"] = signal.Value
	resource.Metadata["signal_unit"] = signal.Unit
	resource.Metadata["signal_threshold"] = signal.Threshold
	resource.Metadata["signal_comparator"] = signal.Comparator
	resource.Metadata["window_start"] = snapshot.WindowStart.Format(time.RFC3339)
	resource.Metadata["window_end"] = snapshot.WindowEnd.Format(time.RFC3339)
	resource.Metadata["snapshot_health"] = snapshot.Health
	return a.applyDiscoveredResourceInferredOwnership(ctx, resource, now)
}

func guessServiceMapping(services []types.Service, workloadName string) (string, string) {
	candidate := normalizeMappingToken(workloadName)
	for _, service := range services {
		if candidate == normalizeMappingToken(service.Slug) || candidate == normalizeMappingToken(service.Name) {
			return service.ID, service.ProjectID
		}
	}
	return "", ""
}

func guessEnvironmentMapping(environments []types.Environment, namespace string) (string, string) {
	candidate := normalizeMappingToken(namespace)
	for _, environment := range environments {
		if candidate == normalizeMappingToken(environment.Slug) || candidate == normalizeMappingToken(environment.Name) {
			return environment.ID, environment.ProjectID
		}
	}
	return "", ""
}

func normalizeMappingToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacements := []string{"deployment", "service", "svc", "api"}
	for _, replacement := range replacements {
		value = strings.TrimSuffix(value, "-"+replacement)
	}
	return strings.Trim(value, " -_/")
}

func findPrometheusQueryTemplate(templates []prometheusQueryTemplate, name string) prometheusQueryTemplate {
	for _, template := range templates {
		if strings.EqualFold(strings.TrimSpace(template.Name), strings.TrimSpace(name)) {
			return template
		}
	}
	return prometheusQueryTemplate{Name: name}
}

func (a *Application) markMissingDiscoveredResources(ctx context.Context, integration types.Integration, resourceType string, seenResourceIDs map[string]struct{}, now time.Time, summary string, health string) (string, error) {
	existingResources, err := a.Store.ListDiscoveredResources(ctx, storage.DiscoveredResourceQuery{
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		ResourceType:   resourceType,
		Limit:          1000,
	})
	if err != nil {
		return "", err
	}
	missingCount := 0
	for _, resource := range existingResources {
		if _, ok := seenResourceIDs[resource.ID]; ok {
			continue
		}
		if resource.Metadata == nil {
			resource.Metadata = types.Metadata{}
		}
		resource.Status = "missing"
		resource.Health = health
		resource.Summary = summary
		resource.Metadata["missing_in_latest_sync"] = true
		resource.Metadata["missing_observed_at"] = now.Format(time.RFC3339)
		resource.UpdatedAt = now
		if err := a.Store.UpdateDiscoveredResource(ctx, resource); err != nil {
			return "", err
		}
		missingCount++
	}
	return fmt.Sprintf("missing_%s=%d", pluralizeMissingResource(resourceType), missingCount), nil
}

func pluralizeMissingResource(resourceType string) string {
	switch resourceType {
	case "kubernetes_workload":
		return "workloads"
	case "prometheus_signal_target":
		return "signal_targets"
	default:
		return "resources"
	}
}
