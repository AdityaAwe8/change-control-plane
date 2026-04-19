package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	liveintegrations "github.com/change-control-plane/change-control-plane/internal/integrations"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func isSCMIntegrationKind(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "github", "gitlab":
		return true
	default:
		return false
	}
}

func scmClientFromIntegration(ctx context.Context, integration types.Integration) (liveintegrations.SCMClient, string, error) {
	switch strings.ToLower(strings.TrimSpace(integration.Kind)) {
	case "github":
		client, scope, err := githubClientFromIntegration(ctx, integration)
		if err != nil {
			return nil, "", err
		}
		return client, scope, nil
	case "gitlab":
		tokenEnv := stringMetadataValue(integration.Metadata, "access_token_env")
		if tokenEnv == "" {
			return nil, "", fmt.Errorf("%w: gitlab access_token_env is required", ErrValidation)
		}
		token := strings.TrimSpace(os.Getenv(tokenEnv))
		if token == "" {
			return nil, "", fmt.Errorf("%w: gitlab token env %s is empty", ErrValidation, tokenEnv)
		}
		baseURL := stringMetadataValue(integration.Metadata, "api_base_url")
		scope := valueOrDefault(stringMetadataValue(integration.Metadata, "group"), stringMetadataValue(integration.Metadata, "namespace"))
		client := liveintegrations.NewGitLabClient(baseURL, token)
		return client, scope, nil
	default:
		return nil, "", fmt.Errorf("%w: unsupported scm integration kind %s", ErrValidation, integration.Kind)
	}
}

func gitlabWebhookClientFromIntegration(_ context.Context, integration types.Integration) (liveintegrations.GitLabClient, string, error) {
	tokenEnv := stringMetadataValue(integration.Metadata, "access_token_env")
	if tokenEnv == "" {
		return liveintegrations.GitLabClient{}, "", fmt.Errorf("%w: gitlab access_token_env is required", ErrValidation)
	}
	token := strings.TrimSpace(os.Getenv(tokenEnv))
	if token == "" {
		return liveintegrations.GitLabClient{}, "", fmt.Errorf("%w: gitlab token env %s is empty", ErrValidation, tokenEnv)
	}
	baseURL := stringMetadataValue(integration.Metadata, "api_base_url")
	scope := valueOrDefault(stringMetadataValue(integration.Metadata, "group"), stringMetadataValue(integration.Metadata, "namespace"))
	return liveintegrations.NewGitLabClient(baseURL, token), scope, nil
}

func (a *Application) syncSCMIntegration(ctx context.Context, integration types.Integration) ([]types.Repository, types.IntegrationSyncRun, error) {
	client, scope, err := scmClientFromIntegration(ctx, integration)
	if err != nil {
		return nil, a.integrationRunForError(integration, "sync", err), err
	}
	items, err := client.DiscoverRepositories(ctx, scope)
	if err != nil {
		return nil, a.integrationRunForError(integration, "sync", err), err
	}
	now := time.Now().UTC()
	repositories := make([]types.Repository, 0, len(items))
	importedOwnership := 0
	missingOwnership := 0
	unavailableOwnership := 0
	for _, discovered := range items {
		repository, err := a.upsertRepositoryFromSCM(ctx, integration, discovered, now)
		if err != nil {
			return nil, a.integrationRunForError(integration, "sync", err), err
		}
		repository, status, err := a.refreshRepositoryOwnershipFromSCM(ctx, integration, repository, discovered, now)
		if err != nil {
			return nil, a.integrationRunForError(integration, "sync", err), err
		}
		switch status {
		case "imported":
			importedOwnership++
		case "not_found":
			missingOwnership++
		default:
			unavailableOwnership++
		}
		repositories = append(repositories, repository)
	}
	run := types.IntegrationSyncRun{
		BaseRecord: types.BaseRecord{
			ID:        commonID("isr"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  types.Metadata{"scope": scope, "provider": integration.Kind},
		},
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		Operation:      "sync",
		Status:         "success",
		Summary:        fmt.Sprintf("discovered %d %s repositories", len(repositories), integration.Kind),
		Details: []string{
			fmt.Sprintf("scope=%s", valueOrDefault(scope, "authenticated-principal")),
			fmt.Sprintf("codeowners_imported=%d", importedOwnership),
			fmt.Sprintf("codeowners_not_found=%d", missingOwnership),
			fmt.Sprintf("codeowners_unavailable=%d", unavailableOwnership),
		},
		ResourceCount:  len(repositories),
		StartedAt:      now,
		CompletedAt:    &now,
	}
	return repositories, run, nil
}

func (a *Application) upsertRepositoryFromSCM(ctx context.Context, integration types.Integration, discovered liveintegrations.SCMRepository, now time.Time) (types.Repository, error) {
	repository, err := a.Store.GetRepositoryByURL(ctx, integration.OrganizationID, discovered.HTMLURL)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return types.Repository{}, err
	}
	metadata := discovered.Metadata
	if metadata == nil {
		metadata = types.Metadata{}
	}
	metadata["full_name"] = discovered.FullName
	metadata["owner"] = discovered.Owner
	metadata["namespace"] = discovered.Namespace
	metadata["private"] = discovered.Private
	metadata["archived"] = discovered.Archived
	metadata["source_integration_id"] = integration.ID
	metadata["source_provider_kind"] = integration.Kind
	metadata["scm_external_id"] = discovered.ExternalID
	metadata["integration_instances"] = appendUniqueMetadataStrings(metadata["integration_instances"], integration.ID)
	if err == nil {
		if repository.Metadata == nil {
			repository.Metadata = types.Metadata{}
		}
		repository.Name = valueOrDefault(discovered.Name, repository.Name)
		repository.Provider = integration.Kind
		repository.URL = discovered.HTMLURL
		repository.DefaultBranch = valueOrDefault(discovered.DefaultBranch, repository.DefaultBranch)
		if repository.SourceIntegrationID == "" {
			repository.SourceIntegrationID = integration.ID
		}
		repository.Status = valueOrDefault(repository.Status, "discovered")
		repository.LastSyncedAt = &now
		for key, value := range metadata {
			repository.Metadata[key] = value
		}
		setRepositoryDiscoveryProvenance(&repository, integration, discovered, now)
		repository.UpdatedAt = now
		if err := a.Store.UpdateRepository(ctx, repository); err != nil {
			return types.Repository{}, err
		}
		return repository, nil
	}
	repository = types.Repository{
		BaseRecord: types.BaseRecord{
			ID:        stableResourceID("repo", integration.OrganizationID, integration.Kind, discovered.HTMLURL),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  metadata,
		},
		OrganizationID:      integration.OrganizationID,
		SourceIntegrationID: integration.ID,
		Name:                discovered.Name,
		Provider:            integration.Kind,
		URL:                 discovered.HTMLURL,
		DefaultBranch:       valueOrDefault(discovered.DefaultBranch, "main"),
		Status:              "discovered",
		LastSyncedAt:        &now,
	}
	setRepositoryDiscoveryProvenance(&repository, integration, discovered, now)
	if err := a.Store.UpsertRepository(ctx, repository); err != nil {
		return types.Repository{}, err
	}
	return repository, nil
}

