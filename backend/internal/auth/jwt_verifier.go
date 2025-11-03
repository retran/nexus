// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package auth provides utilities for verifying JWTs signed by Vault Transit.
package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	vaultapi "github.com/hashicorp/vault/api"
)

const (
	defaultTransitMountPath = "transit"
	defaultCacheTTL         = 5 * time.Minute
)

// ClockFunc returns the current time. It allows deterministic testing.
type ClockFunc func() time.Time

// JWTVerifierConfig configures the JWT verifier.
type JWTVerifierConfig struct {
	VaultClient      *vaultapi.Client
	Clock            ClockFunc
	TransitMountPath string
	KeyName          string
	ParserOptions    []jwt.ParserOption
	CacheTTL         time.Duration
}

// JWTVerifier verifies JWT tokens signed by Vault Transit.
type JWTVerifier struct {
	cacheExpiry   time.Time
	client        *vaultapi.Client
	cache         map[string]crypto.PublicKey
	now           ClockFunc
	mountPath     string
	keyName       string
	parserOptions []jwt.ParserOption
	cacheTTL      time.Duration
	mu            sync.RWMutex
}

// NewJWTVerifier creates a new JWT verifier.
func NewJWTVerifier(cfg *JWTVerifierConfig) (*JWTVerifier, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	config := *cfg

	if config.VaultClient == nil {
		return nil, errors.New("vault client is required")
	}
	if strings.TrimSpace(config.KeyName) == "" {
		return nil, errors.New("transit key name is required")
	}

	mountPath := strings.Trim(config.TransitMountPath, "/")
	if mountPath == "" {
		mountPath = defaultTransitMountPath
	}

	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = defaultCacheTTL
	}

	clock := config.Clock
	if clock == nil {
		clock = time.Now
	}

	return &JWTVerifier{
		client:        config.VaultClient,
		mountPath:     mountPath,
		keyName:       config.KeyName,
		parserOptions: append([]jwt.ParserOption{}, config.ParserOptions...),
		cache:         make(map[string]crypto.PublicKey),
		cacheTTL:      cacheTTL,
		now:           clock,
	}, nil
}

// Verify parses and verifies the provided JWT string. The supplied claims must
// be a pointer type compatible with jwt.ParseWithClaims.
func (v *JWTVerifier) Verify(ctx context.Context, tokenString string, claims jwt.Claims, opts ...jwt.ParserOption) (*jwt.Token, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("token string is required")
	}

	parserOpts := append([]jwt.ParserOption{}, v.parserOptions...)
	parserOpts = append(parserOpts, opts...)

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		var kid string
		if headerValue, ok := token.Header["kid"]; ok {
			if kidStr, ok := headerValue.(string); ok {
				kid = kidStr
			}
		}
		pubKey, err := v.lookupKey(ctx, kid)
		if err != nil {
			return nil, err
		}
		return pubKey, nil
	}

	if claims == nil {
		claims = jwt.MapClaims{}
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc, parserOpts...)
	if err != nil {
		return nil, fmt.Errorf("verify jwt: %w", err)
	}
	return token, nil
}

func (v *JWTVerifier) lookupKey(ctx context.Context, kid string) (crypto.PublicKey, error) {
	v.mu.RLock()
	pubKey, ok := v.getCachedKeyLocked(kid)
	cacheExpired := v.cacheExpiry.Before(v.now())
	v.mu.RUnlock()

	if ok && !cacheExpired {
		return pubKey, nil
	}

	if err := v.refreshKeys(ctx, !ok); err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	pubKey, ok = v.getCachedKeyLocked(kid)
	if !ok {
		return nil, fmt.Errorf("unknown kid %q for key %s", kid, v.keyName)
	}

	return pubKey, nil
}

func (v *JWTVerifier) getCachedKeyLocked(kid string) (crypto.PublicKey, bool) {
	for _, candidate := range v.candidateKids(kid) {
		if key, ok := v.cache[candidate]; ok {
			return key, true
		}
	}
	return nil, false
}

