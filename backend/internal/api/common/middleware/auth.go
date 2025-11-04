// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: Apache-2.0

// Package middleware provides HTTP middleware helpers for the REST gateway.
package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const authInfoKey contextKey = "gateway-auth-info"

// AuthMiddleware validates requests that pass through Oathkeeper before reaching the gateway.
// It uses mTLS client certificate CN validation instead of JWT.
type AuthMiddleware struct {
	// No configuration needed - we just check CN from TLS
}

// AuthInfo describes the authenticated user extracted from Oathkeeper headers via mTLS.
type AuthInfo struct {
	Email     string
	Role      string
	SessionID string
	FullName  string
	UserID    uuid.UUID
}

// NewAuthMiddleware builds a middleware instance that validates requests coming from Oathkeeper via mTLS.
func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{}
}

// RequireAuth rejects requests that do not include a valid Bearer token issued by Oathkeeper.
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, err := m.authenticateRequest(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), authInfoKey, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth attempts to authenticate requests but allows anonymous access when no cert is provided.
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, err := m.authenticateRequest(r)
		if err == nil {
			ctx := context.WithValue(r.Context(), authInfoKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if errors.Is(err, errMissingCert) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

var (
	errMissingCert    = errors.New("missing client certificate")
	errInvalidCN      = errors.New("invalid client certificate CN")
	errInvalidHeaders = errors.New("invalid oathkeeper headers")
)

const expectedCN = "oathkeeper.service.local"

func (m *AuthMiddleware) authenticateRequest(r *http.Request) (*AuthInfo, error) {
	// Check mTLS client certificate CN
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return nil, errMissingCert
	}

	cn := r.TLS.PeerCertificates[0].Subject.CommonName
	if cn != expectedCN {
		return nil, fmt.Errorf("%w: got %q, expected %q", errInvalidCN, cn, expectedCN)
	}

	// CN is valid - now read user info from Oathkeeper headers (Trusted Subsystem pattern)
	info, err := buildAuthInfoFromHeaders(r)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func buildAuthInfoFromHeaders(r *http.Request) (*AuthInfo, error) {
	// Read X-User-ID header
	userIDHeader, err := readAndValidateHeader(r, "X-User-ID")
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(userIDHeader)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user id", errInvalidHeaders)
	}

	// Read X-User-Email header
	email, err := readAndValidateHeader(r, "X-User-Email")
	if err != nil {
		return nil, err
	}

	// Read X-User-Role header
	role, err := readAndValidateHeader(r, "X-User-Role")
	if err != nil {
		return nil, err
	}
	role = strings.ToLower(strings.TrimSpace(role))

	// Read X-Session-ID header (optional for now)
	sessionID := strings.TrimSpace(r.Header.Get("X-Session-ID"))

	// Read X-User-Name header (optional)
	fullName := strings.TrimSpace(r.Header.Get("X-User-Name"))

	return &AuthInfo{
		UserID:    userID,
		Email:     email,
		Role:      role,
		SessionID: sessionID,
		FullName:  fullName,
	}, nil
}

func readAndValidateHeader(r *http.Request, header string) (string, error) {
	value := strings.TrimSpace(r.Header.Get(header))
	if value == "" {
		return "", fmt.Errorf("%w: header %s missing", errInvalidHeaders, header)
	}
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("%w: header %s contains control characters", errInvalidHeaders, header)
	}
	return value, nil
}

// AuthInfoFromContext retrieves the authenticated user from the request context.
func AuthInfoFromContext(ctx context.Context) *AuthInfo {
	if ctx == nil {
		return nil
	}
	if value := ctx.Value(authInfoKey); value != nil {
		if info, ok := value.(*AuthInfo); ok {
			return info
		}
	}
	return nil
}
