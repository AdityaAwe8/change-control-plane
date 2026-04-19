#!/usr/bin/env bash
set -euo pipefail

REPORT_PATH="${REPORT_PATH:-.tmp/live-proof/live-proof-report.json}"

go run ./cmd/live-proof-verify --validate-report "$REPORT_PATH"
