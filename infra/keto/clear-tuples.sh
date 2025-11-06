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

KETO_READ_URL="${KETO_READ_URL:-http://localhost:4466}"
KETO_WRITE_URL="${KETO_WRITE_URL:-http://localhost:4467}"

printf "${BLUE}[INFO]${NC} Fetching all relation tuples from Keto...\n"

# Fetch all relation tuples across all namespaces
# We need to query each namespace separately since Keto doesn't have a "list all" endpoint
NAMESPACES=("households" "users" "files")

total_deleted=0

for namespace in "${NAMESPACES[@]}"; do
  printf "${BLUE}[INFO]${NC} Processing namespace: ${namespace}...\n"

  # Fetch tuples for this namespace
  TUPLES=$(curl -s "${KETO_READ_URL}/relation-tuples?namespace=${namespace}" | \
    jq -r '.relation_tuples[]? | @json')

  if [ -z "$TUPLES" ]; then
    printf "${YELLOW}  ✓${NC} No tuples found in ${namespace}\n"
    continue
  fi

  deleted_count=0

  while IFS= read -r tuple; do
    if [ -z "$tuple" ]; then
      continue
    fi

    # Extract tuple components
    ns=$(echo "$tuple" | jq -r '.namespace')
    obj=$(echo "$tuple" | jq -r '.object')
    rel=$(echo "$tuple" | jq -r '.relation')
    subj_id=$(echo "$tuple" | jq -r '.subject_id // empty')
    subj_set=$(echo "$tuple" | jq -r '.subject_set // empty')

    # Build query parameters
    query="namespace=${ns}&object=${obj}&relation=${rel}"

    if [ -n "$subj_id" ]; then
      query="${query}&subject_id=${subj_id}"
    elif [ -n "$subj_set" ]; then
      subj_ns=$(echo "$tuple" | jq -r '.subject_set.namespace')
      subj_obj=$(echo "$tuple" | jq -r '.subject_set.object')
      subj_rel=$(echo "$tuple" | jq -r '.subject_set.relation')
      query="${query}&subject_set.namespace=${subj_ns}&subject_set.object=${subj_obj}&subject_set.relation=${subj_rel}"
    fi

    printf "  - Deleting: ${ns}:${obj}#${rel}...\n"

    if curl -s -X DELETE "${KETO_WRITE_URL}/admin/relation-tuples?${query}" > /dev/null 2>&1; then
      deleted_count=$((deleted_count + 1))
    else
      printf "${RED}  ✗${NC} Failed to delete tuple\n"
    fi
  done <<< "$TUPLES"

  printf "${GREEN}  ✓${NC} Deleted ${deleted_count} tuples from ${namespace}\n"
  total_deleted=$((total_deleted + deleted_count))
done

printf "\n${GREEN}[OK]${NC} Cleared ${total_deleted} permission tuples total\n"
