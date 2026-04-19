package delivery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestKubernetesProviderSyncAndPauseAgainstHTTPServer(t *testing.T) {
	t.Parallel()

	var patchedPaused bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": patchedPaused},
				"status": map[string]any{
					"replicas":            3,
					"updatedReplicas":     3,
					"availableReplicas":   3,
					"unavailableReplicas": 0,
					"conditions": []map[string]any{{
						"type":   "Available",
						"status": "True",
					}},
				},
			})
		case r.Method == http.MethodPatch:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode patch payload: %v", err)
			}
			spec := payload["spec"].(map[string]any)
			if pausedValue, ok := spec["paused"].(bool); ok {
				patchedPaused = pausedValue
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": patchedPaused},
				"status": map[string]any{
					"replicas":            3,
					"updatedReplicas":     3,
					"availableReplicas":   3,
					"unavailableReplicas": 0,
				},
			})
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	now := time.Now().UTC()
	provider := NewKubernetesDeploymentProvider()
	runtime := types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord: types.BaseRecord{ID: "exec_123", CreatedAt: now, UpdatedAt: now},
			Status:     "in_progress",
		},
		Service:     types.Service{Slug: "checkout"},
		Environment: types.Environment{Slug: "prod"},
		BackendIntegration: &types.Integration{
			BaseRecord: types.BaseRecord{
				Metadata: types.Metadata{
					"api_base_url":    server.URL,
					"namespace":       "prod",
					"deployment_name": "checkout",
				},
			},
		},
	}

	syncResult, err := provider.Sync(t.Context(), runtime)
	if err != nil {
		t.Fatal(err)
	}
	if syncResult.BackendStatus != "awaiting_verification" {
		t.Fatalf("expected awaiting_verification, got %s", syncResult.BackendStatus)
	}

	pauseResult, err := provider.Pause(t.Context(), runtime, "test pause")
	if err != nil {
		t.Fatal(err)
	}
	if pauseResult.BackendStatus != "paused" {
		t.Fatalf("expected paused status, got %s", pauseResult.BackendStatus)
	}
}

func TestNormalizeKubernetesDeploymentStatusClassifiesProviderFailure(t *testing.T) {
	t.Parallel()

	result := NormalizeKubernetesDeploymentStatus(KubernetesDeploymentStatus{
		Namespace:           "prod",
		DeploymentName:      "checkout",
		Replicas:            3,
		UpdatedReplicas:     1,
		AvailableReplicas:   0,
		UnavailableReplicas: 3,
		Conditions: []KubernetesDeploymentCondition{{
			Type:    "Progressing",
			Status:  "False",
			Reason:  "ProgressDeadlineExceeded",
			Message: "deployment exceeded its progress deadline",
		}},
	})

	if result.BackendStatus != "failed" {
		t.Fatalf("expected failed backend status, got %+v", result)
	}
	if result.CurrentStep != "deployment_failed" {
		t.Fatalf("expected failed current step, got %+v", result)
	}
	if result.ProgressPercent < 100 {
		t.Fatalf("expected failed result to clamp progress, got %+v", result)
	}
}

func TestKubernetesProviderSyncReflectsChangingUpstreamState(t *testing.T) {
	t.Parallel()

	var stage atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch stage.Load() {
		case 0:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": false},
				"status": map[string]any{
					"replicas":            4,
					"updatedReplicas":     2,
					"availableReplicas":   1,
					"unavailableReplicas": 3,
					"conditions":          []map[string]any{{"type": "Progressing", "status": "True"}},
				},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": false},
				"status": map[string]any{
					"replicas":            4,
					"updatedReplicas":     4,
					"availableReplicas":   4,
					"unavailableReplicas": 0,
					"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
				},
			})
		}
	}))
	defer server.Close()

	now := time.Now().UTC()
	provider := NewKubernetesDeploymentProvider()
	runtime := types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{BaseRecord: types.BaseRecord{ID: "exec_stage", CreatedAt: now, UpdatedAt: now}, Status: "in_progress"},
		Service:   types.Service{Slug: "checkout"},
		Environment: types.Environment{
			Slug: "prod",
		},
		BackendIntegration: &types.Integration{
			BaseRecord: types.BaseRecord{
				Metadata: types.Metadata{
					"api_base_url":    server.URL,
					"namespace":       "prod",
					"deployment_name": "checkout",
				},
			},
		},
	}

	first, err := provider.Sync(t.Context(), runtime)
	if err != nil {
		t.Fatal(err)
	}
	if first.BackendStatus == "awaiting_verification" {
		t.Fatalf("expected first sync to reflect in-progress rollout, got %+v", first)
	}

	stage.Store(1)
	second, err := provider.Sync(t.Context(), runtime)
	if err != nil {
		t.Fatal(err)
	}
	if second.BackendStatus != "awaiting_verification" {
		t.Fatalf("expected second sync to reflect healthy rollout, got %+v", second)
	}
	if second.ProgressPercent <= first.ProgressPercent {
		t.Fatalf("expected rollout progress to advance, got first=%+v second=%+v", first, second)
	}
}

func TestKubernetesProviderClassifiesRemoteFailures(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"upstream outage"}`))
	}))
	defer server.Close()

	now := time.Now().UTC()
	provider := NewKubernetesDeploymentProvider()
	_, err := provider.Sync(t.Context(), types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{BaseRecord: types.BaseRecord{ID: "exec_err", CreatedAt: now, UpdatedAt: now}},
		Service:   types.Service{Slug: "checkout"},
		Environment: types.Environment{
			Slug: "prod",
		},
		BackendIntegration: &types.Integration{
			BaseRecord: types.BaseRecord{
				Metadata: types.Metadata{
					"api_base_url":    server.URL,
					"namespace":       "prod",
					"deployment_name": "checkout",
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected provider error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected remote status in error, got %v", err)
	}
}

func TestKubernetesProviderRollbackUsesConfiguredImagePatch(t *testing.T) {
	t.Parallel()

	var patchedImage string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected patch rollback request, got %s", r.Method)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode rollback payload: %v", err)
		}
		spec := payload["spec"].(map[string]any)
		template := spec["template"].(map[string]any)
		templateSpec := template["spec"].(map[string]any)
		containers := templateSpec["containers"].([]any)
		container := containers[0].(map[string]any)
		patchedImage = container["image"].(string)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
			"status": map[string]any{
				"replicas":            3,
				"updatedReplicas":     3,
				"availableReplicas":   3,
				"unavailableReplicas": 0,
				"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
			},
		})
	}))
	defer server.Close()

	now := time.Now().UTC()
	provider := NewKubernetesDeploymentProvider()
	runtime := types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{BaseRecord: types.BaseRecord{ID: "exec_rb", CreatedAt: now, UpdatedAt: now}},
		Service:   types.Service{Slug: "checkout"},
		Environment: types.Environment{
			Slug: "prod",
		},
		BackendIntegration: &types.Integration{
			BaseRecord: types.BaseRecord{
				Metadata: types.Metadata{
					"api_base_url":          server.URL,
					"namespace":             "prod",
					"deployment_name":       "checkout",
					"container_name":        "checkout",
					"rollback_target_image": "registry.example.com/checkout:rollback",
				},
			},
		},
	}

	result, err := provider.Rollback(t.Context(), runtime, "rollback test")
	if err != nil {
		t.Fatal(err)
	}
	if patchedImage != "registry.example.com/checkout:rollback" {
		t.Fatalf("expected rollback target image to be patched, got %q", patchedImage)
	}
	if result.BackendExecutionID != "prod/checkout" {
		t.Fatalf("expected normalized backend execution id, got %+v", result)
	}
}
