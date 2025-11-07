# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: Apache-2.0

# Atlas configuration for database migrations

env "local" {
  src = "file://schema.hcl"
  url = getenv("DATABASE_URL")

  migration {
    dir = "file://migrations"
  }

  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

env "dev" {
  src = "file://schema.hcl"
  url = getenv("DATABASE_URL")

  migration {
    dir = "file://migrations"
  }
}

env "prod" {
  src = "file://schema.hcl"
  url = getenv("DATABASE_URL")

  migration {
    dir = "file://migrations"
  }
}
