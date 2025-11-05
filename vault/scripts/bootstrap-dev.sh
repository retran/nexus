#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
STATE_DIR="${REPO_ROOT}/vault/.dev"
VAULT_ADDR_IN_CONTAINER="http://127.0.0.1:8200"
VAULT_SERVICE_NAME="vault"
APPROLE_OUTPUT_DIR="${STATE_DIR}/approles"
ENV_FILE="${ENV_FILE:-${REPO_ROOT}/.env}"

if ! command -v jq >/dev/null 2>&1; then
  echo "[ERROR] jq is required for Vault bootstrap automation" >&2
  exit 1
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
    docker-compose "$@"
  else
    docker compose "$@"
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

mkdir -p "${STATE_DIR}"
mkdir -p "${APPROLE_OUTPUT_DIR}"

echo "[INFO] Ensuring Vault service is running..."
echo "[INFO] Preparing Vault data volume permissions..."
compose run --rm "${VAULT_SERVICE_NAME}" sh -c "chown -R vault:vault /vault/data" >/dev/null 2>&1 || true
compose up -d "${VAULT_SERVICE_NAME}" >/dev/null

CONTAINER_ID=$(compose ps -q "${VAULT_SERVICE_NAME}")
if [ -z "${CONTAINER_ID}" ]; then
  echo "[ERROR] Unable to locate Vault container. Is docker running?" >&2
  exit 1
fi

echo "[INFO] Waiting for Vault API to become reachable..."
status_json=""
for attempt in $(seq 1 60); do
  set +e
  status_json=$(vault_exec "" sh -c "vault status -format=json" 2>/dev/null)
  exit_code=$?
  set -e
  if [ ${exit_code} -eq 0 ] || [ -n "${status_json}" ]; then
    break
  fi
  sleep 2
done

if [ -z "${status_json}" ]; then
  echo "[ERROR] Vault API did not become available in time" >&2
  exit 1
fi

# In dev mode, Vault is always initialized with token "root"
# No need to check sealed/initialized status or perform init/unseal operations
root_token="root"

echo "[INFO] Using dev mode token: root"

echo "[INFO] Validating root token access..."
vault_exec "${root_token}" vault token lookup >/dev/null

echo "[INFO] Enabling KV secrets engine (kv/)..."
secrets_json=$(vault_exec "${root_token}" vault secrets list -format=json)
if ! echo "${secrets_json}" | jq -e 'has("kv/")' >/dev/null; then
  vault_exec "${root_token}" vault secrets enable -path=kv kv-v2 >/dev/null
else
  echo "[INFO] KV secrets engine already enabled."
fi

# Note: Dev mode uses Vault ONLY as KV secrets storage
# Production features NOT configured in dev:
# - NO AppRole auth (services don't authenticate to Vault in dev)
# - NO policies (not needed without AppRole - see init-policies.sh)
# - NO OIDC (Vault UI accessed via token in dev)
#
# For production setup with AppRole/Policies, run manually or see bootstrap-prod.sh

echo "[SUCCESS] Vault dev bootstrap complete"
echo "[INFO] Access Vault UI at http://localhost:8200 (token: root)"
