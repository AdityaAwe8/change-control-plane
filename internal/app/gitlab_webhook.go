package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	liveintegrations "github.com/change-control-plane/change-control-plane/internal/integrations"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) HandleGitLabWebhook(ctx context.Context, integrationID string, headers http.Header, body []byte) (types.IntegrationSyncRun, error) {
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return types.IntegrationSyncRun{}, err
	}
	if !strings.EqualFold(integration.Kind, "gitlab") {
		return types.IntegrationSyncRun{}, fmt.Errorf("%w: integration %s is not gitlab", ErrValidation, integration.ID)
	}
	if !integration.Enabled {
		run := integrationSkippedRun(integration, "gitlab.webhook.ignored", "ignored gitlab webhook because integration is disabled")
		_, _ = a.persistIntegrationRun(ctx, integration, run)
		a.markWebhookDelivery(ctx, integration, nil)
		return run, nil
	}

	secretEnv := stringMetadataValue(integration.Metadata, "webhook_secret_env")
	secret := strings.TrimSpace(os.Getenv(secretEnv))
	if secretEnv == "" || secret == "" {
		run := a.integrationRunForError(integration, "gitlab.webhook.invalid", fmt.Errorf("%w: gitlab webhook secret env is not configured", ErrValidation))
		_, _ = a.persistIntegrationRun(ctx, integration, run)
		a.markWebhookDelivery(ctx, integration, fmt.Errorf("gitlab webhook secret env is not configured"))
		return types.IntegrationSyncRun{}, ErrForbidden
	}
	if !liveintegrations.ValidateGitLabWebhookToken(secret, headers.Get("X-Gitlab-Token")) {
		a.markWebhookDelivery(ctx, integration, fmt.Errorf("gitlab webhook token validation failed"))
		return types.IntegrationSyncRun{}, ErrForbidden
	}

	eventType := strings.TrimSpace(headers.Get("X-Gitlab-Event"))
	if eventType == "" {
		return types.IntegrationSyncRun{}, fmt.Errorf("%w: missing X-Gitlab-Event header", ErrValidation)
	}
	deliveryID := strings.TrimSpace(headers.Get("X-Gitlab-Event-UUID"))
	if deliveryID == "" {
		deliveryID = strings.TrimSpace(headers.Get("X-Request-Id"))
	}
	if deliveryID != "" {
		existing, err := a.Store.ListIntegrationSyncRuns(ctx, storage.IntegrationSyncRunQuery{
			OrganizationID:  integration.OrganizationID,
			IntegrationID:   integration.ID,
			Operation:       "gitlab.webhook." + strings.ToLower(strings.ReplaceAll(strings.TrimSpace(eventType), " ", "_")),
			ExternalEventID: deliveryID,
			Limit:           1,
		})
		if err == nil && len(existing) > 0 {
			a.markWebhookDelivery(ctx, integration, nil)
			return existing[0], nil
		}
	}

	fetchChanges := func(projectID string, iid int) ([]liveintegrations.SCMChangedFile, error) {
		client, _, err := scmClientFromIntegration(ctx, integration)
		if err != nil {
			return nil, err
		}
		gitlabClient, ok := client.(liveintegrations.GitLabClient)
		if !ok {
			return nil, fmt.Errorf("%w: gitlab integration did not resolve to a gitlab client", ErrValidation)
		}
		return gitlabClient.MergeRequestChanges(ctx, projectID, iid)
	}
	parsed, err := liveintegrations.ParseGitLabWebhook(eventType, deliveryID, body, fetchChanges)
	if err != nil {
		run := a.integrationRunForError(integration, "gitlab.webhook."+strings.ToLower(strings.ReplaceAll(strings.TrimSpace(eventType), " ", "_")), err)
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
	_ = a.record(ctx, systemIdentity(), "integration.gitlab.webhook.processed", "integration", integration.ID, integration.OrganizationID, "", compactDetailList(run.Details))
	return run, nil
}
