package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRunIntegrationsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/integrations" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"integration_github","name":"GitHub","kind":"github"}]}`))
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"integrations", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "GitHub") {
		t.Fatalf("expected GitHub in output, got %s", stdout.String())
	}
}

func TestRunGraphListEncodesFilters(t *testing.T) {
	var seenQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/graph/relationships" {
			http.NotFound(w, r)
			return
		}
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"graph_123","relationship_type":"team_repository_owner","from_resource_type":"team","from_resource_id":"team_1","to_resource_type":"repository","to_resource_id":"repo_1","status":"observed","metadata":{"provenance_source":"inferred_owner"}}]}`))
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"graph", "list", "--type", "team_repository_owner", "--from", "team_1", "--limit", "25"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(seenQuery, "relationship_type=team_repository_owner") || !strings.Contains(seenQuery, "from_resource_id=team_1") {
		t.Fatalf("expected encoded graph filters, got %q", seenQuery)
	}
	if !strings.Contains(stdout.String(), `"provenance_source": "inferred_owner"`) {
		t.Fatalf("expected graph provenance in output, got %s", stdout.String())
	}
}

func TestRunAuthLoginPersistsSession(t *testing.T) {
	dir := t.TempDir()
	sessionFile := filepath.Join(dir, "session.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/dev/login":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"token":"token-123","session":{"authenticated":true,"mode":"dev","actor":"owner@acme.local","email":"owner@acme.local","active_organization_id":"org_123","organizations":[{"organization_id":"org_123","organization":"Acme","role":"org_admin"}]}}}`))
		case "/api/v1/auth/session":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"authenticated":true,"mode":"dev","actor":"owner@acme.local","email":"owner@acme.local","active_organization_id":"org_123"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_CLI_SESSION_PATH", sessionFile)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"auth", "login", "--email", "owner@acme.local", "--display-name", "Owner", "--org-name", "Acme", "--org-slug", "acme"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}

	payload, err := os.ReadFile(sessionFile)
	if err != nil {
		t.Fatal(err)
	}
	var session cliSession
	if err := json.Unmarshal(payload, &session); err != nil {
		t.Fatal(err)
	}
	if session.Token != "token-123" || session.OrganizationID != "org_123" {
		t.Fatalf("unexpected session payload: %+v", session)
	}
}

func TestRunAuthSession(t *testing.T) {
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/session" {
			http.NotFound(w, r)
			return
		}
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"authenticated":true,"mode":"password","actor":"owner@acme.local","email":"owner@acme.local","active_organization_id":"org_123","organizations":[{"organization_id":"org_123","organization":"Acme","role":"org_admin"}]}}`))
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"auth", "session"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if authorization != "Bearer token-123" {
		t.Fatalf("expected bearer token authorization header, got %q", authorization)
	}
	if !strings.Contains(stdout.String(), `"authenticated": true`) || !strings.Contains(stdout.String(), `"mode": "password"`) {
		t.Fatalf("expected session payload in output, got %s", stdout.String())
	}
}

func TestRunStatusListEncodesFilters(t *testing.T) {
	var seenQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/status-events/search" {
			http.NotFound(w, r)
			return
		}
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"events":[{"id":"status_123","event_type":"rollout.execution.rollback","summary":"rollback triggered"}],"summary":{"total":1,"returned":1,"limit":25,"offset":5,"rollback_events":1,"automated_events":0}}}`))
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"status", "list", "--service", "svc_123", "--search", "rollback required", "--rollback-only", "--source", "kubernetes", "--event-type", "rollout.execution.rollback", "--automated", "false", "--limit", "25", "--offset", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(seenQuery, "search=rollback+required") {
		t.Fatalf("expected encoded search query, got %q", seenQuery)
	}
	if !strings.Contains(seenQuery, "source=kubernetes") || !strings.Contains(seenQuery, "event_type=rollout.execution.rollback") {
		t.Fatalf("expected richer status filters, got %q", seenQuery)
	}
	if !strings.Contains(stdout.String(), "rollback triggered") {
		t.Fatalf("expected rollback event in output, got %s", stdout.String())
	}
}

func TestRunOutboxRecoveryCommands(t *testing.T) {
	var seenPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/api/v1/outbox-events/evt_error_123/retry":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"evt_error_123","event_type":"integration.sync.requested","organization_id":"org_123","resource_type":"integration","resource_id":"int_123","status":"pending","attempts":2,"last_error":"temporary dispatch failure","metadata":{"last_error_class":"temporary","manual_recovery_last_action":"retry"}}}`))
		case "/api/v1/outbox-events/evt_dead_123/requeue":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"evt_dead_123","event_type":"webhook.received","organization_id":"org_123","resource_type":"integration","resource_id":"int_456","status":"pending","attempts":5,"last_error":"permanent payload failure","metadata":{"last_error_class":"permanent","manual_recovery_last_action":"requeue"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"outbox", "retry", "--id", "evt_error_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from outbox retry, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "pending"`) || !strings.Contains(stdout.String(), `"manual_recovery_last_action": "retry"`) {
		t.Fatalf("expected retry output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"outbox", "requeue", "--id", "evt_dead_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from outbox requeue, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"manual_recovery_last_action": "requeue"`) {
		t.Fatalf("expected requeue output, got %s", stdout.String())
	}

	expected := []string{
		"POST /api/v1/outbox-events/evt_error_123/retry",
		"POST /api/v1/outbox-events/evt_dead_123/requeue",
	}
	if len(seenPaths) != len(expected) {
		t.Fatalf("expected %d outbox recovery calls, got %d (%v)", len(expected), len(seenPaths), seenPaths)
	}
	for index, path := range expected {
		if seenPaths[index] != path {
			t.Fatalf("expected outbox recovery path %s at index %d, got %s", path, index, seenPaths[index])
		}
	}
}

func TestRunPolicyCommands(t *testing.T) {
	var createBody map[string]any
	var updateBodies []map[string]any
	var seenPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/policies":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"pol_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","name":"Production Review","code":"production-review","scope":"environment","applies_to":"rollout_plan","mode":"require_manual_review","enabled":true,"priority":100,"description":"Require manual review for high-risk production rollout planning.","conditions":{"min_risk_level":"high","production_only":true},"triggers":["risk>=high","environment=production"]}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/policies/pol_123":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"pol_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","name":"Production Review","code":"production-review","scope":"environment","applies_to":"rollout_plan","mode":"require_manual_review","enabled":true,"priority":100,"description":"Require manual review for high-risk production rollout planning.","conditions":{"min_risk_level":"high","production_only":true},"triggers":["risk>=high","environment=production"]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/policies":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"pol_456","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","name":"Critical Schema Freeze","code":"critical-schema-freeze","scope":"environment","applies_to":"rollout_plan","mode":"block","enabled":true,"priority":140,"description":"Block critical schema rollout planning.","conditions":{"min_risk_level":"critical","production_only":true,"required_touches":["schema"]},"triggers":["risk>=critical","environment=production","touches=schema"]}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/policies/pol_123":
			updateBody := map[string]any{}
			if err := json.NewDecoder(r.Body).Decode(&updateBody); err != nil {
				t.Fatal(err)
			}
			updateBodies = append(updateBodies, updateBody)
			enabled := true
			if raw, ok := updateBody["enabled"].(bool); ok {
				enabled = raw
			}
			mode := "require_manual_review"
			if raw, ok := updateBody["mode"].(string); ok && raw != "" {
				mode = raw
			}
			priority := 100
			if raw, ok := updateBody["priority"].(float64); ok {
				priority = int(raw)
			}
			description := "Require manual review for high-risk production rollout planning."
			if raw, ok := updateBody["description"].(string); ok && raw != "" {
				description = raw
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"pol_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","name":"Production Review","code":"production-review","scope":"environment","applies_to":"rollout_plan","mode":"` + mode + `","enabled":` + strconv.FormatBool(enabled) + `,"priority":` + strconv.Itoa(priority) + `,"description":"` + description + `","conditions":{"min_risk_level":"high","production_only":true},"triggers":["risk>=high","environment=production"]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"policy", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from policy list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Production Review") {
		t.Fatalf("expected policy list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"policy", "show", "--id", "pol_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from policy show, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code": "production-review"`) {
		t.Fatalf("expected policy show output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"policy", "create", "--org", "org_123", "--project", "proj_123", "--service", "svc_123", "--env", "env_123", "--name", "Critical Schema Freeze", "--code", "critical-schema-freeze", "--applies-to", "rollout_plan", "--mode", "block", "--priority", "140", "--description", "Block critical schema rollout planning.", "--production-only", "--min-risk-level", "critical", "--required-touches", "schema"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from policy create, got %d, stderr=%s", code, stderr.String())
	}
	if createBody["organization_id"] != "org_123" || createBody["service_id"] != "svc_123" || createBody["applies_to"] != "rollout_plan" {
		t.Fatalf("unexpected policy create body: %+v", createBody)
	}
	createConditions, ok := createBody["conditions"].(map[string]any)
	if !ok || createConditions["min_risk_level"] != "critical" {
		t.Fatalf("expected policy create conditions to include min risk, got %+v", createBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"policy", "update", "--id", "pol_123", "--mode", "block", "--priority", "110", "--description", "Block instead of review."}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from policy update, got %d, stderr=%s", code, stderr.String())
	}
	if len(updateBodies) == 0 || updateBodies[0]["mode"] != "block" {
		t.Fatalf("expected policy update body to carry the requested mode, got %+v", updateBodies)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"policy", "disable", "--id", "pol_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from policy disable, got %d, stderr=%s", code, stderr.String())
	}
	if len(updateBodies) < 2 || updateBodies[1]["enabled"] != false {
		t.Fatalf("expected policy disable to send enabled=false, got %+v", updateBodies)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"policy", "enable", "--id", "pol_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from policy enable, got %d, stderr=%s", code, stderr.String())
	}
	if len(updateBodies) < 3 || updateBodies[2]["enabled"] != true {
		t.Fatalf("expected policy enable to send enabled=true, got %+v", updateBodies)
	}

	expectedPaths := []string{
		"GET /api/v1/policies",
		"GET /api/v1/policies/pol_123",
		"POST /api/v1/policies",
		"PATCH /api/v1/policies/pol_123",
		"PATCH /api/v1/policies/pol_123",
		"PATCH /api/v1/policies/pol_123",
	}
	if len(seenPaths) != len(expectedPaths) {
		t.Fatalf("expected %d policy calls, got %d (%v)", len(expectedPaths), len(seenPaths), seenPaths)
	}
	for index, path := range expectedPaths {
		if seenPaths[index] != path {
			t.Fatalf("expected policy path %s at index %d, got %s", path, index, seenPaths[index])
		}
	}
}

