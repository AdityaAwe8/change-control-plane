package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *PostgresStore) CreateRollbackPolicy(ctx context.Context, policy types.RollbackPolicy) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO rollback_policies (
			id, organization_id, project_id, service_id, environment_id, name, description, enabled, priority,
			max_error_rate, max_latency_ms, minimum_throughput, max_unhealthy_instances, max_restart_rate,
			max_verification_failures, rollback_on_provider_failure, rollback_on_critical_signals,
			metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, $17,
			$18, $19, $20
		)
	`, policy.ID, policy.OrganizationID, nullIfEmpty(policy.ProjectID), nullIfEmpty(policy.ServiceID), nullIfEmpty(policy.EnvironmentID), policy.Name, policy.Description, policy.Enabled, policy.Priority,
		policy.MaxErrorRate, policy.MaxLatencyMs, policy.MinimumThroughput, policy.MaxUnhealthyInstances, policy.MaxRestartRate,
		policy.MaxVerificationFailures, policy.RollbackOnProviderFailure, policy.RollbackOnCriticalSignals,
		jsonValue(policy.Metadata), policy.CreatedAt, policy.UpdatedAt)
	return err
}

func (s *PostgresStore) GetRollbackPolicy(ctx context.Context, id string) (types.RollbackPolicy, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, service_id, environment_id, name, description, enabled, priority,
			max_error_rate, max_latency_ms, minimum_throughput, max_unhealthy_instances, max_restart_rate,
			max_verification_failures, rollback_on_provider_failure, rollback_on_critical_signals,
			metadata, created_at, updated_at
		FROM rollback_policies
		WHERE id = $1
	`, id)
	return scanRollbackPolicy(row)
}

func (s *PostgresStore) ListRollbackPolicies(ctx context.Context, query RollbackPolicyQuery) ([]types.RollbackPolicy, error) {
	var enabled *bool
	if query.EnabledOnly {
		trueValue := true
		enabled = &trueValue
	}
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, service_id, environment_id, name, description, enabled, priority,
			max_error_rate, max_latency_ms, minimum_throughput, max_unhealthy_instances, max_restart_rate,
			max_verification_failures, rollback_on_provider_failure, rollback_on_critical_signals,
			metadata, created_at, updated_at
		FROM rollback_policies`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("service_id", query.ServiceID),
		filterEqual("environment_id", query.EnvironmentID),
		filterBool("enabled", enabled),
	)
	sqlQuery = strings.Replace(sqlQuery, " ORDER BY created_at", " ORDER BY priority DESC, created_at DESC", 1)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.RollbackPolicy
	for rows.Next() {
		item, err := scanRollbackPolicy(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateRollbackPolicy(ctx context.Context, policy types.RollbackPolicy) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE rollback_policies
		SET project_id = $2, service_id = $3, environment_id = $4, name = $5, description = $6,
			enabled = $7, priority = $8, max_error_rate = $9, max_latency_ms = $10, minimum_throughput = $11,
			max_unhealthy_instances = $12, max_restart_rate = $13, max_verification_failures = $14,
			rollback_on_provider_failure = $15, rollback_on_critical_signals = $16, metadata = $17, updated_at = $18
		WHERE id = $1
	`, policy.ID, nullIfEmpty(policy.ProjectID), nullIfEmpty(policy.ServiceID), nullIfEmpty(policy.EnvironmentID), policy.Name, policy.Description,
		policy.Enabled, policy.Priority, policy.MaxErrorRate, policy.MaxLatencyMs, policy.MinimumThroughput,
		policy.MaxUnhealthyInstances, policy.MaxRestartRate, policy.MaxVerificationFailures,
		policy.RollbackOnProviderFailure, policy.RollbackOnCriticalSignals, jsonValue(policy.Metadata), policy.UpdatedAt)
	return err
}

func (s *PostgresStore) CreateStatusEvent(ctx context.Context, event types.StatusEvent) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO status_events (
			id, organization_id, project_id, team_id, service_id, environment_id, rollout_execution_id, change_set_id,
			resource_type, resource_id, event_type, category, severity, previous_state, new_state, outcome,
			actor_id, actor_type, actor, source, automated, summary, explanation, correlation_id, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27
		)
	`, event.ID, event.OrganizationID, nullIfEmpty(event.ProjectID), nullIfEmpty(event.TeamID), nullIfEmpty(event.ServiceID), nullIfEmpty(event.EnvironmentID), nullIfEmpty(event.RolloutExecutionID), nullIfEmpty(event.ChangeSetID),
		event.ResourceType, event.ResourceID, event.EventType, event.Category, event.Severity, event.PreviousState, event.NewState, event.Outcome,
		event.ActorID, event.ActorType, event.Actor, event.Source, event.Automated, event.Summary, jsonValue(event.Explanation), event.CorrelationID, jsonValue(event.Metadata), event.CreatedAt, event.UpdatedAt)
	return err
}

func (s *PostgresStore) GetStatusEvent(ctx context.Context, id string) (types.StatusEvent, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, team_id, service_id, environment_id, rollout_execution_id, change_set_id,
			resource_type, resource_id, event_type, category, severity, previous_state, new_state, outcome,
			actor_id, actor_type, actor, source, automated, summary, explanation, correlation_id, metadata, created_at, updated_at
		FROM status_events
		WHERE id = $1
	`, id)
	return scanStatusEvent(row)
}

