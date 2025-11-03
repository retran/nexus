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

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenVerifier abstracts JWT verification for dependency injection and testing.
type TokenVerifier interface {
	Verify(ctx context.Context, tokenString string, claims jwt.Claims, opts ...jwt.ParserOption) (*jwt.Token, error)
}

type contextKey string

const authInfoKey contextKey = "gateway-auth-info"

// Config configures the authentication middleware.
type Config struct {
	Verifier TokenVerifier
	Issuer   string
	Subject  string
	Audience []string
}

// AuthMiddleware validates requests that pass through Oathkeeper before reaching the gateway.
type AuthMiddleware struct {
	verifier TokenVerifier

	expectedIssuer  string
	expectedSubject string
	allowedAudience []string
}

// AuthInfo describes the authenticated user extracted from Oathkeeper headers and JWT claims.
type AuthInfo struct {
	Token     string
	Email     string
	Role      string
	SessionID string
	Issuer    string
	Subject   string
	FullName  string
	Audience  []string
	UserID    uuid.UUID
}

// NewAuthMiddleware builds a middleware instance that validates requests coming from Oathkeeper.
func NewAuthMiddleware(cfg Config) (*AuthMiddleware, error) {
	if cfg.Verifier == nil {
		return nil, errors.New("auth middleware requires a verifier")
	}

	audience := make([]string, 0, len(cfg.Audience))
	seen := make(map[string]struct{}, len(cfg.Audience))
	for _, aud := range cfg.Audience {
		normalized := strings.TrimSpace(aud)
		if normalized == "" {
			continue
		}
		key := strings.ToLower(normalized)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		audience = append(audience, normalized)
	}

	return &AuthMiddleware{
		verifier:        cfg.Verifier,
		expectedIssuer:  strings.TrimSpace(cfg.Issuer),
		expectedSubject: strings.TrimSpace(cfg.Subject),
		allowedAudience: audience,
	}, nil
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

// OptionalAuth attempts to authenticate requests but allows anonymous access when no token is provided.
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info, err := m.authenticateRequest(r)
		if err == nil {
			ctx := context.WithValue(r.Context(), authInfoKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		if errors.Is(err, errMissingToken) {
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

var (
	errMissingToken   = errors.New("missing bearer token")
	errInvalidHeaders = errors.New("invalid oathkeeper headers")
)

func (m *AuthMiddleware) authenticateRequest(r *http.Request) (*AuthInfo, error) {
	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		return nil, errMissingToken
	}

	claims := &oathkeeperClaims{}
	if _, err := m.verifier.Verify(r.Context(), token, claims); err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	if err := m.validateClaims(claims); err != nil {
		return nil, err
	}

	info, err := buildAuthInfo(r, token, claims)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (m *AuthMiddleware) validateClaims(claims *oathkeeperClaims) error {
	if claims == nil {
		return errors.New("missing claims")
	}
	if claims.Issuer != "" && m.expectedIssuer != "" && !strings.EqualFold(claims.Issuer, m.expectedIssuer) {
		return fmt.Errorf("unexpected issuer %q", claims.Issuer)
	}
	if claims.Subject != "" && m.expectedSubject != "" && !strings.EqualFold(claims.Subject, m.expectedSubject) {
		return fmt.Errorf("unexpected subject %q", claims.Subject)
	}
	if len(m.allowedAudience) > 0 && len(claims.Audience) > 0 && !audienceAllowed(m.allowedAudience, claims.Audience) {
		return errors.New("unexpected audience")
	}
	return nil
}

func audienceAllowed(allowed []string, actual jwt.ClaimStrings) bool {
	if len(allowed) == 0 {
		return true
	}

	for _, want := range allowed {
		for _, got := range actual {
			if strings.EqualFold(strings.TrimSpace(got), want) {
				return true
			}
		}
	}
	return false
}

func buildAuthInfo(r *http.Request, token string, claims *oathkeeperClaims) (*AuthInfo, error) {
	if claims == nil {
		return nil, errors.New("claims are required")
	}

	session := claims.Session
	identity := session.Identity
	traits := identity.Traits

	rawIdentityID := strings.TrimSpace(identity.ID)
	if rawIdentityID == "" {
		return nil, fmt.Errorf("%w: missing identity id", errInvalidHeaders)
	}
	userHeader, err := readAndValidateHeader(r, "X-User")
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(userHeader, rawIdentityID) {
		return nil, fmt.Errorf("%w: identity mismatch", errInvalidHeaders)
	}
	identityID, err := uuid.Parse(rawIdentityID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid identity id", errInvalidHeaders)
	}

	emailClaim := strings.TrimSpace(traits.Email)
	if emailClaim == "" {
		return nil, fmt.Errorf("%w: missing email claim", errInvalidHeaders)
	}
	emailHeader, err := readAndValidateHeader(r, "X-User-Email")
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(emailHeader, emailClaim) {
		return nil, fmt.Errorf("%w: email mismatch", errInvalidHeaders)
	}

	roleClaim := strings.ToLower(strings.TrimSpace(traits.Role))
	if roleClaim == "" {
		return nil, fmt.Errorf("%w: missing role claim", errInvalidHeaders)
	}
	roleHeader, err := readAndValidateHeader(r, "X-User-Role")
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(roleHeader, roleClaim) {
		return nil, fmt.Errorf("%w: role mismatch", errInvalidHeaders)
	}

	sessionID := strings.TrimSpace(session.ID)
	if sessionID == "" {
		return nil, fmt.Errorf("%w: missing session id", errInvalidHeaders)
	}

	fullName := deriveFullName(traits.Name.First, traits.Name.Last)

	return &AuthInfo{
		Token:     token,
		UserID:    identityID,
		Email:     emailClaim,
		Role:      roleClaim,
		SessionID: sessionID,
		Issuer:    claims.Issuer,
		Subject:   claims.Subject,
		Audience:  claimStringsToSlice(claims.Audience),
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

func deriveFullName(first, last string) string {
	first = strings.TrimSpace(first)
	last = strings.TrimSpace(last)
	switch {
	case first == "" && last == "":
		return ""
	case first == "":
		return last
	case last == "":
		return first
	default:
		return fmt.Sprintf("%s %s", first, last)
	}
}

func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func claimStringsToSlice(aud jwt.ClaimStrings) []string {
	if len(aud) == 0 {
		return nil
	}
	out := make([]string, 0, len(aud))
	for _, v := range aud {
		out = append(out, strings.TrimSpace(v))
	}
	return out
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

type oathkeeperClaims struct {
	jwt.RegisteredClaims
	Session struct {
		ID       string `json:"id"`
		Identity struct {
			ID     string `json:"id"`
			Traits struct {
				Email string `json:"email"`
				Role  string `json:"role"`
				Name  struct {
					First string `json:"first"`
					Last  string `json:"last"`
				} `json:"name"`
			} `json:"traits"`
		} `json:"identity"`
	} `json:"session"`
}