func TestRunReleaseAndConfigSetCommands(t *testing.T) {
	var configSetBody map[string]any
	var releaseBody map[string]any
	var rolloutBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/config-sets":
			if err := json.NewDecoder(r.Body).Decode(&configSetBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"config_set":{"id":"cfg_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","service_id":"svc_123","name":"production-app","version":"v1","status":"active","entries":[{"key":"DB_PASSWORD_REF","value":"prod/checkout/db/password","value_type":"secret_ref"}]},"validation":{"config_set_id":"cfg_123","status":"valid"},"related_releases":[]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/releases":
			if err := json.NewDecoder(r.Body).Decode(&releaseBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"release":{"id":"rel_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","name":"April Bundle","summary":"checkout changes","change_set_ids":["chg_123"],"config_set_ids":["cfg_123"],"version":"2026.04.23","status":"draft"},"change_sets":[{"id":"chg_123","summary":"checkout changes","status":"ingested","change_types":["code"],"file_count":3,"resource_count":1}],"assessments":[{"id":"risk_123","score":72,"level":"high","recommended_rollout_strategy":"canary","recommended_approval_level":"platform-owner","recommended_guardrails":["health-check-gates"],"explanation":["high risk"]}],"combined_risk_score":72,"combined_risk_level":"high","blast_radius":{"scope":"moderate","summary":"moderate blast radius"},"release_summary":"April bundle summary","rollback_guidance":{"safe":true,"strategy":"rollback_previous_bundle","summary":"safe"},"ops_assistant":{"status":"warning","likely_cause":"bundle complexity","guidance":["review readiness"]},"communications":{}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/rollout-executions":
			if err := json.NewDecoder(r.Body).Decode(&rolloutBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"exec_123","rollout_plan_id":"roll_123","release_id":"rel_123","change_set_id":"chg_123","service_id":"svc_123","environment_id":"env_123","status":"awaiting_approval","current_step":"precheck"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{
		"config-set", "create",
		"--org", "org_123",
		"--project", "proj_123",
		"--env", "env_123",
		"--service", "svc_123",
		"--name", "production-app",
		"--version", "v1",
		"--entries-json", `[{"key":"DB_PASSWORD_REF","value":"prod/checkout/db/password","value_type":"secret_ref"}]`,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected config-set create to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if configSetBody["name"] != "production-app" {
		t.Fatalf("expected config-set create body to include name, got %+v", configSetBody)
	}
	if entries, ok := configSetBody["entries"].([]any); !ok || len(entries) != 1 {
		t.Fatalf("expected config-set create to send one entry, got %+v", configSetBody["entries"])
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"release", "create",
		"--org", "org_123",
		"--project", "proj_123",
		"--env", "env_123",
		"--name", "April Bundle",
		"--summary", "checkout changes",
		"--version", "2026.04.23",
		"--changes", "chg_123",
		"--config-sets", "cfg_123",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected release create to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if releaseBody["name"] != "April Bundle" {
		t.Fatalf("expected release create body to include name, got %+v", releaseBody)
	}
	if changeSetIDs, ok := releaseBody["change_set_ids"].([]any); !ok || len(changeSetIDs) != 1 {
		t.Fatalf("expected release create to include change set ids, got %+v", releaseBody["change_set_ids"])
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"rollout", "execute",
		"--plan", "roll_123",
		"--release", "rel_123",
		"--backend", "simulated",
		"--signal", "simulated",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected rollout execute to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if rolloutBody["release_id"] != "rel_123" {
		t.Fatalf("expected rollout execute to include release id, got %+v", rolloutBody)
	}
}

func TestRunDatabaseGovernanceCommands(t *testing.T) {
	var databaseConnectionBody map[string]any
	var databaseConnectionUpdateBody map[string]any
	var databaseConnectionTestBody map[string]any
	var databaseChangeBody map[string]any
	var databaseCheckBody map[string]any
	var databaseChangeUpdateBody map[string]any
	var databaseCheckUpdateBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/database-connection-references":
			if err := json.NewDecoder(r.Body).Decode(&databaseConnectionBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"connection_reference":{"id":"dbconn_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","service_id":"svc_123","name":"checkout-primary","datastore":"checkout-primary","driver":"postgres","source_type":"env_dsn","dsn_env":"CCP_DB_DSN","status":"defined","summary":"runtime db ref"},"validation_checks":[],"connection_tests":[]}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/database-connection-references/dbconn_123":
			if err := json.NewDecoder(r.Body).Decode(&databaseConnectionUpdateBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"connection_reference":{"id":"dbconn_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","service_id":"svc_123","name":"checkout-primary-updated","datastore":"checkout-primary","driver":"postgres","source_type":"secret_ref_dsn","secret_ref":"prod/checkout/db/runtime_dsn","secret_ref_env":"CCP_CHECKOUT_RUNTIME_DSN","status":"defined","summary":"runtime db ref"},"validation_checks":[],"connection_tests":[]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/database-connection-references/dbconn_123/test":
			if err := json.NewDecoder(r.Body).Decode(&databaseConnectionTestBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"connection_test":{"id":"dbct_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","service_id":"svc_123","connection_ref_id":"dbconn_123","trigger":"manual","status":"passed","summary":"database connection checkout-primary-updated is ready for read-only validation","started_at":"2026-04-23T00:00:00Z"},"connection_reference":{"id":"dbconn_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","service_id":"svc_123","name":"checkout-primary-updated","datastore":"checkout-primary","driver":"postgres","source_type":"secret_ref_dsn","secret_ref":"prod/checkout/db/runtime_dsn","secret_ref_env":"CCP_CHECKOUT_RUNTIME_DSN","status":"ready","summary":"runtime db ref"}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/database-changes":
			if err := json.NewDecoder(r.Body).Decode(&databaseChangeBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"database_change":{"id":"dbchg_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","service_id":"svc_123","change_set_id":"chg_123","name":"Expand schema","datastore":"checkout-primary","operation_type":"schema_change","execution_intent":"pre_deploy","compatibility":"expand_contract","reversibility":"reversible","risk_level":"high","status":"defined","summary":"expand schema"},"validation_checks":[]}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/database-changes/dbchg_123":
			if err := json.NewDecoder(r.Body).Decode(&databaseChangeUpdateBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"database_change":{"id":"dbchg_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","service_id":"svc_123","change_set_id":"chg_123","name":"Expand schema","datastore":"checkout-primary","operation_type":"schema_change","execution_intent":"pre_deploy","compatibility":"expand_contract","reversibility":"reversible","risk_level":"high","status":"reviewed","summary":"expanded"},"validation_checks":[]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/database-validation-checks":
			if err := json.NewDecoder(r.Body).Decode(&databaseCheckBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"validation_check":{"id":"dbchk_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","service_id":"svc_123","change_set_id":"chg_123","database_change_id":"dbchg_123","name":"Compatibility check","phase":"pre_deploy","check_type":"compatibility_check","read_only":true,"required":true,"execution_mode":"manual_attestation","specification":"Confirm compatibility","status":"defined","summary":"pending"}}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/database-validation-checks/dbchk_123":
			if err := json.NewDecoder(r.Body).Decode(&databaseCheckUpdateBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"validation_check":{"id":"dbchk_123","organization_id":"org_123","project_id":"proj_123","environment_id":"env_123","service_id":"svc_123","change_set_id":"chg_123","database_change_id":"dbchg_123","name":"Compatibility check","phase":"pre_deploy","check_type":"compatibility_check","read_only":true,"required":true,"execution_mode":"manual_attestation","specification":"Confirm compatibility","status":"passed","summary":"pending","last_result_summary":"confirmed"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{
		"db-connection", "create",
		"--org", "org_123",
		"--project", "proj_123",
		"--env", "env_123",
		"--service", "svc_123",
		"--name", "checkout-primary",
		"--datastore", "checkout-primary",
		"--driver", "postgres",
		"--source-type", "env_dsn",
		"--dsn-env", "CCP_DB_DSN",
		"--summary", "runtime db ref",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected db-connection create to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if databaseConnectionBody["source_type"] != "env_dsn" || databaseConnectionBody["dsn_env"] != "CCP_DB_DSN" {
		t.Fatalf("unexpected db-connection create body: %+v", databaseConnectionBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"db-connection", "update",
		"--id", "dbconn_123",
		"--name", "checkout-primary-updated",
		"--source-type", "secret_ref_dsn",
		"--secret-ref", "prod/checkout/db/runtime_dsn",
		"--secret-ref-env", "CCP_CHECKOUT_RUNTIME_DSN",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected db-connection update to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if databaseConnectionUpdateBody["source_type"] != "secret_ref_dsn" || databaseConnectionUpdateBody["secret_ref"] != "prod/checkout/db/runtime_dsn" || databaseConnectionUpdateBody["secret_ref_env"] != "CCP_CHECKOUT_RUNTIME_DSN" {
		t.Fatalf("unexpected db-connection update body: %+v", databaseConnectionUpdateBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"db-connection", "test",
		"--id", "dbconn_123",
		"--trigger", "manual",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected db-connection test to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if databaseConnectionTestBody["trigger"] != "manual" {
		t.Fatalf("unexpected db-connection test body: %+v", databaseConnectionTestBody)
	}
	if !strings.Contains(stdout.String(), `"status": "passed"`) {
		t.Fatalf("expected db-connection test output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"db-change", "create",
		"--org", "org_123",
		"--project", "proj_123",
		"--env", "env_123",
		"--service", "svc_123",
		"--change", "chg_123",
		"--name", "Expand schema",
		"--datastore", "checkout-primary",
		"--operation", "schema_change",
		"--intent", "pre_deploy",
		"--compatibility", "expand_contract",
		"--reversibility", "reversible",
		"--risk", "high",
		"--summary", "expand schema",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected db-change create to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if databaseChangeBody["operation_type"] != "schema_change" || databaseChangeBody["change_set_id"] != "chg_123" {
		t.Fatalf("unexpected db-change create body: %+v", databaseChangeBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"db-change", "update",
		"--id", "dbchg_123",
		"--status", "reviewed",
		"--summary", "expanded",
		"--manual-approval",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected db-change update to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if databaseChangeUpdateBody["status"] != "reviewed" || databaseChangeUpdateBody["manual_approval_required"] != true {
		t.Fatalf("unexpected db-change update body: %+v", databaseChangeUpdateBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"db-check", "create",
		"--org", "org_123",
		"--project", "proj_123",
		"--env", "env_123",
		"--service", "svc_123",
		"--change", "chg_123",
		"--db-change", "dbchg_123",
		"--name", "Compatibility check",
		"--phase", "pre_deploy",
		"--type", "compatibility_check",
		"--mode", "manual_attestation",
		"--status", "defined",
		"--spec", "Confirm compatibility",
		"--summary", "pending",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected db-check create to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if databaseCheckBody["database_change_id"] != "dbchg_123" || databaseCheckBody["phase"] != "pre_deploy" {
		t.Fatalf("unexpected db-check create body: %+v", databaseCheckBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"db-check", "update",
		"--id", "dbchk_123",
		"--status", "passed",
		"--result", "confirmed",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected db-check update to succeed, got %d, stderr=%s", code, stderr.String())
	}
	if databaseCheckUpdateBody["status"] != "passed" || databaseCheckUpdateBody["last_result_summary"] != "confirmed" {
		t.Fatalf("unexpected db-check update body: %+v", databaseCheckUpdateBody)
	}
}

func TestRunTeamCommands(t *testing.T) {
	var createBody map[string]any
	var updateBodies []map[string]any
	var listCalled bool
	var showCalled bool
	var archiveCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/teams":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"team_123","organization_id":"org_123","project_id":"proj_123","name":"Core","slug":"core","owner_user_ids":["user_1","user_2"],"status":"active"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/teams":
			listCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"team_123","organization_id":"org_123","project_id":"proj_123","name":"Core","slug":"core","owner_user_ids":["user_1","user_2"],"status":"active"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/teams/team_123":
			showCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"team_123","organization_id":"org_123","project_id":"proj_123","name":"Core","slug":"core","owner_user_ids":["user_1","user_2"],"status":"active"}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/teams/team_123":
			updateBody := map[string]any{}
			if err := json.NewDecoder(r.Body).Decode(&updateBody); err != nil {
				t.Fatal(err)
			}
			updateBodies = append(updateBodies, updateBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"team_123","organization_id":"org_123","project_id":"proj_123","name":"Platform Core","slug":"platform-core","owner_user_ids":["user_3"],"status":"inactive"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/teams/team_123/archive":
			archiveCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"team_123","organization_id":"org_123","project_id":"proj_123","name":"Platform Core","slug":"platform-core","owner_user_ids":["user_3"],"status":"archived"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"team", "create", "--org", "org_123", "--project", "proj_123", "--name", "Core", "--slug", "core", "--owners", "user_1,user_2"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from team create, got %d, stderr=%s", code, stderr.String())
	}
	if createBody["project_id"] != "proj_123" || createBody["name"] != "Core" {
		t.Fatalf("unexpected team create body: %+v", createBody)
	}
	owners, ok := createBody["owner_user_ids"].([]any)
	if !ok || len(owners) != 2 {
		t.Fatalf("expected owner_user_ids to be encoded, got %+v", createBody)
	}
	if !strings.Contains(stdout.String(), `"name": "Core"`) {
		t.Fatalf("expected created team output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"team", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from team list, got %d, stderr=%s", code, stderr.String())
	}
	if !listCalled || !strings.Contains(stdout.String(), `"slug": "core"`) {
		t.Fatalf("expected team list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"team", "show", "--id", "team_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from team show, got %d, stderr=%s", code, stderr.String())
	}
	if !showCalled || !strings.Contains(stdout.String(), `"id": "team_123"`) {
		t.Fatalf("expected team show output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"team", "update", "--id", "team_123", "--name", "Platform Core", "--slug", "platform-core", "--owners", "user_3", "--status", "inactive"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from team update, got %d, stderr=%s", code, stderr.String())
	}
	if len(updateBodies) == 0 {
		t.Fatal("expected team update route to be called")
	}
	updateBody := updateBodies[len(updateBodies)-1]
	if updateBody["name"] != "Platform Core" || updateBody["slug"] != "platform-core" || updateBody["status"] != "inactive" {
		t.Fatalf("unexpected team update body: %+v", updateBody)
	}
	owners, ok = updateBody["owner_user_ids"].([]any)
	if !ok || len(owners) != 1 || owners[0] != "user_3" {
		t.Fatalf("expected owner_user_ids update to be encoded, got %+v", updateBody)
	}
	if !strings.Contains(stdout.String(), `"status": "inactive"`) {
		t.Fatalf("expected updated team output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"team", "update", "--id", "team_123", "--owners", ""}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from team update clearing owners, got %d, stderr=%s", code, stderr.String())
	}
	updateBody = updateBodies[len(updateBodies)-1]
	owners, ok = updateBody["owner_user_ids"].([]any)
	if !ok || len(owners) != 0 {
		t.Fatalf("expected empty owner_user_ids update to be encoded, got %+v", updateBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"team", "archive", "--id", "team_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from team archive, got %d, stderr=%s", code, stderr.String())
	}
	if !archiveCalled || !strings.Contains(stdout.String(), `"status": "archived"`) {
		t.Fatalf("expected archived team output, got %s", stdout.String())
	}
}

func TestRunOrgAndProjectCommands(t *testing.T) {
	var createOrgBody map[string]any
	var createProjectBody map[string]any
	var organizationHeaderValues []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		organizationHeaderValues = append(organizationHeaderValues, r.Header.Get("X-CCP-Organization-ID"))
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organizations":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"org_123","name":"Acme","slug":"acme","tier":"growth","mode":"startup"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/organizations":
			if err := json.NewDecoder(r.Body).Decode(&createOrgBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"org_456","name":"Beta","slug":"beta","tier":"enterprise","mode":"startup"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"proj_123","organization_id":"org_123","name":"Platform","slug":"platform","adoption_mode":"advisory","status":"active"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/projects":
			if err := json.NewDecoder(r.Body).Decode(&createProjectBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"proj_456","organization_id":"org_123","name":"Checkout","slug":"checkout","adoption_mode":"active","status":"active"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"org", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from org list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"slug": "acme"`) {
		t.Fatalf("expected org list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"org", "create", "--name", "Beta", "--slug", "beta", "--tier", "enterprise"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from org create, got %d, stderr=%s", code, stderr.String())
	}
	if createOrgBody["name"] != "Beta" || createOrgBody["slug"] != "beta" || createOrgBody["tier"] != "enterprise" {
		t.Fatalf("unexpected org create body: %+v", createOrgBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"project", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from project list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id": "proj_123"`) {
		t.Fatalf("expected project list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"project", "create", "--name", "Checkout", "--slug", "checkout", "--mode", "active"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from project create, got %d, stderr=%s", code, stderr.String())
	}
	if createProjectBody["organization_id"] != "org_123" || createProjectBody["name"] != "Checkout" || createProjectBody["adoption_mode"] != "active" {
		t.Fatalf("unexpected project create body: %+v", createProjectBody)
	}

	if len(organizationHeaderValues) < 4 {
		t.Fatalf("expected organization headers to be recorded for each request, got %v", organizationHeaderValues)
	}
	if organizationHeaderValues[0] != "org_123" || organizationHeaderValues[2] != "org_123" || organizationHeaderValues[3] != "org_123" {
		t.Fatalf("expected project/org routes to carry org header context, got %v", organizationHeaderValues)
	}
}

func TestRunServiceAndEnvironmentCommands(t *testing.T) {
	var serviceCreateBody map[string]any
	var serviceUpdateBody map[string]any
	var environmentCreateBody map[string]any
	var environmentUpdateBody map[string]any
	var seenHeaders []string
	var seenPaths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHeaders = append(seenHeaders, r.Header.Get("X-CCP-Organization-ID"))
		seenPaths = append(seenPaths, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/services":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"svc_123","organization_id":"org_123","project_id":"proj_123","team_id":"team_123","name":"Checkout API","slug":"checkout-api","criticality":"medium","customer_facing":false,"has_slo":true,"has_observability":true,"status":"active"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services":
			if err := json.NewDecoder(r.Body).Decode(&serviceCreateBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"svc_123","organization_id":"org_123","project_id":"proj_123","team_id":"team_123","name":"Checkout API","slug":"checkout-api","criticality":"high","customer_facing":true,"has_slo":true,"has_observability":true,"status":"active"}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/services/svc_123":
			if err := json.NewDecoder(r.Body).Decode(&serviceUpdateBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"svc_123","organization_id":"org_123","project_id":"proj_123","team_id":"team_123","name":"Checkout API v2","slug":"checkout-api","description":"Critical payments path","criticality":"mission_critical","status":"active"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/services/svc_123/archive":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"svc_123","organization_id":"org_123","project_id":"proj_123","team_id":"team_123","name":"Checkout API v2","slug":"checkout-api","status":"archived"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/environments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"env_123","organization_id":"org_123","project_id":"proj_123","name":"Production","slug":"prod","type":"production","region":"us-central1","production":true,"status":"active"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/environments":
			if err := json.NewDecoder(r.Body).Decode(&environmentCreateBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"env_123","organization_id":"org_123","project_id":"proj_123","name":"Production","slug":"prod","type":"production","production":true,"status":"active"}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/environments/env_123":
			if err := json.NewDecoder(r.Body).Decode(&environmentUpdateBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"env_123","organization_id":"org_123","project_id":"proj_123","name":"Production US","slug":"prod","type":"production","region":"us-east1","production":true,"status":"active"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/environments/env_123/archive":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"env_123","organization_id":"org_123","project_id":"proj_123","name":"Production US","slug":"prod","status":"archived"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"service", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from service list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"name": "Checkout API"`) {
		t.Fatalf("expected service list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"service", "register", "--project", "proj_123", "--team", "team_123", "--name", "Checkout API", "--slug", "checkout-api", "--criticality", "high", "--customer-facing"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from service register, got %d, stderr=%s", code, stderr.String())
	}
	if serviceCreateBody["organization_id"] != "org_123" || serviceCreateBody["project_id"] != "proj_123" || serviceCreateBody["team_id"] != "team_123" {
		t.Fatalf("unexpected service create body: %+v", serviceCreateBody)
	}
	if serviceCreateBody["criticality"] != "high" || serviceCreateBody["customer_facing"] != true {
		t.Fatalf("expected service create flags to persist, got %+v", serviceCreateBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"service", "update", "--id", "svc_123", "--name", "Checkout API v2", "--description", "Critical payments path", "--criticality", "mission_critical"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from service update, got %d, stderr=%s", code, stderr.String())
	}
	if serviceUpdateBody["name"] != "Checkout API v2" || serviceUpdateBody["description"] != "Critical payments path" || serviceUpdateBody["criticality"] != "mission_critical" {
		t.Fatalf("unexpected service update body: %+v", serviceUpdateBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"service", "archive", "--id", "svc_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from service archive, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "archived"`) {
		t.Fatalf("expected archived service output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"env", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from env list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"slug": "prod"`) {
		t.Fatalf("expected environment list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"env", "create", "--project", "proj_123", "--name", "Production", "--slug", "prod", "--type", "production", "--production"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from env create, got %d, stderr=%s", code, stderr.String())
	}
	if environmentCreateBody["organization_id"] != "org_123" || environmentCreateBody["project_id"] != "proj_123" || environmentCreateBody["production"] != true {
		t.Fatalf("unexpected environment create body: %+v", environmentCreateBody)
	}
	if environmentCreateBody["type"] != "production" {
		t.Fatalf("expected environment type to persist, got %+v", environmentCreateBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"env", "update", "--id", "env_123", "--name", "Production US", "--region", "us-east1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from env update, got %d, stderr=%s", code, stderr.String())
	}
	if environmentUpdateBody["name"] != "Production US" || environmentUpdateBody["region"] != "us-east1" {
		t.Fatalf("unexpected environment update body: %+v", environmentUpdateBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"env", "archive", "--id", "env_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from env archive, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "archived"`) {
		t.Fatalf("expected archived environment output, got %s", stdout.String())
	}

	if len(seenHeaders) != len(seenPaths) {
		t.Fatalf("expected one organization header per request, got %d headers for %d paths", len(seenHeaders), len(seenPaths))
	}
	for index, value := range seenHeaders {
		if value != "org_123" {
			t.Fatalf("expected service/env request %d to carry org header, got %q for %s", index, value, seenPaths[index])
		}
	}
}

func TestRunTokenAndRepositoryCommands(t *testing.T) {
	var issueBody map[string]any
	var repositoryMapBody map[string]any
	var repositoryListQuery string
	var seenHeaders []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHeaders = append(seenHeaders, r.Header.Get("X-CCP-Organization-ID"))
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/service-accounts/svcacct_123/tokens":
			if err := json.NewDecoder(r.Body).Decode(&issueBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"token":"ccpt_issued_secret","entry":{"id":"token_123","organization_id":"org_123","service_account_id":"svcacct_123","name":"primary","token_prefix":"ccpt_abcd","status":"active"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/service-accounts/svcacct_123/tokens":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"token_123","organization_id":"org_123","service_account_id":"svcacct_123","name":"primary","token_prefix":"ccpt_abcd","status":"active"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/service-accounts/svcacct_123/tokens/token_123/revoke":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"token_123","organization_id":"org_123","service_account_id":"svcacct_123","name":"primary","token_prefix":"ccpt_abcd","status":"revoked"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repositories":
			repositoryListQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"repo_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","source_integration_id":"int_123","name":"checkout","provider":"github","url":"https://github.com/acme/checkout","default_branch":"main","status":"mapped"}]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/repositories/repo_123":
			if err := json.NewDecoder(r.Body).Decode(&repositoryMapBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"repo_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_456","environment_id":"env_456","source_integration_id":"int_123","name":"checkout","provider":"github","url":"https://github.com/acme/checkout","default_branch":"main","status":"mapped"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"token", "issue", "--service-account", "svcacct_123", "--name", "primary", "--expires-in-hours", "24"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from token issue, got %d, stderr=%s", code, stderr.String())
	}
	if issueBody["name"] != "primary" || issueBody["expires_in_hours"] != float64(24) {
		t.Fatalf("unexpected token issue body: %+v", issueBody)
	}
	if !strings.Contains(stdout.String(), `"token": "ccpt_issued_secret"`) {
		t.Fatalf("expected issued token output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"token", "list", "--service-account", "svcacct_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from token list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "active"`) {
		t.Fatalf("expected token list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"token", "revoke", "--service-account", "svcacct_123", "--id", "token_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from token revoke, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "revoked"`) {
		t.Fatalf("expected revoked token output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"repository", "list", "--provider", "github", "--source-integration", "int_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from repository list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(repositoryListQuery, "provider=github") || !strings.Contains(repositoryListQuery, "source_integration_id=int_123") {
		t.Fatalf("expected repository list query filters, got %q", repositoryListQuery)
	}
	if !strings.Contains(stdout.String(), `"name": "checkout"`) {
		t.Fatalf("expected repository list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"repository", "map", "--id", "repo_123", "--project", "proj_123", "--service", "svc_456", "--env", "env_456", "--status", "mapped"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from repository map, got %d, stderr=%s", code, stderr.String())
	}
	if repositoryMapBody["project_id"] != "proj_123" || repositoryMapBody["service_id"] != "svc_456" || repositoryMapBody["environment_id"] != "env_456" || repositoryMapBody["status"] != "mapped" {
		t.Fatalf("unexpected repository map body: %+v", repositoryMapBody)
	}

	for index, header := range seenHeaders {
		if header != "org_123" {
			t.Fatalf("expected request %d to carry org header, got %q", index, header)
		}
	}
}

func TestRunIncidentCommands(t *testing.T) {
	var listQuery string
	var showCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/incidents":
			listQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"incident_rollout_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","title":"Checkout rollout paused in staging","severity":"high","status":"monitoring","related_change":"change_123","impacted_paths":["Checkout","Staging","pause requested"]}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/incidents/incident_rollout_123":
			showCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"incident":{"id":"incident_rollout_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","title":"Checkout rollout paused in staging","severity":"high","status":"monitoring","related_change":"change_123","impacted_paths":["Checkout","Staging","pause requested"]},"rollout_execution_id":"rollout_123","status_timeline":[]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{
		"incident", "list",
		"--project", "proj_123",
		"--service", "svc_123",
		"--env", "env_123",
		"--change", "change_123",
		"--severity", "high",
		"--status", "monitoring",
		"--search", "checkout pause",
		"--limit", "5",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from incident list, got %d, stderr=%s", code, stderr.String())
	}
	for _, fragment := range []string{
		"project_id=proj_123",
		"service_id=svc_123",
		"environment_id=env_123",
		"change_set_id=change_123",
		"severity=high",
		"status=monitoring",
		"search=checkout+pause",
		"limit=5",
	} {
		if !strings.Contains(listQuery, fragment) {
			t.Fatalf("expected incident list query to contain %q, got %q", fragment, listQuery)
		}
	}
	if !strings.Contains(stdout.String(), `"id": "incident_rollout_123"`) || !strings.Contains(stdout.String(), `"status": "monitoring"`) {
		t.Fatalf("expected incident list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"incident", "show", "--id", "incident_rollout_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from incident show, got %d, stderr=%s", code, stderr.String())
	}
	if !showCalled {
		t.Fatal("expected incident show route to be called")
	}
	if !strings.Contains(stdout.String(), `"rollout_execution_id": "rollout_123"`) {
		t.Fatalf("expected incident detail output, got %s", stdout.String())
	}
}

func TestRunRolloutPauseResumeRollbackUseDedicatedRoutes(t *testing.T) {
	var seenPaths []string
	var seenQueries []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		seenPaths = append(seenPaths, r.URL.Path)
		seenQueries = append(seenQueries, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/rollout-executions/rollout_123/pause":
			_, _ = w.Write([]byte(`{"data":{"id":"rollout_123","rollout_plan_id":"plan_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","status":"paused","current_step":"pause"}}`))
		case "/api/v1/rollout-executions/rollout_123/resume":
			_, _ = w.Write([]byte(`{"data":{"id":"rollout_123","rollout_plan_id":"plan_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","status":"in_progress","current_step":"resume"}}`))
		case "/api/v1/rollout-executions/rollout_123/rollback":
			_, _ = w.Write([]byte(`{"data":{"id":"rollout_123","rollout_plan_id":"plan_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","status":"rolled_back","current_step":"rollback"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"rollout", "pause", "--id", "rollout_123", "--reason", "pause for operator review"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from rollout pause, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "paused"`) {
		t.Fatalf("expected pause output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "resume", "--id", "rollout_123", "--reason", "resume after mitigation"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from rollout resume, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "in_progress"`) {
		t.Fatalf("expected resume output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "rollback", "--id", "rollout_123", "--reason", "rollback due to incident"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from rollout rollback, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"status": "rolled_back"`) {
		t.Fatalf("expected rollback output, got %s", stdout.String())
	}

	expectedPaths := []string{
		"/api/v1/rollout-executions/rollout_123/pause",
		"/api/v1/rollout-executions/rollout_123/resume",
		"/api/v1/rollout-executions/rollout_123/rollback",
	}
	if len(seenPaths) != len(expectedPaths) {
		t.Fatalf("expected %d rollout control calls, got %d (%v)", len(expectedPaths), len(seenPaths), seenPaths)
	}
	for index, expectedPath := range expectedPaths {
		if seenPaths[index] != expectedPath {
			t.Fatalf("expected rollout control path %s at index %d, got %s", expectedPath, index, seenPaths[index])
		}
	}
	for _, fragment := range []string{
		"reason=pause+for+operator+review",
		"reason=resume+after+mitigation",
		"reason=rollback+due+to+incident",
	} {
		found := false
		for _, query := range seenQueries {
			if strings.Contains(query, fragment) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected rollout control query fragment %q in %v", fragment, seenQueries)
		}
	}
}

func TestRunIntegrationsUpdatePersistsScheduleFields(t *testing.T) {
	var seen typesUpdateIntegrationRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/integrations/int_123" {
			http.NotFound(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, &seen); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"int_123","name":"GitHub","kind":"github","mode":"advisory","enabled":true,"schedule_enabled":true,"schedule_interval_seconds":300,"sync_stale_after_seconds":900}}`))
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"integrations", "update", "--id", "int_123", "--enabled", "true", "--schedule-enabled", "true", "--schedule-interval", "300", "--stale-after", "900"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if seen.ScheduleEnabled == nil || !*seen.ScheduleEnabled {
		t.Fatalf("expected schedule_enabled to be true, got %+v", seen)
	}
	if seen.ScheduleIntervalSeconds == nil || *seen.ScheduleIntervalSeconds != 300 {
		t.Fatalf("expected schedule interval 300, got %+v", seen)
	}
	if seen.SyncStaleAfterSeconds == nil || *seen.SyncStaleAfterSeconds != 900 {
		t.Fatalf("expected stale after 900, got %+v", seen)
	}
}

func TestRunDiscoveryListAndMap(t *testing.T) {
	var seenListQuery string
	var seenMapBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/discovered-resources":
			seenListQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"dr_123","integration_id":"int_123","resource_type":"kubernetes_workload","provider":"kubernetes","external_id":"prod/checkout","name":"checkout","status":"candidate"}]}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/discovered-resources/dr_123":
			if err := json.NewDecoder(r.Body).Decode(&seenMapBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"dr_123","integration_id":"int_123","resource_type":"kubernetes_workload","provider":"kubernetes","external_id":"prod/checkout","name":"checkout","status":"mapped","service_id":"svc_123","environment_id":"env_123"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"discovery", "list", "--integration", "int_123", "--type", "kubernetes_workload", "--unmapped-only"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from discovery list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(seenListQuery, "integration_id=int_123") || !strings.Contains(seenListQuery, "unmapped_only=true") {
		t.Fatalf("expected discovery filters in query, got %q", seenListQuery)
	}
	if !strings.Contains(stdout.String(), "checkout") {
		t.Fatalf("expected discovery list output to contain resource name, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"discovery", "map", "--id", "dr_123", "--service", "svc_123", "--env", "env_123", "--status", "mapped"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from discovery map, got %d, stderr=%s", code, stderr.String())
	}
	if seenMapBody["service_id"] != "svc_123" || seenMapBody["environment_id"] != "env_123" || seenMapBody["status"] != "mapped" {
		t.Fatalf("unexpected discovery map body: %+v", seenMapBody)
	}
}

func TestRunIdentityProviderCommands(t *testing.T) {
	var createBody map[string]any
	var updateBody map[string]any
	var testCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/identity-providers":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"idp_123","organization_id":"org_123","name":"Acme Okta","kind":"oidc","enabled":true,"status":"configured","connection_health":"healthy"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/identity-providers":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"idp_123","organization_id":"org_123","name":"Acme Okta","kind":"oidc","enabled":true,"status":"configured","connection_health":"unknown"}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/identity-providers/idp_123":
			if err := json.NewDecoder(r.Body).Decode(&updateBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"idp_123","organization_id":"org_123","name":"Acme Workforce","kind":"oidc","enabled":false,"status":"configured","connection_health":"healthy"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/identity-providers/idp_123/test":
			testCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"provider":{"id":"idp_123","organization_id":"org_123","name":"Acme Workforce","kind":"oidc","enabled":false,"status":"configured","connection_health":"healthy"},"status":"healthy","details":["issuer reachable","userinfo reachable"]}}`))
		default:
			http.NotFound(w, r)
			return
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"identity-provider", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from identity-provider list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Acme Okta") || !strings.Contains(stdout.String(), `"connection_health": "healthy"`) {
		t.Fatalf("expected identity-provider list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"identity-provider", "create",
		"--org", "org_123",
		"--name", "Acme Okta",
		"--issuer-url", "https://issuer.example.com",
		"--client-id", "client-123",
		"--client-secret-env", "CCP_OKTA_SECRET",
		"--allowed-domains", "acme.com, contractors.acme.com",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if createBody["name"] != "Acme Okta" || createBody["issuer_url"] != "https://issuer.example.com" {
		t.Fatalf("unexpected identity-provider create body: %+v", createBody)
	}
	allowedDomains, ok := createBody["allowed_domains"].([]any)
	if !ok || len(allowedDomains) != 2 {
		t.Fatalf("expected allowed_domains to be encoded, got %+v", createBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{
		"identity-provider", "update",
		"--id", "idp_123",
		"--name", "Acme Workforce",
		"--allowed-domains", "acme.com,workforce.acme.com",
		"--enabled", "false",
		"--default-role", "org_admin",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from identity-provider update, got %d, stderr=%s", code, stderr.String())
	}
	if updateBody["name"] != "Acme Workforce" || updateBody["enabled"] != false {
		t.Fatalf("unexpected identity-provider update body: %+v", updateBody)
	}
	updatedDomains, ok := updateBody["allowed_domains"].([]any)
	if !ok || len(updatedDomains) != 2 {
		t.Fatalf("expected allowed_domains on update, got %+v", updateBody)
	}
	if updateBody["default_role"] != "org_admin" {
		t.Fatalf("expected default_role on update, got %+v", updateBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"identity-provider", "test", "--id", "idp_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from identity-provider test, got %d, stderr=%s", code, stderr.String())
	}
	if !testCalled {
		t.Fatal("expected identity-provider test route to be called")
	}
	if !strings.Contains(stdout.String(), `"status": "healthy"`) || !strings.Contains(stdout.String(), "issuer reachable") {
		t.Fatalf("expected identity-provider test output, got %s", stdout.String())
	}
}

func TestRunServiceAccountDeactivateAndTokenRotate(t *testing.T) {
	var deactivateCalled bool
	var rotateBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/service-accounts/svcacct_123/deactivate":
			deactivateCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"svcacct_123","organization_id":"org_123","name":"deployer","description":"","role":"org_member","status":"inactive"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/service-accounts/svcacct_123/tokens/token_123/rotate":
			if err := json.NewDecoder(r.Body).Decode(&rotateBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"token":"ccpt_rotated_secret","entry":{"id":"token_456","organization_id":"org_123","service_account_id":"svcacct_123","name":"rotated","token_prefix":"ccpt_abcd","status":"active"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"service-account", "deactivate", "--id", "svcacct_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from service-account deactivate, got %d, stderr=%s", code, stderr.String())
	}
	if !deactivateCalled {
		t.Fatal("expected service-account deactivate route to be called")
	}
	if !strings.Contains(stdout.String(), `"status": "inactive"`) {
		t.Fatalf("expected inactive status in deactivate output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"token", "rotate", "--service-account", "svcacct_123", "--id", "token_123", "--name", "rotated", "--expires-in-hours", "24"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from token rotate, got %d, stderr=%s", code, stderr.String())
	}
	if rotateBody["name"] != "rotated" {
		t.Fatalf("expected rotate body to contain name, got %+v", rotateBody)
	}
	if rotateBody["expires_in_hours"] != float64(24) {
		t.Fatalf("expected rotate body to contain expires_in_hours=24, got %+v", rotateBody)
	}
	if !strings.Contains(stdout.String(), `"token": "ccpt_rotated_secret"`) {
		t.Fatalf("expected rotated token output, got %s", stdout.String())
	}
}

func TestRunBrowserSessionListAndRevoke(t *testing.T) {
	var seenQuery string
	var seenOrgHeaders []string
	var revokeCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenOrgHeaders = append(seenOrgHeaders, r.Header.Get("X-CCP-Organization-ID"))
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/browser-sessions":
			seenQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"sess_123","user_id":"user_123","user_email":"owner@acme.local","user_display_name":"Owner","auth_method":"oidc","auth_provider_id":"idp_123","auth_provider":"Acme Okta","expires_at":"2026-04-20T12:00:00Z","status":"active","current":false}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/browser-sessions/sess_123/revoke":
			revokeCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"sess_123","user_id":"user_123","user_email":"owner@acme.local","user_display_name":"Owner","auth_method":"oidc","auth_provider_id":"idp_123","auth_provider":"Acme Okta","expires_at":"2026-04-20T12:00:00Z","revoked_at":"2026-04-20T11:30:00Z","status":"revoked","current":false}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"browser-session", "list", "--user", "user_123", "--status", "active", "--limit", "25", "--offset", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from browser-session list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(seenQuery, "user_id=user_123") || !strings.Contains(seenQuery, "status=active") || !strings.Contains(seenQuery, "limit=25") || !strings.Contains(seenQuery, "offset=5") {
		t.Fatalf("expected browser-session list to encode filters, got %q", seenQuery)
	}
	if !strings.Contains(stdout.String(), `"auth_provider": "Acme Okta"`) || !strings.Contains(stdout.String(), `"status": "active"`) {
		t.Fatalf("expected browser-session list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"browser-session", "revoke", "--id", "sess_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from browser-session revoke, got %d, stderr=%s", code, stderr.String())
	}
	if !revokeCalled {
		t.Fatal("expected browser-session revoke route to be called")
	}
	if !strings.Contains(stdout.String(), `"status": "revoked"`) {
		t.Fatalf("expected browser-session revoke output, got %s", stdout.String())
	}

	for _, header := range seenOrgHeaders {
		if header != "org_123" {
			t.Fatalf("expected browser-session commands to carry org scope, got headers %+v", seenOrgHeaders)
		}
	}
}

func TestRunIntegrationsWebhookSyncAndOutboxList(t *testing.T) {
	var webhookSyncCalled bool
	var outboxQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/integrations/int_123/webhook-registration/sync":
			webhookSyncCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"registration":{"id":"whr_123","integration_id":"int_123","provider_kind":"github","callback_url":"https://api.example.com/api/v1/integrations/int_123/webhooks/github","status":"registered","delivery_health":"healthy","auto_managed":true},"details":["status=registered"]}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/outbox-events":
			outboxQuery = r.URL.RawQuery
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"evt_123","event_type":"sync.completed","resource_type":"integration","resource_id":"int_123","status":"processed","attempts":1,"created_at":"2026-04-16T12:00:00Z"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"integrations", "webhook-sync", "--id", "int_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if !webhookSyncCalled || !strings.Contains(stdout.String(), "\"status\": \"registered\"") {
		t.Fatalf("expected webhook sync output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"outbox", "list", "--event-type", "sync.completed", "--status", "processed", "--limit", "25"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(outboxQuery, "event_type=sync.completed") || !strings.Contains(outboxQuery, "status=processed") {
		t.Fatalf("expected outbox filters in query, got %q", outboxQuery)
	}
	if !strings.Contains(stdout.String(), "sync.completed") {
		t.Fatalf("expected outbox output to contain event type, got %s", stdout.String())
	}
}

func TestRunChangeRiskAndRolloutReadCommands(t *testing.T) {
	var createChangeBody map[string]any
	var assessRiskBody map[string]any
	var seenHeaders []string
	var changeShowCalled bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHeaders = append(seenHeaders, r.Header.Get("X-CCP-Organization-ID"))
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/changes":
			if err := json.NewDecoder(r.Body).Decode(&createChangeBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"change_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","summary":"Checkout release","change_types":["code"],"file_count":5,"resource_count":1,"touches_infrastructure":true}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/risk-assessments":
			if err := json.NewDecoder(r.Body).Decode(&assessRiskBody); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"assessment":{"id":"risk_123","organization_id":"org_123","change_set_id":"change_123","risk_level":"high","recommended_rollout_strategy":"canary","score":82,"explanation":["schema touch raises risk"]},"policy_decisions":[{"id":"poldec_123","organization_id":"org_123","policy_id":"pol_123","policy_code":"prod-review","applies_to":"risk_assessment","mode":"require_manual_review","outcome":"require_manual_review","summary":"manual review required"}]}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/changes":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"change_123","organization_id":"org_123","summary":"Checkout release","change_types":["code"],"file_count":5}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/changes/change_123":
			changeShowCalled = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"change_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","summary":"Checkout release","change_types":["code"],"file_count":5,"resource_count":1}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/risk-assessments":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"risk_123","organization_id":"org_123","change_set_id":"change_123","level":"high","recommended_rollout_strategy":"canary","score":82}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/rollout-plans":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[{"id":"plan_123","organization_id":"org_123","change_set_id":"change_123","risk_assessment_id":"risk_123","strategy":"canary","status":"draft","approval_required":true}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{
		"change", "analyze",
		"--org", "org_123",
		"--project", "proj_123",
		"--service", "svc_123",
		"--env", "env_123",
		"--summary", "Checkout release",
		"--files", "5",
		"--resources", "1",
		"--type", "code",
		"--infra",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from change analyze, got %d, stderr=%s", code, stderr.String())
	}
	if createChangeBody["summary"] != "Checkout release" || createChangeBody["project_id"] != "proj_123" {
		t.Fatalf("unexpected change create body: %+v", createChangeBody)
	}
	changeTypes, ok := createChangeBody["change_types"].([]any)
	if !ok || len(changeTypes) != 1 || changeTypes[0] != "code" {
		t.Fatalf("expected change types in create body, got %+v", createChangeBody)
	}
	if assessRiskBody["change_set_id"] != "change_123" {
		t.Fatalf("expected risk assessment request to use created change id, got %+v", assessRiskBody)
	}
	if !strings.Contains(stdout.String(), `"recommended": "canary"`) || !strings.Contains(stdout.String(), `"outcome": "require_manual_review"`) {
		t.Fatalf("expected change analyze output to include recommendation and policies, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"change", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from change list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"id": "change_123"`) {
		t.Fatalf("expected change list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"change", "show", "--id", "change_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from change show, got %d, stderr=%s", code, stderr.String())
	}
	if !changeShowCalled || !strings.Contains(stdout.String(), `"resource_count": 1`) {
		t.Fatalf("expected change show output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"risk", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from risk list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"level": "high"`) {
		t.Fatalf("expected risk list output, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout-plan", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from rollout-plan list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"strategy": "canary"`) || !strings.Contains(stdout.String(), `"approval_required": true`) {
		t.Fatalf("expected rollout-plan list output, got %s", stdout.String())
	}

	for _, header := range seenHeaders {
		if header != "org_123" {
			t.Fatalf("expected all operational read calls to carry organization scope, got headers %+v", seenHeaders)
		}
	}
}

func TestRunRolloutRuntimeAndVerificationCommands(t *testing.T) {
	var createPlanBody map[string]any
	var createExecutionBody map[string]any
	var advanceBody map[string]any
	var verificationBody map[string]any
	var signalBody map[string]any
	var rolloutDetailCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/rollout-plans":
			if err := json.NewDecoder(r.Body).Decode(&createPlanBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"assessment":{"id":"risk_123","organization_id":"org_123","change_set_id":"change_123","risk_level":"high","recommended_rollout_strategy":"canary","score":82},"plan":{"id":"plan_123","organization_id":"org_123","change_set_id":"change_123","risk_assessment_id":"risk_123","strategy":"canary","status":"draft","approval_required":true},"policy_decisions":[]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/rollout-executions":
			if err := json.NewDecoder(r.Body).Decode(&createExecutionBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"rollout_123","organization_id":"org_123","rollout_plan_id":"plan_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","status":"pending_approval","current_step":"approve"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/rollout-executions":
			_, _ = w.Write([]byte(`{"data":[{"id":"rollout_123","organization_id":"org_123","rollout_plan_id":"plan_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","status":"pending_approval","current_step":"approve"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/rollout-executions/rollout_123":
			rolloutDetailCalls++
			_, _ = w.Write([]byte(`{"data":{"execution":{"id":"rollout_123","organization_id":"org_123","rollout_plan_id":"plan_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","status":"in_progress","current_step":"verify"},"verification_results":[],"signal_snapshots":[],"timeline":[],"status_timeline":[],"runtime_summary":{"latest_signal_health":"healthy","latest_signal_summary":"steady","latest_verification_outcome":"passed","recommended_action":"continue","action_disposition":"applied","summary":"steady"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/rollout-executions/rollout_123/evidence-pack":
			_, _ = w.Write([]byte(`{"data":{"summary":{"approval_state":"satisfied","risk_level":"high","risk_score":82,"blast_radius_scope":"service","blast_radius_summary":"checkout impact remains bounded to one service","rollout_strategy":"canary","latest_decision":"continue","latest_verification_outcome":"passed","repository_count":1,"discovered_resource_count":2,"blocking_policy_count":0,"manual_review_policy_count":1,"evidence_highlights":["Risk review scored this rollout high at 82."]},"organization":{"id":"org_123","name":"Acme","slug":"acme"},"project":{"id":"proj_123","organization_id":"org_123","name":"Platform","slug":"platform"},"service":{"id":"svc_123","organization_id":"org_123","project_id":"proj_123","team_id":"team_123","name":"Checkout","slug":"checkout"},"environment":{"id":"env_123","organization_id":"org_123","project_id":"proj_123","name":"Production","slug":"prod","type":"production","region":"us-central1","production":true},"change_set":{"id":"change_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","summary":"release bundle","change_types":["code"],"status":"open"},"assessment":{"id":"risk_123","organization_id":"org_123","project_id":"proj_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","score":82,"level":"high","blast_radius":{"scope":"service","summary":"checkout impact remains bounded to one service"}},"plan":{"id":"plan_123","organization_id":"org_123","project_id":"proj_123","change_set_id":"change_123","risk_assessment_id":"risk_123","strategy":"canary","approval_required":true,"approval_level":"policy-review"},"execution_detail":{"execution":{"id":"rollout_123","organization_id":"org_123","project_id":"proj_123","rollout_plan_id":"plan_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","status":"in_progress","current_step":"verify"},"verification_results":[],"signal_snapshots":[],"timeline":[],"status_timeline":[],"runtime_summary":{"latest_signal_health":"healthy","latest_signal_summary":"steady","latest_decision":"continue"}},"policy_decisions":[{"id":"decision_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","policy_id":"pol_123","policy_name":"Manual Review","policy_code":"manual-review","policy_scope":"environment","applies_to":"rollout_plan","mode":"require_manual_review","change_set_id":"change_123","risk_assessment_id":"risk_123","rollout_plan_id":"plan_123","outcome":"require_manual_review","summary":"manual review required"}],"repositories":[{"id":"repo_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","name":"checkout","provider":"github","url":"https://github.com/acme/checkout","default_branch":"main","status":"mapped"}],"discovered_resources":[{"id":"discovery_123","organization_id":"org_123","integration_id":"int_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","repository_id":"repo_123","resource_type":"kubernetes_workload","provider":"kubernetes","external_id":"checkout","name":"checkout","status":"mapped"}],"graph_relationships":[{"id":"rel_123","organization_id":"org_123","project_id":"proj_123","relationship_type":"service_repository","from_resource_type":"service","from_resource_id":"svc_123","to_resource_type":"repository","to_resource_id":"repo_123","status":"active","last_observed_at":"2026-04-19T12:00:00Z"}],"audit_trail":[{"id":"audit_123","organization_id":"org_123","project_id":"proj_123","actor_id":"user_123","actor_type":"user","actor":"owner@acme.local","action":"rollout.execution.created","resource_type":"rollout_execution","resource_id":"rollout_123","outcome":"success"}]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/rollout-executions/rollout_123/advance":
			if err := json.NewDecoder(r.Body).Decode(&advanceBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"rollout_123","organization_id":"org_123","rollout_plan_id":"plan_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","status":"in_progress","current_step":"start"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/rollout-executions/rollout_123/timeline":
			_, _ = w.Write([]byte(`{"data":[{"id":"status_123","organization_id":"org_123","resource_type":"rollout_execution","resource_id":"rollout_123","event_type":"rollout.execution.started","summary":"rollout started","source":"control_plane","created_at":"2026-04-19T12:00:00Z"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/rollout-executions/rollout_123/reconcile":
			_, _ = w.Write([]byte(`{"data":{"execution":{"id":"rollout_123","organization_id":"org_123","rollout_plan_id":"plan_123","change_set_id":"change_123","service_id":"svc_123","environment_id":"env_123","status":"in_progress","current_step":"verify"},"verification_results":[],"signal_snapshots":[],"timeline":[],"status_timeline":[],"runtime_summary":{"latest_signal_health":"healthy","latest_signal_summary":"steady","latest_verification_outcome":"passed","recommended_action":"continue","action_disposition":"applied","summary":"steady"}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/rollout-executions/rollout_123/verification":
			if err := json.NewDecoder(r.Body).Decode(&verificationBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"verify_123","organization_id":"org_123","rollout_execution_id":"rollout_123","outcome":"passed","decision":"continue","summary":"verification completed"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/rollout-executions/rollout_123/signal-snapshots":
			if err := json.NewDecoder(r.Body).Decode(&signalBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"signal_123","organization_id":"org_123","rollout_execution_id":"rollout_123","provider_type":"prometheus","health":"healthy","summary":"all clear","signals":[{"name":"latency_p95_ms","category":"technical","value":180,"unit":"ms","status":"healthy"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"rollout", "plan", "--change", "change_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from rollout plan, got %d, stderr=%s", code, stderr.String())
	}
	if createPlanBody["change_set_id"] != "change_123" || !strings.Contains(stdout.String(), `"strategy": "canary"`) {
		t.Fatalf("unexpected rollout plan behavior, body=%+v output=%s", createPlanBody, stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "execute", "--plan", "plan_123"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from rollout execute, got %d, stderr=%s", code, stderr.String())
	}
	if createExecutionBody["rollout_plan_id"] != "plan_123" {
		t.Fatalf("unexpected rollout execute body: %+v", createExecutionBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "list"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"id": "rollout_123"`) {
		t.Fatalf("expected rollout list output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "show", "--id", "rollout_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"latest_signal_health": "healthy"`) {
		t.Fatalf("expected rollout show output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "status", "--id", "rollout_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"recommended_action": "continue"`) {
		t.Fatalf("expected rollout status output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if rolloutDetailCalls != 2 {
		t.Fatalf("expected rollout detail route to be used by show and status, got %d calls", rolloutDetailCalls)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "evidence", "--id", "rollout_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"manual_review_policy_count": 1`) || !strings.Contains(stdout.String(), `"graph_relationships"`) {
		t.Fatalf("expected rollout evidence output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "advance", "--id", "rollout_123", "--action", "approve", "--reason", "approved for canary"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from rollout advance, got %d, stderr=%s", code, stderr.String())
	}
	if advanceBody["action"] != "approve" || advanceBody["reason"] != "approved for canary" {
		t.Fatalf("unexpected rollout advance body: %+v", advanceBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "timeline", "--id", "rollout_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"event_type": "rollout.execution.started"`) {
		t.Fatalf("expected rollout timeline output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollout", "reconcile", "--id", "rollout_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"latest_signal_summary": "steady"`) {
		t.Fatalf("expected rollout reconcile output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"signal", "ingest", "--rollout", "rollout_123", "--provider", "prometheus", "--health", "healthy", "--summary", "all clear", "--latency", "180", "--error-rate", "0.4"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from signal ingest, got %d, stderr=%s", code, stderr.String())
	}
	if signalBody["provider_type"] != "prometheus" || signalBody["health"] != "healthy" {
		t.Fatalf("unexpected signal ingest body: %+v", signalBody)
	}
	signals, ok := signalBody["signals"].([]any)
	if !ok || len(signals) != 2 {
		t.Fatalf("expected generated signals in ingest body, got %+v", signalBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"verification", "record", "--rollout", "rollout_123", "--outcome", "passed", "--decision", "continue", "--summary", "verification completed"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from verification record, got %d, stderr=%s", code, stderr.String())
	}
	if verificationBody["decision"] != "continue" || verificationBody["outcome"] != "passed" {
		t.Fatalf("unexpected verification body: %+v", verificationBody)
	}
	if !strings.Contains(stdout.String(), `"summary": "verification completed"`) {
		t.Fatalf("expected verification output, got %s", stdout.String())
	}
}

func TestRunRollbackPolicyAuditAndIntegrationOperationalCommands(t *testing.T) {
	var createRollbackPolicyBody map[string]any
	var updateRollbackPolicyBody map[string]any
	var createIntegrationBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/rollback-policies":
			_, _ = w.Write([]byte(`{"data":[{"id":"rbp_123","organization_id":"org_123","service_id":"svc_123","environment_id":"env_123","name":"Prod Strict","max_error_rate":1,"max_latency_ms":250,"rollback_on_critical_signals":true,"enabled":true}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/rollback-policies":
			if err := json.NewDecoder(r.Body).Decode(&createRollbackPolicyBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"rbp_123","organization_id":"org_123","service_id":"svc_123","environment_id":"env_123","name":"Prod Strict","max_error_rate":1,"max_latency_ms":250,"rollback_on_critical_signals":true,"enabled":true}}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/rollback-policies/rbp_123":
			if err := json.NewDecoder(r.Body).Decode(&updateRollbackPolicyBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"rbp_123","organization_id":"org_123","service_id":"svc_123","environment_id":"env_123","name":"Prod Tightened","max_error_rate":0.8,"max_latency_ms":220,"rollback_on_critical_signals":true,"enabled":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/audit-events":
			_, _ = w.Write([]byte(`{"data":[{"id":"audit_123","organization_id":"org_123","resource_type":"policy","resource_id":"pol_123","action":"policy.updated","actor":"owner@acme.local","created_at":"2026-04-19T12:00:00Z"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/integrations":
			if err := json.NewDecoder(r.Body).Decode(&createIntegrationBody); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"data":{"id":"int_123","organization_id":"org_123","name":"GitHub Production","kind":"github","instance_key":"github-prod","scope_type":"organization","scope_name":"Production","mode":"advisory","enabled":true}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/integrations":
			_, _ = w.Write([]byte(`{"data":[{"id":"int_123","organization_id":"org_123","name":"GitHub Production","kind":"github","instance_key":"github-prod","scope_type":"organization","scope_name":"Production","mode":"advisory","enabled":true}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/integrations/coverage":
			_, _ = w.Write([]byte(`{"data":{"enabled_integrations":1,"stale_integrations":0,"repositories":2,"unmapped_repositories":1,"discovered_resources":1,"unmapped_discovered_resources":1,"workload_coverage_environments":1,"signal_coverage_services":1}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/integrations/int_123/test":
			_, _ = w.Write([]byte(`{"data":{"integration":{"id":"int_123","organization_id":"org_123","name":"GitHub Production","kind":"github","instance_key":"github-prod","mode":"advisory"},"run":{"id":"run_test_123","integration_id":"int_123","operation":"test","trigger":"manual","status":"healthy","summary":"connection ok","started_at":"2026-04-19T12:05:00Z"}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/integrations/int_123/sync":
			_, _ = w.Write([]byte(`{"data":{"integration":{"id":"int_123","organization_id":"org_123","name":"GitHub Production","kind":"github","instance_key":"github-prod","mode":"advisory"},"run":{"id":"sync_123","integration_id":"int_123","operation":"sync","status":"succeeded","summary":"repositories discovered","started_at":"2026-04-19T12:06:00Z"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/integrations/int_123/sync-runs":
			_, _ = w.Write([]byte(`{"data":[{"id":"sync_123","integration_id":"int_123","operation":"sync","trigger":"manual","status":"succeeded","summary":"repositories discovered","started_at":"2026-04-19T12:05:00Z"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/integrations/int_123/github/onboarding/start":
			_, _ = w.Write([]byte(`{"data":{"integration":{"id":"int_123","organization_id":"org_123","name":"GitHub Production","kind":"github","instance_key":"github-prod","mode":"advisory"},"authorize_url":"https://github.com/apps/change-control/installations/new","state":"signed-state"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/integrations/int_123/webhook-registration":
			_, _ = w.Write([]byte(`{"data":{"registration":{"id":"whr_123","integration_id":"int_123","provider_kind":"github","callback_url":"https://api.example.com/api/v1/integrations/int_123/webhooks/github","status":"registered","delivery_health":"healthy","auto_managed":true},"details":["status=registered"]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"rollback-policy", "list"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"name": "Prod Strict"`) {
		t.Fatalf("expected rollback-policy list output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollback-policy", "create", "--org", "org_123", "--service", "svc_123", "--env", "env_123", "--name", "Prod Strict", "--max-error-rate", "1", "--max-latency-ms", "250"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from rollback-policy create, got %d, stderr=%s", code, stderr.String())
	}
	if createRollbackPolicyBody["service_id"] != "svc_123" || createRollbackPolicyBody["environment_id"] != "env_123" {
		t.Fatalf("unexpected rollback-policy create body: %+v", createRollbackPolicyBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"rollback-policy", "update", "--id", "rbp_123", "--name", "Prod Tightened", "--max-error-rate", "0.8", "--max-latency-ms", "220", "--enabled", "false"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from rollback-policy update, got %d, stderr=%s", code, stderr.String())
	}
	if updateRollbackPolicyBody["name"] != "Prod Tightened" || updateRollbackPolicyBody["enabled"] != false {
		t.Fatalf("unexpected rollback-policy update body: %+v", updateRollbackPolicyBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"audit", "list"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"action": "policy.updated"`) {
		t.Fatalf("expected audit list output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"integrations", "create", "--org", "org_123", "--kind", "github", "--name", "GitHub Production", "--instance-key", "github-prod", "--scope-type", "organization", "--scope-name", "Production", "--mode", "advisory", "--auth-strategy", "github_app"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from integrations create, got %d, stderr=%s", code, stderr.String())
	}
	if createIntegrationBody["kind"] != "github" || createIntegrationBody["instance_key"] != "github-prod" || createIntegrationBody["auth_strategy"] != "github_app" {
		t.Fatalf("unexpected integrations create body: %+v", createIntegrationBody)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"integrations", "show", "--id", "int_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"instance_key": "github-prod"`) {
		t.Fatalf("expected integrations show output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"integrations", "coverage"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"repositories": 2`) {
		t.Fatalf("expected integrations coverage output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"integrations", "test", "--id", "int_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"status": "healthy"`) || !strings.Contains(stdout.String(), `"operation": "test"`) {
		t.Fatalf("expected integrations test output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"integrations", "sync", "--id", "int_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"summary": "repositories discovered"`) || !strings.Contains(stdout.String(), `"operation": "sync"`) {
		t.Fatalf("expected integrations sync output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"integrations", "runs", "--id", "int_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"operation": "sync"`) {
		t.Fatalf("expected integrations runs output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"integrations", "github-start", "--id", "int_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"authorize_url": "https://github.com/apps/change-control/installations/new"`) {
		t.Fatalf("expected integrations github-start output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"integrations", "webhook-show", "--id", "int_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"status": "registered"`) || !strings.Contains(stdout.String(), `"delivery_health": "healthy"`) {
		t.Fatalf("expected integrations webhook-show output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
}

func TestRunPolicyDecisionAndScopedStatusHistoryCommands(t *testing.T) {
	var seenHeaders []string
	var policyDecisionQuery string
	var projectStatusQuery string
	var serviceStatusQuery string
	var environmentStatusQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenHeaders = append(seenHeaders, r.Header.Get("X-CCP-Organization-ID"))
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/policy-decisions":
			policyDecisionQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"data":[{"id":"poldec_123","organization_id":"org_123","project_id":"proj_123","policy_id":"pol_123","policy_name":"Prod Review","policy_code":"prod-review","policy_scope":"project","applies_to":"rollout_plan","mode":"require_manual_review","change_set_id":"change_123","risk_assessment_id":"risk_123","rollout_plan_id":"plan_123","rollout_execution_id":"rollout_123","outcome":"require_manual_review","summary":"manual review required","reasons":["production rollout needs approval"]}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/status-events/status_123":
			_, _ = w.Write([]byte(`{"data":{"id":"status_123","organization_id":"org_123","project_id":"proj_123","service_id":"svc_123","environment_id":"env_123","rollout_execution_id":"rollout_123","resource_type":"rollout_execution","resource_id":"rollout_123","event_type":"rollout.execution.paused","category":"rollout","severity":"warning","outcome":"paused","source":"control_plane","automated":false,"summary":"paused for review"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/projects/proj_123/status-events":
			projectStatusQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"data":[{"id":"status_project_123","organization_id":"org_123","project_id":"proj_123","resource_type":"project","resource_id":"proj_123","event_type":"policy.review.requested","category":"governance","severity":"warning","source":"control_plane","automated":false,"summary":"project-level review requested"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/services/svc_123/status-events":
			serviceStatusQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"data":[{"id":"status_service_123","organization_id":"org_123","service_id":"svc_123","resource_type":"service","resource_id":"svc_123","event_type":"rollout.execution.rollback_triggered","category":"runtime","severity":"critical","source":"prometheus","automated":true,"summary":"rollback triggered"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/environments/env_123/status-events":
			environmentStatusQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"data":[{"id":"status_env_123","organization_id":"org_123","environment_id":"env_123","resource_type":"environment","resource_id":"env_123","event_type":"integration.sync.completed","category":"integration","severity":"info","source":"gitlab","automated":true,"summary":"sync completed"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("CCP_API_BASE_URL", server.URL)
	t.Setenv("CCP_API_TOKEN", "token-123")
	t.Setenv("CCP_ORGANIZATION_ID", "org_123")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(context.Background(), []string{"policy-decision", "list", "--project", "proj_123", "--policy", "pol_123", "--risk", "risk_123", "--plan", "plan_123", "--rollout", "rollout_123", "--applies-to", "rollout_plan", "--limit", "25", "--offset", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0 from policy-decision list, got %d, stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"outcome": "require_manual_review"`) || !strings.Contains(stdout.String(), `"summary": "manual review required"`) {
		t.Fatalf("expected policy-decision output, got %s", stdout.String())
	}
	for _, expected := range []string{"project_id=proj_123", "policy_id=pol_123", "risk_assessment_id=risk_123", "rollout_plan_id=plan_123", "rollout_execution_id=rollout_123", "applies_to=rollout_plan", "limit=25", "offset=5"} {
		if !strings.Contains(policyDecisionQuery, expected) {
			t.Fatalf("expected policy-decision query to include %q, got %q", expected, policyDecisionQuery)
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"status", "show", "--id", "status_123"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"event_type": "rollout.execution.paused"`) || !strings.Contains(stdout.String(), `"summary": "paused for review"`) {
		t.Fatalf("expected status show output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"status", "project", "--id", "proj_123", "--rollback-only", "--limit", "20", "--offset", "40"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"resource_type": "project"`) {
		t.Fatalf("expected project status output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(projectStatusQuery, "rollback_only=true") || !strings.Contains(projectStatusQuery, "limit=20") || !strings.Contains(projectStatusQuery, "offset=40") {
		t.Fatalf("unexpected project status query: %q", projectStatusQuery)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"status", "service", "--id", "svc_123", "--rollback-only", "--limit", "10"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"resource_type": "service"`) || !strings.Contains(stdout.String(), `"source": "prometheus"`) {
		t.Fatalf("expected service status output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(serviceStatusQuery, "rollback_only=true") || !strings.Contains(serviceStatusQuery, "limit=10") {
		t.Fatalf("unexpected service status query: %q", serviceStatusQuery)
	}

	stdout.Reset()
	stderr.Reset()
	code = run(context.Background(), []string{"status", "env", "--id", "env_123", "--limit", "15"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), `"resource_type": "environment"`) || !strings.Contains(stdout.String(), `"source": "gitlab"`) {
		t.Fatalf("expected environment status output, got code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(environmentStatusQuery, "limit=15") {
		t.Fatalf("unexpected environment status query: %q", environmentStatusQuery)
	}

	for _, header := range seenHeaders {
		if header != "org_123" {
			t.Fatalf("expected organization scope header on policy-decision/status calls, got %+v", seenHeaders)
		}
	}
}

type typesUpdateIntegrationRequest struct {
	ScheduleEnabled         *bool `json:"schedule_enabled,omitempty"`
	ScheduleIntervalSeconds *int  `json:"schedule_interval_seconds,omitempty"`
	SyncStaleAfterSeconds   *int  `json:"sync_stale_after_seconds,omitempty"`
}
