#!/usr/bin/env sh

# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: Apache-2.0

# This script loads Kratos secrets from Vault and writes them to an env file
# for use by the Kratos container. It's designed to run as an init container.

set -eu

VAULT_ADDR="${VAULT_ADDR:-http://vault:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-}"
OUTPUT_FILE="${1:-/secrets/kratos.env}"

if [ -z "${VAULT_TOKEN}" ]; then
  echo "[ERROR] VAULT_TOKEN environment variable is required" >&2
  exit 1
fi

echo "[INFO] Loading Kratos secrets from Vault..."

# Ensure output directory exists
mkdir -p "$(dirname "${OUTPUT_FILE}")"

# Helper function to get secret from Vault
get_secret() {
  path="$1"
  field="$2"

  vault kv get -address="${VAULT_ADDR}" -field="${field}" "${path}" 2>/dev/null || {
    echo "[ERROR] Failed to read ${path} field ${field}" >&2
    return 1
  }
}

# Load Postgres password (shared secret)
POSTGRES_PASSWORD=$(get_secret "kv/shared/postgres" "current")

# Load Kratos encryption secrets
COOKIE_SECRET=$(get_secret "kv/services/kratos/encryption" "cookie")
CIPHER_SECRET=$(get_secret "kv/services/kratos/encryption" "cipher")

# Load webhook secret (shared)
WEBHOOK_SECRET=$(get_secret "kv/shared/webhook" "current")

# Load Google OIDC credentials (optional)
GOOGLE_CLIENT_ID=""
GOOGLE_CLIENT_SECRET=""

if vault kv get -address="${VAULT_ADDR}" "kv/services/kratos/google-oidc" >/dev/null 2>&1; then
  GOOGLE_CLIENT_ID=$(get_secret "kv/services/kratos/google-oidc" "client_id" || echo "")
  GOOGLE_CLIENT_SECRET=$(get_secret "kv/services/kratos/google-oidc" "client_secret" || echo "")
  echo "[INFO] Loaded Google OIDC credentials"
else
  echo "[INFO] Google OIDC credentials not configured"
fi

# Write secrets to env file
cat > "${OUTPUT_FILE}" <<EOF
# Kratos secrets loaded from Vault
# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Database connection
DSN=postgres://admin:${POSTGRES_PASSWORD}@postgres:5432/kratos_db?sslmode=disable&max_conns=20&max_idle_conns=4

# Encryption secrets
SECRETS_COOKIE=${COOKIE_SECRET}
SECRETS_CIPHER=${CIPHER_SECRET}

# Webhook secret
SELFSERVICE_FLOWS_LOGIN_AFTER_HOOKS_0_CONFIG_AUTH_CONFIG_VALUE=${WEBHOOK_SECRET}
SELFSERVICE_FLOWS_LOGIN_AFTER_OIDC_HOOKS_0_CONFIG_AUTH_CONFIG_VALUE=${WEBHOOK_SECRET}
SELFSERVICE_FLOWS_REGISTRATION_AFTER_PASSWORD_HOOKS_0_CONFIG_AUTH_CONFIG_VALUE=${WEBHOOK_SECRET}

# Google OIDC (optional)
SELFSERVICE_METHODS_OIDC_CONFIG_PROVIDERS_0_CLIENT_ID=${GOOGLE_CLIENT_ID}
SELFSERVICE_METHODS_OIDC_CONFIG_PROVIDERS_0_CLIENT_SECRET=${GOOGLE_CLIENT_SECRET}
EOF

chmod 644 "${OUTPUT_FILE}"

echo "[INFO] Kratos secrets written to ${OUTPUT_FILE}"

echo "[SUCCESS] Kratos secrets written to ${OUTPUT_FILE}"
