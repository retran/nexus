#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
COMPOSE_FILE="${REPO_ROOT}/docker-compose.dev.yaml"

VAULT_TOKEN=${VAULT_TOKEN:-}
VAULT_ADDR_IN_CONTAINER=${VAULT_ADDR:-"http://127.0.0.1:8200"}
VAULT_SERVICE_NAME="vault"
TRANSIT_PATH="${TRANSIT_PATH:-transit}"
TRANSIT_PATH="${TRANSIT_PATH#/}"
TRANSIT_PATH="${TRANSIT_PATH%/}"
if [ -z "${TRANSIT_PATH}" ]; then
  TRANSIT_PATH="transit"
fi
TRANSIT_KEY_NAME=${TRANSIT_KEY_NAME:-"service-jwt-key"}
TRANSIT_KEY_TYPE=${TRANSIT_KEY_TYPE:-"rsa-4096"}

if [ -z "${VAULT_TOKEN}" ]; then
  echo "[ERROR] VAULT_TOKEN environment variable is required" >&2
  exit 1
fi

if [ -n "${DOCKER_COMPOSE_BIN:-}" ]; then
  DOCKER_COMPOSE_BIN_RESOLVED="${DOCKER_COMPOSE_BIN}"
elif command -v docker-compose >/dev/null 2>&1; then
  DOCKER_COMPOSE_BIN_RESOLVED="docker-compose"
else
  DOCKER_COMPOSE_BIN_RESOLVED="docker compose"
fi

compose() {
  if [ "${DOCKER_COMPOSE_BIN_RESOLVED}" = "docker-compose" ]; then
    docker-compose -f "${COMPOSE_FILE}" "$@"
  else
    docker compose -f "${COMPOSE_FILE}" "$@"
  fi
}

vault_exec() {
  compose exec -T \
    -e VAULT_ADDR="${VAULT_ADDR_IN_CONTAINER}" \
    -e VAULT_TOKEN="${VAULT_TOKEN}" \
    "${VAULT_SERVICE_NAME}" "$@"
}

enable_transit() {
  local mount_path="${TRANSIT_PATH}"
  echo "[INFO] Checking transit engine at path '${mount_path}/'..."
  secrets_json=$(vault_exec vault secrets list -format=json)

  if echo "${secrets_json}" | jq -e --arg path "${mount_path}/" 'has($path)' >/dev/null; then
    echo "[INFO] Transit engine already enabled at path '${mount_path}/'."
    return
  fi

  echo "[INFO] Enabling transit engine at path '${mount_path}/'..."
  vault_exec vault secrets enable -path="${mount_path}" transit >/dev/null
  echo "[INFO] Transit engine enabled."
}

create_signing_key() {
  echo "[INFO] Ensuring transit key '${TRANSIT_KEY_NAME}' exists..."

  if vault_exec vault read "${TRANSIT_PATH}/keys/${TRANSIT_KEY_NAME}" >/dev/null 2>&1; then
    echo "[INFO] Transit key '${TRANSIT_KEY_NAME}' already exists."
    return
  fi

  vault_exec vault write "${TRANSIT_PATH}/keys/${TRANSIT_KEY_NAME}" \
    type="${TRANSIT_KEY_TYPE}" \
    exportable=false \
    allow_plaintext_backup=false >/dev/null

  vault_exec vault write "${TRANSIT_PATH}/keys/${TRANSIT_KEY_NAME}/config" \
    min_decryption_version=1 \
    min_encryption_version=1 \
    deletion_allowed=false \
    exportable=false >/dev/null

  echo "[INFO] Transit key '${TRANSIT_KEY_NAME}' created for JWT signing."
}

configure_key_rotation() {
  echo "[INFO] Configuring key rotation policy for '${TRANSIT_KEY_NAME}'..."

  vault_exec vault write "${TRANSIT_PATH}/keys/${TRANSIT_KEY_NAME}/config" \
    min_decryption_version=1 \
    min_encryption_version=1 \
    min_available_version=1 \
    deletion_allowed=false >/dev/null

  echo "[INFO] Transit key rotation settings applied."
}

enable_transit
create_signing_key
configure_key_rotation

echo "[SUCCESS] Transit engine ready. Key '${TRANSIT_KEY_NAME}' configured for JWT signing."
