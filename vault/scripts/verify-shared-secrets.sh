#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
COMPOSE_FILE="${REPO_ROOT}/docker-compose.dev.yaml"
STATE_DIR="${REPO_ROOT}/vault/.dev"
VAULT_ADDR_IN_CONTAINER="http://127.0.0.1:8200"
VAULT_SERVICE_NAME="vault"
ROOT_TOKEN_FILE="${STATE_DIR}/root-token"

if ! command -v jq >/dev/null 2>&1; then
  echo "[ERROR] jq is required for verifying secrets" >&2
  exit 1
fi

if [ -f "${ROOT_TOKEN_FILE}" ]; then
  ROOT_TOKEN=$(<"${ROOT_TOKEN_FILE}")
else
  echo "[ERROR] Root token file not found at ${ROOT_TOKEN_FILE}. Run 'task vault:init' first." >&2
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
    docker-compose -f "${COMPOSE_FILE}" "$@"
  else
    docker compose -f "${COMPOSE_FILE}" "$@"
  fi
}

vault_exec() {
  compose exec -T \
    -e VAULT_ADDR="${VAULT_ADDR_IN_CONTAINER}" \
    -e VAULT_TOKEN="${ROOT_TOKEN}" \
    "${VAULT_SERVICE_NAME}" \
    "$@"
}

ensure_vault_running() {
  compose up -d "${VAULT_SERVICE_NAME}" >/dev/null
}

ensure_vault_running

echo "[INFO] Verifying shared secrets..."

SECRETS_TO_CHECK=(
  "shared/webhook:Kratos webhook shared secret"
  "shared/oathkeeper:Oathkeeper shared secret"
)

failures=0

for spec in "${SECRETS_TO_CHECK[@]}"; do
  IFS=":" read -r path description <<<"${spec}"
  echo "[INFO] Checking ${description} (${path})..."

  set +e
  secret_json=$(vault_exec vault kv get -format=json "kv/${path}" 2>/dev/null)
  status=$?
  set -e

  if [ ${status} -ne 0 ] || [ -z "${secret_json}" ]; then
    echo "[ERROR] Secret not found at kv/${path}"
    failures=$((failures + 1))
    continue
  fi

  current=$(echo "${secret_json}" | jq -r '.data.data.current // ""')
  rotated_at=$(echo "${secret_json}" | jq -r '.data.data.rotated_at // ""')

  if [ -z "${current}" ]; then
    echo "[ERROR] Missing 'current' value for kv/${path}"
    failures=$((failures + 1))
    continue
  fi

  if [ -z "${rotated_at}" ]; then
    echo "[WARN] No rotation timestamp recorded for kv/${path}"
  fi

  echo "[OK] Secret kv/${path} is present (rotated_at=${rotated_at})"
done

if [ "${failures}" -gt 0 ]; then
  echo "[FAIL] Secret verification failed for ${failures} entr(ies)."
  exit 1
fi

echo "[SUCCESS] All shared secrets verified."
