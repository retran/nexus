-- Set comment to schema: "public"
COMMENT ON SCHEMA public IS 'Nexus application schema';
-- Create enum type "user_role"
CREATE TYPE public.user_role AS ENUM ('none', 'member', 'admin');
-- Create enum type "audit_event_type"
CREATE TYPE public.audit_event_type AS ENUM (
  'user_created',
  'user_updated',
  'user_deleted',
  'login_success',
  'login_failed',
  'logout',
  'settings_updated',
  'role_changed'
);
-- Create "rate_limits" table
CREATE TABLE public.rate_limits (
  key CHARACTER VARYING(255) NOT NULL,
  attempts INTEGER NOT NULL DEFAULT 0,
  reset_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (key)
);
-- Create index "rate_limits_reset_at_idx" to table: "rate_limits"
CREATE INDEX rate_limits_reset_at_idx ON public.rate_limits (reset_at);
-- Set comment to table: "rate_limits"
COMMENT ON TABLE public.rate_limits IS 'Rate limiting data for API throttling';
-- Set comment to column: "key" on table: "rate_limits"
COMMENT ON COLUMN public.rate_limits.key IS 'Rate limit key, e.g., ''ip:192.168.1.1:endpoint:/api/users''';
-- Create "users" table
CREATE TABLE public.users (
  id UUID NOT NULL DEFAULT GEN_RANDOM_UUID(),
  kratos_identity_id UUID NOT NULL,
  email CHARACTER VARYING(255) NOT NULL,
  name CHARACTER VARYING(255) NULL,
  picture TEXT NULL,
  role public.user_role NOT NULL DEFAULT 'none',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id)
);
-- Create index "users_email_key" to table: "users"
CREATE UNIQUE INDEX users_email_key ON public.users (email);
-- Create index "users_kratos_identity_id_idx" to table: "users"
CREATE UNIQUE INDEX users_kratos_identity_id_idx ON public.users (kratos_identity_id);
-- Create index "users_role_idx" to table: "users"
CREATE INDEX users_role_idx ON public.users (role);
-- Set comment to table: "users"
COMMENT ON TABLE public.users IS 'Nexus users - cached from Kratos for performance';
-- Set comment to column: "kratos_identity_id" on table: "users"
COMMENT ON COLUMN public.users.kratos_identity_id IS 'Reference to Kratos identity (ory schema: identities.id)';
-- Set comment to column: "email" on table: "users"
COMMENT ON COLUMN public.users.email IS 'Cached from Kratos traits.email';
-- Set comment to column: "name" on table: "users"
COMMENT ON COLUMN public.users.name IS 'Display name cached from Kratos traits.name';
-- Set comment to column: "picture" on table: "users"
COMMENT ON COLUMN public.users.picture IS 'Profile picture URL cached from Kratos traits.picture';
-- Set comment to column: "role" on table: "users"
COMMENT ON COLUMN public.users.role IS 'User role: none (pending approval), member, admin. Cached from Kratos traits.role';
-- Create "audit_logs" table
CREATE TABLE public.audit_logs (
  id UUID NOT NULL DEFAULT GEN_RANDOM_UUID(),
  user_id UUID NULL,
  event_type public.audit_event_type NOT NULL,
  ip_address CHARACTER VARYING(45) NULL,
  user_agent TEXT NULL,
  metadata JSONB NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES public.users (id) ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Create index "audit_logs_created_at_idx" to table: "audit_logs"
CREATE INDEX audit_logs_created_at_idx ON public.audit_logs (created_at);
-- Create index "audit_logs_event_type_idx" to table: "audit_logs"
CREATE INDEX audit_logs_event_type_idx ON public.audit_logs (event_type);
-- Create index "audit_logs_user_id_idx" to table: "audit_logs"
CREATE INDEX audit_logs_user_id_idx ON public.audit_logs (user_id);
-- Set comment to table: "audit_logs"
COMMENT ON TABLE public.audit_logs IS 'Audit trail for user actions and security events';
