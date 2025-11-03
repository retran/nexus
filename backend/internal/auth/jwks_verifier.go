// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package auth provides utilities for verifying JWTs.
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWKSVerifier verifies JWT tokens using a static JWKS file.
type JWKSVerifier struct {
	keys map[string]*rsa.PublicKey
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key.
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// NewJWKSVerifier creates a new JWKS-based JWT verifier.
func NewJWKSVerifier(jwksFilePath string) (*JWKSVerifier, error) {
	if strings.TrimSpace(jwksFilePath) == "" {
		return nil, errors.New("JWKS file path is required")
	}

	// #nosec G304 -- jwksFilePath is from trusted configuration, not user input
	data, err := os.ReadFile(jwksFilePath)
	if err != nil {
		return nil, fmt.Errorf("read JWKS file: %w", err)
	}

	var jwks JWKS
	if err := json.Unmarshal(data, &jwks); err != nil {
		return nil, fmt.Errorf("unmarshal JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey)
	for i := range jwks.Keys {
		jwk := &jwks.Keys[i]
		if jwk.Kty != "RSA" {
			continue
		}

		pubKey, err := jwkToRSAPublicKey(jwk)
		if err != nil {
			return nil, fmt.Errorf("convert JWK to RSA public key (kid=%s): %w", jwk.Kid, err)
		}

		keys[jwk.Kid] = pubKey
	}

	if len(keys) == 0 {
		return nil, errors.New("no valid RSA keys found in JWKS")
	}

	return &JWKSVerifier{
		keys: keys,
	}, nil
}

// Verify parses and verifies the provided JWT string using the JWKS.
func (v *JWKSVerifier) Verify(_ context.Context, tokenString string, claims jwt.Claims, opts ...jwt.ParserOption) (*jwt.Token, error) {
	if strings.TrimSpace(tokenString) == "" {
		return nil, errors.New("token string is required")
	}

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("missing kid in token header")
		}

		pubKey, exists := v.keys[kid]
		if !exists {
			return nil, fmt.Errorf("unknown kid: %s", kid)
		}

		return pubKey, nil
	}

	allOpts := append([]jwt.ParserOption{jwt.WithValidMethods([]string{"RS256"})}, opts...)

	token, err := jwt.ParseWithClaims(tokenString, claims, keyFunc, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	return token, nil
}

// jwkToRSAPublicKey converts a JWK to an RSA public key.
func jwkToRSAPublicKey(jwk *JWK) (*rsa.PublicKey, error) {
	nBytes, err := base64URLDecode(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}

	eBytes, err := base64URLDecode(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

// base64URLDecode decodes a base64url-encoded string.
func base64URLDecode(s string) ([]byte, error) {
	// Convert base64url to base64
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	return decoded, nil
}
