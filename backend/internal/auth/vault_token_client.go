// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	vaultapi "github.com/hashicorp/vault/api"

	"github.com/retran/nexus/backend/internal/secrets"
)

const (
	defaultVersionCacheTTL = time.Minute
	jwtSigningMethod       = "RS256"
)

var reservedClaims = map[string]struct{}{
	"sub": {},
	"aud": {},
	"exp": {},
	"iat": {},
	"nbf": {},
	"iss": {},
	"jti": {},
}

// TokenClientConfig configures the Vault-backed JWT token client.
type TokenClientConfig struct {
	SecretsClient *secrets.Client
	Clock         ClockFunc

	SigningKeyName   string
	TransitMountPath string
	Issuer           string

	VersionCacheTTL time.Duration
}

// TokenClient issues JWT tokens by delegating signing to Vault Transit.
//
//nolint:govet // fieldalignment: layout chosen for readability; padding impact negligible.
type TokenClient struct {
	mu sync.Mutex

	secretsClient *secrets.Client
	now           ClockFunc

	versionExpires time.Time
	versionTTL     time.Duration

	defaultIssuer string
	mountPath     string
	keyName       string

	cachedVersion int
}

// IssueTokenRequest describes the desired JWT.
//
//nolint:govet // fieldalignment: readability preferred over micro-optimising padding.
type IssueTokenRequest struct {
	AdditionalClaims map[string]any
	Audience         []string
	NotBefore        *time.Time
	TTL              time.Duration
	Subject          string
	Issuer           string
}

// IssueTokenResponse contains the signed JWT and metadata.
type IssueTokenResponse struct {
	ExpiresAt  time.Time
	Token      string
	KeyVersion int
}

// NewTokenClient constructs a TokenClient.
func NewTokenClient(cfg *TokenClientConfig) (*TokenClient, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}
	if cfg.SecretsClient == nil {
		return nil, errors.New("secrets client is required")
	}
	if strings.TrimSpace(cfg.SigningKeyName) == "" {
		return nil, errors.New("signing key name is required")
	}

	mountPath := strings.Trim(cfg.TransitMountPath, "/")
	if mountPath == "" {
		mountPath = defaultTransitMountPath
	}

	versionTTL := cfg.VersionCacheTTL
	if versionTTL <= 0 {
		versionTTL = defaultVersionCacheTTL
	}

	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	return &TokenClient{
		secretsClient: cfg.SecretsClient,
		keyName:       cfg.SigningKeyName,
		mountPath:     mountPath,
		defaultIssuer: cfg.Issuer,
		versionTTL:    versionTTL,
		now:           clock,
	}, nil
}

// IssueToken authenticates to Vault via AppRole, signs the JWT using Transit,
// and returns the serialized token string.
func (c *TokenClient) IssueToken(ctx context.Context, req *IssueTokenRequest) (*IssueTokenResponse, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}
	if strings.TrimSpace(req.Subject) == "" {
		return nil, errors.New("subject is required")
	}
	if req.TTL <= 0 {
		return nil, errors.New("ttl must be greater than zero")
	}
	if err := validateAdditionalClaims(req.AdditionalClaims); err != nil {
		return nil, err
	}

	now := c.now()
	client, err := c.secretsClient.APIClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("obtain vault client: %w", err)
	}

	version, err := c.getSigningVersion(ctx, client)
	if err != nil {
		return nil, err
	}

	claims := jwt.MapClaims{
		"sub": req.Subject,
		"iat": now.Unix(),
		"exp": now.Add(req.TTL).Unix(),
	}

	if len(req.Audience) > 0 {
		claims["aud"] = req.Audience
	}

	issuer := strings.TrimSpace(req.Issuer)
	if issuer == "" {
		issuer = c.defaultIssuer
	}
	if issuer != "" {
		claims["iss"] = issuer
	}

	if req.NotBefore != nil {
		claims["nbf"] = req.NotBefore.UTC().Unix()
	}

	for k, v := range req.AdditionalClaims {
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = c.kidForVersion(version)
	token.Header["alg"] = jwtSigningMethod

	signingInput, err := token.SigningString()
	if err != nil {
		return nil, fmt.Errorf("build signing string: %w", err)
	}

	signature, err := c.sign(ctx, client, version, signingInput)
	if err != nil {
		return nil, err
	}

	jwtString := fmt.Sprintf("%s.%s", signingInput, signature)

	return &IssueTokenResponse{
		Token:      jwtString,
		KeyVersion: version,
		ExpiresAt:  now.Add(req.TTL),
	}, nil
}

func (c *TokenClient) getSigningVersion(ctx context.Context, client *vaultapi.Client) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	if c.cachedVersion > 0 && c.versionExpires.After(now) {
		return c.cachedVersion, nil
	}

	data, err := c.fetchTransitKeyData(ctx, client)
	if err != nil {
		return 0, err
	}

	latestRaw, ok := data["latest_version"]
	if !ok {
		return 0, fmt.Errorf("transit key %q metadata missing latest_version", c.keyName)
	}

	latestVersion, err := parseInt(latestRaw)
	if err != nil {
		return 0, fmt.Errorf("parse latest_version: %w", err)
	}

	c.cachedVersion = latestVersion
	c.versionExpires = now.Add(c.versionTTL)
	return latestVersion, nil
}

func (c *TokenClient) fetchTransitKeyData(ctx context.Context, client *vaultapi.Client) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/keys/%s", c.mountPath, c.keyName)
	secret, err := client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("read transit key %q: %w", c.keyName, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("transit key %q not found", c.keyName)
	}
	return secret.Data, nil
}

func (c *TokenClient) sign(ctx context.Context, client *vaultapi.Client, version int, signingInput string) (string, error) {
	path := fmt.Sprintf("%s/sign/%s", c.mountPath, c.keyName)
	payload := map[string]interface{}{
		"input":               base64.StdEncoding.EncodeToString([]byte(signingInput)),
		"key_version":         version,
		"prehashed":           false,
		"signature_algorithm": "pkcs1v15",
	}

	resp, err := client.Logical().WriteWithContext(ctx, path, payload)
	if err != nil {
		return "", fmt.Errorf("vault sign request: %w", err)
	}
	if resp == nil || resp.Data == nil {
		return "", errors.New("vault sign response missing data")
	}

	rawSignature, ok := resp.Data["signature"].(string)
	if !ok || strings.TrimSpace(rawSignature) == "" {
		return "", errors.New("vault sign response missing signature")
	}

	return formatJWSSignature(rawSignature, version)
}

func (c *TokenClient) kidForVersion(version int) string {
	return fmt.Sprintf("%s-v%d", c.keyName, version)
}

func validateAdditionalClaims(claims map[string]any) error {
	for k := range claims {
		if _, reserved := reservedClaims[k]; reserved {
			return fmt.Errorf("additional claim %q is reserved", k)
		}
	}
	return nil
}

func parseInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("convert json number: %w", err)
		}
		return int(i), nil
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse integer %q: %w", v, err)
		}
		return int(i), nil
	default:
		return 0, fmt.Errorf("unsupported number type %T", value)
	}
}

func formatJWSSignature(signature string, expectedVersion int) (string, error) {
	parts := strings.Split(signature, ":")
	if len(parts) != 3 {
		return "", fmt.Errorf("unexpected signature format")
	}
	versionPart := parts[1]
	if versionPart != fmt.Sprintf("v%d", expectedVersion) {
		return "", fmt.Errorf("vault used unexpected key version %s", versionPart)
	}
	rawSig, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("decode vault signature: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(rawSig), nil
}
