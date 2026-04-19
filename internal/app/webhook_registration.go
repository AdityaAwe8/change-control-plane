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

func (a *Application) GetWebhookRegistration(ctx context.Context, integrationID string) (types.WebhookRegistrationResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.WebhookRegistrationResult{}, err
	}
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return types.WebhookRegistrationResult{}, err
	}
	if !a.Authorizer.CanManageIntegrations(identity, integration.OrganizationID) {
		return types.WebhookRegistrationResult{}, a.forbidden(ctx, identity, "webhook_registration.read.denied", "integration", integration.ID, integration.OrganizationID, "", []string{"actor lacks webhook registration visibility"})
	}
	registration, err := a.storeWebhookRegistrationForIntegration(ctx, integration)
	if err != nil {
		return types.WebhookRegistrationResult{}, err
	}
	registration = hydrateWebhookRegistration(integration, registration, time.Now().UTC())
	return types.WebhookRegistrationResult{
		Registration: registration,
		Details:      registrationDetails(registration),
	}, nil
}

func (a *Application) EnsureWebhookRegistration(ctx context.Context, integrationID string) (types.WebhookRegistrationResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.WebhookRegistrationResult{}, err
	}
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return types.WebhookRegistrationResult{}, err
	}
	if !a.Authorizer.CanManageIntegrations(identity, integration.OrganizationID) {
		return types.WebhookRegistrationResult{}, a.forbidden(ctx, identity, "webhook_registration.sync.denied", "integration", integration.ID, integration.OrganizationID, "", []string{"actor lacks webhook registration permission"})
	}
	result, err := a.ensureWebhookRegistration(ctx, integration, true)
	if err != nil {
		return result, err
	}
	if err := a.record(ctx, identity, "integration.webhook_registration.synced", "integration", integration.ID, integration.OrganizationID, "", result.Details); err != nil {
		return types.WebhookRegistrationResult{}, err
	}
	return result, nil
}

func (a *Application) ListOutboxEvents(ctx context.Context, query storage.OutboxEventQuery) ([]types.OutboxEvent, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanManageOrganization(identity, orgID) {
		return nil, a.forbidden(ctx, identity, "outbox_events.list.denied", "outbox_event", "", orgID, "", []string{"actor lacks runtime diagnostics permission"})
	}
	query.OrganizationID = orgID
	return a.Store.ListOutboxEvents(ctx, query)
}

