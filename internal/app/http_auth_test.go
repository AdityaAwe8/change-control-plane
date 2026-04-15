package app_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestProjectsRequireAuthentication(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
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

func TestCrossTenantProjectScopeDenied(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
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
	server := httptest.NewServer(app.NewHTTPServer(application).Handler())
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
