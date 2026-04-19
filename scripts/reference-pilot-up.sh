#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.tmp/reference-pilot"
KUBECONFIG_PATH="${STATE_DIR}/kubeconfig"
ENV_FILE="${STATE_DIR}/reference-pilot.env"

PILOT_API_PORT="${CCP_REFERENCE_PILOT_API_PORT:-38080}"
PILOT_GITLAB_PORT="${CCP_REFERENCE_PILOT_GITLAB_PORT:-39480}"
PILOT_KUBE_PROXY_PORT="${CCP_REFERENCE_PILOT_KUBE_PROXY_PORT:-18091}"
PILOT_WORKLOAD_PORT="${CCP_REFERENCE_PILOT_WORKLOAD_PORT:-18092}"
PILOT_PROMETHEUS_PORT="${CCP_REFERENCE_PILOT_PROMETHEUS_PORT:-19090}"
PILOT_K3S_API_PORT="${CCP_REFERENCE_PILOT_K3S_API_PORT:-16443}"

PILOT_GITLAB_TOKEN="${CCP_REFERENCE_PILOT_GITLAB_TOKEN:-reference-pilot-token}"
PILOT_GITLAB_WEBHOOK_SECRET="${CCP_REFERENCE_PILOT_GITLAB_WEBHOOK_SECRET:-reference-pilot-webhook}"

DB_DSN="postgres://postgres:postgres@localhost:25432/change_control_plane?sslmode=disable"
REDIS_ADDR="localhost:26379"
NATS_URL="nats://localhost:24222"

mkdir -p "${STATE_DIR}"

require_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "required binary not found: $1" >&2
    exit 1
  }
}

wait_for_url() {
  local url="$1"
  local attempts="${2:-60}"
  local delay="${3:-1}"
  for ((i=0; i<attempts; i++)); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep "${delay}"
  done
  echo "timed out waiting for ${url}" >&2
  return 1
}

is_pid_running() {
  local pid_file="$1"
  if [[ ! -f "${pid_file}" ]]; then
    return 1
  fi
  local pid
  pid="$(cat "${pid_file}")"
  [[ -n "${pid}" ]] && kill -0 "${pid}" >/dev/null 2>&1
}

start_background() {
  local name="$1"
  local health_url="$2"
  local command="$3"
  local pid_file="${STATE_DIR}/${name}.pid"
  local log_file="${STATE_DIR}/${name}.log"

  if is_pid_running "${pid_file}" && [[ -n "${health_url}" ]] && curl -fsS "${health_url}" >/dev/null 2>&1; then
    return 0
  fi
  if is_pid_running "${pid_file}"; then
    kill "$(cat "${pid_file}")" >/dev/null 2>&1 || true
    rm -f "${pid_file}"
  fi

  nohup /bin/zsh -lc "${command}" >"${log_file}" 2>&1 &
  echo $! >"${pid_file}"
  if [[ -n "${health_url}" ]]; then
    wait_for_url "${health_url}" 90 1
  fi
}

ensure_k3s_cluster() {
  if ! docker ps --format '{{.Names}}' | grep -qx 'ccp-reference-pilot-k3s'; then
    docker rm -f ccp-reference-pilot-k3s >/dev/null 2>&1 || true
    docker run -d --privileged --name ccp-reference-pilot-k3s -p "${PILOT_K3S_API_PORT}:6443" rancher/k3s:v1.30.0-k3s1 server --disable traefik --write-kubeconfig-mode 644 >/dev/null
  fi

  for ((i=0; i<90; i++)); do
    if docker exec ccp-reference-pilot-k3s kubectl get nodes >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done

  docker exec ccp-reference-pilot-k3s cat /etc/rancher/k3s/k3s.yaml | sed "s#127.0.0.1:6443#127.0.0.1:${PILOT_K3S_API_PORT}#g" >"${KUBECONFIG_PATH}"
}

load_workload_image() {
  docker build -f "${ROOT_DIR}/deploy/reference-pilot/Dockerfile.workload" -t ccp-reference-pilot-workload:local "${ROOT_DIR}" >/dev/null
  docker save ccp-reference-pilot-workload:local | docker exec -i ccp-reference-pilot-k3s ctr -n k8s.io images import - >/dev/null
}

apply_reference_manifests() {
  KUBECONFIG="${KUBECONFIG_PATH}" kubectl apply -f "${ROOT_DIR}/deploy/reference-pilot/k8s/reference-pilot.yaml" >/dev/null
  KUBECONFIG="${KUBECONFIG_PATH}" kubectl -n ccp-pilot rollout status deployment/checkout --timeout=180s >/dev/null
  KUBECONFIG="${KUBECONFIG_PATH}" kubectl -n ccp-pilot rollout status deployment/reference-pilot-prometheus --timeout=180s >/dev/null
}

