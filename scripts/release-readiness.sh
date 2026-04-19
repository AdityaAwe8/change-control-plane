#!/usr/bin/env bash
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPORT_DIR="${CCP_RELEASE_REPORT_DIR:-$ROOT_DIR/.tmp/release-readiness}"
LOG_DIR="$REPORT_DIR/logs"
REPORT_PATH="${CCP_RELEASE_REPORT_PATH:-$REPORT_DIR/release-readiness-report.md}"
REFERENCE_PILOT_REPORT="${CCP_REFERENCE_PILOT_REPORT_PATH:-$ROOT_DIR/.tmp/reference-pilot/reference-pilot-report.json}"
LIVE_PROOF_REPORT="${CCP_LIVE_PROOF_REPORT_PATH:-$ROOT_DIR/.tmp/live-proof/live-proof-report.json}"
ALLOW_PROOF_GAPS_RAW="${CCP_RELEASE_ALLOW_PROOF_GAPS:-false}"

mkdir -p "$LOG_DIR"
find "$LOG_DIR" -type f -name '*.log' -delete 2>/dev/null || true
cd "$ROOT_DIR"

shopt -s nocasematch
case "$ALLOW_PROOF_GAPS_RAW" in
	true|1|yes|y)
		ALLOW_PROOF_GAPS=true
		;;
	*)
		ALLOW_PROOF_GAPS=false
		;;
esac
shopt -u nocasematch

timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
results=()
blockers=()
warnings=()

slugify() {
	printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//'
}

record_result() {
	results+=("$1|$2|$3|$4")
}

record_blocker() {
	blockers+=("$2: $3")
	record_result "FAIL" "$1" "$2" "$3"
}

record_warning() {
	warnings+=("$2: $3")
	record_result "WARN" "$1" "$2" "$3"
}

record_success() {
	record_result "PASS" "$1" "$2" "$3"
}

record_gap() {
	local category="$1"
	local label="$2"
	local detail="$3"
	if [[ "$ALLOW_PROOF_GAPS" == "true" ]]; then
		record_warning "$category" "$label" "$detail"
	else
		record_blocker "$category" "$label" "$detail"
	fi
}

