#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.tmp/live-proof"
REPORT_PATH="${STATE_DIR}/live-proof-report.json"

mkdir -p "${STATE_DIR}"

go run ./cmd/live-proof-verify --report "${REPORT_PATH}" "$@"

echo
echo "Live proof report written to ${REPORT_PATH}"
