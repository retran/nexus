// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package graphql exposes helpers for interacting with GraphQL services.
package graphql

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

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

// NewMTLSClient creates a GraphQL client with mTLS configuration.
func NewMTLSClient(endpoint string) (graphql.Client, error) {
	tlsConfig, err := loadMTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load mTLS config: %w", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return graphql.NewClient(endpoint, httpClient), nil
}

func loadMTLSConfig() (*tls.Config, error) {
	// Load CA certificate
	caCert, err := os.ReadFile("/secrets/vault-ca.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Load client certificate
	cert, err := tls.LoadX509KeyPair("/secrets/tls.crt", "/secrets/tls.key")
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	return &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}, nil
}
