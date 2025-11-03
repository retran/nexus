// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package secrets provides helpers for retrieving configuration from Vault KV
// using AppRole authentication.
package secrets

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

const (
	defaultAuthMountPath = "approle"
	defaultKVMountPath   = "kv"
	defaultRenewalBuffer = time.Minute
)

// ClockFunc returns the current time. It allows deterministic testing by
// injecting a fixed clock.
type ClockFunc func() time.Time

// Config configures a Vault KV client that authenticates using AppRole
// credentials.
type Config struct {
	HTTPClient       *http.Client
	TLSConfig        *vaultapi.TLSConfig
	Clock            ClockFunc
	Address          string
	RoleID           string
	SecretID         string
	AuthMountPath    string
	KVMountPath      string
	Namespace        string
	TokenRenewBuffer time.Duration
}

// Client wraps a Vault API client that performs AppRole login and reads KV v2
// secrets.
type Client struct {
	tokenExpiry      time.Time
	apiClient        *vaultapi.Client
	now              ClockFunc
	authPath         string
	kvPathPrefix     string
	cfg              Config
	tokenRenewBuffer time.Duration
	mu               sync.Mutex
}

// NewClient creates a Vault secrets client from the provided configuration.
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("vault config is required")
	}

	config := *cfg

	if strings.TrimSpace(config.Address) == "" {
		return nil, errors.New("vault address is required")
	}
	if strings.TrimSpace(config.RoleID) == "" {
		return nil, errors.New("vault AppRole role ID is required")
	}
	if strings.TrimSpace(config.SecretID) == "" {
		return nil, errors.New("vault AppRole secret ID is required")
	}

	authMount := strings.Trim(config.AuthMountPath, "/")
	if authMount == "" {
		authMount = defaultAuthMountPath
	}
	kvMount := strings.Trim(config.KVMountPath, "/")
	if kvMount == "" {
		kvMount = defaultKVMountPath
	}

	now := config.Clock
	if now == nil {
		now = time.Now
	}

	renewBuffer := config.TokenRenewBuffer
	if renewBuffer <= 0 {
		renewBuffer = defaultRenewalBuffer
	}

	vaultCfg := vaultapi.DefaultConfig()
	vaultCfg.Address = config.Address

	if config.HTTPClient != nil {
		vaultCfg.HttpClient = config.HTTPClient
	}

	if config.TLSConfig != nil {
		if err := vaultCfg.ConfigureTLS(config.TLSConfig); err != nil {
			return nil, fmt.Errorf("configure vault tls: %w", err)
		}
	}

	apiClient, err := vaultapi.NewClient(vaultCfg)
	if err != nil {
		return nil, fmt.Errorf("create vault client: %w", err)
	}

	if config.Namespace != "" {
		apiClient.SetNamespace(config.Namespace)
	}

	return &Client{
		cfg:              config,
		apiClient:        apiClient,
		now:              now,
		tokenRenewBuffer: renewBuffer,
		authPath:         fmt.Sprintf("auth/%s/login", authMount),
		kvPathPrefix:     fmt.Sprintf("%s/data", kvMount),
	}, nil
}

// APIClient returns the authenticated Vault API client. It ensures the client
// token is valid before returning.
func (c *Client) APIClient(ctx context.Context) (*vaultapi.Client, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}
	return c.apiClient, nil
}

// Read fetches a secret from the configured KV mount. The path is relative to
// the mount, e.g. "services/gateway".
func (c *Client) Read(ctx context.Context, path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("secret path is required")
	}

	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}

	trimmedPath := strings.Trim(path, "/")
	vaultPath := fmt.Sprintf("%s/%s", c.kvPathPrefix, trimmedPath)

	secret, err := c.apiClient.Logical().ReadWithContext(ctx, vaultPath)
	if err != nil {
		return nil, fmt.Errorf("read vault secret %q: %w", trimmedPath, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("vault secret %q not found", trimmedPath)
	}

	rawData, ok := secret.Data["data"]
	if !ok {
		return nil, fmt.Errorf("vault secret %q missing data payload", trimmedPath)
	}

	dataMap, ok := rawData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("vault secret %q has unexpected data type %T", trimmedPath, rawData)
	}

	result := make(map[string]string, len(dataMap))
	for k, v := range dataMap {
		result[k] = fmt.Sprint(v)
	}

	return result, nil
}

// ReadService fetches secrets for a specific service under "services/<name>".
func (c *Client) ReadService(ctx context.Context, service string) (map[string]string, error) {
	if strings.TrimSpace(service) == "" {
		return nil, errors.New("service name is required")
	}
	return c.Read(ctx, fmt.Sprintf("services/%s", strings.Trim(service, "/")))
}

// ReadShared fetches shared secrets under "shared/<name>".
func (c *Client) ReadShared(ctx context.Context, name string) (map[string]string, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("shared secret name is required")
	}
	return c.Read(ctx, fmt.Sprintf("shared/%s", strings.Trim(name, "/")))
}

func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.tokenValidLocked() {
		return nil
	}

	loginData := map[string]interface{}{
		"role_id":   c.cfg.RoleID,
		"secret_id": c.cfg.SecretID,
	}

	secret, err := c.apiClient.Logical().WriteWithContext(ctx, c.authPath, loginData)
	if err != nil {
		return fmt.Errorf("vault approle login failed: %w", err)
	}
	if secret == nil || secret.Auth == nil {
		return errors.New("vault approle login returned no auth info")
	}
	if secret.Auth.ClientToken == "" {
		return errors.New("vault approle login returned empty client token")
	}

	c.apiClient.SetToken(secret.Auth.ClientToken)
	lease := time.Duration(secret.Auth.LeaseDuration) * time.Second
	if lease <= 0 {
		// Non-expiring token. Set a distant future expiry to avoid repeated logins.
		lease = 365 * 24 * time.Hour
	}

	c.tokenExpiry = c.now().Add(lease)
	return nil
}

func (c *Client) tokenValidLocked() bool {
	if c.tokenExpiry.IsZero() {
		return false
	}
	nowPlusBuffer := c.now().Add(c.tokenRenewBuffer)
	return nowPlusBuffer.Before(c.tokenExpiry)
}
