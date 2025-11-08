// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration utilities.
package config

import (
	"context"
	"fmt"
	"os"

	vault "github.com/hashicorp/vault/api"
)

// VaultClient wraps the Vault client for configuration loading.
type VaultClient struct {
	client *vault.Client
}

// NewVaultClient creates a new Vault client for loading secrets.
// In dev mode, it uses VAULT_TOKEN. In production, use AppRole credentials.
func NewVaultClient() (*VaultClient, error) {
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		return nil, fmt.Errorf("VAULT_ADDR environment variable is required")
	}

	config := vault.DefaultConfig()
	config.Address = vaultAddr

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Dev mode: Use token-based authentication
	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken != "" {
		client.SetToken(vaultToken)
		return &VaultClient{client: client}, nil
	}

	// Production mode: Use AppRole authentication
	roleID := os.Getenv("VAULT_ROLE_ID")
	secretID := os.Getenv("VAULT_SECRET_ID")
	if roleID == "" || secretID == "" {
		return nil, fmt.Errorf("VAULT_ROLE_ID and VAULT_SECRET_ID are required for production mode")
	}

	// Authenticate with AppRole
	data := map[string]interface{}{
		"role_id":   roleID,
		"secret_id": secretID,
	}

	resp, err := client.Logical().Write("auth/approle/login", data)
	if err != nil {
		return nil, fmt.Errorf("failed to login with AppRole: %w", err)
	}

	if resp == nil || resp.Auth == nil {
		return nil, fmt.Errorf("AppRole login response is invalid")
	}

	client.SetToken(resp.Auth.ClientToken)

	return &VaultClient{client: client}, nil
}

// GetSharedSecret retrieves a shared secret from Vault.
// Returns the "current" value from the secret rotation structure.
func (vc *VaultClient) GetSharedSecret(ctx context.Context, name string) (string, error) {
	secret, err := vc.client.KVv2("kv").Get(ctx, fmt.Sprintf("shared/%s", name))
	if err != nil {
		return "", fmt.Errorf("failed to read shared secret %q: %w", name, err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("shared secret %q not found", name)
	}

	current, ok := secret.Data["current"].(string)
	if !ok || current == "" {
		return "", fmt.Errorf("shared secret %q does not have a 'current' value", name)
	}

	return current, nil
}

// GetServiceSecret retrieves a service-specific secret field from Vault.
// Example: GetServiceSecret(ctx, "kratos/encryption", "cookie").
func (vc *VaultClient) GetServiceSecret(ctx context.Context, path, field string) (string, error) {
	secret, err := vc.client.KVv2("kv").Get(ctx, fmt.Sprintf("services/%s", path))
	if err != nil {
		return "", fmt.Errorf("failed to read service secret %q: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("service secret %q not found", path)
	}

	value, ok := secret.Data[field].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("service secret %q does not have field %q", path, field)
	}

	return value, nil
}

// GetServiceSecrets retrieves all fields from a service secret path.
func (vc *VaultClient) GetServiceSecrets(ctx context.Context, path string) (map[string]interface{}, error) {
	secret, err := vc.client.KVv2("kv").Get(ctx, fmt.Sprintf("services/%s", path))
	if err != nil {
		return nil, fmt.Errorf("failed to read service secrets %q: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("service secrets %q not found", path)
	}

	return secret.Data, nil
}

// GetPostgresPassword retrieves the PostgreSQL password from shared secrets.
func (vc *VaultClient) GetPostgresPassword(ctx context.Context) (string, error) {
	return vc.GetSharedSecret(ctx, "postgres")
}

// GetRedisPassword retrieves the Redis password from shared secrets.
func (vc *VaultClient) GetRedisPassword(ctx context.Context) (string, error) {
	return vc.GetSharedSecret(ctx, "redis")
}

// GetWebhookSecret retrieves the webhook shared secret.
func (vc *VaultClient) GetWebhookSecret(ctx context.Context) (string, error) {
	return vc.GetSharedSecret(ctx, "webhook")
}

// GetOathkeeperSecret retrieves the Oathkeeper shared secret.
func (vc *VaultClient) GetOathkeeperSecret(ctx context.Context) (string, error) {
	return vc.GetSharedSecret(ctx, "oathkeeper")
}

// GetUnsplashAccessKey retrieves the Unsplash access key from service secrets.
func (vc *VaultClient) GetUnsplashAccessKey(ctx context.Context) (string, error) {
	return vc.GetServiceSecret(ctx, "photos/unsplash", "access_key")
}
