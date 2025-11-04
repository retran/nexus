package data

import "github.com/retran/nexus/backend/internal/repository/postgres"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver provides GraphQL resolver dependencies for the data API.
type Resolver struct {
	Queries *postgres.Queries
}
