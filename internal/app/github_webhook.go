package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/auth"
	liveintegrations "github.com/change-control-plane/change-control-plane/internal/integrations"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) HandleGitHubWebhook(ctx context.Context, integrationID string, headers http.Header, body []byte) (types.IntegrationSyncRun, error) {
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return types.IntegrationSyncRun{}, err
	}
	if !strings.EqualFold(integration.Kind, "github") {
		return types.IntegrationSyncRun{}, fmt.Errorf("%w: integration %s is not github", ErrValidation, integration.ID)
	}
	if !integration.Enabled {
		run := integrationSkippedRun(integration, "github.webhook.ignored", "ignored github webhook because integration is disabled")
		_, _ = a.persistIntegrationRun(ctx, integration, run)
		a.markWebhookDelivery(ctx, integration, nil)
		return run, nil
	}

	secretEnv := stringMetadataValue(integration.Metadata, "webhook_secret_env")
	secret := strings.TrimSpace(os.Getenv(secretEnv))
	if secretEnv == "" || secret == "" {
		run := a.integrationRunForError(integration, "github.webhook.invalid", fmt.Errorf("%w: github webhook secret env is not configured", ErrValidation))
		_, _ = a.persistIntegrationRun(ctx, integration, run)
		a.markWebhookDelivery(ctx, integration, fmt.Errorf("github webhook secret env is not configured"))
		return types.IntegrationSyncRun{}, ErrForbidden
	}
	if !liveintegrations.ValidateGitHubWebhookSignature(secret, body, headers.Get("X-Hub-Signature-256")) {
		a.markWebhookDelivery(ctx, integration, fmt.Errorf("github webhook signature validation failed"))
		return types.IntegrationSyncRun{}, ErrForbidden
	}

	eventType := strings.TrimSpace(headers.Get("X-GitHub-Event"))
	if eventType == "" {
		return types.IntegrationSyncRun{}, fmt.Errorf("%w: missing X-GitHub-Event header", ErrValidation)
	}
	deliveryID := strings.TrimSpace(headers.Get("X-GitHub-Delivery"))
	if deliveryID != "" {
		existing, err := a.Store.ListIntegrationSyncRuns(ctx, storage.IntegrationSyncRunQuery{
			OrganizationID:  integration.OrganizationID,
			IntegrationID:   integration.ID,
			Operation:       "github.webhook." + eventType,
			ExternalEventID: deliveryID,
			Limit:           1,
		})
		if err == nil && len(existing) > 0 {
			a.markWebhookDelivery(ctx, integration, nil)
			return existing[0], nil
		}
	}

	fetchFiles := func(owner, repo string, number int) ([]liveintegrations.GitHubChangedFile, error) {
		client, _, err := githubClientFromIntegration(ctx, integration)
		if err != nil {
			return nil, err
		}
		return client.PullRequestFiles(ctx, owner, repo, number)
	}
	parsed, err := liveintegrations.ParseGitHubWebhook(eventType, deliveryID, body, fetchFiles)
	if err != nil {
		run := a.integrationRunForError(integration, "github.webhook."+eventType, err)
		run.ExternalEventID = deliveryID
		_, _ = a.persistIntegrationRun(ctx, integration, run)
		a.markWebhookDelivery(ctx, integration, err)
		return run, err
	}

	now := time.Now().UTC()
	run := types.IntegrationSyncRun{
		BaseRecord: types.BaseRecord{
			ID:        commonID("isr"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  types.Metadata{"event_type": eventType},
		},
		OrganizationID:  integration.OrganizationID,
		IntegrationID:   integration.ID,
		Operation:       parsed.Operation,
		Trigger:         "webhook",
		Status:          "success",
		Summary:         parsed.Summary,
		Details:         parsed.Details,
		ResourceCount:   parsed.ResourceCount,
		ExternalEventID: deliveryID,
		StartedAt:       now,
		CompletedAt:     &now,
	}
	if parsed.Change != nil {
		details, ingestErr := a.ingestSCMWebhookChange(ctx, integration, *parsed.Change)
		run.Details = append(run.Details, details...)
		if ingestErr != nil {
			run.Status = "error"
			run.Summary = ingestErr.Error()
			run.Details = append(run.Details, ingestErr.Error())
			integration, _ = a.persistIntegrationRun(ctx, integration, run)
			a.markWebhookDelivery(ctx, integration, ingestErr)
			return run, ingestErr
		}
	}

	integration, err = a.persistIntegrationRun(ctx, integration, run)
	if err != nil {
		return types.IntegrationSyncRun{}, err
	}
	a.markWebhookDelivery(ctx, integration, nil)
	_ = a.record(ctx, systemIdentity(), "integration.github.webhook.processed", "integration", integration.ID, integration.OrganizationID, "", compactDetailList(run.Details))
	return run, nil
}

func deriveGitHubChangeAttributes(files []liveintegrations.GitHubChangedFile, changeType string) ([]string, bool, bool, bool, bool, bool) {
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
	switch changeType {
	case "release":
		typesSet["release"] = struct{}{}
	case "pull_request":
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
			typesSet["iam"] = struct{}{}
		}
		switch {
		case strings.Contains(name, "secret"), strings.Contains(name, "vault"), strings.Contains(name, "credential"):
			touchesSecrets = true
			typesSet["secrets"] = struct{}{}
		}
		switch {
		case strings.Contains(name, "migration"), strings.Contains(name, "schema"), strings.HasSuffix(name, ".sql"):
			touchesSchema = true
			typesSet["schema"] = struct{}{}
		}
		switch {
		case strings.HasSuffix(name, "package-lock.json"), strings.HasSuffix(name, "package.json"), strings.HasSuffix(name, "go.mod"), strings.HasSuffix(name, "go.sum"), strings.HasSuffix(name, "requirements.txt"), strings.HasSuffix(name, "poetry.lock"), strings.HasSuffix(name, "cargo.lock"):
			dependencyChanges = true
			typesSet["dependencies"] = struct{}{}
		}
	}
	result := make([]string, 0, len(typesSet))
	for item := range typesSet {
		result = append(result, item)
	}
	return result, touchesInfrastructure, touchesIAM, touchesSecrets, touchesSchema, dependencyChanges
}

func githubFileNames(files []liveintegrations.GitHubChangedFile) []string {
	result := make([]string, 0, len(files))
	for _, file := range files {
		if strings.TrimSpace(file.Filename) != "" {
			result = append(result, file.Filename)
		}
	}
	return result
}

func integrationSkippedRun(integration types.Integration, operation, summary string) types.IntegrationSyncRun {
	now := time.Now().UTC()
	return types.IntegrationSyncRun{
		BaseRecord: types.BaseRecord{
			ID:        commonID("isr"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: integration.OrganizationID,
		IntegrationID:  integration.ID,
		Operation:      operation,
		Status:         "skipped",
		Summary:        summary,
		Details:        []string{summary},
		StartedAt:      now,
		CompletedAt:    &now,
	}
}

func systemIdentity() auth.Identity {
	return auth.Identity{Authenticated: true}
}
