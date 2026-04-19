package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestRunValidateReportSuccess(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "reference-pilot-report.json")
	report := validReferencePilotReport()
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, body, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"--validate-report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"profile": "reference_pilot"`) {
		t.Fatalf("expected normalized reference pilot report, got %s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %s", stderr.String())
	}
}

func TestRunValidateReportRejectsInvalidReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "reference-pilot-report.json")
	report := validReferencePilotReport()
	report.ExecutionDetail.RuntimeSummary.AdvisoryOnly = false
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, body, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"--validate-report", reportPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "invalid reference pilot report") {
		t.Fatalf("expected invalid report error, got %s", stderr.String())
	}
}

func TestRunValidateReportNormalizesLegacyReport(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "reference-pilot-report.json")
	report := validReferencePilotReport()
	report.Profile = ""
	report.VerifiedAt = ""
	report.ProofQuality = ""
	report.EvidenceSummary = nil
	body, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, body, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"--validate-report", reportPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 for legacy report, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"profile": "reference_pilot"`) || !strings.Contains(stdout.String(), `"proof_quality": "meaningful"`) {
		t.Fatalf("expected normalized legacy report output, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"evidence_summary"`) {
		t.Fatalf("expected normalized evidence summary, got %s", stdout.String())
	}
}

func validReferencePilotReport() verificationReport {
	return verificationReport{
		Profile:      referencePilotProfile,
		VerifiedAt:   time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		ProofQuality: referencePilotProofQuality,
		EvidenceSummary: []string{
			"gitlab repository checkout-api is mapped through the reference fixture",
			"runtime advisory evidence retained latest_decision=advisory_rollback disposition=suppressed",
		},
		Organization:          types.Organization{BaseRecord: types.BaseRecord{ID: "org_123"}},
		Project:               types.Project{BaseRecord: types.BaseRecord{ID: "proj_123"}},
		Team:                  types.Team{BaseRecord: types.BaseRecord{ID: "team_123"}},
		Service:               types.Service{BaseRecord: types.BaseRecord{ID: "svc_123"}},
		Environment:           types.Environment{BaseRecord: types.BaseRecord{ID: "env_123"}},
		GitLabIntegration:     types.Integration{BaseRecord: types.BaseRecord{ID: "int_gitlab"}, Status: "connected"},
		KubernetesIntegration: types.Integration{BaseRecord: types.BaseRecord{ID: "int_kube"}, Status: "connected"},
		PrometheusIntegration: types.Integration{BaseRecord: types.BaseRecord{ID: "int_prom"}, Status: "connected"},
		WebhookRegistration: types.WebhookRegistrationResult{
			Registration: types.WebhookRegistration{BaseRecord: types.BaseRecord{ID: "whr_123"}},
		},
		Repository: types.Repository{
			BaseRecord: types.BaseRecord{ID: "repo_123"},
			Name:       "checkout-api",
			Provider:   "gitlab",
			Status:     "mapped",
		},
		KubernetesResource: types.DiscoveredResource{
			BaseRecord:   types.BaseRecord{ID: "dr_kube_123"},
			Name:         "checkout",
			ResourceType: "kubernetes_workload",
			Status:       "mapped",
		},
		PrometheusResource: types.DiscoveredResource{
			BaseRecord:   types.BaseRecord{ID: "dr_prom_123"},
			Name:         "checkout",
			ResourceType: "prometheus_signal_target",
			Status:       "mapped",
			Health:       "critical",
		},
		ChangeSet:   types.ChangeSet{BaseRecord: types.BaseRecord{ID: "chg_123"}},
		RolloutPlan: types.RolloutPlan{BaseRecord: types.BaseRecord{ID: "plan_123"}},
		Execution:   types.RolloutExecution{BaseRecord: types.BaseRecord{ID: "exec_123"}},
		ExecutionDetail: types.RolloutExecutionDetail{
			RuntimeSummary: types.RolloutExecutionRuntimeSummary{
				AdvisoryOnly:          true,
				LatestDecision:        "advisory_rollback",
				LastActionDisposition: "suppressed",
			},
		},
		StatusEventCount:   3,
		TimelineEventCount: 4,
		AuditEventCount:    2,
	}
}
