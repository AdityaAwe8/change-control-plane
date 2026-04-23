package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

var (
	allowedDatabaseConnectionSourceTypes = map[string]struct{}{
		"env_dsn":        {},
		"secret_ref_dsn": {},
	}
	allowedDatabaseConnectionTestStatuses = map[string]struct{}{
		"running":  {},
		"passed":   {},
		"blocked":  {},
		"errored":  {},
	}
)

type databaseConnectionResolution struct {
	Driver          string
	DSN             string
	SourceType      string
	SourceReference string
	ResolvedFrom    string
}

type databaseConnectionHealthResult struct {
	Status              string
	Summary             string
	Details             []string
	ErrorClass          string
	ConnectionStatus    string
	ConnectionErrorText string
}

func (a *Application) ListDatabaseConnectionTests(ctx context.Context, query storage.DatabaseConnectionTestQuery) ([]types.DatabaseConnectionTest, error) {
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
	return a.Store.ListDatabaseConnectionTests(ctx, query)
}

func (a *Application) GetDatabaseConnectionTestDetail(ctx context.Context, id string) (types.DatabaseConnectionTestDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}
	item, err := a.Store.GetDatabaseConnectionTest(ctx, id)
	if err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}
	if !a.Authorizer.CanReadProject(identity, item.OrganizationID, item.ProjectID) {
		return types.DatabaseConnectionTestDetail{}, ErrForbidden
	}
	return a.buildDatabaseConnectionTestDetail(ctx, item)
}

func (a *Application) TestDatabaseConnectionReference(ctx context.Context, id string, req types.TestDatabaseConnectionReferenceRequest) (types.DatabaseConnectionTestDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}
	connectionRef, err := a.Store.GetDatabaseConnectionReference(ctx, id)
	if err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}
	service, team, err := a.databaseGovernanceServiceTeam(ctx, connectionRef.ServiceID)
	if err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}
	if !a.canManageConfigSet(identity, connectionRef.OrganizationID, connectionRef.ProjectID, service, team) {
		return types.DatabaseConnectionTestDetail{}, a.forbidden(ctx, identity, "database_connection.test.denied", "database_connection_reference", connectionRef.ID, connectionRef.OrganizationID, connectionRef.ProjectID, []string{"actor lacks database connection governance permission"})
	}

	now := time.Now().UTC()
	test := types.DatabaseConnectionTest{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("dbct"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:  connectionRef.OrganizationID,
		ProjectID:       connectionRef.ProjectID,
		EnvironmentID:   connectionRef.EnvironmentID,
		ServiceID:       connectionRef.ServiceID,
		ConnectionRefID: connectionRef.ID,
		Trigger:         normalizeDatabaseExecutionTrigger(req.Trigger),
		Status:          "running",
		Summary:         "database connection test started",
		ActorType:       string(identity.ActorType),
		ActorID:         identity.ActorID,
		StartedAt:       now,
	}
	if err := a.Store.CreateDatabaseConnectionTest(ctx, test); err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}

	connectionRef.Status = "testing"
	connectionRef.UpdatedAt = now
	if err := a.Store.UpdateDatabaseConnectionReference(ctx, connectionRef); err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}

	result := a.probeDatabaseConnection(ctx, connectionRef)
	finishedAt := time.Now().UTC()
	test.Status = result.Status
	test.Summary = result.Summary
	test.Details = result.Details
	test.ErrorClass = result.ErrorClass
	test.CompletedAt = &finishedAt
	test.UpdatedAt = finishedAt
	if err := a.Store.UpdateDatabaseConnectionTest(ctx, test); err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}

	connectionRef.Status = result.ConnectionStatus
	connectionRef.LastTestedAt = &finishedAt
	if result.Status == "passed" {
		connectionRef.LastHealthyAt = &finishedAt
		connectionRef.LastErrorClass = ""
		connectionRef.LastErrorSummary = ""
	} else {
		connectionRef.LastErrorClass = result.ErrorClass
		connectionRef.LastErrorSummary = result.ConnectionErrorText
	}
	connectionRef.UpdatedAt = finishedAt
	if err := a.Store.UpdateDatabaseConnectionReference(ctx, connectionRef); err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}

	if err := a.record(ctx, identity, "database_connection.tested", "database_connection_reference", connectionRef.ID, connectionRef.OrganizationID, connectionRef.ProjectID, []string{connectionRef.Name, test.Status, test.ID}); err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}
	return a.buildDatabaseConnectionTestDetail(ctx, test)
}

