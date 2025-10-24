// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package resolvers

import "github.com/retran/nexus/backend/internal/repository/postgres"

type Resolver struct {
	Queries *postgres.Queries
}
