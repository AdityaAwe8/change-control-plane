package types

type RollbackPolicy struct {
	BaseRecord
	OrganizationID           string  `json:"organization_id"`
	ProjectID                string  `json:"project_id,omitempty"`
	ServiceID                string  `json:"service_id,omitempty"`
	EnvironmentID            string  `json:"environment_id,omitempty"`
	Name                     string  `json:"name"`
	Description              string  `json:"description,omitempty"`
	Enabled                  bool    `json:"enabled"`
	Priority                 int     `json:"priority"`
	MaxErrorRate             float64 `json:"max_error_rate,omitempty"`
	MaxLatencyMs             float64 `json:"max_latency_ms,omitempty"`
	MinimumThroughput        float64 `json:"minimum_throughput,omitempty"`
	MaxUnhealthyInstances    int     `json:"max_unhealthy_instances,omitempty"`
	MaxRestartRate           float64 `json:"max_restart_rate,omitempty"`
	MaxVerificationFailures  int     `json:"max_verification_failures,omitempty"`
	RollbackOnProviderFailure bool   `json:"rollback_on_provider_failure"`
	RollbackOnCriticalSignals bool   `json:"rollback_on_critical_signals"`
}

type StatusEvent struct {
	BaseRecord
	OrganizationID    string   `json:"organization_id"`
	ProjectID         string   `json:"project_id,omitempty"`
	TeamID            string   `json:"team_id,omitempty"`
	ServiceID         string   `json:"service_id,omitempty"`
	EnvironmentID     string   `json:"environment_id,omitempty"`
	RolloutExecutionID string  `json:"rollout_execution_id,omitempty"`
	ChangeSetID       string   `json:"change_set_id,omitempty"`
	ResourceType      string   `json:"resource_type"`
	ResourceID        string   `json:"resource_id"`
	EventType         string   `json:"event_type"`
	Category          string   `json:"category"`
	Severity          string   `json:"severity"`
	PreviousState     string   `json:"previous_state,omitempty"`
	NewState          string   `json:"new_state,omitempty"`
	Outcome           string   `json:"outcome,omitempty"`
	ActorID           string   `json:"actor_id,omitempty"`
	ActorType         string   `json:"actor_type,omitempty"`
	Actor             string   `json:"actor,omitempty"`
	Source            string   `json:"source,omitempty"`
	Automated         bool     `json:"automated"`
	Summary           string   `json:"summary"`
	Explanation       []string `json:"explanation,omitempty"`
	CorrelationID     string   `json:"correlation_id,omitempty"`
}
