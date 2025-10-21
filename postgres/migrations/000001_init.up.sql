-- Copyright 2025 Andrew Vasilyev
-- SPDX-License-Identifier: APACHE-2.0
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT GEN_RANDOM_UUID(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  oauth_provider VARCHAR(50) NOT NULL CHECK (oauth_provider IN ('google', 'apple')),
  oauth_id VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL,
  full_name VARCHAR(255) NOT NULL,
  picture TEXT,
  user_role TEXT NOT NULL DEFAULT 'member' CHECK (user_role IN ('admin', 'member')),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  UNIQUE (oauth_provider, oauth_id),
  UNIQUE (oauth_provider, email)
);

CREATE INDEX idx_users_oauth_provider_id ON users (oauth_provider, oauth_id);

CREATE INDEX idx_users_email ON users (email);
