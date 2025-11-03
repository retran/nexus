#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
COMPOSE_FILE="${REPO_ROOT}/docker-compose.dev.yaml"
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

require_var "POSTGRES_USER"
require_var "POSTGRES_PASSWORD"
require_var "POSTGRES_DB"
require_var "REDIS_PASSWORD"
require_var "KRATOS_COOKIE_SECRET"
require_var "KRATOS_CIPHER_SECRET"
require_var "JWT_SECRET"
require_var "COOKIE_SECRET"
require_var "CSRF_COOKIE_SECRET"
require_var "WEBHOOK_SECRET"

POSTGRES_HOST=${POSTGRES_HOST:-postgres}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_SSLMODE=${POSTGRES_SSLMODE:-disable}
TEMPORAL_NAMESPACE=${TEMPORAL_NAMESPACE:-default}
TEMPORAL_TASK_QUEUE=${TEMPORAL_TASK_QUEUE:-nexus-task-queue}

OATHKEEPER_SHARED_SECRET=${OATHKEEPER_SHARED_SECRET:-}
if [ -z "${OATHKEEPER_SHARED_SECRET}" ]; then
  OATHKEEPER_SHARED_SECRET=$(openssl rand -hex 32)
  echo "[WARN] OATHKEEPER_SHARED_SECRET not set; generated random secret."
fi

ensure_vault_running

echo "[INFO] Validating root token access..."
vault_exec "${ROOT_TOKEN}" vault token lookup >/dev/null

put_secret "services/data-api" \
  "postgres_host=${POSTGRES_HOST}" \
  "postgres_port=${POSTGRES_PORT}" \
  "postgres_db=${POSTGRES_DB}" \
  "postgres_user=${POSTGRES_USER}" \
  "postgres_password=${POSTGRES_PASSWORD}" \
  "postgres_sslmode=${POSTGRES_SSLMODE}"

put_secret "services/gateway" \
  "postgres_host=${POSTGRES_HOST}" \
  "postgres_port=${POSTGRES_PORT}" \
  "postgres_db=${POSTGRES_DB}" \
  "postgres_user=${POSTGRES_USER}" \
  "postgres_password=${POSTGRES_PASSWORD}" \
  "redis_host=${REDIS_HOST:-redis}" \
  "redis_port=${REDIS_PORT:-6379}" \
  "redis_password=${REDIS_PASSWORD}" \
  "jwt_secret=${JWT_SECRET}" \
  "google_client_id=${GOOGLE_CLIENT_ID:-}" \
  "google_client_secret=${GOOGLE_CLIENT_SECRET:-}" \
  "frontend_url=${FRONTEND_URL:-http://nexus.local}" \
  "temporal_host=${TEMPORAL_HOST:-temporal:7233}" \
  "temporal_namespace=${TEMPORAL_NAMESPACE}" \
  "temporal_task_queue=${TEMPORAL_TASK_QUEUE}"

put_secret "services/worker" \
  "postgres_host=${POSTGRES_HOST}" \
  "postgres_port=${POSTGRES_PORT}" \
  "postgres_db=${POSTGRES_DB}" \
  "postgres_user=${POSTGRES_USER}" \
  "postgres_password=${POSTGRES_PASSWORD}" \
  "temporal_host=${TEMPORAL_HOST:-temporal:7233}" \
  "temporal_namespace=${TEMPORAL_NAMESPACE}" \
  "temporal_task_queue=${TEMPORAL_TASK_QUEUE}" \
  "jwt_secret=${JWT_SECRET}"

put_secret "services/kratos" \
  "cookie_secret=${KRATOS_COOKIE_SECRET}" \
  "cipher_secret=${KRATOS_CIPHER_SECRET}" \
  "public_url=${KRATOS_PUBLIC_URL:-http://auth.nexus.local}" \
  "admin_url=${KRATOS_ADMIN_URL:-http://kratos:4434}" \
  "webhook_secret=${WEBHOOK_SECRET:-}" \
  "google_client_id=${GOOGLE_CLIENT_ID:-}" \
  "google_client_secret=${GOOGLE_CLIENT_SECRET:-}" \
  "apple_client_id=${APPLE_CLIENT_ID:-}" \
  "apple_client_secret=${APPLE_CLIENT_SECRET:-}" \
  "apple_team_id=${APPLE_TEAM_ID:-}" \
  "apple_key_id=${APPLE_KEY_ID:-}"

put_secret "services/frontend" \
  "api_url=${VITE_API_URL:-http://api.nexus.local}" \
  "auth_url=${VITE_AUTH_URL:-http://auth.nexus.local}" \
  "cookie_secret=${COOKIE_SECRET}" \
  "csrf_cookie_name=${CSRF_COOKIE_NAME:-__HOST-nexus_csrf}" \
  "csrf_cookie_secret=${CSRF_COOKIE_SECRET}"

put_secret "services/postgres" \
  "user=${POSTGRES_USER}" \
  "password=${POSTGRES_PASSWORD}" \
  "database=${POSTGRES_DB}"

put_secret "services/redis" \
  "password=${REDIS_PASSWORD}"

seed_shared_secret "shared/webhook" "${WEBHOOK_SECRET:-}"
seed_shared_secret "shared/oathkeeper" "${OATHKEEPER_SHARED_SECRET}"

echo "[SUCCESS] Vault dev secrets seeded from ${ENV_FILE}"
