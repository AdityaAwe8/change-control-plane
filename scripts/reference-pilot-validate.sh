#!/usr/bin/env bash
set -euo pipefail

REPORT_PATH="${REPORT_PATH:-.tmp/reference-pilot/reference-pilot-report.json}"

go run ./cmd/reference-pilot-verify --validate-report "$REPORT_PATH"