func (a *Application) ensureWebhookRegistration(ctx context.Context, integration types.Integration, strict bool) (types.WebhookRegistrationResult, error) {
	registration, err := a.storeWebhookRegistrationForIntegration(ctx, integration)
	if err != nil {
		return types.WebhookRegistrationResult{}, err
	}
	registration = hydrateWebhookRegistration(integration, registration, time.Now().UTC())
	callbackURL := webhookCallbackURL(a.Config.APIBaseURL, integration)
	registration.CallbackURL = callbackURL
	registration.ProviderKind = integration.Kind
	registration.ScopeIdentifier = integrationWebhookScope(integration)
	registration.AutoManaged = true
	now := time.Now().UTC()

	webhookSecretEnv := stringMetadataValue(integration.Metadata, "webhook_secret_env")
	if webhookSecretEnv == "" {
		registration.Status = "manual_required"
		registration.LastError = "webhook_secret_env is not configured"
		registration.UpdatedAt = now
		_ = a.persistWebhookRegistration(ctx, registration)
		err := fmt.Errorf("%w: webhook_secret_env is required for automatic webhook registration", ErrValidation)
		if strict {
			return types.WebhookRegistrationResult{Registration: registration, Details: registrationDetails(registration)}, err
		}
		return types.WebhookRegistrationResult{Registration: registration, Details: registrationDetails(registration)}, nil
	}
	secret := strings.TrimSpace(os.Getenv(webhookSecretEnv))
	if secret == "" {
		registration.Status = "error"
		registration.LastError = fmt.Sprintf("webhook secret env %s is empty", webhookSecretEnv)
		registration.FailureCount++
		registration.UpdatedAt = now
		_ = a.persistWebhookRegistration(ctx, registration)
		err := fmt.Errorf("%w: webhook secret env %s is empty", ErrValidation, webhookSecretEnv)
		if strict {
			return types.WebhookRegistrationResult{Registration: registration, Details: registrationDetails(registration)}, err
		}
		return types.WebhookRegistrationResult{Registration: registration, Details: registrationDetails(registration)}, nil
	}
	result, err := a.registerWebhookWithProvider(ctx, integration, callbackURL, secret)
	if err != nil {
		registration.Status = "error"
		registration.LastError = err.Error()
		registration.FailureCount++
		registration.UpdatedAt = now
		_ = a.persistWebhookRegistration(ctx, registration)
		if strict {
			return types.WebhookRegistrationResult{Registration: registration, Details: registrationDetails(registration)}, err
		}
		return types.WebhookRegistrationResult{Registration: registration, Details: registrationDetails(registration)}, nil
	}
	registration.ExternalHookID = result.ExternalHookID
	registration.ScopeIdentifier = result.ScopeIdentifier
	registration.Status = valueOrDefault(strings.TrimSpace(result.Status), "registered")
	registration.DeliveryHealth = valueOrDefault(strings.TrimSpace(result.DeliveryHealth), registration.DeliveryHealth)
	registration.LastRegisteredAt = &now
	registration.LastValidatedAt = &now
	registration.LastError = ""
	registration.FailureCount = 0
	if result.Metadata != nil {
		registration.Metadata = result.Metadata
	}
	registration.UpdatedAt = now
	if err := a.persistWebhookRegistration(ctx, registration); err != nil {
		return types.WebhookRegistrationResult{}, err
	}
	return types.WebhookRegistrationResult{Registration: registration, Details: append(result.Details, registrationDetails(registration)...)}, nil
}

func (a *Application) persistWebhookRegistration(ctx context.Context, registration types.WebhookRegistration) error {
	if _, err := a.Store.GetWebhookRegistrationByIntegration(ctx, registration.IntegrationID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return a.Store.CreateWebhookRegistration(ctx, registration)
		}
		return err
	}
	return a.Store.UpdateWebhookRegistration(ctx, registration)
}

