#!/usr/bin/env bash
# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m'

KRATOS_ADMIN_URL="${KRATOS_ADMIN_URL:-http://localhost:4434}"
PRESERVE_ADMIN="${PRESERVE_ADMIN:-true}"

printf "${BLUE}[INFO]${NC} Fetching all identities from Kratos...\n"

# Fetch all identities
IDENTITIES=$(curl -s "${KRATOS_ADMIN_URL}/admin/identities" | jq -r '.[].id')

if [ -z "$IDENTITIES" ]; then
  printf "${YELLOW}[INFO]${NC} No identities found in Kratos\n"
  exit 0
fi

# Get admin email if we need to preserve it
ADMIN_EMAIL="admin@nexus.local"
ADMIN_ID=""

if [ "$PRESERVE_ADMIN" = "true" ]; then
  printf "${BLUE}[INFO]${NC} Finding admin identity (${ADMIN_EMAIL})...\n"
  ADMIN_ID=$(curl -s "${KRATOS_ADMIN_URL}/admin/identities" | \
    jq -r --arg email "$ADMIN_EMAIL" '.[] | select(.traits.email == $email) | .id')

  if [ -n "$ADMIN_ID" ]; then
    printf "${YELLOW}[INFO]${NC} Admin identity found: ${ADMIN_ID} (will be preserved)\n"
  fi
fi

# Delete identities
deleted_count=0
preserved_count=0

for id in $IDENTITIES; do
  if [ "$PRESERVE_ADMIN" = "true" ] && [ "$id" = "$ADMIN_ID" ]; then
    printf "${YELLOW}  ✓${NC} Preserving admin identity: ${id}\n"
    preserved_count=$((preserved_count + 1))
    continue
  fi

  printf "  - Deleting identity: ${id}...\n"
  if curl -s -X DELETE "${KRATOS_ADMIN_URL}/admin/identities/${id}" > /dev/null 2>&1; then
    deleted_count=$((deleted_count + 1))
  else
    printf "${RED}  ✗${NC} Failed to delete: ${id}\n"
  fi
done

printf "\n${GREEN}[OK]${NC} Cleared ${deleted_count} identities"
if [ "$PRESERVE_ADMIN" = "true" ]; then
  printf " (preserved ${preserved_count} admin)"
fi
printf "\n"
