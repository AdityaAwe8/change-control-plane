package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	allowedDatabaseConnectionDrivers = map[string]struct{}{
		"postgres": {},
	}
	allowedDatabaseConnectionStatuses = map[string]struct{}{
		"defined":     {},
		"testing":     {},
		"ready":       {},
		"error":       {},
		"unresolved":  {},
		"unsupported": {},
	}
	allowedDatabaseExecutionTriggers = map[string]struct{}{
		"manual":            {},
		"release_analysis":  {},
		"rollout_pre_deploy": {},
		"rollout_post_deploy": {},
		"rollback":          {},
	}
	allowedDatabaseExecutionStatuses = map[string]struct{}{
		"queued":   {},
		"running":  {},
		"passed":   {},
		"failed":   {},
		"blocked":  {},
		"errored":  {},
	}
	envNamePattern        = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)
	databaseIdentPattern  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type runtimeDatabaseCheckSpec struct {
	Subject       string `json:"subject,omitempty"`
	Schema        string `json:"schema,omitempty"`
	Table         string `json:"table,omitempty"`
	Column        string `json:"column,omitempty"`
	Index         string `json:"index,omitempty"`
	Operator      string `json:"operator,omitempty"`
	ExpectedCount *int64 `json:"expected_count,omitempty"`
}

type runtimeDatabaseCheckResult struct {
	Status              string
	Summary             string
	Details             []string
	Evidence            []string
	ErrorClass          string
	ConnectionStatus    string
	ConnectionErrorText string
}

func (a *Application) ListDatabaseConnectionReferences(ctx context.Context) ([]types.DatabaseConnectionReference, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListDatabaseConnectionReferences(ctx, storage.DatabaseConnectionReferenceQuery{
		OrganizationID: orgID,
		Limit:          500,
	})
}

