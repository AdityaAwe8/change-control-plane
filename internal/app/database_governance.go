package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

var (
	allowedDatabaseOperationTypes = map[string]struct{}{
		"schema_change":      {},
		"data_backfill":      {},
		"index_change":       {},
		"destructive_change": {},
		"expand_contract":    {},
	}
	allowedDatabaseExecutionIntents = map[string]struct{}{
		"pre_deploy":    {},
		"during_deploy": {},
		"post_deploy":   {},
		"out_of_band":   {},
	}
	allowedDatabaseCompatibility = map[string]struct{}{
		"backward_compatible":  {},
		"expand_contract":      {},
		"forward_incompatible": {},
	}
	allowedDatabaseReversibility = map[string]struct{}{
		"reversible":   {},
		"irreversible": {},
		"manual_only":  {},
	}
	allowedDatabaseChangeStatuses = map[string]struct{}{
		"defined":   {},
		"reviewed":  {},
		"approved":  {},
		"completed": {},
		"blocked":   {},
	}
	allowedDatabaseCheckPhases = map[string]struct{}{
		"pre_deploy":  {},
		"post_deploy": {},
		"rollback":    {},
	}
	allowedDatabaseCheckTypes = map[string]struct{}{
		"migration_completion": {},
		"compatibility_check":  {},
		"row_count_assertion":  {},
		"existence_assertion":  {},
		"custom_read_only":     {},
	}
	allowedDatabaseCheckModes = map[string]struct{}{
		"manual_attestation": {},
		"advisory_only":      {},
		"runtime_read_only":  {},
	}
	allowedDatabaseCheckStatuses = map[string]struct{}{
		"defined":       {},
		"passed":        {},
		"failed":        {},
		"blocked":       {},
		"advisory_only": {},
	}
)

type databaseGovernanceSnapshot struct {
	Connections      []types.DatabaseConnectionReference
	ConnectionTests  []types.DatabaseConnectionTest
	Changes          []types.DatabaseChange
	Checks           []types.DatabaseValidationCheck
	Executions       []types.DatabaseValidationExecution
	Findings         []string
	PolicyHighlights []string
	Warnings         []string
	Blockers         []string
	Posture          types.DatabasePosture
}

func (a *Application) ListDatabaseChanges(ctx context.Context) ([]types.DatabaseChange, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListDatabaseChanges(ctx, storage.DatabaseChangeQuery{OrganizationID: orgID, Limit: 500})
}

func (a *Application) CreateDatabaseChange(ctx context.Context, req types.CreateDatabaseChangeRequest) (types.DatabaseChangeDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	_, environment, changeSet, service, team, err := a.validateDatabaseGovernanceScope(ctx, req.OrganizationID, req.ProjectID, req.EnvironmentID, req.ServiceID, req.ChangeSetID)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	if !a.canManageConfigSet(identity, req.OrganizationID, req.ProjectID, service, team) {
		return types.DatabaseChangeDetail{}, a.forbidden(ctx, identity, "database_change.create.denied", "database_change", "", req.OrganizationID, req.ProjectID, []string{"actor lacks database governance mutation permission"})
	}

	riskLevel, err := normalizeDatabaseRiskLevel(req.RiskLevel)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	operationType, err := normalizeDatabaseOperationType(req.OperationType)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	executionIntent, err := normalizeDatabaseExecutionIntent(req.ExecutionIntent)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	compatibility, err := normalizeDatabaseCompatibility(req.Compatibility)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	reversibility, err := normalizeDatabaseReversibility(req.Reversibility)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}

	now := time.Now().UTC()
	item := types.DatabaseChange{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("dbchg"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:         req.OrganizationID,
		ProjectID:              req.ProjectID,
		EnvironmentID:          environment.ID,
		ServiceID:              changeSet.ServiceID,
		ChangeSetID:            changeSet.ID,
		Name:                   strings.TrimSpace(req.Name),
		Datastore:              strings.TrimSpace(req.Datastore),
		OperationType:          operationType,
		ExecutionIntent:        executionIntent,
		Compatibility:          compatibility,
		Reversibility:          reversibility,
		RiskLevel:              riskLevel,
		LockRisk:               req.LockRisk,
		ManualApprovalRequired: req.ManualApprovalRequired || databaseChangeRequiresManualApproval(environment, operationType, reversibility, req.LockRisk),
		Status:                 normalizeDatabaseChangeStatus(""),
		Summary:                strings.TrimSpace(req.Summary),
		Evidence:               normalizeDatabaseEvidence(req.Evidence),
	}
	if item.Name == "" || item.Datastore == "" || item.Summary == "" {
		return types.DatabaseChangeDetail{}, fmt.Errorf("%w: name, datastore, and summary are required", ErrValidation)
	}
	if err := a.Store.CreateDatabaseChange(ctx, item); err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	if err := a.record(ctx, identity, "database_change.created", "database_change", item.ID, item.OrganizationID, item.ProjectID, []string{item.Name, item.Datastore, item.OperationType}); err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	return a.buildDatabaseChangeDetail(ctx, item)
}

func (a *Application) GetDatabaseChangeDetail(ctx context.Context, id string) (types.DatabaseChangeDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	item, err := a.Store.GetDatabaseChange(ctx, id)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	if !a.Authorizer.CanReadProject(identity, item.OrganizationID, item.ProjectID) {
		return types.DatabaseChangeDetail{}, ErrForbidden
	}
	return a.buildDatabaseChangeDetail(ctx, item)
}

