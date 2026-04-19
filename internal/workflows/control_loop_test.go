package workflows_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/app"
	"github.com/change-control-plane/change-control-plane/internal/auth"
	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/internal/workflows"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestControlLoopAdvancesExecutableRollouts(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENABLE_PYTHON_INTELLIGENCE", "false")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())

	adminSession, err := application.DevLogin(context.Background(), types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := authenticatedContext(t, application, adminSession.Token, adminSession.Session.ActiveOrganizationID)

	project, err := application.CreateProject(adminCtx, types.CreateProjectRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	})
	if err != nil {
		t.Fatal(err)
	}
	team, err := application.CreateTeam(adminCtx, types.CreateTeamRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{adminSession.Session.ActorID},
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.CreateService(adminCtx, types.CreateServiceRequest{
		OrganizationID:   adminSession.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Preview API",
		Slug:             "preview-api",
		Criticality:      "low",
		HasSLO:           true,
		HasObservability: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	environment, err := application.CreateEnvironment(adminCtx, types.CreateEnvironmentRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Staging",
		Slug:           "staging",
		Type:           "staging",
		Region:         "us-central1",
	})
	if err != nil {
		t.Fatal(err)
	}
	change, err := application.CreateChangeSet(adminCtx, types.CreateChangeSetRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "tune retry policy",
		ChangeTypes:    []string{"code"},
		FileCount:      2,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := application.CreateRolloutPlan(adminCtx, types.CreateRolloutPlanRequest{ChangeSetID: change.ID})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := application.CreateRolloutExecution(adminCtx, types.CreateRolloutExecutionRequest{RolloutPlanID: plan.Plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != "planned" {
		t.Fatalf("expected planned execution, got %s", execution.Status)
	}

	serviceAccount, err := application.CreateServiceAccount(adminCtx, types.CreateServiceAccountRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "worker-bot",
		Role:           "org_member",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := application.IssueServiceAccountToken(adminCtx, serviceAccount.ID, types.IssueAPITokenRequest{Name: "worker"})
	if err != nil {
		t.Fatal(err)
	}
	workerCtx := authenticatedContext(t, application, issued.Token, adminSession.Session.ActiveOrganizationID)

	controller := workflows.NewControlLoop(application, true)
	summary, err := controller.RunOnce(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Started != 1 {
		t.Fatalf("expected one started rollout, got %+v", summary)
	}

	detail, err := application.GetRolloutExecutionDetail(workerCtx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Execution.Status != "in_progress" {
		t.Fatalf("expected in_progress after worker start, got %s", detail.Execution.Status)
	}

	if _, err := application.RecordVerificationResult(workerCtx, execution.ID, types.RecordVerificationResultRequest{
		Outcome:  "pass",
		Decision: "continue",
		Summary:  "all technical and business signals remain healthy",
		Signals:  []string{"latency", "error-rate"},
	}); err != nil {
		t.Fatal(err)
	}

	summary, err = controller.RunOnce(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Completed != 1 {
		t.Fatalf("expected one completed rollout, got %+v", summary)
	}

	detail, err = application.GetRolloutExecutionDetail(workerCtx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Execution.Status != "completed" {
		t.Fatalf("expected completed after worker reconciliation, got %s", detail.Execution.Status)
	}
}

func TestControlLoopDoesNotBypassApproval(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENABLE_PYTHON_INTELLIGENCE", "false")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())

	adminSession, err := application.DevLogin(context.Background(), types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := authenticatedContext(t, application, adminSession.Token, adminSession.Session.ActiveOrganizationID)

	project, err := application.CreateProject(adminCtx, types.CreateProjectRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	})
	if err != nil {
		t.Fatal(err)
	}
	team, err := application.CreateTeam(adminCtx, types.CreateTeamRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{adminSession.Session.ActorID},
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.CreateService(adminCtx, types.CreateServiceRequest{
		OrganizationID:   adminSession.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "mission_critical",
		CustomerFacing:   true,
		HasSLO:           true,
		HasObservability: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	environment, err := application.CreateEnvironment(adminCtx, types.CreateEnvironmentRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	change, err := application.CreateChangeSet(adminCtx, types.CreateChangeSetRequest{
		OrganizationID:        adminSession.Session.ActiveOrganizationID,
		ProjectID:             project.ID,
		ServiceID:             service.ID,
		EnvironmentID:         environment.ID,
		Summary:               "ship high risk release",
		ChangeTypes:           []string{"code", "iam"},
		FileCount:             10,
		TouchesInfrastructure: true,
		TouchesIAM:            true,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := application.CreateRolloutPlan(adminCtx, types.CreateRolloutPlanRequest{ChangeSetID: change.ID})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := application.CreateRolloutExecution(adminCtx, types.CreateRolloutExecutionRequest{RolloutPlanID: plan.Plan.ID})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != "awaiting_approval" {
		t.Fatalf("expected awaiting_approval execution, got %s", execution.Status)
	}

	serviceAccount, err := application.CreateServiceAccount(adminCtx, types.CreateServiceAccountRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "worker-bot",
		Role:           "org_member",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := application.IssueServiceAccountToken(adminCtx, serviceAccount.ID, types.IssueAPITokenRequest{Name: "worker"})
	if err != nil {
		t.Fatal(err)
	}
	workerCtx := authenticatedContext(t, application, issued.Token, adminSession.Session.ActiveOrganizationID)

	controller := workflows.NewControlLoop(application, true)
	summary, err := controller.RunOnce(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Started != 0 || summary.Completed != 0 {
		t.Fatalf("expected no automatic transition for awaiting approval, got %+v", summary)
	}

	detail, err := application.GetRolloutExecutionDetail(workerCtx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Execution.Status != "awaiting_approval" {
		t.Fatalf("expected awaiting_approval to remain unchanged, got %s", detail.Execution.Status)
	}
}

func TestScheduledIntegrationSyncClaimPreventsDuplicateWorkers(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())

	adminSession, err := application.DevLogin(context.Background(), types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-sync-claim",
	})
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := authenticatedContext(t, application, adminSession.Token, adminSession.Session.ActiveOrganizationID)

	integrations, err := application.IntegrationsList(adminCtx)
	if err != nil {
		t.Fatal(err)
	}
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

	due := time.Now().UTC().Add(-time.Minute)
	githubIntegration.Enabled = true
	githubIntegration.ScheduleEnabled = true
	githubIntegration.ScheduleIntervalSeconds = 300
	githubIntegration.SyncStaleAfterSeconds = 900
	githubIntegration.NextScheduledSyncAt = &due
	githubIntegration.UpdatedAt = time.Now().UTC()
	if err := application.Store.UpdateIntegration(adminCtx, githubIntegration); err != nil {
		t.Fatal(err)
	}

	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claimed, claimErr := application.ClaimScheduledIntegrationSync(adminCtx, githubIntegration.ID, time.Now().UTC())
			if claimErr != nil {
				t.Error(claimErr)
				return
			}
			if claimed {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()

	if successes.Load() != 1 {
		t.Fatalf("expected exactly one successful claim, got %d", successes.Load())
	}
}

func TestControlLoopRunsScheduledIntegrationSyncs(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")

	kubeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/apis/apps/v1/namespaces/prod/deployments/checkout":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
				"spec":     map[string]any{"paused": false},
				"status": map[string]any{
					"replicas":            3,
					"updatedReplicas":     3,
					"availableReplicas":   3,
					"unavailableReplicas": 0,
					"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
				},
			})
		case "/apis/apps/v1/namespaces/prod/deployments":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"items": []map[string]any{{
					"metadata": map[string]any{"name": "checkout", "namespace": "prod"},
					"spec":     map[string]any{"paused": false},
					"status": map[string]any{
						"replicas":            3,
						"updatedReplicas":     3,
						"availableReplicas":   3,
						"unavailableReplicas": 0,
						"conditions":          []map[string]any{{"type": "Available", "status": "True"}},
					},
				}},
			})
		default:
			t.Fatalf("unexpected kubernetes path %s", r.URL.Path)
		}
	}))
	defer kubeServer.Close()

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())

	adminSession, err := application.DevLogin(context.Background(), types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme-runtime",
	})
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := authenticatedContext(t, application, adminSession.Token, adminSession.Session.ActiveOrganizationID)

	project, err := application.CreateProject(adminCtx, types.CreateProjectRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	})
	if err != nil {
		t.Fatal(err)
	}
	team, err := application.CreateTeam(adminCtx, types.CreateTeamRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{adminSession.Session.ActorID},
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.CreateService(adminCtx, types.CreateServiceRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		TeamID:         team.ID,
		Name:           "Checkout",
		Slug:           "checkout",
	})
	if err != nil {
		t.Fatal(err)
	}
	environment, err := application.CreateEnvironment(adminCtx, types.CreateEnvironmentRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Production:     true,
	})
	if err != nil {
		t.Fatal(err)
	}

	integrations, err := application.IntegrationsList(adminCtx)
	if err != nil {
		t.Fatal(err)
	}
	var kubeIntegration types.Integration
	for _, integration := range integrations {
		if integration.Kind == "kubernetes" {
			kubeIntegration = integration
			break
		}
	}
	if kubeIntegration.ID == "" {
		t.Fatal("expected kubernetes integration")
	}

	scheduleEnabled := true
	scheduleInterval := 300
	staleAfter := 900
	enabled := true
	mode := "advisory"
	kubeIntegration, err = application.UpdateIntegration(adminCtx, kubeIntegration.ID, types.UpdateIntegrationRequest{
		Enabled:                 &enabled,
		Mode:                    &mode,
		ScheduleEnabled:         &scheduleEnabled,
		ScheduleIntervalSeconds: &scheduleInterval,
		SyncStaleAfterSeconds:   &staleAfter,
		Metadata: types.Metadata{
			"api_base_url":    kubeServer.URL,
			"namespace":       "prod",
			"deployment_name": "checkout",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if kubeIntegration.NextScheduledSyncAt == nil {
		t.Fatalf("expected scheduled integration to have a due time, got %+v", kubeIntegration)
	}

	serviceAccount, err := application.CreateServiceAccount(adminCtx, types.CreateServiceAccountRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "worker-bot",
		Role:           "org_member",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := application.IssueServiceAccountToken(adminCtx, serviceAccount.ID, types.IssueAPITokenRequest{Name: "worker"})
	if err != nil {
		t.Fatal(err)
	}
	workerCtx := authenticatedContext(t, application, issued.Token, adminSession.Session.ActiveOrganizationID)

	controller := workflows.NewControlLoop(application, true)
	summary, err := controller.RunOnce(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.SyncsClaimed != 1 || summary.SyncsCompleted != 1 {
		t.Fatalf("expected one scheduled sync to run, got %+v", summary)
	}

	refreshedList, err := application.IntegrationsList(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	var refreshed types.Integration
	for _, integration := range refreshedList {
		if integration.ID == kubeIntegration.ID {
			refreshed = integration
			break
		}
	}
	if refreshed.ID == "" {
		t.Fatalf("expected refreshed integration %s in list", kubeIntegration.ID)
	}
	if refreshed.LastSyncSucceededAt == nil || refreshed.FreshnessState != "fresh" {
		t.Fatalf("expected integration to be fresh after scheduled sync, got %+v", refreshed)
	}
	if refreshed.NextScheduledSyncAt == nil || !refreshed.NextScheduledSyncAt.After(time.Now().UTC().Add(4*time.Minute)) {
		t.Fatalf("expected next scheduled sync to move forward, got %+v", refreshed)
	}

	discoveredResources, err := application.ListDiscoveredResources(workerCtx, storage.DiscoveredResourceQuery{
		IntegrationID: kubeIntegration.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(discoveredResources) == 0 {
		t.Fatal("expected scheduled sync to persist discovered kubernetes workloads")
	}
	if discoveredResources[0].ServiceID != service.ID || discoveredResources[0].EnvironmentID != environment.ID {
		t.Fatalf("expected discovered workload to auto-map to matching service and environment, got %+v", discoveredResources[0])
	}

	coverage, err := application.CoverageSummary(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if coverage.WorkloadCoverageEnvironments == 0 {
		t.Fatalf("expected workload coverage after scheduled sync, got %+v", coverage)
	}
}

func TestControlLoopAutomatesVerificationFromSignals(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENABLE_PYTHON_INTELLIGENCE", "false")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())

	adminSession, err := application.DevLogin(context.Background(), types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := authenticatedContext(t, application, adminSession.Token, adminSession.Session.ActiveOrganizationID)

	project, err := application.CreateProject(adminCtx, types.CreateProjectRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	})
	if err != nil {
		t.Fatal(err)
	}
	team, err := application.CreateTeam(adminCtx, types.CreateTeamRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{adminSession.Session.ActorID},
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.CreateService(adminCtx, types.CreateServiceRequest{
		OrganizationID:   adminSession.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Checkout",
		Slug:             "checkout",
		Criticality:      "medium",
		HasSLO:           true,
		HasObservability: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	environment, err := application.CreateEnvironment(adminCtx, types.CreateEnvironmentRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Staging",
		Slug:           "staging",
		Type:           "staging",
		Region:         "us-central1",
	})
	if err != nil {
		t.Fatal(err)
	}
	change, err := application.CreateChangeSet(adminCtx, types.CreateChangeSetRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "ship checkout change",
		ChangeTypes:    []string{"code"},
		FileCount:      3,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := application.CreateRolloutPlan(adminCtx, types.CreateRolloutPlanRequest{ChangeSetID: change.ID})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := application.CreateRolloutExecution(adminCtx, types.CreateRolloutExecutionRequest{
		RolloutPlanID:      plan.Plan.ID,
		BackendType:        "simulated",
		SignalProviderType: "simulated",
	})
	if err != nil {
		t.Fatal(err)
	}

	serviceAccount, err := application.CreateServiceAccount(adminCtx, types.CreateServiceAccountRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "worker-bot",
		Role:           "org_member",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := application.IssueServiceAccountToken(adminCtx, serviceAccount.ID, types.IssueAPITokenRequest{Name: "worker"})
	if err != nil {
		t.Fatal(err)
	}
	workerCtx := authenticatedContext(t, application, issued.Token, adminSession.Session.ActiveOrganizationID)
	controller := workflows.NewControlLoop(application, true)

	first, err := controller.RunOnce(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if first.Started != 1 {
		t.Fatalf("expected control loop to start the rollout, got %+v", first)
	}

	if _, err := application.CreateSignalSnapshot(workerCtx, execution.ID, types.CreateSignalSnapshotRequest{
		ProviderType: "simulated",
		Health:       "healthy",
		Summary:      "latency and error rate remain healthy",
		Signals: []types.SignalValue{
			{Name: "latency_p95_ms", Category: "technical", Value: 145, Unit: "ms", Status: "healthy", Threshold: 250, Comparator: ">"},
			{Name: "error_rate", Category: "technical", Value: 0.2, Unit: "%", Status: "healthy", Threshold: 1, Comparator: ">"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	second, err := controller.RunOnce(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if second.AutomatedDecisions == 0 {
		t.Fatalf("expected an automated verification decision, got %+v", second)
	}

	detail, err := application.GetRolloutExecutionDetail(workerCtx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Execution.Status != "verified" {
		t.Fatalf("expected verified execution after automated evaluation, got %s", detail.Execution.Status)
	}
	latestVerification := detail.VerificationResults[len(detail.VerificationResults)-1]
	if !latestVerification.Automated || latestVerification.Decision != "verified" {
		t.Fatalf("expected automated verified result, got %+v", latestVerification)
	}

	third, err := controller.RunOnce(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if third.Completed != 1 {
		t.Fatalf("expected rollout completion after verified state, got %+v", third)
	}

	detail, err = application.GetRolloutExecutionDetail(workerCtx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Execution.Status != "completed" {
		t.Fatalf("expected completed rollout after final reconcile, got %s", detail.Execution.Status)
	}
}

func TestControlLoopRollsBackCriticalProductionSignals(t *testing.T) {
	t.Setenv("CCP_AUTH_MODE", "dev")
	t.Setenv("CCP_ENABLE_PYTHON_INTELLIGENCE", "false")

	cfg := common.LoadConfig()
	application := app.NewApplicationWithStore(cfg, app.NewInMemoryStore())

	adminSession, err := application.DevLogin(context.Background(), types.DevLoginRequest{
		Email:            "owner@acme.local",
		DisplayName:      "Owner",
		OrganizationName: "Acme",
		OrganizationSlug: "acme",
	})
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := authenticatedContext(t, application, adminSession.Token, adminSession.Session.ActiveOrganizationID)

	project, err := application.CreateProject(adminCtx, types.CreateProjectRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "Platform",
		Slug:           "platform",
	})
	if err != nil {
		t.Fatal(err)
	}
	team, err := application.CreateTeam(adminCtx, types.CreateTeamRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Core",
		Slug:           "core",
		OwnerUserIDs:   []string{adminSession.Session.ActorID},
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.CreateService(adminCtx, types.CreateServiceRequest{
		OrganizationID:   adminSession.Session.ActiveOrganizationID,
		ProjectID:        project.ID,
		TeamID:           team.ID,
		Name:             "Payments",
		Slug:             "payments",
		Criticality:      "mission_critical",
		CustomerFacing:   true,
		HasSLO:           true,
		HasObservability: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	environment, err := application.CreateEnvironment(adminCtx, types.CreateEnvironmentRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		Name:           "Production",
		Slug:           "prod",
		Type:           "production",
		Region:         "us-central1",
		Production:     true,
	})
	if err != nil {
		t.Fatal(err)
	}
	change, err := application.CreateChangeSet(adminCtx, types.CreateChangeSetRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		ProjectID:      project.ID,
		ServiceID:      service.ID,
		EnvironmentID:  environment.ID,
		Summary:        "ship risky payment change",
		ChangeTypes:    []string{"code"},
		FileCount:      6,
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := application.CreateRolloutPlan(adminCtx, types.CreateRolloutPlanRequest{ChangeSetID: change.ID})
	if err != nil {
		t.Fatal(err)
	}
	execution, err := application.CreateRolloutExecution(adminCtx, types.CreateRolloutExecutionRequest{
		RolloutPlanID:      plan.Plan.ID,
		BackendType:        "simulated",
		SignalProviderType: "simulated",
	})
	if err != nil {
		t.Fatal(err)
	}
	execution, err = application.AdvanceRolloutExecution(adminCtx, execution.ID, types.AdvanceRolloutExecutionRequest{
		Action: "approve",
		Reason: "production approval granted",
	})
	if err != nil {
		t.Fatal(err)
	}
	if execution.Status != "approved" {
		t.Fatalf("expected approved execution before worker start, got %s", execution.Status)
	}

	serviceAccount, err := application.CreateServiceAccount(adminCtx, types.CreateServiceAccountRequest{
		OrganizationID: adminSession.Session.ActiveOrganizationID,
		Name:           "worker-bot",
		Role:           "org_member",
	})
	if err != nil {
		t.Fatal(err)
	}
	issued, err := application.IssueServiceAccountToken(adminCtx, serviceAccount.ID, types.IssueAPITokenRequest{Name: "worker"})
	if err != nil {
		t.Fatal(err)
	}
	workerCtx := authenticatedContext(t, application, issued.Token, adminSession.Session.ActiveOrganizationID)
	controller := workflows.NewControlLoop(application, true)

	if _, err := controller.RunOnce(workerCtx); err != nil {
		t.Fatal(err)
	}
	if _, err := application.CreateSignalSnapshot(workerCtx, execution.ID, types.CreateSignalSnapshotRequest{
		ProviderType: "simulated",
		Health:       "critical",
		Summary:      "checkout latency and error rate breached the rollback threshold",
		Signals: []types.SignalValue{
			{Name: "latency_p95_ms", Category: "technical", Value: 710, Unit: "ms", Status: "critical", Threshold: 250, Comparator: ">"},
			{Name: "error_rate", Category: "technical", Value: 4.8, Unit: "%", Status: "critical", Threshold: 1, Comparator: ">"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	summary, err := controller.RunOnce(workerCtx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.AutomatedDecisions == 0 {
		t.Fatalf("expected automated rollback decision, got %+v", summary)
	}

	detail, err := application.GetRolloutExecutionDetail(workerCtx, execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Execution.Status != "rolled_back" {
		t.Fatalf("expected rolled_back status, got %s", detail.Execution.Status)
	}
	latestVerification := detail.VerificationResults[len(detail.VerificationResults)-1]
	if latestVerification.Decision != "rollback" || !latestVerification.Automated {
		t.Fatalf("expected automated rollback verification, got %+v", latestVerification)
	}
	if len(detail.StatusTimeline) == 0 {
		t.Fatal("expected status timeline entries after automated rollback")
	}
}

func authenticatedContext(t *testing.T, application *app.Application, token, organizationID string) context.Context {
	t.Helper()
	identity, err := application.Auth.LoadIdentity(context.Background(), "Bearer "+token, organizationID)
	if err != nil {
		t.Fatal(err)
	}
	return auth.WithIdentity(context.Background(), identity)
}