func (a *Application) CreateDatabaseConnectionReference(ctx context.Context, req types.CreateDatabaseConnectionReferenceRequest) (types.DatabaseConnectionReferenceDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	project, environment, service, team, err := a.validateDatabaseConnectionScope(ctx, req.OrganizationID, req.ProjectID, req.EnvironmentID, req.ServiceID)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	if !a.canManageConfigSet(identity, req.OrganizationID, project.ID, service, team) {
		return types.DatabaseConnectionReferenceDetail{}, a.forbidden(ctx, identity, "database_connection.create.denied", "database_connection_reference", "", req.OrganizationID, project.ID, []string{"actor lacks database connection governance permission"})
	}
	driver, err := normalizeDatabaseConnectionDriver(req.Driver)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	sourceType, dsnEnv, secretRef, secretRefEnv, err := normalizeDatabaseConnectionSourceValues(req.SourceType, req.DSNEnv, req.SecretRef, req.SecretRefEnv)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	now := time.Now().UTC()
	item := types.DatabaseConnectionReference{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("dbconn"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:  req.OrganizationID,
		ProjectID:       project.ID,
		EnvironmentID:   environment.ID,
		Name:            strings.TrimSpace(req.Name),
		Datastore:       strings.TrimSpace(req.Datastore),
		Driver:          driver,
		SourceType:      sourceType,
		DSNEnv:          dsnEnv,
		SecretRef:       secretRef,
		SecretRefEnv:    secretRefEnv,
		ReadOnlyCapable: req.ReadOnlyCapable || driver == "postgres",
		Summary:         strings.TrimSpace(req.Summary),
	}
	resetDatabaseConnectionHealth(&item)
	if service != nil {
		item.ServiceID = service.ID
	}
	if item.Name == "" || item.Datastore == "" || item.Summary == "" {
		return types.DatabaseConnectionReferenceDetail{}, fmt.Errorf("%w: name, datastore, and summary are required", ErrValidation)
	}
	if err := a.Store.CreateDatabaseConnectionReference(ctx, item); err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	if err := a.record(ctx, identity, "database_connection.created", "database_connection_reference", item.ID, item.OrganizationID, item.ProjectID, []string{item.Name, item.Driver, databaseConnectionSourceSummary(item)}); err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	return a.buildDatabaseConnectionReferenceDetail(ctx, item)
}

func (a *Application) GetDatabaseConnectionReferenceDetail(ctx context.Context, id string) (types.DatabaseConnectionReferenceDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	item, err := a.Store.GetDatabaseConnectionReference(ctx, id)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	if !a.Authorizer.CanReadProject(identity, item.OrganizationID, item.ProjectID) {
		return types.DatabaseConnectionReferenceDetail{}, ErrForbidden
	}
	return a.buildDatabaseConnectionReferenceDetail(ctx, item)
}

func (a *Application) UpdateDatabaseConnectionReference(ctx context.Context, id string, req types.UpdateDatabaseConnectionReferenceRequest) (types.DatabaseConnectionReferenceDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	item, err := a.Store.GetDatabaseConnectionReference(ctx, id)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	service, team, err := a.databaseGovernanceServiceTeam(ctx, item.ServiceID)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	if !a.canManageConfigSet(identity, item.OrganizationID, item.ProjectID, service, team) {
		return types.DatabaseConnectionReferenceDetail{}, a.forbidden(ctx, identity, "database_connection.update.denied", "database_connection_reference", item.ID, item.OrganizationID, item.ProjectID, []string{"actor lacks database connection governance permission"})
	}
	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if req.Datastore != nil {
		item.Datastore = strings.TrimSpace(*req.Datastore)
	}
	previousHealthState := item
	if req.Driver != nil {
		item.Driver, err = normalizeDatabaseConnectionDriver(*req.Driver)
		if err != nil {
			return types.DatabaseConnectionReferenceDetail{}, err
		}
	}
	sourceType := item.SourceType
	if req.SourceType != nil {
		sourceType = *req.SourceType
	}
	dsnEnv := item.DSNEnv
	if req.DSNEnv != nil {
		dsnEnv = *req.DSNEnv
	} else if req.SourceType != nil && strings.TrimSpace(strings.ToLower(*req.SourceType)) == "secret_ref_dsn" {
		dsnEnv = ""
	}
	secretRef := item.SecretRef
	if req.SecretRef != nil {
		secretRef = *req.SecretRef
	} else if req.SourceType != nil && strings.TrimSpace(strings.ToLower(*req.SourceType)) == "env_dsn" {
		secretRef = ""
	}
	secretRefEnv := item.SecretRefEnv
	if req.SecretRefEnv != nil {
		secretRefEnv = *req.SecretRefEnv
	} else if req.SourceType != nil && strings.TrimSpace(strings.ToLower(*req.SourceType)) == "env_dsn" {
		secretRefEnv = ""
	}
	item.SourceType, item.DSNEnv, item.SecretRef, item.SecretRefEnv, err = normalizeDatabaseConnectionSourceValues(sourceType, dsnEnv, secretRef, secretRefEnv)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	if req.ReadOnlyCapable != nil {
		item.ReadOnlyCapable = *req.ReadOnlyCapable
	}
	if req.Summary != nil {
		item.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.Metadata != nil {
		item.Metadata = req.Metadata
	}
	if item.Name == "" || item.Datastore == "" || item.Summary == "" {
		return types.DatabaseConnectionReferenceDetail{}, fmt.Errorf("%w: name, datastore, and summary are required", ErrValidation)
	}
	if databaseConnectionHealthInputsChanged(previousHealthState, item) {
		resetDatabaseConnectionHealth(&item)
	}
	item.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateDatabaseConnectionReference(ctx, item); err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	if err := a.record(ctx, identity, "database_connection.updated", "database_connection_reference", item.ID, item.OrganizationID, item.ProjectID, []string{item.Name, item.Driver, databaseConnectionSourceSummary(item)}); err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	return a.buildDatabaseConnectionReferenceDetail(ctx, item)
}

func (a *Application) ListDatabaseValidationExecutions(ctx context.Context, query storage.DatabaseValidationExecutionQuery) ([]types.DatabaseValidationExecution, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	query.OrganizationID = orgID
	if query.Limit <= 0 {
		query.Limit = 500
	}
	return a.Store.ListDatabaseValidationExecutions(ctx, query)
}

func (a *Application) GetDatabaseValidationExecutionDetail(ctx context.Context, id string) (types.DatabaseValidationExecutionDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}
	item, err := a.Store.GetDatabaseValidationExecution(ctx, id)
	if err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}
	if !a.Authorizer.CanReadProject(identity, item.OrganizationID, item.ProjectID) {
		return types.DatabaseValidationExecutionDetail{}, ErrForbidden
	}
	return a.buildDatabaseValidationExecutionDetail(ctx, item)
}

