// Copyright 2025 Andrew Vasilyev
// SPDX-License-Identifier: APACHE-2.0

package middleware

import (
	"context"
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
	AuthContextKey contextKey = "auth"
)

// AuthInfo contains authenticated user information.
type AuthInfo struct {
	Email    string
	FullName string
	Role     clientgraphql.UserRole
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
		// Try to extract token from Authorization header first (for API clients)
		tokenString := ExtractTokenFromHeader(r)

		// If not in header, try cookie (for browser clients)
		if tokenString == "" {
			cookie, err := r.Cookie("nexus_auth")
			if err != nil {
				http.Error(w, "Unauthorized: No authentication token", http.StatusUnauthorized)
				return
			}
			tokenString = cookie.Value
		}

		// Parse and validate JWT
		claims, err := m.validateToken(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid token - "+err.Error(), http.StatusUnauthorized)
			return
		}

		// Parse user ID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			http.Error(w, "Unauthorized: Invalid user ID", http.StatusUnauthorized)
			return
		}

		// Check if user exists and is active
		userResp, err := clientgraphql.GetUser(r.Context(), m.graphqlClient, userID)
		if err != nil || userResp.User == nil {
			http.Error(w, "Unauthorized: User not found", http.StatusUnauthorized)
			return
		}

		user := userResp.User
		if user.Role == clientgraphql.UserRoleNone {
			http.Error(w, "Forbidden: Account pending approval", http.StatusForbidden)
			return
		}

		// Create auth info
		name := ""
		if user.Name != nil {
			name = *user.Name
		}

		authInfo := &AuthInfo{
			UserID:   user.Id,
			Email:    user.Email,
			FullName: name,
			Role:     user.Role,
		} // Add auth info to context
		ctx := context.WithValue(r.Context(), AuthContextKey, authInfo)

		// Call next handler
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

		if authInfo.Role != clientgraphql.UserRoleAdmin {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}

// OptionalAuth adds auth info to context if token is present, but doesn't require it.
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to extract token from header first, then cookie
		tokenString := ExtractTokenFromHeader(r)
		if tokenString == "" {
			cookie, err := r.Cookie("nexus_auth")
			if err != nil {
				// No token, continue without auth
				next.ServeHTTP(w, r)
				return
			}
			tokenString = cookie.Value
		}

		// Try to validate token
		claims, err := m.validateToken(tokenString)
		if err != nil {
			// Invalid token, continue without auth
			next.ServeHTTP(w, r)
			return
		}

		// Parse user ID
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		// Try to get user
		userResp, err := clientgraphql.GetUser(r.Context(), m.graphqlClient, userID)
		if err != nil || userResp.User == nil || userResp.User.Role == clientgraphql.UserRoleNone {
			// User not found or pending approval, continue without auth
			next.ServeHTTP(w, r)
			return
		}

		user := userResp.User

		// Create auth info
		name := ""
		if user.Name != nil {
			name = *user.Name
		}

		authInfo := &AuthInfo{
			UserID:   user.Id,
			Email:    user.Email,
			FullName: name,
			Role:     user.Role,
		} // Add auth info to context
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
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
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
