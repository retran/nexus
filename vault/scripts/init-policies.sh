#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
COMPOSE_FILE="${REPO_ROOT}/docker-compose.dev.yaml"

VAULT_TOKEN=${VAULT_TOKEN:-}
VAULT_ADDR_IN_CONTAINER=${VAULT_ADDR:-"http://127.0.0.1:8200"}

if [ -z "${VAULT_TOKEN}" ]; then
  echo "[ERROR] VAULT_TOKEN environment variable is required" >&2
  exit 1
fi

if [ -z "${VAULT_ADDR_IN_CONTAINER}" ]; then
  echo "[ERROR] VAULT_ADDR environment variable is required" >&2
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
    vault "$@"
}

apply_policy() {
  local name="$1"
  local file_path="$2"
  compose exec -T vault sh -c "cat > /tmp/${name}.hcl" < "${file_path}"
  vault_exec vault policy write "${name}" "/tmp/${name}.hcl" >/dev/null
  compose exec -T vault rm -f "/tmp/${name}.hcl" >/dev/null
}

tmpdir=$(mktemp -d)
trap 'rm -rf "${tmpdir}"' EXIT

services=(data-api gateway worker internal-api oathkeeper kratos temporal)

echo "[INFO] Writing service policies..."
for service in "${services[@]}"; do
  policy_name="app-${service}"
  policy_file="${tmpdir}/${policy_name}.hcl"

  cat > "${policy_file}" <<EOF
path "kv/data/services/${service}" {
  capabilities = ["read"]
}

path "kv/data/services/${service}/*" {
  capabilities = ["read"]
}

path "kv/data/shared" {
  capabilities = ["read"]
}

path "kv/data/shared/*" {
  capabilities = ["read"]
}

path "kv/metadata/services/${service}" {
  capabilities = ["list"]
}

path "kv/metadata/services/${service}/*" {
  capabilities = ["list"]
}

path "kv/metadata/services" {
  capabilities = ["list"]
}

path "kv/metadata/shared" {
  capabilities = ["list"]
}

path "kv/metadata/shared/*" {
  capabilities = ["list"]
}

# PKI Engine - Issue mTLS certificates for this service
path "pki_int/issue/${service}-role" {
  capabilities = ["update"]
}

# PKI Engine - Read CA certificate for client validation
path "pki_int/cert/ca" {
  capabilities = ["read"]
}

path "pki_int/ca/pem" {
  capabilities = ["read"]
}

path "sys/health" {
  capabilities = ["read"]
}
EOF

  apply_policy "${policy_name}" "${policy_file}"
  echo "[INFO] Policy '${policy_name}' applied."
done

admin_policy_file="${tmpdir}/vault-admin.hcl"
cat > "${admin_policy_file}" <<EOF
path "*" {
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}
EOF

apply_policy "vault-admin" "${admin_policy_file}"
echo "[INFO] Policy 'vault-admin' applied."

echo "[INFO] Policies up to date."
