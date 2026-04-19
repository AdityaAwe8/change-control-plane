#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.tmp/reference-pilot"
ENV_FILE="${STATE_DIR}/reference-pilot.env"
REPORT_PATH="${STATE_DIR}/reference-pilot-report.json"

if [[ ! -f "${ENV_FILE}" ]]; then
  "${ROOT_DIR}/scripts/reference-pilot-up.sh"
fi

source "${ENV_FILE}"

go run ./cmd/reference-pilot-verify --report "${REPORT_PATH}"

echo
echo "Reference pilot report written to ${REPORT_PATH}"
