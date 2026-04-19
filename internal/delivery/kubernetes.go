package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type KubernetesDeploymentCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type KubernetesDeploymentStatus struct {
	Namespace           string                          `json:"namespace"`
	DeploymentName      string                          `json:"deployment_name"`
	ObservedGeneration  int64                           `json:"observed_generation,omitempty"`
	Paused              bool                            `json:"paused,omitempty"`
	Replicas            int                             `json:"replicas,omitempty"`
	UpdatedReplicas     int                             `json:"updated_replicas,omitempty"`
	AvailableReplicas   int                             `json:"available_replicas,omitempty"`
	UnavailableReplicas int                             `json:"unavailable_replicas,omitempty"`
	Conditions          []KubernetesDeploymentCondition `json:"conditions,omitempty"`
}

type kubernetesProviderConfig struct {
	APIBaseURL         string
	StatusPath         string
	SubmitPath         string
	PausePath          string
	ResumePath         string
	RollbackPath       string
	Namespace          string
	DeploymentName     string
	ContainerName      string
	RollbackTargetImage string
	BearerTokenEnv     string
}

type KubernetesDeploymentProvider struct {
	httpClient *http.Client
	now        func() time.Time
}

func NewKubernetesDeploymentProvider() Provider {
	return KubernetesDeploymentProvider{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (p KubernetesDeploymentProvider) Kind() string {
	return "kubernetes"
}

func (p KubernetesDeploymentProvider) Submit(ctx context.Context, runtime types.RolloutExecutionRuntimeContext) (SyncResult, error) {
	cfg, err := loadKubernetesProviderConfig(runtime)
	if err != nil {
		return SyncResult{}, err
	}
	if cfg.SubmitPath != "" {
		return p.invokeAction(ctx, cfg, http.MethodPost, cfg.SubmitPath, map[string]any{
			"rollout_execution_id": runtime.Execution.ID,
			"change_set_id":        runtime.Execution.ChangeSetID,
			"service_id":           runtime.Execution.ServiceID,
			"environment_id":       runtime.Execution.EnvironmentID,
		}, "submit")
	}

	result, err := p.Sync(ctx, runtime)
	if err != nil {
		return SyncResult{}, err
	}
	if result.BackendExecutionID == "" {
		result.BackendExecutionID = kubernetesExecutionID(cfg.Namespace, cfg.DeploymentName)
	}
	result.Summary = "attached rollout execution to kubernetes deployment target"
	result.Explanation = compactStrings(append(result.Explanation, "no explicit submit endpoint configured; control plane attached to the configured deployment target"))
	return result, nil
}

func (p KubernetesDeploymentProvider) Sync(ctx context.Context, runtime types.RolloutExecutionRuntimeContext) (SyncResult, error) {
	cfg, err := loadKubernetesProviderConfig(runtime)
	if err != nil {
		return SyncResult{}, err
	}
	statusPath := cfg.StatusPath
	if statusPath == "" {
		statusPath = defaultKubernetesStatusPath(cfg.Namespace, cfg.DeploymentName)
	}
	body, statusCode, err := p.doRequest(ctx, cfg, http.MethodGet, statusPath, nil)
	if err != nil {
		return SyncResult{}, err
	}
	result, err := normalizeKubernetesHTTPResponse(body)
	if err != nil {
		return SyncResult{}, &ProviderError{Operation: "sync", StatusCode: statusCode, Temporary: false, Err: err}
	}
	result.BackendType = "kubernetes"
	result.BackendExecutionID = kubernetesExecutionID(cfg.Namespace, cfg.DeploymentName)
	if result.LastUpdatedAt.IsZero() {
		result.LastUpdatedAt = p.now()
	}
	return result, nil
}

func (p KubernetesDeploymentProvider) Pause(ctx context.Context, runtime types.RolloutExecutionRuntimeContext, reason string) (SyncResult, error) {
	cfg, err := loadKubernetesProviderConfig(runtime)
	if err != nil {
		return SyncResult{}, err
	}
	if cfg.PausePath != "" {
		return p.invokeAction(ctx, cfg, http.MethodPost, cfg.PausePath, map[string]any{"reason": reason}, "pause")
	}
	statusPath := cfg.StatusPath
	if statusPath == "" {
		statusPath = defaultKubernetesStatusPath(cfg.Namespace, cfg.DeploymentName)
	}
	return p.invokeAction(ctx, cfg, http.MethodPatch, statusPath, map[string]any{
		"spec": map[string]any{"paused": true},
	}, "pause")
}

func (p KubernetesDeploymentProvider) Resume(ctx context.Context, runtime types.RolloutExecutionRuntimeContext, reason string) (SyncResult, error) {
	cfg, err := loadKubernetesProviderConfig(runtime)
	if err != nil {
		return SyncResult{}, err
	}
	if cfg.ResumePath != "" {
		return p.invokeAction(ctx, cfg, http.MethodPost, cfg.ResumePath, map[string]any{"reason": reason}, "resume")
	}
	statusPath := cfg.StatusPath
	if statusPath == "" {
		statusPath = defaultKubernetesStatusPath(cfg.Namespace, cfg.DeploymentName)
	}
	return p.invokeAction(ctx, cfg, http.MethodPatch, statusPath, map[string]any{
		"spec": map[string]any{"paused": false},
	}, "resume")
}

func (p KubernetesDeploymentProvider) Rollback(ctx context.Context, runtime types.RolloutExecutionRuntimeContext, reason string) (SyncResult, error) {
	cfg, err := loadKubernetesProviderConfig(runtime)
	if err != nil {
		return SyncResult{}, err
	}
	if cfg.RollbackPath != "" {
		return p.invokeAction(ctx, cfg, http.MethodPost, cfg.RollbackPath, map[string]any{
			"reason":               reason,
			"rollback_target_image": cfg.RollbackTargetImage,
		}, "rollback")
	}
	if cfg.RollbackTargetImage == "" || cfg.ContainerName == "" {
		return SyncResult{}, fmt.Errorf("%w: kubernetes rollback requires rollback_path or container_name + rollback_target_image metadata", ErrProviderUnavailable)
	}
	statusPath := cfg.StatusPath
	if statusPath == "" {
		statusPath = defaultKubernetesStatusPath(cfg.Namespace, cfg.DeploymentName)
	}
	return p.invokeAction(ctx, cfg, http.MethodPatch, statusPath, map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []map[string]any{{
						"name":  cfg.ContainerName,
						"image": cfg.RollbackTargetImage,
					}},
				},
			},
		},
	}, "rollback")
}

