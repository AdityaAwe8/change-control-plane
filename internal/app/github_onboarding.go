package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

const gitHubOnboardingTTL = 15 * time.Minute

type gitHubOnboardingState struct {
	IntegrationID  string `json:"integration_id"`
	OrganizationID string `json:"organization_id"`
	ActorID        string `json:"actor_id"`
	ScopeName      string `json:"scope_name,omitempty"`
	Owner          string `json:"owner,omitempty"`
	ExpiresAt      string `json:"expires_at"`
}

func (a *Application) StartGitHubOnboarding(ctx context.Context, integrationID string) (types.GitHubOnboardingStartResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.GitHubOnboardingStartResult{}, err
	}
	integration, err := a.Store.GetIntegration(ctx, integrationID)
	if err != nil {
		return types.GitHubOnboardingStartResult{}, err
	}
	if !a.Authorizer.CanManageIntegrations(identity, integration.OrganizationID) {
		return types.GitHubOnboardingStartResult{}, a.forbidden(ctx, identity, "integration.github_onboarding.denied", "integration", integration.ID, integration.OrganizationID, "", []string{"actor lacks github onboarding permission"})
	}
	if !strings.EqualFold(integration.Kind, "github") {
		return types.GitHubOnboardingStartResult{}, fmt.Errorf("%w: integration %s is not github", ErrValidation, integration.ID)
	}
	integration.AuthStrategy = normalizeIntegrationAuthStrategy(integration.Kind, integration.AuthStrategy, integration.Metadata)
	if integration.AuthStrategy != "github_app" {
		return types.GitHubOnboardingStartResult{}, fmt.Errorf("%w: github onboarding start requires auth_strategy github_app", ErrValidation)
	}
	if stringMetadataValue(integration.Metadata, "app_slug") == "" {
		return types.GitHubOnboardingStartResult{}, fmt.Errorf("%w: github app onboarding requires app_slug", ErrValidation)
	}
	if stringMetadataValue(integration.Metadata, "app_id") == "" {
		return types.GitHubOnboardingStartResult{}, fmt.Errorf("%w: github app onboarding requires app_id", ErrValidation)
	}
	if stringMetadataValue(integration.Metadata, "private_key_env") == "" {
		return types.GitHubOnboardingStartResult{}, fmt.Errorf("%w: github app onboarding requires private_key_env", ErrValidation)
	}
	expiresAt := time.Now().UTC().Add(gitHubOnboardingTTL)
	state, err := a.signGitHubOnboardingState(gitHubOnboardingState{
		IntegrationID:  integration.ID,
		OrganizationID: integration.OrganizationID,
		ActorID:        identity.ActorID,
		ScopeName:      integration.ScopeName,
		Owner:          stringMetadataValue(integration.Metadata, "owner"),
		ExpiresAt:      expiresAt.Format(time.RFC3339),
	})
	if err != nil {
		return types.GitHubOnboardingStartResult{}, err
	}
	webBaseURL := strings.TrimRight(valueOrDefault(stringMetadataValue(integration.Metadata, "web_base_url"), "https://github.com"), "/")
	appSlug := strings.TrimSpace(stringMetadataValue(integration.Metadata, "app_slug"))
	authorizeURL := webBaseURL + "/apps/" + url.PathEscape(appSlug) + "/installations/new?state=" + url.QueryEscape(state)
	callbackURL := strings.TrimRight(a.Config.APIBaseURL, "/") + "/api/v1/integrations/github/callback"
	integration.OnboardingStatus = "awaiting_callback"
	integration.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateIntegration(ctx, integration); err != nil {
		return types.GitHubOnboardingStartResult{}, err
	}
	if err := a.record(ctx, identity, "integration.github_onboarding.started", "integration", integration.ID, integration.OrganizationID, "", []string{integration.InstanceKey, authorizeURL}); err != nil {
		return types.GitHubOnboardingStartResult{}, err
	}
	return types.GitHubOnboardingStartResult{
		Integration:  hydrateIntegrationRuntimeState(integration, time.Now().UTC()),
		AuthorizeURL: authorizeURL,
		CallbackURL:  callbackURL,
		ExpiresAt:    expiresAt.Format(time.RFC3339),
		Strategy:     "github_app",
		StatePreview: truncateString(state, 24),
	}, nil
}

