#!/usr/bin/env bash
# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: Apache-2.0

# Script to seed initial Keto relationship tuples for testing
# Creates admin role and assigns test user to it

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/../.." && pwd)
COMPOSE_FILE="${REPO_ROOT}/docker-compose.yaml"

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

keto_exec() {
  compose exec -T keto "$@"
}

echo "[INFO] Getting admin user ID from Kratos..."
ADMIN_ID=$(curl -s http://localhost:4434/admin/identities | jq -r '.[] | select(.traits.email=="admin@nexus.local") | .id')

if [ -z "$ADMIN_ID" ] || [ "$ADMIN_ID" = "null" ]; then
  echo "[ERROR] Admin user not found in Kratos. Please run 'task infra:kratos:seed' first." >&2
  exit 1
fi

echo "[INFO] Creating admin role for user: $ADMIN_ID..."
keto_exec keto relation-tuple create \
  --insecure-disable-transport-security \
  /dev/stdin <<EOF
{
  "namespace": "Role",
  "object": "admin",
  "relation": "members",
  "subject_id": "$ADMIN_ID"
}
EOF

echo "[INFO] Creating test resource with admin access..."
keto_exec keto relation-tuple create \
  --insecure-disable-transport-security \
  /dev/stdin <<EOF
{
  "namespace": "Resource",
  "object": "test-document-1",
  "relation": "admins",
  "subject_set": {
    "namespace": "Role",
    "object": "admin",
    "relation": "members"
  }
}
EOF

echo "[INFO] Creating test resource with viewer access for specific user..."
keto_exec keto relation-tuple create \
  --insecure-disable-transport-security \
  /dev/stdin <<EOF
{
  "namespace": "Resource",
  "object": "test-document-2",
  "relation": "viewers",
  "subject_id": "test-viewer-user-id"
}
EOF

echo "[INFO] Listing all relationship tuples..."
echo "Role tuples:"
keto_exec keto relation-tuple get \
  --insecure-disable-transport-security \
  --namespace Role
echo "Resource tuples:"
keto_exec keto relation-tuple get \
  --insecure-disable-transport-security \
  --namespace Resource

echo ""
echo "[SUCCESS] Keto relationship tuples created successfully!"
echo ""
echo "[INFO] Testing permission checks..."
echo "1. Check if test-admin-user-id can delete test-document-1 (should be ALLOWED):"
keto_exec keto check \
  --insecure-disable-transport-security \
  test-admin-user-id delete Resource test-document-1

echo ""
echo "2. Check if test-viewer-user-id can view test-document-2 (should be ALLOWED):"
keto_exec keto check \
  --insecure-disable-transport-security \
  test-viewer-user-id view Resource test-document-2

echo ""
echo "3. Check if test-viewer-user-id can delete test-document-2 (should be DENIED):"
keto_exec keto check \
  --insecure-disable-transport-security \
  test-viewer-user-id delete Resource test-document-2

echo ""
echo "[SUCCESS] All permission checks completed!"