func (p KubernetesDeploymentProvider) invokeAction(ctx context.Context, cfg kubernetesProviderConfig, method, path string, payload any, operation string) (SyncResult, error) {
	body, statusCode, err := p.doRequest(ctx, cfg, method, path, payload)
	if err != nil {
		return SyncResult{}, err
	}
	if len(body) > 0 {
		if result, ok := tryDecodeSyncResult(body); ok {
			result.BackendType = "kubernetes"
			if result.BackendExecutionID == "" {
				result.BackendExecutionID = kubernetesExecutionID(cfg.Namespace, cfg.DeploymentName)
			}
			if result.LastUpdatedAt.IsZero() {
				result.LastUpdatedAt = p.now()
			}
			return result, nil
		}
		if result, err := normalizeKubernetesHTTPResponse(body); err == nil {
			result.BackendType = "kubernetes"
			result.BackendExecutionID = kubernetesExecutionID(cfg.Namespace, cfg.DeploymentName)
			if result.LastUpdatedAt.IsZero() {
				result.LastUpdatedAt = p.now()
			}
			return result, nil
		}
	}
	return SyncResult{
		BackendType:        "kubernetes",
		BackendExecutionID: kubernetesExecutionID(cfg.Namespace, cfg.DeploymentName),
		BackendStatus:      actionBackendStatus(operation),
		ProgressPercent:    100,
		CurrentStep:        operation,
		Summary:            fmt.Sprintf("kubernetes %s accepted", operation),
		Explanation:        []string{"provider action completed without a structured rollout status payload"},
		LastUpdatedAt:      p.now(),
		Metadata: types.Metadata{
			"http_status": statusCode,
		},
	}, nil
}