func (a *Application) ExecuteDatabaseValidationCheck(ctx context.Context, id string, req types.ExecuteDatabaseValidationCheckRequest) (types.DatabaseValidationExecutionDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}
	check, err := a.Store.GetDatabaseValidationCheck(ctx, id)
	if err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}
	service, team, err := a.databaseGovernanceServiceTeam(ctx, check.ServiceID)
	if err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}
	if !a.canManageConfigSet(identity, check.OrganizationID, check.ProjectID, service, team) {
		return types.DatabaseValidationExecutionDetail{}, a.forbidden(ctx, identity, "database_check.execute.denied", "database_validation_check", check.ID, check.OrganizationID, check.ProjectID, []string{"actor lacks database validation execution permission"})
	}
	if check.ExecutionMode != "runtime_read_only" {
		return types.DatabaseValidationExecutionDetail{}, fmt.Errorf("%w: only runtime_read_only checks can be executed", ErrValidation)
	}
	if strings.TrimSpace(check.ConnectionRefID) == "" {
		return types.DatabaseValidationExecutionDetail{}, fmt.Errorf("%w: runtime_read_only checks require connection_ref_id", ErrValidation)
	}
	connectionRef, err := a.Store.GetDatabaseConnectionReference(ctx, check.ConnectionRefID)
	if err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}
	spec, err := parseRuntimeDatabaseCheckSpec(check)
	if err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}

	now := time.Now().UTC()
	execution := types.DatabaseValidationExecution{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("dbexec"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:    check.OrganizationID,
		ProjectID:         check.ProjectID,
		EnvironmentID:     check.EnvironmentID,
		ServiceID:         check.ServiceID,
		ChangeSetID:       check.ChangeSetID,
		DatabaseChangeID:  check.DatabaseChangeID,
		ValidationCheckID: check.ID,
		ConnectionRefID:   connectionRef.ID,
		Trigger:           normalizeDatabaseExecutionTrigger(req.Trigger),
		ExecutionMode:     check.ExecutionMode,
		Status:            "running",
		Summary:           "database validation execution started",
		ActorType:         string(identity.ActorType),
		ActorID:           identity.ActorID,
		StartedAt:         now,
	}
	if err := a.Store.CreateDatabaseValidationExecution(ctx, execution); err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}

	result := a.executeRuntimeDatabaseValidation(ctx, connectionRef, check, spec)
	finishedAt := time.Now().UTC()
	execution.Status = result.Status
	execution.Summary = result.Summary
	execution.ResultDetails = result.Details
	execution.Evidence = result.Evidence
	execution.ErrorClass = result.ErrorClass
	execution.CompletedAt = &finishedAt
	execution.UpdatedAt = finishedAt
	if err := a.Store.UpdateDatabaseValidationExecution(ctx, execution); err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}

	connectionRef.LastTestedAt = &finishedAt
	connectionRef.Status = result.ConnectionStatus
	connectionRef.LastErrorSummary = result.ConnectionErrorText
	connectionRef.LastErrorClass = result.ErrorClass
	if result.Status == "passed" {
		connectionRef.LastHealthyAt = &finishedAt
		connectionRef.LastErrorClass = ""
		connectionRef.LastErrorSummary = ""
	}
	connectionRef.UpdatedAt = finishedAt
	if err := a.Store.UpdateDatabaseConnectionReference(ctx, connectionRef); err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}

	check.LastRunAt = &finishedAt
	check.LastResultSummary = result.Summary
	check.UpdatedAt = finishedAt
	switch result.Status {
	case "passed":
		check.Status = "passed"
	case "failed":
		check.Status = "failed"
	default:
		check.Status = "blocked"
	}
	if err := a.Store.UpdateDatabaseValidationCheck(ctx, check); err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}

	if err := a.record(ctx, identity, "database_check.executed", "database_validation_check", check.ID, check.OrganizationID, check.ProjectID, []string{check.Name, execution.Status, execution.ID}); err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}
	return a.buildDatabaseValidationExecutionDetail(ctx, execution)
}

