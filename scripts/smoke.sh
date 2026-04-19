#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
STAMP="$(date +%s)"
EMAIL="smoke-${STAMP}@acme.local"
ORG_SLUG="smoke-${STAMP}"

extract_token() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["token"])'
}

extract_org_id() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["session"]["active_organization_id"])'
}

extract_id() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["id"])'
}

extract_plan_id() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["plan"]["id"])'
}

extract_latest_decision() {
  python3 -c 'import json,sys; data=json.load(sys.stdin)["data"]["verification_results"]; print(data[-1]["decision"] if data else "")'
}

extract_latest_automated() {
  python3 -c 'import json,sys; data=json.load(sys.stdin)["data"]["verification_results"]; print(str(data[-1].get("automated", False)).lower() if data else "false")'
}

extract_execution_status() {
  python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["execution"]["status"])'
}

extract_len() {
  python3 -c 'import json,sys; print(len(json.load(sys.stdin)["data"]))'
}

echo "Smoke check against ${BASE_URL}"
curl -fsS "${BASE_URL}/healthz" >/dev/null
curl -fsS "${BASE_URL}/readyz" >/dev/null

LOGIN_RESPONSE="$(curl -fsS \
  -H 'Content-Type: application/json' \
  -d "$(printf '{"email":"%s","display_name":"Smoke Admin","organization_name":"Smoke %s","organization_slug":"%s"}' "$EMAIL" "$STAMP" "$ORG_SLUG")" \
  "${BASE_URL}/api/v1/auth/dev/login")"
TOKEN="$(printf '%s' "$LOGIN_RESPONSE" | extract_token)"
ORG_ID="$(printf '%s' "$LOGIN_RESPONSE" | extract_org_id)"

api_post() {
  local path="$1"
  local payload="$2"
  curl -fsS \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${TOKEN}" \
    -H "X-CCP-Organization-ID: ${ORG_ID}" \
    -d "${payload}" \
    "${BASE_URL}${path}"
}

api_get() {
  local path="$1"
  local token="${2:-$TOKEN}"
  curl -fsS \
    -H "Authorization: Bearer ${token}" \
    -H "X-CCP-Organization-ID: ${ORG_ID}" \
    "${BASE_URL}${path}"
}

machine_post() {
  local path="$1"
  local payload="$2"
  curl -fsS \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${MACHINE_TOKEN}" \
    -H "X-CCP-Organization-ID: ${ORG_ID}" \
    -d "${payload}" \
    "${BASE_URL}${path}"
}

PROJECT_ID="$(api_post /api/v1/projects "$(printf '{"organization_id":"%s","name":"Smoke Platform","slug":"smoke-platform-%s"}' "$ORG_ID" "$STAMP")" | extract_id)"
TEAM_ID="$(api_post /api/v1/teams "$(printf '{"organization_id":"%s","project_id":"%s","name":"Smoke Team","slug":"smoke-team-%s"}' "$ORG_ID" "$PROJECT_ID" "$STAMP")" | extract_id)"
SERVICE_ID="$(api_post /api/v1/services "$(printf '{"organization_id":"%s","project_id":"%s","team_id":"%s","name":"Smoke Service","slug":"smoke-service-%s","criticality":"low","has_slo":true,"has_observability":true}' "$ORG_ID" "$PROJECT_ID" "$TEAM_ID" "$STAMP")" | extract_id)"
ENVIRONMENT_ID="$(api_post /api/v1/environments "$(printf '{"organization_id":"%s","project_id":"%s","name":"Smoke Staging","slug":"smoke-staging-%s","type":"staging","region":"us-central1"}' "$ORG_ID" "$PROJECT_ID" "$STAMP")" | extract_id)"
CHANGE_ID="$(api_post /api/v1/changes "$(printf '{"organization_id":"%s","project_id":"%s","service_id":"%s","environment_id":"%s","summary":"smoke rollout","change_types":["code"],"file_count":2}' "$ORG_ID" "$PROJECT_ID" "$SERVICE_ID" "$ENVIRONMENT_ID")" | extract_id)"
SERVICE_ACCOUNT_ID="$(api_post /api/v1/service-accounts "$(printf '{"organization_id":"%s","name":"smoke-worker-%s","role":"org_member"}' "$ORG_ID" "$STAMP")" | extract_id)"
MACHINE_RESPONSE="$(api_post "/api/v1/service-accounts/${SERVICE_ACCOUNT_ID}/tokens" '{"name":"smoke-token"}')"
MACHINE_TOKEN="$(printf '%s' "$MACHINE_RESPONSE" | extract_token)"