func (a *Application) refreshRepositoryOwnershipFromSCM(ctx context.Context, integration types.Integration, repository types.Repository, discovered liveintegrations.SCMRepository, now time.Time) (types.Repository, string, error) {
	ownership, err := a.loadRepositoryOwnershipFromSCM(ctx, integration, repository, discovered)
	if err != nil {
		return repository, "", err
	}
	repository = applyRepositoryOwnershipImport(repository, ownership, now)
	repository, err = a.applyRepositoryInferredOwnership(ctx, repository, now)
	if err != nil {
		return repository, "", err
	}
	if err := a.Store.UpdateRepository(ctx, repository); err != nil {
		return repository, "", err
	}
	return repository, ownership.Status, nil
}

func (a *Application) loadRepositoryOwnershipFromSCM(ctx context.Context, integration types.Integration, repository types.Repository, discovered liveintegrations.SCMRepository) (liveintegrations.RepositoryOwnershipImport, error) {
	switch strings.ToLower(strings.TrimSpace(integration.Kind)) {
	case "github":
		client, _, err := githubClientFromIntegration(ctx, integration)
		if err != nil {
			return liveintegrations.RepositoryOwnershipImport{}, err
		}
		return client.LoadCODEOWNERS(ctx, discovered)
	case "gitlab":
		client, _, err := gitlabWebhookClientFromIntegration(ctx, integration)
		if err != nil {
			return liveintegrations.RepositoryOwnershipImport{}, err
		}
		if strings.TrimSpace(discovered.ExternalID) == "" {
			discovered.ExternalID = stringMetadataValue(repository.Metadata, "scm_external_id")
		}
		return client.LoadCODEOWNERS(ctx, discovered)
	default:
		return liveintegrations.RepositoryOwnershipImport{
			Provider: integration.Kind,
			Source:   ownershipSourceCodeowners,
			Status:   "unavailable",
			Error:    "CODEOWNERS import is only supported for GitHub and GitLab integrations",
		}, nil
	}
}

