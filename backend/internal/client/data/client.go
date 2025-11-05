// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package data exposes helpers for interacting with GraphQL data API.
package data

import (
	"github.com/Khan/genqlient/graphql"
)

// NewClient creates a new GraphQL client.
func NewClient(endpoint string) graphql.Client {
	return graphql.NewClient(endpoint, nil)
}

// NewClientWithHTTPClient creates a new GraphQL client using the provided HTTP client.
func NewClientWithHTTPClient(endpoint string, httpClient graphql.Doer) graphql.Client {
	return graphql.NewClient(endpoint, httpClient)
}
