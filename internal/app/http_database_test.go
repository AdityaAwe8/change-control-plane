package app_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/pkg/types"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestDatabaseGovernanceRoutesDriveReleaseReadinessAndExecutionSafety(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-database@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Database",
		OrganizationSlug: "acme-database",
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
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "Expand orders schema before guarded API rollout",
		ChangeTypes:    []string{"code", "database"},
		FileCount:      7,
		ResourceCount:  2,
		TouchesSchema:  true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rollout := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	databaseChange := postItemAuth[types.DatabaseChangeDetail](t, server.URL+"/api/v1/database-changes", types.CreateDatabaseChangeRequest{
		OrganizationID:  admin.Session.ActiveOrganizationID,
		ProjectID:       project.ID,
		EnvironmentID:   environment.ID,
		ServiceID:       service.ID,
		ChangeSetID:     change.ID,
		Name:            "Expand orders schema",
		Datastore:       "checkout-primary",
		OperationType:   "schema_change",
		ExecutionIntent: "pre_deploy",
		Compatibility:   "expand_contract",
		Reversibility:   "reversible",
		RiskLevel:       types.RiskLevelHigh,
		Summary:         "Add nullable support columns before the application reads them.",
		Evidence:        []string{"ticket:DB-42", "runbook:orders-schema-plan"},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if databaseChange.DatabaseChange.ID == "" {
		t.Fatal("expected database change to be created")
	}

	databaseCheck := postItemAuth[types.DatabaseValidationCheckDetail](t, server.URL+"/api/v1/database-validation-checks", types.CreateDatabaseValidationCheckRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		EnvironmentID:    environment.ID,
		ServiceID:        service.ID,
		ChangeSetID:      change.ID,
		DatabaseChangeID: databaseChange.DatabaseChange.ID,
		Name:             "Pre-deploy compatibility confirmation",
		Phase:            "pre_deploy",
		CheckType:        "compatibility_check",
		ReadOnly:         true,
		Required:         true,
		ExecutionMode:    "manual_attestation",
		Specification:    "Confirm the current application revision tolerates the expanded schema before rollout starts.",
		Status:           "defined",
		Summary:          "Pending manual DBA confirmation.",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if databaseCheck.ValidationCheck.ID == "" {
		t.Fatal("expected database validation check to be created")
	}

	loadedDatabaseChange := getItemAuth[types.DatabaseChangeDetail](t, server.URL+"/api/v1/database-changes/"+databaseChange.DatabaseChange.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(loadedDatabaseChange.ValidationChecks) != 1 {
		t.Fatalf("expected linked validation checks on database change detail, got %+v", loadedDatabaseChange)
	}

	release := postItemAuth[types.ReleaseAnalysis](t, server.URL+"/api/v1/releases", types.CreateReleaseRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		Name:           "Database-aware release",
		Summary:        "Bundle with explicit DB governance records",
		ChangeSetIDs:   []string{change.ID},
		Version:        "2026.04.23",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if release.DatabasePosture.Status != "blocked" {
		t.Fatalf("expected release to be blocked while required pre-deploy DB check is pending, got %+v", release.DatabasePosture)
	}
	if len(release.DatabaseChanges) != 1 || len(release.DatabaseChecks) != 1 {
		t.Fatalf("expected release analysis to include database governance context, got %+v", release)
	}
	if status := requestStatus(t, http.MethodPost, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rollout.Plan.ID,
		ReleaseID:     release.Release.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID); status != http.StatusBadRequest {
		t.Fatalf("expected blocked database posture to prevent rollout execution creation, got %d", status)
	}

	updatedCheck := patchItemAuth[types.DatabaseValidationCheckDetail](t, server.URL+"/api/v1/database-validation-checks/"+databaseCheck.ValidationCheck.ID, types.UpdateDatabaseValidationCheckRequest{
		Status:            databaseStringPtr("passed"),
		LastResultSummary: databaseStringPtr("DBA confirmed compatibility against the active application version."),
	}, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if updatedCheck.ValidationCheck.Status != "passed" || updatedCheck.ValidationCheck.LastRunAt == nil {
		t.Fatalf("expected passed database check with recorded run timestamp, got %+v", updatedCheck.ValidationCheck)
	}

	released := getItemAuth[types.ReleaseAnalysis](t, server.URL+"/api/v1/releases/"+release.Release.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if released.DatabasePosture.Status != "review_required" {
		t.Fatalf("expected post-update database posture to require review instead of blocking, got %+v", released.DatabasePosture)
	}

	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rollout.Plan.ID,
		ReleaseID:     release.Release.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if execution.ReleaseID != release.Release.ID {
		t.Fatalf("expected rollout execution to retain release mapping, got %+v", execution)
	}

	pack := getItemAuth[types.RolloutEvidencePack](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/evidence-pack", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(pack.DatabaseChanges) != 1 || len(pack.DatabaseChecks) != 1 {
		t.Fatalf("expected evidence pack to include database governance context, got %+v", pack)
	}
	if pack.DatabasePosture == nil || pack.DatabasePosture.Status == "none" {
		t.Fatalf("expected evidence pack to include database posture, got %+v", pack.DatabasePosture)
	}

	otherOrg := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-database-other@acme.local",
		DisplayName:      "Other",
		OrganizationName: "Other Database",
		OrganizationSlug: "other-database",
	})
	if status := requestStatus(t, http.MethodGet, server.URL+"/api/v1/database-changes/"+databaseChange.DatabaseChange.ID, nil, otherOrg.Token, otherOrg.Session.ActiveOrganizationID); status != http.StatusForbidden {
		t.Fatalf("expected cross-org database change request to be forbidden, got %d", status)
	}
}

func databaseStringPtr(value string) *string {
	return &value
}

func TestRuntimeDatabaseValidationExecutionRoutesPersistExecutionTruth(t *testing.T) {
	sourceDSN := databaseTestDSN()
	freshDSN, cleanup, err := createTemporaryDatabaseForAppTests(sourceDSN)
	if err != nil {
		t.Skipf("postgres unavailable for runtime db execution test: %v", err)
	}
	defer cleanup()

	db, err := sql.Open("pgx", freshDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE runtime_validation_target (id SERIAL PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO runtime_validation_target (name) VALUES ('seed')`); err != nil {
		t.Fatal(err)
	}

	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_RUNTIME_SECRET_REF_DSN", freshDSN)
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-runtime-database@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Runtime Database",
		OrganizationSlug: "acme-runtime-database",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform-runtime",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core-runtime",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout-runtime",
		Criticality:      "high",
		CustomerFacing:   true,
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod-runtime",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	change := postItemAuth[types.ChangeSet](t, server.URL+"/api/v1/changes", types.CreateChangeSetRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "Runtime DB validation for guarded release",
		ChangeTypes:    []string{"code", "database"},
		FileCount:      4,
		ResourceCount:  1,
		TouchesSchema:  true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	rollout := postItemAuth[types.RolloutPlanResult](t, server.URL+"/api/v1/rollout-plans", types.CreateRolloutPlanRequest{
		ChangeSetID: change.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	connectionRef := postItemAuth[types.DatabaseConnectionReferenceDetail](t, server.URL+"/api/v1/database-connection-references", types.CreateDatabaseConnectionReferenceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		ServiceID:      service.ID,
		Name:           "checkout-primary-runtime",
		Datastore:      "checkout-primary",
		Driver:         "postgres",
		SourceType:     "secret_ref_dsn",
		SecretRef:      "prod/checkout/db/runtime_dsn",
		SecretRefEnv:   "CCP_RUNTIME_SECRET_REF_DSN",
		Summary:        "Temporary logical secret-ref-backed runtime connection for read-only validation tests.",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if connectionRef.ConnectionReference.ID == "" {
		t.Fatal("expected database connection reference to be created")
	}
	connectionTest := postItemAuth[types.DatabaseConnectionTestDetail](t, server.URL+"/api/v1/database-connection-references/"+connectionRef.ConnectionReference.ID+"/test", types.TestDatabaseConnectionReferenceRequest{
		Trigger: "manual",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if connectionTest.ConnectionTest.Status != "passed" {
		t.Fatalf("expected connection health test to pass, got %+v", connectionTest.ConnectionTest)
	}
	if connectionTest.ConnectionReference.Status != "ready" || connectionTest.ConnectionReference.LastHealthyAt == nil {
		t.Fatalf("expected connection health test to mark the reference ready, got %+v", connectionTest.ConnectionReference)
	}
	for _, candidate := range append([]string{connectionTest.ConnectionTest.Summary, connectionTest.ConnectionReference.LastErrorSummary}, connectionTest.ConnectionTest.Details...) {
		if strings.Contains(candidate, freshDSN) || strings.Contains(candidate, "postgres://") || strings.Contains(strings.ToLower(candidate), "password=") {
			t.Fatalf("expected connection health output to stay redacted, got %q", candidate)
		}
	}
	connectionDetail := getItemAuth[types.DatabaseConnectionReferenceDetail](t, server.URL+"/api/v1/database-connection-references/"+connectionRef.ConnectionReference.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(connectionDetail.ConnectionTests) != 1 || connectionDetail.ConnectionTests[0].ID != connectionTest.ConnectionTest.ID {
		t.Fatalf("expected connection detail to include persisted test history, got %+v", connectionDetail.ConnectionTests)
	}

	databaseChange := postItemAuth[types.DatabaseChangeDetail](t, server.URL+"/api/v1/database-changes", types.CreateDatabaseChangeRequest{
		OrganizationID:  admin.Session.ActiveOrganizationID,
		ProjectID:       project.ID,
		EnvironmentID:   environment.ID,
		ServiceID:       service.ID,
		ChangeSetID:     change.ID,
		Name:            "Schema validation release",
		Datastore:       "checkout-primary",
		OperationType:   "schema_change",
		ExecutionIntent: "pre_deploy",
		Compatibility:   "expand_contract",
		Reversibility:   "reversible",
		RiskLevel:       types.RiskLevelHigh,
		Summary:         "Need runtime DB validation before rollout.",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	runtimeCheck := postItemAuth[types.DatabaseValidationCheckDetail](t, server.URL+"/api/v1/database-validation-checks", types.CreateDatabaseValidationCheckRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		EnvironmentID:    environment.ID,
		ServiceID:        service.ID,
		ChangeSetID:      change.ID,
		DatabaseChangeID: databaseChange.DatabaseChange.ID,
		ConnectionRefID:  connectionRef.ConnectionReference.ID,
		Name:             "Runtime table existence check",
		Phase:            "pre_deploy",
		CheckType:        "existence_assertion",
		ReadOnly:         true,
		Required:         true,
		ExecutionMode:    "runtime_read_only",
		Specification:    `{"subject":"table","schema":"public","table":"runtime_validation_target"}`,
		Summary:          "Confirm the runtime validation target table exists before rollout.",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if runtimeCheck.ValidationCheck.ConnectionRefID != connectionRef.ConnectionReference.ID {
		t.Fatalf("expected runtime check to retain connection ref, got %+v", runtimeCheck.ValidationCheck)
	}

	initialRelease := postItemAuth[types.ReleaseAnalysis](t, server.URL+"/api/v1/releases", types.CreateReleaseRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		Name:           "Runtime DB release",
		Summary:        "Release requiring runtime DB validation execution",
		ChangeSetIDs:   []string{change.ID},
		Version:        "2026.04.23-runtime",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if initialRelease.DatabasePosture.Status != "blocked" {
		t.Fatalf("expected release to be blocked before runtime validation executes, got %+v", initialRelease.DatabasePosture)
	}
	if len(initialRelease.DatabaseConnections) != 1 {
		t.Fatalf("expected release analysis to include connection refs, got %+v", initialRelease.DatabaseConnections)
	}
	if len(initialRelease.DatabaseConnectionTests) != 1 || initialRelease.DatabaseConnectionTests[0].ID != connectionTest.ConnectionTest.ID {
		t.Fatalf("expected release analysis to include connection-test evidence, got %+v", initialRelease.DatabaseConnectionTests)
	}

	if status := requestStatus(t, http.MethodPatch, server.URL+"/api/v1/database-validation-checks/"+runtimeCheck.ValidationCheck.ID, types.UpdateDatabaseValidationCheckRequest{
		Status: databaseStringPtr("passed"),
	}, admin.Token, admin.Session.ActiveOrganizationID); status != http.StatusBadRequest {
		t.Fatalf("expected runtime check manual status override to be rejected, got %d", status)
	}

	executed := postItemAuth[types.DatabaseValidationExecutionDetail](t, server.URL+"/api/v1/database-validation-checks/"+runtimeCheck.ValidationCheck.ID+"/execute", types.ExecuteDatabaseValidationCheckRequest{
		Trigger: "manual",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if executed.Execution.Status != "passed" {
		t.Fatalf("expected runtime database execution to pass, got %+v", executed.Execution)
	}
	if executed.ConnectionReference == nil || executed.ConnectionReference.SecretRef != "prod/checkout/db/runtime_dsn" || executed.ConnectionReference.SecretRefEnv != "CCP_RUNTIME_SECRET_REF_DSN" {
		t.Fatalf("expected execution detail to include redacted connection reference, got %+v", executed.ConnectionReference)
	}
	for _, candidate := range append(append([]string{executed.Execution.Summary}, executed.Execution.ResultDetails...), executed.Execution.Evidence...) {
		if strings.Contains(candidate, freshDSN) || strings.Contains(candidate, "postgres://") || strings.Contains(strings.ToLower(candidate), "password=") {
			t.Fatalf("expected execution output to stay redacted, got %q", candidate)
		}
	}

	checkDetail := getItemAuth[types.DatabaseValidationCheckDetail](t, server.URL+"/api/v1/database-validation-checks/"+runtimeCheck.ValidationCheck.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if checkDetail.ValidationCheck.Status != "passed" || checkDetail.ValidationCheck.LastRunAt == nil {
		t.Fatalf("expected runtime check to be updated from execution truth, got %+v", checkDetail.ValidationCheck)
	}
	if len(checkDetail.Executions) != 1 {
		t.Fatalf("expected check detail to include execution history, got %+v", checkDetail.Executions)
	}

	executionList := getListAuth[types.DatabaseValidationExecution](t, server.URL+"/api/v1/database-validation-executions?validation_check_id="+url.QueryEscape(runtimeCheck.ValidationCheck.ID), admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(executionList) != 1 || executionList[0].ID != executed.Execution.ID {
		t.Fatalf("expected execution list to filter by validation check, got %+v", executionList)
	}

	released := getItemAuth[types.ReleaseAnalysis](t, server.URL+"/api/v1/releases/"+initialRelease.Release.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if released.DatabasePosture.Status == "blocked" {
		t.Fatalf("expected executed runtime validation to clear the blocking posture, got %+v", released.DatabasePosture)
	}
	if len(released.DatabaseExecutions) != 1 {
		t.Fatalf("expected release analysis to include runtime execution evidence, got %+v", released.DatabaseExecutions)
	}

	execution := postItemAuth[types.RolloutExecution](t, server.URL+"/api/v1/rollout-executions", types.CreateRolloutExecutionRequest{
		RolloutPlanID: rollout.Plan.ID,
		ReleaseID:     initialRelease.Release.ID,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	pack := getItemAuth[types.RolloutEvidencePack](t, server.URL+"/api/v1/rollout-executions/"+execution.ID+"/evidence-pack", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(pack.DatabaseConnections) != 1 || len(pack.DatabaseConnectionTests) != 1 || len(pack.DatabaseExecutions) != 1 {
		t.Fatalf("expected evidence pack to include runtime db connection and execution context, got %+v", pack)
	}

	failingRuntimeCheck := postItemAuth[types.DatabaseValidationCheckDetail](t, server.URL+"/api/v1/database-validation-checks", types.CreateDatabaseValidationCheckRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		EnvironmentID:    environment.ID,
		ServiceID:        service.ID,
		ChangeSetID:      change.ID,
		DatabaseChangeID: databaseChange.DatabaseChange.ID,
		ConnectionRefID:  connectionRef.ConnectionReference.ID,
		Name:             "Runtime missing table check",
		Phase:            "post_deploy",
		CheckType:        "existence_assertion",
		ReadOnly:         true,
		Required:         false,
		ExecutionMode:    "runtime_read_only",
		Specification:    `{"subject":"table","schema":"public","table":"runtime_validation_missing"}`,
		Summary:          "Confirm failed runtime read-only checks are persisted truthfully.",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	failedExecution := postItemAuth[types.DatabaseValidationExecutionDetail](t, server.URL+"/api/v1/database-validation-checks/"+failingRuntimeCheck.ValidationCheck.ID+"/execute", types.ExecuteDatabaseValidationCheckRequest{
		Trigger: "manual",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if failedExecution.Execution.Status != "failed" || failedExecution.Execution.CompletedAt == nil {
		t.Fatalf("expected missing-table runtime execution to persist as failed, got %+v", failedExecution.Execution)
	}
	if failedExecution.ConnectionReference == nil || failedExecution.ConnectionReference.Status != "ready" {
		t.Fatalf("expected failed assertion to keep connection health separate from validation outcome, got %+v", failedExecution.ConnectionReference)
	}
	failedCheckDetail := getItemAuth[types.DatabaseValidationCheckDetail](t, server.URL+"/api/v1/database-validation-checks/"+failingRuntimeCheck.ValidationCheck.ID, admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if failedCheckDetail.ValidationCheck.Status != "failed" || failedCheckDetail.ValidationCheck.LastResultSummary != failedExecution.Execution.Summary {
		t.Fatalf("expected failed execution to update validation-check truth, got %+v", failedCheckDetail.ValidationCheck)
	}
	failedExecutionList := getListAuth[types.DatabaseValidationExecution](t, server.URL+"/api/v1/database-validation-executions?status=failed", admin.Token, admin.Session.ActiveOrganizationID, http.StatusOK)
	if len(failedExecutionList) != 1 || failedExecutionList[0].ID != failedExecution.Execution.ID {
		t.Fatalf("expected failed execution to be listable by status, got %+v", failedExecutionList)
	}
}

func TestDatabaseConnectionTestingRejectsUnboundSecretRuntimeWithoutLeakingSecrets(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	application := app.NewApplicationWithStore(common.LoadConfig(), app.NewInMemoryStore())
	server := newLocalIPv4Server(t, app.NewHTTPServer(application).Handler())
	defer server.Close()

	admin := loginDev(t, server.URL, types.DevLoginRequest{
		Email:            "owner-secret-database@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme Secret Database",
		OrganizationSlug: "acme-secret-database",
	})

	project := postItemAuth[types.Project](t, server.URL+"/api/v1/projects", types.CreateProjectRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform-secret",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	team := postItemAuth[types.Team](t, server.URL+"/api/v1/teams", types.CreateTeamRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core-secret",
		OwnerUserIDs:   []string{admin.Session.ActorID},
	}, admin.Token, admin.Session.ActiveOrganizationID)
	service := postItemAuth[types.Service](t, server.URL+"/api/v1/services", types.CreateServiceRequest{
		OrganizationID:   admin.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout-secret",
		Criticality:      "high",
		CustomerFacing:   true,
		HasSLO:           true,
		HasObservability: true,
	}, admin.Token, admin.Session.ActiveOrganizationID)
	environment := postItemAuth[types.Environment](t, server.URL+"/api/v1/environments", types.CreateEnvironmentRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod-secret",
		Type:           "production",
		Production:     true,
	}, admin.Token, admin.Session.ActiveOrganizationID)

	connectionRef := postItemAuth[types.DatabaseConnectionReferenceDetail](t, server.URL+"/api/v1/database-connection-references", types.CreateDatabaseConnectionReferenceRequest{
		OrganizationID: admin.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		EnvironmentID:  environment.ID,
		ServiceID:      service.ID,
		Name:           "checkout-secret-ref",
		Datastore:      "checkout-primary",
		Driver:         "postgres",
		SourceType:     "secret_ref_dsn",
		SecretRef:      "prod/checkout/db/runtime_dsn",
		Summary:        "Secret-backed reference that should remain unresolved without a runtime env binding.",
	}, admin.Token, admin.Session.ActiveOrganizationID)

	testDetail := postItemAuth[types.DatabaseConnectionTestDetail](t, server.URL+"/api/v1/database-connection-references/"+connectionRef.ConnectionReference.ID+"/test", types.TestDatabaseConnectionReferenceRequest{
		Trigger: "manual",
	}, admin.Token, admin.Session.ActiveOrganizationID)
	if testDetail.ConnectionTest.Status != "blocked" {
		t.Fatalf("expected secret-backed runtime test to be blocked, got %+v", testDetail.ConnectionTest)
	}
	if testDetail.ConnectionReference.Status != "unresolved" {
		t.Fatalf("expected secret-backed connection to remain unresolved, got %+v", testDetail.ConnectionReference)
	}
	if testDetail.ConnectionTest.ErrorClass != "missing_secret_ref_env" {
		t.Fatalf("expected missing secret-ref env error class, got %+v", testDetail.ConnectionTest)
	}
	for _, candidate := range append([]string{testDetail.ConnectionTest.Summary, testDetail.ConnectionReference.LastErrorSummary}, testDetail.ConnectionTest.Details...) {
		if strings.Contains(candidate, "postgres://") || strings.Contains(strings.ToLower(candidate), "password=") {
			t.Fatalf("expected connection test output to stay redacted, got %q", candidate)
		}
	}
}

func databaseTestDSN() string {
	if dsn := getenvFirst("CCP_TEST_DB_DSN", "CCP_DB_DSN"); dsn != "" {
		return dsn
	}
	return "postgres://postgres:postgres@localhost:5432/change_control_plane?sslmode=disable"
}

func getenvFirst(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func createTemporaryDatabaseForAppTests(sourceDSN string) (string, func(), error) {
	adminDB, err := sql.Open("pgx", sourceDSN)
	if err != nil {
		return "", nil, err
	}
	cleanupDB := func() {
		_ = adminDB.Close()
	}
	if err := adminDB.Ping(); err != nil {
		cleanupDB()
		return "", nil, err
	}
	parsed, err := url.Parse(sourceDSN)
	if err != nil {
		cleanupDB()
		return "", nil, err
	}
	dbName := fmt.Sprintf("ccp_app_runtime_%d", time.Now().UTC().UnixNano())
	if _, err := adminDB.Exec(`CREATE DATABASE ` + dbName); err != nil {
		cleanupDB()
		return "", nil, err
	}
	fresh := *parsed
	fresh.Path = "/" + dbName
	fresh.RawPath = ""
	cleanup := func() {
		_, _ = adminDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, dbName)
		_, _ = adminDB.Exec(`DROP DATABASE ` + dbName)
		cleanupDB()
	}
	return fresh.String(), cleanup, nil
}
