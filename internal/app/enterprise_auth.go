package app

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type identityProviderStartState struct {
	ProviderID string `json:"provider_id"`
	ReturnTo   string `json:"return_to,omitempty"`
	ExpiresAt  int64  `json:"exp"`
}

func (a *Application) ListIdentityProviders(ctx context.Context) ([]types.IdentityProvider, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanManageOrganization(identity, orgID) {
		return nil, a.forbidden(ctx, identity, "identity_provider.list.denied", "identity_provider", "", orgID, "", []string{"actor lacks enterprise identity configuration permission"})
	}
	return a.Store.ListIdentityProviders(ctx, storage.IdentityProviderQuery{OrganizationID: orgID, Limit: 100})
}

func (a *Application) ListPublicIdentityProviders(ctx context.Context) ([]types.PublicIdentityProvider, error) {
	enabled := true
	providers, err := a.Store.ListIdentityProviders(ctx, storage.IdentityProviderQuery{
		Enabled: &enabled,
		Limit:   100,
	})
	if err != nil {
		return nil, err
	}
	items := make([]types.PublicIdentityProvider, 0, len(providers))
	for _, provider := range providers {
		items = append(items, types.PublicIdentityProvider{
			ID:             provider.ID,
			OrganizationID: provider.OrganizationID,
			Name:           provider.Name,
			Kind:           provider.Kind,
		})
	}
	return items, nil
}

func (a *Application) CreateIdentityProvider(ctx context.Context, req types.CreateIdentityProviderRequest) (types.IdentityProvider, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.IdentityProvider{}, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return types.IdentityProvider{}, err
	}
	if req.OrganizationID != "" && req.OrganizationID != orgID {
		return types.IdentityProvider{}, fmt.Errorf("%w: organization_id must match active organization", ErrValidation)
	}
	if !a.Authorizer.CanManageOrganization(identity, orgID) {
		return types.IdentityProvider{}, a.forbidden(ctx, identity, "identity_provider.create.denied", "identity_provider", "", orgID, "", []string{"actor lacks enterprise identity configuration permission"})
	}
	now := time.Now().UTC()
	provider := types.IdentityProvider{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("idp"),
			CreatedAt: now,
			UpdatedAt: now,
			Metadata:  req.Metadata,
		},
		OrganizationID:        orgID,
		Name:                  strings.TrimSpace(req.Name),
		Kind:                  normalizeIdentityProviderKind(req.Kind),
		IssuerURL:             strings.TrimSpace(req.IssuerURL),
		AuthorizationEndpoint: strings.TrimSpace(req.AuthorizationEndpoint),
		TokenEndpoint:         strings.TrimSpace(req.TokenEndpoint),
		UserInfoEndpoint:      strings.TrimSpace(req.UserInfoEndpoint),
		JWKSURI:               strings.TrimSpace(req.JWKSURI),
		ClientID:              strings.TrimSpace(req.ClientID),
		ClientSecretEnv:       strings.TrimSpace(req.ClientSecretEnv),
		Scopes:                normalizeIdentityProviderScopes(req.Scopes),
		ClaimMappings:         req.ClaimMappings,
		RoleMappings:          req.RoleMappings,
		AllowedDomains:        normalizeStringList(req.AllowedDomains),
		DefaultRole:           normalizeOrganizationRole(req.DefaultRole),
		Enabled:               req.Enabled,
		Status:                "configured",
		ConnectionHealth:      "unconfigured",
	}
	if err := validateIdentityProvider(provider); err != nil {
		return types.IdentityProvider{}, err
	}
	if err := a.Store.CreateIdentityProvider(ctx, provider); err != nil {
		return types.IdentityProvider{}, err
	}
	if err := a.record(ctx, identity, "identity_provider.created", "identity_provider", provider.ID, orgID, "", []string{provider.Name, provider.Kind}); err != nil {
		return types.IdentityProvider{}, err
	}
	return provider, nil
}