func (a *Application) buildDatabaseConnectionReferenceDetail(ctx context.Context, item types.DatabaseConnectionReference) (types.DatabaseConnectionReferenceDetail, error) {
	checks, err := a.Store.ListDatabaseValidationChecks(ctx, storage.DatabaseValidationCheckQuery{
		OrganizationID: item.OrganizationID,
		ProjectID:      item.ProjectID,
		ConnectionRefID: item.ID,
		Limit:          200,
	})
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	connectionTests, err := listDatabaseConnectionTestsForReference(ctx, a.Store, item, 20)
	if err != nil {
		return types.DatabaseConnectionReferenceDetail{}, err
	}
	return types.DatabaseConnectionReferenceDetail{
		ConnectionReference: item,
		ValidationChecks:    checks,
		ConnectionTests:     connectionTests,
	}, nil
}

func (a *Application) buildDatabaseValidationExecutionDetail(ctx context.Context, item types.DatabaseValidationExecution) (types.DatabaseValidationExecutionDetail, error) {
	check, err := a.Store.GetDatabaseValidationCheck(ctx, item.ValidationCheckID)
	if err != nil {
		return types.DatabaseValidationExecutionDetail{}, err
	}
	result := types.DatabaseValidationExecutionDetail{
		Execution:       item,
		ValidationCheck: check,
	}
	if strings.TrimSpace(item.DatabaseChangeID) != "" {
		change, err := a.Store.GetDatabaseChange(ctx, item.DatabaseChangeID)
		if err != nil {
			return types.DatabaseValidationExecutionDetail{}, err
		}
		result.DatabaseChange = &change
	}
	if strings.TrimSpace(item.ConnectionRefID) != "" {
		connectionRef, err := a.Store.GetDatabaseConnectionReference(ctx, item.ConnectionRefID)
		if err != nil {
			return types.DatabaseValidationExecutionDetail{}, err
		}
		result.ConnectionReference = &connectionRef
	}
	return result, nil
}

func (a *Application) validateDatabaseConnectionScope(ctx context.Context, organizationID, projectID, environmentID, serviceID string) (types.Project, types.Environment, *types.Service, *types.Team, error) {
	project, environment, err := a.validateProjectEnvironment(ctx, organizationID, projectID, environmentID)
	if err != nil {
		return types.Project{}, types.Environment{}, nil, nil, err
	}
	if strings.TrimSpace(serviceID) == "" {
		return project, environment, nil, nil, nil
	}
	service, err := a.Store.GetService(ctx, strings.TrimSpace(serviceID))
	if err != nil {
		return types.Project{}, types.Environment{}, nil, nil, err
	}
	if service.OrganizationID != organizationID || service.ProjectID != projectID {
		return types.Project{}, types.Environment{}, nil, nil, fmt.Errorf("%w: service scope mismatch", ErrValidation)
	}
	team, err := a.Store.GetTeam(ctx, service.TeamID)
	if err != nil {
		return types.Project{}, types.Environment{}, nil, nil, err
	}
	return project, environment, &service, &team, nil
}

