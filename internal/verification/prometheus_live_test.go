package verification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestPrometheusProviderCollectsSnapshotFromHTTPServer(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query_range" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		query := r.URL.Query().Get("query")
		value := "0.3"
		if query == "error_rate" {
			value = "2.8"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result": []map[string]any{{
					"values": [][]any{
						{float64(time.Now().Unix() - 60), "0.1"},
						{float64(time.Now().Unix()), value},
					},
				}},
			},
		})
	}))
	defer server.Close()

	now := time.Now().UTC()
	provider := NewPrometheusProvider()
	collection, err := provider.Collect(t.Context(), types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord: types.BaseRecord{ID: "exec_123", CreatedAt: now, UpdatedAt: now},
			Status:     "in_progress",
		},
		SignalIntegration: &types.Integration{
			BaseRecord: types.BaseRecord{
				ID: "int_prom",
				Metadata: types.Metadata{
					"api_base_url": server.URL,
					"queries": []map[string]any{
						{"name": "latency_p95_ms", "category": "technical", "query": "latency", "threshold": 500, "comparator": ">", "unit": "ms"},
						{"name": "error_rate", "category": "technical", "query": "error_rate", "threshold": 1, "comparator": ">", "unit": "%", "severity": "critical"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(collection.Snapshots) != 1 {
		t.Fatalf("expected one snapshot, got %d", len(collection.Snapshots))
	}
	if collection.Snapshots[0].Health != "critical" {
		t.Fatalf("expected critical health, got %s", collection.Snapshots[0].Health)
	}
	if len(collection.Snapshots[0].Signals) != 2 {
		t.Fatalf("expected two signals, got %d", len(collection.Snapshots[0].Signals))
	}
}

func TestDecodePrometheusValueRejectsMissingSamples(t *testing.T) {
	t.Parallel()

	_, err := decodePrometheusValue([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"values":[[123]]}]}}`))
	if err == nil {
		t.Fatal("expected missing sample value error")
	}
}

func TestPrometheusProviderTreatsEmptyResultsAsZeroValue(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result":     []map[string]any{},
			},
		})
	}))
	defer server.Close()

	now := time.Now().UTC()
	provider := NewPrometheusProvider()
	collection, err := provider.Collect(t.Context(), types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord: types.BaseRecord{ID: "exec_empty", CreatedAt: now, UpdatedAt: now},
			Status:     "in_progress",
		},
		SignalIntegration: &types.Integration{
			BaseRecord: types.BaseRecord{
				ID: "int_prom_empty",
				Metadata: types.Metadata{
					"api_base_url": server.URL,
					"queries": []map[string]any{
						{"name": "latency_p95_ms", "category": "technical", "query": "latency", "threshold": 500, "comparator": ">", "unit": "ms"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(collection.Snapshots) != 1 || len(collection.Snapshots[0].Signals) != 1 {
		t.Fatalf("expected one zero-valued signal snapshot, got %+v", collection)
	}
	if collection.Snapshots[0].Signals[0].Value != 0 {
		t.Fatalf("expected empty result to normalize to zero, got %+v", collection.Snapshots[0].Signals[0])
	}
	if collection.Snapshots[0].Signals[0].Status != "warning" {
		t.Fatalf("expected empty result to surface warning status, got %+v", collection.Snapshots[0].Signals[0])
	}
	if collection.Snapshots[0].Health != "warning" {
		t.Fatalf("expected empty result to degrade collection health to warning, got %+v", collection.Snapshots[0])
	}
}

func TestPrometheusProviderCollectsChangingSamplesAcrossRepeatedRuns(t *testing.T) {
	t.Parallel()

	var stage atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		value := "0.2"
		if stage.Load() > 0 {
			value = "3.4"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result": []map[string]any{{
					"values": [][]any{
						{float64(time.Now().Unix() - 60), "0.1"},
						{float64(time.Now().Unix()), value},
					},
				}},
			},
		})
	}))
	defer server.Close()

	now := time.Now().UTC()
	provider := NewPrometheusProvider()
	runtime := types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord: types.BaseRecord{ID: "exec_repeat", CreatedAt: now, UpdatedAt: now},
			Status:     "in_progress",
		},
		SignalIntegration: &types.Integration{
			BaseRecord: types.BaseRecord{
				ID: "int_prom_repeat",
				Metadata: types.Metadata{
					"api_base_url": server.URL,
					"queries": []map[string]any{
						{"name": "error_rate", "category": "technical", "query": "error_rate", "threshold": 1, "comparator": ">", "unit": "%"},
					},
				},
			},
		},
	}

	first, err := provider.Collect(t.Context(), runtime)
	if err != nil {
		t.Fatal(err)
	}
	if first.Snapshots[0].Health != "healthy" {
		t.Fatalf("expected first collection to be healthy, got %+v", first.Snapshots[0])
	}

	stage.Store(1)
	second, err := provider.Collect(t.Context(), runtime)
	if err != nil {
		t.Fatal(err)
	}
	if second.Snapshots[0].Health != "degraded" {
		t.Fatalf("expected second collection to degrade, got %+v", second.Snapshots[0])
	}
}

func TestPrometheusProviderReturnsStructuredServerErrors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`service unavailable`))
	}))
	defer server.Close()

	now := time.Now().UTC()
	provider := NewPrometheusProvider()
	_, err := provider.Collect(t.Context(), types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord: types.BaseRecord{ID: "exec_err", CreatedAt: now, UpdatedAt: now},
			Status:     "in_progress",
		},
		SignalIntegration: &types.Integration{
			BaseRecord: types.BaseRecord{
				ID: "int_prom_err",
				Metadata: types.Metadata{
					"api_base_url": server.URL,
					"queries": []map[string]any{
						{"name": "latency_p95_ms", "category": "technical", "query": "latency", "threshold": 500, "comparator": ">", "unit": "ms"},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected structured signal provider error")
	}
	if !strings.Contains(err.Error(), "service unavailable") {
		t.Fatalf("expected upstream failure details, got %v", err)
	}
}

func TestPrometheusProviderUsesConfiguredWindowStepAndBearerToken(t *testing.T) {
	t.Setenv("CCP_PROM_TOKEN_TEST", "prom-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer prom-secret" {
			t.Fatalf("expected bearer token header, got %q", got)
		}
		if got := r.URL.Query().Get("step"); got != "15" {
			t.Fatalf("expected configured step seconds, got %q", got)
		}
		start, err := strconv.ParseFloat(r.URL.Query().Get("start"), 64)
		if err != nil {
			t.Fatalf("parse start: %v", err)
		}
		end, err := strconv.ParseFloat(r.URL.Query().Get("end"), 64)
		if err != nil {
			t.Fatalf("parse end: %v", err)
		}
		if int(end-start) != 120 {
			t.Fatalf("expected configured 120-second window, got %f", end-start)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "matrix",
				"result": []map[string]any{{
					"values": [][]any{
						{end - 15, "0.2"},
						{end, "0.4"},
					},
				}},
			},
		})
	}))
	defer server.Close()

	now := time.Unix(1_700_000_000, 0).UTC()
	provider := PrometheusProvider{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		now:        func() time.Time { return now },
	}
	collection, err := provider.Collect(t.Context(), types.RolloutExecutionRuntimeContext{
		Execution: types.RolloutExecution{
			BaseRecord: types.BaseRecord{ID: "exec_window", CreatedAt: now, UpdatedAt: now},
			Status:     "in_progress",
		},
		SignalIntegration: &types.Integration{
			BaseRecord: types.BaseRecord{
				ID: "int_prom_window",
				Metadata: types.Metadata{
					"api_base_url":     server.URL,
					"window_seconds":   "120",
					"step_seconds":     "15",
					"bearer_token_env": "CCP_PROM_TOKEN_TEST",
					"queries": []map[string]any{
						{"name": "latency_p95_ms", "category": "technical", "query": "latency", "threshold": 500, "comparator": ">", "unit": "ms"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if collection.Snapshots[0].WindowEnd.Sub(collection.Snapshots[0].WindowStart) != 120*time.Second {
		t.Fatalf("expected configured collection window, got %+v", collection.Snapshots[0])
	}
}
