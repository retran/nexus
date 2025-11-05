#!/bin/bash
# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: Apache-2.0

# Generate .env file from .env.example with secure random secrets for development

set -e

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"
ENV_EXAMPLE="$ROOT_DIR/.env.example"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}[INFO]${NC} Generating .env file for development..."

# Check if .env already exists
if [ -f "$ENV_FILE" ]; then
    echo -e "${YELLOW}[WARN]${NC} .env file already exists"
    read -p "Do you want to overwrite it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}[INFO]${NC} Keeping existing .env file"
        exit 0
    fi
fi

# Check if .env.example exists
if [ ! -f "$ENV_EXAMPLE" ]; then
    echo -e "${RED}[ERROR]${NC} .env.example not found"
    exit 1
fi

# Generate random secrets
generate_secret_32() {
    # Generate 32 character hex string (16 bytes)
    openssl rand -hex 16
}

generate_secret_64() {
    # Generate 64 character hex string (32 bytes)
    openssl rand -hex 32
}

generate_password() {
    openssl rand -base64 24 | tr -d "=+/" | cut -c1-20
}

echo -e "${BLUE}[INFO]${NC} Generating random secrets..."

# Generate all required secrets
POSTGRES_PASSWORD=$(generate_password)
REDIS_PASSWORD=$(generate_password)
KRATOS_COOKIE_SECRET=$(generate_secret_32)
KRATOS_CIPHER_SECRET=$(generate_secret_32)
COOKIE_SECRET=$(generate_secret_64)
CSRF_COOKIE_SECRET=$(generate_secret_64)
KRATOS_WEBHOOK_SECRET=$(generate_secret_64)
OATHKEEPER_SHARED_SECRET=$(generate_secret_64)
VAULT_ROOT_TOKEN=$(generate_secret_64)

# Copy .env.example to .env
cp "$ENV_EXAMPLE" "$ENV_FILE"

# Replace secrets in .env
echo -e "${BLUE}[INFO]${NC} Replacing secrets in .env..."

# PostgreSQL password
sed -i.bak "s/POSTGRES_PASSWORD=SUPER_SECRET_PASSWORD/POSTGRES_PASSWORD=$POSTGRES_PASSWORD/" "$ENV_FILE"
sed -i.bak "s|DATABASE_URL=postgres://admin:SUPER_SECRET_PASSWORD@|DATABASE_URL=postgres://admin:$POSTGRES_PASSWORD@|" "$ENV_FILE"

# Redis password
sed -i.bak "s/REDIS_PASSWORD=redis_password/REDIS_PASSWORD=$REDIS_PASSWORD/" "$ENV_FILE"

# Update DSN with new password (for Kratos)
sed -i.bak "s|DSN=postgres://admin:\${POSTGRES_PASSWORD}@|DSN=postgres://admin:$POSTGRES_PASSWORD@|" "$ENV_FILE"

# Update KETO_DSN with new password
sed -i.bak "s|KETO_DSN=postgres://admin:\${POSTGRES_PASSWORD}@|KETO_DSN=postgres://admin:$POSTGRES_PASSWORD@|" "$ENV_FILE"

# Kratos secrets
sed -i.bak "s/KRATOS_COOKIE_SECRET=CHANGEME_SECRET_EXACTLY_32CHARSS/KRATOS_COOKIE_SECRET=$KRATOS_COOKIE_SECRET/" "$ENV_FILE"
sed -i.bak "s/KRATOS_CIPHER_SECRET=CHANGEME_SECRET_EXACTLY_32CHARSS/KRATOS_CIPHER_SECRET=$KRATOS_CIPHER_SECRET/" "$ENV_FILE"

# Kratos Selfservice UI secrets
sed -i.bak "s/COOKIE_SECRET=CHANGE_ME_GENERATE_WITH_OPENSSL_RAND_HEX_32_FOR_COOKIE/COOKIE_SECRET=$COOKIE_SECRET/" "$ENV_FILE"
sed -i.bak "s/CSRF_COOKIE_SECRET=CHANGE_ME_GENERATE_WITH_OPENSSL_RAND_HEX_32_FOR_CSRF/CSRF_COOKIE_SECRET=$CSRF_COOKIE_SECRET/" "$ENV_FILE"

# Webhook and Oathkeeper secrets
sed -i.bak "s/KRATOS_WEBHOOK_SECRET=CHANGE_ME_GENERATE_WITH_OPENSSL_RAND_HEX_32/KRATOS_WEBHOOK_SECRET=$KRATOS_WEBHOOK_SECRET/" "$ENV_FILE"
sed -i.bak "s/OATHKEEPER_SHARED_SECRET=CHANGE_ME_GENERATE_WITH_OPENSSL_RAND_HEX_32/OATHKEEPER_SHARED_SECRET=$OATHKEEPER_SHARED_SECRET/" "$ENV_FILE"

# Vault secrets
sed -i.bak "s/VAULT_ROOT_TOKEN=dev-root-token/VAULT_ROOT_TOKEN=$VAULT_ROOT_TOKEN/" "$ENV_FILE"

# Remove backup file
rm -f "$ENV_FILE.bak"

echo -e "${GREEN}[OK]${NC} .env file generated successfully!"
echo ""
echo -e "${YELLOW}[NOTE]${NC} Generated secrets:"
echo -e "  POSTGRES_PASSWORD: $POSTGRES_PASSWORD"
echo -e "  REDIS_PASSWORD: $REDIS_PASSWORD"
echo -e "  VAULT_ROOT_TOKEN: $VAULT_ROOT_TOKEN"
echo ""
echo -e "${YELLOW}[TODO]${NC} You still need to configure:"
echo -e "  - GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET (if using Google OAuth)"
echo -e "  - APPLE_CLIENT_ID, APPLE_CLIENT_SECRET, APPLE_TEAM_ID (if using Apple Sign In)"
echo -e "  - DOMAIN (for production deployment)"
echo ""
echo -e "${GREEN}[NEXT]${NC} Run: task bootstrap:dev"
