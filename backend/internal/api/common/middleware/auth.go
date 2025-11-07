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
// It validates user information from Oathkeeper-provided headers.
type AuthMiddleware struct {
	// No configuration needed - we trust headers from Oathkeeper
}

// AuthInfo describes the authenticated user extracted from Oathkeeper headers.
type AuthInfo struct {
	Email     string
	Role      string
	SessionID string
	FullName  string
	UserID    uuid.UUID
}

// NewAuthMiddleware builds a middleware instance that validates requests coming from Oathkeeper.
func NewAuthMiddleware() *AuthMiddleware {
	return &AuthMiddleware{}
}

// RequireAuth rejects requests that do not include valid authentication headers set by Oathkeeper.
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

// OptionalAuth attempts to authenticate requests but allows anonymous access when no auth headers are provided.
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, err := m.authenticateRequest(r)
		if err == nil {
			ctx := context.WithValue(r.Context(), authInfoKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if errors.Is(err, errMissingAuth) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

var (
	errMissingAuth    = errors.New("missing authentication")
	errInvalidHeaders = errors.New("invalid oathkeeper headers")
)

func (m *AuthMiddleware) authenticateRequest(r *http.Request) (*AuthInfo, error) {
	info, err := buildAuthInfoFromHeaders(r)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func buildAuthInfoFromHeaders(r *http.Request) (*AuthInfo, error) {
	userIDHeader, err := readAndValidateHeader(r, "X-User-ID")
	if err != nil {
		return nil, err
	}

	userID, err := uuid.Parse(userIDHeader)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user id", errInvalidHeaders)
	}

	email, err := readAndValidateHeader(r, "X-User-Email")
	if err != nil {
		return nil, err
	}

	role, err := readAndValidateHeader(r, "X-User-Role")
	if err != nil {
		return nil, err
	}

	sessionID, err := readAndValidateHeader(r, "X-Session-ID")
	if err != nil {
		return nil, err
	}

	fullName, err := readAndValidateHeader(r, "X-User-Name")
	if err != nil {
		return nil, err
	}

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
