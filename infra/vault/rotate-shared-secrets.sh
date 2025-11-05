#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
STATE_DIR="${REPO_ROOT}/vault/.dev"
VAULT_ADDR_IN_CONTAINER="http://127.0.0.1:8200"
VAULT_SERVICE_NAME="vault"
ROOT_TOKEN_FILE="${STATE_DIR}/root-token"
UNSEAL_KEY_FILE="${STATE_DIR}/unseal-key"
ROTATION_LOG="${STATE_DIR}/rotation-log.jsonl"

REQUIRED_BINARIES=(jq openssl python3)

for bin in "${REQUIRED_BINARIES[@]}"; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "[ERROR] ${bin} is required for secret rotation" >&2
    exit 1
  fi
done

if [ -f "${ROOT_TOKEN_FILE}" ]; then
  ROOT_TOKEN=$(<"${ROOT_TOKEN_FILE}")
else
  echo "[ERROR] Root token file not found at ${ROOT_TOKEN_FILE}. Run 'task vault:init' first." >&2
  exit 1
fi

if [ -f "${UNSEAL_KEY_FILE}" ]; then
  UNSEAL_KEY=$(<"${UNSEAL_KEY_FILE}")
else
  UNSEAL_KEY=""
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
      echo "[ERROR] Vault is sealed and no unseal key file found at ${UNSEAL_KEY_FILE}" >&2
      exit 1
    fi
    echo "[INFO] Vault is sealed. Unsealing..."
    vault_exec "" vault operator unseal "${UNSEAL_KEY}" >/dev/null
  fi
}

generate_future_timestamp() {
  local hours="$1"
  python3 - <<PY
import datetime
print((datetime.datetime.utcnow() + datetime.timedelta(hours=${hours})).replace(microsecond=0).isoformat() + "Z")
PY
}

rotate_secret() {
  local path="$1"
  local description="$2"
  local length="$3"

  echo "[INFO] Rotating secret '${path}' (${description})..."

  local current_json
  set +e
  current_json=$(vault_exec "${ROOT_TOKEN}" vault kv get -format=json "kv/${path}" 2>/dev/null)
  local status=$?
  set -e

  local current_secret=""
  local previous_secret=""
  local current_rotated_at=""
  local rotation_count=0

  if [ ${status} -eq 0 ] && [ -n "${current_json}" ]; then
    current_secret=$(echo "${current_json}" | jq -r '.data.data.current // ""')
    previous_secret=$(echo "${current_json}" | jq -r '.data.data.previous // ""')
    current_rotated_at=$(echo "${current_json}" | jq -r '.data.data.rotated_at // ""')
    rotation_count=$(echo "${current_json}" | jq -r '.data.data.rotation_count // 0')
  fi

  local new_secret
  new_secret=$(openssl rand -hex "${length}")
  local rotated_at
  rotated_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  local expires_at
  expires_at=$(generate_future_timestamp 24)
  local rotation_id="rot-$(date -u +"%Y%m%dT%H%M%SZ")"
  local new_rotation_count=$((rotation_count + 1))

  vault_exec "${ROOT_TOKEN}" vault kv put "kv/${path}" \
    current="${new_secret}" \
    previous="${current_secret}" \
    previous_rotated_at="${current_rotated_at}" \
    rotated_at="${rotated_at}" \
    expires_at="${expires_at}" \
    rotation_id="${rotation_id}" \
    rotation_count="${new_rotation_count}" \
    managed_by="rotate-shared-secrets.sh" >/dev/null

  mkdir -p "$(dirname "${ROTATION_LOG}")"
  cat >> "${ROTATION_LOG}" <<EOF
{"timestamp":"${rotated_at}","path":"${path}","rotation_id":"${rotation_id}","description":"${description}","rotation_count":${new_rotation_count}}
EOF

  echo "[INFO] Rotation complete for '${path}'."
}

ensure_vault_running

echo "[INFO] Validating root token access..."
vault_exec "${ROOT_TOKEN}" vault token lookup >/dev/null

SECRETS_TO_ROTATE=(
  "shared/webhook:Kratos webhook shared secret:32"
  "shared/oathkeeper:Oathkeeper shared secret:32"
)

for spec in "${SECRETS_TO_ROTATE[@]}"; do
  IFS=":" read -r path description size <<<"${spec}"
  rotate_secret "${path}" "${description}" "${size}"
done

echo "[SUCCESS] Shared secret rotation complete."
