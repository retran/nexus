// Package middleware provides helpers for securing the internal API.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/retran/nexus/backend/internal/auth"
)

type contextKey string

// #nosec G101 -- identifier stored only in memory for context scoping.
const tokenInfoKey contextKey = "internal-api-jwt"

// TokenInfo captures details extracted from a verified JWT.
type TokenInfo struct {
	Token    string
	Subject  string
	Audience []string
}

// JWTMiddleware verifies JWTs issued by Vault transit and adds metadata to the request context.
type JWTMiddleware struct {
	verifier        *auth.JWTVerifier
	allowedAudience []string
}

// NewJWTMiddleware constructs a new middleware instance.
func NewJWTMiddleware(verifier *auth.JWTVerifier, allowedAudience []string) *JWTMiddleware {
	normalized := make([]string, 0, len(allowedAudience))
	for _, aud := range allowedAudience {
		aud = strings.TrimSpace(aud)
		if aud != "" {
			normalized = append(normalized, aud)
		}
	}
	return &JWTMiddleware{verifier: verifier, allowedAudience: normalized}
}

// Require ensures a valid Bearer token is present before allowing the request through.
func (m *JWTMiddleware) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims := &jwt.RegisteredClaims{}
		if _, err := m.verifier.Verify(r.Context(), token, claims); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !m.audienceAllowed(claims.Audience) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		info := &TokenInfo{
			Token:    token,
			Subject:  claims.Subject,
			Audience: claimStringsToSlice(claims.Audience),
		}

		ctx := context.WithValue(r.Context(), tokenInfoKey, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *JWTMiddleware) audienceAllowed(aud jwt.ClaimStrings) bool {
	if len(m.allowedAudience) == 0 || len(aud) == 0 {
		return true
	}

	for _, allowed := range m.allowedAudience {
		for _, actual := range aud {
			if strings.EqualFold(actual, allowed) {
				return true
			}
		}
	}
	return false
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
		out = append(out, v)
	}
	return out
}

// TokenInfoFromContext returns the JWT metadata extracted by the middleware.
func TokenInfoFromContext(ctx context.Context) *TokenInfo {
	if ctx == nil {
		return nil
	}
	if value := ctx.Value(tokenInfoKey); value != nil {
		if info, ok := value.(*TokenInfo); ok {
			return info
		}
	}
	return nil
}
