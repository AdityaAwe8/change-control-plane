#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.tmp/reference-pilot"

stop_pid_file() {
  local pid_file="$1"
  if [[ -f "${pid_file}" ]]; then
    local pid
    pid="$(cat "${pid_file}")"
    if [[ -n "${pid}" ]]; then
      kill "${pid}" >/dev/null 2>&1 || true
    fi
    rm -f "${pid_file}"
  fi
}

stop_pid_file "${STATE_DIR}/reference-pilot-prometheus-forward.pid"
stop_pid_file "${STATE_DIR}/reference-pilot-workload-forward.pid"
stop_pid_file "${STATE_DIR}/reference-pilot-kube-proxy.pid"
stop_pid_file "${STATE_DIR}/reference-pilot-api.pid"
stop_pid_file "${STATE_DIR}/reference-pilot-gitlab.pid"

docker rm -f ccp-reference-pilot-k3s >/dev/null 2>&1 || true
docker compose -f "${ROOT_DIR}/deploy/reference-pilot/docker-compose.yml" down -v >/dev/null 2>&1 || true

echo "Reference pilot environment stopped."
