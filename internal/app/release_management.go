package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/common"
	policylib "github.com/change-control-plane/change-control-plane/internal/policies"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) ListConfigSets(ctx context.Context) ([]types.ConfigSet, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListConfigSets(ctx, storage.ConfigSetQuery{OrganizationID: orgID})
}

func (a *Application) CreateConfigSet(ctx context.Context, req types.CreateConfigSetRequest) (types.ConfigSetDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	project, environment, service, team, err := a.validateConfigSetScope(ctx, req.OrganizationID, req.ProjectID, req.EnvironmentID, req.ServiceID)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	if !a.canManageConfigSet(identity, req.OrganizationID, req.ProjectID, service, team) {
		return types.ConfigSetDetail{}, a.forbidden(ctx, identity, "config_set.create.denied", "config_set", "", req.OrganizationID, req.ProjectID, []string{"actor lacks config set mutation permission"})
	}

	now := time.Now().UTC()
	configSet := types.ConfigSet{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("cfg"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID: req.OrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		ServiceID:      strings.TrimSpace(req.ServiceID),
		Name:           strings.TrimSpace(req.Name),
		Version:        strings.TrimSpace(req.Version),
		Status:         "active",
		Entries:        normalizeConfigEntries(req.Entries),
	}
	validation, err := a.buildConfigSetValidation(ctx, configSet)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	if configSetValidationBlocked(validation) {
		return types.ConfigSetDetail{}, fmt.Errorf("%w: config set validation blocked due to %s", ErrValidation, strings.Join(blockingValidationReasons(validation), ", "))
	}
	if err := a.Store.CreateConfigSet(ctx, configSet); err != nil {
		return types.ConfigSetDetail{}, err
	}
	if err := a.record(ctx, identity, "config_set.created", "config_set", configSet.ID, configSet.OrganizationID, configSet.ProjectID, []string{configSet.Name, configSet.Version}); err != nil {
		return types.ConfigSetDetail{}, err
	}
	return a.buildConfigSetDetail(ctx, configSet)
}

func (a *Application) GetConfigSetDetail(ctx context.Context, id string) (types.ConfigSetDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	configSet, err := a.Store.GetConfigSet(ctx, id)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	if !a.Authorizer.CanReadProject(identity, configSet.OrganizationID, configSet.ProjectID) {
		return types.ConfigSetDetail{}, ErrForbidden
	}
	return a.buildConfigSetDetail(ctx, configSet)
}

func (a *Application) UpdateConfigSet(ctx context.Context, id string, req types.UpdateConfigSetRequest) (types.ConfigSetDetail, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	configSet, err := a.Store.GetConfigSet(ctx, id)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	var service *types.Service
	var team *types.Team
	if strings.TrimSpace(configSet.ServiceID) != "" {
		loadedService, err := a.Store.GetService(ctx, configSet.ServiceID)
		if err != nil {
			return types.ConfigSetDetail{}, err
		}
		loadedTeam, err := a.Store.GetTeam(ctx, loadedService.TeamID)
		if err != nil {
			return types.ConfigSetDetail{}, err
		}
		service = &loadedService
		team = &loadedTeam
	}
	if !a.canManageConfigSet(identity, configSet.OrganizationID, configSet.ProjectID, service, team) {
		return types.ConfigSetDetail{}, a.forbidden(ctx, identity, "config_set.update.denied", "config_set", configSet.ID, configSet.OrganizationID, configSet.ProjectID, []string{"actor lacks config set mutation permission"})
	}

	if req.Name != nil {
		configSet.Name = strings.TrimSpace(*req.Name)
	}
	if req.Version != nil {
		configSet.Version = strings.TrimSpace(*req.Version)
	}
	if req.Status != nil {
		configSet.Status = strings.TrimSpace(*req.Status)
	}
	if req.Entries != nil {
		configSet.Entries = normalizeConfigEntries(*req.Entries)
	}
	if req.Metadata != nil {
		configSet.Metadata = req.Metadata
	}
	configSet.UpdatedAt = time.Now().UTC()

	validation, err := a.buildConfigSetValidation(ctx, configSet)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	if configSetValidationBlocked(validation) {
		return types.ConfigSetDetail{}, fmt.Errorf("%w: config set validation blocked due to %s", ErrValidation, strings.Join(blockingValidationReasons(validation), ", "))
	}
	if err := a.Store.UpdateConfigSet(ctx, configSet); err != nil {
		return types.ConfigSetDetail{}, err
	}
	if err := a.record(ctx, identity, "config_set.updated", "config_set", configSet.ID, configSet.OrganizationID, configSet.ProjectID, []string{configSet.Name, configSet.Version, configSet.Status}); err != nil {
		return types.ConfigSetDetail{}, err
	}
	return a.buildConfigSetDetail(ctx, configSet)
}

func (a *Application) ListReleases(ctx context.Context) ([]types.Release, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	return a.Store.ListReleases(ctx, storage.ReleaseQuery{OrganizationID: orgID})
}

func (a *Application) CreateRelease(ctx context.Context, req types.CreateReleaseRequest) (types.ReleaseAnalysis, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	project, environment, err := a.validateProjectEnvironment(ctx, req.OrganizationID, req.ProjectID, req.EnvironmentID)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	if !a.Authorizer.CanManageProject(identity, req.OrganizationID, req.ProjectID) {
		return types.ReleaseAnalysis{}, a.forbidden(ctx, identity, "release.create.denied", "release", "", req.OrganizationID, req.ProjectID, []string{"actor lacks release mutation permission"})
	}
	changeSetIDs := trimDedupedIDs(req.ChangeSetIDs)
	if len(changeSetIDs) == 0 {
		return types.ReleaseAnalysis{}, fmt.Errorf("%w: at least one change_set_id is required", ErrValidation)
	}
	configSetIDs := trimDedupedIDs(req.ConfigSetIDs)
	if _, _, err := a.loadReleaseScope(ctx, req.OrganizationID, req.ProjectID, environment.ID, changeSetIDs, configSetIDs); err != nil {
		return types.ReleaseAnalysis{}, err
	}

	now := time.Now().UTC()
	release := types.Release{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("rel"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID: req.OrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		Name:           strings.TrimSpace(req.Name),
		Summary:        strings.TrimSpace(req.Summary),
		ChangeSetIDs:   changeSetIDs,
		ConfigSetIDs:   configSetIDs,
		Version:        strings.TrimSpace(req.Version),
		Status:         "draft",
	}
	if err := a.Store.CreateRelease(ctx, release); err != nil {
		return types.ReleaseAnalysis{}, err
	}
	if err := a.record(ctx, identity, "release.created", "release", release.ID, release.OrganizationID, release.ProjectID, []string{release.Name, release.Version}); err != nil {
		return types.ReleaseAnalysis{}, err
	}
	return a.buildReleaseAnalysis(ctx, release)
}

func (a *Application) GetReleaseAnalysis(ctx context.Context, id string) (types.ReleaseAnalysis, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	release, err := a.Store.GetRelease(ctx, id)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	if !a.Authorizer.CanReadProject(identity, release.OrganizationID, release.ProjectID) {
		return types.ReleaseAnalysis{}, ErrForbidden
	}
	return a.buildReleaseAnalysis(ctx, release)
}

func (a *Application) UpdateRelease(ctx context.Context, id string, req types.UpdateReleaseRequest) (types.ReleaseAnalysis, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	release, err := a.Store.GetRelease(ctx, id)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	if !a.Authorizer.CanManageProject(identity, release.OrganizationID, release.ProjectID) {
		return types.ReleaseAnalysis{}, a.forbidden(ctx, identity, "release.update.denied", "release", release.ID, release.OrganizationID, release.ProjectID, []string{"actor lacks release mutation permission"})
	}

	if req.Name != nil {
		release.Name = strings.TrimSpace(*req.Name)
	}
	if req.Summary != nil {
		release.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.ChangeSetIDs != nil {
		release.ChangeSetIDs = trimDedupedIDs(*req.ChangeSetIDs)
	}
	if req.ConfigSetIDs != nil {
		release.ConfigSetIDs = trimDedupedIDs(*req.ConfigSetIDs)
	}
	if req.Version != nil {
		release.Version = strings.TrimSpace(*req.Version)
	}
	if req.Status != nil {
		release.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		release.Metadata = req.Metadata
	}
	if len(release.ChangeSetIDs) == 0 {
		return types.ReleaseAnalysis{}, fmt.Errorf("%w: at least one change_set_id is required", ErrValidation)
	}
	if _, _, err := a.loadReleaseScope(ctx, release.OrganizationID, release.ProjectID, release.EnvironmentID, release.ChangeSetIDs, release.ConfigSetIDs); err != nil {
		return types.ReleaseAnalysis{}, err
	}
	release.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateRelease(ctx, release); err != nil {
		return types.ReleaseAnalysis{}, err
	}
	if err := a.record(ctx, identity, "release.updated", "release", release.ID, release.OrganizationID, release.ProjectID, []string{release.Name, release.Version, release.Status}); err != nil {
		return types.ReleaseAnalysis{}, err
	}
	return a.buildReleaseAnalysis(ctx, release)
}

func (a *Application) buildConfigSetDetail(ctx context.Context, configSet types.ConfigSet) (types.ConfigSetDetail, error) {
	validation, err := a.buildConfigSetValidation(ctx, configSet)
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	releases, err := a.Store.ListReleases(ctx, storage.ReleaseQuery{
		OrganizationID: configSet.OrganizationID,
		ProjectID:      configSet.ProjectID,
		EnvironmentID:  configSet.EnvironmentID,
	})
	if err != nil {
		return types.ConfigSetDetail{}, err
	}
	related := make([]types.Release, 0, len(releases))
	for _, release := range releases {
		if containsID(release.ConfigSetIDs, configSet.ID) {
			related = append(related, release)
		}
	}
	return types.ConfigSetDetail{
		ConfigSet:       configSet,
		Validation:      validation,
		RelatedReleases: related,
	}, nil
}

func (a *Application) buildConfigSetValidation(ctx context.Context, configSet types.ConfigSet) (types.ConfigSetValidation, error) {
	history, err := a.Store.ListConfigSets(ctx, storage.ConfigSetQuery{
		OrganizationID: configSet.OrganizationID,
		ProjectID:      configSet.ProjectID,
		EnvironmentID:  configSet.EnvironmentID,
		ServiceID:      configSet.ServiceID,
	})
	if err != nil {
		return types.ConfigSetValidation{}, err
	}
	validation := types.ConfigSetValidation{
		ConfigSetID: configSet.ID,
		Status:      "valid",
	}

	currentByKey := make(map[string]types.ConfigEntry, len(configSet.Entries))
	duplicateKeys := make([]string, 0, 2)
	for _, entry := range configSet.Entries {
		key := strings.TrimSpace(entry.Key)
		if key == "" {
			continue
		}
		if _, ok := currentByKey[key]; ok {
			duplicateKeys = append(duplicateKeys, key)
			continue
		}
		currentByKey[key] = entry
		if entry.Required && strings.TrimSpace(entry.Value) == "" {
			validation.MissingRequiredKeys = append(validation.MissingRequiredKeys, key)
		}
		if entry.Deprecated {
			validation.DeprecatedKeys = append(validation.DeprecatedKeys, key)
		}
		if strings.EqualFold(entry.ValueType, "secret_ref") {
			validation.SecretReferenceKeys = append(validation.SecretReferenceKeys, key)
			if !looksLikeSecretReference(entry.Value) {
				validation.InvalidSecretRefs = append(validation.InvalidSecretRefs, key)
			}
			continue
		}
		if entryLooksSensitive(key) {
			validation.InvalidSecretRefs = append(validation.InvalidSecretRefs, key)
		}
	}
	if len(duplicateKeys) > 0 {
		validation.Warnings = append(validation.Warnings, "duplicate keys detected: "+strings.Join(duplicateKeys, ", "))
	}

	var previous *types.ConfigSet
	for idx := len(history) - 1; idx >= 0; idx-- {
		candidate := history[idx]
		if candidate.ID == configSet.ID || candidate.Name != configSet.Name {
			continue
		}
		previous = &candidate
		break
	}
	if previous != nil {
		previousByKey := make(map[string]types.ConfigEntry, len(previous.Entries))
		for _, entry := range previous.Entries {
			previousByKey[strings.TrimSpace(entry.Key)] = entry
		}
		for key, entry := range currentByKey {
			previousEntry, ok := previousByKey[key]
			switch {
			case !ok:
				validation.DiffSummary = append(validation.DiffSummary, fmt.Sprintf("added key %s", key))
			case !strings.EqualFold(previousEntry.ValueType, entry.ValueType):
				validation.DiffSummary = append(validation.DiffSummary, fmt.Sprintf("key %s changed value type from %s to %s", key, previousEntry.ValueType, entry.ValueType))
			case strings.TrimSpace(previousEntry.Source) != strings.TrimSpace(entry.Source) && strings.TrimSpace(entry.Source) != "":
				validation.DiffSummary = append(validation.DiffSummary, fmt.Sprintf("key %s source changed to %s", key, entry.Source))
			case strings.TrimSpace(previousEntry.Value) != strings.TrimSpace(entry.Value):
				validation.DiffSummary = append(validation.DiffSummary, fmt.Sprintf("key %s value changed", key))
			}
			delete(previousByKey, key)
		}
		for key, entry := range previousByKey {
			if entry.Required {
				validation.MissingRequiredKeys = append(validation.MissingRequiredKeys, key)
			}
			validation.DiffSummary = append(validation.DiffSummary, fmt.Sprintf("removed key %s", key))
		}
	}

	sort.Strings(validation.MissingRequiredKeys)
	sort.Strings(validation.DeprecatedKeys)
	sort.Strings(validation.InvalidSecretRefs)
	sort.Strings(validation.SecretReferenceKeys)
	sort.Strings(validation.DiffSummary)

	switch {
	case len(validation.InvalidSecretRefs) > 0 || len(validation.MissingRequiredKeys) > 0:
		validation.Status = "blocked"
	case len(validation.DeprecatedKeys) > 0 || len(validation.Warnings) > 0:
		validation.Status = "warning"
	default:
		validation.Status = "valid"
	}
	return validation, nil
}

func (a *Application) buildReleaseAnalysis(ctx context.Context, release types.Release) (types.ReleaseAnalysis, error) {
	changes, configSets, err := a.loadReleaseScope(ctx, release.OrganizationID, release.ProjectID, release.EnvironmentID, release.ChangeSetIDs, release.ConfigSetIDs)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	environment, err := a.Store.GetEnvironment(ctx, release.EnvironmentID)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}

	servicesByID := make(map[string]types.Service, len(changes))
	assessments := make([]types.RiskAssessment, 0, len(changes))
	highRiskChanges := make([]string, 0, len(changes))
	uniqueServiceIDs := map[string]struct{}{}
	for _, change := range changes {
		service, ok := servicesByID[change.ServiceID]
		if !ok {
			service, err = a.Store.GetService(ctx, change.ServiceID)
			if err != nil {
				return types.ReleaseAnalysis{}, err
			}
			servicesByID[service.ID] = service
		}
		uniqueServiceIDs[service.ID] = struct{}{}
		changeAssessments, err := a.Store.ListRiskAssessments(ctx, storage.RiskAssessmentQuery{
			OrganizationID: release.OrganizationID,
			ProjectID:      release.ProjectID,
			ChangeSetID:    change.ID,
		})
		if err != nil {
			return types.ReleaseAnalysis{}, err
		}
		var assessment types.RiskAssessment
		if len(changeAssessments) > 0 {
			assessment = changeAssessments[len(changeAssessments)-1]
		} else {
			assessment = a.RiskEngine.Assess(change, service, environment)
			assessment.ID = ""
		}
		if assessment.Level == types.RiskLevelHigh || assessment.Level == types.RiskLevelCritical {
			highRiskChanges = append(highRiskChanges, change.Summary)
		}
		assessments = append(assessments, assessment)
	}

	linkedExecutions, err := a.Store.ListRolloutExecutions(ctx, storage.RolloutExecutionQuery{
		OrganizationID: release.OrganizationID,
		ProjectID:      release.ProjectID,
		EnvironmentID:  release.EnvironmentID,
	})
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	filteredExecutions := make([]types.RolloutExecution, 0, len(linkedExecutions))
	for _, execution := range linkedExecutions {
		if execution.ReleaseID == release.ID {
			filteredExecutions = append(filteredExecutions, execution)
		}
	}

	policyDecisions, err := a.collectReleasePolicyDecisions(ctx, release, changes, assessments, filteredExecutions)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}

	configValidation := make([]types.ConfigSetValidation, 0, len(configSets))
	for _, configSet := range configSets {
		validation, err := a.buildConfigSetValidation(ctx, configSet)
		if err != nil {
			return types.ReleaseAnalysis{}, err
		}
		configValidation = append(configValidation, validation)
	}

	combinedScore, combinedLevel := combineReleaseRisk(assessments, len(uniqueServiceIDs), len(configSets))
	blastRadius := buildCombinedBlastRadius(changes, servicesByID, environment)
	dependencyPlan, err := a.buildReleaseDependencyPlan(ctx, release, servicesByID)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	databaseSnapshot, err := a.buildDatabaseGovernanceSnapshot(ctx, release.OrganizationID, release.ProjectID, environment, changes)
	if err != nil {
		return types.ReleaseAnalysis{}, err
	}
	databaseFindings := buildDatabaseFindings(changes, databaseSnapshot, environment)
	windowFindings := buildWindowFindings(environment, filteredExecutions, assessments)
	policyHighlights, blockersFromPolicies, warningsFromPolicies := summarizePolicyDecisions(policyDecisions)
	policyHighlights = append(policyHighlights, databaseSnapshot.PolicyHighlights...)
	warnings := make([]string, 0, 8)
	blockers := make([]string, 0, 8)
	warnings = append(warnings, warningsFromPolicies...)
	warnings = append(warnings, databaseSnapshot.Warnings...)
	blockers = append(blockers, blockersFromPolicies...)
	blockers = append(blockers, databaseSnapshot.Blockers...)
	if len(uniqueServiceIDs) > 1 {
		warnings = append(warnings, fmt.Sprintf("bundle spans %d services; validate deployment order and shared dependencies", len(uniqueServiceIDs)))
	}
	if len(filteredExecutions) > 0 {
		warnings = append(warnings, fmt.Sprintf("%d rollout execution(s) are already linked to this bundle", len(filteredExecutions)))
	}
	for _, validation := range configValidation {
		if validation.Status == "blocked" {
			blockers = append(blockers, fmt.Sprintf("config set %s is blocked by validation findings", validation.ConfigSetID))
		}
		if validation.Status == "warning" {
			warnings = append(warnings, fmt.Sprintf("config set %s requires review before deployment", validation.ConfigSetID))
		}
	}
	if len(databaseFindings) > 0 && environment.Production {
		warnings = append(warnings, "bundle includes database-affecting changes; confirm backward compatibility and rollback posture")
	}

	readiness := buildReadinessReview(changes, assessments, configValidation, dependencyPlan, databaseSnapshot.Posture, databaseFindings, windowFindings, blockers, environment)
	rollbackGuidance := buildRollbackGuidance(release, changes, configValidation, dependencyPlan, databaseSnapshot.Posture, environment)
	teamMemory := buildTeamMemoryInsights(changes, servicesByID, filteredExecutions)
	releaseSummary := summarizeRelease(release, changes, configSets, combinedLevel, blastRadius)
	opsSummary := buildReleaseOpsSummary(highRiskChanges, blockers, warnings, rollbackGuidance)
	communications := buildCommunicationDrafts(release, releaseSummary, readiness, rollbackGuidance, blockers)

	return types.ReleaseAnalysis{
		Release:                 release,
		ChangeSets:              changes,
		Assessments:             assessments,
		ConfigSets:              configSets,
		DatabaseConnections:     databaseSnapshot.Connections,
		DatabaseConnectionTests: databaseSnapshot.ConnectionTests,
		DatabaseChanges:         databaseSnapshot.Changes,
		DatabaseChecks:          databaseSnapshot.Checks,
		DatabaseExecutions:      databaseSnapshot.Executions,
		LinkedRolloutExecutions: filteredExecutions,
		CombinedRiskScore:       combinedScore,
		CombinedRiskLevel:       combinedLevel,
		BlastRadius:             blastRadius,
		ReleaseSummary:          releaseSummary,
		DependencyPlan:          dependencyPlan,
		ConfigValidation:        configValidation,
		DatabasePosture:         databaseSnapshot.Posture,
		DatabaseFindings:        databaseFindings,
		WindowFindings:          windowFindings,
		PolicyHighlights:        dedupeStrings(policyHighlights),
		Warnings:                dedupeStrings(warnings),
		Blockers:                dedupeStrings(blockers),
		ReadinessReview:         readiness,
		RollbackGuidance:        rollbackGuidance,
		OpsAssistant:            opsSummary,
		TeamMemory:              teamMemory,
		Communications:          communications,
	}, nil
}

func (a *Application) loadReleaseScope(ctx context.Context, organizationID, projectID, environmentID string, changeSetIDs, configSetIDs []string) ([]types.ChangeSet, []types.ConfigSet, error) {
	changes := make([]types.ChangeSet, 0, len(changeSetIDs))
	for _, id := range changeSetIDs {
		change, err := a.Store.GetChangeSet(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if change.OrganizationID != organizationID || change.ProjectID != projectID {
			return nil, nil, fmt.Errorf("%w: change set %s scope mismatch", ErrValidation, id)
		}
		if environmentID != "" && change.EnvironmentID != environmentID {
			return nil, nil, fmt.Errorf("%w: change set %s belongs to environment %s, expected %s", ErrValidation, id, change.EnvironmentID, environmentID)
		}
		changes = append(changes, change)
	}
	configSets := make([]types.ConfigSet, 0, len(configSetIDs))
	for _, id := range configSetIDs {
		configSet, err := a.Store.GetConfigSet(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if configSet.OrganizationID != organizationID || configSet.ProjectID != projectID {
			return nil, nil, fmt.Errorf("%w: config set %s scope mismatch", ErrValidation, id)
		}
		if environmentID != "" && configSet.EnvironmentID != environmentID {
			return nil, nil, fmt.Errorf("%w: config set %s belongs to environment %s, expected %s", ErrValidation, id, configSet.EnvironmentID, environmentID)
		}
		configSets = append(configSets, configSet)
	}
	return changes, configSets, nil
}

func (a *Application) validateProjectEnvironment(ctx context.Context, organizationID, projectID, environmentID string) (types.Project, types.Environment, error) {
	if strings.TrimSpace(organizationID) == "" || strings.TrimSpace(projectID) == "" || strings.TrimSpace(environmentID) == "" {
		return types.Project{}, types.Environment{}, fmt.Errorf("%w: organization_id, project_id, and environment_id are required", ErrValidation)
	}
	project, err := a.Store.GetProject(ctx, projectID)
	if err != nil {
		return types.Project{}, types.Environment{}, fmt.Errorf("%w: project %s", storage.ErrNotFound, projectID)
	}
	if project.OrganizationID != organizationID {
		return types.Project{}, types.Environment{}, fmt.Errorf("%w: project %s does not belong to organization %s", ErrValidation, projectID, organizationID)
	}
	environment, err := a.Store.GetEnvironment(ctx, environmentID)
	if err != nil {
		return types.Project{}, types.Environment{}, fmt.Errorf("%w: environment %s", storage.ErrNotFound, environmentID)
	}
	if environment.OrganizationID != organizationID || environment.ProjectID != projectID {
		return types.Project{}, types.Environment{}, fmt.Errorf("%w: environment scope mismatch", ErrValidation)
	}
	return project, environment, nil
}

func (a *Application) validateConfigSetScope(ctx context.Context, organizationID, projectID, environmentID, serviceID string) (types.Project, types.Environment, *types.Service, *types.Team, error) {
	project, environment, err := a.validateProjectEnvironment(ctx, organizationID, projectID, environmentID)
	if err != nil {
		return types.Project{}, types.Environment{}, nil, nil, err
	}
	nameRequired := strings.TrimSpace(serviceID)
	if nameRequired == "" {
		return project, environment, nil, nil, nil
	}
	service, err := a.Store.GetService(ctx, serviceID)
	if err != nil {
		return types.Project{}, types.Environment{}, nil, nil, fmt.Errorf("%w: service %s", storage.ErrNotFound, serviceID)
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

func (a *Application) canManageConfigSet(identity auth.Identity, organizationID, projectID string, service *types.Service, team *types.Team) bool {
	if a.Authorizer.CanManageProject(identity, organizationID, projectID) {
		return true
	}
	if service != nil && team != nil {
		return a.Authorizer.CanManageService(identity, *service, *team)
	}
	return false
}

func (a *Application) collectReleasePolicyDecisions(ctx context.Context, release types.Release, changes []types.ChangeSet, assessments []types.RiskAssessment, executions []types.RolloutExecution) ([]types.PolicyDecision, error) {
	merged := make([]types.PolicyDecision, 0, 16)
	seen := map[string]struct{}{}
	appendDecisionSet := func(items []types.PolicyDecision) {
		for _, item := range items {
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			merged = append(merged, item)
		}
	}
	for _, change := range changes {
		items, err := a.Store.ListPolicyDecisions(ctx, storage.PolicyDecisionQuery{
			OrganizationID: release.OrganizationID,
			ProjectID:      release.ProjectID,
			ChangeSetID:    change.ID,
			Limit:          200,
		})
		if err != nil {
			return nil, err
		}
		appendDecisionSet(items)

		plans, err := a.Store.ListRolloutPlans(ctx, storage.RolloutPlanQuery{
			OrganizationID: release.OrganizationID,
			ProjectID:      release.ProjectID,
			ChangeSetID:    change.ID,
			Limit:          200,
		})
		if err != nil {
			return nil, err
		}
		for _, plan := range plans {
			planItems, err := a.Store.ListPolicyDecisions(ctx, storage.PolicyDecisionQuery{
				OrganizationID: release.OrganizationID,
				ProjectID:      release.ProjectID,
				RolloutPlanID:  plan.ID,
				Limit:          200,
			})
			if err != nil {
				return nil, err
			}
			appendDecisionSet(planItems)
		}
	}
	for _, assessment := range assessments {
		if assessment.ID == "" {
			continue
		}
		items, err := a.Store.ListPolicyDecisions(ctx, storage.PolicyDecisionQuery{
			OrganizationID:   release.OrganizationID,
			ProjectID:        release.ProjectID,
			RiskAssessmentID: assessment.ID,
			Limit:            200,
		})
		if err != nil {
			return nil, err
		}
		appendDecisionSet(items)
	}
	for _, execution := range executions {
		items, err := a.Store.ListPolicyDecisions(ctx, storage.PolicyDecisionQuery{
			OrganizationID:     release.OrganizationID,
			ProjectID:          release.ProjectID,
			RolloutExecutionID: execution.ID,
			Limit:              200,
		})
		if err != nil {
			return nil, err
		}
		appendDecisionSet(items)
	}
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].CreatedAt.Equal(merged[j].CreatedAt) {
			return merged[i].ID < merged[j].ID
		}
		return merged[i].CreatedAt.Before(merged[j].CreatedAt)
	})
	return merged, nil
}

func (a *Application) buildReleaseDependencyPlan(ctx context.Context, release types.Release, servicesByID map[string]types.Service) ([]types.ReleaseDependency, error) {
	relationships, err := a.Store.ListGraphRelationships(ctx, storage.GraphRelationshipQuery{
		OrganizationID:   release.OrganizationID,
		RelationshipType: "service_dependency",
		Limit:            1000,
	})
	if err != nil {
		return nil, err
	}
	plan := make([]types.ReleaseDependency, 0, len(relationships))
	for _, relationship := range relationships {
		if relationship.FromResourceType != "service" || relationship.ToResourceType != "service" {
			continue
		}
		service, serviceOK := servicesByID[relationship.FromResourceID]
		dependency, dependencyOK := servicesByID[relationship.ToResourceID]
		if !serviceOK || !dependencyOK {
			continue
		}
		plan = append(plan, types.ReleaseDependency{
			ServiceID:            service.ID,
			ServiceName:          service.Name,
			DependsOnServiceID:   dependency.ID,
			DependsOnServiceName: dependency.Name,
			Critical:             metadataBool(relationship.Metadata, "critical_dependency"),
			Summary:              fmt.Sprintf("%s should be sequenced after %s because a dependency relationship was discovered.", service.Name, dependency.Name),
		})
	}
	sort.Slice(plan, func(i, j int) bool {
		if plan[i].ServiceName == plan[j].ServiceName {
			return plan[i].DependsOnServiceName < plan[j].DependsOnServiceName
		}
		return plan[i].ServiceName < plan[j].ServiceName
	})
	return plan, nil
}

func summarizePolicyDecisions(decisions []types.PolicyDecision) ([]string, []string, []string) {
	highlights := make([]string, 0, len(decisions))
	blockers := make([]string, 0, 4)
	warnings := make([]string, 0, 4)
	for _, decision := range decisions {
		if decision.Summary != "" {
			highlights = append(highlights, fmt.Sprintf("%s: %s", decision.PolicyName, decision.Summary))
		}
		switch decision.Outcome {
		case policylib.ModeBlock:
			blockers = append(blockers, fmt.Sprintf("policy %s blocks this bundle", decision.PolicyName))
		case policylib.ModeRequireManualReview, "require_approval":
			warnings = append(warnings, fmt.Sprintf("policy %s requires manual review", decision.PolicyName))
		}
	}
	return dedupeStrings(highlights), dedupeStrings(blockers), dedupeStrings(warnings)
}

func combineReleaseRisk(assessments []types.RiskAssessment, uniqueServices, configSets int) (int, types.RiskLevel) {
	if len(assessments) == 0 {
		return 0, types.RiskLevelLow
	}
	maxScore := 0
	total := 0
	for _, assessment := range assessments {
		if assessment.Score > maxScore {
			maxScore = assessment.Score
		}
		total += assessment.Score
	}
	combined := maxScore
	if average := total / len(assessments); average > combined {
		combined = average
	}
	combined += bundleMinInt(24, bundleMaxInt(0, len(assessments)-1)*6)
	combined += bundleMinInt(12, bundleMaxInt(0, uniqueServices-1)*4)
	combined += bundleMinInt(8, configSets*2)
	if combined > 100 {
		combined = 100
	}
	switch {
	case combined >= 80:
		return combined, types.RiskLevelCritical
	case combined >= 55:
		return combined, types.RiskLevelHigh
	case combined >= 30:
		return combined, types.RiskLevelMedium
	default:
		return combined, types.RiskLevelLow
	}
}

func buildCombinedBlastRadius(changes []types.ChangeSet, servicesByID map[string]types.Service, environment types.Environment) types.BlastRadius {
	serviceCount := 0
	resourceCount := 0
	customerFacing := false
	regulated := false
	for _, service := range servicesByID {
		serviceCount++
		if service.CustomerFacing {
			customerFacing = true
		}
		if service.RegulatedZone {
			regulated = true
		}
	}
	for _, change := range changes {
		resourceCount += bundleMaxInt(1, change.ResourceCount)
	}
	scope := "contained"
	switch {
	case environment.Production && customerFacing && serviceCount > 1:
		scope = "broad"
	case environment.Production || serviceCount > 1:
		scope = "moderate"
	}
	journeys := make([]string, 0, 2)
	if customerFacing {
		journeys = append(journeys, "customer-facing-paths")
	}
	if environment.Production {
		journeys = append(journeys, "production-traffic")
	}
	return types.BlastRadius{
		Scope:                scope,
		ServicesImpacted:     bundleMaxInt(1, serviceCount),
		ResourcesImpacted:    bundleMaxInt(1, resourceCount),
		CustomerJourneys:     journeys,
		RegulatedSystems:     regulated || strings.TrimSpace(environment.ComplianceZone) != "",
		ProductionImpact:     environment.Production,
		CustomerFacingImpact: customerFacing,
		Summary:              fmt.Sprintf("%d service(s), %d resource touchpoint(s), environment %s", bundleMaxInt(1, serviceCount), bundleMaxInt(1, resourceCount), environment.Name),
	}
}

func buildDatabaseFindings(changes []types.ChangeSet, snapshot databaseGovernanceSnapshot, environment types.Environment) []string {
	findings := append([]string{}, snapshot.Findings...)
	if len(snapshot.Changes) == 0 {
		for _, change := range changes {
			if !change.TouchesSchema {
				continue
			}
			findings = append(findings, fmt.Sprintf("change %s affects schema compatibility and should be treated as rollback-sensitive", change.ID))
			if environment.Production {
				findings = append(findings, fmt.Sprintf("production schema change %s should run with pre-deploy compatibility checks and post-deploy validation checks", change.ID))
			}
		}
	}
	if snapshot.Posture.Status != "none" && snapshot.Posture.Summary != "" {
		findings = append(findings, snapshot.Posture.Summary)
	}
	return dedupeStrings(findings)
}

func buildWindowFindings(environment types.Environment, linkedExecutions []types.RolloutExecution, assessments []types.RiskAssessment) []string {
	findings := make([]string, 0, 4)
	if environment.Production {
		findings = append(findings, "production environment requires explicit change-window awareness")
	}
	for _, assessment := range assessments {
		if strings.TrimSpace(assessment.RecommendedDeploymentWindow) != "" {
			findings = append(findings, fmt.Sprintf("recommended window: %s", assessment.RecommendedDeploymentWindow))
		}
	}
	activeExecutions := 0
	for _, execution := range linkedExecutions {
		if !rolloutExecutionTerminal(execution.Status) {
			activeExecutions++
		}
	}
	if activeExecutions > 0 {
		findings = append(findings, fmt.Sprintf("%d linked rollout execution(s) are still active for this environment", activeExecutions))
	}
	return dedupeStrings(findings)
}

func buildReadinessReview(changes []types.ChangeSet, assessments []types.RiskAssessment, validations []types.ConfigSetValidation, dependencies []types.ReleaseDependency, databasePosture types.DatabasePosture, databaseFindings, windowFindings, blockers []string, environment types.Environment) []types.ReadinessReviewItem {
	items := make([]types.ReadinessReviewItem, 0, 8)
	if len(blockers) > 0 {
		items = append(items, types.ReadinessReviewItem{
			Severity:               "critical",
			Category:               "policy",
			Question:               "Have you resolved every blocking policy or governance finding for this bundle?",
			Reason:                 strings.Join(blockers, "; "),
			Evidence:               blockers,
			AcknowledgmentRequired: true,
		})
	}
	for _, validation := range validations {
		if validation.Status != "blocked" && validation.Status != "warning" {
			continue
		}
		items = append(items, types.ReadinessReviewItem{
			Severity:               ternary(validation.Status == "blocked", "high", "medium"),
			Category:               "configuration",
			Question:               "Did you review the config-set diff, missing keys, and secret reference posture for this release?",
			Reason:                 fmt.Sprintf("config set %s produced %s findings", validation.ConfigSetID, validation.Status),
			Evidence:               append(append([]string{}, validation.DiffSummary...), validation.InvalidSecretRefs...),
			AcknowledgmentRequired: validation.Status == "blocked" || environment.Production,
		})
	}
	if databasePosture.ChangeCount > 0 || len(databaseFindings) > 0 {
		question := "Have you confirmed backward-compatible migration sequencing and post-deploy validation for database-affecting changes?"
		reason := "database changes are rollback-sensitive"
		if databasePosture.RollbackSafety == "unsafe" {
			question = "Have you explicitly agreed on a fix-forward strategy for irreversible or rollback-unsafe database changes?"
			reason = "database rollback safety is currently marked unsafe"
		} else if databasePosture.PendingCheckCount > 0 {
			question = "Have all required database validation checks been completed or explicitly acknowledged before release?"
			reason = fmt.Sprintf("%d required database validation check(s) remain pending", databasePosture.PendingCheckCount)
		} else if databasePosture.ManualApprovalRequired {
			question = "Has the required DBA or operator approved the manual-review database work in this bundle?"
			reason = "database governance marked one or more changes for manual approval"
		}
		items = append(items, types.ReadinessReviewItem{
			Severity:               "high",
			Category:               "database",
			Question:               question,
			Reason:                 reason,
			Evidence:               databaseFindings,
			AcknowledgmentRequired: environment.Production || databasePosture.ManualApprovalRequired || databasePosture.RollbackSafety == "unsafe",
		})
	}
	if len(dependencies) > 0 {
		items = append(items, types.ReadinessReviewItem{
			Severity:               "medium",
			Category:               "dependencies",
			Question:               "Have you verified deployment ordering for interdependent services in this bundle?",
			Reason:                 "service dependency relationships were detected inside the selected bundle",
			Evidence:               dependencySummaries(dependencies),
			AcknowledgmentRequired: environment.Production,
		})
	}
	for _, assessment := range assessments {
		if assessment.Level != types.RiskLevelHigh && assessment.Level != types.RiskLevelCritical {
			continue
		}
		items = append(items, types.ReadinessReviewItem{
			Severity:               "high",
			Category:               "risk",
			Question:               "Have the required approvers reviewed the high-risk portions of this release bundle?",
			Reason:                 fmt.Sprintf("change risk reached %s with recommended approval level %s", assessment.Level, assessment.RecommendedApprovalLevel),
			Evidence:               append([]string{}, assessment.Explanation...),
			AcknowledgmentRequired: true,
		})
		break
	}
	if len(windowFindings) > 0 {
		items = append(items, types.ReadinessReviewItem{
			Severity:               ternary(environment.Production, "medium", "low"),
			Category:               "change_window",
			Question:               "Does this release fit the recommended deployment window and avoid other active change collisions?",
			Reason:                 "window and execution findings were detected for the target environment",
			Evidence:               windowFindings,
			AcknowledgmentRequired: environment.Production,
		})
	}
	return items
}

func buildRollbackGuidance(release types.Release, changes []types.ChangeSet, validations []types.ConfigSetValidation, dependencies []types.ReleaseDependency, databasePosture types.DatabasePosture, environment types.Environment) types.RollbackGuidance {
	blockers := make([]string, 0, 4)
	steps := []string{
		"revert the selected release bundle to the last known healthy release boundary",
		"reapply the prior config-set revision for the same environment before re-enabling traffic",
	}
	if databasePosture.ChangeCount > 0 {
		switch databasePosture.RollbackSafety {
		case "unsafe":
			blockers = append(blockers, "database rollback posture is unsafe and should prefer fix-forward recovery")
			steps = append(steps, "do not issue a blind rollback; confirm whether the database must remain forward-only while application code is stabilized")
		case "manual_review":
			blockers = append(blockers, "database rollback posture requires manual review before reverting runtime traffic")
			steps = append(steps, "coordinate application rollback order with database compatibility constraints and required manual checks")
		default:
			steps = append(steps, "confirm that database validation checks still pass after the previous bundle is restored")
		}
	} else {
		for _, change := range changes {
			if change.TouchesSchema {
				blockers = append(blockers, "schema changes may make rollback unsafe without explicit backward-compatibility guarantees")
				steps = append(steps, "confirm whether schema expansion can remain in place while code rolls back, or whether a manual data remediation step is required")
				break
			}
		}
	}
	for _, validation := range validations {
		if len(validation.SecretReferenceKeys) > 0 {
			steps = append(steps, fmt.Sprintf("restore the previous secret reference mapping for config set %s if runtime credentials changed", validation.ConfigSetID))
		}
	}
	if len(dependencies) > 0 {
		steps = append(steps, "roll back dependent services in reverse order of the detected dependency chain when shared contracts changed")
	}
	safe := len(blockers) == 0
	strategy := "rollback_previous_bundle"
	if !safe {
		strategy = ternary(databasePosture.RollbackSafety == "manual_review", "guarded_manual_rollback", "fix_forward_preferred")
	}
	if environment.Production && safe {
		steps = append(steps, "keep verification and signal gates active while the previous bundle is restored")
	}
	return types.RollbackGuidance{
		Safe:     safe,
		Strategy: strategy,
		Summary:  fmt.Sprintf("release %s rollback posture: %s", release.Name, ternary(safe, "safe with standard guardrails", "manual review required")),
		Steps:    dedupeStrings(steps),
		Blockers: dedupeStrings(blockers),
	}
}

func buildReleaseOpsSummary(suspiciousChanges, blockers, warnings []string, rollbackGuidance types.RollbackGuidance) types.OpsAssistantSummary {
	status := "ready"
	likelyCause := "bundle composition and change context look consistent"
	if len(blockers) > 0 {
		status = "blocked"
		likelyCause = "policy, configuration, or migration blockers are still unresolved"
	} else if len(warnings) > 0 {
		status = "warning"
		likelyCause = "bundle complexity or rollout timing needs operator review"
	}
	guidance := make([]string, 0, 4)
	if len(blockers) > 0 {
		guidance = append(guidance, blockers...)
	}
	if len(rollbackGuidance.Blockers) > 0 {
		guidance = append(guidance, rollbackGuidance.Blockers...)
	}
	if len(guidance) == 0 {
		guidance = append(guidance, rollbackGuidance.Summary)
	}
	return types.OpsAssistantSummary{
		Status:            status,
		LikelyCause:       likelyCause,
		SuspiciousChanges: dedupeStrings(suspiciousChanges),
		Guidance:          dedupeStrings(guidance),
	}
}

func buildTeamMemoryInsights(changes []types.ChangeSet, servicesByID map[string]types.Service, executions []types.RolloutExecution) []types.TeamMemoryInsight {
	insights := make([]types.TeamMemoryInsight, 0, 6)
	for _, change := range changes {
		if change.HistoricalIncidentCount > 0 {
			serviceName := change.ServiceID
			if service, ok := servicesByID[change.ServiceID]; ok && service.Name != "" {
				serviceName = service.Name
			}
			insights = append(insights, types.TeamMemoryInsight{
				Title:   fmt.Sprintf("%s has prior incident history", serviceName),
				Summary: fmt.Sprintf("similar changes have been associated with %d historical incident(s)", change.HistoricalIncidentCount),
				Evidence: []string{
					change.Summary,
				},
			})
		}
		if change.PoorRollbackHistory {
			insights = append(insights, types.TeamMemoryInsight{
				Title:   "Rollback history is weak",
				Summary: "previous changes on this surface were marked as difficult to roll back cleanly",
				Evidence: []string{
					change.ID,
				},
			})
		}
	}
	for _, execution := range executions {
		if execution.LastDecision == "rollback" || strings.Contains(strings.ToLower(execution.Status), "rollback") {
			insights = append(insights, types.TeamMemoryInsight{
				Title:   "This bundle already triggered rollback activity",
				Summary: "linked rollout executions show rollback-oriented operator or verification decisions",
				Evidence: []string{
					execution.ID,
				},
			})
		}
	}
	return dedupeInsights(insights)
}

func summarizeRelease(release types.Release, changes []types.ChangeSet, configSets []types.ConfigSet, level types.RiskLevel, blast types.BlastRadius) string {
	parts := []string{
		fmt.Sprintf("%s (%s)", valueOrDefault(release.Name, release.Version), valueOrDefault(release.Version, "unversioned")),
		fmt.Sprintf("%d change set(s)", len(changes)),
		fmt.Sprintf("risk %s", level),
		fmt.Sprintf("blast radius %s", blast.Scope),
	}
	if len(configSets) > 0 {
		parts = append(parts, fmt.Sprintf("%d config set(s)", len(configSets)))
	}
	if strings.TrimSpace(release.Summary) != "" {
		parts = append(parts, release.Summary)
	}
	return strings.Join(parts, " | ")
}

func buildCommunicationDrafts(release types.Release, releaseSummary string, readiness []types.ReadinessReviewItem, rollback types.RollbackGuidance, blockers []string) types.CommunicationDrafts {
	approverLines := []string{releaseSummary}
	for _, item := range readiness {
		if !item.AcknowledgmentRequired {
			continue
		}
		approverLines = append(approverLines, fmt.Sprintf("- %s: %s", item.Category, item.Question))
	}
	stakeholder := fmt.Sprintf("Release %s is prepared for %s. %s", valueOrDefault(release.Name, release.Version), valueOrDefault(release.Status, "draft"), releaseSummary)
	maintenance := fmt.Sprintf("Maintenance window notice for %s: rollout guidance is %s.", valueOrDefault(release.Name, release.Version), rollback.Strategy)
	if len(blockers) > 0 {
		maintenance += " Current blockers: " + strings.Join(blockers, "; ")
	}
	return types.CommunicationDrafts{
		ReleaseNotes:      "Release notes: " + releaseSummary,
		ApproverSummary:   strings.Join(approverLines, "\n"),
		StakeholderUpdate: stakeholder,
		MaintenanceNotice: maintenance,
		IncidentHandoff:   fmt.Sprintf("If %s degrades, start with %s.", valueOrDefault(release.Name, release.Version), rollback.Strategy),
		PostmortemStarter: fmt.Sprintf("Postmortem starter for %s:\n- expected change\n- observed impact\n- decision timeline\n- rollback posture", valueOrDefault(release.Name, release.Version)),
	}
}

func buildIncidentAssistantSummary(detail types.RolloutExecutionDetail, releaseAnalysis *types.ReleaseAnalysis) types.OpsAssistantSummary {
	likelyCause := "recent rollout execution is the closest correlated change"
	suspicious := []string{}
	guidance := []string{}
	if releaseAnalysis != nil {
		likelyCause = releaseAnalysis.OpsAssistant.LikelyCause
		suspicious = append(suspicious, releaseAnalysis.OpsAssistant.SuspiciousChanges...)
		guidance = append(guidance, releaseAnalysis.RollbackGuidance.Steps...)
	}
	if detail.Execution.LastDecision != "" {
		likelyCause = fmt.Sprintf("rollout decision %s and status %s correlate most strongly with the incident", detail.Execution.LastDecision, detail.Execution.Status)
	}
	if detail.Execution.LastDecisionReason != "" {
		guidance = append(guidance, detail.Execution.LastDecisionReason)
	}
	if len(suspicious) == 0 {
		suspicious = append(suspicious, detail.Execution.ChangeSetID)
	}
	if len(guidance) == 0 {
		guidance = append(guidance, "inspect verification results, signal snapshots, and rollback readiness for the linked rollout execution")
	}
	status := "investigate"
	if detail.Execution.LastDecision == "rollback" || strings.Contains(strings.ToLower(detail.Execution.Status), "rollback") {
		status = "rollback_recommended"
	}
	return types.OpsAssistantSummary{
		Status:            status,
		LikelyCause:       likelyCause,
		SuspiciousChanges: dedupeStrings(suspicious),
		Guidance:          dedupeStrings(guidance),
	}
}

func normalizeConfigEntries(entries []types.ConfigEntry) []types.ConfigEntry {
	normalized := make([]types.ConfigEntry, 0, len(entries))
	for _, entry := range entries {
		normalized = append(normalized, types.ConfigEntry{
			Key:         strings.TrimSpace(entry.Key),
			Value:       strings.TrimSpace(entry.Value),
			ValueType:   valueOrDefault(strings.TrimSpace(entry.ValueType), "literal"),
			Required:    entry.Required,
			Deprecated:  entry.Deprecated,
			Description: strings.TrimSpace(entry.Description),
			Source:      strings.TrimSpace(entry.Source),
		})
	}
	return normalized
}

func configSetValidationBlocked(validation types.ConfigSetValidation) bool {
	return validation.Status == "blocked"
}

func blockingValidationReasons(validation types.ConfigSetValidation) []string {
	reasons := make([]string, 0, 4)
	if len(validation.InvalidSecretRefs) > 0 {
		reasons = append(reasons, "invalid secret references")
	}
	if len(validation.MissingRequiredKeys) > 0 {
		reasons = append(reasons, "missing required keys")
	}
	return reasons
}

func entryLooksSensitive(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(key, "secret") || strings.Contains(key, "password") || strings.Contains(key, "token") || strings.Contains(key, "api_key") || strings.HasSuffix(key, "_key")
}

func looksLikeSecretReference(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	return strings.Count(value, "/") >= 1 || strings.Count(value, ":") >= 1
}

func trimDedupedIDs(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func containsID(values []string, expected string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == strings.TrimSpace(expected) {
			return true
		}
	}
	return false
}

func dedupeStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func dedupeInsights(values []types.TeamMemoryInsight) []types.TeamMemoryInsight {
	result := make([]types.TeamMemoryInsight, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		key := strings.TrimSpace(value.Title) + "|" + strings.TrimSpace(value.Summary)
		if key == "|" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value.Evidence = dedupeStrings(value.Evidence)
		result = append(result, value)
	}
	return result
}

func dependencySummaries(values []types.ReleaseDependency) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Summary)
	}
	return result
}

func rolloutExecutionTerminal(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "completed", "verified", "failed", "rolled_back", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func metadataBool(metadata types.Metadata, key string) bool {
	if metadata == nil {
		return false
	}
	value, ok := metadata[key]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func bundleMinInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func bundleMaxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func ternary[T any](condition bool, whenTrue, whenFalse T) T {
	if condition {
		return whenTrue
	}
	return whenFalse
}