func (a *Application) UpdateIdentityProvider(ctx context.Context, id string, req types.UpdateIdentityProviderRequest) (types.IdentityProvider, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.IdentityProvider{}, err
	}
	provider, err := a.Store.GetIdentityProvider(ctx, id)
	if err != nil {
		return types.IdentityProvider{}, err
	}
	if !a.Authorizer.CanManageOrganization(identity, provider.OrganizationID) {
		return types.IdentityProvider{}, a.forbidden(ctx, identity, "identity_provider.update.denied", "identity_provider", provider.ID, provider.OrganizationID, "", []string{"actor lacks enterprise identity configuration permission"})
	}
	if req.Name != nil {
		provider.Name = strings.TrimSpace(*req.Name)
	}
	if req.IssuerURL != nil {
		provider.IssuerURL = strings.TrimSpace(*req.IssuerURL)
	}
	if req.AuthorizationEndpoint != nil {
		provider.AuthorizationEndpoint = strings.TrimSpace(*req.AuthorizationEndpoint)
	}
	if req.TokenEndpoint != nil {
		provider.TokenEndpoint = strings.TrimSpace(*req.TokenEndpoint)
	}
	if req.UserInfoEndpoint != nil {
		provider.UserInfoEndpoint = strings.TrimSpace(*req.UserInfoEndpoint)
	}
	if req.JWKSURI != nil {
		provider.JWKSURI = strings.TrimSpace(*req.JWKSURI)
	}
	if req.ClientID != nil {
		provider.ClientID = strings.TrimSpace(*req.ClientID)
	}
	if req.ClientSecretEnv != nil {
		provider.ClientSecretEnv = strings.TrimSpace(*req.ClientSecretEnv)
	}
	if req.Scopes != nil {
		provider.Scopes = normalizeIdentityProviderScopes(*req.Scopes)
	}
	if req.ClaimMappings != nil {
		provider.ClaimMappings = req.ClaimMappings
	}
	if req.RoleMappings != nil {
		provider.RoleMappings = req.RoleMappings
	}
	if req.AllowedDomains != nil {
		provider.AllowedDomains = normalizeStringList(*req.AllowedDomains)
	}
	if req.DefaultRole != nil {
		provider.DefaultRole = normalizeOrganizationRole(*req.DefaultRole)
	}
	if req.Enabled != nil {
		provider.Enabled = *req.Enabled
	}
	if req.Status != nil {
		provider.Status = strings.TrimSpace(*req.Status)
	}
	if req.Metadata != nil {
		provider.Metadata = req.Metadata
	}
	provider.UpdatedAt = time.Now().UTC()
	if err := validateIdentityProvider(provider); err != nil {
		return types.IdentityProvider{}, err
	}
	if err := a.Store.UpdateIdentityProvider(ctx, provider); err != nil {
		return types.IdentityProvider{}, err
	}
	if err := a.record(ctx, identity, "identity_provider.updated", "identity_provider", provider.ID, provider.OrganizationID, "", []string{provider.Name, provider.Kind}); err != nil {
		return types.IdentityProvider{}, err
	}
	return provider, nil
}

func (a *Application) TestIdentityProvider(ctx context.Context, id string) (types.IdentityProviderTestResult, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.IdentityProviderTestResult{}, err
	}
	provider, err := a.Store.GetIdentityProvider(ctx, id)
	if err != nil {
		return types.IdentityProviderTestResult{}, err
	}
	if !a.Authorizer.CanManageOrganization(identity, provider.OrganizationID) {
		return types.IdentityProviderTestResult{}, a.forbidden(ctx, identity, "identity_provider.test.denied", "identity_provider", provider.ID, provider.OrganizationID, "", []string{"actor lacks enterprise identity configuration permission"})
	}
	details, err := a.testIdentityProviderConnection(ctx, provider)
	now := time.Now().UTC()
	provider.LastTestedAt = &now
	provider.UpdatedAt = now
	if err != nil {
		provider.ConnectionHealth = "error"
		provider.LastError = err.Error()
		_ = a.Store.UpdateIdentityProvider(ctx, provider)
		return types.IdentityProviderTestResult{Provider: provider, Status: "error", Details: append(details, err.Error())}, err
	}
	provider.ConnectionHealth = "healthy"
	provider.LastError = ""
	if provider.Status == "" || provider.Status == "not_started" {
		provider.Status = "configured"
	}
	if err := a.Store.UpdateIdentityProvider(ctx, provider); err != nil {
		return types.IdentityProviderTestResult{}, err
	}
	if err := a.record(ctx, identity, "identity_provider.tested", "identity_provider", provider.ID, provider.OrganizationID, "", details); err != nil {
		return types.IdentityProviderTestResult{}, err
	}
	return types.IdentityProviderTestResult{Provider: provider, Status: "success", Details: details}, nil
}

