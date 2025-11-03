#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
COMPOSE_FILE="${REPO_ROOT}/docker-compose.dev.yaml"
STATE_DIR="${REPO_ROOT}/vault/.dev"
VAULT_SERVICE_NAME="vault"
VAULT_ADDR_IN_CONTAINER="${VAULT_ADDR:-http://127.0.0.1:8200}"
ROOT_TOKEN_FILE="${STATE_DIR}/root-token"
UNSEAL_KEY_FILE="${STATE_DIR}/unseal-key"
TRANSIT_PATH="${TRANSIT_PATH:-transit}"
TRANSIT_KEY_NAME="${TRANSIT_KEY_NAME:-service-jwt-key}"

TRANSIT_PATH="${TRANSIT_PATH#/}"
TRANSIT_PATH="${TRANSIT_PATH%/}"
if [ -z "${TRANSIT_PATH}" ]; then
  TRANSIT_PATH="transit"
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "[ERROR] jq is required to rotate transit keys" >&2
  exit 1
fi

if [ -n "${VAULT_TOKEN:-}" ]; then
  ROOT_TOKEN="${VAULT_TOKEN}"
elif [ -f "${ROOT_TOKEN_FILE}" ]; then
  ROOT_TOKEN=$(<"${ROOT_TOKEN_FILE}")
else
  echo "[ERROR] Root token not provided. Set VAULT_TOKEN or run 'task vault:init' to generate ${ROOT_TOKEN_FILE}" >&2
  exit 1
fi

UNSEAL_KEY=""
if [ -f "${UNSEAL_KEY_FILE}" ]; then
  UNSEAL_KEY=$(<"${UNSEAL_KEY_FILE}")
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
  local token="${1:-}"
  shift || true
  if [ -n "${token}" ]; then
    compose exec -T \
      -e VAULT_ADDR="${VAULT_ADDR_IN_CONTAINER}" \
      -e VAULT_TOKEN="${token}" \
      "${VAULT_SERVICE_NAME}" "$@"
  else
    compose exec -T \
      -e VAULT_ADDR="${VAULT_ADDR_IN_CONTAINER}" \
      "${VAULT_SERVICE_NAME}" "$@"
  fi
}

ensure_vault_ready() {
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
      echo "[ERROR] Vault is sealed and no unseal key available. Provide UNSEAL_KEY_FILE or set VAULT_TOKEN for an unsealed instance." >&2
      exit 1
    fi
    echo "[INFO] Vault is sealed. Unsealing with stored key..."
    vault_exec "" vault operator unseal "${UNSEAL_KEY}" >/dev/null
  fi
}

rotate_transit_key() {
  echo "[INFO] Rotating transit key '${TRANSIT_KEY_NAME}'..."
  local rotation_json
  set +e
  rotation_json=$(vault_exec "${ROOT_TOKEN}" vault write -format=json -f "${TRANSIT_PATH}/keys/${TRANSIT_KEY_NAME}/rotate" 2>/dev/null)
  local exit_code=$?
  set -e

  if [ ${exit_code} -ne 0 ] || [ -z "${rotation_json}" ]; then
    echo "[ERROR] Failed to rotate transit key '${TRANSIT_KEY_NAME}' via mount '${TRANSIT_PATH}'" >&2
    exit 1
  fi

  local latest_version
  latest_version=$(echo "${rotation_json}" | jq -r '.data.latest_version // empty')
  if [ -n "${latest_version}" ]; then
    echo "[SUCCESS] Transit key rotated. Latest version: ${latest_version}"
  else
    echo "[SUCCESS] Transit key rotated."
  fi
}

ensure_vault_ready
rotate_transit_key
