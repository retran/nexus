#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
STATE_DIR="${REPO_ROOT}/vault/.dev"
VAULT_ADDR_IN_CONTAINER="http://127.0.0.1:8200"
VAULT_SERVICE_NAME="vault"
ROOT_TOKEN_FILE="${STATE_DIR}/root-token"
UNSEAL_KEY_FILE="${STATE_DIR}/unseal-key"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env}"

REQUIRED_BINARIES=(jq openssl python3)

for bin in "${REQUIRED_BINARIES[@]}"; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "[ERROR] ${bin} is required for seeding Vault secrets" >&2
    exit 1
  fi
done

if [ ! -f "${ENV_FILE}" ]; then
  echo "[ERROR] Environment file not found at ${ENV_FILE}. Set ENV_FILE or create .env" >&2
  exit 1
fi

if [ -f "${ROOT_TOKEN_FILE}" ]; then
  ROOT_TOKEN=$(<"${ROOT_TOKEN_FILE}")
else
  echo "[ERROR] Root token file not found at ${ROOT_TOKEN_FILE}. Run 'task vault:init' first." >&2
  exit 1
fi

UNSEAL_KEY=""
if [ -f "${UNSEAL_KEY_FILE}" ]; then
  UNSEAL_KEY=$(<"${UNSEAL_KEY_FILE}")
fi

if command -v docker-compose >/dev/null 2>&1; then
  DOCKER_COMPOSE_BIN="docker-compose"
elif command -v docker >/dev/null 2>&1; then
  DOCKER_COMPOSE_BIN="docker compose"
else
  echo "[ERROR] docker-compose or docker compose is required" >&2
  exit 1
fi

compose() {
  if [ "${DOCKER_COMPOSE_BIN}" = "docker-compose" ]; then
    docker compose "$@"
  else
    docker compose -f "${COMPOSE_FILE}" "$@"
  fi
}

vault_exec() {
  local token="${1:-}"
  shift || true
  if [ -n "${token}" ]; then
    compose exec -T \
      -e VAULT_ADDR="${VAULT_ADDR_IN_CONTAINER}" \
      -e VAULT_TOKEN="${token}" \
      "${VAULT_SERVICE_NAME}" \
      "$@"
  else
    compose exec -T \
      -e VAULT_ADDR="${VAULT_ADDR_IN_CONTAINER}" \
      "${VAULT_SERVICE_NAME}" \
      "$@"
  fi
}

ensure_vault_running() {
  compose up -d "${VAULT_SERVICE_NAME}" >/dev/null

  local status_json=""
  for attempt in $(seq 1 30); do
    set +e
    status_json=$(vault_exec "" sh -c "vault status -format=json" 2>/dev/null)
    exit_code=$?
    set -e
    if [ ${exit_code} -eq 0 ] && [ -n "${status_json}" ]; then
      break
    fi
    sleep 1
  done

  if [ -z "${status_json}" ]; then
    echo "[ERROR] Vault API did not respond. Ensure the service is healthy." >&2
    exit 1
  fi

  local sealed
  sealed=$(echo "${status_json}" | jq -r '.sealed')
  if [ "${sealed}" = "true" ]; then
    if [ -z "${UNSEAL_KEY}" ]; then
      echo "[ERROR] Vault is sealed and no unseal key found at ${UNSEAL_KEY_FILE}" >&2
      exit 1
    fi
    echo "[INFO] Vault is sealed. Unsealing..."
    vault_exec "" vault operator unseal "${UNSEAL_KEY}" >/dev/null
  fi
}

import_env() {
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
}

require_var() {
  local name="$1"
  local value="${!name:-}"
  if [ -z "${value}" ]; then
    echo "[ERROR] Required variable ${name} is not set in ${ENV_FILE}" >&2
    exit 1
  fi
}

put_secret() {
  local path="$1"
  shift
  local args=()
  while [ $# -gt 0 ]; do
    local entry="$1"
    shift
    if [ "${entry#*=}" != "" ]; then
      args+=("${entry}")
    fi
  done

  if [ "${#args[@]}" -eq 0 ]; then
    echo "[WARN] No values provided for kv/${path}; skipping write"
    return
  fi

  vault_exec "${ROOT_TOKEN}" vault kv put "kv/${path}" "${args[@]}" >/dev/null
  echo "[INFO] Seeded kv/${path}"
}

generate_future_timestamp() {
  local hours="$1"
  python3 - <<PY
import datetime
print((datetime.datetime.utcnow() + datetime.timedelta(hours=${hours})).replace(microsecond=0).isoformat() + "Z")
PY
}

seed_shared_secret() {
  local path="$1"
  local value="$2"

  if [ -z "${value}" ]; then
    value=$(openssl rand -hex 32)
    echo "[WARN] No value provided for ${path}; generated random secret."
  fi

  local now
  now=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  local expires_at
  expires_at=$(generate_future_timestamp 24)

  vault_exec "${ROOT_TOKEN}" vault kv put "kv/${path}" \
    current="${value}" \
    previous="" \
    rotated_at="${now}" \
    previous_rotated_at="" \
    expires_at="${expires_at}" \
    rotation_id="seed-${now//[:TZ-]/}" \
    rotation_count=1 \
    managed_by="seed-dev-secrets.sh" >/dev/null

  echo "[INFO] Seeded shared secret kv/${path}"
}

ensure_vault_running
import_env

require_var "KRATOS_WEBHOOK_SECRET"
require_var "OATHKEEPER_SHARED_SECRET"

ensure_vault_running

echo "[INFO] Validating root token access..."
vault_exec "${ROOT_TOKEN}" vault token lookup >/dev/null

# Note: In dev mode, services read configuration from environment variables,
# NOT from Vault. Vault is used ONLY for shared secrets that need rotation.
#
# Shared secrets stored in Vault (used across multiple services):
# - shared/webhook: Kratos → Gateway/System webhook authentication
# - shared/oathkeeper: Oathkeeper → downstream services authentication

echo "[INFO] Seeding shared secrets..."

seed_shared_secret "shared/webhook" "${KRATOS_WEBHOOK_SECRET}"
seed_shared_secret "shared/oathkeeper" "${OATHKEEPER_SHARED_SECRET}"

echo "[SUCCESS] Vault dev secrets seeded from ${ENV_FILE}"