func (a *Application) StartIdentityProviderSignIn(ctx context.Context, id string, req types.IdentityProviderStartRequest) (types.IdentityProviderStartResult, error) {
	provider, err := a.Store.GetIdentityProvider(ctx, id)
	if err != nil {
		return types.IdentityProviderStartResult{}, err
	}
	if !provider.Enabled {
		return types.IdentityProviderStartResult{}, ErrForbidden
	}
	resolved, err := a.resolveIdentityProviderEndpoints(ctx, provider)
	if err != nil {
		return types.IdentityProviderStartResult{}, err
	}
	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	stateValue, err := signIdentityProviderState(a.Config.AuthTokenSecret, identityProviderStartState{
		ProviderID: provider.ID,
		ReturnTo:   normalizeReturnToForConfig(a.Config, req.ReturnTo),
		ExpiresAt:  expiresAt.Unix(),
	})
	if err != nil {
		return types.IdentityProviderStartResult{}, err
	}
	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", resolved.ClientID)
	query.Set("scope", strings.Join(normalizeIdentityProviderScopes(resolved.Scopes), " "))
	query.Set("redirect_uri", a.identityProviderCallbackURL())
	query.Set("state", stateValue)
	authorizeURL := strings.TrimRight(resolved.AuthorizationEndpoint, "?") + "?" + query.Encode()
	return types.IdentityProviderStartResult{
		Provider:     resolved,
		AuthorizeURL: authorizeURL,
		CallbackURL:  a.identityProviderCallbackURL(),
		ExpiresAt:    expiresAt.Format(time.RFC3339),
		Strategy:     "oidc_authorization_code",
		StatePreview: previewToken(stateValue),
	}, nil
}

func (a *Application) CompleteIdentityProviderSignIn(ctx context.Context, rawState string, values url.Values) (types.AuthResponse, string, error) {
	state, err := verifyIdentityProviderState(a.Config.AuthTokenSecret, rawState)
	if err != nil {
		return types.AuthResponse{}, "", ErrUnauthorized
	}
	provider, err := a.Store.GetIdentityProvider(ctx, state.ProviderID)
	if err != nil {
		return types.AuthResponse{}, "", err
	}
	resolved, err := a.resolveIdentityProviderEndpoints(ctx, provider)
	if err != nil {
		return types.AuthResponse{}, state.ReturnTo, err
	}
	code := strings.TrimSpace(values.Get("code"))
	if code == "" {
		return types.AuthResponse{}, state.ReturnTo, fmt.Errorf("%w: missing authorization code", ErrValidation)
	}
	if callbackErr := strings.TrimSpace(values.Get("error")); callbackErr != "" {
		return types.AuthResponse{}, state.ReturnTo, fmt.Errorf("%w: %s", ErrUnauthorized, callbackErr)
	}
	claims, details, err := a.exchangeIdentityProviderCode(ctx, resolved, code)
	if err != nil {
		now := time.Now().UTC()
		provider.ConnectionHealth = "error"
		provider.LastError = err.Error()
		provider.UpdatedAt = now
		_ = a.Store.UpdateIdentityProvider(ctx, provider)
		return types.AuthResponse{}, state.ReturnTo, err
	}
	user, identityDetails, err := a.reconcileIdentityProviderUser(ctx, resolved, claims)
	if err != nil {
		return types.AuthResponse{}, state.ReturnTo, err
	}
	now := time.Now().UTC()
	provider.ConnectionHealth = "healthy"
	provider.Status = "active"
	provider.LastError = ""
	provider.LastAuthenticatedAt = &now
	provider.UpdatedAt = now
	if err := a.Store.UpdateIdentityProvider(ctx, provider); err != nil {
		return types.AuthResponse{}, state.ReturnTo, err
	}
	authResponse, err := a.issueAuthResponse(ctx, user, "auth.oidc_sign_in", append(details, identityDetails...), "oidc", provider.ID, provider.Name)
	if err != nil {
		return types.AuthResponse{}, state.ReturnTo, err
	}
	return authResponse, state.ReturnTo, nil
}