func normalizeDatabaseConnectionDriver(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		normalized = "postgres"
	}
	if _, ok := allowedDatabaseConnectionDrivers[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database connection driver %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseEnvName(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%w: dsn_env is required", ErrValidation)
	}
	if !envNamePattern.MatchString(trimmed) {
		return "", fmt.Errorf("%w: invalid env var name %q", ErrValidation, value)
	}
	return trimmed, nil
}

func normalizeDatabaseExecutionTrigger(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if _, ok := allowedDatabaseExecutionTriggers[normalized]; ok {
		return normalized
	}
	return "manual"
}

func parseRuntimeDatabaseCheckSpec(check types.DatabaseValidationCheck) (runtimeDatabaseCheckSpec, error) {
	var spec runtimeDatabaseCheckSpec
	if check.ExecutionMode != "runtime_read_only" {
		return spec, nil
	}
	if err := json.Unmarshal([]byte(check.Specification), &spec); err != nil {
		return spec, fmt.Errorf("%w: runtime_read_only database checks require JSON specification: %v", ErrValidation, err)
	}
	if spec.Schema == "" {
		spec.Schema = "public"
	}
	switch check.CheckType {
	case "existence_assertion", "migration_completion":
		if spec.Subject == "" {
			spec.Subject = "table"
		}
		switch spec.Subject {
		case "table":
			if spec.Table == "" {
				return spec, fmt.Errorf("%w: existence-style checks require table in specification", ErrValidation)
			}
		case "column":
			if spec.Table == "" || spec.Column == "" {
				return spec, fmt.Errorf("%w: column existence checks require table and column in specification", ErrValidation)
			}
		case "index":
			if spec.Index == "" {
				return spec, fmt.Errorf("%w: index existence checks require index in specification", ErrValidation)
			}
		default:
			return spec, fmt.Errorf("%w: unsupported existence subject %q", ErrValidation, spec.Subject)
		}
	case "row_count_assertion":
		if spec.Table == "" || spec.ExpectedCount == nil {
			return spec, fmt.Errorf("%w: row_count_assertion requires table and expected_count in specification", ErrValidation)
		}
		switch spec.Operator {
		case "eq", "gt", "gte", "lt", "lte":
		default:
			return spec, fmt.Errorf("%w: row_count_assertion requires operator eq,gt,gte,lt,lte", ErrValidation)
		}
	default:
		return spec, fmt.Errorf("%w: runtime_read_only execution is unsupported for check type %q", ErrValidation, check.CheckType)
	}
	for _, identifier := range []string{spec.Schema, spec.Table, spec.Column, spec.Index} {
		if identifier == "" {
			continue
		}
		if !databaseIdentPattern.MatchString(identifier) {
			return spec, fmt.Errorf("%w: unsupported identifier %q in runtime database specification", ErrValidation, identifier)
		}
	}
	return spec, nil
}