write_env_file() {
  cat >"${ENV_FILE}" <<EOF
export CCP_REFERENCE_PILOT_STATE_DIR="${STATE_DIR}"
export CCP_REFERENCE_PILOT_KUBECONFIG="${KUBECONFIG_PATH}"
export CCP_REFERENCE_PILOT_API_BASE_URL="http://127.0.0.1:${PILOT_API_PORT}"
export CCP_REFERENCE_PILOT_GITLAB_BASE_URL="http://127.0.0.1:${PILOT_GITLAB_PORT}/api/v4"
export CCP_REFERENCE_PILOT_KUBE_API_BASE_URL="http://127.0.0.1:${PILOT_KUBE_PROXY_PORT}"
export CCP_REFERENCE_PILOT_WORKLOAD_ADMIN_URL="http://127.0.0.1:${PILOT_WORKLOAD_PORT}/admin/state"
export CCP_REFERENCE_PILOT_PROMETHEUS_BASE_URL="http://127.0.0.1:${PILOT_PROMETHEUS_PORT}"
export CCP_REFERENCE_PILOT_GITLAB_TOKEN="${PILOT_GITLAB_TOKEN}"
export CCP_REFERENCE_PILOT_GITLAB_WEBHOOK_SECRET="${PILOT_GITLAB_WEBHOOK_SECRET}"
export CCP_REFERENCE_PILOT_GITLAB_TOKEN_ENV="CCP_REFERENCE_PILOT_GITLAB_TOKEN"
export CCP_REFERENCE_PILOT_GITLAB_WEBHOOK_SECRET_ENV="CCP_REFERENCE_PILOT_GITLAB_WEBHOOK_SECRET"
EOF
}

require_bin docker
require_bin kubectl
require_bin curl
require_bin go

docker compose -f "${ROOT_DIR}/deploy/reference-pilot/docker-compose.yml" up -d --wait postgres redis nats
ensure_k3s_cluster
load_workload_image
apply_reference_manifests

start_background \
  "reference-pilot-gitlab" \
  "http://127.0.0.1:${PILOT_GITLAB_PORT}/readyz" \
  "cd '${ROOT_DIR}' && PORT='${PILOT_GITLAB_PORT}' REFERENCE_PILOT_GITLAB_TOKEN='${PILOT_GITLAB_TOKEN}' go run ./cmd/reference-pilot-gitlab"

start_background \
  "reference-pilot-api" \
  "http://127.0.0.1:${PILOT_API_PORT}/readyz" \
  "cd '${ROOT_DIR}' && CCP_DB_DSN='${DB_DSN}' CCP_REDIS_ADDR='${REDIS_ADDR}' CCP_NATS_URL='${NATS_URL}' CCP_API_PORT='${PILOT_API_PORT}' CCP_API_BASE_URL='http://127.0.0.1:${PILOT_API_PORT}' CCP_ALLOWED_ORIGINS='http://127.0.0.1:5173,http://localhost:5173,http://127.0.0.1:4173,http://localhost:4173' CCP_REFERENCE_PILOT_GITLAB_TOKEN='${PILOT_GITLAB_TOKEN}' CCP_REFERENCE_PILOT_GITLAB_WEBHOOK_SECRET='${PILOT_GITLAB_WEBHOOK_SECRET}' go run ./cmd/api"

start_background \
  "reference-pilot-kube-proxy" \
  "http://127.0.0.1:${PILOT_KUBE_PROXY_PORT}/api/" \
  "KUBECONFIG='${KUBECONFIG_PATH}' kubectl proxy --port='${PILOT_KUBE_PROXY_PORT}' --accept-hosts='.*'"

start_background \
  "reference-pilot-workload-forward" \
  "http://127.0.0.1:${PILOT_WORKLOAD_PORT}/healthz" \
  "KUBECONFIG='${KUBECONFIG_PATH}' kubectl -n ccp-pilot port-forward svc/checkout '${PILOT_WORKLOAD_PORT}:8080'"

start_background \
  "reference-pilot-prometheus-forward" \
  "http://127.0.0.1:${PILOT_PROMETHEUS_PORT}/-/ready" \
  "KUBECONFIG='${KUBECONFIG_PATH}' kubectl -n ccp-pilot port-forward svc/reference-pilot-prometheus '${PILOT_PROMETHEUS_PORT}:9090'"

write_env_file

cat <<EOF
Reference pilot environment is ready.

Source this file before validation commands:
  source "${ENV_FILE}"

Primary endpoints:
  Control Plane API: http://127.0.0.1:${PILOT_API_PORT}
  GitLab Fixture API: http://127.0.0.1:${PILOT_GITLAB_PORT}/api/v4
  Kubernetes API Proxy: http://127.0.0.1:${PILOT_KUBE_PROXY_PORT}
  Workload Admin: http://127.0.0.1:${PILOT_WORKLOAD_PORT}/admin/state
  Prometheus API: http://127.0.0.1:${PILOT_PROMETHEUS_PORT}

Next step:
  "${ROOT_DIR}/scripts/reference-pilot-verify.sh"
EOF