func (a *Application) storeWebhookRegistrationForIntegration(ctx context.Context, integration types.Integration) (types.WebhookRegistration, error) {
	registration, err := a.Store.GetWebhookRegistrationByIntegration(ctx, integration.ID)
	if err == nil {
		return registration, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return types.WebhookRegistration{}, err
	}
	now := time.Now().UTC()
	return types.WebhookRegistration{
		BaseRecord: types.BaseRecord{
			ID:        commonID("whr"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID:  integration.OrganizationID,
		IntegrationID:   integration.ID,
		ProviderKind:    integration.Kind,
		ScopeIdentifier: integrationWebhookScope(integration),
		CallbackURL:     webhookCallbackURL(a.Config.APIBaseURL, integration),
		Status:          "not_registered",
		DeliveryHealth:  "unknown",
		AutoManaged:     true,
	}, nil
}

func (a *Application) registerWebhookWithProvider(ctx context.Context, integration types.Integration, callbackURL, secret string) (liveintegrations.SCMWebhookRegistration, error) {
	switch strings.ToLower(strings.TrimSpace(integration.Kind)) {
	case "github":
		client, owner, err := githubClientFromIntegration(ctx, integration)
		if err != nil {
			return liveintegrations.SCMWebhookRegistration{}, err
		}
		registration, err := client.EnsureOrganizationWebhook(ctx, owner, callbackURL, secret)
		if err != nil {
			return liveintegrations.SCMWebhookRegistration{}, err
		}
		return registration, nil
	case "gitlab":
		gitlabClient, scope, err := gitlabWebhookClientFromIntegration(ctx, integration)
		if err != nil {
			return liveintegrations.SCMWebhookRegistration{}, err
		}
		registration, err := gitlabClient.EnsureGroupWebhook(ctx, scope, callbackURL, secret)
		if err != nil {
			return liveintegrations.SCMWebhookRegistration{}, err
		}
		return registration, nil
	default:
		return liveintegrations.SCMWebhookRegistration{}, fmt.Errorf("%w: automatic webhook registration is only supported for github and gitlab", ErrValidation)
	}
}

func webhookCallbackURL(apiBaseURL string, integration types.Integration) string {
	return strings.TrimRight(apiBaseURL, "/") + "/api/v1/integrations/" + integration.ID + "/webhooks/" + strings.ToLower(strings.TrimSpace(integration.Kind))
}

func integrationWebhookScope(integration types.Integration) string {
	switch strings.ToLower(strings.TrimSpace(integration.Kind)) {
	case "github":
		return stringMetadataValue(integration.Metadata, "owner")
	case "gitlab":
		return valueOrDefault(stringMetadataValue(integration.Metadata, "group"), stringMetadataValue(integration.Metadata, "namespace"))
	default:
		return integration.ScopeName
	}
}

func registrationDetails(registration types.WebhookRegistration) []string {
	details := []string{fmt.Sprintf("status=%s", registration.Status)}
	if registration.ScopeIdentifier != "" {
		details = append(details, "scope="+registration.ScopeIdentifier)
	}
	if registration.ExternalHookID != "" {
		details = append(details, "external_hook_id="+registration.ExternalHookID)
	}
	if registration.LastValidatedAt != nil {
		details = append(details, "last_validated_at="+registration.LastValidatedAt.Format(time.RFC3339))
	}
	if registration.LastError != "" {
		details = append(details, "last_error="+registration.LastError)
	}
	return details
}

func (a *Application) markWebhookDelivery(ctx context.Context, integration types.Integration, err error) {
	registration, lookupErr := a.storeWebhookRegistrationForIntegration(ctx, integration)
	if lookupErr != nil {
		return
	}
	now := time.Now().UTC()
	registration.LastDeliveryAt = &now
	registration.LastValidatedAt = &now
	registration.UpdatedAt = now
	registration.Metadata = ensureWebhookMetadata(registration.Metadata)
	if err != nil {
		registration.DeliveryHealth = "error"
		registration.LastError = err.Error()
		registration.FailureCount++
		registration.Status = "repair_recommended"
		registration.Metadata["last_delivery_status"] = "error"
	} else {
		registration.DeliveryHealth = "healthy"
		registration.LastError = ""
		registration.FailureCount = 0
		registration.Status = "registered"
		registration.Metadata["last_delivery_status"] = "healthy"
	}
	_ = a.persistWebhookRegistration(ctx, registration)
}

func hydrateWebhookRegistration(integration types.Integration, registration types.WebhookRegistration, now time.Time) types.WebhookRegistration {
	registration.ProviderKind = integration.Kind
	registration.ScopeIdentifier = integrationWebhookScope(integration)
	if !integration.Enabled {
		registration.Status = "disabled"
		registration.DeliveryHealth = "disabled"
		return registration
	}
	if stringMetadataValue(integration.Metadata, "webhook_secret_env") == "" {
		registration.Status = "manual_required"
		return registration
	}
	if registration.Status == "" {
		registration.Status = "not_registered"
	}
	if registration.Status == "registered" && (registration.DeliveryHealth == "error" || registration.FailureCount > 0) {
		registration.Status = "repair_recommended"
	}
	if registration.Status == "registered" && registration.LastValidatedAt != nil && registration.LastValidatedAt.Before(now.Add(-24*time.Hour)) {
		registration.Status = "repair_recommended"
	}
	return registration
}

func ensureWebhookMetadata(metadata types.Metadata) types.Metadata {
	if metadata == nil {
		return types.Metadata{}
	}
	return metadata
}