func (s *PostgresStore) ListStatusEvents(ctx context.Context, query StatusEventQuery) ([]types.StatusEvent, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, team_id, service_id, environment_id, rollout_execution_id, change_set_id,
			resource_type, resource_id, event_type, category, severity, previous_state, new_state, outcome,
			actor_id, actor_type, actor, source, automated, summary, explanation, correlation_id, metadata, created_at, updated_at
		FROM status_events`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("team_id", query.TeamID),
		filterEqual("service_id", query.ServiceID),
		filterEqual("environment_id", query.EnvironmentID),
		filterEqual("rollout_execution_id", query.RolloutExecutionID),
		filterEqual("change_set_id", query.ChangeSetID),
		filterEqual("resource_type", query.ResourceType),
		filterEqual("resource_id", query.ResourceID),
		filterAny("event_type", query.EventTypes),
		filterEqual("actor_type", query.ActorType),
		filterEqual("actor_id", query.ActorID),
		filterEqual("source", query.Source),
		filterEqual("outcome", query.Outcome),
		filterBool("automated", query.Automated),
		filterRollbackEvents(query.RollbackOnly),
		filterTSQuery(query.Search),
		filterTimeAfter("created_at", query.Since),
		filterTimeBefore("created_at", query.Until),
	)
	sqlQuery = strings.Replace(sqlQuery, " ORDER BY created_at", " ORDER BY created_at DESC", 1)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.StatusEvent
	for rows.Next() {
		item, err := scanStatusEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) CountStatusEvents(ctx context.Context, query StatusEventQuery) (int, error) {
	sqlQuery, args := buildCountQuery(
		`SELECT COUNT(*) FROM status_events`,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("team_id", query.TeamID),
		filterEqual("service_id", query.ServiceID),
		filterEqual("environment_id", query.EnvironmentID),
		filterEqual("rollout_execution_id", query.RolloutExecutionID),
		filterEqual("change_set_id", query.ChangeSetID),
		filterEqual("resource_type", query.ResourceType),
		filterEqual("resource_id", query.ResourceID),
		filterAny("event_type", query.EventTypes),
		filterEqual("actor_type", query.ActorType),
		filterEqual("actor_id", query.ActorID),
		filterEqual("source", query.Source),
		filterEqual("outcome", query.Outcome),
		filterBool("automated", query.Automated),
		filterRollbackEvents(query.RollbackOnly),
		filterTSQuery(query.Search),
		filterTimeAfter("created_at", query.Since),
		filterTimeBefore("created_at", query.Until),
	)
	var total int
	if err := s.runner(ctx).QueryRowContext(ctx, sqlQuery, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func scanRollbackPolicy(row scanner) (types.RollbackPolicy, error) {
	var item types.RollbackPolicy
	var projectID sql.NullString
	var serviceID sql.NullString
	var environmentID sql.NullString
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &projectID, &serviceID, &environmentID, &item.Name, &item.Description, &item.Enabled, &item.Priority,
		&item.MaxErrorRate, &item.MaxLatencyMs, &item.MinimumThroughput, &item.MaxUnhealthyInstances, &item.MaxRestartRate,
		&item.MaxVerificationFailures, &item.RollbackOnProviderFailure, &item.RollbackOnCriticalSignals,
		&metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.ProjectID = projectID.String
	item.ServiceID = serviceID.String
	item.EnvironmentID = environmentID.String
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanStatusEvent(row scanner) (types.StatusEvent, error) {
	var item types.StatusEvent
	var projectID sql.NullString
	var teamID sql.NullString
	var serviceID sql.NullString
	var environmentID sql.NullString
	var rolloutExecutionID sql.NullString
	var changeSetID sql.NullString
	var explanation []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &projectID, &teamID, &serviceID, &environmentID, &rolloutExecutionID, &changeSetID,
		&item.ResourceType, &item.ResourceID, &item.EventType, &item.Category, &item.Severity, &item.PreviousState, &item.NewState, &item.Outcome,
		&item.ActorID, &item.ActorType, &item.Actor, &item.Source, &item.Automated, &item.Summary, &explanation, &item.CorrelationID, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.ProjectID = projectID.String
	item.TeamID = teamID.String
	item.ServiceID = serviceID.String
	item.EnvironmentID = environmentID.String
	item.RolloutExecutionID = rolloutExecutionID.String
	item.ChangeSetID = changeSetID.String
	_ = json.Unmarshal(explanation, &item.Explanation)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}
