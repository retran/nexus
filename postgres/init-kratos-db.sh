#!/bin/bash
# Copyright 2025 Andrew Vasilyev
# SPDX-License-Identifier: Apache-2.0

set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE kratos_db;
    GRANT ALL PRIVILEGES ON DATABASE kratos_db TO $POSTGRES_USER;
EOSQL
