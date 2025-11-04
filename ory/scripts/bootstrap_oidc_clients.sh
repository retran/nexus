#!/usr/bin/env bash
# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: Apache-2.0

# Bootstrap OIDC clients in Kratos for SSO integrations
# This script creates OAuth2 clients for Temporal UI and Grafana (future)

set -e

KRATOS_ADMIN_URL="${KRATOS_ADMIN_URL:-http://localhost:4434}"

echo "🔐 Bootstrapping OIDC clients in Kratos..."
echo "Kratos Admin URL: ${KRATOS_ADMIN_URL}"

# Function to create or update OIDC client
create_or_update_client() {
    local client_id="$1"
    local client_name="$2"
    local redirect_uris="$3"
    local client_secret="$4"

    echo "📝 Processing client: ${client_name} (${client_id})"

    # Check if client exists
    if curl -sf "${KRATOS_ADMIN_URL}/admin/clients/${client_id}" > /dev/null 2>&1; then
        echo "   ✓ Client already exists, updating..."
        curl -sf -X PUT "${KRATOS_ADMIN_URL}/admin/clients/${client_id}" \
            -H "Content-Type: application/json" \
            -d @- <<EOF
{
    "client_id": "${client_id}",
    "client_name": "${client_name}",
    "client_secret": "${client_secret}",
    "redirect_uris": ${redirect_uris},
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "scope": "openid profile email offline_access",
    "token_endpoint_auth_method": "client_secret_basic"
}
EOF
        echo "   ✅ Client updated successfully"
    else
        echo "   ➕ Client not found, creating..."
        curl -sf -X POST "${KRATOS_ADMIN_URL}/admin/clients" \
            -H "Content-Type: application/json" \
            -d @- <<EOF
{
    "client_id": "${client_id}",
    "client_name": "${client_name}",
    "client_secret": "${client_secret}",
    "redirect_uris": ${redirect_uris},
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "scope": "openid profile email offline_access",
    "token_endpoint_auth_method": "client_secret_basic"
}
EOF
        echo "   ✅ Client created successfully"
    fi
}

# Temporal UI client
TEMPORAL_CLIENT_ID="${TEMPORAL_OIDC_CLIENT_ID:-dev-temporal-ui}"
TEMPORAL_CLIENT_SECRET="${TEMPORAL_OIDC_CLIENT_SECRET:-dev-temporal-secret-change-in-production}"
TEMPORAL_REDIRECT_URIS='["http://localhost:8088/auth/sso/callback"]'

create_or_update_client \
    "${TEMPORAL_CLIENT_ID}" \
    "Temporal UI" \
    "${TEMPORAL_REDIRECT_URIS}" \
    "${TEMPORAL_CLIENT_SECRET}"

# Grafana client (for future use)
GRAFANA_CLIENT_ID="${GRAFANA_OIDC_CLIENT_ID:-dev-grafana-ui}"
GRAFANA_CLIENT_SECRET="${GRAFANA_OIDC_CLIENT_SECRET:-dev-grafana-secret-change-in-production}"
GRAFANA_REDIRECT_URIS='["http://localhost:3001/login/generic_oauth"]'

create_or_update_client \
    "${GRAFANA_CLIENT_ID}" \
    "Grafana" \
    "${GRAFANA_REDIRECT_URIS}" \
    "${GRAFANA_CLIENT_SECRET}"

echo ""
echo "✅ All OIDC clients bootstrapped successfully!"
echo ""
echo "📋 Client credentials:"
echo "   Temporal UI:"
echo "      Client ID:     ${TEMPORAL_CLIENT_ID}"
echo "      Client Secret: ${TEMPORAL_CLIENT_SECRET}"
echo ""
echo "   Grafana:"
echo "      Client ID:     ${GRAFANA_CLIENT_ID}"
echo "      Client Secret: ${GRAFANA_CLIENT_SECRET}"
echo ""
echo "⚠️  Remember to update these secrets in production!"