func (a *Application) buildDatabaseConnectionTestDetail(ctx context.Context, item types.DatabaseConnectionTest) (types.DatabaseConnectionTestDetail, error) {
	connectionRef, err := a.Store.GetDatabaseConnectionReference(ctx, item.ConnectionRefID)
	if err != nil {
		return types.DatabaseConnectionTestDetail{}, err
	}
	return types.DatabaseConnectionTestDetail{
		ConnectionTest:      item,
		ConnectionReference: connectionRef,
	}, nil
}

func listDatabaseConnectionTestsForReference(ctx context.Context, store storage.Store, item types.DatabaseConnectionReference, limit int) ([]types.DatabaseConnectionTest, error) {
	if limit <= 0 {
		limit = 20
	}
	return store.ListDatabaseConnectionTests(ctx, storage.DatabaseConnectionTestQuery{
		OrganizationID:  item.OrganizationID,
		ProjectID:       item.ProjectID,
		ConnectionRefID: item.ID,
		Limit:           limit,
	})
}

func normalizeDatabaseConnectionSourceType(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		normalized = "env_dsn"
	}
	if _, ok := allowedDatabaseConnectionSourceTypes[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database connection source_type %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseConnectionSourceValues(sourceType, dsnEnv, secretRef, secretRefEnv string) (string, string, string, string, error) {
	normalizedSourceType, err := normalizeDatabaseConnectionSourceType(sourceType)
	if err != nil {
		return "", "", "", "", err
	}
	switch normalizedSourceType {
	case "env_dsn":
		normalizedDSNEnv, err := normalizeDatabaseEnvName(dsnEnv)
		if err != nil {
			return "", "", "", "", err
		}
		if strings.TrimSpace(secretRef) != "" {
			return "", "", "", "", fmt.Errorf("%w: secret_ref is unsupported when source_type is env_dsn", ErrValidation)
		}
		if strings.TrimSpace(secretRefEnv) != "" {
			return "", "", "", "", fmt.Errorf("%w: secret_ref_env is unsupported when source_type is env_dsn", ErrValidation)
		}
		return normalizedSourceType, normalizedDSNEnv, "", "", nil
	case "secret_ref_dsn":
		trimmedSecretRef := strings.TrimSpace(secretRef)
		if trimmedSecretRef == "" {
			return "", "", "", "", fmt.Errorf("%w: secret_ref is required when source_type is secret_ref_dsn", ErrValidation)
		}
		if !looksLikeSecretReference(trimmedSecretRef) {
			return "", "", "", "", fmt.Errorf("%w: invalid secret_ref %q", ErrValidation, secretRef)
		}
		if strings.TrimSpace(dsnEnv) != "" {
			return "", "", "", "", fmt.Errorf("%w: dsn_env is unsupported when source_type is secret_ref_dsn", ErrValidation)
		}
		trimmedSecretRefEnv := strings.TrimSpace(secretRefEnv)
		if trimmedSecretRefEnv != "" {
			trimmedSecretRefEnv, err = normalizeDatabaseEnvName(trimmedSecretRefEnv)
			if err != nil {
				return "", "", "", "", err
			}
		}
		return normalizedSourceType, "", trimmedSecretRef, trimmedSecretRefEnv, nil
	default:
		return "", "", "", "", fmt.Errorf("%w: unsupported database connection source_type %q", ErrValidation, sourceType)
	}
}

func databaseConnectionHealthInputsChanged(before, after types.DatabaseConnectionReference) bool {
	return before.Driver != after.Driver ||
		before.SourceType != after.SourceType ||
		before.DSNEnv != after.DSNEnv ||
		before.SecretRef != after.SecretRef ||
		before.SecretRefEnv != after.SecretRefEnv ||
		before.ReadOnlyCapable != after.ReadOnlyCapable
}

func resetDatabaseConnectionHealth(item *types.DatabaseConnectionReference) {
	item.Status = databaseConnectionBaseStatus(*item)
	item.LastTestedAt = nil
	item.LastHealthyAt = nil
	item.LastErrorClass = ""
	item.LastErrorSummary = ""
}

func databaseConnectionBaseStatus(item types.DatabaseConnectionReference) string {
	if strings.TrimSpace(item.SourceType) == "secret_ref_dsn" && strings.TrimSpace(item.SecretRefEnv) == "" {
		return "unresolved"
	}
	return "defined"
}

func databaseConnectionSourceSummary(item types.DatabaseConnectionReference) string {
	switch strings.TrimSpace(item.SourceType) {
	case "secret_ref_dsn":
		ref := strings.TrimSpace(item.SecretRef)
		if ref == "" {
			ref = "unset"
		}
		if strings.TrimSpace(item.SecretRefEnv) == "" {
			return "secret_ref:" + ref + " via env:unbound"
		}
		return "secret_ref:" + ref + " via env:" + item.SecretRefEnv
	default:
		return "env:" + item.DSNEnv
	}
}

func (a *Application) databaseConnectionRuntimeValue(envName string) (string, bool) {
	trimmedEnvName := strings.TrimSpace(envName)
	if trimmedEnvName == "" {
		return "", false
	}
	if value := strings.TrimSpace(os.Getenv(trimmedEnvName)); value != "" {
		return value, true
	}
	if trimmedEnvName == "CCP_DB_DSN" && strings.TrimSpace(a.Config.DBDSN) != "" {
		return strings.TrimSpace(a.Config.DBDSN), true
	}
	return "", false
}

func (a *Application) resolveDatabaseConnectionRuntime(item types.DatabaseConnectionReference) (databaseConnectionResolution, string, string, error) {
	sourceType := strings.TrimSpace(item.SourceType)
	if sourceType == "" {
		sourceType = "env_dsn"
	}
	switch sourceType {
	case "env_dsn":
		if strings.TrimSpace(item.DSNEnv) == "" {
			return databaseConnectionResolution{}, "unresolved", "missing_env_ref", fmt.Errorf("%w: connection env reference is not configured", ErrValidation)
		}
		if value, ok := a.databaseConnectionRuntimeValue(item.DSNEnv); ok {
			return databaseConnectionResolution{
				Driver:          item.Driver,
				DSN:             value,
				SourceType:      sourceType,
				SourceReference: item.DSNEnv,
				ResolvedFrom:    item.DSNEnv,
			}, "ready", "", nil
		}
		return databaseConnectionResolution{}, "unresolved", "missing_env_value", fmt.Errorf("%w: env reference %s is not set in the runtime environment", ErrValidation, item.DSNEnv)
	case "secret_ref_dsn":
		if strings.TrimSpace(item.SecretRefEnv) == "" {
			return databaseConnectionResolution{}, "unresolved", "missing_secret_ref_env", fmt.Errorf("%w: secret ref %s does not have a supported runtime env binding", ErrValidation, item.SecretRef)
		}
		if value, ok := a.databaseConnectionRuntimeValue(item.SecretRefEnv); ok {
			return databaseConnectionResolution{
				Driver:          item.Driver,
				DSN:             value,
				SourceType:      sourceType,
				SourceReference: item.SecretRef,
				ResolvedFrom:    item.SecretRefEnv,
			}, "ready", "", nil
		}
		return databaseConnectionResolution{}, "unresolved", "missing_secret_ref_env_value", fmt.Errorf("%w: secret ref %s is bound to env %s but no runtime value is available", ErrValidation, item.SecretRef, item.SecretRefEnv)
	default:
		return databaseConnectionResolution{}, "unsupported", "unsupported_source", fmt.Errorf("%w: unsupported connection source type %s", ErrValidation, sourceType)
	}
}

func (a *Application) databaseConnectionRuntimeCapability(item types.DatabaseConnectionReference) (string, string, string) {
	_, status, errorClass, err := a.resolveDatabaseConnectionRuntime(item)
	if err != nil {
		return status, errorClass, err.Error()
	}
	return status, "", "runtime connection source can be resolved"
}

func (a *Application) openReadOnlyDatabaseTransaction(ctx context.Context, resolution databaseConnectionResolution) (*sql.DB, *sql.Tx, context.Context, context.CancelFunc, string, error) {
	runCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	db, err := sql.Open("pgx", resolution.DSN)
	if err != nil {
		cancel()
		return nil, nil, nil, nil, "driver_open", err
	}
	if err := db.PingContext(runCtx); err != nil {
		_ = db.Close()
		cancel()
		return nil, nil, nil, nil, "connectivity", err
	}
	tx, err := db.BeginTx(runCtx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		_ = db.Close()
		cancel()
		return nil, nil, nil, nil, "begin_tx", err
	}
	return db, tx, runCtx, cancel, "", nil
}

func (a *Application) probeDatabaseConnection(ctx context.Context, item types.DatabaseConnectionReference) databaseConnectionHealthResult {
	resolution, connectionStatus, errorClass, err := a.resolveDatabaseConnectionRuntime(item)
	if err != nil {
		return databaseConnectionHealthResult{
			Status:              ternary(connectionStatus == "unsupported", "blocked", "blocked"),
			Summary:             "database connection test could not resolve runtime access",
			Details:             []string{err.Error(), "source=" + databaseConnectionSourceSummary(item)},
			ErrorClass:          errorClass,
			ConnectionStatus:    connectionStatus,
			ConnectionErrorText: err.Error(),
		}
	}
	db, tx, runCtx, cancel, openErrorClass, err := a.openReadOnlyDatabaseTransaction(ctx, resolution)
	if err != nil {
		safe := sanitizeDatabaseConnectionError(&resolution, err)
		return databaseConnectionHealthResult{
			Status:              "errored",
			Summary:             "database connection test failed during read-only connection setup",
			Details:             append([]string{safe, "source=" + resolution.SourceType + ":" + resolution.SourceReference}, databaseConnectionResolutionDetails(resolution)...),
			ErrorClass:          openErrorClass,
			ConnectionStatus:    "error",
			ConnectionErrorText: safe,
		}
	}
	defer cancel()
	defer db.Close()
	defer tx.Rollback()

	var currentDatabase string
	if err := tx.QueryRowContext(runCtx, `SELECT current_database()`).Scan(&currentDatabase); err != nil {
		safe := sanitizeDatabaseConnectionError(&resolution, err)
		return databaseConnectionHealthResult{
			Status:              "errored",
			Summary:             "database connection test failed during read-only verification",
			Details:             append([]string{safe, "source=" + resolution.SourceType + ":" + resolution.SourceReference}, databaseConnectionResolutionDetails(resolution)...),
			ErrorClass:          "query_failed",
			ConnectionStatus:    "error",
			ConnectionErrorText: safe,
		}
	}
	if err := tx.Commit(); err != nil {
		safe := sanitizeDatabaseConnectionError(&resolution, err)
		return databaseConnectionHealthResult{
			Status:              "errored",
			Summary:             "database connection test could not finish the read-only verification transaction",
			Details:             append([]string{safe, "source=" + resolution.SourceType + ":" + resolution.SourceReference}, databaseConnectionResolutionDetails(resolution)...),
			ErrorClass:          "commit",
			ConnectionStatus:    "error",
			ConnectionErrorText: safe,
		}
	}
	return databaseConnectionHealthResult{
		Status:           "passed",
		Summary:          fmt.Sprintf("database connection %s is ready for read-only validation", item.Name),
		Details:          append([]string{"source=" + resolution.SourceType + ":" + resolution.SourceReference, "driver=" + resolution.Driver, "read_only_transaction=true", "current_database=" + currentDatabase}, databaseConnectionResolutionDetails(resolution)...),
		ConnectionStatus: "ready",
	}
}

func databaseConnectionResolutionDetails(resolution databaseConnectionResolution) []string {
	if strings.TrimSpace(resolution.ResolvedFrom) == "" {
		return nil
	}
	return []string{"resolved_via_env=" + resolution.ResolvedFrom}
}

func databaseConnectionResolutionEvidence(resolution databaseConnectionResolution) []string {
	if strings.TrimSpace(resolution.ResolvedFrom) == "" {
		return nil
	}
	return []string{"resolved_via_env:" + resolution.ResolvedFrom}
}

func sanitizeDatabaseConnectionError(resolution *databaseConnectionResolution, err error) string {
	message := strings.TrimSpace(err.Error())
	if resolution != nil {
		for _, token := range databaseConnectionSensitiveTokens(resolution.DSN) {
			if token == "" {
				continue
			}
			message = strings.ReplaceAll(message, token, "[redacted]")
		}
	}
	return message
}

func databaseConnectionSensitiveTokens(dsn string) []string {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" {
		return nil
	}
	tokens := []string{trimmed}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return tokens
	}
	if parsed.User != nil {
		if username := parsed.User.Username(); username != "" {
			tokens = append(tokens, username)
		}
		if password, ok := parsed.User.Password(); ok && password != "" {
			tokens = append(tokens, password)
		}
	}
	queryValues := parsed.Query()
	for _, key := range []string{"password", "pass", "pwd", "token", "secret", "api_key", "apikey"} {
		for _, value := range queryValues[key] {
			if strings.TrimSpace(value) != "" {
				tokens = append(tokens, value)
			}
		}
	}
	return tokens
}

func latestDatabaseConnectionTestByReference(items []types.DatabaseConnectionTest) map[string]types.DatabaseConnectionTest {
	latest := make(map[string]types.DatabaseConnectionTest)
	sort.Slice(items, func(i, j int) bool {
		if items[i].StartedAt.Equal(items[j].StartedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].StartedAt.After(items[j].StartedAt)
	})
	for _, item := range items {
		if _, ok := latest[item.ConnectionRefID]; ok {
			continue
		}
		latest[item.ConnectionRefID] = item
	}
	return latest
}
