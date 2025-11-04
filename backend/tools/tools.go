//go:build tools

package tools

import (
	_ "github.com/99designs/gqlgen"
	_ "github.com/99designs/gqlgen/graphql/introspection"
	_ "github.com/Khan/genqlient"
	_ "github.com/air-verse/air"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
)
