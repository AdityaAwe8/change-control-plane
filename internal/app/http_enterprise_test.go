package app_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestOIDCIdentityProviderStartAndCallbackIssueEnterpriseSession(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_OIDC_CLIENT_SECRET_TEST", "super-secret")

	var oidcServer *httptest.Server
	oidcServer = newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oidc/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                 oidcServer.URL + "/oidc",
				"authorization_endpoint": oidcServer.URL + "/oidc/authorize",
				"token_endpoint":         oidcServer.URL + "/oidc/token",
				"userinfo_endpoint":      oidcServer.URL + "/oidc/userinfo",
				"jwks_uri":               oidcServer.URL + "/oidc/jwks",
			})
		case "/oidc/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if got := r.Form.Get("client_id"); got != "oidc-client-123" {
				t.Fatalf("expected client_id oidc-client-123, got %q", got)
			}
			if got := r.Form.Get("client_secret"); got != "super-secret" {
				t.Fatalf("expected client secret from env, got %q", got)
			}
			if got := r.Form.Get("code"); got != "good-code" {
				t.Fatalf("expected authorization code good-code, got %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "access-token-123",
				"token_type":   "Bearer",
				"id_token":     "not-used-in-this-milestone",
			})
		case "/oidc/userinfo":
			if got := r.Header.Get("Authorization"); got != "Bearer access-token-123" {
				t.Fatalf("expected bearer access token, got %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub":                "oidc-user-123",
				"email":              "owner@acme.com",
				"name":               "Acme Owner",
				"preferred_username": "owner@acme.com",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer oidcServer.Close()

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()
	application.Config.APIBaseURL = server.URL

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "bootstrap-owner@acme.local",
		DisplayName:      "Bootstrap Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})

	provider := postItemAuth[types.IdentityProvider](t, server.URL+"/api/v1/identity-providers", types.CreateIdentityProviderRequest{
		OrganizationID:  admin.Session.ActiveOrganizationID,
		Name:            "Acme Okta",
		Kind:            "oidc",
		IssuerURL:       oidcServer.URL + "/oidc",
		ClientID:        "oidc-client-123",
		ClientSecretEnv: "CCP_OIDC_CLIENT_SECRET_TEST",
		AllowedDomains:  []string{"acme.com"},
		DefaultRole:     "org_member",
		Enabled:         true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	testResult := postItemAuth[types.IdentityProviderTestResult](t, server.URL+"/api/v1/identity-providers/"+provider.ID+"/test", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if testResult.Status != "success" {
		t.Fatalf("expected provider test success, got %+v", testResult)
	}

	publicProviders := getPublicList[types.PublicIdentityProvider](t, server.URL+"/api/v1/auth/providers")
	if len(publicProviders) != 1 || publicProviders[0].ID != provider.ID {
		t.Fatalf("expected public identity provider listing, got %+v", publicProviders)
	}

	start := postPublicItem[types.IdentityProviderStartResult](t, server.URL+"/api/v1/auth/providers/"+provider.ID+"/start", types.IdentityProviderStartRequest{
		ReturnTo: "http://127.0.0.1:5173/#/dashboard",
	})
	authorizeURL, err := url.Parse(start.AuthorizeURL)
	if err != nil {
		t.Fatal(err)
	}
	state := authorizeURL.Query().Get("state")
	if strings.TrimSpace(state) == "" {
		t.Fatalf("expected signed state in authorize_url, got %q", start.AuthorizeURL)
	}

	redirectClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	callbackURL := server.URL + "/api/v1/auth/providers/callback?state=" + url.QueryEscape(state) + "&code=good-code"
	resp, err := redirectClient.Get(callbackURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected callback redirect, got %d", resp.StatusCode)
	}
	cookie := findBrowserSessionCookie(resp.Cookies())
	if cookie == nil {
		t.Fatalf("expected browser session cookie on callback redirect, got %+v", resp.Cookies())
	}
	location := resp.Header.Get("Location")
	redirected, err := url.Parse(location)
	if err != nil {
		t.Fatal(err)
	}
	fragmentQuery := redirected.Fragment
	if idx := strings.Index(fragmentQuery, "?"); idx >= 0 {
		fragmentQuery = fragmentQuery[idx+1:]
	} else {
		fragmentQuery = ""
	}
	redirectValues, err := url.ParseQuery(fragmentQuery)
	if err != nil {
		t.Fatal(err)
	}
	if redirectValues.Get("auth_token") != "" {
		t.Fatalf("expected callback redirect to avoid exposed auth token, got %q", location)
	}
	orgID := redirectValues.Get("organization_id")
	if orgID != admin.Session.ActiveOrganizationID {
		t.Fatalf("expected organization id %q, got %q", admin.Session.ActiveOrganizationID, orgID)
	}
	if redirectValues.Get("auth_complete") != "1" {
		t.Fatalf("expected auth completion marker in redirect, got %q", location)
	}

	session := getSessionWithCookie(t, server.URL, cookie, orgID)
	if session.AuthMethod != "oidc" {
		t.Fatalf("expected oidc auth method, got %+v", session)
	}
	if session.AuthProviderID != provider.ID || session.AuthProvider != provider.Name {
		t.Fatalf("expected provider attribution on session, got %+v", session)
	}
	if session.Email != "owner@acme.com" {
		t.Fatalf("expected enterprise email on session, got %+v", session)
	}

	storedProvider, err := application.Store.GetIdentityProvider(context.Background(), provider.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedProvider.LastAuthenticatedAt == nil {
		t.Fatalf("expected provider last_authenticated_at to be recorded, got %+v", storedProvider)
	}
}

func TestIdentityProviderRoutesEnforceScopeAndRBAC(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	member := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "member@acme.local",
		DisplayName:      "Member",
		OrganizationSlug: "acme",
		Roles:            []string{"org_member"},
	})
	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@other.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other Org",
		OrganizationSlug: "other-org",
	})

	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/identity-providers", types.CreateIdentityProviderRequest{
		OrganizationID:  admin.Session.ActiveOrganizationID,
		Name:            "Denied Provider",
		Kind:            "oidc",
		IssuerURL:       "https://issuer.acme.local",
		ClientID:        "client-id",
		ClientSecretEnv: "CCP_OIDC_CLIENT_SECRET_TEST",
	}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member provider create to be forbidden, got %d", status)
	}

	provider := postItemAuth[types.IdentityProvider](t, server.URL+"/api/v1/identity-providers", types.CreateIdentityProviderRequest{
		OrganizationID:  admin.Session.ActiveOrganizationID,
		Name:            "Acme SSO",
		Kind:            "oidc",
		IssuerURL:       "https://issuer.acme.local",
		ClientID:        "client-id",
		ClientSecretEnv: "CCP_OIDC_CLIENT_SECRET_TEST",
		Enabled:         true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/identity-providers/"+provider.ID, types.UpdateIdentityProviderRequest{
		Name: stringPtr("Denied Rename"),
	}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member provider update to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/identity-providers/"+provider.ID+"/test", struct{}{}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member provider test to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/identity-providers/"+provider.ID, types.UpdateIdentityProviderRequest{
		Name: stringPtr("Cross Org Rename"),
	}, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org provider update to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/identity-providers/"+provider.ID+"/test", struct{}{}, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org provider test to be forbidden, got %d", status)
	}

	updated := patchItemAuth[types.IdentityProvider](t, server.URL+"/api/v1/identity-providers/"+provider.ID, types.UpdateIdentityProviderRequest{
		Name: stringPtr("Acme SSO Updated"),
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if updated.Name != "Acme SSO Updated" {
		t.Fatalf("expected admin update to succeed, got %+v", updated)
	}
}

func TestBrowserSessionAdminRoutesEnforceScopeRBACAndCurrentSessionRevocation(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	adminLogin, adminCookie := loginDevWithBrowserSessionCookie(t, server.URL, types.DevLoginRequest{
		Email:            "owner-sessions@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-sessions",
	})
	memberLogin, memberCookie := loginDevWithBrowserSessionCookie(t, server.URL, types.DevLoginRequest{
		Email:            "member-sessions@acme.local",
		DisplayName:      "Member",
		OrganizationSlug: "acme-sessions",
		Roles:            []string{"org_member"},
	})
	otherOrg, otherCookie := loginDevWithBrowserSessionCookie(t, server.URL, types.DevLoginRequest{
		Email:            "owner-sessions@other.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other Org",
		OrganizationSlug: "other-sessions",
	})

	listBody := doJSONWithCookie(t, http.MethodGet, server.URL+"/api/v1/browser-sessions?status=active&limit=10", nil, adminCookie, adminLogin.Session.ActiveOrganizationID, http.StatusOK)
	var listed types.ListResponse[types.BrowserSessionInfo]
	if err := json.Unmarshal(listBody, &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Data) < 2 {
		t.Fatalf("expected same-org browser sessions in admin listing, got %+v", listed.Data)
	}

	var adminSessionID string
	var memberSessionID string
	for _, session := range listed.Data {
		switch session.UserEmail {
		case adminLogin.Session.Email:
			if !session.Current {
				t.Fatalf("expected admin browser session to be marked current, got %+v", session)
			}
			adminSessionID = session.ID
		case memberLogin.Session.Email:
			memberSessionID = session.ID
		case otherOrg.Session.Email:
			t.Fatalf("expected cross-org browser session to be excluded, got %+v", listed.Data)
		}
	}
	if adminSessionID == "" || memberSessionID == "" {
		t.Fatalf("expected admin and member browser sessions in listing, got %+v", listed.Data)
	}

	doJSONWithCookie(t, http.MethodGet, server.URL+"/api/v1/browser-sessions", nil, memberCookie, memberLogin.Session.ActiveOrganizationID, http.StatusForbidden)
	doJSONWithCookieAndHeaders(t, http.MethodPost, server.URL+"/api/v1/browser-sessions/"+memberSessionID+"/revoke", nil, otherCookie, otherOrg.Session.ActiveOrganizationID, map[string]string{
		"Origin": "http://127.0.0.1:5173",
	}, http.StatusForbidden)

	revokeMemberBody := doJSONWithCookieAndHeaders(t, http.MethodPost, server.URL+"/api/v1/browser-sessions/"+memberSessionID+"/revoke", nil, adminCookie, adminLogin.Session.ActiveOrganizationID, map[string]string{
		"Origin": "http://127.0.0.1:5173",
	}, http.StatusOK)
	var revokedMember types.ItemResponse[types.BrowserSessionInfo]
	if err := json.Unmarshal(revokeMemberBody, &revokedMember); err != nil {
		t.Fatal(err)
	}
	if revokedMember.Data.Status != "revoked" || revokedMember.Data.Current {
		t.Fatalf("expected member browser session revoke to succeed, got %+v", revokedMember.Data)
	}
	storedMemberSession, err := application.Store.GetBrowserSession(t.Context(), memberSessionID)
	if err != nil {
		t.Fatal(err)
	}
	if storedMemberSession.RevokedAt == nil {
		t.Fatalf("expected member browser session to persist revocation, got %+v", storedMemberSession)
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/browser-sessions/"+adminSessionID+"/revoke", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(adminCookie)
	request.Header.Set("X-CCP-Organization-ID", adminLogin.Session.ActiveOrganizationID)
	request.Header.Set("Origin", "http://127.0.0.1:5173")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected current browser session revoke to succeed, got %d", response.StatusCode)
	}
	clearedCookie := findBrowserSessionCookie(response.Cookies())
	if clearedCookie == nil || clearedCookie.Value != "" || clearedCookie.MaxAge >= 0 {
		t.Fatalf("expected current session revoke to clear browser session cookie, got %+v", response.Cookies())
	}

	doJSONWithCookie(t, http.MethodGet, server.URL+"/api/v1/auth/session", nil, adminCookie, adminLogin.Session.ActiveOrganizationID, http.StatusUnauthorized)
}

func TestWebhookRegistrationSyncAndDeliveryHealthForGitHub(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_GITHUB_TOKEN_TEST", "ghs-test-token")
	t.Setenv("CCP_GITHUB_WEBHOOK_SECRET_TEST", "hook-secret")

	githubServer := newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/orgs/acme/hooks":
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]map[string]any{})
				return
			}
			if r.Method == http.MethodPost {
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 4242})
				return
			}
			t.Fatalf("unexpected github hooks method %s", r.Method)
		case "/orgs/acme":
			_ = json.NewEncoder(w).Encode(map[string]any{"login": "acme"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer githubServer.Close()

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()
	application.Config.APIBaseURL = server.URL

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var githubIntegration types.Integration
	for _, integration := range integrations {
		if integration.Kind == "github" {
			githubIntegration = integration
			break
		}
	}
	if githubIntegration.ID == "" {
		t.Fatal("expected github integration")
	}

	_ = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID, types.UpdateIntegrationRequest{
		Enabled: boolPtr(true),
		Mode:    stringPtr("advisory"),
		Metadata: types.Metadata{
			"api_base_url":       githubServer.URL,
			"owner":              "acme",
			"access_token_env":   "CCP_GITHUB_TOKEN_TEST",
			"webhook_secret_env": "CCP_GITHUB_WEBHOOK_SECRET_TEST",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	registration := postItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/webhook-registration/sync", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if registration.Registration.Status != "registered" || registration.Registration.ExternalHookID == "" {
		t.Fatalf("expected registered webhook, got %+v", registration.Registration)
	}

	fetched := getItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/webhook-registration", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if fetched.Registration.CallbackURL == "" {
		t.Fatalf("expected stored callback url, got %+v", fetched.Registration)
	}

	payload := []byte(`{"ref":"refs/heads/main","after":"abc123","repository":{"name":"checkout","full_name":"acme/checkout","html_url":"https://github.com/acme/checkout","default_branch":"main","owner":{"login":"acme"}}}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/webhooks/github", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "push")
	req.Header.Set("X-GitHub-Delivery", "delivery-invalid")
	req.Header.Set("X-Hub-Signature-256", "sha256=deadbeef")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected invalid signature to be forbidden, got %d", resp.StatusCode)
	}

	afterInvalid := getItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/webhook-registration", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if afterInvalid.Registration.DeliveryHealth != "error" {
		t.Fatalf("expected error delivery health after invalid signature, got %+v", afterInvalid.Registration)
	}

	mac := hmac.New(sha256.New, []byte("hook-secret"))
	_, _ = mac.Write(payload)
	validReq, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/webhooks/github", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	validReq.Header.Set("Content-Type", "application/json")
	validReq.Header.Set("X-GitHub-Event", "push")
	validReq.Header.Set("X-GitHub-Delivery", "delivery-valid")
	validReq.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	validResp, err := http.DefaultClient.Do(validReq)
	if err != nil {
		t.Fatal(err)
	}
	validResp.Body.Close()
	if validResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid webhook to be accepted, got %d", validResp.StatusCode)
	}

	afterValid := getItemAuth[types.WebhookRegistrationResult](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/webhook-registration", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if afterValid.Registration.DeliveryHealth != "healthy" {
		t.Fatalf("expected healthy delivery after valid webhook, got %+v", afterValid.Registration)
	}
	if afterValid.Registration.LastDeliveryAt == nil {
		t.Fatalf("expected last_delivery_at to be recorded, got %+v", afterValid.Registration)
	}
}

func TestOIDCCallbackDoesNotRedirectToUnsafeReturnToOrigin(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_OIDC_CLIENT_SECRET_TEST", "super-secret")

	var oidcServer *httptest.Server
	oidcServer = newLocalIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oidc/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":                 oidcServer.URL + "/oidc",
				"authorization_endpoint": oidcServer.URL + "/oidc/authorize",
				"token_endpoint":         oidcServer.URL + "/oidc/token",
				"userinfo_endpoint":      oidcServer.URL + "/oidc/userinfo",
			})
		case "/oidc/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "access-token-unsafe",
				"token_type":   "Bearer",
			})
		case "/oidc/userinfo":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub":   "oidc-user-unsafe",
				"email": "owner@acme.com",
				"name":  "Acme Owner",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer oidcServer.Close()

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()
	application.Config.APIBaseURL = server.URL

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "bootstrap-owner@acme.local",
		DisplayName:      "Bootstrap Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-unsafe",
	})

	provider := postItemAuth[types.IdentityProvider](t, server.URL+"/api/v1/identity-providers", types.CreateIdentityProviderRequest{
		OrganizationID:  admin.Session.ActiveOrganizationID,
		Name:            "Acme Okta Unsafe",
		Kind:            "oidc",
		IssuerURL:       oidcServer.URL + "/oidc",
		ClientID:        "oidc-client-unsafe",
		ClientSecretEnv: "CCP_OIDC_CLIENT_SECRET_TEST",
		AllowedDomains:  []string{"acme.com"},
		DefaultRole:     "org_member",
		Enabled:         true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	start := postPublicItem[types.IdentityProviderStartResult](t, server.URL+"/api/v1/auth/providers/"+provider.ID+"/start", types.IdentityProviderStartRequest{
		ReturnTo: "https://evil.example/phish",
	})
	authorizeURL, err := url.Parse(start.AuthorizeURL)
	if err != nil {
		t.Fatal(err)
	}
	state := authorizeURL.Query().Get("state")

	redirectClient := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := redirectClient.Get(server.URL + "/api/v1/auth/providers/callback?state=" + url.QueryEscape(state) + "&code=good-code")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected callback redirect, got %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")
	if strings.Contains(location, "evil.example") {
		t.Fatalf("expected unsafe return_to to be stripped, got %q", location)
	}
	if !strings.HasPrefix(location, "/?auth_complete=1") {
		t.Fatalf("expected callback to fall back to safe relative redirect, got %q", location)
	}
}

func TestOutboxRecoveryRoutesResetDispatchStatePreserveForensicsAndStayDispatchable(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "outbox-admin@acme.local",
		DisplayName:      "Outbox Admin",
		OrganizationName: "Acme Outbox",
		OrganizationSlug: "acme-outbox",
	})

	handled := map[string]int{}
	application.Events.Subscribe("outbox.retry.test", func(_ context.Context, event types.DomainEvent) error {
		handled[event.ID]++
		return nil
	})
	application.Events.Subscribe("outbox.requeue.test", func(_ context.Context, event types.DomainEvent) error {
		handled[event.ID]++
		return nil
	})

	retryID := "evt_retry_123"
	requeueID := "evt_requeue_123"
	mustCreateOutboxEvent(t, application, types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        retryID,
			CreatedAt: time.Now().UTC().Add(-5 * time.Minute),
			UpdatedAt: time.Now().UTC().Add(-2 * time.Minute),
			Metadata: types.Metadata{
				"last_error_class": "temporary",
				"recovery_hint":    "check upstream dependency health before forcing an immediate retry",
			},
		},
		EventType:      "outbox.retry.test",
		OrganizationID: admin.Session.ActiveOrganizationID,
		ResourceType:   "integration",
		ResourceID:     "integration_retry",
		Status:         "error",
		Attempts:       2,
		LastError:      "temporary dispatch failure",
		NextAttemptAt:  ptrTime(time.Now().UTC().Add(10 * time.Minute)),
		ClaimedAt:      ptrTime(time.Now().UTC().Add(-30 * time.Second)),
	})
	mustCreateOutboxEvent(t, application, types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        requeueID,
			CreatedAt: time.Now().UTC().Add(-10 * time.Minute),
			UpdatedAt: time.Now().UTC().Add(-1 * time.Minute),
			Metadata: types.Metadata{
				"last_error_class": "permanent",
				"dead_lettered_at": time.Now().UTC().Add(-1 * time.Minute).Format(time.RFC3339Nano),
				"recovery_hint":    "fix the handler or payload before replaying this event",
			},
		},
		EventType:      "outbox.requeue.test",
		OrganizationID: admin.Session.ActiveOrganizationID,
		ResourceType:   "webhook",
		ResourceID:     "delivery_123",
		Status:         "dead_letter",
		Attempts:       5,
		LastError:      "permanent payload failure",
		ClaimedAt:      ptrTime(time.Now().UTC().Add(-90 * time.Second)),
	})

	retried := postItemAuth[types.OutboxEvent](t, server.URL+"/api/v1/outbox-events/"+retryID+"/retry", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if retried.Status != "pending" || retried.NextAttemptAt != nil || retried.ClaimedAt != nil {
		t.Fatalf("expected retried outbox event to become pending and unclaimed, got %+v", retried)
	}
	if retried.Attempts != 2 || retried.LastError != "temporary dispatch failure" {
		t.Fatalf("expected retry to preserve attempts and failure details, got %+v", retried)
	}
	retryHistory, ok := retried.Metadata["manual_recovery_history"].([]any)
	if !ok || len(retryHistory) != 1 {
		t.Fatalf("expected retry recovery history metadata, got %+v", retried.Metadata)
	}

	requeued := postItemAuth[types.OutboxEvent](t, server.URL+"/api/v1/outbox-events/"+requeueID+"/requeue", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if requeued.Status != "pending" || requeued.NextAttemptAt != nil || requeued.ClaimedAt != nil {
		t.Fatalf("expected requeued outbox event to become pending and unclaimed, got %+v", requeued)
	}
	if requeued.Attempts != 5 || requeued.LastError != "permanent payload failure" {
		t.Fatalf("expected requeue to preserve attempts and failure details, got %+v", requeued)
	}

	retriedStored, err := application.Store.GetOutboxEvent(context.Background(), retryID)
	if err != nil {
		t.Fatal(err)
	}
	if retriedStored.Status != "pending" || retriedStored.ProcessedAt != nil {
		t.Fatalf("expected stored retry event to remain pending before dispatch, got %+v", retriedStored)
	}
	requeuedStored, err := application.Store.GetOutboxEvent(context.Background(), requeueID)
	if err != nil {
		t.Fatal(err)
	}
	if requeuedStored.Status != "pending" || requeuedStored.ProcessedAt != nil {
		t.Fatalf("expected stored requeue event to remain pending before dispatch, got %+v", requeuedStored)
	}

	auditEvents, err := application.Store.ListAuditEvents(context.Background(), storage.AuditEventQuery{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Limit:          100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsAuditAction(auditEvents, "outbox_event.retry") || !containsAuditAction(auditEvents, "outbox_event.requeue") {
		t.Fatalf("expected audit trail to record recovery actions, got %+v", auditEvents)
	}

	if _, err := application.Events.DispatchPending(context.Background(), 100); err != nil {
		t.Fatal(err)
	}
	retriedProcessed, err := application.Store.GetOutboxEvent(context.Background(), retryID)
	if err != nil {
		t.Fatal(err)
	}
	requeuedProcessed, err := application.Store.GetOutboxEvent(context.Background(), requeueID)
	if err != nil {
		t.Fatal(err)
	}
	if retriedProcessed.Status != "processed" || retriedProcessed.ProcessedAt == nil || handled[retryID] != 1 {
		t.Fatalf("expected retried event to become dispatchable and processed, got %+v handled=%d", retriedProcessed, handled[retryID])
	}
	if requeuedProcessed.Status != "processed" || requeuedProcessed.ProcessedAt == nil || handled[requeueID] != 1 {
		t.Fatalf("expected requeued event to become dispatchable and processed, got %+v handled=%d", requeuedProcessed, handled[requeueID])
	}
	if history, ok := retriedProcessed.Metadata["manual_recovery_history"].([]any); !ok || len(history) != 1 {
		t.Fatalf("expected retry recovery history to remain after success, got %+v", retriedProcessed.Metadata)
	}
	if history, ok := requeuedProcessed.Metadata["manual_recovery_history"].([]any); !ok || len(history) != 1 {
		t.Fatalf("expected requeue recovery history to remain after success, got %+v", requeuedProcessed.Metadata)
	}
}

func TestOutboxRecoveryRoutesEnforceScopeRBACAndSupportedStatuses(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "outbox-owner@acme.local",
		DisplayName:      "Outbox Owner",
		OrganizationName: "Acme Scope",
		OrganizationSlug: "acme-scope",
	})
	member := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "outbox-member@acme.local",
		DisplayName:      "Outbox Member",
		OrganizationSlug: "acme-scope",
		Roles:            []string{"org_member"},
	})
	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "other-owner@acme.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other Org",
		OrganizationSlug: "other-org",
	})

	retryID := "evt_retry_scope_123"
	processedID := "evt_processed_scope_123"
	mustCreateOutboxEvent(t, application, types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        retryID,
			CreatedAt: time.Now().UTC().Add(-2 * time.Minute),
			UpdatedAt: time.Now().UTC().Add(-1 * time.Minute),
		},
		EventType:      "outbox.retry.scope.test",
		OrganizationID: admin.Session.ActiveOrganizationID,
		ResourceType:   "integration",
		ResourceID:     "integration_scope",
		Status:         "error",
		Attempts:       1,
		LastError:      "temporary dispatch failure",
	})
	mustCreateOutboxEvent(t, application, types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        processedID,
			CreatedAt: time.Now().UTC().Add(-2 * time.Minute),
			UpdatedAt: time.Now().UTC().Add(-1 * time.Minute),
		},
		EventType:      "outbox.processed.scope.test",
		OrganizationID: admin.Session.ActiveOrganizationID,
		ResourceType:   "status_event",
		ResourceID:     "status_scope",
		Status:         "processed",
		Attempts:       1,
		ProcessedAt:    ptrTime(time.Now().UTC().Add(-30 * time.Second)),
	})

	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/outbox-events/"+retryID+"/retry", nil, member.Token, member.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org member retry to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/outbox-events/"+retryID+"/retry", nil, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org retry to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/outbox-events/"+processedID+"/retry", nil, admin.Token, admin.Session.ActiveOrganizationID); status != http.StatusBadRequest {
		t.Fatalf("expected processed-event retry to be rejected, got %d", status)
	}

	auditEvents, err := application.Store.ListAuditEvents(context.Background(), storage.AuditEventQuery{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Limit:          100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsAuditAction(auditEvents, "outbox_event.retry.denied") {
		t.Fatalf("expected denied retry audit event, got %+v", auditEvents)
	}
}

func TestOutboxRecoveryRoutesRejectRepeatedRecoveryAttemptsWithTruthfulCurrentStatus(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "outbox-repeat@acme.local",
		DisplayName:      "Outbox Repeat",
		OrganizationName: "Acme Repeat",
		OrganizationSlug: "acme-repeat",
	})

	now := time.Now().UTC()
	retryID := "evt_retry_repeat_123"
	requeueID := "evt_requeue_repeat_123"
	mustCreateOutboxEvent(t, application, types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        retryID,
			CreatedAt: now.Add(-2 * time.Minute),
			UpdatedAt: now.Add(-1 * time.Minute),
		},
		EventType:      "outbox.retry.repeat.test",
		OrganizationID: admin.Session.ActiveOrganizationID,
		ResourceType:   "integration",
		ResourceID:     "integration_repeat",
		Status:         "error",
		Attempts:       2,
		LastError:      "temporary dispatch failure",
	})
	mustCreateOutboxEvent(t, application, types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        requeueID,
			CreatedAt: now.Add(-3 * time.Minute),
			UpdatedAt: now.Add(-90 * time.Second),
			Metadata: types.Metadata{
				"dead_lettered_at": now.Add(-90 * time.Second).Format(time.RFC3339Nano),
			},
		},
		EventType:      "outbox.requeue.repeat.test",
		OrganizationID: admin.Session.ActiveOrganizationID,
		ResourceType:   "webhook",
		ResourceID:     "delivery_repeat",
		Status:         "dead_letter",
		Attempts:       5,
		LastError:      "permanent payload failure",
	})

	postItemAuth[types.OutboxEvent](t, server.URL+"/api/v1/outbox-events/"+retryID+"/retry", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	retryErrBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/outbox-events/"+retryID+"/retry", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusBadRequest)
	var retryErr types.ErrorResponse
	if err := json.Unmarshal(retryErrBody, &retryErr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(retryErr.Error.Message, "current status is pending") {
		t.Fatalf("expected repeat retry to report pending status truthfully, got %+v", retryErr)
	}

	postItemAuth[types.OutboxEvent](t, server.URL+"/api/v1/outbox-events/"+requeueID+"/requeue", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	requeueErrBody := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/outbox-events/"+requeueID+"/requeue", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusBadRequest)
	var requeueErr types.ErrorResponse
	if err := json.Unmarshal(requeueErrBody, &requeueErr); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(requeueErr.Error.Message, "current status is pending") {
		t.Fatalf("expected repeat requeue to report pending status truthfully, got %+v", requeueErr)
	}

	retried, err := application.Store.GetOutboxEvent(context.Background(), retryID)
	if err != nil {
		t.Fatal(err)
	}
	if retried.Status != "pending" {
		t.Fatalf("expected repeated retry attempt to leave event pending, got %+v", retried)
	}
	requeued, err := application.Store.GetOutboxEvent(context.Background(), requeueID)
	if err != nil {
		t.Fatal(err)
	}
	if requeued.Status != "pending" {
		t.Fatalf("expected repeated requeue attempt to leave event pending, got %+v", requeued)
	}

	auditEvents, err := application.Store.ListAuditEvents(context.Background(), storage.AuditEventQuery{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Limit:          100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if countAuditAction(auditEvents, "outbox_event.retry") != 1 {
		t.Fatalf("expected exactly one successful retry audit record, got %+v", auditEvents)
	}
	if countAuditAction(auditEvents, "outbox_event.requeue") != 1 {
		t.Fatalf("expected exactly one successful requeue audit record, got %+v", auditEvents)
	}
}

func TestOutboxRecoveryRoutesRejectRecoveryWhenWorkerClaimsDuringCommitAndStaleReclaimStillDispatches(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	cfg := common.LoadConfig()
	baseStore := app.NewInMemoryStore()
	racingStore := &raceOnOutboxRecoveryStore{Store: baseStore}
	application := app.NewApplicationWithStore(cfg, racingStore)
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "outbox-race@acme.local",
		DisplayName:      "Outbox Race",
		OrganizationName: "Acme Race",
		OrganizationSlug: "acme-race",
	})

	handled := 0
	application.Events.Subscribe("outbox.retry.race.test", func(_ context.Context, _ types.DomainEvent) error {
		handled++
		return nil
	})

	retryID := "evt_retry_race_123"
	mustCreateOutboxEvent(t, application, types.OutboxEvent{
		BaseRecord: types.BaseRecord{
			ID:        retryID,
			CreatedAt: time.Now().UTC().Add(-2 * time.Minute),
			UpdatedAt: time.Now().UTC().Add(-1 * time.Minute),
		},
		EventType:      "outbox.retry.race.test",
		OrganizationID: admin.Session.ActiveOrganizationID,
		ResourceType:   "integration",
		ResourceID:     "integration_race",
		Status:         "error",
		Attempts:       1,
		LastError:      "temporary dispatch failure",
	})

	racingStore.beforeConditionalUpdate = func(ctx context.Context) {
		claimed, err := baseStore.ClaimOutboxEvents(ctx, time.Now().UTC(), 1, time.Now().UTC().Add(time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if len(claimed) != 1 || claimed[0].ID != retryID || claimed[0].Status != "processing" {
			t.Fatalf("expected worker-claim simulation to move the retry event into processing, got %+v", claimed)
		}
	}

	body := doAuthenticatedJSON(t, http.MethodPost, server.URL+"/api/v1/outbox-events/"+retryID+"/retry", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusBadRequest)
	var response types.ErrorResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Error.Message, "current status is processing") {
		t.Fatalf("expected racing retry to report processing status truthfully, got %+v", response)
	}

	processing, err := application.Store.GetOutboxEvent(context.Background(), retryID)
	if err != nil {
		t.Fatal(err)
	}
	if processing.Status != "processing" || processing.ClaimedAt == nil {
		t.Fatalf("expected racing retry to leave the event in processing, got %+v", processing)
	}

	staleClaim := time.Now().UTC().Add(-5 * time.Minute)
	processing.ClaimedAt = &staleClaim
	processing.UpdatedAt = staleClaim
	if err := application.Store.UpdateOutboxEvent(context.Background(), processing); err != nil {
		t.Fatal(err)
	}

	dispatched, err := application.Events.DispatchPending(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if dispatched < 1 || handled != 1 {
		t.Fatalf("expected stale processing event to be reclaimed and dispatched after rejected retry, dispatched=%d handled=%d", dispatched, handled)
	}

	processed, err := application.Store.GetOutboxEvent(context.Background(), retryID)
	if err != nil {
		t.Fatal(err)
	}
	if processed.Status != "processed" || processed.ProcessedAt == nil {
		t.Fatalf("expected reclaimed event to finish as processed, got %+v", processed)
	}

	auditEvents, err := application.Store.ListAuditEvents(context.Background(), storage.AuditEventQuery{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Limit:          100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if containsAuditAction(auditEvents, "outbox_event.retry") {
		t.Fatalf("expected racing retry to avoid recording a success audit event, got %+v", auditEvents)
	}
}

func mustCreateOutboxEvent(t *testing.T, application *app.Application, event types.OutboxEvent) {
	t.Helper()
	if err := application.Store.CreateOutboxEvent(context.Background(), event); err != nil {
		t.Fatal(err)
	}
}

func containsAuditAction(events []types.AuditEvent, action string) bool {
	for _, event := range events {
		if event.Action == action {
			return true
		}
	}
	return false
}

func countAuditAction(events []types.AuditEvent, action string) int {
	count := 0
	for _, event := range events {
		if event.Action == action {
			count++
		}
	}
	return count
}

type raceOnOutboxRecoveryStore struct {
	storage.Store
	beforeConditionalUpdate func(context.Context)
	triggered               bool
}

func (s *raceOnOutboxRecoveryStore) UpdateOutboxEventIfStatus(ctx context.Context, event types.OutboxEvent, expectedStatus string) (bool, error) {
	if !s.triggered && s.beforeConditionalUpdate != nil {
		s.triggered = true
		s.beforeConditionalUpdate(ctx)
	}
	return s.Store.UpdateOutboxEventIfStatus(ctx, event, expectedStatus)
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func getPublicList[T any](t *testing.T, requestURL string) []T {
	t.Helper()
	resp, err := http.Get(requestURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	var envelope types.ListResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func postPublicItem[T any](t *testing.T, requestURL string, body any) T {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(requestURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
	var envelope types.ItemResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}
