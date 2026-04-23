package app_test

import (
	"bytes"
	"context"
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

func TestServiceAccountTokenLifecycleAndAuth(t *testing.T) {
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

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Ledger",
		Slug:           "ledger",
		Criticality:    "high",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	serviceAccount := postItemAuth[types.ServiceAccount](t, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "deployment-agent",
		Role:           "org_member",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	issued := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name: "primary",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	services := getListAuth[types.Service](t, server.URL+"/api/v1/services", issued.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(services) != 1 {
		t.Fatalf("expected one service through machine actor, got %d", len(services))
	}

	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-b@acme.local",
		DisplayName:      "Owner B",
		OrganizationName: "Other",
		OrganizationSlug: "other",
	})
	getListAuth[types.Service](t, server.URL+"/api/v1/services", issued.Token, otherOrg.Session.ActiveOrganizationID, http.StatusForbidden)

	_ = postItemAuth[types.APIToken](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens/"+issued.Entry.ID+"/revoke", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	getListAuth[types.Service](t, server.URL+"/api/v1/services", issued.Token, admin.Session.ActiveOrganizationID, http.StatusUnauthorized)
}

func TestServiceAccountDeactivateAndRotateRoutes(t *testing.T) {
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

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Ledger",
		Slug:           "ledger",
		Criticality:    "high",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	serviceAccount := postItemAuth[types.ServiceAccount](t, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "deployment-agent",
		Role:           "org_member",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	issued := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name:           "primary",
		ExpiresInHours: 12,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	services := getListAuth[types.Service](t, server.URL+"/api/v1/services", issued.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(services) != 1 {
		t.Fatalf("expected one service through original machine token, got %d", len(services))
	}

	rotated := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens/"+issued.Entry.ID+"/rotate", types.RotateAPITokenRequest{
		Name:           "rotated",
		ExpiresInHours: 24,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if rotated.Token == "" {
		t.Fatal("expected rotated token to include a new raw token")
	}
	if rotated.Entry.ID == issued.Entry.ID {
		t.Fatalf("expected rotated token to create a new token entry, got same id %s", rotated.Entry.ID)
	}
	if rotated.Entry.Name != "rotated" {
		t.Fatalf("expected rotated token name to persist, got %s", rotated.Entry.Name)
	}
	if rotated.Entry.ExpiresAt == nil {
		t.Fatal("expected rotated token expiry to be set")
	}

	tokens := getListAuth[types.APIToken](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(tokens) != 2 {
		t.Fatalf("expected two tokens after rotation, got %d", len(tokens))
	}
	statusByID := make(map[string]string, len(tokens))
	for _, token := range tokens {
		statusByID[token.ID] = token.Status
	}
	if statusByID[issued.Entry.ID] != "revoked" {
		t.Fatalf("expected original token to be revoked after rotation, got %q", statusByID[issued.Entry.ID])
	}
	if statusByID[rotated.Entry.ID] != "active" {
		t.Fatalf("expected rotated token to be active, got %q", statusByID[rotated.Entry.ID])
	}

	getListAuth[types.Service](t, server.URL+"/api/v1/services", issued.Token, admin.Session.ActiveOrganizationID, http.StatusUnauthorized)
	services = getListAuth[types.Service](t, server.URL+"/api/v1/services", rotated.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(services) != 1 {
		t.Fatalf("expected one service through rotated machine token, got %d", len(services))
	}

	deactivated := postItemAuth[types.ServiceAccount](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/deactivate", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if deactivated.Status != "inactive" {
		t.Fatalf("expected deactivated service account status inactive, got %s", deactivated.Status)
	}

	getListAuth[types.Service](t, server.URL+"/api/v1/services", rotated.Token, admin.Session.ActiveOrganizationID, http.StatusUnauthorized)

	auditEvents := getListAuth[types.AuditEvent](t, server.URL+"/api/v1/audit-events", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawRotate, sawDeactivate bool
	for _, event := range auditEvents {
		switch event.Action {
		case "api_token.rotated":
			sawRotate = true
		case "service_account.deactivated":
			sawDeactivate = true
		}
	}
	if !sawRotate {
		t.Fatal("expected audit trail to include api_token.rotated")
	}
	if !sawDeactivate {
		t.Fatal("expected audit trail to include service_account.deactivated")
	}
}

func TestTeamCRUDRoutes(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-teams@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-teams",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	created := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if created.Status != "active" {
		t.Fatalf("expected created team to be active, got %s", created.Status)
	}

	teams := getListAuth[types.Team](t, server.URL+"/api/v1/teams", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(teams) != 1 {
		t.Fatalf("expected one team from list route, got %d", len(teams))
	}
	if teams[0].ID != created.ID {
		t.Fatalf("expected listed team id %s, got %s", created.ID, teams[0].ID)
	}

	fetched := getItemAuth[types.Team](t, server.URL+"/api/v1/teams/"+created.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if fetched.Name != "Core" || fetched.ProjectID != project.ID {
		t.Fatalf("unexpected fetched team: %+v", fetched)
	}

	updatedName := "Platform Core"
	updatedSlug := "platform-core"
	updatedOwners := []string{"user_2", "user_3"}
	updatedStatus := "inactive"
	updated := patchItemAuth[types.Team](t, server.URL+"/api/v1/teams/"+created.ID, types.UpdateTeamRequest{
		Name:         &updatedName,
		Slug:         &updatedSlug,
		OwnerUserIDs: &updatedOwners,
		Status:       &updatedStatus,
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if updated.Name != "Platform Core" || updated.Slug != "platform-core" {
		t.Fatalf("unexpected updated team naming: %+v", updated)
	}
	if updated.Status != "inactive" {
		t.Fatalf("expected updated team status inactive, got %s", updated.Status)
	}
	if len(updated.OwnerUserIDs) != 2 || updated.OwnerUserIDs[0] != "user_2" || updated.OwnerUserIDs[1] != "user_3" {
		t.Fatalf("unexpected updated team owners: %+v", updated.OwnerUserIDs)
	}

	archived := postItemAuth[types.Team](t, server.URL+"/api/v1/teams/"+created.ID+"/archive", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if archived.Status != "archived" {
		t.Fatalf("expected archived team status archived, got %s", archived.Status)
	}
}

func TestOrganizationProjectServiceAndEnvironmentCRUDRoutes(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-crud@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme CRUD",
		OrganizationSlug: "acme-crud",
	})
	other := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-crud-other@acme.local",
		DisplayName:      "Owner Other",
		OrganizationName: "Other CRUD",
		OrganizationSlug: "other-crud",
	})

	org := getItemAuth[types.Organization](t, server.URL+"/api/v1/organizations/"+admin.Session.ActiveOrganizationID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if org.ID != admin.Session.ActiveOrganizationID {
		t.Fatalf("expected organization %s, got %+v", admin.Session.ActiveOrganizationID, org)
	}
	if status := requestStatus(t, http.MethodGet, server.URL+"/api/v1/organizations/"+org.ID, nil, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org organization get to return 403, got %d", status)
	}

	updatedOrgName := "Acme CRUD Updated"
	updatedOrgTier := "enterprise"
	updatedOrgMode := "governed"
	updatedOrg := patchItemAuth[types.Organization](t, server.URL+"/api/v1/organizations/"+org.ID, types.UpdateOrganizationRequest{
		Name: &updatedOrgName,
		Tier: &updatedOrgTier,
		Mode: &updatedOrgMode,
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if updatedOrg.Name != updatedOrgName || updatedOrg.Tier != updatedOrgTier || updatedOrg.Mode != updatedOrgMode {
		t.Fatalf("unexpected updated organization %+v", updatedOrg)
	}
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/organizations/"+org.ID, types.UpdateOrganizationRequest{Name: &updatedOrgName}, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org organization update to return 403, got %d", status)
	}

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
		Description:    "Platform services",
		AdoptionMode:   "advisory",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	fetchedProject := getItemAuth[types.Project](t, server.URL+"/api/v1/projects/"+project.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if fetchedProject.ID != project.ID || fetchedProject.Description != "Platform services" {
		t.Fatalf("unexpected fetched project %+v", fetchedProject)
	}
	if status := requestStatus(t, http.MethodGet, server.URL+"/api/v1/projects/"+project.ID, nil, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org project get to return 403, got %d", status)
	}

	updatedProjectName := "Platform Updated"
	updatedProjectSlug := "platform-updated"
	updatedProjectDescription := "Updated platform services"
	updatedAdoptionMode := "governed"
	updatedProject := patchItemAuth[types.Project](t, server.URL+"/api/v1/projects/"+project.ID, types.UpdateProjectRequest{
		Name:         &updatedProjectName,
		Slug:         &updatedProjectSlug,
		Description:  &updatedProjectDescription,
		AdoptionMode: &updatedAdoptionMode,
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if updatedProject.Name != updatedProjectName || updatedProject.Slug != updatedProjectSlug || updatedProject.AdoptionMode != updatedAdoptionMode {
		t.Fatalf("unexpected updated project %+v", updatedProject)
	}
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/projects/"+project.ID, types.UpdateProjectRequest{Name: &updatedProjectName}, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org project update to return 403, got %d", status)
	}

	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Description:      "Checkout service",
		Criticality:      "high",
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	fetchedService := getItemAuth[types.Service](t, server.URL+"/api/v1/services/"+service.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if fetchedService.ID != service.ID || fetchedService.Name != service.Name {
		t.Fatalf("unexpected fetched service %+v", fetchedService)
	}
	if status := requestStatus(t, http.MethodGet, server.URL+"/api/v1/services/"+service.ID, nil, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org service get to return 403, got %d", status)
	}

	updatedServiceName := "Checkout API"
	updatedServiceSlug := "checkout-api"
	updatedCriticality := "mission_critical"
	updatedDescription := "Updated checkout service"
	updatedCustomerFacing := true
	updatedDependencyCount := 4
	updatedService := patchItemAuth[types.Service](t, server.URL+"/api/v1/services/"+service.ID, types.UpdateServiceRequest{
		Name:                   &updatedServiceName,
		Slug:                   &updatedServiceSlug,
		Description:            &updatedDescription,
		Criticality:            &updatedCriticality,
		CustomerFacing:         &updatedCustomerFacing,
		DependentServicesCount: &updatedDependencyCount,
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if updatedService.Name != updatedServiceName || updatedService.Slug != updatedServiceSlug || updatedService.Criticality != updatedCriticality {
		t.Fatalf("unexpected updated service %+v", updatedService)
	}
	if !updatedService.CustomerFacing || updatedService.DependentServicesCount != updatedDependencyCount {
		t.Fatalf("unexpected updated service posture %+v", updatedService)
	}
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/services/"+service.ID, types.UpdateServiceRequest{Name: &updatedServiceName}, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org service update to return 403, got %d", status)
	}

	archivedService := postItemAuth[types.Service](t, server.URL+"/api/v1/services/"+service.ID+"/archive", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if archivedService.Status != "archived" {
		t.Fatalf("expected archived service status archived, got %s", archivedService.Status)
	}

	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	fetchedEnvironment := getItemAuth[types.Environment](t, server.URL+"/api/v1/environments/"+environment.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if fetchedEnvironment.ID != environment.ID || fetchedEnvironment.Name != environment.Name {
		t.Fatalf("unexpected fetched environment %+v", fetchedEnvironment)
	}
	if status := requestStatus(t, http.MethodGet, server.URL+"/api/v1/environments/"+environment.ID, nil, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org environment get to return 403, got %d", status)
	}

	updatedEnvironmentName := "Production Updated"
	updatedEnvironmentSlug := "prod-updated"
	updatedEnvironmentType := "canary"
	updatedEnvironmentRegion := "us-east1"
	updatedProduction := false
	updatedComplianceZone := "pci"
	updatedEnvironment := patchItemAuth[types.Environment](t, server.URL+"/api/v1/environments/"+environment.ID, types.UpdateEnvironmentRequest{
		Name:           &updatedEnvironmentName,
		Slug:           &updatedEnvironmentSlug,
		Type:           &updatedEnvironmentType,
		Region:         &updatedEnvironmentRegion,
		Production:     &updatedProduction,
		ComplianceZone: &updatedComplianceZone,
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if updatedEnvironment.Name != updatedEnvironmentName || updatedEnvironment.Slug != updatedEnvironmentSlug || updatedEnvironment.Type != updatedEnvironmentType {
		t.Fatalf("unexpected updated environment %+v", updatedEnvironment)
	}
	if updatedEnvironment.Region != updatedEnvironmentRegion || updatedEnvironment.Production != updatedProduction || updatedEnvironment.ComplianceZone != updatedComplianceZone {
		t.Fatalf("unexpected updated environment posture %+v", updatedEnvironment)
	}
	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/environments/"+environment.ID, types.UpdateEnvironmentRequest{Name: &updatedEnvironmentName}, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org environment update to return 403, got %d", status)
	}

	archivedEnvironment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments/"+environment.ID+"/archive", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if archivedEnvironment.Status != "archived" {
		t.Fatalf("expected archived environment status archived, got %s", archivedEnvironment.Status)
	}

	archivedProject := postItemAuth[types.Project](t, server.URL+"/api/v1/projects/"+project.ID+"/archive", struct{}{}, admin.Token, admin.Session.ActiveOrganizationID)
	if archivedProject.Status != "archived" {
		t.Fatalf("expected archived project status archived, got %s", archivedProject.Status)
	}
}

func TestIncidentDetailRoute(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-incidents@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-incidents",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
		Criticality:    "high",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Staging",
		Slug:           "staging",
		Type:           "staging",
		Region:         "us-central1",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "pause rollout for incident detail",
		ChangeTypes:    []string{"code"},
		FileCount:      2,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rollout := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rollout.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	approved := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approve for incident setup",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if approved.Status != "approved" {
		t.Fatalf("expected approved execution status approved, got %s", approved.Status)
	}

	started := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start for incident setup",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if started.Status != "in_progress" {
		t.Fatalf("expected started execution to be in_progress, got %s", started.Status)
	}

	paused := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/pause", struct {
		Reason string `json:"reason"`
	}{
		Reason: "pause for incident detail route",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if paused.Status != "paused" {
		t.Fatalf("expected paused execution status paused, got %s", paused.Status)
	}

	detail := getItemAuth[types.IncidentDetail](t, server.URL+"/api/v1/incidents/incident_"+execution.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if detail.Incident.ID != "incident_"+execution.ID {
		t.Fatalf("expected incident id incident_%s, got %s", execution.ID, detail.Incident.ID)
	}
	if detail.RolloutExecutionID != execution.ID {
		t.Fatalf("expected rollout execution id %s, got %s", execution.ID, detail.RolloutExecutionID)
	}
	if detail.Incident.Status != "monitoring" || detail.Incident.Severity != "high" {
		t.Fatalf("unexpected derived incident state %+v", detail.Incident)
	}
	if detail.Incident.RelatedChange != change.ID {
		t.Fatalf("expected related change %s, got %s", change.ID, detail.Incident.RelatedChange)
	}
	if len(detail.StatusTimeline) == 0 {
		t.Fatal("expected incident detail to include correlated status timeline")
	}

	other := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-other-incidents@acme.local",
		DisplayName:      "Owner Other",
		OrganizationName: "Other",
		OrganizationSlug: "other-incidents",
	})
	req, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/incidents/incident_"+execution.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+other.Token)
	req.Header.Set("X-CCP-Organization-ID", other.Session.ActiveOrganizationID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected cross-org incident detail request to be forbidden, got %d", resp.StatusCode)
	}

	plannedExecution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rollout.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	req, err = http.NewRequest(http.MethodGet, server.URL+"/api/v1/incidents/incident_"+plannedExecution.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+admin.Token)
	req.Header.Set("X-CCP-Organization-ID", admin.Session.ActiveOrganizationID)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected non-incident rollout detail request to return 404, got %d", resp.StatusCode)
	}
}

func TestRolloutEvidencePackRoute(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	store := app.NewInMemoryStore()
	application := app.NewApplicationWithStore(common.LoadConfig(), store)
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-evidence@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Evidence",
		OrganizationSlug: "acme-evidence",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "high",
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	_ = postItemAuth[types.Policy](t, server.URL+"/api/v1/policies", types.CreatePolicyRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Name:           "Production Manual Review",
		Code:           "production-manual-review",
		AppliesTo:      "rollout_plan",
		Mode:           "require_manual_review",
		Priority:       100,
		Description:    "Require review for production releases.",
		Conditions: types.PolicyCondition{
			ProductionOnly: true,
		},
	}, admin.Token, admin.Session.ActiveOrganizationID)

	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "release checkout with mapped evidence",
		ChangeTypes:    []string{"code", "config"},
		FileCount:      5,
		ResourceCount:  2,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rollout := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	now := time.Now().UTC()
	backendIntegration := types.Integration{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("int"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Production Kubernetes",
		Kind:           "kubernetes",
		InstanceKey:    "kube-prod",
		ScopeType:      "cluster",
		ScopeName:      "prod",
		Mode:           "advisory",
		Enabled:        true,
		Status:         "connected",
	}
	if err := store.CreateIntegration(context.Background(), backendIntegration); err != nil {
		t.Fatal(err)
	}
	signalIntegration := types.Integration{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("int"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Production Prometheus",
		Kind:           "prometheus",
		InstanceKey:    "prom-prod",
		ScopeType:      "environment",
		ScopeName:      "prod",
		Mode:           "advisory",
		Enabled:        true,
		Status:         "connected",
	}
	if err := store.CreateIntegration(context.Background(), signalIntegration); err != nil {
		t.Fatal(err)
	}

	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID:        rollout.Plan.ID,
		BackendType:          "simulated",
		BackendIntegrationID: backendIntegration.ID,
		SignalProviderType:   "prometheus",
		SignalIntegrationID:  signalIntegration.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	approved := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approved for evidence pack route",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if approved.Status != "approved" {
		t.Fatalf("expected approved execution, got %+v", approved)
	}
	started := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start for evidence pack route",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if started.Status != "in_progress" {
		t.Fatalf("expected in-progress execution, got %+v", started)
	}

	_ = postItemAuth[types.SignalSnapshot](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/signal-snapshots", types.CreateSignalSnapshotRequest{
		ProviderType: "prometheus",
		Health:       "healthy",
		Summary:      "latency steady",
		Signals: []types.SignalValue{{
			Name:       "latency_p95_ms",
			Category:   "technical",
			Value:      180,
			Unit:       "ms",
			Status:     "healthy",
			Threshold:  250,
			Comparator: "<=",
		}},
		WindowSeconds: 300,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	_ = postItemAuth[types.VerificationResult](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/verification", types.RecordVerificationResultRequest{
		Outcome:        "passed",
		Decision:       "continue",
		Summary:        "verification remains healthy",
		DecisionSource: "operator",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	paused := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/pause", struct {
		Reason string `json:"reason"`
	}{
		Reason: "pause to preserve incident evidence",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if paused.Status != "paused" {
		t.Fatalf("expected paused execution, got %+v", paused)
	}

	repository := types.Repository{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("repo"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Name:           "checkout-service",
		Provider:       "github",
		URL:            "https://github.com/acme/checkout-service",
		DefaultBranch:  "main",
		Status:         "mapped",
	}
	if err := store.UpsertRepository(context.Background(), repository); err != nil {
		t.Fatal(err)
	}
	resource := types.DiscoveredResource{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("discovery"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID: admin.Session.ActiveOrganizationID,
		IntegrationID:  backendIntegration.ID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		RepositoryID:   repository.ID,
		ResourceType:   "kubernetes_workload",
		Provider:       "kubernetes",
		ExternalID:     "checkout",
		Namespace:      "prod",
		Name:           "checkout",
		Status:         "mapped",
		Health:         "healthy",
		Summary:        "checkout workload healthy",
	}
	if err := store.UpsertDiscoveredResource(context.Background(), resource); err != nil {
		t.Fatal(err)
	}
	relationship := types.GraphRelationship{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("rel"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		RelationshipType: "service_repository",
		FromResourceType: "service",
		FromResourceID:   service.ID,
		ToResourceType:   "repository",
		ToResourceID:     repository.ID,
		Status:           "active",
		LastObservedAt:   now,
	}
	if err := store.UpsertGraphRelationship(context.Background(), relationship); err != nil {
		t.Fatal(err)
	}

	pack := getItemAuth[types.RolloutEvidencePack](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/evidence-pack", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if pack.ExecutionDetail.Execution.ID != execution.ID {
		t.Fatalf("expected rollout execution %s in evidence pack, got %+v", execution.ID, pack.ExecutionDetail.Execution)
	}
	if pack.Organization.ID != admin.Session.ActiveOrganizationID || pack.Project.ID != project.ID {
		t.Fatalf("expected organization/project context in evidence pack, got %+v / %+v", pack.Organization, pack.Project)
	}
	if pack.Summary.ManualReviewPolicyCount < 1 {
		t.Fatalf("expected at least one manual-review policy decision, got %+v", pack.Summary)
	}
	if pack.Summary.LatestVerificationOutcome != "passed" || pack.Summary.ApprovalState != "satisfied" {
		t.Fatalf("unexpected evidence pack summary %+v", pack.Summary)
	}
	if len(pack.PolicyDecisions) == 0 || len(pack.Repositories) != 1 || len(pack.DiscoveredResources) != 1 || len(pack.GraphRelationships) != 1 {
		t.Fatalf("expected mapped policy/repository/resource/graph evidence, got %+v", pack)
	}
	if len(pack.AuditTrail) == 0 {
		t.Fatal("expected audit trail in rollout evidence pack")
	}
	if len(pack.Incidents) == 0 || pack.Incidents[0].RelatedChange != change.ID {
		t.Fatalf("expected related incident evidence, got %+v", pack.Incidents)
	}
	if pack.ExecutionDetail.RuntimeSummary.ControlMode != "advisory" {
		t.Fatalf("expected advisory control mode in evidence pack runtime summary, got %+v", pack.ExecutionDetail.RuntimeSummary)
	}
}

func TestIncidentListRouteSupportsFilters(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-incident-list@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Incident List",
		OrganizationSlug: "acme-incident-list",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	checkoutService := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout API",
		Slug:           "checkout-api",
		Criticality:    "high",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	ledgerService := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Ledger Worker",
		Slug:           "ledger-worker",
		Criticality:    "high",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	staging := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Staging",
		Slug:           "staging",
		Type:           "staging",
		Region:         "us-central1",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	production := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "production",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	pausedChange := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      checkoutService.ID,
		EnvironmentID:  staging.ID,
		Summary:        "checkout pause candidate",
		ChangeTypes:    []string{"code"},
		FileCount:      2,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	pausedPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: pausedChange.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	pausedExecution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: pausedPlan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	pausedExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+pausedExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approve paused execution",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	pausedExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+pausedExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start paused execution",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	pausedExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+pausedExecution.ID+"/pause", struct {
		Reason string `json:"reason"`
	}{
		Reason: "pause checkout rollout",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if pausedExecution.Status != "paused" {
		t.Fatalf("expected paused execution, got %s", pausedExecution.Status)
	}

	rolledBackChange := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      ledgerService.ID,
		EnvironmentID:  production.ID,
		Summary:        "ledger rollback candidate",
		ChangeTypes:    []string{"code"},
		FileCount:      3,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolledBackPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: rolledBackChange.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolledBackExecution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rolledBackPlan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolledBackExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+rolledBackExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approve rollback execution",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolledBackExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+rolledBackExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start rollback execution",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolledBackExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+rolledBackExecution.ID+"/rollback", struct {
		Reason string `json:"reason"`
	}{
		Reason: "rollback ledger rollout",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if rolledBackExecution.Status != "rolled_back" {
		t.Fatalf("expected rolled back execution, got %s", rolledBackExecution.Status)
	}

	completedChange := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      checkoutService.ID,
		EnvironmentID:  production.ID,
		Summary:        "checkout completed candidate",
		ChangeTypes:    []string{"code"},
		FileCount:      1,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	completedPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: completedChange.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	completedExecution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: completedPlan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	completedExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+completedExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approve completed execution",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	completedExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+completedExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start completed execution",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	completedExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+completedExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "complete",
		Reason: "complete non-incident execution",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if completedExecution.Status != "completed" {
		t.Fatalf("expected completed execution, got %s", completedExecution.Status)
	}

	all := getListAuth[types.Incident](t, server.URL+"/api/v1/incidents", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(all) != 2 {
		t.Fatalf("expected two derived incidents and one non-incident execution to be excluded, got %d", len(all))
	}

	pausedIncidents := getListAuth[types.Incident](t, server.URL+"/api/v1/incidents?status=monitoring&service_id="+checkoutService.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(pausedIncidents) != 1 || pausedIncidents[0].ID != "incident_"+pausedExecution.ID {
		t.Fatalf("expected paused incident filter to return incident_%s, got %+v", pausedExecution.ID, pausedIncidents)
	}

	criticalIncidents := getListAuth[types.Incident](t, server.URL+"/api/v1/incidents?severity=critical&environment_id="+production.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(criticalIncidents) != 1 || criticalIncidents[0].ID != "incident_"+rolledBackExecution.ID {
		t.Fatalf("expected critical incident filter to return incident_%s, got %+v", rolledBackExecution.ID, criticalIncidents)
	}

	changeScoped := getListAuth[types.Incident](t, server.URL+"/api/v1/incidents?change_set_id="+pausedChange.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(changeScoped) != 1 || changeScoped[0].RelatedChange != pausedChange.ID {
		t.Fatalf("expected change-scoped incidents to return %s, got %+v", pausedChange.ID, changeScoped)
	}

	searchScoped := getListAuth[types.Incident](t, server.URL+"/api/v1/incidents?search=checkout&limit=1", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(searchScoped) != 1 || searchScoped[0].ServiceID != checkoutService.ID {
		t.Fatalf("expected search and limit filters to narrow incident list, got %+v", searchScoped)
	}

	other := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-other-incident-list@acme.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other Incident List",
		OrganizationSlug: "other-incident-list",
	})
	getListAuth[types.Incident](t, server.URL+"/api/v1/incidents", admin.Token, other.Session.ActiveOrganizationID, http.StatusForbidden)
}

func TestChangeRiskRolloutAuditAndRollbackPolicyRoutes(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-older-routes@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Older Routes",
		OrganizationSlug: "acme-older-routes",
	})
	other := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-older-routes-other@acme.local",
		DisplayName:      "Owner Other",
		OrganizationName: "Other Older Routes",
		OrganizationSlug: "other-older-routes",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "mission_critical",
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID:        admin.Session.ActiveOrganizationID,
		ProjectID:             project.ID,
		ServiceID:             service.ID,
		EnvironmentID:         environment.ID,
		Summary:               "older CRUD route proof change",
		ChangeTypes:           []string{"code", "config"},
		FileCount:             6,
		ResourceCount:         2,
		TouchesInfrastructure: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	changes := getListAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawChange bool
	for _, candidate := range changes {
		if candidate.ID == change.ID {
			sawChange = true
			break
		}
	}
	if !sawChange {
		t.Fatalf("expected change list to include %s, got %+v", change.ID, changes)
	}

	detail := getItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes/"+change.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if detail.ID != change.ID || detail.Summary != change.Summary {
		t.Fatalf("unexpected change detail %+v", detail)
	}

	otherChanges := getListAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", other.Token, other.Session.ActiveOrganizationID, http.StatusOK)
	for _, candidate := range otherChanges {
		if candidate.ID == change.ID {
			t.Fatalf("expected other tenant change list to exclude %s, got %+v", change.ID, otherChanges)
		}
	}

	if status := requestStatus(t, http.MethodGet, server.URL+"/api/v1/changes/"+change.ID, nil, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org change detail request to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodGet, server.URL+"/api/v1/changes/chg_missing", nil, admin.Token, admin.Session.ActiveOrganizationID); status != http.StatusNotFound {
		t.Fatalf("expected missing change detail request to return 404, got %d", status)
	}

	assessment := postItemAuth[types.RiskAssessmentResult](t, server.URL+"/api/v1/risk-assessments", types.CreateRiskAssessmentRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if assessment.Assessment.ChangeSetID != change.ID {
		t.Fatalf("expected risk assessment to target %s, got %+v", change.ID, assessment.Assessment)
	}

	assessments := getListAuth[types.RiskAssessment](t, server.URL+"/api/v1/risk-assessments", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawAssessment bool
	for _, candidate := range assessments {
		if candidate.ID == assessment.Assessment.ID {
			sawAssessment = true
			break
		}
	}
	if !sawAssessment {
		t.Fatalf("expected risk list to include %s, got %+v", assessment.Assessment.ID, assessments)
	}

	otherAssessments := getListAuth[types.RiskAssessment](t, server.URL+"/api/v1/risk-assessments", other.Token, other.Session.ActiveOrganizationID, http.StatusOK)
	for _, candidate := range otherAssessments {
		if candidate.ID == assessment.Assessment.ID {
			t.Fatalf("expected other tenant risk list to exclude %s, got %+v", assessment.Assessment.ID, otherAssessments)
		}
	}

	rollout := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if rollout.Plan.ChangeSetID != change.ID {
		t.Fatalf("expected rollout plan to target %s, got %+v", change.ID, rollout.Plan)
	}

	plans := getListAuth[types.RolloutPlan](t, server.URL+"/api/v1/rollout-plans", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawPlan bool
	for _, candidate := range plans {
		if candidate.ID == rollout.Plan.ID {
			sawPlan = true
			break
		}
	}
	if !sawPlan {
		t.Fatalf("expected rollout plan list to include %s, got %+v", rollout.Plan.ID, plans)
	}

	otherPlans := getListAuth[types.RolloutPlan](t, server.URL+"/api/v1/rollout-plans", other.Token, other.Session.ActiveOrganizationID, http.StatusOK)
	for _, candidate := range otherPlans {
		if candidate.ID == rollout.Plan.ID {
			t.Fatalf("expected other tenant rollout list to exclude %s, got %+v", rollout.Plan.ID, otherPlans)
		}
	}

	auditEvents := getListAuth[types.AuditEvent](t, server.URL+"/api/v1/audit-events", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawChangeIngest, sawRiskAssessed, sawRolloutPlanned bool
	for _, event := range auditEvents {
		switch event.Action {
		case "change.ingested":
			if event.ResourceID == change.ID {
				sawChangeIngest = true
			}
		case "risk.assessed":
			if event.ResourceID == assessment.Assessment.ID {
				sawRiskAssessed = true
			}
		case "rollout.planned":
			if event.ResourceID == rollout.Plan.ID {
				sawRolloutPlanned = true
			}
		}
	}
	if !sawChangeIngest || !sawRiskAssessed || !sawRolloutPlanned {
		t.Fatalf("expected audit trail to include change/risk/rollout evidence, got %+v", auditEvents)
	}

	otherAuditEvents := getListAuth[types.AuditEvent](t, server.URL+"/api/v1/audit-events", other.Token, other.Session.ActiveOrganizationID, http.StatusOK)
	for _, event := range otherAuditEvents {
		if event.ResourceID == change.ID || event.ResourceID == assessment.Assessment.ID || event.ResourceID == rollout.Plan.ID {
			t.Fatalf("expected other tenant audit trail to exclude admin resources, got %+v", otherAuditEvents)
		}
	}

	policy := postItemAuth[types.RollbackPolicy](t, server.URL+"/api/v1/rollback-policies", types.CreateRollbackPolicyRequest{
		OrganizationID:            admin.Session.ActiveOrganizationID,
		ProjectID:                 project.ID,
		ServiceID:                 service.ID,
		EnvironmentID:             environment.ID,
		Name:                      "Prod strict",
		Description:               "Rollback aggressively for production signal regressions.",
		Priority:                  80,
		MaxErrorRate:              1.1,
		MaxLatencyMs:              300,
		MaxVerificationFailures:   1,
		RollbackOnProviderFailure: ptrBool(true),
		RollbackOnCriticalSignals: ptrBool(true),
	}, admin.Token, admin.Session.ActiveOrganizationID)

	policies := getListAuth[types.RollbackPolicy](t, server.URL+"/api/v1/rollback-policies", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawPolicy bool
	for _, candidate := range policies {
		if candidate.ID == policy.ID {
			sawPolicy = true
			break
		}
	}
	if !sawPolicy {
		t.Fatalf("expected rollback policy list to include %s, got %+v", policy.ID, policies)
	}

	otherPolicies := getListAuth[types.RollbackPolicy](t, server.URL+"/api/v1/rollback-policies", other.Token, other.Session.ActiveOrganizationID, http.StatusOK)
	for _, candidate := range otherPolicies {
		if candidate.ID == policy.ID {
			t.Fatalf("expected other tenant rollback policy list to exclude %s, got %+v", policy.ID, otherPolicies)
		}
	}

	updatedName := "Prod strict updated"
	updatedDescription := "Rollback aggressively for verified latency or error regressions."
	updatedEnabled := false
	updatedPriority := 95
	updatedLatency := 275.0
	updatedMetadata := types.Metadata{"operator": "http-route-proof"}
	updated := patchItemAuth[types.RollbackPolicy](t, server.URL+"/api/v1/rollback-policies/"+policy.ID, types.UpdateRollbackPolicyRequest{
		Name:         &updatedName,
		Description:  &updatedDescription,
		Enabled:      &updatedEnabled,
		Priority:     &updatedPriority,
		MaxLatencyMs: &updatedLatency,
		Metadata:     updatedMetadata,
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if updated.Name != updatedName || updated.Description != updatedDescription {
		t.Fatalf("unexpected updated rollback policy naming %+v", updated)
	}
	if updated.Enabled != updatedEnabled || updated.Priority != updatedPriority {
		t.Fatalf("unexpected updated rollback policy state %+v", updated)
	}
	if updated.MaxLatencyMs != updatedLatency {
		t.Fatalf("expected updated rollback latency %v, got %v", updatedLatency, updated.MaxLatencyMs)
	}
	if updated.Metadata["operator"] != "http-route-proof" {
		t.Fatalf("expected updated rollback policy metadata to persist, got %+v", updated.Metadata)
	}

	policies = getListAuth[types.RollbackPolicy](t, server.URL+"/api/v1/rollback-policies", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawUpdatedPolicy bool
	for _, candidate := range policies {
		if candidate.ID == policy.ID {
			sawUpdatedPolicy = candidate.Name == updatedName && !candidate.Enabled && candidate.Priority == updatedPriority && candidate.MaxLatencyMs == updatedLatency
			break
		}
	}
	if !sawUpdatedPolicy {
		t.Fatalf("expected rollback policy list to reflect updated policy %+v, got %+v", updated, policies)
	}

	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/rollback-policies/rpol_missing", types.UpdateRollbackPolicyRequest{Name: &updatedName}, admin.Token, admin.Session.ActiveOrganizationID); status != http.StatusNotFound {
		t.Fatalf("expected missing rollback policy update to return 404, got %d", status)
	}

	auditEvents = getListAuth[types.AuditEvent](t, server.URL+"/api/v1/audit-events", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawPolicyCreated, sawPolicyUpdated bool
	for _, event := range auditEvents {
		switch event.Action {
		case "rollback_policy.created":
			if event.ResourceID == policy.ID {
				sawPolicyCreated = true
			}
		case "rollback_policy.updated":
			if event.ResourceID == policy.ID {
				sawPolicyUpdated = true
			}
		}
	}
	if !sawPolicyCreated || !sawPolicyUpdated {
		t.Fatalf("expected audit trail to include rollback policy create/update evidence, got %+v", auditEvents)
	}
}

func TestGraphIngestionIsIdempotent(t *testing.T) {
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

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "Ship release",
		ChangeTypes:    []string{"code"},
	}, admin.Token, admin.Session.ActiveOrganizationID)

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

	payload := types.IntegrationGraphIngestRequest{
		Repositories: []types.IntegrationRepositoryInput{
			{
				ServiceID:     service.ID,
				Name:          "checkout",
				Provider:      "github",
				URL:           "https://github.com/acme/checkout",
				DefaultBranch: "main",
			},
		},
		ChangeRepositories: []types.ChangeRepositoryBindingInput{
			{
				ChangeSetID:   change.ID,
				RepositoryURL: "https://github.com/acme/checkout",
			},
		},
		ServiceEnvironments: []types.ServiceEnvironmentBindingInput{
			{ServiceID: service.ID, EnvironmentID: environment.ID},
		},
	}

	first := postListAuth[types.GraphRelationship](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/graph-ingest", payload, admin.Token, admin.Session.ActiveOrganizationID)
	second := postListAuth[types.GraphRelationship](t, server.URL+"/api/v1/integrations/"+githubIntegration.ID+"/graph-ingest", payload, admin.Token, admin.Session.ActiveOrganizationID)
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("expected relationships to be ingested")
	}

	relationships := getListAuth[types.GraphRelationship](t, server.URL+"/api/v1/graph/relationships", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(relationships) != 5 {
		t.Fatalf("expected five unique relationships after repeated ingest, got %d", len(relationships))
	}

	repositories := getListAuth[types.Repository](t, server.URL+"/api/v1/repositories", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(repositories) != 1 {
		t.Fatalf("expected one repository after ingest, got %d", len(repositories))
	}
	if repositories[0].SourceIntegrationID != githubIntegration.ID {
		t.Fatalf("expected repository source integration %s, got %s", githubIntegration.ID, repositories[0].SourceIntegrationID)
	}
	if repositories[0].ServiceID != service.ID {
		t.Fatalf("expected graph ingest to persist repository service mapping, got %+v", repositories[0])
	}
	inferredOwner, _ := repositories[0].Metadata["inferred_owner"].(map[string]any)
	if inferredOwner["team_id"] != team.ID {
		t.Fatalf("expected graph ingest to infer repository owner team, got %+v", repositories[0].Metadata)
	}

	filteredRelationships := getListAuth[types.GraphRelationship](t, server.URL+"/api/v1/graph/relationships?relationship_type=team_repository_owner&source_integration_id="+githubIntegration.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(filteredRelationships) != 1 {
		t.Fatalf("expected one filtered ownership relationship, got %+v", filteredRelationships)
	}
	if filteredRelationships[0].Metadata["provenance_source"] != "inferred_owner" {
		t.Fatalf("expected filtered relationship provenance metadata, got %+v", filteredRelationships[0])
	}
}

func TestRolloutExecutionLifecycleAndVerification(t *testing.T) {
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

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "mission_critical",
		CustomerFacing:   true,
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID:        admin.Session.ActiveOrganizationID,
		ProjectID:             project.ID,
		ServiceID:             service.ID,
		EnvironmentID:         environment.ID,
		Summary:               "Update payment routing",
		ChangeTypes:           []string{"code", "iam"},
		FileCount:             8,
		TouchesInfrastructure: true,
		TouchesIAM:            true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolloutPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rolloutPlan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "awaiting_approval" {
		t.Fatalf("expected awaiting_approval, got %s", execution.Status)
	}

	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approval granted",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "begin rollout",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "in_progress" {
		t.Fatalf("expected in_progress, got %s", execution.Status)
	}

	_ = postItemAuth[types.VerificationResult](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/verification", types.RecordVerificationResultRequest{
		Outcome:  "fail",
		Decision: "pause",
		Summary:  "latency regression detected",
		Signals:  []string{"latency", "error-rate"},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "continue",
		Reason: "manual approval after mitigation",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "in_progress" {
		t.Fatalf("expected in_progress after continue, got %s", execution.Status)
	}

	_ = postItemAuth[types.VerificationResult](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/verification", types.RecordVerificationResultRequest{
		Outcome:  "pass",
		Decision: "continue",
		Summary:  "signals recovered",
		Signals:  []string{"latency"},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "complete",
		Reason: "promotion finished",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "completed" {
		t.Fatalf("expected completed, got %s", execution.Status)
	}

	detail := getItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+execution.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(detail.VerificationResults) != 2 {
		t.Fatalf("expected two verification results, got %d", len(detail.VerificationResults))
	}
}

func TestRolloutPauseResumeRollbackRoutes(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-rollout-controls@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Rollout Controls",
		OrganizationSlug: "acme-rollout-controls",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
		Criticality:    "mission_critical",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "rollout control route proof",
		ChangeTypes:    []string{"code"},
		FileCount:      4,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolloutPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rolloutPlan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approval granted for manual control proof",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start rollout for manual control proof",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "in_progress" {
		t.Fatalf("expected in_progress before manual controls, got %s", execution.Status)
	}

	paused := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/pause", struct {
		Reason string `json:"reason"`
	}{
		Reason: "pause via dedicated route",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if paused.Status != "paused" || paused.LastDecision != "pause" {
		t.Fatalf("expected paused execution after pause route, got %+v", paused)
	}

	resumed := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/resume", struct {
		Reason string `json:"reason"`
	}{
		Reason: "resume via dedicated route",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if resumed.Status != "in_progress" || resumed.LastDecision != "resume" {
		t.Fatalf("expected resumed execution after resume route, got %+v", resumed)
	}

	rolledBack := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/rollback", struct {
		Reason string `json:"reason"`
	}{
		Reason: "rollback via dedicated route",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if rolledBack.Status != "rolled_back" || rolledBack.LastDecision != "rollback" {
		t.Fatalf("expected rolled_back execution after rollback route, got %+v", rolledBack)
	}

	timeline := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/timeline", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawPause, sawResume, sawRollback bool
	for _, event := range timeline {
		summary := strings.ToLower(event.Summary + " " + strings.Join(event.Explanation, " "))
		if strings.Contains(summary, "via pause") {
			sawPause = true
		}
		if strings.Contains(summary, "via resume") {
			sawResume = true
		}
		if strings.Contains(summary, "via rollback") {
			sawRollback = true
		}
	}
	if !sawPause || !sawResume || !sawRollback {
		t.Fatalf("expected rollout timeline to include pause/resume/rollback evidence, got %+v", timeline)
	}
}

func TestRecordVerificationResultRouteActiveAndAdvisory(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-verification@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Verification",
		OrganizationSlug: "acme-verification",
	})
	other := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-verification-other@acme.local",
		DisplayName:      "Owner Other",
		OrganizationName: "Other Verification",
		OrganizationSlug: "other-verification",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "mission_critical",
		CustomerFacing:   true,
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "verification route proof",
		ChangeTypes:    []string{"code"},
		FileCount:      4,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolloutPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	activeExecution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rolloutPlan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	activeExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+activeExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approve for verification route proof",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	activeExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+activeExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start for verification route proof",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if activeExecution.Status != "in_progress" {
		t.Fatalf("expected active execution to be in_progress, got %s", activeExecution.Status)
	}

	activeResult := postItemAuth[types.VerificationResult](t, server.URL+"/api/v1/rollout-executions/"+activeExecution.ID+"/verification", types.RecordVerificationResultRequest{
		Outcome:  "fail",
		Decision: "pause",
		Summary:  "manual verification caught a latency regression",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if activeResult.Decision != "pause" || activeResult.DecisionSource != "manual" {
		t.Fatalf("unexpected active verification result %+v", activeResult)
	}

	activeDetail := getItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+activeExecution.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if activeDetail.Execution.Status != "paused" {
		t.Fatalf("expected active execution status paused after verification, got %+v", activeDetail.Execution)
	}
	if len(activeDetail.VerificationResults) != 1 || activeDetail.VerificationResults[0].ID != activeResult.ID {
		t.Fatalf("expected one persisted active verification result, got %+v", activeDetail.VerificationResults)
	}

	activeTimeline := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/rollout-executions/"+activeExecution.ID+"/timeline", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var sawVerificationRecorded bool
	for _, event := range activeTimeline {
		if event.EventType == "verification.recorded" && strings.Contains(strings.ToLower(event.Summary), "latency regression") {
			sawVerificationRecorded = true
			break
		}
	}
	if !sawVerificationRecorded {
		t.Fatalf("expected active rollout timeline to include verification.recorded, got %+v", activeTimeline)
	}

	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+activeExecution.ID+"/verification", types.RecordVerificationResultRequest{
		Outcome:  "pass",
		Decision: "verified",
		Summary:  "cross-tenant verification attempt",
	}, other.Token, other.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org verification record to return 403, got %d", status)
	}

	integrations := getListAuth[types.Integration](t, server.URL+"/api/v1/integrations", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	var kubernetes types.Integration
	for _, integration := range integrations {
		if integration.Kind == "kubernetes" {
			kubernetes = integration
			break
		}
	}
	if kubernetes.ID == "" {
		t.Fatal("expected seeded kubernetes integration for advisory verification proof")
	}

	controlDisabled := false
	advisoryMode := "advisory"
	enabled := true
	kubernetes = patchItemAuth[types.Integration](t, server.URL+"/api/v1/integrations/"+kubernetes.ID, types.UpdateIntegrationRequest{
		Enabled:        &enabled,
		Mode:           &advisoryMode,
		ControlEnabled: &controlDisabled,
		Metadata: types.Metadata{
			"api_base_url":     "https://cluster.example.com",
			"namespace":        "prod",
			"deployment_name":  "checkout",
			"kubeconfig_env":   "CCP_KUBECONFIG",
			"rollout_resource": "deployment/checkout",
		},
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)

	advisoryExecution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID:        rolloutPlan.Plan.ID,
		BackendType:          "kubernetes",
		BackendIntegrationID: kubernetes.ID,
		SignalProviderType:   "simulated",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	advisoryExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+advisoryExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approve advisory verification route proof",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	advisoryExecution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+advisoryExecution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start advisory verification route proof",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if advisoryExecution.Status != "in_progress" {
		t.Fatalf("expected advisory execution to remain in_progress before verification, got %s", advisoryExecution.Status)
	}

	advisoryResult := postItemAuth[types.VerificationResult](t, server.URL+"/api/v1/rollout-executions/"+advisoryExecution.ID+"/verification", types.RecordVerificationResultRequest{
		Outcome:  "fail",
		Decision: "rollback",
		Summary:  "manual advisory verification recommends rollback",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if advisoryResult.Decision != "advisory_rollback" {
		t.Fatalf("expected advisory verification decision rewrite, got %+v", advisoryResult)
	}
	if len(advisoryResult.Explanation) == 0 {
		t.Fatalf("expected advisory verification to include explanation, got %+v", advisoryResult)
	}

	advisoryDetail := getItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+advisoryExecution.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if advisoryDetail.Execution.Status != "in_progress" {
		t.Fatalf("expected advisory execution status to remain in_progress, got %+v", advisoryDetail.Execution)
	}
	if len(advisoryDetail.VerificationResults) != 1 || advisoryDetail.VerificationResults[0].Decision != "advisory_rollback" {
		t.Fatalf("expected persisted advisory verification result, got %+v", advisoryDetail.VerificationResults)
	}
}

func TestRolloutExecutionReconcileAndSignalSnapshots(t *testing.T) {
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

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "medium",
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Staging",
		Slug:           "staging",
		Type:           "staging",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "Ship change",
		ChangeTypes:    []string{"code"},
		FileCount:      3,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolloutPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID:      rolloutPlan.Plan.ID,
		BackendType:        "simulated",
		SignalProviderType: "simulated",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	serviceAccount := postItemAuth[types.ServiceAccount](t, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "worker-bot",
		Role:           "org_member",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	issued := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name: "worker",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "begin rollout",
	}, issued.Token, admin.Session.ActiveOrganizationID)
	if execution.Status != "in_progress" {
		t.Fatalf("expected in_progress after start, got %s", execution.Status)
	}

	_ = postItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/reconcile", struct{}{}, issued.Token, admin.Session.ActiveOrganizationID)

	snapshot := postItemAuth[types.SignalSnapshot](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/signal-snapshots", types.CreateSignalSnapshotRequest{
		ProviderType: "simulated",
		Health:       "healthy",
		Summary:      "latency and error rate remain healthy",
		Signals: []types.SignalValue{
			{Name: "latency_p95_ms", Category: "technical", Value: 120, Unit: "ms", Status: "healthy", Threshold: 250, Comparator: ">"},
		},
	}, issued.Token, admin.Session.ActiveOrganizationID)
	if snapshot.RolloutExecutionID != execution.ID {
		t.Fatalf("expected signal snapshot to belong to rollout execution, got %+v", snapshot)
	}

	detail := postItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/reconcile", struct{}{}, issued.Token, admin.Session.ActiveOrganizationID)
	if detail.Execution.BackendExecutionID == "" {
		t.Fatal("expected backend execution id after reconcile")
	}
	if len(detail.SignalSnapshots) != 1 {
		t.Fatalf("expected one signal snapshot, got %d", len(detail.SignalSnapshots))
	}
	if len(detail.VerificationResults) == 0 {
		t.Fatal("expected automated verification result after reconcile")
	}
	latestVerification := detail.VerificationResults[len(detail.VerificationResults)-1]
	if !latestVerification.Automated || latestVerification.Decision != "verified" {
		t.Fatalf("expected automated verified result, got %+v", latestVerification)
	}
	if len(detail.Timeline) == 0 {
		t.Fatal("expected rollout timeline events")
	}
	if len(detail.StatusTimeline) == 0 {
		t.Fatal("expected rollout status timeline events")
	}
}

func TestRollbackPoliciesAndStatusEventsAPI(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-status@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-status",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "mission_critical",
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	policy := postItemAuth[types.RollbackPolicy](t, server.URL+"/api/v1/rollback-policies", types.CreateRollbackPolicyRequest{
		OrganizationID:            admin.Session.ActiveOrganizationID,
		ProjectID:                 project.ID,
		ServiceID:                 service.ID,
		EnvironmentID:             environment.ID,
		Name:                      "Prod strict",
		MaxErrorRate:              1,
		RollbackOnCriticalSignals: ptrBool(true),
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if policy.Name != "Prod strict" {
		t.Fatalf("expected rollback policy to be created, got %+v", policy)
	}

	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "release checkout rollback test",
		ChangeTypes:    []string{"code"},
		FileCount:      3,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolloutPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID:      rolloutPlan.Plan.ID,
		BackendType:        "simulated",
		SignalProviderType: "simulated",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.Status == "awaiting_approval" {
		execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
			Action: "approve",
			Reason: "approval granted",
		}, admin.Token, admin.Session.ActiveOrganizationID)
	}
	serviceAccount := postItemAuth[types.ServiceAccount](t, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "worker-bot",
		Role:           "org_member",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	issued := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name: "worker",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	_ = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "begin rollout",
	}, issued.Token, admin.Session.ActiveOrganizationID)
	_ = postItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/reconcile", struct{}{}, issued.Token, admin.Session.ActiveOrganizationID)
	_ = postItemAuth[types.SignalSnapshot](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/signal-snapshots", types.CreateSignalSnapshotRequest{
		ProviderType: "simulated",
		Health:       "critical",
		Summary:      "error rate crossed rollback threshold",
		Signals: []types.SignalValue{
			{Name: "error_rate", Category: "technical", Value: 2.7, Unit: "%", Status: "critical", Threshold: 1, Comparator: ">"},
		},
	}, issued.Token, admin.Session.ActiveOrganizationID)

	detail := postItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/reconcile", struct{}{}, issued.Token, admin.Session.ActiveOrganizationID)
	if detail.Execution.Status != "rolled_back" {
		t.Fatalf("expected rolled back execution, got %s", detail.Execution.Status)
	}
	if detail.EffectiveRollbackPolicy == nil || detail.EffectiveRollbackPolicy.Name != "Prod strict" {
		t.Fatalf("expected effective rollback policy to be attached, got %+v", detail.EffectiveRollbackPolicy)
	}
	if len(detail.VerificationResults) == 0 {
		t.Fatal("expected at least one verification result after rollback-triggering reconcile")
	}
	initialVerificationID := detail.VerificationResults[len(detail.VerificationResults)-1].ID
	initialVerificationCount := len(detail.VerificationResults)

	repeatDetail := postItemAuth[types.RolloutExecutionDetail](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/reconcile", struct{}{}, issued.Token, admin.Session.ActiveOrganizationID)
	if repeatDetail.Execution.Status != "rolled_back" {
		t.Fatalf("expected repeated reconcile to keep rolled back execution, got %s", repeatDetail.Execution.Status)
	}
	if len(repeatDetail.VerificationResults) != initialVerificationCount {
		t.Fatalf("expected repeated reconcile to avoid duplicate automated verification records, got %+v", repeatDetail.VerificationResults)
	}
	if repeatDetail.VerificationResults[len(repeatDetail.VerificationResults)-1].ID != initialVerificationID {
		t.Fatalf("expected repeated reconcile to preserve the latest rollback verification id, got %+v", repeatDetail.VerificationResults)
	}

	statusEvents := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/status-events?rollout_execution_id="+execution.ID+"&rollback_only=true", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(statusEvents) == 0 {
		t.Fatal("expected rollback-related status events")
	}
	if !strings.Contains(strings.ToLower(statusEvents[0].Summary), "rollback") && !strings.Contains(strings.ToLower(strings.Join(statusEvents[0].Explanation, " ")), "rollback") {
		t.Fatalf("expected rollback evidence in status event, got %+v", statusEvents[0])
	}

	filtered := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/status-events?search=rollback&rollback_only=true&service_id="+service.ID+"&limit=10", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(filtered) == 0 {
		t.Fatal("expected filtered rollback status events")
	}
	searchResult := getItemAuth[types.StatusEventQueryResult](t, server.URL+"/api/v1/status-events/search?search=rollback&rollback_only=true&service_id="+service.ID+"&limit=1&offset=0", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if searchResult.Summary.Total == 0 || searchResult.Summary.Returned != 1 {
		t.Fatalf("expected paginated search summary, got %+v", searchResult.Summary)
	}
	if len(searchResult.Events) != 1 {
		t.Fatalf("expected one event in paginated search result, got %+v", searchResult)
	}
	if searchResult.Filters["service_id"] != service.ID {
		t.Fatalf("expected service filter to round-trip, got %+v", searchResult.Filters)
	}
	automatedVerificationEvents := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/status-events?rollout_execution_id="+execution.ID+"&event_type=rollout.execution.verified_automatically&limit=10", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(automatedVerificationEvents) != 1 {
		t.Fatalf("expected exactly one automatic rollback verification status event after repeated reconcile, got %+v", automatedVerificationEvents)
	}

	timeline := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/timeline", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(timeline) == 0 {
		t.Fatal("expected rollout execution timeline")
	}
	projectTimeline := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/projects/"+project.ID+"/status-events", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(projectTimeline) == 0 {
		t.Fatal("expected project status timeline")
	}
	serviceTimeline := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/services/"+service.ID+"/status-events", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(serviceTimeline) == 0 {
		t.Fatal("expected service status timeline")
	}
	environmentTimeline := getListAuth[types.StatusEvent](t, server.URL+"/api/v1/environments/"+environment.ID+"/status-events", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(environmentTimeline) == 0 {
		t.Fatal("expected environment status timeline")
	}

	statusEvent := getItemAuth[types.StatusEvent](t, server.URL+"/api/v1/status-events/"+statusEvents[0].ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if statusEvent.ID != statusEvents[0].ID {
		t.Fatalf("expected status event lookup by id, got %+v", statusEvent)
	}

	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-other-status@acme.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other Status",
		OrganizationSlug: "other-status",
	})
	getItemAuth[types.StatusEvent](t, server.URL+"/api/v1/status-events/"+statusEvents[0].ID, otherOrg.Token, otherOrg.Session.ActiveOrganizationID, http.StatusForbidden)
}

func TestOrgMemberCannotManageRollbackPolicies(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-rbac@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme RBAC",
		OrganizationSlug: "acme-rbac",
	})
	member := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "member-rbac@acme.local",
		DisplayName:      "Member",
		OrganizationSlug: "acme-rbac",
	})
	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	payload, err := json.Marshal(types.CreateRollbackPolicyRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Denied policy",
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/rollback-policies", bytes.NewReader(payload))
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

func TestOrgMemberCannotArchiveService(t *testing.T) {
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

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/services/"+service.ID+"/archive", bytes.NewReader([]byte("{}")))
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

func TestServiceAccountRoutesEnforceScopeAndRBAC(t *testing.T) {
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
		Roles:            []string{"org_member"},
	})
	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "other-owner@acme.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other Org",
		OrganizationSlug: "other-org",
	})

	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "denied-bot",
		Role:           "org_member",
	}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member service-account create to be forbidden, got %d", status)
	}

	serviceAccount := postItemAuth[types.ServiceAccount](t, server.URL+"/api/v1/service-accounts", types.CreateServiceAccountRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "deployer",
		Role:           "org_member",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	issued := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name: "primary",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name: "denied",
	}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member token issue to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens/"+issued.Entry.ID+"/revoke", struct{}{}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member token revoke to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens", types.IssueAPITokenRequest{
		Name: "cross-org",
	}, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org token issue to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens/"+issued.Entry.ID+"/rotate", types.RotateAPITokenRequest{
		Name: "cross-org-rotate",
	}, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org token rotate to be forbidden, got %d", status)
	}

	rotated := postItemAuth[types.IssuedAPITokenResponse](t, server.URL+"/api/v1/service-accounts/"+serviceAccount.ID+"/tokens/"+issued.Entry.ID+"/rotate", types.RotateAPITokenRequest{
		Name: "rotated",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if rotated.Entry.Status != "active" {
		t.Fatalf("expected admin rotate to succeed, got %+v", rotated)
	}
}

func TestRolloutOverrideRoutesEnforceScopeAndRBAC(t *testing.T) {
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
		Roles:            []string{"org_member"},
	})
	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "other-owner@acme.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other Org",
		OrganizationSlug: "other-org",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "Rollout override scope check",
		ChangeTypes:    []string{"code"},
		FileCount:      4,
		ResourceCount:  1,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	risk := postItemAuth[types.RiskAssessmentResult](t, server.URL+"/api/v1/risk-assessments", types.CreateRiskAssessmentRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	plan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: risk.Assessment.ChangeSetID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: plan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "approve before override checks",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution = postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/advance", types.AdvanceRolloutExecutionRequest{
		Action: "start",
		Reason: "start before override checks",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/pause", struct {
		Reason string `json:"reason"`
	}{Reason: "member pause"}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member pause to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/rollback", struct {
		Reason string `json:"reason"`
	}{Reason: "member rollback"}, member.Token, admin.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected org-member rollback to be forbidden, got %d", status)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/pause", struct {
		Reason string `json:"reason"`
	}{Reason: "cross-org pause"}, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org pause to be forbidden, got %d", status)
	}

	paused := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/pause", struct {
		Reason string `json:"reason"`
	}{Reason: "admin pause"}, admin.Token, admin.Session.ActiveOrganizationID)
	if paused.Status != "paused" {
		t.Fatalf("expected admin pause to succeed, got %+v", paused)
	}
}

func TestSignalSnapshotIsTenantScoped(t *testing.T) {
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
	otherAdmin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner@other.local",
		DisplayName:      "Other Owner",
		OrganizationName: "Other",
		OrganizationSlug: "other",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Staging",
		Slug:           "staging",
		Type:           "staging",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "Ship change",
		ChangeTypes:    []string{"code"},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rolloutPlan := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rolloutPlan.Plan.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	payload, err := json.Marshal(types.CreateSignalSnapshotRequest{
		ProviderType: "simulated",
		Health:       "healthy",
		Summary:      "cross-tenant attempt",
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/signal-snapshots", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+otherAdmin.Token)
	req.Header.Set("X-CCP-Organization-ID", otherAdmin.Session.ActiveOrganizationID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
}

func postItemAuth[T any](t *testing.T, url string, body any, token, organizationID string) T {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if organizationID != "" {
		req.Header.Set("X-CCP-Organization-ID", organizationID)
	}
	resp, err := http.DefaultClient.Do(req)
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

func postListAuth[T any](t *testing.T, url string, body any, token, organizationID string) []T {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-CCP-Organization-ID", organizationID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
	var envelope types.ListResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func getItemAuth[T any](t *testing.T, url, token, organizationID string, expectedStatus int) T {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-CCP-Organization-ID", organizationID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}
	var envelope types.ItemResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func getListAuth[T any](t *testing.T, url, token, organizationID string, expectedStatus int) []T {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if organizationID != "" {
		req.Header.Set("X-CCP-Organization-ID", organizationID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, resp.StatusCode)
	}
	if expectedStatus >= http.StatusBadRequest {
		return nil
	}
	var envelope types.ListResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Data
}

func requestStatus(t *testing.T, method, url string, body any, token, organizationID string) int {
	t.Helper()
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if organizationID != "" {
		req.Header.Set("X-CCP-Organization-ID", organizationID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func ptrBool(value bool) *bool {
	return &value
}
