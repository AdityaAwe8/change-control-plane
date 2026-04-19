package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type prometheusProviderConfig struct {
	APIBaseURL     string
	QueryPath      string
	BearerTokenEnv string
	WindowSeconds  int
	StepSeconds    int
	Queries        []prometheusSignalQuery
}

type prometheusSignalQuery struct {
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Query      string  `json:"query"`
	Threshold  float64 `json:"threshold"`
	Comparator string  `json:"comparator"`
	Unit       string  `json:"unit,omitempty"`
	Severity   string  `json:"severity,omitempty"`
}

type prometheusQuerySample struct {
	Value       float64
	SampleCount int
	Empty       bool
}

type PrometheusProvider struct {
	httpClient *http.Client
	now        func() time.Time
}

func NewPrometheusProvider() SignalProvider {
	return PrometheusProvider{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func (p PrometheusProvider) Kind() string {
	return "prometheus"
}

func (p PrometheusProvider) Collect(ctx context.Context, runtime types.RolloutExecutionRuntimeContext) (Collection, error) {
	cfg, err := loadPrometheusProviderConfig(runtime)
	if err != nil {
		return Collection{}, err
	}
	windowEnd := p.now()
	windowStart := windowEnd.Add(-time.Duration(cfg.WindowSeconds) * time.Second)
	signals := make([]types.SignalValue, 0, len(cfg.Queries))
	explanation := make([]string, 0, len(cfg.Queries))
	overallHealth := "healthy"

	for _, query := range cfg.Queries {
		sample, err := p.queryRange(ctx, cfg, query.Query, windowStart, windowEnd)
		if err != nil {
			return Collection{}, err
		}
		signal := NormalizePrometheusSignal(query.Name, query.Category, sample.Value, query.Threshold, query.Comparator)
		signal.Unit = query.Unit
		if strings.EqualFold(query.Severity, "critical") && signal.Status == "degraded" {
			signal.Status = "critical"
		}
		if sample.Empty {
			signal.Status = "warning"
		}
		signals = append(signals, signal)
		if sample.Empty {
			explanation = append(explanation, fmt.Sprintf("%s returned no samples in the %ds collection window", signal.Name, cfg.WindowSeconds))
		} else {
			explanation = append(explanation, fmt.Sprintf("%s=%0.3f%s (%s %0.3f, samples=%d)", signal.Name, signal.Value, signal.Unit, signal.Comparator, signal.Threshold, sample.SampleCount))
		}
		overallHealth = combineSignalHealth(overallHealth, signal.Status)
	}

	snapshot := types.SignalSnapshot{
		BaseRecord: types.BaseRecord{
			ID:        runtime.Execution.ID + "-signal-" + strconv.FormatInt(windowEnd.Unix(), 10),
			CreatedAt: windowEnd,
			UpdatedAt: windowEnd,
		},
		OrganizationID:      runtime.Execution.OrganizationID,
		ProjectID:           runtime.Execution.ProjectID,
		RolloutExecutionID:  runtime.Execution.ID,
		RolloutPlanID:       runtime.Execution.RolloutPlanID,
		ChangeSetID:         runtime.Execution.ChangeSetID,
		ServiceID:           runtime.Execution.ServiceID,
		EnvironmentID:       runtime.Execution.EnvironmentID,
		ProviderType:        "prometheus",
		Health:              overallHealth,
		Summary:             prometheusSummary(overallHealth, signals),
		Signals:             signals,
		WindowStart:         windowStart,
		WindowEnd:           windowEnd,
	}
	if runtime.SignalIntegration != nil {
		snapshot.SourceIntegrationID = runtime.SignalIntegration.ID
	}
	return Collection{
		Snapshots:   []types.SignalSnapshot{snapshot},
		Source:      "prometheus",
		Explanation: explanation,
		CollectedAt: windowEnd,
	}, nil
}

func (p PrometheusProvider) queryRange(ctx context.Context, cfg prometheusProviderConfig, query string, start, end time.Time) (prometheusQuerySample, error) {
	endpoint, err := url.Parse(strings.TrimRight(cfg.APIBaseURL, "/") + cfg.QueryPath)
	if err != nil {
		return prometheusQuerySample{}, &SignalProviderError{Operation: "query_range", Temporary: false, Err: err}
	}
	params := endpoint.Query()
	params.Set("query", query)
	params.Set("start", strconv.FormatFloat(float64(start.Unix()), 'f', -1, 64))
	params.Set("end", strconv.FormatFloat(float64(end.Unix()), 'f', -1, 64))
	params.Set("step", strconv.Itoa(cfg.StepSeconds))
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return prometheusQuerySample{}, &SignalProviderError{Operation: "query_range", Temporary: false, Err: err}
	}
	req.Header.Set("Accept", "application/json")
	if cfg.BearerTokenEnv != "" {
		if token := strings.TrimSpace(os.Getenv(cfg.BearerTokenEnv)); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return prometheusQuerySample{}, &SignalProviderError{Operation: "query_range", Temporary: true, Err: err}
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return prometheusQuerySample{}, &SignalProviderError{Operation: "query_range", StatusCode: resp.StatusCode, Temporary: true, Err: readErr}
	}
	if resp.StatusCode >= http.StatusBadRequest {
		temporary := resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests
		return prometheusQuerySample{}, &SignalProviderError{Operation: "query_range", StatusCode: resp.StatusCode, Temporary: temporary, Err: fmt.Errorf("%s", strings.TrimSpace(string(body)))}
	}
	return decodePrometheusValue(body)
}

func loadPrometheusProviderConfig(runtime types.RolloutExecutionRuntimeContext) (prometheusProviderConfig, error) {
	metadata := types.Metadata{}
	if runtime.SignalIntegration != nil && runtime.SignalIntegration.Metadata != nil {
		for key, value := range runtime.SignalIntegration.Metadata {
			metadata[key] = value
		}
	}
	for key, value := range runtime.Execution.Metadata {
		metadata[key] = value
	}
	cfg := prometheusProviderConfig{
		APIBaseURL:     stringMetadataValue(metadata, "api_base_url"),
		QueryPath:      valueOrMetadata(stringMetadataValue(metadata, "query_path"), "/api/v1/query_range"),
		BearerTokenEnv: stringMetadataValue(metadata, "bearer_token_env"),
		WindowSeconds:  intMetadataValue(metadata, "window_seconds", 300),
		StepSeconds:    intMetadataValue(metadata, "step_seconds", 60),
		Queries:        decodePrometheusQueries(metadata["queries"]),
	}
	if cfg.APIBaseURL == "" {
		return cfg, fmt.Errorf("%w: prometheus integration metadata must include api_base_url", ErrSignalProviderUnavailable)
	}
	if len(cfg.Queries) == 0 {
		return cfg, fmt.Errorf("%w: prometheus queries configuration is required", ErrSignalProviderUnavailable)
	}
	return cfg, nil
}

func decodePrometheusValue(body []byte) (prometheusQuerySample, error) {
	var payload struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Value  []any   `json:"value"`
				Values [][]any `json:"values"`
			} `json:"result"`
		} `json:"data"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return prometheusQuerySample{}, err
	}
	if payload.Status != "success" {
		return prometheusQuerySample{}, fmt.Errorf("prometheus query failed: %s", payload.Error)
	}
	if len(payload.Data.Result) == 0 {
		return prometheusQuerySample{Empty: true}, nil
	}
	result := payload.Data.Result[0]
	if len(result.Values) > 0 {
		last := result.Values[len(result.Values)-1]
		if len(last) < 2 {
			return prometheusQuerySample{}, fmt.Errorf("prometheus matrix result missing sample value")
		}
		value, err := parsePrometheusNumber(last[1])
		if err != nil {
			return prometheusQuerySample{}, err
		}
		return prometheusQuerySample{Value: value, SampleCount: len(result.Values)}, nil
	}
	if len(result.Value) > 1 {
		value, err := parsePrometheusNumber(result.Value[1])
		if err != nil {
			return prometheusQuerySample{}, err
		}
		return prometheusQuerySample{Value: value, SampleCount: 1}, nil
	}
	return prometheusQuerySample{}, fmt.Errorf("prometheus result missing value")
}

func parsePrometheusNumber(value any) (float64, error) {
	switch typed := value.(type) {
	case string:
		return strconv.ParseFloat(typed, 64)
	case float64:
		return typed, nil
	default:
		return 0, fmt.Errorf("unsupported prometheus sample value type %T", value)
	}
}

func NormalizePrometheusSignal(name, category string, value, threshold float64, comparator string) types.SignalValue {
	status := "healthy"
	switch comparator {
	case ">=", ">":
		if value > threshold || (comparator == ">=" && value >= threshold) {
			status = "degraded"
		}
	case "<=", "<":
		if value < threshold || (comparator == "<=" && value <= threshold) {
			status = "degraded"
		}
	}
	return types.SignalValue{
		Name:       name,
		Category:   category,
		Value:      value,
		Threshold:  threshold,
		Comparator: comparator,
		Status:     status,
	}
}

func PrometheusCollectionPlaceholder() Collection {
	return Collection{
		Source:      "prometheus",
		Explanation: []string{"prometheus integration is modeled with normalization helpers but needs a configured query client to collect live samples"},
		CollectedAt: time.Now().UTC(),
	}
}

func decodePrometheusQueries(raw any) []prometheusSignalQuery {
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var queries []prometheusSignalQuery
	if err := json.Unmarshal(data, &queries); err != nil {
		return nil
	}
	return queries
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

func intMetadataValue(metadata types.Metadata, key string, fallback int) int {
	if metadata == nil {
		return fallback
	}
	value, ok := metadata[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed
		}
		return fallback
	default:
		return fallback
	}
}

func valueOrMetadata(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func combineSignalHealth(current, next string) string {
	order := map[string]int{
		"healthy":  0,
		"pass":     0,
		"warning":  1,
		"degraded": 1,
		"critical": 2,
		"fail":     2,
		"unhealthy": 2,
	}
	if order[strings.ToLower(strings.TrimSpace(next))] > order[strings.ToLower(strings.TrimSpace(current))] {
		return strings.ToLower(strings.TrimSpace(next))
	}
	return strings.ToLower(strings.TrimSpace(current))
}

func prometheusSummary(health string, signals []types.SignalValue) string {
	if len(signals) == 0 {
		return "no runtime signals returned from prometheus"
	}
	breaches := make([]string, 0, len(signals))
	for _, signal := range signals {
		if signal.Status == "healthy" {
			continue
		}
		breaches = append(breaches, signal.Name)
	}
	if len(breaches) == 0 {
		return "prometheus signals remained within configured thresholds"
	}
	return fmt.Sprintf("prometheus reported %s rollout health due to %s", health, strings.Join(breaches, ", "))
}
