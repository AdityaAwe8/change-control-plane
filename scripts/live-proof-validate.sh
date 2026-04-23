#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=/dev/null
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/scripts/load-local-env.sh"

REPORT_PATH="${REPORT_PATH:-.tmp/live-proof/live-proof-report.json}"

go run ./cmd/live-proof-verify --validate-report "$REPORT_PATH"
