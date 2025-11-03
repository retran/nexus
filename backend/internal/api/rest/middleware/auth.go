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

	"github.com/Khan/genqlient/graphql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	clientgraphql "github.com/retran/nexus/backend/internal/client/graphql"
)

// AuthContextKey is the key for storing auth info in context.
type contextKey string

const (
	// AuthContextKey identifies the auth info stored in a request context.
	AuthContextKey contextKey = "auth"
)

// AuthInfo contains authenticated user information.
type AuthInfo struct {
	Email    string
	FullName string
	Role     string
	UserID   uuid.UUID
}

// JWTClaims represents the JWT token claims.
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens and checks user status.
type AuthMiddleware struct {
	graphqlClient graphql.Client
	jwtSecret     []byte
}

var (
	errNoToken         = errors.New("no authentication token")
	errInvalidUserID   = errors.New("invalid user id")
	errUserNotFound    = errors.New("user not found")
	errPendingApproval = errors.New("account pending approval")
)

// NewAuthMiddleware creates a new authentication middleware.
func NewAuthMiddleware(graphqlClient graphql.Client, jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		graphqlClient: graphqlClient,
		jwtSecret:     []byte(jwtSecret),
	}
}

// RequireAuth is middleware that requires valid authentication.
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := m.resolveToken(r)
		if err != nil {
			http.Error(w, "Unauthorized: No authentication token", http.StatusUnauthorized)
			return
		}

		claims, err := m.validateToken(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token - "+err.Error(), http.StatusUnauthorized)
			return
		}

		authInfo, err := m.loadAuthInfo(r.Context(), claims)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidUserID):
				http.Error(w, "Unauthorized: Invalid user ID", http.StatusUnauthorized)
			case errors.Is(err, errUserNotFound):
				http.Error(w, "Unauthorized: User not found", http.StatusUnauthorized)
			case errors.Is(err, errPendingApproval):
				http.Error(w, "Forbidden: Account pending approval", http.StatusForbidden)
			default:
				http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			}
			return
		}

		ctx := context.WithValue(r.Context(), AuthContextKey, authInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin is middleware that requires admin role.
func (m *AuthMiddleware) RequireAdmin(next http.Handler) http.Handler {
	return m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo := GetAuthInfo(r.Context())
		if authInfo == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !strings.EqualFold(authInfo.Role, "admin") {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}

// OptionalAuth adds auth info to context if token is present, but doesn't require it.
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := m.resolveToken(r)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		claims, err := m.validateToken(tokenString)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		authInfo, err := m.loadAuthInfo(r.Context(), claims)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), AuthContextKey, authInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateToken validates and parses a JWT token.
func (m *AuthMiddleware) validateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse jwt: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

func (m *AuthMiddleware) resolveToken(r *http.Request) (string, error) {
	if token := ExtractTokenFromHeader(r); token != "" {
		return token, nil
	}

	cookie, err := r.Cookie("nexus_auth")
	if err != nil {
		return "", errNoToken
	}

	return cookie.Value, nil
}

func (m *AuthMiddleware) loadAuthInfo(ctx context.Context, claims *JWTClaims) (*AuthInfo, error) {
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, errInvalidUserID
	}

	userResp, err := clientgraphql.GetUser(ctx, m.graphqlClient, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if userResp.User == nil {
		return nil, errUserNotFound
	}

	role := strings.ToLower(strings.TrimSpace(claims.Role))
	if role == "" || role == "none" {
		return nil, errPendingApproval
	}

	name := ""
	if userResp.User.Name != nil {
		name = *userResp.User.Name
	}
	if name == "" && claims.FullName != "" {
		name = claims.FullName
	}

	// Fallbacks in case user record lacks email information
	email := userResp.User.Email
	if email == "" {
		email = claims.Email
	}

	return &AuthInfo{
		UserID:   userResp.User.Id,
		Email:    email,
		FullName: name,
		Role:     role,
	}, nil
}

// GetAuthInfo retrieves auth info from context.
func GetAuthInfo(ctx context.Context) *AuthInfo {
	authInfo, ok := ctx.Value(AuthContextKey).(*AuthInfo)
	if !ok {
		return nil
	}
	return authInfo
}

// ExtractTokenFromHeader extracts JWT from Authorization header (Bearer token).
func ExtractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	// Bearer token format: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