func (a *Application) executeRuntimeDatabaseValidation(ctx context.Context, connectionRef types.DatabaseConnectionReference, check types.DatabaseValidationCheck, spec runtimeDatabaseCheckSpec) runtimeDatabaseCheckResult {
	result := runtimeDatabaseCheckResult{
		ConnectionStatus: connectionRef.Status,
	}
	resolution, connectionStatus, errorClass, err := a.resolveDatabaseConnectionRuntime(connectionRef)
	if err != nil {
		result.Status = "blocked"
		result.Summary = "database validation execution blocked"
		result.Details = []string{err.Error(), "source=" + databaseConnectionSourceSummary(connectionRef)}
		result.Evidence = []string{"connection_ref:" + connectionRef.ID, "source:" + databaseConnectionSourceSummary(connectionRef)}
		result.ErrorClass = errorClass
		result.ConnectionStatus = connectionStatus
		result.ConnectionErrorText = err.Error()
		return result
	}
	db, tx, runCtx, cancel, openErrorClass, err := a.openReadOnlyDatabaseTransaction(ctx, resolution)
	if err != nil {
		safe := sanitizeDatabaseConnectionError(&resolution, err)
		result.Status = "errored"
		result.Summary = "database validation execution could not start a read-only database session"
		result.Details = append([]string{safe, "source=" + resolution.SourceType + ":" + resolution.SourceReference}, databaseConnectionResolutionDetails(resolution)...)
		result.Evidence = append([]string{"connection_ref:" + connectionRef.ID, "source:" + resolution.SourceType + ":" + resolution.SourceReference}, databaseConnectionResolutionEvidence(resolution)...)
		result.ErrorClass = openErrorClass
		result.ConnectionStatus = "error"
		result.ConnectionErrorText = safe
		return result
	}
	defer db.Close()
	defer cancel()
	defer tx.Rollback()

	switch check.CheckType {
	case "existence_assertion", "migration_completion":
		result = executeDatabaseExistenceAssertion(runCtx, tx, connectionRef, check, spec)
	case "row_count_assertion":
		result = executeDatabaseRowCountAssertion(runCtx, tx, connectionRef, check, spec)
	default:
		result = runtimeDatabaseCheckResult{
			Status:              "blocked",
			Summary:             "database validation execution is unsupported for this check type",
			Details:             []string{fmt.Sprintf("check type %s cannot run in runtime_read_only mode", check.CheckType)},
			Evidence:            []string{"connection_ref:" + connectionRef.ID, "source:" + databaseConnectionSourceSummary(connectionRef)},
			ErrorClass:          "unsupported_check_type",
			ConnectionStatus:    "defined",
			ConnectionErrorText: "",
		}
	}
	if result.Status == "passed" || result.Status == "failed" {
		if err := tx.Commit(); err != nil {
			safe := sanitizeDatabaseConnectionError(&resolution, err)
			result.Status = "errored"
			result.Summary = "database validation execution could not commit the read-only transaction"
			result.Details = append(result.Details, safe)
			result.ErrorClass = "commit"
			result.ConnectionStatus = "error"
			result.ConnectionErrorText = safe
		}
	}
	return result
}

