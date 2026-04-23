package app_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestProjectsRequireAuthentication(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/projects")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestCORSPreflightAllowedInDevelopment(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENV", "development")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	req, err := http.NewRequest(http.MethodOptions, server.URL+"/api/v1/projects", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	req.Header.Set("Access-Control-Request-Headers", "authorization,content-type,x-ccp-organization-id")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "http://127.0.0.1:5173" {
		t.Fatalf("expected echoed allow-origin, got %q", resp.Header.Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(resp.Header.Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("expected authorization header to be allowed, got %q", resp.Header.Get("Access-Control-Allow-Headers"))
	}
	if resp.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("expected credentials support, got %q", resp.Header.Get("Access-Control-Allow-Credentials"))
	}
}

func TestDisallowedCORSOriginRejectedWhenConfigured(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENV", "production")
	t.Setenv("CCP_ALLOWED_ORIGINS", "https://console.acme.local")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	req, err := http.NewRequest(http.MethodOptions, server.URL+"/api/v1/projects", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://evil.local")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestSignUpCreatesPasswordAccountWithoutOrganizationScope(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	response := signUp(t, server.URL, types.SignUpRequest{
		Email:                "owner@acme.local",
		DisplayName:          "Owner",
		Password:             "ChangeMe123!",
		PasswordConfirmation: "ChangeMe123!",
	})

	if response.Session.ActiveOrganizationID != "" {
		t.Fatalf("expected no active organization, got %q", response.Session.ActiveOrganizationID)
	}
	if len(response.Session.Organizations) != 0 {
		t.Fatalf("expected no organizations, got %d", len(response.Session.Organizations))
	}
}

func TestSignInWithPasswordRestoresSession(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	signUp(t, server.URL, types.SignUpRequest{
		Email:                "owner@acme.local",
		DisplayName:          "Owner",
		Password:             "ChangeMe123!",
		PasswordConfirmation: "ChangeMe123!",
	})

	response := signIn(t, server.URL, types.SignInRequest{
		Email:    "owner@acme.local",
		Password: "ChangeMe123!",
	})

	if response.Session.Email != "owner@acme.local" {
		t.Fatalf("expected signed-in email, got %q", response.Session.Email)
	}
}

func TestPasswordSignInEstablishesHttpOnlyBrowserSessionAndLogoutRevokesIt(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	signUp(t, server.URL, types.SignUpRequest{
		Email:                "owner@acme.local",
		DisplayName:          "Owner",
		Password:             "ChangeMe123!",
		PasswordConfirmation: "ChangeMe123!",
	})

	response, cookie := signInWithBrowserSessionCookie(t, server.URL, types.SignInRequest{
		Email:    "owner@acme.local",
		Password: "ChangeMe123!",
	})
	if cookie == nil {
		t.Fatal("expected browser session cookie from sign-in")
	}
	if !cookie.HttpOnly {
		t.Fatalf("expected browser session cookie to be HttpOnly, got %+v", cookie)
	}
	if response.Session.Email != "owner@acme.local" {
		t.Fatalf("expected password session email, got %+v", response.Session)
	}

	session := getSessionWithCookie(t, server.URL, cookie, "")
	if !session.Authenticated || session.Email != "owner@acme.local" {
		t.Fatalf("expected cookie-backed session, got %+v", session)
	}

	logoutBody := doJSONWithCookieAndHeaders(t, http.MethodPost, server.URL+"/api/v1/auth/logout", nil, cookie, "", map[string]string{
		"Origin": "http://127.0.0.1:5173",
	}, http.StatusOK)
	var logoutEnvelope types.ItemResponse[types.SessionInfo]
	if err := json.Unmarshal(logoutBody, &logoutEnvelope); err != nil {
		t.Fatal(err)
	}
	if logoutEnvelope.Data.Authenticated {
		t.Fatalf("expected logout to return anonymous session, got %+v", logoutEnvelope.Data)
	}

	sessionHash := application.Auth.TokenService().HashOpaqueToken(cookie.Value)
	storedSession, err := application.Store.GetBrowserSessionByHash(t.Context(), sessionHash)
	if err != nil {
		t.Fatal(err)
	}
	if storedSession.RevokedAt == nil {
		t.Fatalf("expected browser session revocation to be persisted, got %+v", storedSession)
	}

	doJSONWithCookie(t, http.MethodGet, server.URL+"/api/v1/auth/session", nil, cookie, "", http.StatusUnauthorized)
}

