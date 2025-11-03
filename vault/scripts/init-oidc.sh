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

REQUIRED_BINARIES=(jq)

for bin in "${REQUIRED_BINARIES[@]}"; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "[ERROR] ${bin} is required for Vault OIDC configuration" >&2
    exit 1
  fi
done

if [ ! -f "${ENV_FILE}" ]; then
  echo "[ERROR] Environment file not found at ${ENV_FILE}. Set ENV_FILE or create .env" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "${ENV_FILE}"
set +a

is_placeholder() {
  local value="$1"
  if [ -z "${value}" ]; then
    return 0
  fi
  case "${value}" in
    CHANGE_ME*|changeme*|TODO*|____*) return 0 ;;
  esac
  return 1
}

VAULT_OIDC_CLIENT_ID=${VAULT_OIDC_CLIENT_ID:-}
VAULT_OIDC_CLIENT_SECRET=${VAULT_OIDC_CLIENT_SECRET:-}

if is_placeholder "${VAULT_OIDC_CLIENT_ID}" || is_placeholder "${VAULT_OIDC_CLIENT_SECRET}"; then
  echo "[WARN] OIDC configuration skipped: set VAULT_OIDC_CLIENT_ID and VAULT_OIDC_CLIENT_SECRET in ${ENV_FILE}."
  exit 0
fi

if [ -z "${VAULT_OIDC_CLIENT_ID}" ] || [ -z "${VAULT_OIDC_CLIENT_SECRET}" ]; then
  echo "[ERROR] VAULT_OIDC_CLIENT_ID and VAULT_OIDC_CLIENT_SECRET must be set in ${ENV_FILE}" >&2
  exit 1
fi

OIDC_DISCOVERY_URL="${VAULT_OIDC_DISCOVERY_URL:-http://kratos:4433/.well-known/openid-configuration}"
OIDC_ISSUER="${VAULT_OIDC_ISSUER:-http://auth.nexus.local}"
OIDC_DEFAULT_ROLE="${VAULT_OIDC_DEFAULT_ROLE:-nexus-admin}"
OIDC_REDIRECT_URIS=${VAULT_OIDC_REDIRECT_URIS:-}

if [ -z "${OIDC_REDIRECT_URIS}" ]; then
  OIDC_REDIRECT_URIS="http://localhost:8200/ui/vault/auth/oidc/oidc/callback,http://localhost:8200/oidc/callback,http://vault:8200/ui/vault/auth/oidc/oidc/callback,http://vault:8200/oidc/callback"
fi

trim() {
  local var="$1"
  var="${var#"${var%%[![:space:]]*}"}"
  var="${var%"${var##*[![:space:]]}"}"
  printf '%s' "${var}"
}

IFS=',' read -r -a REDIRECT_URIS_RAW <<<"${OIDC_REDIRECT_URIS}"
REDIRECT_URIS=()
for uri in "${REDIRECT_URIS_RAW[@]}"; do
  trimmed=$(trim "${uri}")
  if [ -n "${trimmed}" ]; then
    REDIRECT_URIS+=("${trimmed}")
  fi
done

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

ensure_vault_running

echo "[INFO] Validating root token access..."
vault_exec "${ROOT_TOKEN}" vault token lookup >/dev/null

echo "[INFO] Enabling OIDC auth method..."
auths=$(vault_exec "${ROOT_TOKEN}" vault auth list -format=json)
if echo "${auths}" | jq -e 'has("oidc/")' >/dev/null; then
  echo "[INFO] OIDC auth method already enabled."
else
  vault_exec "${ROOT_TOKEN}" vault auth enable oidc >/dev/null
fi

echo "[INFO] Configuring OIDC provider (Kratos)..."
vault_exec "${ROOT_TOKEN}" vault write auth/oidc/config \
  oidc_discovery_url="${OIDC_DISCOVERY_URL}" \
  oidc_client_id="${VAULT_OIDC_CLIENT_ID}" \
  oidc_client_secret="${VAULT_OIDC_CLIENT_SECRET}" \
  default_role="${OIDC_DEFAULT_ROLE}" \
  bound_issuer="${OIDC_ISSUER}" \
  listing_visibility="unauth" >/dev/null

redirect_args=()
for uri in "${REDIRECT_URIS[@]}"; do
  redirect_args+=("allowed_redirect_uris=${uri}")
done

claim_mappings=""
if [ -n "${VAULT_OIDC_CLAIM_MAPPINGS:-}" ]; then
  claim_mappings="claim_mappings=${VAULT_OIDC_CLAIM_MAPPINGS}"
fi

policies="${VAULT_OIDC_POLICIES:-vault-admin}"

echo "[INFO] Writing OIDC role '${OIDC_DEFAULT_ROLE}'..."
role_args=("auth/oidc/role/${OIDC_DEFAULT_ROLE}" "user_claim=email" "policies=${policies}" "ttl=1h" "bound_audiences=${VAULT_OIDC_CLIENT_ID}")
for uri in "${REDIRECT_URIS[@]}"; do
  role_args+=("allowed_redirect_uris=${uri}")
done
if [ -n "${claim_mappings}" ]; then
  role_args+=("${claim_mappings}")
fi
vault_exec "${ROOT_TOKEN}" vault write "${role_args[@]}" >/dev/null

if [ -n "${VAULT_OIDC_ROLE_MAPPINGS:-}" ]; then
  IFS=',' read -r -a role_specs <<<"${VAULT_OIDC_ROLE_MAPPINGS}"
  for spec in "${role_specs[@]}"; do
    role_name=${spec%%:*}
    policies_list=${spec#*:}
    if [ -z "${role_name}" ] || [ -z "${policies_list}" ]; then
      continue
    fi
    echo "[INFO] Upserting OIDC role '${role_name}' with policies '${policies_list}'..."
    args=("auth/oidc/role/${role_name}" "user_claim=email" "policies=${policies_list}" "ttl=1h" "bound_audiences=${VAULT_OIDC_CLIENT_ID}")
    for uri in "${REDIRECT_URIS[@]}"; do
      args+=("allowed_redirect_uris=${uri}")
    done
    if [ -n "${claim_mappings}" ]; then
      args+=("${claim_mappings}")
    fi
    vault_exec "${ROOT_TOKEN}" vault write "${args[@]}" >/dev/null
  done
fi

echo "[SUCCESS] Vault OIDC auth configured. You can now log in via Kratos."
