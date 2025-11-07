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


-- Create "users" table
CREATE TABLE public.users (
  id UUID NOT NULL DEFAULT GEN_RANDOM_UUID(),
  kratos_identity_id UUID NOT NULL,
  email CHARACTER VARYING(255) NOT NULL,
  display_name CHARACTER VARYING(255) NULL,
  picture TEXT NULL,
  user_role public.USER_ROLE NOT NULL DEFAULT 'none',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id)
);


-- Create index "users_email_key" to table: "users"
CREATE UNIQUE INDEX users_email_key ON public.users (email);


-- Create index "users_kratos_identity_id_idx" to table: "users"
CREATE UNIQUE INDEX users_kratos_identity_id_idx ON public.users (kratos_identity_id);


-- Create index "users_user_role_idx" to table: "users"
CREATE INDEX users_user_role_idx ON public.users (user_role);


-- Set comment to table: "users"
COMMENT ON TABLE public.users IS 'Nexus users - cached from Kratos for performance';


-- Set comment to column: "kratos_identity_id" on table: "users"
COMMENT ON COLUMN public.users.kratos_identity_id IS 'Reference to Kratos identity (ory schema: identities.id)';


-- Set comment to column: "email" on table: "users"
COMMENT ON COLUMN public.users.email IS 'Cached from Kratos traits.email';


-- Set comment to column: "display_name" on table: "users"
COMMENT ON COLUMN public.users.display_name IS 'Display name cached from Kratos traits.name';


-- Set comment to column: "picture" on table: "users"
COMMENT ON COLUMN public.users.picture IS 'Profile picture URL cached from Kratos traits.picture';


-- Set comment to column: "user_role" on table: "users"
COMMENT ON COLUMN public.users.user_role IS 'User role: none (pending approval), member, admin. Cached from Kratos traits.role';


-- Create "audit_logs" table
CREATE TABLE public.audit_logs (
  id UUID NOT NULL DEFAULT GEN_RANDOM_UUID(),
  user_id UUID NULL,
  event_type public.AUDIT_EVENT_TYPE NOT NULL,
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