func TestDevLoginEstablishesBrowserSessionCookieAndSessionEndpointUsesIt(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	response, cookie := loginDevWithBrowserSessionCookie(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if cookie == nil {
		t.Fatal("expected browser session cookie from dev login")
	}
	session := getSessionWithCookie(t, server.URL, cookie, response.Session.ActiveOrganizationID)
	if !session.Authenticated || session.ActiveOrganizationID != response.Session.ActiveOrganizationID {
		t.Fatalf("expected dev-login cookie session, got %+v", session)
	}
	if session.AuthMethod != "dev_bootstrap" {
		t.Fatalf("expected dev bootstrap auth method, got %+v", session)
	}
}

func TestExpiredAndRevokedBrowserSessionsAreRejected(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})

	expiredCookie := mustCreateBrowserSessionCookie(t, application, admin.Session.ActorID, "dev_bootstrap", "", "", time.Now().UTC().Add(-5*time.Minute), nil)
	doJSONWithCookie(t, http.MethodGet, server.URL+"/api/v1/auth/session", nil, expiredCookie, admin.Session.ActiveOrganizationID, http.StatusUnauthorized)

	revokedAt := time.Now().UTC().Add(-2 * time.Minute)
	revokedCookie := mustCreateBrowserSessionCookie(t, application, admin.Session.ActorID, "dev_bootstrap", "", "", time.Now().UTC().Add(30*time.Minute), &revokedAt)
	doJSONWithCookie(t, http.MethodGet, server.URL+"/api/v1/auth/session", nil, revokedCookie, admin.Session.ActiveOrganizationID, http.StatusUnauthorized)
}