func (a *Application) UpdateDatabaseChange(ctx context.Context, id string, req types.UpdateDatabaseChangeRequest) (types.DatabaseChangeDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	item, err := a.Store.GetDatabaseChange(ctx, id)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	environment, err := a.Store.GetEnvironment(ctx, item.EnvironmentID)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	service, team, err := a.databaseGovernanceServiceTeam(ctx, item.ServiceID)
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	if !a.canManageConfigSet(identity, item.OrganizationID, item.ProjectID, service, team) {
		return types.DatabaseChangeDetail{}, a.forbidden(ctx, identity, "database_change.update.denied", "database_change", item.ID, item.OrganizationID, item.ProjectID, []string{"actor lacks database governance mutation permission"})
	}

	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if req.Datastore != nil {
		item.Datastore = strings.TrimSpace(*req.Datastore)
	}
	if req.OperationType != nil {
		item.OperationType, err = normalizeDatabaseOperationType(*req.OperationType)
		if err != nil {
			return types.DatabaseChangeDetail{}, err
		}
	}
	if req.ExecutionIntent != nil {
		item.ExecutionIntent, err = normalizeDatabaseExecutionIntent(*req.ExecutionIntent)
		if err != nil {
			return types.DatabaseChangeDetail{}, err
		}
	}
	if req.Compatibility != nil {
		item.Compatibility, err = normalizeDatabaseCompatibility(*req.Compatibility)
		if err != nil {
			return types.DatabaseChangeDetail{}, err
		}
	}
	if req.Reversibility != nil {
		item.Reversibility, err = normalizeDatabaseReversibility(*req.Reversibility)
		if err != nil {
			return types.DatabaseChangeDetail{}, err
		}
	}
	if req.RiskLevel != nil {
		item.RiskLevel, err = normalizeDatabaseRiskLevel(*req.RiskLevel)
		if err != nil {
			return types.DatabaseChangeDetail{}, err
		}
	}
	if req.LockRisk != nil {
		item.LockRisk = *req.LockRisk
	}
	if req.ManualApprovalRequired != nil {
		item.ManualApprovalRequired = *req.ManualApprovalRequired
	}
	item.ManualApprovalRequired = item.ManualApprovalRequired || databaseChangeRequiresManualApproval(environment, item.OperationType, item.Reversibility, item.LockRisk)
	if req.Status != nil {
		item.Status = normalizeDatabaseChangeStatus(*req.Status)
		if _, ok := allowedDatabaseChangeStatuses[item.Status]; !ok {
			return types.DatabaseChangeDetail{}, fmt.Errorf("%w: unsupported database change status %q", ErrValidation, *req.Status)
		}
	}
	if req.Summary != nil {
		item.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.Evidence != nil {
		item.Evidence = normalizeDatabaseEvidence(*req.Evidence)
	}
	if req.Metadata != nil {
		item.Metadata = req.Metadata
	}
	if item.Name == "" || item.Datastore == "" || item.Summary == "" {
		return types.DatabaseChangeDetail{}, fmt.Errorf("%w: name, datastore, and summary are required", ErrValidation)
	}
	item.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateDatabaseChange(ctx, item); err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	if err := a.record(ctx, identity, "database_change.updated", "database_change", item.ID, item.OrganizationID, item.ProjectID, []string{item.Name, item.Status, item.OperationType}); err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	return a.buildDatabaseChangeDetail(ctx, item)
}

func (a *Application) ListDatabaseValidationChecks(ctx context.Context) ([]types.DatabaseValidationCheck, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListDatabaseValidationChecks(ctx, storage.DatabaseValidationCheckQuery{OrganizationID: orgID, Limit: 500})
}

func (a *Application) CreateDatabaseValidationCheck(ctx context.Context, req types.CreateDatabaseValidationCheckRequest) (types.DatabaseValidationCheckDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	_, environment, changeSet, service, team, err := a.validateDatabaseGovernanceScope(ctx, req.OrganizationID, req.ProjectID, req.EnvironmentID, req.ServiceID, req.ChangeSetID)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	if !a.canManageConfigSet(identity, req.OrganizationID, req.ProjectID, service, team) {
		return types.DatabaseValidationCheckDetail{}, a.forbidden(ctx, identity, "database_check.create.denied", "database_validation_check", "", req.OrganizationID, req.ProjectID, []string{"actor lacks database governance mutation permission"})
	}

	phase, err := normalizeDatabaseCheckPhase(req.Phase)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	checkType, err := normalizeDatabaseCheckType(req.CheckType)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	executionMode, err := normalizeDatabaseCheckMode(req.ExecutionMode)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	status, err := normalizeDatabaseCheckStatus(req.Status)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	var databaseChange *types.DatabaseChange
	if strings.TrimSpace(req.DatabaseChangeID) != "" {
		item, err := a.Store.GetDatabaseChange(ctx, strings.TrimSpace(req.DatabaseChangeID))
		if err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
		if item.OrganizationID != req.OrganizationID || item.ProjectID != req.ProjectID || item.EnvironmentID != environment.ID || item.ChangeSetID != changeSet.ID {
			return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: database change scope mismatch", ErrValidation)
		}
		databaseChange = &item
	}
	var connectionRef *types.DatabaseConnectionReference
	if strings.TrimSpace(req.ConnectionRefID) != "" {
		item, err := a.Store.GetDatabaseConnectionReference(ctx, strings.TrimSpace(req.ConnectionRefID))
		if err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
		if item.OrganizationID != req.OrganizationID || item.ProjectID != req.ProjectID || item.EnvironmentID != environment.ID {
			return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: database connection reference scope mismatch", ErrValidation)
		}
		if item.ServiceID != "" && item.ServiceID != changeSet.ServiceID {
			return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: database connection reference service scope mismatch", ErrValidation)
		}
		connectionRef = &item
	}

	now := time.Now().UTC()
	item := types.DatabaseValidationCheck{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("dbchk"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID: req.OrganizationID,
		ProjectID:      req.ProjectID,
		EnvironmentID:  environment.ID,
		ServiceID:      changeSet.ServiceID,
		ChangeSetID:    changeSet.ID,
		ConnectionRefID: strings.TrimSpace(req.ConnectionRefID),
		Name:           strings.TrimSpace(req.Name),
		Phase:          phase,
		CheckType:      checkType,
		ReadOnly:       req.ReadOnly || executionMode != "manual_attestation",
		Required:       req.Required,
		ExecutionMode:  executionMode,
		Specification:  strings.TrimSpace(req.Specification),
		Status:         status,
		Summary:        strings.TrimSpace(req.Summary),
		Evidence:       normalizeDatabaseEvidence(req.Evidence),
	}
	if databaseChange != nil {
		item.DatabaseChangeID = databaseChange.ID
	}
	if connectionRef != nil {
		item.ConnectionRefID = connectionRef.ID
	}
	if item.Name == "" || item.Specification == "" || item.Summary == "" {
		return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: name, specification, and summary are required", ErrValidation)
	}
	if item.ExecutionMode == "runtime_read_only" {
		if item.ConnectionRefID == "" {
			return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: runtime_read_only checks require connection_ref_id", ErrValidation)
		}
		if strings.TrimSpace(req.Status) != "" && item.Status != "defined" {
			return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: runtime_read_only checks cannot be created with terminal status %q", ErrValidation, req.Status)
		}
		if _, err := parseRuntimeDatabaseCheckSpec(item); err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
		item.Status = "defined"
		item.LastRunAt = nil
		item.LastResultSummary = ""
	}
	if item.Status == "passed" || item.Status == "failed" || item.Status == "blocked" {
		item.LastRunAt = &now
		item.LastResultSummary = item.Summary
	}
	if err := a.Store.CreateDatabaseValidationCheck(ctx, item); err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	if err := a.record(ctx, identity, "database_check.created", "database_validation_check", item.ID, item.OrganizationID, item.ProjectID, []string{item.Name, item.Phase, item.Status}); err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	return a.buildDatabaseValidationCheckDetail(ctx, item)
}

func (a *Application) GetDatabaseValidationCheckDetail(ctx context.Context, id string) (types.DatabaseValidationCheckDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	item, err := a.Store.GetDatabaseValidationCheck(ctx, id)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	if !a.Authorizer.CanReadProject(identity, item.OrganizationID, item.ProjectID) {
		return types.DatabaseValidationCheckDetail{}, ErrForbidden
	}
	return a.buildDatabaseValidationCheckDetail(ctx, item)
}

func (a *Application) UpdateDatabaseValidationCheck(ctx context.Context, id string, req types.UpdateDatabaseValidationCheckRequest) (types.DatabaseValidationCheckDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	item, err := a.Store.GetDatabaseValidationCheck(ctx, id)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	service, team, err := a.databaseGovernanceServiceTeam(ctx, item.ServiceID)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	if !a.canManageConfigSet(identity, item.OrganizationID, item.ProjectID, service, team) {
		return types.DatabaseValidationCheckDetail{}, a.forbidden(ctx, identity, "database_check.update.denied", "database_validation_check", item.ID, item.OrganizationID, item.ProjectID, []string{"actor lacks database governance mutation permission"})
	}

	if req.DatabaseChangeID != nil {
		item.DatabaseChangeID = strings.TrimSpace(*req.DatabaseChangeID)
		if item.DatabaseChangeID != "" {
			databaseChange, err := a.Store.GetDatabaseChange(ctx, item.DatabaseChangeID)
			if err != nil {
				return types.DatabaseValidationCheckDetail{}, err
			}
			if databaseChange.OrganizationID != item.OrganizationID || databaseChange.ProjectID != item.ProjectID || databaseChange.EnvironmentID != item.EnvironmentID || databaseChange.ChangeSetID != item.ChangeSetID {
				return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: database change scope mismatch", ErrValidation)
			}
		}
	}
	if req.ConnectionRefID != nil {
		item.ConnectionRefID = strings.TrimSpace(*req.ConnectionRefID)
		if item.ConnectionRefID != "" {
			connectionRef, err := a.Store.GetDatabaseConnectionReference(ctx, item.ConnectionRefID)
			if err != nil {
				return types.DatabaseValidationCheckDetail{}, err
			}
			if connectionRef.OrganizationID != item.OrganizationID || connectionRef.ProjectID != item.ProjectID || connectionRef.EnvironmentID != item.EnvironmentID {
				return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: database connection reference scope mismatch", ErrValidation)
			}
			if connectionRef.ServiceID != "" && connectionRef.ServiceID != item.ServiceID {
				return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: database connection reference service scope mismatch", ErrValidation)
			}
		}
	}
	if req.Name != nil {
		item.Name = strings.TrimSpace(*req.Name)
	}
	if req.Phase != nil {
		item.Phase, err = normalizeDatabaseCheckPhase(*req.Phase)
		if err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
	}
	if req.CheckType != nil {
		item.CheckType, err = normalizeDatabaseCheckType(*req.CheckType)
		if err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
	}
	if req.ReadOnly != nil {
		item.ReadOnly = *req.ReadOnly
	}
	if req.Required != nil {
		item.Required = *req.Required
	}
	if req.ExecutionMode != nil {
		item.ExecutionMode, err = normalizeDatabaseCheckMode(*req.ExecutionMode)
		if err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
	}
	if req.Specification != nil {
		item.Specification = strings.TrimSpace(*req.Specification)
	}
	statusChanged := false
	if req.Status != nil {
		if item.ExecutionMode == "runtime_read_only" {
			return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: runtime_read_only checks cannot be updated via manual status changes", ErrValidation)
		}
		item.Status, err = normalizeDatabaseCheckStatus(*req.Status)
		if err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
		statusChanged = true
	}
	if req.Summary != nil {
		item.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.LastRunAt != nil {
		if item.ExecutionMode == "runtime_read_only" {
			return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: runtime_read_only checks cannot set last_run_at manually", ErrValidation)
		}
		item.LastRunAt = req.LastRunAt
	}
	if req.LastResultSummary != nil {
		if item.ExecutionMode == "runtime_read_only" {
			return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: runtime_read_only checks cannot set last_result_summary manually", ErrValidation)
		}
		item.LastResultSummary = strings.TrimSpace(*req.LastResultSummary)
	}
	if req.Evidence != nil {
		item.Evidence = normalizeDatabaseEvidence(*req.Evidence)
	}
	if req.Metadata != nil {
		item.Metadata = req.Metadata
	}
	if item.ExecutionMode != "manual_attestation" {
		item.ReadOnly = true
	}
	if item.ExecutionMode == "runtime_read_only" {
		if item.ConnectionRefID == "" {
			return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: runtime_read_only checks require connection_ref_id", ErrValidation)
		}
		if _, err := parseRuntimeDatabaseCheckSpec(item); err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
		if req.ExecutionMode != nil {
			item.Status = "defined"
			item.LastRunAt = nil
			item.LastResultSummary = ""
		}
	}
	if statusChanged && item.LastRunAt == nil && (item.Status == "passed" || item.Status == "failed" || item.Status == "blocked") {
		now := time.Now().UTC()
		item.LastRunAt = &now
	}
	if item.Name == "" || item.Specification == "" || item.Summary == "" {
		return types.DatabaseValidationCheckDetail{}, fmt.Errorf("%w: name, specification, and summary are required", ErrValidation)
	}
	item.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateDatabaseValidationCheck(ctx, item); err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	if err := a.record(ctx, identity, "database_check.updated", "database_validation_check", item.ID, item.OrganizationID, item.ProjectID, []string{item.Name, item.Status, item.Phase}); err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	return a.buildDatabaseValidationCheckDetail(ctx, item)
}

func (a *Application) buildDatabaseChangeDetail(ctx context.Context, item types.DatabaseChange) (types.DatabaseChangeDetail, error) {
	checks, err := a.Store.ListDatabaseValidationChecks(ctx, storage.DatabaseValidationCheckQuery{
		OrganizationID:   item.OrganizationID,
		ProjectID:        item.ProjectID,
		EnvironmentID:    item.EnvironmentID,
		DatabaseChangeID: item.ID,
		Limit:            200,
	})
	if err != nil {
		return types.DatabaseChangeDetail{}, err
	}
	return types.DatabaseChangeDetail{
		DatabaseChange:   item,
		ValidationChecks: checks,
	}, nil
}

func (a *Application) buildDatabaseValidationCheckDetail(ctx context.Context, item types.DatabaseValidationCheck) (types.DatabaseValidationCheckDetail, error) {
	result := types.DatabaseValidationCheckDetail{ValidationCheck: item}
	if strings.TrimSpace(item.DatabaseChangeID) == "" {
		if strings.TrimSpace(item.ConnectionRefID) != "" {
			connectionRef, err := a.Store.GetDatabaseConnectionReference(ctx, item.ConnectionRefID)
			if err != nil {
				return types.DatabaseValidationCheckDetail{}, err
			}
			result.ConnectionReference = &connectionRef
		}
		executions, err := a.Store.ListDatabaseValidationExecutions(ctx, storage.DatabaseValidationExecutionQuery{
			OrganizationID:    item.OrganizationID,
			ProjectID:         item.ProjectID,
			ValidationCheckID: item.ID,
			Limit:             20,
		})
		if err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
		result.Executions = executions
		return result, nil
	}
	databaseChange, err := a.Store.GetDatabaseChange(ctx, item.DatabaseChangeID)
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	result.DatabaseChange = &databaseChange
	if strings.TrimSpace(item.ConnectionRefID) != "" {
		connectionRef, err := a.Store.GetDatabaseConnectionReference(ctx, item.ConnectionRefID)
		if err != nil {
			return types.DatabaseValidationCheckDetail{}, err
		}
		result.ConnectionReference = &connectionRef
	}
	executions, err := a.Store.ListDatabaseValidationExecutions(ctx, storage.DatabaseValidationExecutionQuery{
		OrganizationID:    item.OrganizationID,
		ProjectID:         item.ProjectID,
		ValidationCheckID: item.ID,
		Limit:             20,
	})
	if err != nil {
		return types.DatabaseValidationCheckDetail{}, err
	}
	result.Executions = executions
	return result, nil
}

func (a *Application) validateDatabaseGovernanceScope(ctx context.Context, organizationID, projectID, environmentID, serviceID, changeSetID string) (types.Project, types.Environment, types.ChangeSet, *types.Service, *types.Team, error) {
	project, environment, err := a.validateProjectEnvironment(ctx, organizationID, projectID, environmentID)
	if err != nil {
		return types.Project{}, types.Environment{}, types.ChangeSet{}, nil, nil, err
	}
	if strings.TrimSpace(changeSetID) == "" {
		return types.Project{}, types.Environment{}, types.ChangeSet{}, nil, nil, fmt.Errorf("%w: change_set_id is required", ErrValidation)
	}
	changeSet, err := a.Store.GetChangeSet(ctx, strings.TrimSpace(changeSetID))
	if err != nil {
		return types.Project{}, types.Environment{}, types.ChangeSet{}, nil, nil, err
	}
	if changeSet.OrganizationID != organizationID || changeSet.ProjectID != projectID || changeSet.EnvironmentID != environmentID {
		return types.Project{}, types.Environment{}, types.ChangeSet{}, nil, nil, fmt.Errorf("%w: change set scope mismatch", ErrValidation)
	}
	effectiveServiceID := changeSet.ServiceID
	if trimmed := strings.TrimSpace(serviceID); trimmed != "" && trimmed != changeSet.ServiceID {
		return types.Project{}, types.Environment{}, types.ChangeSet{}, nil, nil, fmt.Errorf("%w: service scope mismatch", ErrValidation)
	}
	service, err := a.Store.GetService(ctx, effectiveServiceID)
	if err != nil {
		return types.Project{}, types.Environment{}, types.ChangeSet{}, nil, nil, err
	}
	team, err := a.Store.GetTeam(ctx, service.TeamID)
	if err != nil {
		return types.Project{}, types.Environment{}, types.ChangeSet{}, nil, nil, err
	}
	return project, environment, changeSet, &service, &team, nil
}

func (a *Application) databaseGovernanceServiceTeam(ctx context.Context, serviceID string) (*types.Service, *types.Team, error) {
	if strings.TrimSpace(serviceID) == "" {
		return nil, nil, nil
	}
	service, err := a.Store.GetService(ctx, serviceID)
	if err != nil {
		return nil, nil, err
	}
	team, err := a.Store.GetTeam(ctx, service.TeamID)
	if err != nil {
		return nil, nil, err
	}
	return &service, &team, nil
}

func (a *Application) buildDatabaseGovernanceSnapshot(ctx context.Context, organizationID, projectID string, environment types.Environment, bundleChanges []types.ChangeSet) (databaseGovernanceSnapshot, error) {
	changeSetIDs := make(map[string]types.ChangeSet, len(bundleChanges))
	fallbackSchemaCount := 0
	for _, change := range bundleChanges {
		changeSetIDs[change.ID] = change
		if change.TouchesSchema {
			fallbackSchemaCount++
		}
	}

	items, err := a.Store.ListDatabaseChanges(ctx, storage.DatabaseChangeQuery{
		OrganizationID: organizationID,
		ProjectID:      projectID,
		EnvironmentID:  environment.ID,
		Limit:          500,
	})
	if err != nil {
		return databaseGovernanceSnapshot{}, err
	}
	databaseChanges := make([]types.DatabaseChange, 0, len(items))
	changeIDs := make(map[string]struct{}, len(items))
	for _, item := range items {
		if _, ok := changeSetIDs[item.ChangeSetID]; !ok {
			continue
		}
		databaseChanges = append(databaseChanges, item)
		changeIDs[item.ID] = struct{}{}
	}
	sort.Slice(databaseChanges, func(i, j int) bool {
		if databaseChanges[i].CreatedAt.Equal(databaseChanges[j].CreatedAt) {
			return databaseChanges[i].ID < databaseChanges[j].ID
		}
		return databaseChanges[i].CreatedAt.Before(databaseChanges[j].CreatedAt)
	})

	checkItems, err := a.Store.ListDatabaseValidationChecks(ctx, storage.DatabaseValidationCheckQuery{
		OrganizationID: organizationID,
		ProjectID:      projectID,
		EnvironmentID:  environment.ID,
		Limit:          1000,
	})
	if err != nil {
		return databaseGovernanceSnapshot{}, err
	}
	checks := make([]types.DatabaseValidationCheck, 0, len(checkItems))
	for _, item := range checkItems {
		if _, ok := changeSetIDs[item.ChangeSetID]; ok {
			checks = append(checks, item)
			continue
		}
		if _, ok := changeIDs[item.DatabaseChangeID]; ok {
			checks = append(checks, item)
		}
	}
	sort.Slice(checks, func(i, j int) bool {
		if checks[i].Phase == checks[j].Phase {
			if checks[i].Name == checks[j].Name {
				return checks[i].ID < checks[j].ID
			}
			return checks[i].Name < checks[j].Name
		}
		return checks[i].Phase < checks[j].Phase
	})

	connectionRefIDs := make(map[string]struct{}, len(checks))
	checkIDs := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		checkIDs[check.ID] = struct{}{}
		if strings.TrimSpace(check.ConnectionRefID) != "" {
			connectionRefIDs[check.ConnectionRefID] = struct{}{}
		}
	}
	connectionItems, err := a.Store.ListDatabaseConnectionReferences(ctx, storage.DatabaseConnectionReferenceQuery{
		OrganizationID: organizationID,
		ProjectID:      projectID,
		EnvironmentID:  environment.ID,
		Limit:          500,
	})
	if err != nil {
		return databaseGovernanceSnapshot{}, err
	}
	connections := make([]types.DatabaseConnectionReference, 0, len(connectionItems))
	connectionsByID := make(map[string]types.DatabaseConnectionReference, len(connectionItems))
	for _, item := range connectionItems {
		if _, ok := connectionRefIDs[item.ID]; !ok {
			continue
		}
		connections = append(connections, item)
		connectionsByID[item.ID] = item
	}
	connectionTestItems, err := a.Store.ListDatabaseConnectionTests(ctx, storage.DatabaseConnectionTestQuery{
		OrganizationID: organizationID,
		ProjectID:      projectID,
		EnvironmentID:  environment.ID,
		Limit:          1000,
	})
	if err != nil {
		return databaseGovernanceSnapshot{}, err
	}
	connectionTests := make([]types.DatabaseConnectionTest, 0, len(connectionTestItems))
	for _, item := range connectionTestItems {
		if _, ok := connectionRefIDs[item.ConnectionRefID]; !ok {
			continue
		}
		connectionTests = append(connectionTests, item)
	}
	executionItems, err := a.Store.ListDatabaseValidationExecutions(ctx, storage.DatabaseValidationExecutionQuery{
		OrganizationID: organizationID,
		ProjectID:      projectID,
		EnvironmentID:  environment.ID,
		Limit:          1000,
	})
	if err != nil {
		return databaseGovernanceSnapshot{}, err
	}
	executions := make([]types.DatabaseValidationExecution, 0, len(executionItems))
	for _, item := range executionItems {
		if _, ok := checkIDs[item.ValidationCheckID]; !ok {
			continue
		}
		executions = append(executions, item)
	}
	latestExecutions := latestDatabaseExecutionByCheck(executions)
	latestConnectionTests := latestDatabaseConnectionTestByReference(connectionTests)

	checksByDatabaseChangeID := make(map[string][]types.DatabaseValidationCheck)
	checksByChangeSetID := make(map[string][]types.DatabaseValidationCheck)
	requiredCheckCount := 0
	pendingCheckCount := 0
	blockers := make([]string, 0, 6)
	warnings := make([]string, 0, 6)
	findings := make([]string, 0, 8)
	policyHighlights := make([]string, 0, 6)
	compatibilityRank := "none"
	rollbackSafety := "standard"
	manualApprovalRequired := false

	for _, check := range checks {
		if check.DatabaseChangeID != "" {
			checksByDatabaseChangeID[check.DatabaseChangeID] = append(checksByDatabaseChangeID[check.DatabaseChangeID], check)
		}
		checksByChangeSetID[check.ChangeSetID] = append(checksByChangeSetID[check.ChangeSetID], check)
		if check.Required {
			requiredCheckCount++
			if databaseCheckIsPending(check) {
				pendingCheckCount++
			}
			switch check.Phase {
			case "pre_deploy":
				if check.Status != "passed" {
					blockers = append(blockers, fmt.Sprintf("required pre-deploy database check %s is %s", check.Name, check.Status))
				}
			case "post_deploy", "rollback":
				if check.Status == "failed" || check.Status == "blocked" {
					blockers = append(blockers, fmt.Sprintf("required %s database check %s is %s", strings.ReplaceAll(check.Phase, "_", "-"), check.Name, check.Status))
				} else if check.Status != "passed" {
					warnings = append(warnings, fmt.Sprintf("required %s database check %s remains %s", strings.ReplaceAll(check.Phase, "_", "-"), check.Name, check.Status))
				}
			}
		}
		if check.ExecutionMode == "runtime_read_only" {
			if check.ConnectionRefID == "" {
				blockers = append(blockers, fmt.Sprintf("runtime database check %s does not reference a database connection", check.Name))
			} else if connectionRef, ok := connectionsByID[check.ConnectionRefID]; !ok {
				blockers = append(blockers, fmt.Sprintf("runtime database check %s references an unavailable database connection", check.Name))
			} else {
				findings = append(findings, fmt.Sprintf("runtime database check %s uses connection %s via %s", check.Name, connectionRef.Name, databaseConnectionSourceSummary(connectionRef)))
				runtimeCapabilityStatus, runtimeCapabilityErrorClass, runtimeCapabilitySummary := a.databaseConnectionRuntimeCapability(connectionRef)
				switch runtimeCapabilityStatus {
				case "unsupported":
					if connectionRef.Status != "unsupported" {
						blockers = append(blockers, fmt.Sprintf("database connection %s uses unsupported runtime source posture %s", connectionRef.Name, databaseConnectionSourceSummary(connectionRef)))
					}
				case "unresolved":
					if connectionRef.Status != "unresolved" {
						blockers = append(blockers, fmt.Sprintf("database connection %s cannot currently resolve runtime access: %s", connectionRef.Name, runtimeCapabilitySummary))
					}
					if runtimeCapabilityErrorClass != "" {
						findings = append(findings, fmt.Sprintf("database connection %s runtime capability is %s", connectionRef.Name, runtimeCapabilityErrorClass))
					}
				}
				switch connectionRef.Status {
				case "unsupported":
					blockers = append(blockers, fmt.Sprintf("database connection %s uses unsupported source posture %s", connectionRef.Name, databaseConnectionSourceSummary(connectionRef)))
				case "unresolved":
					blockers = append(blockers, fmt.Sprintf("database connection %s is unresolved and cannot execute runtime validation", connectionRef.Name))
				case "error":
					blockers = append(blockers, fmt.Sprintf("database connection %s last failed health verification", connectionRef.Name))
				case "defined":
					if check.Required && check.Phase == "pre_deploy" {
						blockers = append(blockers, fmt.Sprintf("required runtime database check %s depends on untested connection %s", check.Name, connectionRef.Name))
					} else {
						warnings = append(warnings, fmt.Sprintf("database connection %s has not been health-tested yet", connectionRef.Name))
					}
				case "testing":
					warnings = append(warnings, fmt.Sprintf("database connection %s is currently testing and should be confirmed before rollout", connectionRef.Name))
				}
				if latestTest, ok := latestConnectionTests[connectionRef.ID]; ok {
					findings = append(findings, fmt.Sprintf("database connection %s last tested with %s status", connectionRef.Name, latestTest.Status))
					if latestTest.Status != "passed" {
						if check.Required && check.Phase == "pre_deploy" {
							blockers = append(blockers, fmt.Sprintf("required runtime database check %s depends on connection test status %s for %s", check.Name, latestTest.Status, connectionRef.Name))
						} else {
							warnings = append(warnings, fmt.Sprintf("database connection %s last tested with %s status", connectionRef.Name, latestTest.Status))
						}
					}
				} else {
					if check.Required && check.Phase == "pre_deploy" {
						blockers = append(blockers, fmt.Sprintf("required runtime database check %s has no persisted connection health evidence for %s", check.Name, connectionRef.Name))
					} else {
						warnings = append(warnings, fmt.Sprintf("database connection %s has no persisted health-test evidence yet", connectionRef.Name))
					}
				}
			}
			if execution, ok := latestExecutions[check.ID]; ok {
				findings = append(findings, fmt.Sprintf("runtime database check %s last executed with %s status", check.Name, execution.Status))
			} else {
				warnings = append(warnings, fmt.Sprintf("runtime database check %s has no persisted execution evidence yet", check.Name))
			}
		}
	}

	for _, change := range databaseChanges {
		findings = append(findings, fmt.Sprintf("%s targets %s as %s with %s compatibility and %s reversibility", change.Name, change.Datastore, strings.ReplaceAll(change.OperationType, "_", " "), strings.ReplaceAll(change.Compatibility, "_", " "), strings.ReplaceAll(change.Reversibility, "_", " ")))
		compatibilityRank = databaseCompatibilityRank(compatibilityRank, change.Compatibility)
		if change.LockRisk {
			warnings = append(warnings, fmt.Sprintf("database change %s may require exclusive locks or elevated migration supervision", change.Name))
		}
		if change.ManualApprovalRequired {
			manualApprovalRequired = true
			policyHighlights = append(policyHighlights, fmt.Sprintf("database governance requires manual approval for %s", change.Name))
			warnings = append(warnings, fmt.Sprintf("manual database approval is required for %s", change.Name))
		}
		if change.Reversibility == "irreversible" {
			rollbackSafety = "unsafe"
			policyHighlights = append(policyHighlights, fmt.Sprintf("database change %s is irreversible, so fix-forward planning is required", change.Name))
			warnings = append(warnings, fmt.Sprintf("database change %s is irreversible and should prefer fix-forward recovery", change.Name))
		}
		if change.Reversibility == "manual_only" && rollbackSafety != "unsafe" {
			rollbackSafety = "manual_review"
			warnings = append(warnings, fmt.Sprintf("database change %s needs manual rollback coordination", change.Name))
		}
		if change.Compatibility == "forward_incompatible" {
			if rollbackSafety != "unsafe" {
				rollbackSafety = "manual_review"
			}
			warnings = append(warnings, fmt.Sprintf("database change %s is forward-incompatible and needs compatibility sequencing review", change.Name))
			policyHighlights = append(policyHighlights, fmt.Sprintf("forward-incompatible database change %s should not deploy without explicit compatibility review", change.Name))
		}

		linkedChecks := append([]types.DatabaseValidationCheck{}, checksByDatabaseChangeID[change.ID]...)
		linkedChecks = append(linkedChecks, checksByChangeSetID[change.ChangeSetID]...)
		if databaseChangeNeedsPreDeployCheck(change) && !databaseChangeHasRequiredCheck(linkedChecks, "pre_deploy") {
			blockers = append(blockers, fmt.Sprintf("database change %s is missing a required pre-deploy validation check", change.Name))
		}
		if databaseChangeNeedsPostDeployCheck(change) && !databaseChangeHasRequiredCheck(linkedChecks, "post_deploy") {
			warnings = append(warnings, fmt.Sprintf("database change %s is missing a required post-deploy validation check", change.Name))
		}
	}

	changeCount := len(databaseChanges)
	if changeCount == 0 && fallbackSchemaCount > 0 {
		changeCount = fallbackSchemaCount
		findings = append(findings, fmt.Sprintf("%d schema-affecting change set(s) were detected heuristically without persisted database evidence", fallbackSchemaCount))
		if compatibilityRank == "none" {
			compatibilityRank = "schema_affecting"
		}
		if rollbackSafety == "standard" {
			rollbackSafety = "manual_review"
		}
		warnings = append(warnings, "schema-affecting changes exist without fully classified database governance records")
	}
	if changeCount == 0 {
		return databaseGovernanceSnapshot{
			Connections:     connections,
			ConnectionTests: connectionTests,
			Changes:         databaseChanges,
			Checks:          checks,
			Executions:      executions,
			Findings:        nil,
			Posture: types.DatabasePosture{
				Status:         "none",
				Summary:        "No database-specific deployment governance records are attached to this release.",
				Compatibility:  "none",
				RollbackSafety: "standard",
			},
		}, nil
	}

	status := "advisory"
	switch {
	case len(blockers) > 0:
		status = "blocked"
	case manualApprovalRequired || len(warnings) > 0 || pendingCheckCount > 0:
		status = "review_required"
	}
	if rollbackSafety == "standard" && (manualApprovalRequired || len(warnings) > 0) {
		rollbackSafety = "guarded"
	}
	posture := types.DatabasePosture{
		Status:                 status,
		Summary:                summarizeDatabasePosture(changeCount, len(checks), status, compatibilityRank, rollbackSafety),
		Compatibility:          compatibilityRank,
		RollbackSafety:         rollbackSafety,
		ManualApprovalRequired: manualApprovalRequired,
		ChangeCount:            changeCount,
		RequiredCheckCount:     requiredCheckCount,
		PendingCheckCount:      pendingCheckCount,
		BlockingFindings:       dedupeStrings(blockers),
		WarningFindings:        dedupeStrings(warnings),
	}
	return databaseGovernanceSnapshot{
		Connections:      connections,
		ConnectionTests:  connectionTests,
		Changes:          databaseChanges,
		Checks:           checks,
		Executions:       executions,
		Findings:         dedupeStrings(findings),
		PolicyHighlights: dedupeStrings(policyHighlights),
		Warnings:         posture.WarningFindings,
		Blockers:         posture.BlockingFindings,
		Posture:          posture,
	}, nil
}

func normalizeDatabaseOperationType(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if _, ok := allowedDatabaseOperationTypes[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database operation_type %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseExecutionIntent(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if _, ok := allowedDatabaseExecutionIntents[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database execution_intent %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseCompatibility(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if _, ok := allowedDatabaseCompatibility[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database compatibility %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseReversibility(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if _, ok := allowedDatabaseReversibility[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database reversibility %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseRiskLevel(value types.RiskLevel) (types.RiskLevel, error) {
	if value == "" {
		return types.RiskLevelMedium, nil
	}
	switch strings.TrimSpace(strings.ToLower(string(value))) {
	case string(types.RiskLevelLow):
		return types.RiskLevelLow, nil
	case string(types.RiskLevelMedium):
		return types.RiskLevelMedium, nil
	case string(types.RiskLevelHigh):
		return types.RiskLevelHigh, nil
	case string(types.RiskLevelCritical):
		return types.RiskLevelCritical, nil
	default:
		return "", fmt.Errorf("%w: unsupported database risk_level %q", ErrValidation, value)
	}
}

func normalizeDatabaseChangeStatus(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return "defined"
	}
	return normalized
}

func normalizeDatabaseCheckPhase(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if _, ok := allowedDatabaseCheckPhases[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database check phase %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseCheckType(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if _, ok := allowedDatabaseCheckTypes[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database check type %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseCheckMode(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return "manual_attestation", nil
	}
	if _, ok := allowedDatabaseCheckModes[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database check execution_mode %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseCheckStatus(value string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return "defined", nil
	}
	if _, ok := allowedDatabaseCheckStatuses[normalized]; !ok {
		return "", fmt.Errorf("%w: unsupported database check status %q", ErrValidation, value)
	}
	return normalized, nil
}

func normalizeDatabaseEvidence(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return dedupeStrings(normalized)
}

func databaseChangeRequiresManualApproval(environment types.Environment, operationType, reversibility string, lockRisk bool) bool {
	if !environment.Production {
		return false
	}
	return lockRisk || operationType == "destructive_change" || reversibility == "irreversible"
}

func databaseCheckIsPending(check types.DatabaseValidationCheck) bool {
	return check.Status == "defined" || check.Status == "advisory_only"
}

func databaseChangeNeedsPreDeployCheck(change types.DatabaseChange) bool {
	return change.OperationType == "schema_change" || change.OperationType == "destructive_change" || change.Compatibility == "forward_incompatible" || change.LockRisk
}

func databaseChangeNeedsPostDeployCheck(change types.DatabaseChange) bool {
	return change.OperationType == "schema_change" || change.OperationType == "data_backfill" || change.OperationType == "expand_contract" || change.OperationType == "destructive_change"
}

func databaseChangeHasRequiredCheck(checks []types.DatabaseValidationCheck, phase string) bool {
	for _, check := range checks {
		if check.Required && check.Phase == phase {
			return true
		}
	}
	return false
}

func databaseCompatibilityRank(current, candidate string) string {
	rank := map[string]int{
		"none":                 0,
		"backward_compatible":  1,
		"expand_contract":      2,
		"schema_affecting":     3,
		"forward_incompatible": 4,
	}
	if rank[candidate] > rank[current] {
		return candidate
	}
	return current
}

func summarizeDatabasePosture(changeCount, checkCount int, status, compatibility, rollbackSafety string) string {
	return fmt.Sprintf("%d database change(s), %d validation check(s), posture %s, compatibility %s, rollback %s", changeCount, checkCount, status, compatibility, rollbackSafety)
}