func executeDatabaseExistenceAssertion(ctx context.Context, tx *sql.Tx, connectionRef types.DatabaseConnectionReference, check types.DatabaseValidationCheck, spec runtimeDatabaseCheckSpec) runtimeDatabaseCheckResult {
	var exists bool
	var err error
	switch spec.Subject {
	case "table":
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = $1 AND table_name = $2
			)
		`, spec.Schema, spec.Table).Scan(&exists)
	case "column":
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
			)
		`, spec.Schema, spec.Table, spec.Column).Scan(&exists)
	case "index":
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_indexes
				WHERE schemaname = $1 AND indexname = $2
			)
		`, spec.Schema, spec.Index).Scan(&exists)
	default:
		return runtimeDatabaseCheckResult{
			Status:              "blocked",
			Summary:             "database validation execution is missing a supported subject",
			Details:             []string{fmt.Sprintf("subject %q is unsupported", spec.Subject)},
			Evidence:            []string{"connection_ref:" + connectionRef.ID, "source:" + databaseConnectionSourceSummary(connectionRef)},
			ErrorClass:          "unsupported_subject",
			ConnectionStatus:    "defined",
			ConnectionErrorText: "",
		}
	}
	if err != nil {
		safe := sanitizeDatabaseExecutionError("", err)
		return runtimeDatabaseCheckResult{
			Status:              "errored",
			Summary:             "database validation execution could not query existence state",
			Details:             []string{safe},
			Evidence:            []string{"connection_ref:" + connectionRef.ID, "source:" + databaseConnectionSourceSummary(connectionRef)},
			ErrorClass:          "query_failed",
			ConnectionStatus:    "error",
			ConnectionErrorText: safe,
		}
	}
	target := spec.Schema + "."
	switch spec.Subject {
	case "table":
		target += spec.Table
	case "column":
		target += spec.Table + "." + spec.Column
	case "index":
		target += spec.Index
	}
	status := "failed"
	summary := fmt.Sprintf("runtime database check %s did not find %s", check.Name, target)
	if exists {
		status = "passed"
		summary = fmt.Sprintf("runtime database check %s confirmed %s", check.Name, target)
	}
	return runtimeDatabaseCheckResult{
		Status:           status,
		Summary:          summary,
		Details:          []string{fmt.Sprintf("subject=%s exists=%t", spec.Subject, exists)},
		Evidence:         []string{"connection_ref:" + connectionRef.ID, "source:" + databaseConnectionSourceSummary(connectionRef), "subject:" + spec.Subject, "target:" + target},
		ConnectionStatus: "ready",
	}
}

func executeDatabaseRowCountAssertion(ctx context.Context, tx *sql.Tx, connectionRef types.DatabaseConnectionReference, check types.DatabaseValidationCheck, spec runtimeDatabaseCheckSpec) runtimeDatabaseCheckResult {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s.%s`, quoteDatabaseIdentifier(spec.Schema), quoteDatabaseIdentifier(spec.Table))
	var count int64
	if err := tx.QueryRowContext(ctx, query).Scan(&count); err != nil {
		safe := sanitizeDatabaseExecutionError("", err)
		return runtimeDatabaseCheckResult{
			Status:              "errored",
			Summary:             "database validation execution could not query row count",
			Details:             []string{safe},
			Evidence:            []string{"connection_ref:" + connectionRef.ID, "source:" + databaseConnectionSourceSummary(connectionRef), "target:" + spec.Schema + "." + spec.Table},
			ErrorClass:          "query_failed",
			ConnectionStatus:    "error",
			ConnectionErrorText: safe,
		}
	}
	passed := compareRowCount(count, spec.Operator, *spec.ExpectedCount)
	status := "failed"
	summary := fmt.Sprintf("runtime database row-count check %s failed", check.Name)
	if passed {
		status = "passed"
		summary = fmt.Sprintf("runtime database row-count check %s passed", check.Name)
	}
	return runtimeDatabaseCheckResult{
		Status:           status,
		Summary:          summary,
		Details:          []string{fmt.Sprintf("row_count=%d operator=%s expected=%d", count, spec.Operator, *spec.ExpectedCount)},
		Evidence:         []string{"connection_ref:" + connectionRef.ID, "source:" + databaseConnectionSourceSummary(connectionRef), "target:" + spec.Schema + "." + spec.Table},
		ConnectionStatus: "ready",
	}
}

func compareRowCount(actual int64, operator string, expected int64) bool {
	switch operator {
	case "eq":
		return actual == expected
	case "gt":
		return actual > expected
	case "gte":
		return actual >= expected
	case "lt":
		return actual < expected
	case "lte":
		return actual <= expected
	default:
		return false
	}
}

func quoteDatabaseIdentifier(value string) string {
	return `"` + value + `"`
}

func sanitizeDatabaseExecutionError(dsn string, err error) string {
	return sanitizeDatabaseConnectionError(&databaseConnectionResolution{DSN: dsn}, err)
}

func latestDatabaseExecutionByCheck(items []types.DatabaseValidationExecution) map[string]types.DatabaseValidationExecution {
	latest := make(map[string]types.DatabaseValidationExecution)
	sort.Slice(items, func(i, j int) bool {
		if items[i].StartedAt.Equal(items[j].StartedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].StartedAt.After(items[j].StartedAt)
	})
	for _, item := range items {
		if _, ok := latest[item.ValidationCheckID]; ok {
			continue
		}
		latest[item.ValidationCheckID] = item
	}
	return latest
}