func TestCookieAuthenticatedMutationRejectsDisallowedOrigin(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENV", "production")
	t.Setenv("CCP_ALLOWED_ORIGINS", "https://console.acme.local")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	login, cookie := loginDevWithBrowserSessionCookie(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if cookie == nil {
		t.Fatal("expected browser session cookie from dev login")
	}

	doJSONWithCookieAndHeaders(t, http.MethodPost, server.URL+"/api/v1/auth/logout", nil, cookie, login.Session.ActiveOrganizationID, map[string]string{
		"Origin":  "https://evil.local",
		"Referer": "https://evil.local/console",
	}, http.StatusForbidden)

	session := getSessionWithCookie(t, server.URL, cookie, login.Session.ActiveOrganizationID)
	if !session.Authenticated {
		t.Fatalf("expected disallowed-origin logout to leave session active, got %+v", session)
	}
}

func TestCookieAuthenticatedMutationAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENV", "production")
	t.Setenv("CCP_ALLOWED_ORIGINS", "https://console.acme.local")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	login, cookie := loginDevWithBrowserSessionCookie(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if cookie == nil {
		t.Fatal("expected browser session cookie from dev login")
	}

	doJSONWithCookieAndHeaders(t, http.MethodPost, server.URL+"/api/v1/auth/logout", nil, cookie, login.Session.ActiveOrganizationID, map[string]string{
		"Origin":  "https://console.acme.local",
		"Referer": "https://console.acme.local/control-plane",
	}, http.StatusOK)
}

func TestBearerAndAPITokenAuthStillWorkAfterBrowserSessionHardening(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENV", "production")
	t.Setenv("CCP_ALLOWED_ORIGINS", "https://console.acme.local")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})

	session := getItemAuth[types.SessionInfo](t, server.URL+"/api/v1/auth/session", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if !session.Authenticated || session.Email != admin.Session.Email {
		t.Fatalf("expected bearer session lookup to keep working, got %+v", session)
	}

	serviceAccount := postItemAuth[types.ServiceAccount](t, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "browser-hardening-bot",
		Role:           "org_member",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	issued := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name: "primary",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = getListAuth[types.Service](t, server.URL+"/api/v1/services", issued.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Bearer Project",
		Slug:           "bearer-project",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if project.Slug != "bearer-project" {
		t.Fatalf("expected bearer-authenticated mutation to keep working without browser origin headers, got %+v", project)
	}
}

func TestSignInRejectsInvalidCredentialsWithHelpfulMessage(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	payload, err := json.Marshal(types.SignInRequest{
		Email:    "missing@acme.local",
		Password: "wrong-password",
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(server.URL+"/api/v1/auth/sign-in", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	var envelope types.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(envelope.Error.Message, "invalid email or password") {
		t.Fatalf("expected invalid credential message, got %q", envelope.Error.Message)
	}
}

func TestDemoAdminCanSignInWithDefaultPassword(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	response := signIn(t, server.URL, types.SignInRequest{
		Email:    "admin@changecontrolplane.local",
		Password: "ChangeMe123!",
	})

	if response.Session.Email != "admin@changecontrolplane.local" {
		t.Fatalf("expected demo admin email, got %q", response.Session.Email)
	}
	if response.Session.ActiveOrganizationID == "" {
		t.Fatalf("expected demo admin organization scope to be seeded")
	}
}

func TestSignUpClaimsExistingMembershipWithoutOrgFields(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "member@acme.local",
		DisplayName:      "Member",
		OrganizationSlug: "acme",
	})

	response := signUp(t, server.URL, types.SignUpRequest{
		Email:                "member@acme.local",
		DisplayName:          "Member",
		Password:             "ChangeMe123!",
		PasswordConfirmation: "ChangeMe123!",
	})

	if response.Session.ActiveOrganizationID != admin.Session.ActiveOrganizationID {
		t.Fatalf("expected claimed account to keep organization access")
	}

	signedIn := signIn(t, server.URL, types.SignInRequest{
		Email:    "member@acme.local",
		Password: "ChangeMe123!",
	})
	if signedIn.Session.ActiveOrganizationID != admin.Session.ActiveOrganizationID {
		t.Fatalf("expected password sign-in to retain organization scope")
	}
}

func TestCreateProjectRejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})

	body := `{"organization_id":"` + admin.Session.ActiveOrganizationID + `","name":"Platform","slug":"platform","unexpected":"value"}`
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/projects", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+admin.Token)
	req.Header.Set("X-CCP-Organization-ID", admin.Session.ActiveOrganizationID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d", resp.StatusCode)
	}

	body = `{"organization_id":"` + admin.Session.ActiveOrganizationID + `","name":"Platform","slug":"platform"}{"extra":true}`
	req, err = http.NewRequest(http.MethodPost, server.URL+"/api/v1/projects", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+admin.Token)
	req.Header.Set("X-CCP-Organization-ID", admin.Session.ActiveOrganizationID)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for trailing json payload, got %d", resp.StatusCode)
	}
}

