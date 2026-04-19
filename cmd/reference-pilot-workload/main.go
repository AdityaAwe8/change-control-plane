package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type workloadState struct {
	mu           sync.RWMutex
	version      string
	latencyMS    float64
	errorRatio   float64
	requestTotal float64
	errorTotal   float64
}

func main() {
	port := valueOrDefault("PORT", "8080")
	service := valueOrDefault("REFERENCE_PILOT_SERVICE", "checkout")
	environment := valueOrDefault("REFERENCE_PILOT_ENVIRONMENT", "pilot")

	state := &workloadState{
		version:    valueOrDefault("REFERENCE_PILOT_VERSION", "v1"),
		latencyMS:  valueOrDefaultFloat("REFERENCE_PILOT_LATENCY_MS", 85),
		errorRatio: valueOrDefaultFloat("REFERENCE_PILOT_ERROR_RATIO", 0.002),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		snapshot := state.recordRequest()
		writeJSON(w, http.StatusOK, map[string]any{
			"service":       service,
			"environment":   environment,
			"version":       snapshot.version,
			"latency_ms":    snapshot.latencyMS,
			"error_ratio":   snapshot.errorRatio,
			"request_total": snapshot.requestTotal,
			"error_total":   snapshot.errorTotal,
		})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		snapshot := state.snapshot()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(w,
			"# HELP reference_pilot_request_latency_ms Simulated request latency for the reference pilot workload.\n"+
				"# TYPE reference_pilot_request_latency_ms gauge\n"+
				"reference_pilot_request_latency_ms{service=%q,environment=%q,version=%q} %0.3f\n"+
				"# HELP reference_pilot_error_ratio Simulated error ratio for the reference pilot workload.\n"+
				"# TYPE reference_pilot_error_ratio gauge\n"+
				"reference_pilot_error_ratio{service=%q,environment=%q,version=%q} %0.6f\n"+
				"# HELP reference_pilot_http_requests_total Simulated request volume.\n"+
				"# TYPE reference_pilot_http_requests_total counter\n"+
				"reference_pilot_http_requests_total{service=%q,environment=%q,version=%q,code=%q} %0.0f\n"+
				"reference_pilot_http_requests_total{service=%q,environment=%q,version=%q,code=%q} %0.0f\n"+
				"# HELP reference_pilot_workload_info Static workload identity.\n"+
				"# TYPE reference_pilot_workload_info gauge\n"+
				"reference_pilot_workload_info{service=%q,environment=%q,version=%q} 1\n",
			service, environment, snapshot.version, snapshot.latencyMS,
			service, environment, snapshot.version, snapshot.errorRatio,
			service, environment, snapshot.version, "200", math.Max(snapshot.requestTotal-snapshot.errorTotal, 0),
			service, environment, snapshot.version, "500", snapshot.errorTotal,
			service, environment, snapshot.version,
		)
	})
	mux.HandleFunc("POST /admin/state", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var request struct {
			Version    *string  `json:"version,omitempty"`
			LatencyMS  *float64 `json:"latency_ms,omitempty"`
			ErrorRatio *float64 `json:"error_ratio,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}
		snapshot := state.update(request.Version, request.LatencyMS, request.ErrorRatio)
		writeJSON(w, http.StatusOK, map[string]any{
			"service":       service,
			"environment":   environment,
			"version":       snapshot.version,
			"latency_ms":    snapshot.latencyMS,
			"error_ratio":   snapshot.errorRatio,
			"request_total": snapshot.requestTotal,
			"error_total":   snapshot.errorTotal,
			"updated_at":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	log.Printf("reference pilot workload listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

type workloadSnapshot struct {
	version      string
	latencyMS    float64
	errorRatio   float64
	requestTotal float64
	errorTotal   float64
}

func (s *workloadState) snapshot() workloadSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return workloadSnapshot{
		version:      s.version,
		latencyMS:    s.latencyMS,
		errorRatio:   s.errorRatio,
		requestTotal: s.requestTotal,
		errorTotal:   s.errorTotal,
	}
}

func (s *workloadState) recordRequest() workloadSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestTotal++
	if s.errorRatio > 0 {
		s.errorTotal = math.Round(s.requestTotal * s.errorRatio)
	}
	return workloadSnapshot{
		version:      s.version,
		latencyMS:    s.latencyMS,
		errorRatio:   s.errorRatio,
		requestTotal: s.requestTotal,
		errorTotal:   s.errorTotal,
	}
}

func (s *workloadState) update(version *string, latencyMS, errorRatio *float64) workloadSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	if version != nil && strings.TrimSpace(*version) != "" {
		s.version = strings.TrimSpace(*version)
	}
	if latencyMS != nil {
		s.latencyMS = math.Max(*latencyMS, 0)
	}
	if errorRatio != nil {
		s.errorRatio = clamp(*errorRatio, 0, 1)
		s.errorTotal = math.Round(s.requestTotal * s.errorRatio)
	}
	return workloadSnapshot{
		version:      s.version,
		latencyMS:    s.latencyMS,
		errorRatio:   s.errorRatio,
		requestTotal: s.requestTotal,
		errorTotal:   s.errorTotal,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func valueOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func valueOrDefaultFloat(key string, fallback float64) float64 {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
