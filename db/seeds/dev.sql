-- Nexus Development Seed Data
-- This file populates the database with sample audit logs and other non-identity data
--
-- NOTE: Users are NOT seeded here!
-- Users are automatically created via Kratos webhook when identities are created.
-- See ory/seeds/*.json for identity definitions that trigger user creation.
--
-- Copyright 2025 Andrew Vasilyev. Licensed under the Apache License, Version 2.0.

-- =============================================================================
-- AUDIT LOGS (Sample Events)
-- =============================================================================
-- These are example audit events for testing.
-- In reality, audit logs are created by application activity.

-- System initialization event
INSERT INTO audit_logs (
  user_id,
  event_type,
  metadata,
  ip_address,
  user_agent,
  created_at
) VALUES
  (
    NULL,
    'settings_updated',
    '{"component": "system", "changed_settings": ["timezone", "locale"], "values": {"timezone": "UTC", "locale": "en-US"}}'::JSONB,
    '127.0.0.1',
    'Nexus Internal',
    NOW() - INTERVAL '30 days'
  );

-- =============================================================================
-- VERIFICATION
-- =============================================================================

DO $$
DECLARE
  user_count INTEGER;
  audit_count INTEGER;
BEGIN
  SELECT COUNT(*) INTO user_count FROM users;
  SELECT COUNT(*) INTO audit_count FROM audit_logs;

  RAISE NOTICE '✅ Development seed data loaded:';
  RAISE NOTICE '   Users: % (created via Kratos webhook)', user_count;
  RAISE NOTICE '   Audit Logs: %', audit_count;
END $$;