func TestCrossTenantProjectScopeDenied(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	loginA := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-a@acme.local",
		DisplayName:      "Owner A",
		OrganizationName: "Acme A",
		OrganizationSlug: "acme-a",
	})
	loginB := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-b@acme.local",
		DisplayName:      "Owner B",
		OrganizationName: "Acme B",
		OrganizationSlug: "acme-b",
	})

	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/projects", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+loginA.Token)
	req.Header.Set("X-CCP-Organization-ID", loginB.Session.ActiveOrganizationID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func TestOrgMemberCannotCreateProject(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
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
	})

	reqBody, err := json.Marshal(types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Denied Project",
		Slug:           "denied-project",
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/projects", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+member.Token)
	req.Header.Set("X-CCP-Organization-ID", admin.Session.ActiveOrganizationID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func loginDev(t *testing.T, serverURL string, request types.DevLoginRequest) types.DevLoginResponse {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(serverURL+"/api/v1/auth/dev/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("unexpected login status %d", resp.StatusCode)
	}

	var envelope types.ItemResponse[types.DevLoginResponse]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func signUp(t *testing.T, serverURL string, request types.SignUpRequest) types.AuthResponse {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(serverURL+"/api/v1/auth/sign-up", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("unexpected sign-up status %d", resp.StatusCode)
	}

	var envelope types.ItemResponse[types.AuthResponse]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func signIn(t *testing.T, serverURL string, request types.SignInRequest) types.AuthResponse {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(serverURL+"/api/v1/auth/sign-in", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("unexpected sign-in status %d", resp.StatusCode)
	}

	var envelope types.ItemResponse[types.AuthResponse]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func signInWithBrowserSessionCookie(t *testing.T, serverURL string, request types.SignInRequest) (types.AuthResponse, *http.Cookie) {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(serverURL+"/api/v1/auth/sign-in", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("unexpected sign-in status %d", resp.StatusCode)
	}

	var envelope types.ItemResponse[types.AuthResponse]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data, findBrowserSessionCookie(resp.Cookies())
}

func loginDevWithBrowserSessionCookie(t *testing.T, serverURL string, request types.DevLoginRequest) (types.DevLoginResponse, *http.Cookie) {
	t.Helper()
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(serverURL+"/api/v1/auth/dev/login", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("unexpected login status %d", resp.StatusCode)
	}

	var envelope types.ItemResponse[types.DevLoginResponse]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data, findBrowserSessionCookie(resp.Cookies())
}

func getSessionWithCookie(t *testing.T, serverURL string, cookie *http.Cookie, organizationID string) types.SessionInfo {
	t.Helper()
	body := doJSONWithCookie(t, http.MethodGet, serverURL+"/api/v1/auth/session", nil, cookie, organizationID, http.StatusOK)
	var envelope types.ItemResponse[types.SessionInfo]
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func doJSONWithCookie(t *testing.T, method, url string, body any, cookie *http.Cookie, organizationID string, expectedStatus int) []byte {
	t.Helper()
	return doJSONWithCookieAndHeaders(t, method, url, body, cookie, organizationID, nil, expectedStatus)
}

func doJSONWithCookieAndHeaders(t *testing.T, method, url string, body any, cookie *http.Cookie, organizationID string, headers map[string]string, expectedStatus int) []byte {
	t.Helper()
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	if organizationID != "" {
		request.Header.Set("X-CCP-Organization-ID", organizationID)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, response.StatusCode)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func findBrowserSessionCookie(cookies []*http.Cookie) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == "ccp_session" {
			return cookie
		}
	}
	return nil
}

func mustCreateBrowserSessionCookie(t *testing.T, application *app.Application, userID, authMethod, authProviderID, authProvider string, expiresAt time.Time, revokedAt *time.Time) *http.Cookie {
	t.Helper()
	rawToken, hash, err := application.Auth.TokenService().GenerateBrowserSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := application.Store.CreateBrowserSession(t.Context(), types.BrowserSession{
		BaseRecord: types.BaseRecord{
			ID:        "sess_" + strings.ReplaceAll(rawToken[:12], "_", ""),
			CreatedAt: now.Add(-time.Minute),
			UpdatedAt: now,
		},
		UserID:         userID,
		SessionHash:    hash,
		AuthMethod:     authMethod,
		AuthProviderID: authProviderID,
		AuthProvider:   authProvider,
		ExpiresAt:      expiresAt,
		RevokedAt:      revokedAt,
	}); err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: "ccp_session", Value: rawToken, Path: "/"}
}
