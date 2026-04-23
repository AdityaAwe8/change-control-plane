#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.tmp/live-proof"
PREFLIGHT_REPORT_PATH="${STATE_DIR}/live-proof-preflight.json"
CHECKLIST_PATH="${STATE_DIR}/live-proof-operator-checklist.md"

# shellcheck source=/dev/null
source "${ROOT_DIR}/scripts/load-local-env.sh"

mkdir -p "${STATE_DIR}"

go run ./cmd/live-proof-verify \
  --preflight-only \
  --preflight-report "${PREFLIGHT_REPORT_PATH}" \
  --operator-checklist "${CHECKLIST_PATH}" \
  "$@"

echo
echo "Live proof preflight report written to ${PREFLIGHT_REPORT_PATH}"
echo "Live proof operator checklist written to ${CHECKLIST_PATH}"