render_report() {
	{
		printf '# Release Readiness Report\n\n'
		printf -- '- Generated at `%s`\n' "$timestamp"
		printf -- '- Overall status: `%s`\n' "$overall_status"
		printf -- '- Proof-gap override: `%s`\n' "$ALLOW_PROOF_GAPS"
		printf -- '- Reference pilot artifact: `%s`\n' "${REFERENCE_PILOT_REPORT#$ROOT_DIR/}"
		printf -- '- Live proof artifact: `%s`\n\n' "${LIVE_PROOF_REPORT#$ROOT_DIR/}"
		printf '## Checks\n\n'
		printf '| Status | Category | Check | Detail |\n'
		printf '| --- | --- | --- | --- |\n'
		for entry in "${results[@]}"; do
			IFS='|' read -r status category label detail <<<"$entry"
			printf '| %s | %s | %s | %s |\n' "$status" "$category" "$label" "$detail"
		done
		printf '\n## Remaining Blockers\n\n'
		if [[ ${#blockers[@]} -eq 0 ]]; then
			printf -- '- none\n'
		else
			for item in "${blockers[@]}"; do
				printf -- '- %s\n' "$item"
			done
		fi
		printf '\n## Warnings\n\n'
		if [[ ${#warnings[@]} -eq 0 ]]; then
			printf -- '- none\n'
		else
			for item in "${warnings[@]}"; do
				printf -- '- %s\n' "$item"
			done
		fi
		printf '\n## Evidence Classification\n\n'
		printf -- '- `local`: tests and build checks rerun directly from this repository.\n'
		printf -- '- `harness`: contract and provider-harness proof rerun locally against repo-managed fixtures.\n'
		printf -- '- `artifact`: saved proof bundles revalidated without rerunning their source environments.\n'
		printf -- '- `operator-proof`: preserved external proof still required for truthful hosted/customer deployment claims.\n'
		printf '\n## Notes\n\n'
		printf -- '- This gate does not itself execute real hosted/customer environments; it validates preserved proof artifacts and local/harness evidence.\n'
		printf -- '- The gate now also scans the generated release report, its supporting logs, and any preserved proof artifacts for configured secret-backed environment values without printing those secret values back out.\n'
		printf -- '- Set `CCP_RELEASE_ALLOW_PROOF_GAPS=true` to dry-run the gate before reference-pilot or external proof bundles have been captured.\n'
		printf -- '- Dedicated browser interaction proof still lives in the existing Playwright/CI path; this gate reruns web typecheck/build and the highest-value backend/contract/harness checks.\n'
	} >"$REPORT_PATH"
}

run_check() {
	local category="$1"
	local label="$2"
	shift 2
	local slug
	slug="$(slugify "$label")"
	local log_path="$LOG_DIR/${slug}.log"

	printf 'Running %s\n' "$label" >&2
	if "$@" >"$log_path" 2>&1; then
		record_success "$category" "$label" "command succeeded; see $(basename "$log_path")"
	else
		local exit_code=$?
		record_blocker "$category" "$label" "command failed with exit ${exit_code}; see $(basename "$log_path")"
	fi
}

validate_saved_report() {
	local category="$1"
	local label="$2"
	local report_path="$3"
	local slug="$4"
	shift 4
	local log_path="$LOG_DIR/${slug}.log"

	if [[ ! -f "$report_path" ]]; then
		record_gap "$category" "$label" "missing saved proof artifact at ${report_path#$ROOT_DIR/}"
		return
	fi

	printf 'Validating %s\n' "$label" >&2
	if REPORT_PATH="$report_path" "$@" >"$log_path" 2>&1; then
		record_success "$category" "$label" "validated saved artifact at ${report_path#$ROOT_DIR/}; see $(basename "$log_path")"
	else
		local exit_code=$?
		record_blocker "$category" "$label" "artifact validation failed with exit ${exit_code}; see $(basename "$log_path")"
	fi
}

run_check "local" "Go command tests" go test ./cmd/...
run_check "local" "App HTTP and workflow tests" go test ./internal/app/...
run_check "local" "Storage tests" go test ./internal/storage/...
run_check "local" "Integration tests" go test ./internal/integrations/...
run_check "local" "Event bus tests" go test ./internal/events/...
run_check "local" "Web typecheck" make web-typecheck
run_check "local" "Web build" make web-build

run_check "harness" "OpenAPI contract proof" make proof-contract
run_check "harness" "Provider harness proof" make proof-harness

validate_saved_report "artifact" "Reference pilot proof artifact" "$REFERENCE_PILOT_REPORT" "reference-pilot-validate" ./scripts/reference-pilot-validate.sh
live_proof_present=false
if [[ -f "$LIVE_PROOF_REPORT" ]]; then
	live_proof_present=true
fi
validate_saved_report "artifact" "External live proof artifact" "$LIVE_PROOF_REPORT" "live-proof-validate" ./scripts/live-proof-validate.sh

if [[ "$live_proof_present" == "true" ]]; then
	if grep -Eq '"environment_class"[[:space:]]*:[[:space:]]*"(customer_environment|hosted_saas)"' "$LIVE_PROOF_REPORT"; then
		record_success "operator-proof" "Hosted or customer proof classification" "live proof artifact preserves operator-run environment_class at ${LIVE_PROOF_REPORT#$ROOT_DIR/}"
	elif grep -Eq '"environment_class"[[:space:]]*:[[:space:]]*"hosted_like"' "$LIVE_PROOF_REPORT"; then
		record_gap "operator-proof" "Hosted or customer proof classification" "saved live proof report is only hosted_like; capture customer_environment or hosted_saas evidence to close release readiness honestly"
	else
		record_gap "operator-proof" "Hosted or customer proof classification" "live proof artifact is present but missing a recognized environment_class classification"
	fi
fi

overall_status="READY"
if [[ ${#blockers[@]} -gt 0 ]]; then
	overall_status="BLOCKED"
elif [[ ${#warnings[@]} -gt 0 ]]; then
	overall_status="READY_WITH_WARNINGS"
fi

render_report

artifact_safety_log="$LOG_DIR/artifact-secret-safety.log"
artifact_safety_args=(./cmd/artifact-safety-check --path "$REPORT_PATH" --path "$LOG_DIR")
if [[ -f "$REFERENCE_PILOT_REPORT" ]]; then
	artifact_safety_args+=(--path "$REFERENCE_PILOT_REPORT")
fi
if [[ -f "$LIVE_PROOF_REPORT" ]]; then
	artifact_safety_args+=(--path "$LIVE_PROOF_REPORT")
fi
printf 'Scanning release artifacts for secret leakage\n' >&2
if go run "${artifact_safety_args[@]}" >"$artifact_safety_log" 2>&1; then
	record_success "security" "Artifact secret-safety scan" "release report, logs, and preserved proof artifacts scanned without leaking configured secret-backed env values; see $(basename "$artifact_safety_log")"
else
	exit_code=$?
	record_blocker "security" "Artifact secret-safety scan" "artifact scan failed with exit ${exit_code}; see $(basename "$artifact_safety_log")"
fi

overall_status="READY"
if [[ ${#blockers[@]} -gt 0 ]]; then
	overall_status="BLOCKED"
elif [[ ${#warnings[@]} -gt 0 ]]; then
	overall_status="READY_WITH_WARNINGS"
fi

render_report

printf 'Wrote release-readiness report to %s\n' "$REPORT_PATH" >&2

if [[ ${#blockers[@]} -gt 0 ]]; then
	exit 1
fi
exit 0
