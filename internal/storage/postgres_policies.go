package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *PostgresStore) CreatePolicy(ctx context.Context, policy types.Policy) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO policies (
			id, organization_id, project_id, service_id, environment_id, name, code, scope, applies_to, mode,
			enabled, priority, description, conditions, triggers, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18
		)
	`, policy.ID, policy.OrganizationID, nullIfEmpty(policy.ProjectID), nullIfEmpty(policy.ServiceID), nullIfEmpty(policy.EnvironmentID),
		policy.Name, policy.Code, policy.Scope, policy.AppliesTo, policy.Mode,
		policy.Enabled, policy.Priority, policy.Description, jsonValue(policy.Conditions), jsonValue(policy.Triggers), jsonValue(policy.Metadata), policy.CreatedAt, policy.UpdatedAt)
	return err
}

func (s *PostgresStore) GetPolicy(ctx context.Context, id string) (types.Policy, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, project_id, service_id, environment_id, name, code, scope, applies_to, mode,
			enabled, priority, description, conditions, triggers, metadata, created_at, updated_at
		FROM policies
		WHERE id = $1
	`, id)
	return scanPolicy(row)
}

func (s *PostgresStore) ListPolicies(ctx context.Context, query PolicyQuery) ([]types.Policy, error) {
	var enabled *bool
	if query.EnabledOnly {
		trueValue := true
		enabled = &trueValue
	}
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, service_id, environment_id, name, code, scope, applies_to, mode,
			enabled, priority, description, conditions, triggers, metadata, created_at, updated_at
		FROM policies`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("service_id", query.ServiceID),
		filterEqual("environment_id", query.EnvironmentID),
		filterEqual("applies_to", query.AppliesTo),
		filterBool("enabled", enabled),
	)
	sqlQuery = strings.Replace(sqlQuery, " ORDER BY created_at", " ORDER BY priority DESC, created_at DESC", 1)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.Policy
	for rows.Next() {
		item, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdatePolicy(ctx context.Context, policy types.Policy) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE policies
		SET project_id = $2, service_id = $3, environment_id = $4, name = $5, code = $6, scope = $7, applies_to = $8, mode = $9,
			enabled = $10, priority = $11, description = $12, conditions = $13, triggers = $14, metadata = $15, updated_at = $16
		WHERE id = $1
	`, policy.ID, nullIfEmpty(policy.ProjectID), nullIfEmpty(policy.ServiceID), nullIfEmpty(policy.EnvironmentID), policy.Name, policy.Code, policy.Scope, policy.AppliesTo, policy.Mode,
		policy.Enabled, policy.Priority, policy.Description, jsonValue(policy.Conditions), jsonValue(policy.Triggers), jsonValue(policy.Metadata), policy.UpdatedAt)
	return err
}

func (s *PostgresStore) CreatePolicyDecision(ctx context.Context, decision types.PolicyDecision) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO policy_decisions (
			id, organization_id, project_id, service_id, environment_id, policy_id, policy_name, policy_code, policy_scope, applies_to, mode,
			change_set_id, risk_assessment_id, rollout_plan_id, rollout_execution_id, outcome, summary, reasons, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`, decision.ID, decision.OrganizationID, nullIfEmpty(decision.ProjectID), nullIfEmpty(decision.ServiceID), nullIfEmpty(decision.EnvironmentID), decision.PolicyID,
		decision.PolicyName, decision.PolicyCode, decision.PolicyScope, decision.AppliesTo, decision.Mode,
		nullIfEmpty(decision.ChangeSetID), nullIfEmpty(decision.RiskAssessmentID), nullIfEmpty(decision.RolloutPlanID), nullIfEmpty(decision.RolloutExecutionID),
		decision.Outcome, decision.Summary, jsonValue(decision.Reasons), jsonValue(decision.Metadata), decision.CreatedAt, decision.UpdatedAt)
	return err
}

func (s *PostgresStore) ListPolicyDecisions(ctx context.Context, query PolicyDecisionQuery) ([]types.PolicyDecision, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, project_id, service_id, environment_id, policy_id, policy_name, policy_code, policy_scope, applies_to, mode,
			change_set_id, risk_assessment_id, rollout_plan_id, rollout_execution_id, outcome, summary, reasons, metadata, created_at, updated_at
		FROM policy_decisions`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("project_id", query.ProjectID),
		filterEqual("policy_id", query.PolicyID),
		filterEqual("change_set_id", query.ChangeSetID),
		filterEqual("risk_assessment_id", query.RiskAssessmentID),
		filterEqual("rollout_plan_id", query.RolloutPlanID),
		filterEqual("rollout_execution_id", query.RolloutExecutionID),
		filterEqual("applies_to", query.AppliesTo),
	)
	sqlQuery = strings.Replace(sqlQuery, " ORDER BY created_at", " ORDER BY created_at DESC", 1)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.PolicyDecision
	for rows.Next() {
		item, err := scanPolicyDecision(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanPolicy(row scanner) (types.Policy, error) {
	var item types.Policy
	var projectID sql.NullString
	var serviceID sql.NullString
	var environmentID sql.NullString
	var conditions []byte
	var triggers []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &projectID, &serviceID, &environmentID, &item.Name, &item.Code, &item.Scope, &item.AppliesTo, &item.Mode,
		&item.Enabled, &item.Priority, &item.Description, &conditions, &triggers, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.ProjectID = projectID.String
	item.ServiceID = serviceID.String
	item.EnvironmentID = environmentID.String
	_ = json.Unmarshal(conditions, &item.Conditions)
	_ = json.Unmarshal(triggers, &item.Triggers)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanPolicyDecision(row scanner) (types.PolicyDecision, error) {
	var item types.PolicyDecision
	var projectID sql.NullString
	var serviceID sql.NullString
	var environmentID sql.NullString
	var changeSetID sql.NullString
	var riskAssessmentID sql.NullString
	var rolloutPlanID sql.NullString
	var rolloutExecutionID sql.NullString
	var reasons []byte
	var metadata []byte
	err := row.Scan(
		&item.ID, &item.OrganizationID, &projectID, &serviceID, &environmentID, &item.PolicyID, &item.PolicyName, &item.PolicyCode, &item.PolicyScope, &item.AppliesTo, &item.Mode,
		&changeSetID, &riskAssessmentID, &rolloutPlanID, &rolloutExecutionID, &item.Outcome, &item.Summary, &reasons, &metadata, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.ProjectID = projectID.String
	item.ServiceID = serviceID.String
	item.EnvironmentID = environmentID.String
	item.ChangeSetID = changeSetID.String
	item.RiskAssessmentID = riskAssessmentID.String
	item.RolloutPlanID = rolloutPlanID.String
	item.RolloutExecutionID = rolloutExecutionID.String
	_ = json.Unmarshal(reasons, &item.Reasons)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}
