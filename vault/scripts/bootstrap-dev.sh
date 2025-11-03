#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
COMPOSE_FILE="${REPO_ROOT}/docker-compose.dev.yaml"
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

initialized=$(echo "${status_json}" | jq -r '.initialized')
sealed=$(echo "${status_json}" | jq -r '.sealed')

root_token_file="${STATE_DIR}/root-token"
unseal_key_file="${STATE_DIR}/unseal-key"
root_token=""
unseal_key=""

if [ "${initialized}" = "false" ]; then
  echo "[INFO] Initializing Vault (1 key share, 1 key threshold)..."
  init_json=$(vault_exec "" vault operator init -key-shares=1 -key-threshold=1 -format=json)
  unseal_key=$(echo "${init_json}" | jq -r '.unseal_keys_b64[0]')
  root_token=$(echo "${init_json}" | jq -r '.root_token')

  printf "%s\n" "${unseal_key}" > "${unseal_key_file}"
  printf "%s\n" "${root_token}" > "${root_token_file}"
  chmod 600 "${unseal_key_file}" "${root_token_file}"

  echo "[INFO] Vault initialized. Credentials written to vault/.dev"

  echo "[INFO] Unsealing Vault..."
  vault_exec "" vault operator unseal "${unseal_key}" >/dev/null
else
  if [ -f "${unseal_key_file}" ]; then
    unseal_key=$(<"${unseal_key_file}")
  fi
  if [ -f "${root_token_file}" ]; then
    root_token=$(<"${root_token_file}")
  fi
fi

if [ "${sealed}" = "true" ]; then
  if [ -z "${unseal_key}" ]; then
    echo "[ERROR] Vault is sealed and no unseal key found at ${unseal_key_file}" >&2
    exit 1
  fi
  echo "[INFO] Vault is sealed. Unsealing..."
  vault_exec "" vault operator unseal "${unseal_key}" >/dev/null
fi

if [ -z "${root_token}" ]; then
  echo "[ERROR] Root token not available. Expected at ${root_token_file}" >&2
  exit 1
fi

echo "[INFO] Validating root token access..."
vault_exec "${root_token}" vault token lookup >/dev/null

echo "[INFO] Enabling KV secrets engine (kv/)..."
secrets_json=$(vault_exec "${root_token}" vault secrets list -format=json)
if ! echo "${secrets_json}" | jq -e 'has("kv/")' >/dev/null; then
  vault_exec "${root_token}" vault secrets enable -path=kv kv-v2 >/dev/null
else
  echo "[INFO] KV secrets engine already enabled."
fi

echo "[INFO] Enabling AppRole auth method..."
auth_json=$(vault_exec "${root_token}" vault auth list -format=json)
if ! echo "${auth_json}" | jq -e 'has("approle/")' >/dev/null; then
  vault_exec "${root_token}" vault auth enable approle >/dev/null
else
  echo "[INFO] AppRole auth method already enabled."
fi

echo "[INFO] Applying Vault policies..."
VAULT_TOKEN="${root_token}" \
  VAULT_ADDR="${VAULT_ADDR_IN_CONTAINER}" \
  DOCKER_COMPOSE_BIN="${DOCKER_COMPOSE_BIN}" \
  "${SCRIPT_DIR}/init-policies.sh"

echo "[INFO] Configuring Vault transit engine for JWT signing..."
VAULT_TOKEN="${root_token}" \
  VAULT_ADDR="${VAULT_ADDR_IN_CONTAINER}" \
  DOCKER_COMPOSE_BIN="${DOCKER_COMPOSE_BIN}" \
  "${SCRIPT_DIR}/init-transit.sh"

services=(data-api gateway worker webhooks)

for service in "${services[@]}"; do
  role_name="app-${service}"
  policy_name="${role_name}"

  echo "[INFO] Configuring AppRole '${role_name}'..."
  vault_exec "${root_token}" vault write "auth/approle/role/${role_name}" \
    token_policies="${policy_name}" \
    secret_id_ttl=0 \
    token_ttl=1h \
    token_max_ttl=4h \
    secret_id_num_uses=0 \
    >/dev/null

  role_id_json=$(vault_exec "${root_token}" vault read -format=json "auth/approle/role/${role_name}/role-id")
  role_id=$(echo "${role_id_json}" | jq -r '.data.role_id')

  secret_id_json=$(vault_exec "${root_token}" vault write -format=json -f "auth/approle/role/${role_name}/secret-id")
  secret_id=$(echo "${secret_id_json}" | jq -r '.data.secret_id')
  secret_id_accessor=$(echo "${secret_id_json}" | jq -r '.data.secret_id_accessor')

  output_file="${APPROLE_OUTPUT_DIR}/${service}.json"
  cat > "${output_file}" <<EOF
{
  "role_name": "${role_name}",
  "policy": "${policy_name}",
  "role_id": "${role_id}",
  "secret_id": "${secret_id}",
  "secret_id_accessor": "${secret_id_accessor}"
}
EOF
  chmod 600 "${output_file}"
  echo "[INFO] AppRole credentials written to ${output_file}"
done

echo "[INFO] Configuring OIDC auth for Vault UI..."
VAULT_TOKEN="${root_token}" \
  VAULT_ADDR="${VAULT_ADDR_IN_CONTAINER}" \
  DOCKER_COMPOSE_BIN="${DOCKER_COMPOSE_BIN}" \
  ENV_FILE="${ENV_FILE}" \
  "${SCRIPT_DIR}/init-oidc.sh"

echo "[SUCCESS] Vault bootstrap complete. Root token stored at ${root_token_file}"