func (a *Application) ingestSCMWebhookChange(ctx context.Context, integration types.Integration, change liveintegrations.SCMWebhookChange) ([]string, error) {
	repository, err := a.upsertRepositoryFromSCM(ctx, integration, change.Repository, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if err := a.ensureRepositoryGraphMappings(ctx, repository); err != nil {
		return nil, err
	}
	if repository.ServiceID == "" || repository.EnvironmentID == "" || repository.ProjectID == "" {
		return []string{"repository is discovered but not fully mapped to project, service, and environment; change metadata was retained without creating a change set"}, nil
	}
	now := time.Now().UTC()
	changeTypes, touchesInfrastructure, touchesIAM, touchesSecrets, touchesSchema, dependencyChanges := deriveSCMChangeAttributes(change.Files, change.ChangeType)
	metadata := types.Metadata{
		"source":               "scm_webhook",
		"source_event_source":  integration.Kind + "_webhook",
		"scm_provider":         integration.Kind,
		"repository_url":       repository.URL,
		"commit_sha":           change.CommitSHA,
		"branch":               change.Branch,
		"tag":                  change.Tag,
		"issue_keys":           change.IssueKeys,
		"approvers":            change.Approvers,
		"reviewers":            change.Reviewers,
		"labels":               change.Labels,
		"source_integration":   integration.ID,
		"source_provider_kind": integration.Kind,
		"file_names":           scmFileNames(change.Files),
		"provider_metadata":    change.Metadata,
	}
	changeSet := types.ChangeSet{
		BaseRecord: types.BaseRecord{
			ID:        commonID("chg"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  metadata,
		},
		OrganizationID:        integration.OrganizationID,
		ProjectID:             repository.ProjectID,
		ServiceID:             repository.ServiceID,
		EnvironmentID:         repository.EnvironmentID,
		Summary:               change.Summary,
		ChangeTypes:           changeTypes,
		FileCount:             change.FileCount,
		ResourceCount:         len(change.Files),
		TouchesInfrastructure: touchesInfrastructure,
		TouchesIAM:            touchesIAM,
		TouchesSecrets:        touchesSecrets,
		TouchesSchema:         touchesSchema,
		DependencyChanges:     dependencyChanges,
		Status:                "ingested",
	}
	if err := a.Store.CreateChangeSet(ctx, changeSet); err != nil {
		return nil, err
	}
	relationship := newGraphRelationship(now, integration.ID, integration.OrganizationID, changeSet.ProjectID, "change_repository", "change_set", changeSet.ID, "repository", repository.ID)
	if err := a.Store.UpsertGraphRelationship(ctx, relationship); err != nil {
		return nil, err
	}
	if err := a.record(ctx, systemIdentity(), "change.ingested", "change_set", changeSet.ID, changeSet.OrganizationID, changeSet.ProjectID, []string{changeSet.Summary, repository.URL, integration.Kind}); err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf("created change set %s from %s %s event", changeSet.ID, integration.Kind, change.ChangeType)}, nil
}

func deriveSCMChangeAttributes(files []liveintegrations.SCMChangedFile, changeType string) ([]string, bool, bool, bool, bool, bool) {
	typesSet := map[string]struct{}{
		"code": {},
	}
	var (
		touchesInfrastructure bool
		touchesIAM            bool
		touchesSecrets        bool
		touchesSchema         bool
		dependencyChanges     bool
	)
	switch strings.ToLower(strings.TrimSpace(changeType)) {
	case "release", "tag_push":
		typesSet["release"] = struct{}{}
	case "pull_request", "merge_request":
		typesSet["code_review"] = struct{}{}
	}
	for _, file := range files {
		name := strings.ToLower(strings.TrimSpace(file.Filename))
		switch {
		case strings.Contains(name, "terraform"), strings.Contains(name, "helm"), strings.Contains(name, "kustomize"), strings.Contains(name, "/deploy"), strings.Contains(name, "/infra"), strings.Contains(name, "dockerfile"), strings.HasSuffix(name, ".yaml"), strings.HasSuffix(name, ".yml"):
			touchesInfrastructure = true
			typesSet["infrastructure"] = struct{}{}
		}
		switch {
		case strings.Contains(name, "iam"), strings.Contains(name, "rbac"), strings.Contains(name, "policy"):
			touchesIAM = true
		}
		switch {
		case strings.Contains(name, "secret"), strings.Contains(name, "vault"), strings.Contains(name, "kms"), strings.Contains(name, ".env"):
			touchesSecrets = true
		}
		switch {
		case strings.Contains(name, "schema"), strings.Contains(name, "migration"), strings.Contains(name, "ddl"), strings.Contains(name, "/db/"):
			touchesSchema = true
		}
		switch {
		case strings.Contains(name, "package-lock"), strings.Contains(name, "pnpm-lock"), strings.Contains(name, "go.mod"), strings.Contains(name, "go.sum"), strings.Contains(name, "requirements"), strings.Contains(name, "poetry.lock"), strings.Contains(name, "cargo.lock"):
			dependencyChanges = true
		}
	}
	changeTypes := make([]string, 0, len(typesSet))
	for value := range typesSet {
		changeTypes = append(changeTypes, value)
	}
	return changeTypes, touchesInfrastructure, touchesIAM, touchesSecrets, touchesSchema, dependencyChanges
}

func scmFileNames(files []liveintegrations.SCMChangedFile) []string {
	names := make([]string, 0, len(files))
	for _, file := range files {
		name := strings.TrimSpace(file.Filename)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}