func (a *Application) CompleteGitHubOnboarding(ctx context.Context, rawState string, values url.Values) (types.Integration, error) {
	state, err := a.verifyGitHubOnboardingState(rawState)
	if err != nil {
		return types.Integration{}, err
	}
	integration, err := a.Store.GetIntegration(ctx, state.IntegrationID)
	if err != nil {
		return types.Integration{}, err
	}
	if !strings.EqualFold(integration.Kind, "github") || integration.OrganizationID != state.OrganizationID {
		return types.Integration{}, ErrForbidden
	}
	if integration.Metadata == nil {
		integration.Metadata = types.Metadata{}
	}
	if callbackError := strings.TrimSpace(values.Get("error")); callbackError != "" {
		integration.OnboardingStatus = "error"
		integration.LastError = strings.TrimSpace(values.Get("error_description"))
		if integration.LastError == "" {
			integration.LastError = callbackError
		}
		integration.UpdatedAt = time.Now().UTC()
		if err := a.Store.UpdateIntegration(ctx, integration); err != nil {
			return types.Integration{}, err
		}
		return types.Integration{}, fmt.Errorf("%w: github onboarding callback returned %s", ErrValidation, callbackError)
	}
	installationID := strings.TrimSpace(values.Get("installation_id"))
	if installationID == "" {
		return types.Integration{}, fmt.Errorf("%w: github onboarding callback missing installation_id", ErrValidation)
	}
	integration.AuthStrategy = "github_app"
	integration.Metadata["installation_id"] = installationID
	if setupAction := strings.TrimSpace(values.Get("setup_action")); setupAction != "" {
		integration.Metadata["setup_action"] = setupAction
	}
	if state.Owner != "" && stringMetadataValue(integration.Metadata, "owner") == "" {
		integration.Metadata["owner"] = state.Owner
	}
	integration.OnboardingStatus = "installed"
	if integration.Status == "available" {
		integration.Status = "configured"
	}
	integration.UpdatedAt = time.Now().UTC()
	if err := a.Store.UpdateIntegration(ctx, integration); err != nil {
		return types.Integration{}, err
	}
	_ = a.record(ctx, systemIdentity(), "integration.github_onboarding.completed", "integration", integration.ID, integration.OrganizationID, "", []string{installationID, integration.InstanceKey})
	return hydrateIntegrationRuntimeState(integration, time.Now().UTC()), nil
}

func (a *Application) signGitHubOnboardingState(state gitHubOnboardingState) (string, error) {
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(a.Config.AuthTokenSecret))
	_, _ = mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encodedPayload + "." + signature, nil
}

func (a *Application) verifyGitHubOnboardingState(raw string) (gitHubOnboardingState, error) {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 {
		return gitHubOnboardingState{}, fmt.Errorf("%w: invalid github onboarding state", ErrValidation)
	}
	mac := hmac.New(sha256.New, []byte(a.Config.AuthTokenSecret))
	_, _ = mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return gitHubOnboardingState{}, ErrForbidden
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return gitHubOnboardingState{}, err
	}
	var state gitHubOnboardingState
	if err := json.Unmarshal(payload, &state); err != nil {
		return gitHubOnboardingState{}, err
	}
	expiresAt, err := time.Parse(time.RFC3339, state.ExpiresAt)
	if err != nil {
		return gitHubOnboardingState{}, err
	}
	if time.Now().UTC().After(expiresAt) {
		return gitHubOnboardingState{}, fmt.Errorf("%w: github onboarding state expired", ErrValidation)
	}
	return state, nil
}