func (v *JWTVerifier) refreshKeys(ctx context.Context, force bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Another goroutine may have refreshed already.
	if !force && v.cacheExpiry.After(v.now()) && len(v.cache) > 0 {
		return nil
	}

	data, err := v.fetchTransitKeyData(ctx)
	if err != nil {
		return err
	}

	newCache, err := v.extractPublicKeys(data)
	if err != nil {
		return err
	}

	v.cache = newCache
	v.cacheExpiry = v.now().Add(v.cacheTTL)
	return nil
}

func (v *JWTVerifier) fetchTransitKeyData(ctx context.Context) (map[string]interface{}, error) {
	path := fmt.Sprintf("%s/keys/%s", v.mountPath, v.keyName)
	secret, err := v.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("read transit key %q: %w", v.keyName, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("transit key %q not found", v.keyName)
	}
	return secret.Data, nil
}

func (v *JWTVerifier) extractPublicKeys(data map[string]interface{}) (map[string]crypto.PublicKey, error) {
	rawKeys, ok := data["keys"].(map[string]interface{})
	if !ok || len(rawKeys) == 0 {
		return nil, fmt.Errorf("transit key %q missing keys data", v.keyName)
	}

	cache := make(map[string]crypto.PublicKey, len(rawKeys)*4)

	for version, raw := range rawKeys {
		pubKey, present, err := v.parseKeyEntry(raw)
		if err != nil {
			return nil, fmt.Errorf("parse public key (version %s): %w", version, err)
		}
		if !present {
			continue
		}
		v.addKeyAliases(cache, version, pubKey)
	}

	if len(cache) == 0 {
		return nil, fmt.Errorf("no usable public keys found for %q", v.keyName)
	}

	return cache, nil
}

func (v *JWTVerifier) parseKeyEntry(raw interface{}) (crypto.PublicKey, bool, error) {
	keyInfo, ok := raw.(map[string]interface{})
	if !ok {
		return nil, false, nil
	}

	publicKeyPEM, ok := stringField(keyInfo, "public_key")
	if !ok || strings.TrimSpace(publicKeyPEM) == "" {
		if fallback, ok := stringField(keyInfo, "public_key_pem"); ok {
			publicKeyPEM = fallback
		}
	}
	if strings.TrimSpace(publicKeyPEM) == "" {
		return nil, false, nil
	}

	pubKey, err := parseRSAPublicKey(publicKeyPEM)
	if err != nil {
		return nil, false, err
	}

	return pubKey, true, nil
}

func (v *JWTVerifier) addKeyAliases(cache map[string]crypto.PublicKey, version string, key crypto.PublicKey) {
	cache[version] = key
	cache[fmt.Sprintf("%s-%s", v.keyName, version)] = key
	cache[fmt.Sprintf("%s-v%s", v.keyName, version)] = key
	cache[fmt.Sprintf("%s:%s", v.keyName, version)] = key
	cache[fmt.Sprintf("v%s", version)] = key
}

func (v *JWTVerifier) candidateKids(kid string) []string {
	if kid == "" {
		return nil
	}

	add := func(list []string, candidate string) []string {
		if candidate == "" {
			return list
		}
		for _, existing := range list {
			if existing == candidate {
				return list
			}
		}
		return append(list, candidate)
	}

	candidates := []string{}
	candidates = add(candidates, kid)

	if trimmed := strings.TrimPrefix(kid, v.keyName+"-"); trimmed != kid {
		candidates = add(candidates, trimmed)
	}
	if trimmed := strings.TrimPrefix(kid, v.keyName+":"); trimmed != kid {
		candidates = add(candidates, trimmed)
	}
	if strings.HasPrefix(kid, "v") {
		candidates = add(candidates, strings.TrimPrefix(kid, "v"))
	}

	return candidates
}

func parseRSAPublicKey(pemString string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return nil, errors.New("failed to decode PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX public key: %w", err)
	}

	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}

	return rsaKey, nil
}

func stringField(data map[string]interface{}, key string) (string, bool) {
	value, ok := data[key]
	if !ok {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}
