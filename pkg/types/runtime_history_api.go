package types

type CreateRollbackPolicyRequest struct {
	OrganizationID            string   `json:"organization_id"`
	ProjectID                 string   `json:"project_id,omitempty"`
	ServiceID                 string   `json:"service_id,omitempty"`
	EnvironmentID             string   `json:"environment_id,omitempty"`
	Name                      string   `json:"name"`
	Description               string   `json:"description,omitempty"`
	Enabled                   *bool    `json:"enabled,omitempty"`
	Priority                  int      `json:"priority,omitempty"`
	MaxErrorRate              float64  `json:"max_error_rate,omitempty"`
	MaxLatencyMs              float64  `json:"max_latency_ms,omitempty"`
	MinimumThroughput         float64  `json:"minimum_throughput,omitempty"`
	MaxUnhealthyInstances     int      `json:"max_unhealthy_instances,omitempty"`
	MaxRestartRate            float64  `json:"max_restart_rate,omitempty"`
	MaxVerificationFailures   int      `json:"max_verification_failures,omitempty"`
	RollbackOnProviderFailure *bool    `json:"rollback_on_provider_failure,omitempty"`
	RollbackOnCriticalSignals *bool    `json:"rollback_on_critical_signals,omitempty"`
	Metadata                  Metadata `json:"metadata,omitempty"`
}

type UpdateRollbackPolicyRequest struct {
	Name                      *string  `json:"name,omitempty"`
	Description               *string  `json:"description,omitempty"`
	Enabled                   *bool    `json:"enabled,omitempty"`
	Priority                  *int     `json:"priority,omitempty"`
	MaxErrorRate              *float64 `json:"max_error_rate,omitempty"`
	MaxLatencyMs              *float64 `json:"max_latency_ms,omitempty"`
	MinimumThroughput         *float64 `json:"minimum_throughput,omitempty"`
	MaxUnhealthyInstances     *int     `json:"max_unhealthy_instances,omitempty"`
	MaxRestartRate            *float64 `json:"max_restart_rate,omitempty"`
	MaxVerificationFailures   *int     `json:"max_verification_failures,omitempty"`
	RollbackOnProviderFailure *bool    `json:"rollback_on_provider_failure,omitempty"`
	RollbackOnCriticalSignals *bool    `json:"rollback_on_critical_signals,omitempty"`
	Metadata                  Metadata `json:"metadata,omitempty"`
}
