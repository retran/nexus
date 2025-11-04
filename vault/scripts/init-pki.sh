#!/usr/bin/env bash
# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: Apache-2.0

# Initialize Vault PKI (Public Key Infrastructure) engine for mTLS
# This script sets up an Intermediate Certificate Authority for issuing
# service certificates used in service-to-service mTLS authentication
#
# Environment variables required:
#   VAULT_TOKEN - Vault root token
#   VAULT_ADDR - Vault API address
#   DOCKER_COMPOSE_BIN - docker-compose or docker compose command

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
COMPOSE_FILE="${REPO_ROOT}/docker-compose.dev.yaml"

vault_exec() {
  if [ "${DOCKER_COMPOSE_BIN:-docker-compose}" = "docker-compose" ]; then
    docker-compose -f "${COMPOSE_FILE}" exec -T \
      -e VAULT_ADDR="${VAULT_ADDR}" \
      -e VAULT_TOKEN="${VAULT_TOKEN}" \
      vault "$@"
  else
    docker compose -f "${COMPOSE_FILE}" exec -T \
      -e VAULT_ADDR="${VAULT_ADDR}" \
      -e VAULT_TOKEN="${VAULT_TOKEN}" \
      vault "$@"
  fi
}

echo "[INFO] Configuring Vault PKI engine for mTLS..."

# Enable PKI secrets engine at pki_int path (intermediate CA)
echo "[INFO] Enabling PKI secrets engine at pki_int..."
vault_exec vault secrets enable -path=pki_int pki 2>/dev/null || {
  echo "[WARN] PKI engine already enabled at pki_int, skipping..."
}

# Configure maximum TTL for certificates (5 years for development)
# In production, this should be shorter (e.g., 1-2 years)
echo "[INFO] Configuring PKI max lease TTL..."
vault_exec vault secrets tune -max-lease-ttl=43800h pki_int

# Generate intermediate CA certificate
# Using 'internal' means Vault generates and stores the private key internally
# For dev mode, we create a self-signed root within pki_int itself
echo "[INFO] Generating self-signed Intermediate CA (dev-mode)..."
vault_exec vault write -field=certificate pki_int/root/generate/internal \
  common_name="Nexus Dev Intermediate CA" \
  ttl=43800h \
  > /dev/null

echo "[INFO] Intermediate CA generated successfully!"

# Configure CA and CRL URLs
# These URLs will be encoded in issued certificates
echo "[INFO] Configuring CA URLs..."
vault_exec vault write pki_int/config/urls \
  issuing_certificates="http://vault:8200/v1/pki_int/ca" \
  crl_distribution_points="http://vault:8200/v1/pki_int/crl"

echo "[SUCCESS] Vault PKI engine configured successfully!"
echo ""
echo "Next steps:"
echo "  - Run init-policies.sh to configure PKI roles and policies"
echo "  - Configure Vault Agent sidecars to fetch certificates"