func (p KubernetesDeploymentProvider) doRequest(ctx context.Context, cfg kubernetesProviderConfig, method, path string, payload any) ([]byte, int, error) {
	if strings.TrimSpace(cfg.APIBaseURL) == "" {
		return nil, 0, fmt.Errorf("%w: kubernetes api_base_url is required", ErrProviderUnavailable)
	}
	endpoint := strings.TrimRight(cfg.APIBaseURL, "/") + path
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, &ProviderError{Operation: method + " " + path, Temporary: false, Err: err}
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, 0, &ProviderError{Operation: method + " " + path, Temporary: false, Err: err}
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/merge-patch+json")
		if method == http.MethodPost {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	if cfg.BearerTokenEnv != "" {
		if token := strings.TrimSpace(os.Getenv(cfg.BearerTokenEnv)); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, &ProviderError{Operation: method + " " + path, Temporary: true, Err: err}
	}
	defer resp.Body.Close()
	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, &ProviderError{Operation: method + " " + path, StatusCode: resp.StatusCode, Temporary: true, Err: readErr}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		temporary := resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests
		return nil, resp.StatusCode, &ProviderError{Operation: method + " " + path, StatusCode: resp.StatusCode, Temporary: temporary, Err: fmt.Errorf("%s", strings.TrimSpace(string(responseBody)))}
	}
	return responseBody, resp.StatusCode, nil
}

func loadKubernetesProviderConfig(runtime types.RolloutExecutionRuntimeContext) (kubernetesProviderConfig, error) {
	metadata := types.Metadata{}
	mergeMetadata(metadata, runtime.BackendIntegration)
	for key, value := range runtime.Execution.Metadata {
		metadata[key] = value
	}
	cfg := kubernetesProviderConfig{
		APIBaseURL:          stringMetadataValue(metadata, "api_base_url"),
		StatusPath:          stringMetadataValue(metadata, "status_path"),
		SubmitPath:          stringMetadataValue(metadata, "submit_path"),
		PausePath:           stringMetadataValue(metadata, "pause_path"),
		ResumePath:          stringMetadataValue(metadata, "resume_path"),
		RollbackPath:        stringMetadataValue(metadata, "rollback_path"),
		Namespace:           valueOrMetadata(stringMetadataValue(metadata, "namespace"), runtime.Environment.Slug),
		DeploymentName:      valueOrMetadata(stringMetadataValue(metadata, "deployment_name"), runtime.Service.Slug),
		ContainerName:       stringMetadataValue(metadata, "container_name"),
		RollbackTargetImage: stringMetadataValue(metadata, "rollback_target_image"),
		BearerTokenEnv:      stringMetadataValue(metadata, "bearer_token_env"),
	}
	if cfg.APIBaseURL == "" {
		return cfg, fmt.Errorf("%w: kubernetes integration metadata must include api_base_url", ErrProviderUnavailable)
	}
	if cfg.Namespace == "" || cfg.DeploymentName == "" {
		return cfg, fmt.Errorf("%w: kubernetes namespace and deployment_name are required", ErrProviderUnavailable)
	}
	return cfg, nil
}