func (a *Application) testIdentityProviderConnection(ctx context.Context, provider types.IdentityProvider) ([]string, error) {
	resolved, err := a.resolveIdentityProviderEndpoints(ctx, provider)
	if err != nil {
		return nil, err
	}
	details := []string{fmt.Sprintf("resolved provider %s", resolved.Name)}
	if resolved.IssuerURL != "" {
		details = append(details, "issuer="+resolved.IssuerURL)
	}
	details = append(details, "authorization_endpoint="+resolved.AuthorizationEndpoint)
	details = append(details, "token_endpoint="+resolved.TokenEndpoint)
	details = append(details, "userinfo_endpoint="+resolved.UserInfoEndpoint)
	if strings.TrimSpace(os.Getenv(resolved.ClientSecretEnv)) == "" {
		return details, fmt.Errorf("%w: client secret env %s is empty", ErrValidation, resolved.ClientSecretEnv)
	}
	return details, nil
}

func (a *Application) resolveIdentityProviderEndpoints(ctx context.Context, provider types.IdentityProvider) (types.IdentityProvider, error) {
	resolved := provider
	if err := validateIdentityProvider(resolved); err != nil {
		return types.IdentityProvider{}, err
	}
	if resolved.IssuerURL == "" || (resolved.AuthorizationEndpoint != "" && resolved.TokenEndpoint != "" && resolved.UserInfoEndpoint != "") {
		return resolved, nil
	}
	discoveryURL := strings.TrimRight(resolved.IssuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return types.IdentityProvider{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return types.IdentityProvider{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.IdentityProvider{}, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return types.IdentityProvider{}, fmt.Errorf("oidc discovery failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		AuthorizationEndpoint string `json:"authorization_endpoint"`
		TokenEndpoint         string `json:"token_endpoint"`
		UserInfoEndpoint      string `json:"userinfo_endpoint"`
		JWKSURI               string `json:"jwks_uri"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return types.IdentityProvider{}, err
	}
	if resolved.AuthorizationEndpoint == "" {
		resolved.AuthorizationEndpoint = strings.TrimSpace(payload.AuthorizationEndpoint)
	}
	if resolved.TokenEndpoint == "" {
		resolved.TokenEndpoint = strings.TrimSpace(payload.TokenEndpoint)
	}
	if resolved.UserInfoEndpoint == "" {
		resolved.UserInfoEndpoint = strings.TrimSpace(payload.UserInfoEndpoint)
	}
	if resolved.JWKSURI == "" {
		resolved.JWKSURI = strings.TrimSpace(payload.JWKSURI)
	}
	if err := validateIdentityProvider(resolved); err != nil {
		return types.IdentityProvider{}, err
	}
	return resolved, nil
}

func (a *Application) exchangeIdentityProviderCode(ctx context.Context, provider types.IdentityProvider, code string) (map[string]any, []string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", a.identityProviderCallbackURL())
	form.Set("client_id", provider.ClientID)
	form.Set("client_secret", strings.TrimSpace(os.Getenv(provider.ClientSecretEnv)))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, nil, fmt.Errorf("oidc token exchange failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(tokenResponse.AccessToken) == "" {
		return nil, nil, fmt.Errorf("%w: oidc provider did not return access_token", ErrUnauthorized)
	}
	userInfoRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.UserInfoEndpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	userInfoRequest.Header.Set("Authorization", "Bearer "+tokenResponse.AccessToken)
	userInfoResponse, err := http.DefaultClient.Do(userInfoRequest)
	if err != nil {
		return nil, nil, err
	}
	defer userInfoResponse.Body.Close()
	userInfoBody, err := io.ReadAll(userInfoResponse.Body)
	if err != nil {
		return nil, nil, err
	}
	if userInfoResponse.StatusCode >= http.StatusBadRequest {
		return nil, nil, fmt.Errorf("oidc userinfo failed with status %d: %s", userInfoResponse.StatusCode, strings.TrimSpace(string(userInfoBody)))
	}
	claims := map[string]any{}
	if err := json.Unmarshal(userInfoBody, &claims); err != nil {
		return nil, nil, err
	}
	return claims, []string{"authorization code exchanged", "userinfo claims resolved"}, nil
}

func (a *Application) reconcileIdentityProviderUser(ctx context.Context, provider types.IdentityProvider, claims map[string]any) (types.User, []string, error) {
	subject := claimStringValue(claims, claimMappingValue(provider.ClaimMappings, "subject", "sub"))
	email := strings.ToLower(claimStringValue(claims, claimMappingValue(provider.ClaimMappings, "email", "email")))
	displayName := claimStringValue(claims, claimMappingValue(provider.ClaimMappings, "name", "name"))
	if subject == "" {
		return types.User{}, nil, fmt.Errorf("%w: oidc userinfo did not include subject", ErrUnauthorized)
	}
	if email == "" {
		return types.User{}, nil, fmt.Errorf("%w: oidc userinfo did not include email", ErrUnauthorized)
	}
	if err := validateAllowedDomain(provider.AllowedDomains, email); err != nil {
		return types.User{}, nil, err
	}
	if displayName == "" {
		displayName = email
	}
	now := time.Now().UTC()
	var user types.User
	err := a.Store.WithinTransaction(ctx, func(txCtx context.Context) error {
		var lookupErr error
		existingLink, linkErr := a.Store.GetIdentityLinkBySubject(txCtx, provider.ID, subject)
		switch {
		case linkErr == nil:
			user, lookupErr = a.Store.GetUser(txCtx, existingLink.UserID)
			if lookupErr != nil {
				return lookupErr
			}
			existingLink.Email = email
			existingLink.Status = "active"
			existingLink.LastLoginAt = &now
			existingLink.UpdatedAt = now
			if err := a.Store.UpdateIdentityLink(txCtx, existingLink); err != nil {
				return err
			}
		case linkErr != nil && !errors.Is(linkErr, storage.ErrNotFound):
			return linkErr
		default:
			user, lookupErr = a.Store.GetUserByEmail(txCtx, email)
			switch {
			case lookupErr == nil:
				if strings.TrimSpace(user.DisplayName) == "" {
					user.DisplayName = displayName
				}
				if user.OrganizationID == "" {
					user.OrganizationID = provider.OrganizationID
				}
				user.Status = "active"
				user.UpdatedAt = now
				if err := a.Store.UpdateUser(txCtx, user); err != nil {
					return err
				}
			case lookupErr != nil && errors.Is(lookupErr, storage.ErrNotFound):
				user = types.User{
					BaseRecord: types.BaseRecord{
						ID:        common.NewID("usr"),
						CreatedAt: now,
						UpdatedAt: now,
					},
					OrganizationID: provider.OrganizationID,
					Email:          email,
					DisplayName:    displayName,
					Status:         "active",
				}
				if err := a.Store.CreateUser(txCtx, user); err != nil {
					return err
				}
			default:
				return lookupErr
			}
			if _, err := a.Store.GetOrganizationMembership(txCtx, user.ID, provider.OrganizationID); err != nil {
				if !errors.Is(err, storage.ErrNotFound) {
					return err
				}
				membership := types.OrganizationMembership{
					BaseRecord: types.BaseRecord{
						ID:        common.NewID("orgm"),
						CreatedAt: now,
						UpdatedAt: now,
					},
					UserID:         user.ID,
					OrganizationID: provider.OrganizationID,
					Role:           mapOrganizationRole(provider, claims),
					Status:         "active",
				}
				if createErr := a.Store.CreateOrganizationMembership(txCtx, membership); createErr != nil {
					return createErr
				}
			}
			link := types.IdentityLink{
				BaseRecord: types.BaseRecord{
					ID:        common.NewID("idl"),
					CreatedAt: now,
					UpdatedAt: now,
				},
				OrganizationID: provider.OrganizationID,
				ProviderID:     provider.ID,
				UserID:         user.ID,
				ExternalSubject: subject,
				Email:          email,
				Status:         "active",
				LastLoginAt:    &now,
			}
			if err := a.Store.CreateIdentityLink(txCtx, link); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return types.User{}, nil, err
	}
	_ = a.record(ctx, systemIdentity(), "identity_provider.authentication.succeeded", "identity_provider", provider.ID, provider.OrganizationID, "", []string{provider.Name, email})
	return user, []string{"enterprise sign-in completed", "organization membership reconciled"}, nil
}

func validateIdentityProvider(provider types.IdentityProvider) error {
	if strings.TrimSpace(provider.OrganizationID) == "" {
		return fmt.Errorf("%w: organization_id is required", ErrValidation)
	}
	if strings.TrimSpace(provider.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	if normalizeIdentityProviderKind(provider.Kind) != "oidc" {
		return fmt.Errorf("%w: only oidc identity providers are supported in this milestone", ErrValidation)
	}
	if strings.TrimSpace(provider.ClientID) == "" {
		return fmt.Errorf("%w: client_id is required", ErrValidation)
	}
	if strings.TrimSpace(provider.ClientSecretEnv) == "" {
		return fmt.Errorf("%w: client_secret_env is required", ErrValidation)
	}
	if strings.TrimSpace(provider.IssuerURL) == "" && (strings.TrimSpace(provider.AuthorizationEndpoint) == "" || strings.TrimSpace(provider.TokenEndpoint) == "" || strings.TrimSpace(provider.UserInfoEndpoint) == "") {
		return fmt.Errorf("%w: issuer_url or explicit authorization/token/userinfo endpoints are required", ErrValidation)
	}
	return nil
}

func normalizeIdentityProviderKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "oidc", "oauth2", "google_workspace", "okta", "entra_id", "azure_ad":
		return "oidc"
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
}

func normalizeIdentityProviderScopes(scopes []string) []string {
	items := normalizeStringList(scopes)
	if len(items) == 0 {
		return []string{"openid", "profile", "email"}
	}
	return items
}

func normalizeStringList(values []string) []string {
	items := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	return items
}

func normalizeOrganizationRole(role string) string {
	switch strings.TrimSpace(role) {
	case "", "member", "org_member":
		return "org_member"
	case "admin", "org_admin":
		return "org_admin"
	case "viewer":
		return "viewer"
	default:
		return strings.TrimSpace(role)
	}
}

func claimMappingValue(mappings types.Metadata, key, fallback string) string {
	if mappings == nil {
		return fallback
	}
	if value, ok := mappings[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func claimStringValue(claims map[string]any, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		for _, item := range typed {
			if candidate, ok := item.(string); ok && strings.TrimSpace(candidate) != "" {
				return strings.TrimSpace(candidate)
			}
		}
	}
	return ""
}

func claimStringSlice(claims map[string]any, key string) []string {
	value, ok := claims[key]
	if !ok {
		return nil
	}
	result := []string{}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				result = append(result, strings.TrimSpace(text))
			}
		}
	case []string:
		return normalizeStringList(typed)
	case string:
		for _, part := range strings.Split(typed, ",") {
			if strings.TrimSpace(part) != "" {
				result = append(result, strings.TrimSpace(part))
			}
		}
	}
	return normalizeStringList(result)
}

func mapOrganizationRole(provider types.IdentityProvider, claims map[string]any) string {
	mappedRole := ""
	rolesClaim := claimMappingValue(provider.ClaimMappings, "roles", "groups")
	values := claimStringSlice(claims, rolesClaim)
	for _, value := range values {
		if provider.RoleMappings == nil {
			continue
		}
		if mapped, ok := provider.RoleMappings[value].(string); ok && strings.TrimSpace(mapped) != "" {
			mappedRole = normalizeOrganizationRole(mapped)
			break
		}
	}
	if mappedRole != "" {
		return mappedRole
	}
	return normalizeOrganizationRole(provider.DefaultRole)
}

func validateAllowedDomain(allowedDomains []string, email string) error {
	if len(allowedDomains) == 0 {
		return nil
	}
	parts := strings.Split(strings.ToLower(strings.TrimSpace(email)), "@")
	if len(parts) != 2 {
		return fmt.Errorf("%w: enterprise identity email is invalid", ErrUnauthorized)
	}
	domain := parts[1]
	for _, allowed := range allowedDomains {
		if strings.EqualFold(strings.TrimSpace(allowed), domain) {
			return nil
		}
	}
	return fmt.Errorf("%w: email domain %s is not allowed for this identity provider", ErrForbidden, domain)
}

func signIdentityProviderState(secret string, payload identityProviderStartState) (string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(body)
	signature := signIdentityProviderStatePayload(secret, encoded)
	return encoded + "." + signature, nil
}

func verifyIdentityProviderState(secret, raw string) (identityProviderStartState, error) {
	var payload identityProviderStartState
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 2 {
		return payload, ErrUnauthorized
	}
	if !hmac.Equal([]byte(parts[1]), []byte(signIdentityProviderStatePayload(secret, parts[0]))) {
		return payload, ErrUnauthorized
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return payload, ErrUnauthorized
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return payload, ErrUnauthorized
	}
	if time.Now().UTC().Unix() > payload.ExpiresAt {
		return payload, ErrUnauthorized
	}
	return payload, nil
}

func signIdentityProviderStatePayload(secret, encoded string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(encoded))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *Application) identityProviderCallbackURL() string {
	return strings.TrimRight(a.Config.APIBaseURL, "/") + "/api/v1/auth/providers/callback"
}

func normalizeReturnToForConfig(config common.Config, returnTo string) string {
	trimmed := strings.TrimSpace(returnTo)
	if trimmed == "" {
		return "/"
	}
	if strings.HasPrefix(trimmed, "/") && !strings.HasPrefix(trimmed, "//") {
		return trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "/"
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "/"
	}
	if !isAllowedReturnOrigin(config, parsed) {
		return "/"
	}
	return parsed.String()
}

func isAllowedReturnOrigin(config common.Config, parsed *url.URL) bool {
	if parsed == nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "localhost" || host == "127.0.0.1" || strings.HasSuffix(host, ".local") {
		return true
	}
	allowedOrigins := normalizeStringList(strings.Split(config.AllowedOrigins, ","))
	if baseURL, err := url.Parse(strings.TrimSpace(config.APIBaseURL)); err == nil && baseURL.Scheme != "" && baseURL.Host != "" {
		allowedOrigins = append(allowedOrigins, baseURL.Scheme+"://"+baseURL.Host)
	}
	targetOrigin := parsed.Scheme + "://" + parsed.Host
	for _, allowed := range allowedOrigins {
		if strings.EqualFold(strings.TrimSpace(allowed), targetOrigin) {
			return true
		}
	}
	return false
}

func previewToken(value string) string {
	if len(value) <= 10 {
		return value
	}
	return value[:10] + "..."
}
