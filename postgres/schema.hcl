# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: APACHE-2.0

schema "public" {
  comment = "Nexus application schema"
}

enum "user_role" {
  schema = schema.public
  values = ["none", "member", "admin"]
  comment = "User roles: none = inactive/pending, member = regular user, admin = administrator"
}

enum "audit_event_type" {
  schema = schema.public
  values = [
    "user_created",
    "user_updated",
    "user_deleted",
    "login_success",
    "login_failed",
    "logout",
    "settings_updated",
    "role_changed"
  ]
}

# Users table - cached data from Kratos for quick access
table "users" {
  schema = schema.public
  comment = "Nexus users - cached from Kratos for performance"

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "kratos_identity_id" {
    type    = uuid
    null    = false
    comment = "Reference to Kratos identity (ory schema: identities.id)"
  }
  column "email" {
    type    = varchar(255)
    null    = false
    comment = "Cached from Kratos traits.email"
  }
  column "name" {
    type    = varchar(255)
    null    = true
    comment = "Display name cached from Kratos traits.name"
  }
  column "picture" {
    type    = text
    null    = true
    comment = "Profile picture URL cached from Kratos traits.picture"
  }
  column "role" {
    type    = sql("public.user_role")
    default = "none"
    null    = false
    comment = "User role: none (pending approval), member, admin. Cached from Kratos traits.role"
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
    null    = false
  }
  column "updated_at" {
    type    = timestamptz
    default = sql("now()")
    null    = false
  }

  primary_key {
    columns = [column.id]
  }

  index "users_email_key" {
    columns = [column.email]
    unique  = true
  }

  index "users_kratos_identity_id_idx" {
    columns = [column.kratos_identity_id]
    unique  = true
  }

  index "users_role_idx" {
    columns = [column.role]
  }
}

# Audit logs for security and compliance
table "audit_logs" {
  schema = schema.public
  comment = "Audit trail for user actions and security events"

  column "id" {
    type    = uuid
    default = sql("gen_random_uuid()")
  }
  column "user_id" {
    type = uuid
    null = true
  }
  column "event_type" {
    type = sql("public.audit_event_type")
    null = false
  }
  column "ip_address" {
    type = varchar(45)
    null = true
  }
  column "user_agent" {
    type = text
    null = true
  }
  column "metadata" {
    type = jsonb
    null = true
  }
  column "created_at" {
    type    = timestamptz
    default = sql("now()")
    null    = false
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_user" {
    columns     = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete   = SET_NULL
  }

  index "audit_logs_user_id_idx" {
    columns = [column.user_id]
  }

  index "audit_logs_event_type_idx" {
    columns = [column.event_type]
  }

  index "audit_logs_created_at_idx" {
    columns = [column.created_at]
  }
}

# Rate limiting for API endpoints
table "rate_limits" {
  schema = schema.public
  comment = "Rate limiting data for API throttling"

  column "key" {
    type    = varchar(255)
    null    = false
    comment = "Rate limit key, e.g., 'ip:192.168.1.1:endpoint:/api/users'"
  }
  column "attempts" {
    type    = integer
    null    = false
    default = 0
  }
  column "reset_at" {
    type = timestamptz
    null = false
  }

  primary_key {
    columns = [column.key]
  }

  index "rate_limits_reset_at_idx" {
    columns = [column.reset_at]
  }
}
