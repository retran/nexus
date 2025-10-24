// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

package graphql

import (
	"github.com/Khan/genqlient/graphql"
)

// NewClient creates a new GraphQL client.
func NewClient(endpoint string) graphql.Client {
	return graphql.NewClient(endpoint, nil)
}
