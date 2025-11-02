// Package resolvers contains GraphQL resolver implementations.
package resolvers

import "github.com/retran/nexus/backend/internal/repository/postgres"

// Resolver provides gqlgen with the configured data access dependencies.
type Resolver struct {
	Queries *postgres.Queries
}
