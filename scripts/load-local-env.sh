#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOCAL_ENV_PRESERVED_CCP_NAMES=()
LOCAL_ENV_PRESERVED_CCP_VALUES=()

preserve_existing_ccp_vars() {
	local name
	LOCAL_ENV_PRESERVED_CCP_NAMES=()
	LOCAL_ENV_PRESERVED_CCP_VALUES=()
	while IFS= read -r name; do
		LOCAL_ENV_PRESERVED_CCP_NAMES+=("$name")
		LOCAL_ENV_PRESERVED_CCP_VALUES+=("${!name}")
	done < <(compgen -v | grep '^CCP_' || true)
}

restore_existing_ccp_vars() {
	local index
	local name
	for index in "${!LOCAL_ENV_PRESERVED_CCP_NAMES[@]}"; do
		name="${LOCAL_ENV_PRESERVED_CCP_NAMES[$index]}"
		printf -v "$name" '%s' "${LOCAL_ENV_PRESERVED_CCP_VALUES[$index]}"
		export "$name"
	done
}

load_local_env_defaults() {
	local env_file
	preserve_existing_ccp_vars
	for env_file in \
		"$ROOT_DIR/.env" \
		"$ROOT_DIR/.env.live-proof.local" \
		"$ROOT_DIR/.env.live-proof.secrets"
	do
		if [[ -f "$env_file" ]]; then
			set -a
			# shellcheck source=/dev/null
			source "$env_file"
			set +a
		fi
	done
	restore_existing_ccp_vars
}

load_local_env_defaults
