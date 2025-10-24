#!/bin/bash
# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: APACHE-2.0

# Test Kratos webhook handler

WEBHOOK_SECRET=${KRATOS_WEBHOOK_SECRET:-"change-me-in-production"}
GATEWAY_URL=${GATEWAY_URL:-"http://localhost:8080"}

echo "Testing Kratos webhook handler..."
echo "Gateway URL: $GATEWAY_URL"
echo "Webhook Secret: $WEBHOOK_SECRET"
echo ""

# Test payload
PAYLOAD='{
  "identity_id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "test@example.com",
  "name": {
    "first": "Test",
    "last": "User"
  },
  "picture": "https://example.com/avatar.jpg",
  "provider": "google",
  "provider_user_id": "google-123456"
}'

echo "Sending webhook request..."
echo "Payload:"
echo "$PAYLOAD" | jq .
echo ""

# Send request
RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Secret: $WEBHOOK_SECRET" \
  -d "$PAYLOAD" \
  "$GATEWAY_URL/api/webhooks/kratos/registration")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | head -n-1)

echo "HTTP Status: $HTTP_CODE"
echo "Response:"
echo "$BODY" | jq .
echo ""

if [ "$HTTP_CODE" = "201" ] || [ "$HTTP_CODE" = "200" ]; then
  echo "✅ Webhook test PASSED"

  # Query database to verify user was created
  echo ""
  echo "Checking database..."
  docker-compose -f docker-compose.dev.yaml exec -T postgres \
    psql -U admin -d nexus_db -c \
    "SELECT id, email, name, role, created_at FROM users WHERE email='test@example.com';"
else
  echo "❌ Webhook test FAILED"
  exit 1
fi