func normalizeKubernetesHTTPResponse(body []byte) (SyncResult, error) {
	if result, ok := tryDecodeSyncResult(body); ok {
		return result, nil
	}
	status := KubernetesDeploymentStatus{}
	if err := json.Unmarshal(body, &status); err == nil && status.DeploymentName != "" {
		return NormalizeKubernetesDeploymentStatus(status), nil
	}

	var deployment struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Paused bool `json:"paused"`
		} `json:"spec"`
		Status struct {
			ObservedGeneration  int64                           `json:"observedGeneration"`
			Replicas            int                             `json:"replicas"`
			UpdatedReplicas     int                             `json:"updatedReplicas"`
			AvailableReplicas   int                             `json:"availableReplicas"`
			UnavailableReplicas int                             `json:"unavailableReplicas"`
			Conditions          []KubernetesDeploymentCondition `json:"conditions"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &deployment); err != nil {
		return SyncResult{}, fmt.Errorf("unable to parse kubernetes deployment status payload")
	}
	if deployment.Metadata.Name == "" {
		return SyncResult{}, fmt.Errorf("kubernetes deployment payload did not include metadata.name")
	}
	return NormalizeKubernetesDeploymentStatus(KubernetesDeploymentStatus{
		Namespace:           deployment.Metadata.Namespace,
		DeploymentName:      deployment.Metadata.Name,
		ObservedGeneration:  deployment.Status.ObservedGeneration,
		Paused:              deployment.Spec.Paused,
		Replicas:            deployment.Status.Replicas,
		UpdatedReplicas:     deployment.Status.UpdatedReplicas,
		AvailableReplicas:   deployment.Status.AvailableReplicas,
		UnavailableReplicas: deployment.Status.UnavailableReplicas,
		Conditions:          deployment.Status.Conditions,
	}), nil
}

func NormalizeKubernetesDeploymentStatus(status KubernetesDeploymentStatus) SyncResult {
	now := time.Now().UTC()
	progress := 0
	if status.Replicas > 0 {
		progress = (status.AvailableReplicas * 100) / status.Replicas
	}
	result := SyncResult{
		BackendType:     "kubernetes",
		BackendStatus:   "progressing",
		ProgressPercent: progress,
		CurrentStep:     "deployment",
		Summary:         "kubernetes deployment is progressing",
		LastUpdatedAt:   now,
		Metadata: types.Metadata{
			"namespace":           status.Namespace,
			"deployment_name":     status.DeploymentName,
			"replicas":            status.Replicas,
			"updated_replicas":    status.UpdatedReplicas,
			"available_replicas":  status.AvailableReplicas,
			"unavailable_replicas": status.UnavailableReplicas,
			"paused":              status.Paused,
		},
	}
	if status.Paused {
		result.BackendStatus = "paused"
		result.CurrentStep = "paused"
		result.Summary = "kubernetes deployment is paused"
		result.Explanation = []string{"deployment spec indicates the rollout is paused"}
		return result
	}
	for _, condition := range status.Conditions {
		conditionType := strings.ToLower(condition.Type)
		conditionStatus := strings.ToLower(condition.Status)
		if conditionType == "progressing" && conditionStatus == "false" {
			result.BackendStatus = "failed"
			result.CurrentStep = "deployment_failed"
			result.Summary = fallbackStatus(condition.Message, "kubernetes deployment reported a failed progressing condition")
			result.Explanation = compactStrings([]string{condition.Reason, condition.Message})
			result.ProgressPercent = maxInt(result.ProgressPercent, 100)
			return result
		}
		if conditionType == "available" && conditionStatus == "true" && status.Replicas > 0 && status.AvailableReplicas >= status.Replicas && status.UpdatedReplicas >= status.Replicas {
			result.BackendStatus = "awaiting_verification"
			result.ProgressPercent = 100
			result.CurrentStep = "available"
			result.Summary = "kubernetes deployment is fully available and ready for verification"
			result.Explanation = []string{"updated and available replica counts match the desired replica count"}
			return result
		}
	}
	if status.Replicas > 0 && status.AvailableReplicas >= status.Replicas && status.UpdatedReplicas >= status.Replicas {
		result.BackendStatus = "awaiting_verification"
		result.ProgressPercent = 100
		result.CurrentStep = "available"
		result.Summary = "kubernetes deployment is fully available and ready for verification"
		result.Explanation = []string{"updated and available replica counts match the desired replica count"}
		return result
	}
	if status.UnavailableReplicas > 0 {
		result.Explanation = []string{"kubernetes deployment still has unavailable replicas"}
	}
	return result
}

func tryDecodeSyncResult(body []byte) (SyncResult, bool) {
	result := SyncResult{}
	if err := json.Unmarshal(body, &result); err != nil {
		return SyncResult{}, false
	}
	return result, result.BackendStatus != ""
}

func defaultKubernetesStatusPath(namespace, deployment string) string {
	return fmt.Sprintf("/apis/apps/v1/namespaces/%s/deployments/%s", namespace, deployment)
}

func kubernetesExecutionID(namespace, deployment string) string {
	return strings.Trim(strings.Join([]string{namespace, deployment}, "/"), "/")
}

func actionBackendStatus(operation string) string {
	switch operation {
	case "rollback":
		return "rollback_requested"
	case "pause":
		return "paused"
	case "resume":
		return "progressing"
	case "submit":
		return "submitted"
	default:
		return "progressing"
	}
}

func mergeMetadata(target types.Metadata, integration *types.Integration) {
	if integration == nil || integration.Metadata == nil {
		return
	}
	for key, value := range integration.Metadata {
		target[key] = value
	}
}

func stringMetadataValue(metadata types.Metadata, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func valueOrMetadata(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}
