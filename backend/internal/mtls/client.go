// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package mtls provides utilities for configuring mTLS clients.
package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
)

const (
	defaultCAPath   = "/secrets/vault-ca.pem"
	defaultCertPath = "/secrets/tls.crt"
	defaultKeyPath  = "/secrets/tls.key"
)

// LoadClientTLSConfig loads TLS configuration for mTLS client connections.
// It reads the CA certificate, client certificate, and client key from the specified paths.
// If paths are empty, it uses default paths from Vault Agent.
func LoadClientTLSConfig(caPath, certPath, keyPath string) (*tls.Config, error) {
	if caPath == "" {
		caPath = defaultCAPath
	}
	if certPath == "" {
		certPath = defaultCertPath
	}
	if keyPath == "" {
		keyPath = defaultKeyPath
	}

	// Load CA certificate for server validation
	// #nosec G304 -- caPath is controlled and comes from container environment
	caCert, err := os.ReadFile(caPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate from %s: %w", caPath, err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Load client certificate and key
	clientCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate and key: %w", err)
	}

	return &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{clientCert},
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// NewHTTPClient creates a new HTTP client configured for mTLS.
// It uses the default certificate paths from Vault Agent.
func NewHTTPClient() (*http.Client, error) {
	tlsConfig, err := LoadClientTLSConfig("", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to load mTLS config: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: transport,
	}, nil
}