api_post /api/v1/risk-assessments "$(printf '{"change_set_id":"%s"}' "$CHANGE_ID")" >/dev/null
PLAN_ID="$(api_post /api/v1/rollout-plans "$(printf '{"change_set_id":"%s"}' "$CHANGE_ID")" | extract_plan_id)"
EXECUTION_ID="$(api_post /api/v1/rollout-executions "$(printf '{"rollout_plan_id":"%s","backend_type":"simulated","signal_provider_type":"simulated"}' "$PLAN_ID")" | extract_id)"
machine_post "/api/v1/rollout-executions/${EXECUTION_ID}/advance" '{"action":"start","reason":"smoke start"}' >/dev/null
machine_post "/api/v1/rollout-executions/${EXECUTION_ID}/reconcile" '{}' >/dev/null
machine_post "/api/v1/rollout-executions/${EXECUTION_ID}/signal-snapshots" '{"provider_type":"simulated","health":"healthy","summary":"smoke verification healthy","signals":[{"name":"latency_p95_ms","category":"technical","value":145,"unit":"ms","status":"healthy","threshold":250,"comparator":">"},{"name":"error_rate","category":"technical","value":0.2,"unit":"%","status":"healthy","threshold":1,"comparator":">"}]}' >/dev/null
RECONCILE_DETAIL="$(machine_post "/api/v1/rollout-executions/${EXECUTION_ID}/reconcile" '{}')"
LATEST_DECISION="$(printf '%s' "$RECONCILE_DETAIL" | extract_latest_decision)"
LATEST_AUTOMATED="$(printf '%s' "$RECONCILE_DETAIL" | extract_latest_automated)"
if [ "$LATEST_DECISION" != "verified" ] || [ "$LATEST_AUTOMATED" != "true" ]; then
  echo "Expected automated verified decision but got decision=${LATEST_DECISION} automated=${LATEST_AUTOMATED}"
  exit 1
fi

api_get /api/v1/services "$MACHINE_TOKEN" >/dev/null

ROLLBACK_EXECUTION_ID="$(api_post /api/v1/rollout-executions "$(printf '{"rollout_plan_id":"%s","backend_type":"simulated","signal_provider_type":"simulated"}' "$PLAN_ID")" | extract_id)"
machine_post "/api/v1/rollout-executions/${ROLLBACK_EXECUTION_ID}/advance" '{"action":"start","reason":"smoke rollback start"}' >/dev/null
machine_post "/api/v1/rollout-executions/${ROLLBACK_EXECUTION_ID}/reconcile" '{}' >/dev/null
machine_post "/api/v1/rollout-executions/${ROLLBACK_EXECUTION_ID}/signal-snapshots" '{"provider_type":"simulated","health":"critical","summary":"smoke verification failure","signals":[{"name":"error_rate","category":"technical","value":4.8,"unit":"%","status":"critical","threshold":1,"comparator":">"},{"name":"latency_p95_ms","category":"technical","value":710,"unit":"ms","status":"critical","threshold":250,"comparator":">"}]}' >/dev/null
ROLLBACK_DETAIL="$(machine_post "/api/v1/rollout-executions/${ROLLBACK_EXECUTION_ID}/reconcile" '{}')"
ROLLBACK_DECISION="$(printf '%s' "$ROLLBACK_DETAIL" | extract_latest_decision)"
ROLLBACK_STATUS="$(printf '%s' "$ROLLBACK_DETAIL" | extract_execution_status)"
if [ "$ROLLBACK_DECISION" != "rollback" ] || [ "$ROLLBACK_STATUS" != "rolled_back" ]; then
  echo "Expected automated rollback but got decision=${ROLLBACK_DECISION} status=${ROLLBACK_STATUS}"
  exit 1
fi

STATUS_COUNT="$(api_get "/api/v1/status-events?rollout_execution_id=${ROLLBACK_EXECUTION_ID}&rollback_only=true" | extract_len)"
if [ "$STATUS_COUNT" -lt 1 ]; then
  echo "Expected rollback-related status events but found ${STATUS_COUNT}"
  exit 1
fi

AUDIT_COUNT="$(api_get /api/v1/audit-events | extract_len)"
echo "Smoke flow complete: organization=${ORG_ID} project=${PROJECT_ID} service=${SERVICE_ID} verified_execution=${EXECUTION_ID} rollback_execution=${ROLLBACK_EXECUTION_ID} status_events=${STATUS_COUNT} audit_events=${AUDIT_COUNT}"
